package cmd

// Language is retained as a type alias for test helpers only.
// The CLI is English-only; language selection has no runtime effect.
type Language string

const (
	LanguageEnglish           Language = "EN"
	LanguageSimplifiedChinese Language = "ZH"
)

// currentLanguage is always English in this product.
var currentLanguage = LanguageEnglish

// setCurrentLanguage is a no-op for English-only CLI.
func setCurrentLanguage(language Language) {
	currentLanguage = LanguageEnglish
}

// setLanguageForTest is a no-op restore function for tests ported from volcengine-cli.
func setLanguageForTest(language Language) func() {
	return func() {}
}

// resolveLanguage is a test helper stub: strips system flags, ignores language env.
func resolveLanguage(args []string, lookupEnv func(string) (string, bool)) ([]string, Language, error) {
	resolution, err := resolveSystemFlagsWithRegistry(args, systemFlags)
	if err != nil {
		return nil, LanguageEnglish, err
	}
	return resolution.args, LanguageEnglish, nil
}

// mapEnvironment adapts a map to os.LookupEnv-style for tests.
func mapEnvironment(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}
