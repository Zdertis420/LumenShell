package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func chmodExec(p string) error { return os.Chmod(p, 0o755) }

// The scope operation (spec FR-014, research R9) and the quote-aware word split it uses.

func scopeOf(t *testing.T, e *searchEnv, query string) searchData {
	t.Helper()
	return e.run(map[string]any{"op": "scope", "query": query})
}

func TestScopeNoDirectory(t *testing.T) {
	e := newSearchEnv(t)
	d := scopeOf(t, e, "report final")
	if d.Dir != "" || d.Name != "report final" || d.Hint != "" {
		t.Fatalf("got %+v", d)
	}
}

func TestScopeDirectoryForms(t *testing.T) {
	e := newSearchEnv(t)
	docs := filepath.Join(e.home, "Documents", "work")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"report ~/Documents/work", "~/Documents/work report", "report ./Documents/work", "report " + docs} {
		d := scopeOf(t, e, q)
		if d.Dir != docs || d.Name != "report" {
			t.Errorf("%q: got dir=%q name=%q", q, d.Dir, d.Name)
		}
	}
	d := scopeOf(t, e, "report ~")
	if d.Dir != e.home || d.Name != "report" {
		t.Errorf("~ alone: got %+v", d)
	}
}

func TestScopeQuotedPathWithSpaces(t *testing.T) {
	e := newSearchEnv(t)
	dir := filepath.Join(e.home, "My Files")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d := scopeOf(t, e, `notes "~/My Files"`)
	if d.Dir != dir || d.Name != "notes" {
		t.Fatalf("got %+v", d)
	}
}

func TestScopeMissingDirectoryStaysInName(t *testing.T) {
	e := newSearchEnv(t)
	d := scopeOf(t, e, "report ~/nope")
	if d.Dir != "" || d.Name != "report ~/nope" {
		t.Fatalf("got %+v", d)
	}
	if d.Hint != "directory not found: ~/nope" {
		t.Fatalf("hint = %q", d.Hint)
	}
}

func TestScopeFileIsNotADirectory(t *testing.T) {
	e := newSearchEnv(t)
	e.file("a.txt", "x")
	d := scopeOf(t, e, "report ~/a.txt")
	if d.Dir != "" || d.Hint != "" {
		t.Fatalf("a path to a file is plain text, got %+v", d)
	}
}
