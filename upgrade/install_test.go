package upgrade

import (
	"strings"
	"testing"
)

func TestNormalizeInstallPath(t *testing.T) {
	got := normalizeInstallPath(`C:\Users\x\node_modules\@byteplus\cli\bin\bp.exe`)
	if !strings.Contains(got, "/node_modules/@byteplus/cli/") {
		t.Fatalf("normalize: %q", got)
	}
	if got != strings.ToLower(got) {
		t.Fatalf("expected lower case: %q", got)
	}
}

func TestDetectInstall_PathTable(t *testing.T) {
	// 隔离 env，只测路径启发式
	origEnv := installLookupEnv
	defer func() { installLookupEnv = origEnv }()
	installLookupEnv = func(string) string { return "" }

	tests := []struct {
		path string
		want Method
	}{
		// Homebrew macOS 常见布局
		{"/opt/homebrew/Cellar/byteplus-cli/1.0.49/bin/bp", MethodHomebrew},
		{"/opt/homebrew/opt/byteplus-cli/bin/bp", MethodHomebrew},
		{"/usr/local/Cellar/byteplus-cli/1.0.49/bin/bp", MethodHomebrew},
		{"/usr/local/homebrew/bin/bp", MethodHomebrew},
		// Linux 上 Homebrew / Linuxbrew
		{"/home/linuxbrew/.linuxbrew/Cellar/byteplus-cli/1.0.49/bin/bp", MethodHomebrew},
		{"/home/linuxbrew/.linuxbrew/bin/bp", MethodHomebrew},
		// 同时含 linuxbrew 与 homebrew 字样时仍判为 brew 族
		{"/linuxbrew/prefix/homebrew/name/bp", MethodHomebrew},
		// npm
		{"/usr/local/lib/node_modules/@byteplus/cli/bin/bp", MethodNPM},
		{`C:\Users\me\AppData\Roaming\npm\node_modules\@byteplus\cli\bin\bp.exe`, MethodNPM},
		// standalone（含易误判的否定用例）
		{"/usr/local/bin/bp", MethodStandalone},
		{"/tmp/bp", MethodStandalone},
		{`C:\tools\bp.exe`, MethodStandalone},
		{"/opt/not-homebrew/bin/bp", MethodStandalone},
		{"/home/me/src/homebrew-tools/bin/bp", MethodStandalone},
		{"/tmp/homebrew-mirror/bp", MethodStandalone},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			info := detectInstall(tt.path)
			if info.Method != tt.want {
				t.Fatalf("path %q: got %q want %q (detectedBy=%s)", tt.path, info.Method, tt.want, info.DetectedBy)
			}
			if info.ExecPath != tt.path {
				t.Fatalf("ExecPath: got %q", info.ExecPath)
			}
		})
	}
}

func TestPathHasSegment(t *testing.T) {
	if !pathHasSegment("/opt/homebrew/bin/bp", "homebrew") {
		t.Fatal("expected homebrew segment")
	}
	if pathHasSegment("/opt/not-homebrew/bin/bp", "homebrew") {
		t.Fatal("not-homebrew must not match homebrew segment")
	}
	if !pathHasSegment("/usr/local/cellar/byteplus-cli/1/bin/bp", "cellar") {
		t.Fatal("expected cellar segment")
	}
}

func TestDetectInstall_EnvOverridesPath(t *testing.T) {
	origEnv := installLookupEnv
	defer func() { installLookupEnv = origEnv }()
	installLookupEnv = func(k string) string {
		if k == EnvInstallMethod {
			return "npm"
		}
		return ""
	}
	// Path looks standalone, env forces npm
	info := detectInstall("/tmp/bp")
	if info.Method != MethodNPM || info.DetectedBy != DetectedByEnv {
		t.Fatalf("got %+v", info)
	}
}

func TestDetectInstall_InvalidEnvFallsThrough(t *testing.T) {
	origEnv := installLookupEnv
	defer func() { installLookupEnv = origEnv }()
	installLookupEnv = func(k string) string {
		if k == EnvInstallMethod {
			return "not-a-real-method"
		}
		return ""
	}

	info := detectInstall("/usr/local/lib/node_modules/@byteplus/cli/bin/bp")
	if info.Method != MethodNPM || info.DetectedBy != DetectedByPath {
		t.Fatalf("got %+v", info)
	}
}

func TestDetectInstall_EnvWinsOverPath(t *testing.T) {
	origEnv := installLookupEnv
	defer func() { installLookupEnv = origEnv }()
	installLookupEnv = func(k string) string {
		if k == EnvInstallMethod {
			return "standalone"
		}
		return ""
	}
	// Path looks like npm, env forces standalone
	info := detectInstall("/usr/local/lib/node_modules/@byteplus/cli/bin/bp")
	if info.Method != MethodStandalone || info.DetectedBy != DetectedByEnv {
		t.Fatalf("got %+v", info)
	}
}

func TestDetectInstall_LinuxbrewEnvAlias(t *testing.T) {
	origEnv := installLookupEnv
	defer func() { installLookupEnv = origEnv }()
	installLookupEnv = func(k string) string {
		if k == EnvInstallMethod {
			return "linuxbrew"
		}
		return ""
	}
	info := detectInstall("/tmp/bp")
	if info.Method != MethodHomebrew || info.DetectedBy != DetectedByEnv {
		t.Fatalf("got %+v", info)
	}
}

func TestNPMUpgradeCommand(t *testing.T) {
	if got := NPMUpgradeCommand(""); got != "npm install -g @byteplus/cli@latest" {
		t.Fatalf("latest: %q", got)
	}
	if got := NPMUpgradeCommand("1.0.49"); got != "npm install -g @byteplus/cli@1.0.49" {
		t.Fatalf("pin: %q", got)
	}
	// Display command and exec package-spec share npmPackageSpec.
	if got := npmPackageSpec(""); got != "@byteplus/cli@latest" {
		t.Fatalf("spec latest: %q", got)
	}
	if got := npmPackageSpec("v1.0.49"); got != "@byteplus/cli@1.0.49" {
		t.Fatalf("spec pin: %q", got)
	}
	if NPMUpgradeCommand("1.0.49") != "npm install -g "+npmPackageSpec("1.0.49") {
		t.Fatal("NPMUpgradeCommand must be npm install -g + npmPackageSpec")
	}
}
