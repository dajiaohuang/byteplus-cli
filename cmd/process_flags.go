package cmd

import (
	"os"
)

// processFlagsResolution holds preprocessed CLI args and extracted system flags.
// Replaces volcengine's language-aware processLanguageResolution for English-only CLI.
type processFlagsResolution struct {
	args       []string
	fixedFlags map[string]string
	err        error
}

// Pass systemFlags explicitly so package init builds the registry before
// process argument preprocessing runs.
var processLanguageResolution = resolveProcessFlags(systemFlags)

func resolveProcessFlags(registry *systemFlagRegistry) processFlagsResolution {
	resolution, err := resolveSystemFlagsWithRegistry(os.Args[1:], registry)
	if err != nil {
		return processFlagsResolution{err: err}
	}
	return processFlagsResolution{
		args:       resolution.args,
		fixedFlags: resolution.fixedFlags,
	}
}
