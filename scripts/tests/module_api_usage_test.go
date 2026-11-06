package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// documentedAPIMembers reads specs/002-modules/contracts/module-api.md and returns every
// `api.xxx` member named in backticks, without call arguments.
func documentedAPIMembers(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "specs", "002-modules", "contracts", "module-api.md"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("`(api\\.[A-Za-z_][A-Za-z0-9_.]*)")
	members := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		members[strings.TrimRight(m[1], ".")] = true
	}
	if len(members) < 15 {
		t.Fatalf("found only %d documented members; did the contract format change?", len(members))
	}
	return members
}

// stripQML removes comments and string literals, so words in them are not mistaken for code.
func stripQML(src string) string {
	block := regexp.MustCompile(`(?s)/\*.*?\*/`)
	line := regexp.MustCompile(`//[^\n]*`)
	str := regexp.MustCompile(`"(\\.|[^"\\])*"`)
	return str.ReplaceAllString(line.ReplaceAllString(block.ReplaceAllString(src, ""), ""), `""`)
}

// A documented member covers itself and anything below it (api.theme covers api.theme.background).
func covered(members map[string]bool, ref string) bool {
	parts := strings.Split(ref, ".")
	for n := len(parts); n >= 2; n-- {
		if members[strings.Join(parts[:n], ".")] {
			return true
		}
	}
	return false
}

// SC-010: bundled modules and examples use only documented members of the module
// interface, and never reach for Lumen's singletons directly.
func TestBundledModulesUseOnlyTheDocumentedAPI(t *testing.T) {
	members := documentedAPIMembers(t)
	var files []string
	for _, pattern := range []string{"modules/*/*.qml", "examples/*/*.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		files = append(files, m...)
	}
	surface := filepath.Join(repoRoot, "modules", "test-surface", "Main.qml")
	if _, err := os.Stat(surface); err != nil {
		t.Fatalf("the bundled test-surface module must exist: %v", err)
	}
	refRe := regexp.MustCompile(`\bapi\.[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*`)
	singletonRe := regexp.MustCompile(`\b(Backend|Config|Modules|Log|Theme|Style)\b`)
	used := map[string]bool{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		code := stripQML(string(b))
		rel, _ := filepath.Rel(repoRoot, f)
		for _, ref := range refRe.FindAllString(code, -1) {
			used[ref] = true
			if !covered(members, ref) {
				t.Errorf("%s uses %s, which is not in the documented module interface", rel, ref)
			}
		}
		for _, name := range singletonRe.FindAllString(code, -1) {
			t.Errorf("%s refers to the singleton %s directly; use the api object instead", rel, name)
		}
	}
	var list []string
	for r := range used {
		list = append(list, r)
	}
	sort.Strings(list)
	t.Logf("members used by bundled modules and examples: %s", strings.Join(list, ", "))
}
