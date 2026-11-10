package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Opening files and grep matches in EDITOR (inside a terminal) and the web search in BROWSER
// (spec FR-005, FR-005a, FR-005b, FR-018a, FR-026, FR-028, FR-029; research R10, R13).

func TestOpenFileInEditorInTerminal(t *testing.T) {
	e := newSearchEnv(t)
	rec := e.fake("foot")
	f := e.file("notes.txt", "x")
	e.set("EDITOR=code -w", "TERMINAL=foot")
	d := e.run(map[string]any{"op": "open", "path": f})
	if d.Started != "code" {
		t.Fatalf("started = %q", d.Started)
	}
	if got := e.args(rec); !reflect.DeepEqual(got, []string{"-e", "code", "-w", f}) {
		t.Fatalf("got %v", got)
	}
}

func TestOpenLineArguments(t *testing.T) {
	cases := map[string][]string{
		"nvim":    {"+7", "FILE"},
		"vim":     {"+7", "FILE"},
		"nano":    {"+7", "FILE"},
		"emacs":   {"+7", "FILE"},
		"hx":      {"FILE:7"},
		"micro":   {"FILE:7"},
		"zeditor": {"FILE:7"},
		"code":    {"-g", "FILE:7"},
		"codium":  {"-g", "FILE:7"},
		"unknown": {"FILE"},
	}
	for editor, want := range cases {
		e := newSearchEnv(t)
		rec := e.fake("foot")
		f := e.file("a.go", "x")
		e.set("EDITOR=/usr/bin/"+editor, "TERMINAL=foot")
		e.run(map[string]any{"op": "open", "path": f, "line": 7})
		var w []string
		for _, a := range want {
			w = append(w, strings.ReplaceAll(a, "FILE", f))
		}
		got := e.args(rec)
		if !reflect.DeepEqual(got, append([]string{"-e", "/usr/bin/" + editor}, w...)) {
			t.Errorf("%s: got %v", editor, got)
		}
	}
}

func TestOpenWithoutLineGivesFileOnly(t *testing.T) {
	e := newSearchEnv(t)
	rec := e.fake("foot")
	f := e.file("a.txt", "x")
	e.set("EDITOR=nvim", "TERMINAL=foot")
	e.run(map[string]any{"op": "open", "path": f})
	if got := e.args(rec); !reflect.DeepEqual(got, []string{"-e", "nvim", f}) {
		t.Fatalf("got %v", got)
	}
}

func TestOpenErrors(t *testing.T) {
	e := newSearchEnv(t)
	f := e.file("a.txt", "x")
	code, msg := e.fail(map[string]any{"op": "open", "path": f})
	if code != "no_editor" || !contains(msg, "EDITOR is not set") || !contains(msg, ".bashrc") || !contains(msg, "environment {") {
		t.Errorf("no EDITOR: code=%q msg=%q", code, msg)
	}
	e.set("EDITOR=nvim")
	code, msg = e.fail(map[string]any{"op": "open", "path": f})
	if code != "no_terminal" || !contains(msg, "TERMINAL") || !contains(msg, "modules.prism-search.terminal") {
		t.Errorf("no terminal: code=%q msg=%q", code, msg)
	}
	e.set("TERMINAL=foot-not-installed")
	code, _ = e.fail(map[string]any{"op": "open", "path": f})
	if code != "start_failed" {
		t.Errorf("terminal missing from PATH: code=%q", code)
	}
	code, msg = e.fail(map[string]any{"op": "open", "path": e.home})
	if code != "is_directory" || msg != "Only files are opened." {
		t.Errorf("directory: code=%q msg=%q", code, msg)
	}
	code, msg = e.fail(map[string]any{"op": "open", "path": filepath.Join(e.home, "gone.txt")})
	if code != "not_found" || !contains(msg, "gone.txt") || contains(msg, e.home) {
		t.Errorf("vanished: code=%q msg=%q", code, msg)
	}
}

