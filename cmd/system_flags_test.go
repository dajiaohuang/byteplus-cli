package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveSystemFlagsRejectsFlagsBeforeAction(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
	}{
		{name: "double dash before service", args: []string{"--region", "ap-southeast-1", "sts", "GetCallerIdentity"}, flag: "--region"},
		{name: "double dash between service and action", args: []string{"sts", "--region", "ap-southeast-1", "GetCallerIdentity"}, flag: "--region"},
		{name: "triple dash before service", args: []string{"---region", "ap-southeast-1", "sts", "GetCallerIdentity"}, flag: "---region"},
		{name: "triple dash between service and action", args: []string{"sts", "---region", "ap-southeast-1", "GetCallerIdentity"}, flag: "---region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveSystemFlags(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.flag+" must be specified after action") {
				t.Fatalf("position error = %v", err)
			}
		})
	}
}

func TestResolveSystemFlagsRejectsAnotherFlagAsValue(t *testing.T) {
	// Leading preprocessable flags are extracted before the command; a following
	// --flag must not be taken as the value.
	_, err := resolveSystemFlags([]string{
		"--region", "--profile", "prod", "configure", "list",
	})
	if err == nil || !strings.Contains(err.Error(), "--region requires a value") {
		t.Fatalf("missing value error = %v", err)
	}
}

func TestParserRoutesSystemFlagsAfterAction(t *testing.T) {
	c := NewContext()
	parser := NewParser([]string{
		"--region", "ap-southeast-1",
		"--profile", "prod",
		"--endpoint", "sts.byteplusapi.com",
		"--version", "2024-01-01",
		"--method", "POST",
		"--force",
	}, map[string]struct{}{})
	if _, err := parser.ReadArgs(c); err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}
	for name, want := range map[string]string{
		"region": "ap-southeast-1", "profile": "prod",
		"endpoint": "sts.byteplusapi.com",
		"version":  "2024-01-01", "method": "POST", "force": "true",
	} {
		flag := c.fixedFlags.GetByName(name)
		if flag == nil || flag.GetValue() != want {
			t.Fatalf("fixed flag %q = %#v, want %q", name, flag, want)
		}
		if dynamic := c.dynamicFlags.GetByName(name); dynamic != nil {
			t.Fatalf("system flag %q also entered dynamicFlags: %#v", name, dynamic)
		}
	}
}

func TestParserUsesExactActionParameterConflict(t *testing.T) {
	c := NewContext()
	params := map[string]struct{}{
		"profile": {}, "region": {}, "endpoint": {},
		"force": {}, "version": {}, "method": {},
	}
	parser := NewParser([]string{
		"--profile", "business-profile",
		"--region", "business-region",
		"--endpoint", "business-endpoint",
		"--force", "business-force",
		"--version", "business-version",
		"--method", "business-method",
		"--Region", "business-cased-region",
	}, params)
	if _, err := parser.ReadArgs(c); err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}
	for name, want := range map[string]string{
		"profile": "business-profile", "region": "business-region",
		"endpoint": "business-endpoint",
		"force":    "business-force", "version": "business-version", "method": "business-method",
		"Region": "business-cased-region",
	} {
		flag := c.dynamicFlags.GetByName(name)
		if flag == nil || flag.GetValue() != want {
			t.Fatalf("dynamic flag %q = %#v, want %q", name, flag, want)
		}
	}
	for name := range systemFlags.public {
		if c.fixedFlags.GetByName(name) != nil {
			t.Fatalf("conflicting --%s must not also enter fixedFlags", name)
		}
	}
}

