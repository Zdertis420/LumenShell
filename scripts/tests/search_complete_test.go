package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The complete operation (specs/006-prism-modes, FR-038 to FR-042): Tab completion of the last
// directory word (files) or file word (grep). The word spans it relies on are covered through
// the quoted-word cases.

type completion struct {
	Query   string `json:"query"`
	Cursor  int    `json:"cursor"`
	Changed bool   `json:"changed"`
	Message *struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	} `json:"message"`
}

func complete(t *testing.T, e *searchEnv, mode, query string) completion {
	t.Helper()
	env, r := e.runRaw(map[string]any{"op": "complete", "mode": mode, "query": query})
	if !env.OK {
		t.Fatalf("complete failed: %s", r.stdout)
	}
	var c completion
	if err := json.Unmarshal(env.Data, &c); err != nil {
		t.Fatalf("bad data: %v\n%s", err, r.stdout)
	}
	return c
}

func mkdirs(t *testing.T, e *searchEnv, rels ...string) {
	t.Helper()
	for _, r := range rels {
		if err := os.MkdirAll(filepath.Join(e.home, r), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompleteSingleDirectory(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "Documents", "Downloads")
	c := complete(t, e, "files", "report ~/Doc")
	if !c.Changed || c.Query != "report ~/Documents/" || c.Cursor != len([]rune(c.Query)) || c.Message != nil {
		t.Fatalf("got %+v", c)
	}
	// "~" alone only lacks the slash.
	c = complete(t, e, "files", "report ~")
	if c.Query != "report ~/" || !c.Changed {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteSeveralExtendsAndListsThem(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "Documents", "Downloads", "Desktop")
	c := complete(t, e, "files", "x ~/D")
	if c.Changed || c.Query != "x ~/D" {
		t.Fatalf("D is already the common start: %+v", c)
	}
	if c.Message == nil || c.Message.Kind != "info" || !strings.HasPrefix(c.Message.Text, "3 matches: Desktop/, Documents/, Downloads/") {
		t.Fatalf("message: %+v", c.Message)
	}
	mkdirs(t, e, "Projects/alpha", "Projects/alpine", "Projects/beta")
	c = complete(t, e, "files", "~/Projects/al")
	if !c.Changed || c.Query != "~/Projects/alp" {
		t.Fatalf("common start: %+v", c)
	}
}

func TestCompleteNamesInTheMessageAreLimited(t *testing.T) {
	e := newSearchEnv(t)
	for i := 0; i < 12; i++ {
		mkdirs(t, e, fmt.Sprintf("many/item%02d", i))
	}
	c := complete(t, e, "files", "~/many/i")
	if c.Message == nil || !strings.HasPrefix(c.Message.Text, "12 matches: ") || !strings.HasSuffix(c.Message.Text, ", …") {
		t.Fatalf("message: %+v", c.Message)
	}
	if strings.Count(c.Message.Text, "item") != 8 {
		t.Fatalf("expected 8 names: %s", c.Message.Text)
	}
}

func TestCompleteNoMatchAndMissingDirectory(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "Documents")
	c := complete(t, e, "files", "~/Zzz")
	if c.Changed || c.Query != "~/Zzz" || c.Message == nil || c.Message.Text != "no match" {
		t.Fatalf("got %+v", c)
	}
	c = complete(t, e, "files", "~/Nowhere/x")
	if c.Changed || c.Message == nil || c.Message.Kind != "error" || !strings.Contains(c.Message.Text, "directory not found") {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteUnreadableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	e := newSearchEnv(t)
	mkdirs(t, e, "locked")
	p := filepath.Join(e.home, "locked")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p, 0o755)
	c := complete(t, e, "files", "~/locked/a")
	if c.Changed || c.Message == nil || c.Message.Kind != "error" {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteHiddenOnlyAfterADot(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, ".config", ".cache", "Docs", "Dots")
	c := complete(t, e, "files", "~/")
	if c.Message == nil || strings.Contains(c.Message.Text, ".config") {
		t.Fatalf("hidden names must not be offered: %+v", c)
	}
	c = complete(t, e, "files", "~/.co")
	if c.Query != "~/.config/" {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteCase(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "Documents")
	c := complete(t, e, "files", "~/doc")
	if c.Query != "~/Documents/" {
		t.Fatalf("ignoring case when nothing matches exactly: %+v", c)
	}
	mkdirs(t, e, "doc-old")
	c = complete(t, e, "files", "~/doc")
	if c.Query != "~/doc-old/" {
		t.Fatalf("exact case wins: %+v", c)
	}
}

func TestCompleteQuotesNamesWithSpaces(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "My Docs")
	c := complete(t, e, "files", "~/My")
	if c.Query != "\"~/My Docs/\"" {
		t.Fatalf("got %+v", c)
	}
	if want := len([]rune(c.Query)) - 1; c.Cursor != want {
		t.Fatalf("cursor %d, want %d (before the closing quote)", c.Cursor, want)
	}
	// A single file ends the word: the cursor stays at the end, after the closing quote.
	e.file("Odd Name.txt", "x")
	c = complete(t, e, "grep", "needle Odd")
	if c.Query != "needle \"Odd Name.txt\"" || c.Cursor != len([]rune(c.Query)) {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteSkipsNamesWithADoubleQuote(t *testing.T) {
	e := newSearchEnv(t)
	e.file("say\"hi\".txt", "x")
	c := complete(t, e, "grep", "needle say")
	if c.Changed || c.Message == nil || !strings.Contains(c.Message.Text, "double quote") {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteGrepFilesAndDirectories(t *testing.T) {
	e := newSearchEnv(t)
	e.file("notes/report-note.md", "needle")
	mkdirs(t, e, "notes2")
	c := complete(t, e, "grep", "TODO notes/re")
	if c.Query != "TODO notes/report-note.md" {
		t.Fatalf("a file: %+v", c)
	}
	c = complete(t, e, "grep", "TODO notes2")
	if c.Query != "TODO notes2/" {
		t.Fatalf("a directory: %+v", c)
	}
	// Several candidates in the home: relative words are taken from the home.
	e.file("alpha.txt", "x")
	e.file("alpine.txt", "x")
	c = complete(t, e, "grep", "TODO al")
	if c.Query != "TODO alp" {
		t.Fatalf("relative word: %+v", c)
	}
}

func TestCompleteChangesNothingElsewhere(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "Documents")
	for _, tc := range []struct{ mode, q string }{
		{"grep", "Doc"},            // the pattern word
		{"grep", "TODO notes/ "},   // trailing space
		{"files", "Doc"},           // not a path word
		{"files", "report "},       // trailing space
		{"files", "~/Documents/ "}, // trailing space after a path
		{"apps", "~/Doc"},          // another mode
		{"commands", "~/Doc"},      // another mode
		{"web", "~/Doc"},           // another mode
		{"default", "~/Doc"},       // the default mode
		{"files", ""},              // nothing typed
	} {
		c := complete(t, e, tc.mode, tc.q)
		if c.Changed || c.Query != tc.q || c.Message != nil {
			t.Errorf("%s %q: got %+v", tc.mode, tc.q, c)
		}
	}
}

func TestCompleteFilesModeOffersDirectoriesOnly(t *testing.T) {
	e := newSearchEnv(t)
	e.file("Docs.txt", "x")
	c := complete(t, e, "files", "~/Doc")
	if c.Changed || c.Message == nil || c.Message.Text != "no match" {
		t.Fatalf("a file is not a directory word: %+v", c)
	}
}

func TestCompleteReadsAtMost5000Entries(t *testing.T) {
	e := newSearchEnv(t)
	dir := filepath.Join(e.home, "big")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5200; i++ {
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("f%05d", i)))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	ms := runProgramTimed(e, map[string]any{"op": "complete", "mode": "grep", "query": "x ~/big/f0"})
	if ms > 1500 {
		t.Fatalf("took %d ms", ms)
	}
	c := complete(t, e, "grep", "x ~/big/f0")
	if c.Message == nil || !strings.Contains(c.Message.Text, "matches") {
		t.Fatalf("got %+v", c)
	}
}

func TestCompleteLogsCountsNeverNames(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "SecretFolderName")
	e.set("LUMEN_LOG_LEVEL=debug")
	_, r := e.runRaw(map[string]any{"op": "complete", "mode": "files", "query": "~/Secret"})
	if strings.Contains(r.stderr, "Secret") {
		t.Fatalf("the log carries a name: %s", r.stderr)
	}
	if !strings.Contains(r.stderr, "candidates") {
		t.Fatalf("no count logged: %s", r.stderr)
	}
}

func TestCompleteQuotedWordIsReplacedWhole(t *testing.T) {
	e := newSearchEnv(t)
	mkdirs(t, e, "My Docs/Reports")
	c := complete(t, e, "files", "name \"~/My Docs/Re\"")
	if c.Query != "name \"~/My Docs/Reports/\"" {
		t.Fatalf("got %+v", c)
	}
}