func TestOpenStartsEditorDetached(t *testing.T) {
	e := newSearchEnv(t)
	// The fake terminal sleeps; the helper must answer without waiting for it.
	script := "#!/bin/sh\nsleep 5\n"
	if err := os.WriteFile(filepath.Join(e.bin, "slowterm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	f := e.file("a.txt", "x")
	e.set("EDITOR=nvim", "TERMINAL=slowterm")
	r := runProgramTimed(e, map[string]any{"op": "open", "path": f})
	if r > 2000 {
		t.Fatalf("the helper waited %d ms for the terminal", r)
	}
}

// ---- web ----

const testEngine = "https://engine.test/?q=%s"

func TestWebEncodesExactly(t *testing.T) {
	e := newSearchEnv(t)
	rec := e.fake("mybrowser")
	e.set("BROWSER=mybrowser")
	query := "a b&c#d%e \"q\" é 日本 😀 + =/?"
	e.run(map[string]any{"op": "web", "query": query, "searchUrl": testEngine})
	got := e.args(rec)
	want := "https://engine.test/?q=a%20b%26c%23d%25e%20%22q%22%20%C3%A9%20%E6%97%A5%E6%9C%AC%20%F0%9F%98%80%20%2B%20%3D%2F%3F"
	if !reflect.DeepEqual(got, []string{want}) {
		t.Fatalf("got %v\nwant %s", got, want)
	}
}

func TestWebBrowserForms(t *testing.T) {
	// A browser with %s in its arguments; and colon-separated alternatives (the first that exists).
	e := newSearchEnv(t)
	rec := e.fake("mybrowser")
	e.set("BROWSER=missing-one:mybrowser --new-tab %s --private")
	e.run(map[string]any{"op": "web", "query": "x y", "searchUrl": testEngine})
	want := []string{"--new-tab", "https://engine.test/?q=x%20y", "--private"}
	if got := e.args(rec); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWebErrors(t *testing.T) {
	e := newSearchEnv(t)
	code, msg := e.fail(map[string]any{"op": "web", "query": "x", "searchUrl": testEngine})
	if code != "no_browser" || !contains(msg, "BROWSER is not set") || !contains(msg, ".bashrc") {
		t.Errorf("no browser: code=%q msg=%q", code, msg)
	}
	e.set("BROWSER=mybrowser")
	for _, bad := range []string{"", "ftp://x/?q=%s", "https://x/?q=", "https://x/?q=%s&r=%s", "javascript:%s"} {
		code, _ = e.fail(map[string]any{"op": "web", "query": "x", "searchUrl": bad})
		if code != "bad_engine" {
			t.Errorf("template %q: code=%q", bad, code)
		}
	}
	code, _ = e.fail(map[string]any{"op": "web", "query": strings.Repeat("a", 2100), "searchUrl": testEngine})
	if code != "too_long" {
		t.Errorf("long query: code=%q", code)
	}
	e2 := newSearchEnv(t)
	e2.set("BROWSER=not-installed-browser")
	code, _ = e2.fail(map[string]any{"op": "web", "query": "x", "searchUrl": testEngine})
	if code != "start_failed" {
		t.Errorf("missing browser program: code=%q", code)
	}
}

func TestWebAndOpenLogNoUserText(t *testing.T) {
	e := newSearchEnv(t)
	e.set("BROWSER=not-installed-browser")
	_, r := e.runRaw(map[string]any{"op": "web", "query": "secret-query-text", "searchUrl": testEngine})
	if strings.Contains(r.stderr, "secret-query-text") || strings.Contains(r.stdout, "secret-query-text") {
		t.Fatalf("the query leaked: stdout=%q stderr=%q", r.stdout, r.stderr)
	}
	_, r = e.runRaw(map[string]any{"op": "grep", "query": "secret-pattern /no/such/secret-file", "limit": 5})
	if strings.Contains(r.stderr, "secret-pattern") {
		t.Fatalf("the pattern leaked into diagnostics: %q", r.stderr)
	}
}

func TestUnknownOperation(t *testing.T) {
	e := newSearchEnv(t)
	if code, _ := e.fail(map[string]any{"op": "nope"}); code != "bad_request" {
		t.Fatalf("code = %q", code)
	}
	r := runProgram("search", "not json", e.env())
	if env := parseEnvelope(t, r); env.OK || env.Error.Code != "bad_request" {
		t.Fatalf("bad JSON: %s", r.stdout)
	}
}
