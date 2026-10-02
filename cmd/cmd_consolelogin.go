package cmd

import (
	"fmt"

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

Login uses the OAuth 2.0 Device Authorization Grant: the CLI prints a
verification URL and a user code, then polls for the token until you finish
authorizing on any device.

Use --no-browser to skip opening the default browser.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			warnDeprecatedRemoteFlag(cmd)
			return login.Login()
		},
	}

	cmd.SetUsageTemplate(loginUsageTemplate())

	// Register flags.
	cmd.Flags().StringVarP(&login.Profile, "profile", "p", "default", "Configuration profile name")
	cmd.Flags().StringVarP(&login.Region, "region", "r", "", "Region (prompts when omitted; empty input defaults to ap-southeast-1)")
	cmd.Flags().Bool("remote", false, "Deprecated: cross-device login is now the only mode")
	cmd.Flags().BoolVar(&login.NoBrowser, "no-browser", false, "Do not automatically open the browser during login")
	cmd.Flags().StringVar(&login.EndpointURL, "endpoint-url", "https://signin.byteplus.com", "Override signin service endpoint URL")

	// --remote shipped in released versions and is kept as a hidden no-op so
	// invocations do not fail with "unknown flag". This preserves flag parsing
	// compatibility only; scripts that automate the old authorization-code
	// prompts or stdin input must migrate to the device-code flow. Nothing reads
	// its value. pflag's MarkDeprecated is deliberately avoided: it prints its
	// own notice on stdout, which would pollute output that callers already parse.
	_ = cmd.Flags().MarkHidden("remote")

	return cmd
}

// warnDeprecatedRemoteFlag notifies scripts that still pass --remote. It writes
// to stderr so that callers parsing stdout are unaffected.
func warnDeprecatedRemoteFlag(cmd *cobra.Command) {
	if !cmd.Flags().Changed("remote") {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Warning: --remote is deprecated and ignored; 'bp login' always uses cross-device device code authorization.")
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
