package cmd

import (
	"strings"
	"testing"
)

func TestParserReadsFixedFlags(t *testing.T) {
	ctx := NewContext()
	parser := NewParser([]string{
		"---profile", "release",
		"---region", "ap-southeast-1",
		"---endpoint", "sts.byteplusapi.com",
		"--Limit", "10",
	})

	args, err := parser.ReadArgs(ctx)
	if err != nil {
		t.Fatalf("ReadArgs() error = %v", err)
	}
	if len(args) != 0 {
		t.Fatalf("ReadArgs() args = %v, want empty", args)
	}
	if got := ctx.fixedFlags.GetByName("profile").GetValue(); got != "release" {
		t.Fatalf("profile fixed flag = %q, want release", got)
	}
	if got := ctx.fixedFlags.GetByName("region").GetValue(); got != "ap-southeast-1" {
		t.Fatalf("region fixed flag = %q, want ap-southeast-1", got)
	}
	if got := ctx.fixedFlags.GetByName("endpoint").GetValue(); got != "sts.byteplusapi.com" {
		t.Fatalf("endpoint fixed flag = %q, want sts.byteplusapi.com", got)
	}
	if got := ctx.dynamicFlags.GetByName("Limit").GetValue(); got != "10" {
		t.Fatalf("dynamic flag Limit = %q, want 10", got)
	}
}

func TestParserRejectsUnsupportedFixedFlag(t *testing.T) {
	ctx := NewContext()
	parser := NewParser([]string{"---trace", "true"})

	_, err := parser.ReadArgs(ctx)
	if err == nil {
		t.Fatal("ReadArgs() error = nil, want unsupported fixed flag error")
	}
	if !strings.Contains(err.Error(), "---trace is not supported") {
		t.Fatalf("ReadArgs() error = %q, want unsupported fixed flag message", err)
	}
}
func TestParserRejectsDebugFixedFlags(t *testing.T) {
	ctx := NewContext()
	parser := NewParser([]string{"---debug", "true"})

	_, err := parser.ReadArgs(ctx)
	if err == nil {
		t.Fatal("ReadArgs() error = nil, want unsupported debug fixed flag error")
	}
	if !strings.Contains(err.Error(), "---debug is not supported") {
		t.Fatalf("ReadArgs() error = %q, want unsupported debug fixed flag message", err)
	}
}
func TestParserRequiresFixedFlagValue(t *testing.T) {
	ctx := NewContext()
	parser := NewParser([]string{"---region"})

	_, err := parser.ReadArgs(ctx)
	if err == nil {
		t.Fatal("ReadArgs() error = nil, want missing fixed flag value error")
	}
	if !strings.Contains(err.Error(), "---region must set value") {
		t.Fatalf("ReadArgs() error = %q, want missing value message", err)
	}
}

func TestParserAcceptsPEMValuesStartingWithHyphens(t *testing.T) {
	publicKey := "-----BEGIN CERTIFICATE-----\ncertificate-data\n-----END CERTIFICATE-----"
	privateKey := "-----BEGIN RSA PRIVATE KEY-----\nprivate-key-data\n-----END RSA PRIVATE KEY-----"
	parser := NewParser([]string{
		"--CertificateName", "repro-test",
		"--PublicKey", publicKey,
		"--PrivateKey", privateKey,
		"---region", "ap-southeast-1",
	})
	ctx := NewContext()

	if _, err := parser.ReadArgs(ctx); err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}

	for name, want := range map[string]string{
		"CertificateName": "repro-test",
		"PublicKey":       publicKey,
		"PrivateKey":      privateKey,
	} {
		flag := ctx.dynamicFlags.GetByName(name)
		if flag == nil {
			t.Fatalf("expected dynamic flag %q", name)
		}
		if got := flag.GetValue(); got != want {
			t.Fatalf("dynamic flag %q value = %q, want %q", name, got, want)
		}
	}

	region := ctx.fixedFlags.GetByName("region")
	if region == nil {
		t.Fatal("expected fixed flag \"region\"")
	}
	if got := region.GetValue(); got != "ap-southeast-1" {
		t.Fatalf("fixed flag \"region\" value = %q, want %q", got, "ap-southeast-1")
	}
}

func TestParserTreatsNextTokenAsValueRegardlessOfLeadingHyphens(t *testing.T) {
	parser := NewParser([]string{
		"--SingleHyphen", "-value",
		"--DoubleHyphen", "--value",
		"--TripleHyphen", "---value",
		"--FourHyphens", "----value",
		"--FiveHyphens", "-----value",
		"---profile", "---profile-value",
		"---region", "ap-southeast-1",
	})
	ctx := NewContext()

	if _, err := parser.ReadArgs(ctx); err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}

	for name, want := range map[string]string{
		"SingleHyphen": "-value",
		"DoubleHyphen": "--value",
		"TripleHyphen": "---value",
		"FourHyphens":  "----value",
		"FiveHyphens":  "-----value",
	} {
		flag := ctx.dynamicFlags.GetByName(name)
		if flag == nil {
			t.Fatalf("expected dynamic flag %q", name)
		}
		if got := flag.GetValue(); got != want {
			t.Fatalf("dynamic flag %q value = %q, want %q", name, got, want)
		}
	}

	profile := ctx.fixedFlags.GetByName("profile")
	if profile == nil || profile.GetValue() != "---profile-value" {
		t.Fatalf("fixed flag \"profile\" = %#v, want value %q", profile, "---profile-value")
	}
	region := ctx.fixedFlags.GetByName("region")
	if region == nil || region.GetValue() != "ap-southeast-1" {
		t.Fatalf("fixed flag \"region\" = %#v, want value %q", region, "ap-southeast-1")
	}
}

