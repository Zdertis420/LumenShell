package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Components must not hardcode colors (FR-014). Theme.qml owns the palette; everything
// else reads it. Hex literals and named colors are not allowed outside the data files.
func TestNoColorLiteralsInQML(t *testing.T) {
	var files []string
	for _, pattern := range []string{"shell.qml", "singletons/*.qml", "modules/*/*.qml", "modules/*/*.js", "examples/*/*.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no QML files found")
	}
	hex := regexp.MustCompile(`"#[0-9a-fA-F]{3,8}"`)
	named := regexp.MustCompile(`(?i)\bcolor\s*:\s*"(white|black|red|green|blue|gray|grey|yellow|orange|purple|pink|cyan|magenta)"`)
	rgba := regexp.MustCompile(`\b(Qt\.rgba|Qt\.hsla|Qt\.hsva|Qt\.darker|Qt\.lighter)\(`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if hex.MatchString(line) || named.MatchString(line) || rgba.MatchString(line) {
				rel, _ := filepath.Rel(repoRoot, f)
				t.Errorf("%s:%d hardcodes a color: %s", rel, i+1, trimmed)
			}
		}
	}
}
