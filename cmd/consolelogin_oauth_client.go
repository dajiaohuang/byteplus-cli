package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// defaultConsoleEndpoint is the default Byteplus console sign-in endpoint.
	defaultConsoleEndpoint = "https://signin.byteplus.com"

	// consoleTokenPath is the path appended to the endpoint for the token URL.
	consoleTokenPath = "/authorize/oauth/token"

	// consoleDeviceAuthorizationPath is the path used to start device authorization.
	consoleDeviceAuthorizationPath = "/authorize/oauth/device_authorization"

	// consoleTokenRequestTimeout is the HTTP timeout for console token exchange requests.
	consoleTokenRequestTimeout = 30 * time.Second

	// consoleTokenRetryAttempts is the number of retry attempts for token exchange.
	consoleTokenRetryAttempts = 3

	// ConsoleClientIDSameDevice is the legacy public client ID issued by the
	// removed authorization code flow. Login never mints it again; refresh
	// replays whatever client ID the cache holds, so caches written by older
	// CLI versions keep working. Kept to document that value and to pin it in
	// the compatibility tests.
	ConsoleClientIDSameDevice = "trn:signin:::devtools/same-device"

	// ConsoleClientIDCrossDevice is the public client ID used by device code login.
	ConsoleClientIDCrossDevice = "trn:signin:::devtools/cross-device"
)

// ---------------------------------------------------------------------------
// Error types (independent from SSO OAuthAPIError)
// ---------------------------------------------------------------------------

type ConsoleOAuthErrorResponse struct {
	State            string `json:"state,omitempty"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

type ConsoleOAuthAPIError struct {
	StatusCode int
	Response   ConsoleOAuthErrorResponse
	RawBody    string
	RequestID  string // X-Tt-Logid header
}

func (e *ConsoleOAuthAPIError) Error() string {
	if e == nil {
		return ""
	}

	var parts []string
	if e.Response.Error != "" {
		parts = append(parts, e.Response.Error)
	}
	if e.Response.ErrorDescription != "" {
		parts = append(parts, e.Response.ErrorDescription)
	}

	msg := strings.Join(parts, ": ")
	if msg == "" {
		if e.RawBody != "" {
			msg = e.RawBody
		} else {
			msg = "unknown error"
		}
	}

	suffix := fmt.Sprintf("[status %d", e.StatusCode)
	if e.RequestID != "" {
		suffix += ", requestId: " + e.RequestID
	}
	suffix += "]"

	return fmt.Sprintf("console oauth request failed: %s %s", msg, suffix)
}

func (e *ConsoleOAuthAPIError) IsRetryable() bool {
	if e == nil {
		return false
	}
	return e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode == http.StatusRequestTimeout ||
		e.StatusCode/100 == 5
}

// ---------------------------------------------------------------------------
// Client config & types
// ---------------------------------------------------------------------------

type ConsoleOAuthClientConfig struct {
	EndpointURL string
	HTTPClient  *http.Client
}

type ConsoleOAuthClient struct {
	endpointURL            string
	tokenURL               string
	deviceAuthorizationURL string
	httpClient             *http.Client
}

type ConsoleTokenRequest struct {
	GrantType    string // deviceCodeGrantType or "refresh_token"
	ClientID     string
	Scope        string
	RefreshToken string // for refresh_token grant
	DeviceCode   string // for device code grant
}

type ConsoleDeviceAuthorizationRequest struct {
	ClientID   string
	Scope      string
	DeviceInfo string
}

type ConsoleDeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

type ConsoleTokenResponse struct {
	AccessToken  string `json:"access_token"` // JSON string containing STS credentials
	TokenType    string `json:"token_type"`   // e.g. "urn:ietf:params:oauth:token-type:access_token_sts"
	ExpiresIn    int    `json:"expires_in"`   // seconds, e.g. 900
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"` // JWT
}

type STSCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func NewConsoleOAuthClient(cfg *ConsoleOAuthClientConfig) *ConsoleOAuthClient {
	endpoint := defaultConsoleEndpoint
	if cfg != nil && strings.TrimSpace(cfg.EndpointURL) != "" {
		endpoint = strings.TrimSpace(cfg.EndpointURL)
	}
	endpoint = strings.TrimRight(endpoint, "/")

	client := &http.Client{Timeout: consoleTokenRequestTimeout}
	if cfg != nil && cfg.HTTPClient != nil {
		client = cfg.HTTPClient
	}

	return &ConsoleOAuthClient{
		endpointURL:            endpoint,
		tokenURL:               endpoint + consoleTokenPath,
		deviceAuthorizationURL: endpoint + consoleDeviceAuthorizationPath,
		httpClient:             client,
	}
}

