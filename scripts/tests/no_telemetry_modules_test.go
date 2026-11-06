package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The module feature adds no network use and no telemetry (FR-026). The Foundation
// guards (no_network_test.go) scan the same patterns; this test makes sure that the
// module code is among the files they cover, so the coverage cannot drop silently.
func TestModuleCodeIsFreeOfNetworkUse(t *testing.T) {
	files := []string{
		"scripts/src/modules/main.go", "scripts/src/modules/manifest.go",
		"scripts/src/modules/scan.go", "scripts/src/modules/schema.go",
		"singletons/Modules.qml", "singletons/ModuleApi.qml",
		"modules/test-surface/Main.qml", "examples/hello-module/Main.qml",
	}
	banned := regexp.MustCompile(`"net"|"net/[a-z/]+"|XMLHttpRequest|WebSocket|NetworkRequest|Quickshell\.Networking|https?://[a-z0-9.-]+\.[a-z]{2,}`)
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(repoRoot, f))
		if err != nil {
			t.Errorf("%s must exist: %v", f, err)
			continue
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if banned.MatchString(line) {
				t.Errorf("%s:%d uses the network: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// The user modules directory is never created: scanning a missing one leaves no trace.
func TestScanDoesNotCreateTheUserModulesDirectory(t *testing.T) {
	base := t.TempDir()
	missing := filepath.Join(base, "lumen", "modules")
	scanDirs(t, "", missing)
	entries, _ := os.ReadDir(base)
	if len(entries) != 0 {
		t.Fatalf("scan created %v", entries)
	}
}
