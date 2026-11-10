package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// The quote-aware word split and the terminal resolver, seen through grep (patterns and files
// are split as words) and open (the terminal and its flag).

func TestWordsQuotedPatternIsOneWord(t *testing.T) {
	e := newSearchEnv(t)
	e.file("a.txt", "one two\nonly one\n")
	d := e.run(map[string]any{"op": "grep", "query": `"one two" a.txt`, "limit": 10})
	if len(d.Rows) != 1 || d.Rows[0].Title != "one two" {
		t.Fatalf("got %+v", d.Rows)
	}
	d = e.run(map[string]any{"op": "grep", "query": `'one two' a.txt`, "limit": 10})
	if len(d.Rows) != 1 {
		t.Fatalf("single quotes: got %+v", d.Rows)
	}
}

func TestWordsUnbalancedQuoteIsLiteral(t *testing.T) {
	e := newSearchEnv(t)
	e.file("a.txt", "it's here\n")
	d := e.run(map[string]any{"op": "grep", "query": `it's a.txt`, "limit": 10})
	if len(d.Rows) != 1 {
		t.Fatalf("got %+v", d)
	}
}

func TestTerminalFlags(t *testing.T) {
	cases := []struct {
		term string
		flag string
		want []string // arguments before the editor
	}{
		{"kitty", "", []string{}},
		{"foot", "", []string{"-e"}},
		{"alacritty", "", []string{"-e"}},
		{"wezterm", "", []string{"start", "--"}},
		{"gnome-terminal", "", []string{"--"}},
		{"foot", "--run", []string{"--run"}},
	}
	for _, c := range cases {
		e := newSearchEnv(t)
		rec := e.fake(c.term)
		file := e.file("f.txt", "x")
		e.set("EDITOR=myedit", "TERMINAL="+c.term)
		e.run(map[string]any{"op": "open", "path": file, "terminalExecFlag": c.flag})
		got := e.args(rec)
		want := append(append([]string{}, c.want...), "myedit", file)
		if !equalStrings(got, want) {
			t.Errorf("%s: got %v, want %v", c.term, got, want)
		}
	}
}

func TestTerminalSettingBeatsVariable(t *testing.T) {
	e := newSearchEnv(t)
	fromSetting := e.fake("fromsetting")
	fromVar := e.fake("fromvar")
	file := e.file("f.txt", "x")
	e.set("EDITOR=ed", "TERMINAL=fromvar")
	e.run(map[string]any{"op": "open", "path": file, "terminal": "fromsetting"})
	e.args(fromSetting)
	e.neverRan(fromVar)
}

func TestTerminalWithArguments(t *testing.T) {
	e := newSearchEnv(t)
	rec := e.fake("kitty")
	file := e.file("f.txt", "x")
	e.set("EDITOR=ed")
	e.run(map[string]any{"op": "open", "path": file, "terminal": "kitty --class prism"})
	got := e.args(rec)
	want := []string{"--class", "prism", "ed", file}
	if !equalStrings(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestWordsHomeExpansion(t *testing.T) {
	e := newSearchEnv(t)
	p := e.file("notes/a.txt", "needle\n")
	d := e.run(map[string]any{"op": "grep", "query": "needle ~/notes/a.txt", "limit": 10})
	if len(d.Rows) != 1 || filepath.Dir(d.Rows[0].Key) == "" {
		t.Fatalf("got %+v", d)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
}