func TestTripleDashEscapesForceVersionMethodConflicts(t *testing.T) {
	c := NewContext()
	params := map[string]struct{}{"force": {}, "version": {}, "method": {}}
	parser := NewParser([]string{
		"--force", "api-force",
		"--version", "api-version",
		"--method", "api-method",
		"---force",
		"---version", "2024-01-01",
		"---method", "POST",
	}, params)
	if _, err := parser.ReadArgs(c); err != nil {
		t.Fatalf("ReadArgs returned error: %v", err)
	}
	if !isForceEnabled(c) {
		t.Fatal("expected ---force system escape")
	}
	if got := c.fixedFlags.GetByName("version"); got == nil || got.GetValue() != "2024-01-01" {
		t.Fatalf("fixed version = %#v", got)
	}
	if got := c.fixedFlags.GetByName("method"); got == nil || got.GetValue() != "POST" {
		t.Fatalf("fixed method = %#v", got)
	}
	if got := c.dynamicFlags.GetByName("force"); got == nil || got.GetValue() != "api-force" {
		t.Fatalf("dynamic force = %#v", got)
	}
}

func TestResolveSystemFlagsLeavesActionScopedFlagsForParser(t *testing.T) {
	// profile/region/endpoint after an action stay in args for Parser
	// (only leading root preprocessable flags are stripped here).
	raw := []string{
		"sts", "GetCallerIdentity", "--region", "ap-southeast-1", "--profile", "prod",
	}
	resolution, err := resolveSystemFlags(raw)
	if err != nil {
		t.Fatalf("resolveSystemFlags returned error: %v", err)
	}
	if !reflect.DeepEqual(resolution.args, raw) {
		t.Fatalf("args = %#v, want unchanged %#v", resolution.args, raw)
	}
	if len(resolution.fixedFlags) != 0 {
		t.Fatalf("fixedFlags = %#v, want empty", resolution.fixedFlags)
	}
}

// TestResolveSystemFlagsDetailIsPresenceOnly ensures help meta --detail does not
// consume the following token during preprocess pairing.
func TestResolveSystemFlagsDetailIsPresenceOnly(t *testing.T) {
	// After action, preprocessable system flags are left for Parser; --detail must
	// not pair-consume the next token when scanning value-taking flags.
	args := []string{"ecs", "RunInstances", "-h", "--detail", "--region", "ap-southeast-1"}
	resolution, err := resolveSystemFlags(args)
	if err != nil {
		t.Fatalf("resolveSystemFlags: %v", err)
	}
	if !reflect.DeepEqual(resolution.args, args) {
		t.Fatalf("args=%#v, want unchanged", resolution.args)
	}
	if isValueTakingFlag("--detail", nil) {
		t.Fatal("isValueTakingFlag(--detail) must be false (help presence-only)")
	}
	if isValueTakingFlag("--force", nil) {
		t.Fatal("isValueTakingFlag(--force) must be false (presence-only system flag)")
	}
}

// TestResolveSystemFlagsDoesNotTreatNextFlagAsValue: a missing value must not
// swallow a following --flag.
func TestResolveSystemFlagsDoesNotTreatNextFlagAsValue(t *testing.T) {
	// After action nothing is extracted; args stay intact for Parser.
	args := []string{"sts", "GetCallerIdentity", "--profile", "--region", "ap-southeast-1"}
	resolution, err := resolveSystemFlags(args)
	if err != nil {
		t.Fatalf("resolveSystemFlags: %v", err)
	}
	if !reflect.DeepEqual(resolution.args, args) {
		t.Fatalf("args=%#v", resolution.args)
	}
}

func TestLegacyAndNewSystemFlagDuplicatesAreRejected(t *testing.T) {
	// Leading preprocessable flags before a non-API command: duplicate --region errors.
	_, err := resolveSystemFlags([]string{
		"--region", "a", "--region", "b", "configure", "list",
	})
	if err == nil || !strings.Contains(err.Error(), "--region cannot be specified more than once") {
		t.Fatalf("duplicate --region error = %v", err)
	}

	// Mixed double/triple dash for the same preprocessable name is also a duplicate.
	_, err = resolveSystemFlags([]string{
		"--region", "a", "---region", "b", "configure", "list",
	})
	if err == nil || !strings.Contains(err.Error(), "--region cannot be specified more than once") {
		t.Fatalf("duplicate --region/---region error = %v", err)
	}
}