func TestParserUsesLegacyDiagnosticsForOddArgumentCount(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "dynamic flag before dynamic flag",
			args:    []string{"--Foo", "--Bar", "1"},
			wantErr: "--Foo must set value.",
		},
		{
			name:    "dynamic flag before fixed flag",
			args:    []string{"--Foo", "---region", "ap-southeast-1"},
			wantErr: "--Foo must set value.",
		},
		{
			name:    "fixed flag before dynamic flag",
			args:    []string{"---profile", "--Foo", "1"},
			wantErr: "---profile must set value.",
		},
		{
			name:    "trailing flag after complete pair",
			args:    []string{"--Foo", "value", "--Bar"},
			wantErr: "--Bar must set value.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewParser(tt.args).ReadArgs(NewContext())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParserContinuesToIgnoreUnpairedPositionalArgument(t *testing.T) {
	ctx := NewContext()
	positional, err := NewParser([]string{"--Foo", "value", "unexpected"}).ReadArgs(ctx)
	if err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}
	if len(positional) != 1 || positional[0] != "unexpected" {
		t.Fatalf("positional arguments = %#v, want []string{%q}", positional, "unexpected")
	}
	foo := ctx.dynamicFlags.GetByName("Foo")
	if foo == nil || foo.GetValue() != "value" {
		t.Fatalf("dynamic flag \"Foo\" = %#v, want value %q", foo, "value")
	}
}

func TestParserContinuesToIgnorePositionalArguments(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantPositional  []string
		wantDynamicName string
		wantDynamicVal  string
	}{
		{
			name:           "two positional arguments",
			args:           []string{"unexpected", "value"},
			wantPositional: []string{"unexpected", "value"},
		},
		{
			name:            "two trailing positional arguments",
			args:            []string{"--Foo", "value", "extra1", "extra2"},
			wantPositional:  []string{"extra1", "extra2"},
			wantDynamicName: "Foo",
			wantDynamicVal:  "value",
		},
		{
			name:            "leading and trailing positional arguments",
			args:            []string{"extra1", "--Foo", "value", "extra2"},
			wantPositional:  []string{"extra1", "extra2"},
			wantDynamicName: "Foo",
			wantDynamicVal:  "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			positional, err := NewParser(tt.args).ReadArgs(ctx)
			if err != nil {
				t.Fatalf("ReadArgs returned error: %v", err)
			}
			if len(positional) != len(tt.wantPositional) {
				t.Fatalf("positional arguments = %#v, want %#v", positional, tt.wantPositional)
			}
			for i := range positional {
				if positional[i] != tt.wantPositional[i] {
					t.Fatalf("positional arguments = %#v, want %#v", positional, tt.wantPositional)
				}
			}
			if tt.wantDynamicName == "" {
				return
			}
			flag := ctx.dynamicFlags.GetByName(tt.wantDynamicName)
			if flag == nil || flag.GetValue() != tt.wantDynamicVal {
				t.Fatalf("dynamic flag %q = %#v, want value %q", tt.wantDynamicName, flag, tt.wantDynamicVal)
			}
		})
	}
}

func TestParserContinuesToRejectEqualsSyntax(t *testing.T) {
	parser := NewParser([]string{"--Description=value"})

	_, err := parser.ReadArgs(NewContext())
	if err == nil {
		t.Fatal("expected missing value error, got nil")
	}
	if !strings.Contains(err.Error(), "--Description=value must set value.") {
		t.Fatalf("error = %q, want missing value error", err.Error())
	}
}

func TestParserTreatsEqualsAsLiteralFlagNameInPairedMode(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantName  string
		wantValue string
	}{
		{
			name:      "two equals-style tokens",
			args:      []string{"--A=1", "--B=2"},
			wantName:  "A=1",
			wantValue: "--B=2",
		},
		{
			name:      "equals-style name with plain value",
			args:      []string{"--A=1", "value"},
			wantName:  "A=1",
			wantValue: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			if _, err := NewParser(tt.args).ReadArgs(ctx); err != nil {
				t.Fatalf("ReadArgs returned error: %v", err)
			}
			flag := ctx.dynamicFlags.GetByName(tt.wantName)
			if flag == nil || flag.GetValue() != tt.wantValue {
				t.Fatalf("dynamic flag %q = %#v, want value %q", tt.wantName, flag, tt.wantValue)
			}
			if splitFlag := ctx.dynamicFlags.GetByName("A"); splitFlag != nil {
				t.Fatalf("unexpected split dynamic flag \"A\": %#v", splitFlag)
			}
		})
	}
}

func TestParserAllowsEqualsSyntaxInPairedValuePosition(t *testing.T) {
	ctx := NewContext()
	if _, err := NewParser([]string{"--Description", "--A=1"}).ReadArgs(ctx); err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}

	flag := ctx.dynamicFlags.GetByName("Description")
	if flag == nil || flag.GetValue() != "--A=1" {
		t.Fatalf("dynamic flag \"Description\" = %#v, want value %q", flag, "--A=1")
	}
}

func TestParserRejectsEqualsSyntaxForFixedFlags(t *testing.T) {
	_, err := NewParser([]string{"---region=ap-southeast-1"}).ReadArgs(NewContext())
	if err == nil {
		t.Fatal("expected unsupported fixed flag error, got nil")
	}
	if !strings.Contains(err.Error(), "---region=ap-southeast-1 is not supported") {
		t.Fatalf("error = %q, want unsupported fixed flag error", err.Error())
	}
}
