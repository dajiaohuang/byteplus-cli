package cmd

import (
	"strings"
)

// Flag prefix contract (normative — do not relax without updating docs/4-Usage.md):
//
//  1. Double dash `--name` is the ONLY public form of a system flag. Help,
//     completion, docs and examples must advertise `--name` only. Diagnostics may
//     echo a `---name` the user actually typed (systemFlagDisplayName), which is
//     feedback about the input rather than advertising the form.
//  2. Triple dash `---name` is NOT a public form. It exists solely as a
//     conflict escape: when the current action publishes an API parameter whose
//     name matches a system flag exactly (case-sensitive), `--name` is routed to
//     the API parameter and `---name` is the only way to still reach the system
//     flag. Published BytePlus metadata currently declares no such collision, so
//     the escape is a forward-compatibility path rather than a documented remedy.
//  3. `---name` must never be advertised anywhere user-facing: not in
//     localizedSystemFlagsHelp, not in shell completion, and not in the public
//     docs (README.MD, README.CN.MD, docs/*.md) — those may not even contain the
//     words "三横线" / "triple-dash". When an action's API parameter shadows a
//     system flag, public docs describe a non-flag workaround (environment
//     variable, downstream filtering) rather than the escape syntax. The escape
//     stays an internal compatibility path, guarded by
//     TestPublicDocsOnlyAdvertiseDoubleDashSystemFlags.
//  4. Every public system flag must keep a reachable `---name` escape route, so
//     the hatch already exists before a future metadata release introduces a
//     colliding API parameter name. Collisions only happen after an API action,
//     and there the Parser's legacyEscape route is the only one that fires:
//     shouldExtractSystemFlagWithRegistry returns false for state
//     afterAPIAction, so the preprocess route (resolveSystemFlags stripping
//     `---name` before Cobra) covers pre-action positions only. Every entry in
//     systemFlagDefs therefore keeps legacyEscape: true.
//
// Guards: TestSystemFlagHelpMatchesDefs (help text) and
// TestSystemFlagsAreExposedToCompletionWithoutLegacyAliases (shell completion)
// assert clauses (1) and (3) for the surfaces they cover;
// TestPublicDocsOnlyAdvertiseDoubleDashSystemFlags asserts (3) for the public
// docs listed there. Clauses (2) and (4) are conventions with no automated
// check yet: keep them in mind when editing systemFlagDefs.
//
// systemFlagDef is the single source of truth for CLI system flags.
// Parser routing, preprocess stripping, and completion registration derive
// from this table. Public help strings are English literals in localizedSystemFlagsHelp,
// guarded by TestSystemFlagHelpMatchesDefs.
type systemFlagDef struct {
	name string
	// public marks the flag as a double-dash system flag (publicSystemFlags).
	public bool
	// legacyEscape marks ---name as a parser-accepted conflict escape
	// (systemFlags.legacyEscapes, consumed by Parser.ReadArgs). This is the only
	// escape route that works after an API action, so clause (4) of the prefix
	// contract requires it on every public flag.
	legacyEscape bool
	// preprocess marks flags that resolveSystemFlags may strip before cobra/parser
	// (profile/region/endpoint). force/version/method stay in-place for Parser
	// so root `bp --version` and presence-only --force keep working.
	preprocess bool
	// presenceOnly: flag appears without consuming the next token when routed as system.
	presenceOnly bool
}

// systemFlagDefs order is the public help / supported-list order.
var systemFlagDefs = []systemFlagDef{
	{name: "profile", public: true, legacyEscape: true, preprocess: true},
	{name: "region", public: true, legacyEscape: true, preprocess: true},
	{name: "endpoint", public: true, legacyEscape: true, preprocess: true},
	{name: "version", public: true, legacyEscape: true, preprocess: false},
	{name: "method", public: true, legacyEscape: true, preprocess: false},
	{name: "force", public: true, legacyEscape: true, preprocess: false, presenceOnly: true},
	{name: "output", public: true, legacyEscape: true, preprocess: false},
	{name: "query", public: true, legacyEscape: true, preprocess: false},
}

type systemFlagRegistry struct {
	public           map[string]struct{}
	legacyEscapes    map[string]struct{}
	preprocessable   map[string]struct{}
	presenceOnly     map[string]struct{}
	supportedMessage string
}

// systemFlags is initialized as one value so process argument preprocessing
// never observes partially initialized lookup maps during package startup.
var systemFlags = newSystemFlagRegistry(systemFlagDefs)

func newSystemFlagRegistry(defs []systemFlagDef) *systemFlagRegistry {
	registry := &systemFlagRegistry{
		public:         make(map[string]struct{}, len(defs)),
		legacyEscapes:  make(map[string]struct{}, len(defs)),
		preprocessable: make(map[string]struct{}, len(defs)),
		presenceOnly:   make(map[string]struct{}),
	}

	publicNames := make([]string, 0, len(defs))
	for _, d := range defs {
		if d.public {
			registry.public[d.name] = struct{}{}
			publicNames = append(publicNames, "--"+d.name)
		}
		if d.legacyEscape {
			registry.legacyEscapes[d.name] = struct{}{}
		}
		if d.preprocess {
			registry.preprocessable[d.name] = struct{}{}
		}
		if d.presenceOnly {
			registry.presenceOnly[d.name] = struct{}{}
		}
	}
	registry.supportedMessage = strings.Join(publicNames, ", ")
	return registry
}

func isPresenceOnlyFixedFlag(name string) bool {
	_, ok := systemFlags.presenceOnly[name]
	return ok
}

// isSystemPresenceOnlyToken reports whether arg is a presence-only system flag
// in the current action context (no value token should be consumed).
func isSystemPresenceOnlyToken(arg string, actionParameters map[string]struct{}) bool {
	name, legacy := flagNameFromToken(arg)
	if name == "" || !isPresenceOnlyFixedFlag(name) {
		return false
	}
	if legacy {
		return true
	}
	_, conflict := actionParameters[name]
	return !conflict
}

// publicSystemFlagNames returns public system flag bare names in help order.
func publicSystemFlagNames() []string {
	out := make([]string, 0, len(systemFlagDefs))
	for _, d := range systemFlagDefs {
		if d.public {
			out = append(out, d.name)
		}
	}
	return out
}

// localizedSystemFlagsHelp returns public double-dash system flag help plus
// reserved double-dash controls. Triple-dash aliases are intentionally omitted.
func localizedSystemFlagsHelp() string {
	// Keep in lockstep with systemFlagDefs (name order and semantics).
	return `  --profile string     ` + "Use a configured profile only for this invocation." + `
  --region string      ` + "Override the region only for this invocation." + `
  --endpoint string    ` + "Override the endpoint only for this invocation." + `
  --version string     ` + "API version; uses metadata when omitted (required with --force for unlisted services)." + `
  --method string      ` + "HTTP method GET or POST; explicit value overrides metadata, else metadata, else GET." + `
  --force              ` + "Skip service/action metadata validation and force the call (presence-only; write --force alone, not --force true)." + `
  --output string      ` + "Set response output format (json|table|table-num|text|yaml|off). Default: json. table-num is table plus a row-number column. off still calls the API but skips response-dependent --query evaluation." + `
  --query string       ` + "JMESPath expression to filter/project the full response (paths usually start at Result.*) before formatting." + `

` + "Reserved double-dash controls (not API parameters):" + `
  --header string      ` + "Add a custom HTTP header as Name=Value; repeatable. Content-Type overrides metadata when set. Host/Authorization/Content-Length are blocked." + `
  --body string        ` + "JSON request body for application/json style calls; mutually exclusive with other API parameters."
}
