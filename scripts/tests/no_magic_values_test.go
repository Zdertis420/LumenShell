package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A visual property of a bundled or example module is never given a number (not even 0 or 1)
// in code: its value is a design token, one of the module's declared settings, or a named
// constant marked "documented constant" (FR-017 of specs/004-bar-customization-styles, and
// the quality gate of Principle VII). The test above rejects every literal but 0 and 1; this
// one rejects 0 and 1 too where they are the value of radius, spacing, a margin, a padding, a
// border width, a text size or an implicit size.
func TestNoFixedVisualValuesInModules(t *testing.T) {
	var files []string
	for _, pattern := range []string{"modules/*/*.qml", "examples/*/*.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no bundled or example module QML files found")
	}
	visual := regexp.MustCompile(`^\s*(radius|spacing|border\.width|anchors\.margins|margins\.(top|bottom|left|right)|(left|right|top|bottom)?[mM]argin|(left|right|top|bottom)?[pP]adding|font\.pixelSize|implicitWidth|implicitHeight)\s*:\s*-?\d+(\.\d+)?\s*(//.*)?$`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "// documented constant") {
				continue
			}
			if visual.MatchString(line) {
				rel, _ := filepath.Rel(repoRoot, f)
				t.Errorf("%s:%d fixes a visual value in code; use a token or a setting: %s", rel, i+1, strings.TrimSpace(line))
			}
		}
	}
}

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
			// A literal that is a fact, not a design value (for example the 100 in "percent"),
			// is declared once with this marker on its line.
			if strings.Contains(line, "// documented constant") {
				continue
			}
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
