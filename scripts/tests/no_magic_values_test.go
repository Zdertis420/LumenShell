package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Test-surface elements take spacing, radii, font sizes and durations from Style, not
// from numeric literals (FR-019). Allowed: the literals 0 and 1, and any number on a
// line that uses a Style value (for example "Style.spacing.xl * 3").
// If this test proves flaky, drop it and rely on review.
func TestNoMagicNumbersInTestSurface(t *testing.T) {
	var files []string
	for _, pattern := range []string{"modules/*/*.qml", "examples/*/*.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no bundled or example module QML files found")
	}
	str := regexp.MustCompile(`"(\\.|[^"\\])*"`)
	num := regexp.MustCompile(`(^|[^\w.])(\d+(\.\d+)?)\b`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			code := line
			if idx := strings.Index(code, "//"); idx >= 0 {
				code = code[:idx]
			}
			code = str.ReplaceAllString(code, `""`)
			if strings.Contains(code, "Style.") || strings.Contains(code, "api.style.") {
				continue
			}
			for _, m := range num.FindAllStringSubmatch(code, -1) {
				if m[2] == "0" || m[2] == "1" {
					continue
				}
				rel, _ := filepath.Rel(repoRoot, f)
				t.Errorf("%s:%d uses the number %s; take it from Style: %s", rel, i+1, m[2], strings.TrimSpace(line))
			}
		}
	}
}
