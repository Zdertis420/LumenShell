package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Fixtures for the tests of the search helper (specs/006-prism-modes): a temporary home with
// its own XDG folders and PATH, fake programs that record their arguments, and a runner that
// returns the "data" of an answer.

type searchEnv struct {
	t     *testing.T
	home  string
	data  string // XDG_DATA_HOME
	sys   string // a system data dir, listed in XDG_DATA_DIRS
	cache string
	bin   string // a folder of fake programs, first in PATH
	extra []string
}

func newSearchEnv(t *testing.T) *searchEnv {
	t.Helper()
	root := t.TempDir()
	e := &searchEnv{
		t:     t,
		home:  filepath.Join(root, "home"),
		data:  filepath.Join(root, "home", ".local", "share"),
		sys:   filepath.Join(root, "sysdata"),
		cache: filepath.Join(root, "cache"),
		bin:   filepath.Join(root, "fakebin"),
	}
	for _, d := range []string{e.home, e.data, e.sys, e.cache, e.bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return e
}

// env is the environment of a helper run. Unrelated variables are cleared so the tests do
// not depend on the developer's own EDITOR, BROWSER or TERMINAL.
func (e *searchEnv) env() []string {
	env := []string{
		"HOME=" + e.home,
		"XDG_DATA_HOME=" + e.data,
		"XDG_DATA_DIRS=" + e.sys,
		"XDG_CACHE_HOME=" + e.cache,
		"XDG_CURRENT_DESKTOP=niri",
		"LANG=C",
		"LC_MESSAGES=C",
		"PATH=" + e.bin + ":" + os.Getenv("PATH"),
		"EDITOR=", "BROWSER=", "TERMINAL=",
	}
	return append(env, e.extra...)
}

func (e *searchEnv) set(kv ...string) { e.extra = append(e.extra, kv...) }

// file writes a file under the home (or an absolute path) and creates its folders.
func (e *searchEnv) file(rel, body string) string {
	e.t.Helper()
	p := rel
	if !filepath.IsAbs(rel) {
		p = filepath.Join(e.home, rel)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		e.t.Fatal(err)
	}
	return p
}

// desktop writes a desktop entry into the user's (dir "user") or the system folder.
func (e *searchEnv) desktop(where, id string, lines ...string) {
	e.t.Helper()
	dir := filepath.Join(e.sys, "applications")
	if where == "user" {
		dir = filepath.Join(e.data, "applications")
	}
	body := "[Desktop Entry]\n" + strings.Join(lines, "\n") + "\n"
	e.file(filepath.Join(dir, id), body)
}

// fake installs a program that records its arguments, one per line, in <bin>/<name>.args.
func (e *searchEnv) fake(name string) string {
	e.t.Helper()
	rec := filepath.Join(e.bin, name+".args")
	script := "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > '" + rec + "'\n"
	p := filepath.Join(e.bin, name)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		e.t.Fatal(err)
	}
	return rec
}

// args waits briefly for a fake program to have run and returns the arguments it recorded.
func (e *searchEnv) args(rec string) []string {
	e.t.Helper()
	for i := 0; i < 100; i++ {
		if b, err := os.ReadFile(rec); err == nil && len(b) > 0 {
			return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
		}
		sleepMs(20)
	}
	e.t.Fatalf("the fake program %s never ran", filepath.Base(rec))
	return nil
}

func (e *searchEnv) neverRan(rec string) {
	e.t.Helper()
	sleepMs(200)
	if _, err := os.Stat(rec); err == nil {
		e.t.Fatalf("%s ran but must not have", filepath.Base(rec))
	}
}

type searchRow struct {
	Key      string   `json:"key"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Icon     string   `json:"icon"`
	Marks    [][2]int `json:"marks"`
}

type searchData struct {
	Rows     []searchRow `json:"rows"`
	Total    int         `json:"total"`
	Cut      bool        `json:"cut"`
	Hint     string      `json:"hint"`
	Dir      string      `json:"dir"`
	Name     string      `json:"name"`
	Started  string      `json:"started"`
	Messages []struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	} `json:"messages"`
}

// run sends one request and returns the data; the test fails if the answer is not ok.
func (e *searchEnv) run(req map[string]any) searchData {
	e.t.Helper()
	env, r := e.runRaw(req)
	if !env.OK {
		e.t.Fatalf("op %v failed: %s", req["op"], r.stdout)
	}
	var d searchData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		e.t.Fatalf("bad data: %v\n%s", err, r.stdout)
	}
	return d
}

// runRaw returns the envelope, for tests of error answers.
func (e *searchEnv) runRaw(req map[string]any) (envelope, result) {
	e.t.Helper()
	r := runProgram("search", mustJSON(req), e.env())
	return parseEnvelope(e.t, r), r
}

func (e *searchEnv) fail(req map[string]any) (code, message string) {
	e.t.Helper()
	env, r := e.runRaw(req)
	if env.OK || env.Error == nil {
		e.t.Fatalf("op %v should have failed: %s", req["op"], r.stdout)
	}
	return env.Error.Code, env.Error.Message
}

func sleepMs(n int) { time.Sleep(time.Duration(n) * time.Millisecond) }

// runProgramTimed returns how long, in milliseconds, one helper run took.
func runProgramTimed(e *searchEnv, req map[string]any) int {
	start := time.Now()
	e.run(req)
	return int(time.Since(start) / time.Millisecond)
}

func titles(rows []searchRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Title)
	}
	return out
}

func keys(rows []searchRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Key)
	}
	return out
}
