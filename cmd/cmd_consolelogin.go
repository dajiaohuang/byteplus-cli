package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	loginCmd := newLoginCmd()
	logoutCmd := newLogoutCmd()
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}

func newLoginCmd() *cobra.Command {
	login := &ConsoleLogin{}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Byteplus Console",
		Long: `Authenticate with Byteplus Console and cache temporary STS credentials locally.

Supports three modes:
  - Local (default): Authorization Code + PKCE with a local callback
  - Remote (--remote): Authorization Code + PKCE with manual code input
  - Device code (--use-device-code): Device Authorization Grant with token polling

Use --no-browser with --use-device-code to skip opening the default browser.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return login.Login()
		},
	}

	cmd.SetUsageTemplate(loginUsageTemplate())

	// Register flags.
	cmd.Flags().StringVarP(&login.Profile, "profile", "p", "default", "Configuration profile name")
	cmd.Flags().StringVarP(&login.Region, "region", "r", "", "Region (prompts when omitted; empty input defaults to ap-southeast-1)")
	cmd.Flags().BoolVar(&login.Remote, "remote", false, "Enable cross-device (remote) login mode")
	cmd.Flags().BoolVar(&login.UseDeviceCode, "use-device-code", false, "Use the OAuth 2.0 Device Authorization Grant")
	cmd.Flags().BoolVar(&login.NoBrowser, "no-browser", false, "Do not automatically open the browser during device code login")
	cmd.Flags().StringVar(&login.EndpointURL, "endpoint-url", "https://signin.byteplus.com", "Override signin service endpoint URL")

	return cmd
}

func newLogoutCmd() *cobra.Command {
	logout := &ConsoleLogout{}

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of Byteplus Console and clear cached credentials",
		Long: `Remove locally cached login credentials for the specified profile or all profiles.

This is a purely local operation — no network requests are made to any server.
It deletes the cached STS token files from disk and clears the login_session
from the CLI configuration.

Modes:
  - Default: Logs out the specified profile (or "default" if not specified)
  - --all:   Logs out all configured console-login profiles with active sessions`,
		Example: `  # Log out the default profile
  bp logout

  # Log out a specific profile
  bp logout --profile my-profile

  # Log out all profiles and clear all cached login credentials
  bp logout --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return logout.Logout()
		},
	}

	cmd.SetUsageTemplate(loginUsageTemplate())

	// Register flags.
	cmd.Flags().StringVarP(&logout.Profile, "profile", "p", "default", "Configuration profile name")
	cmd.Flags().BoolVar(&logout.All, "all", false, "Log out all profiles and remove all cached login credentials")

	return cmd
}

func loginUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}