func (c *ConsoleOAuthClient) StartDeviceAuthorization(
	ctx context.Context,
	req *ConsoleDeviceAuthorizationRequest,
) (*ConsoleDeviceAuthorizationResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	q := url.Values{}
	q.Set("client_id", req.ClientID)
	if req.Scope != "" {
		q.Set("scope", req.Scope)
	}
	if req.DeviceInfo != "" {
		q.Set("device_info", req.DeviceInfo)
	}

	var authResp ConsoleDeviceAuthorizationResponse
	if err := c.postForm(ctx, c.deviceAuthorizationURL, q, &authResp, consoleTokenRetryAttempts); err != nil {
		return nil, err
	}
	if strings.TrimSpace(authResp.DeviceCode) == "" {
		return nil, fmt.Errorf("device authorization response missing device_code")
	}
	if strings.TrimSpace(authResp.UserCode) == "" {
		return nil, fmt.Errorf("device authorization response missing user_code")
	}
	if strings.TrimSpace(authResp.VerificationURI) == "" {
		return nil, fmt.Errorf("device authorization response missing verification_uri")
	}
	if authResp.ExpiresIn <= 0 {
		return nil, fmt.Errorf("device authorization response has invalid expires_in")
	}

	return &authResp, nil
}

// ---------------------------------------------------------------------------
// ExchangeToken
// ---------------------------------------------------------------------------

func (c *ConsoleOAuthClient) ExchangeToken(ctx context.Context, req *ConsoleTokenRequest) (*ConsoleTokenResponse, error) {
	return c.exchangeToken(ctx, req, consoleTokenRetryAttempts)
}

func (c *ConsoleOAuthClient) exchangeToken(ctx context.Context, req *ConsoleTokenRequest, attempts int) (*ConsoleTokenResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if strings.TrimSpace(req.GrantType) == "" {
		return nil, fmt.Errorf("grant_type is required")
	}
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	q := url.Values{}
	q.Set("grant_type", req.GrantType)
	q.Set("client_id", req.ClientID)

	if req.Scope != "" {
		q.Set("scope", req.Scope)
	}

	switch req.GrantType {
	case "refresh_token":
		if strings.TrimSpace(req.RefreshToken) == "" {
			return nil, fmt.Errorf("refresh_token is required for refresh_token grant")
		}
		q.Set("refresh_token", req.RefreshToken)

	case deviceCodeGrantType:
		if strings.TrimSpace(req.DeviceCode) == "" {
			return nil, fmt.Errorf("device_code is required for device code grant")
		}
		q.Set("device_code", req.DeviceCode)

	default:
		return nil, fmt.Errorf("unsupported grant_type: %s", req.GrantType)
	}

	var tokenResp ConsoleTokenResponse
	if err := c.postForm(ctx, c.tokenURL, q, &tokenResp, attempts); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" && tokenResp.TokenType == "" &&
		tokenResp.RefreshToken == "" && tokenResp.ExpiresIn == 0 {
		return nil, fmt.Errorf("ExchangeToken succeeded but response was empty")
	}

	return &tokenResp, nil
}

func (c *ConsoleOAuthClient) postForm(ctx context.Context, endpoint string, form url.Values, out interface{}, attempts int) error {
	requestBody := form.Encode()

	return doWithRetry(ctx, retryOptions{maxAttempts: attempts}, func() error {
		httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(requestBody))
		if reqErr != nil {
			return fmt.Errorf("failed to build request: %w", reqErr)
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if customHeaders := os.Getenv("BYTEPLUS_LOGIN_HEADERS"); customHeaders != "" {
			for _, entry := range strings.Split(customHeaders, ";") {
				if idx := strings.Index(entry, "="); idx > 0 {
					httpReq.Header.Set(strings.TrimSpace(entry[:idx]), strings.TrimSpace(entry[idx+1:]))
				}
			}
		}
		resp, doErr := c.httpClient.Do(httpReq)
		if doErr != nil {
			return fmt.Errorf("request failed: %w", doErr)
		}
		defer resp.Body.Close()

		respBytes, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response: %w", readErr)
		}

		requestID := resp.Header.Get("X-Tt-Logid")

		if resp.StatusCode/100 != 2 {
			apiErr := &ConsoleOAuthAPIError{
				StatusCode: resp.StatusCode,
				RequestID:  requestID,
				RawBody:    string(respBytes),
			}

			if len(respBytes) > 0 {
				var errResp ConsoleOAuthErrorResponse
				if json.Unmarshal(respBytes, &errResp) == nil && errResp.Error != "" {
					apiErr.Response = errResp
				}
			}

			return apiErr
		}

		if len(respBytes) > 0 && out != nil {
			if unmarshalErr := json.Unmarshal(respBytes, out); unmarshalErr != nil {
				return fmt.Errorf(
					"failed to decode oauth response (status %d, requestId: %s): %w",
					resp.StatusCode, requestID, unmarshalErr,
				)
			}
		}

		return nil
	})
}

// ---------------------------------------------------------------------------
// ParseSTSCredentials
// ---------------------------------------------------------------------------

func ParseSTSCredentials(accessToken string) (*STSCredentials, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("access_token is empty")
	}

	var creds STSCredentials
	if err := json.Unmarshal([]byte(accessToken), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse STS credentials from access_token: %w", err)
	}

	if creds.AccessKeyID == "" {
		return nil, fmt.Errorf("parsed STS credentials missing access_key_id")
	}
	if creds.SecretAccessKey == "" {
		return nil, fmt.Errorf("parsed STS credentials missing secret_access_key")
	}
	if creds.SessionToken == "" {
		return nil, fmt.Errorf("parsed STS credentials missing session_token")
	}

	return &creds, nil
}