func TestNestedJSONFieldDoesNotConflictWithSystemFlag(t *testing.T) {
	// Top-level system flag names must not be pulled from nested MetaTypes paths
	// (e.g. Payload.region). Only exact top-level API parameter names conflict.
	meta := &ByteplusMeta{
		ApiInfo: &ApiInfo{ContentType: "application/json"},
	}
	// Flattened nested field via GetRequestParams path is not what public names use
	// for conflict scan; empty basics → no conflicts.
	names := exposedActionParameterNames(meta, nil)
	if _, ok := names["region"]; ok {
		t.Fatal("empty meta must not expose region as an action parameter conflict")
	}
	if _, ok := names["profile"]; ok {
		t.Fatal("empty meta must not expose profile as an action parameter conflict")
	}
	// Parser: with no action params, --region routes to fixedFlags (system).
	c := NewContext()
	if _, err := NewParser([]string{"--region", "ap-southeast-1"}, names).ReadArgs(c); err != nil {
		t.Fatalf("ReadArgs: %v", err)
	}
	if got := c.fixedFlags.GetByName("region"); got == nil || got.GetValue() != "ap-southeast-1" {
		t.Fatalf("system --region not fixed: %#v", got)
	}
	if c.dynamicFlags.GetByName("region") != nil {
		t.Fatal("system --region must not enter dynamicFlags without API conflict")
	}
}

func TestSystemFlagsAreExposedToCompletionWithoutLegacyAliases(t *testing.T) {
	root := &cobra.Command{Use: "bp"}
	action := &cobra.Command{
		Use: "demo",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	registerActionSystemFlags(action, nil)
	root.AddCommand(action)

	for _, name := range publicSystemFlagNames() {
		if action.Flags().Lookup(name) == nil {
			t.Fatalf("action completion missing public system flag --%s", name)
		}
	}

	var completion bytes.Buffer
	if err := root.GenBashCompletion(&completion); err != nil {
		t.Fatalf("GenBashCompletion returned error: %v", err)
	}
	output := completion.String()
	for _, name := range publicSystemFlagNames() {
		if !strings.Contains(output, "--"+name) {
			t.Fatalf("completion missing public system flag --%s", name)
		}
	}
	for _, name := range publicSystemFlagNames() {
		alias := "---" + name
		if strings.Contains(output, alias) {
			t.Fatalf("completion exposes legacy system flag alias %s", alias)
		}
	}
}

func TestRegisterActionSystemFlagsSkipsExactAPIConflicts(t *testing.T) {
	action := &cobra.Command{Use: "demo"}
	conflicts := map[string]struct{}{"force": {}, "version": {}}
	registerActionSystemFlags(action, conflicts)

	for name := range conflicts {
		if action.Flags().Lookup(name) != nil {
			t.Fatalf("system flag --%s must not override an exact API parameter conflict", name)
		}
	}
	for _, name := range publicSystemFlagNames() {
		if _, conflict := conflicts[name]; conflict {
			continue
		}
		if action.Flags().Lookup(name) == nil {
			t.Fatalf("non-conflicting system flag --%s was not registered", name)
		}
	}
}

func TestSystemFlagHelpMatchesDefs(t *testing.T) {
	help := localizedSystemFlagsHelp()
	for _, name := range publicSystemFlagNames() {
		if !strings.Contains(help, "--"+name) {
			t.Fatalf("help missing --%s:\n%s", name, help)
		}
	}
	if strings.Contains(help, "--lang") {
		t.Fatal("English-only CLI must not advertise --lang")
	}
	if strings.Contains(help, "---") {
		t.Fatal("public help must not show triple-dash aliases")
	}
	for _, reserved := range []string{"--header", "--body"} {
		if !strings.Contains(help, reserved) {
			t.Fatalf("help missing reserved control %s:\n%s", reserved, help)
		}
	}
}

func TestPublicSystemFlagNamesOrder(t *testing.T) {
	got := publicSystemFlagNames()
	want := []string{"profile", "region", "endpoint", "version", "method", "force"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("publicSystemFlagNames=%v want %v", got, want)
	}
}
