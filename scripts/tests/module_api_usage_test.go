package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// documentedAPIMembers reads the module interface contracts (1.0 in specs/002-modules and
// the 1.1 additions in specs/003-bar) and returns every `api.xxx` member named in
// backticks, without call arguments.
func documentedAPIMembers(t *testing.T) map[string]bool {
	t.Helper()
	re := regexp.MustCompile("`(api\\.[A-Za-z_][A-Za-z0-9_.]*)")
	members := map[string]bool{}
	for _, contract := range []string{
		filepath.Join("specs", "002-modules", "contracts", "module-api.md"),
		filepath.Join("specs", "003-bar", "contracts", "module-api-1.1.md"),
	} {
		b, err := os.ReadFile(filepath.Join(repoRoot, contract))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			members[strings.TrimRight(m[1], ".")] = true
		}
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

// The documentation (docs/module-api.md and docs/bar.md) names every api member the bundled
// modules use, so what the code uses and what authors read agree (SC-013).
func TestDocsNameTheMembersBundledModulesUse(t *testing.T) {
	docs := map[string]bool{}
	re := regexp.MustCompile("`(api\\.[A-Za-z_][A-Za-z0-9_.]*)")
	for _, f := range []string{"module-api.md", "bar.md"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, "docs", f))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			docs[strings.TrimRight(m[1], ".")] = true
		}
	}
	var files []string
	for _, pattern := range []string{"modules/*/*.qml", "examples/*/*.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		files = append(files, m...)
	}
	refRe := regexp.MustCompile(`\bapi\.[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*`)
	for _, f := range files {
		b, _ := os.ReadFile(f)
		rel, _ := filepath.Rel(repoRoot, f)
		for _, ref := range refRe.FindAllString(stripQML(string(b)), -1) {
			if !covered(docs, ref) {
				t.Errorf("%s uses %s, which docs/module-api.md and docs/bar.md do not document", rel, ref)
			}
		}
	}
}

// The members of `slot` that items use are documented in docs/bar.md, and the built-in items
// do not open popups (FR-019).
func TestItemsUseOnlyDocumentedSlotMembers(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs", "bar.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := map[string]bool{}
	for _, m := range regexp.MustCompile("`(slot\\.[A-Za-z_][A-Za-z0-9_.]*)").FindAllStringSubmatch(string(b), -1) {
		documented[strings.TrimRight(m[1], ".")] = true
	}
	if len(documented) < 7 {
		t.Fatalf("docs/bar.md documents only %d slot members", len(documented))
	}
	var items []string
	// Every QML file of the item modules (the item and ItemLook.qml), and the example's.
	for _, pattern := range []string{"modules/bar-*/*.qml", "examples/*/Item.qml"} {
		m, _ := filepath.Glob(filepath.Join(repoRoot, pattern))
		items = append(items, m...)
	}
	if len(items) < 4 {
		t.Fatalf("expected the built-in items and the example, found %v", items)
	}
	slotRe := regexp.MustCompile(`\bslot\.[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*`)
	for _, f := range items {
		src, _ := os.ReadFile(f)
		rel, _ := filepath.Rel(repoRoot, f)
		for _, ref := range slotRe.FindAllString(stripQML(string(src)), -1) {
			if !covered(documented, ref) {
				t.Errorf("%s uses %s, which docs/bar.md does not document", rel, ref)
			}
			if strings.HasPrefix(rel, "modules"+string(filepath.Separator)) && strings.HasPrefix(ref, "slot.popup") {
				t.Errorf("%s uses %s; built-in items must not open popups in this feature (FR-019)", rel, ref)
			}
		}
	}
}
