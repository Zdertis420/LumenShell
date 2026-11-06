package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Lumen uses no network and sends no telemetry (FR-024).
func TestNoNetworkInGoPrograms(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join(repoRoot, "scripts", "src", "*", "*.go"))
	if len(files) == 0 {
		t.Fatal("no Go sources found")
	}
	banned := regexp.MustCompile(`^\s*"(net|net/[a-z/]+|crypto/tls|golang\.org/x/net[a-z/]*)"\s*$`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if banned.MatchString(line) {
				rel, _ := filepath.Rel(repoRoot, f)
				t.Errorf("%s:%d imports a network package: %s", rel, i+1, strings.TrimSpace(line))
			}
		}
	}
}

func TestNoNetworkInQML(t *testing.T) {
	var files []string
	for _, pattern := range []string{"shell.qml", "singletons/*.qml", "modules/*/*.qml", "examples/*/*.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		files = append(files, m...)
	}
	banned := regexp.MustCompile(`XMLHttpRequest|WebSocket|NetworkRequest|Quickshell\.Networking|https?://[a-z0-9.-]+\.[a-z]{2,}`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if banned.MatchString(line) {
				rel, _ := filepath.Rel(repoRoot, f)
				t.Errorf("%s:%d uses the network: %s", rel, i+1, strings.TrimSpace(line))
			}
		}
	}
}
