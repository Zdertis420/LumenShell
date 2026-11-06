package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type resolved struct {
	Values  map[string]any    `json:"values"`
	Errors  []resolvedError   `json:"errors"`
	Sources map[string]string `json:"sources"`
}

type resolvedError struct {
	Key      string `json:"key"`
	Problem  string `json:"problem"`
	Fallback any    `json:"fallback"`
}

func resolveConfig(t *testing.T, userFile, themeOverride string) resolved {
	t.Helper()
	in := map[string]any{"userFile": userFile, "dataDir": dataDir()}
	if themeOverride != "" {
		in["themeOverride"] = themeOverride
	}
	r := runProgram("config", mustJSON(in), nil, "resolve")
	e := parseEnvelope(t, r)
	if !e.OK {
		t.Fatalf("resolve failed: %+v", e.Error)
	}
	var out resolved
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func get(values map[string]any, a, b string) any {
	m, _ := values[a].(map[string]any)
	return m[b]
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "lumen", "config.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveNoFileGivesDefaultsAndCreatesNothing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lumen", "config.json")
	out := resolveConfig(t, p, "")
	if len(out.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", out.Errors)
	}
	if got := get(out.Values, "theme", "preset"); got != "catppuccin-mocha" {
		t.Fatalf("theme.preset = %v", got)
	}
	if got := get(out.Values, "log", "level"); got != "info" {
		t.Fatalf("log.level = %v", got)
	}
	if _, err := os.Stat(filepath.Dir(p)); !os.IsNotExist(err) {
		t.Fatalf("resolve must not create the config directory: %v", err)
	}
}

func TestResolveOverrideMerges(t *testing.T) {
	p := writeFile(t, `{"theme":{"preset":"tokyo-night"},"log":{"level":"debug"}}`)
	out := resolveConfig(t, p, "")
	if got := get(out.Values, "theme", "preset"); got != "tokyo-night" {
		t.Fatalf("theme.preset = %v", got)
	}
	if out.Sources["theme.preset"] != "user" || out.Sources["theme.scheme"] != "default" {
		t.Fatalf("sources = %+v", out.Sources)
	}
}

func TestResolvePresetSuppliesModeAndUserWins(t *testing.T) {
	out := resolveConfig(t, writeFile(t, `{"theme":{"preset":"catppuccin-latte"}}`), "")
	if got := get(out.Values, "theme", "mode"); got != "light" || out.Sources["theme.mode"] != "preset" {
		t.Fatalf("mode = %v source = %s", got, out.Sources["theme.mode"])
	}
	out = resolveConfig(t, writeFile(t, `{"theme":{"preset":"catppuccin-latte","mode":"dark"}}`), "")
	if got := get(out.Values, "theme", "mode"); got != "dark" || out.Sources["theme.mode"] != "user" {
		t.Fatalf("user mode must win: %v %s", got, out.Sources["theme.mode"])
	}
	out = resolveConfig(t, writeFile(t, `{"theme":{"source":"wallpaper","preset":"catppuccin-latte"}}`), "")
	if got := get(out.Values, "theme", "mode"); got != "dark" {
		t.Fatalf("preset mode must not apply to wallpaper source: %v", got)
	}
}

func TestResolveInvalidValueFallsBackAlone(t *testing.T) {
	p := writeFile(t, `{"log":{"level":"loud"},"theme":{"preset":"tokyo-night"},"helper":{"timeoutMs":5}}`)
	out := resolveConfig(t, p, "")
	if len(out.Errors) != 2 {
		t.Fatalf("errors = %+v", out.Errors)
	}
	byKey := map[string]resolvedError{}
	for _, e := range out.Errors {
		byKey[e.Key] = e
	}
	if e := byKey["log.level"]; e.Problem == "" || e.Fallback != "info" {
		t.Fatalf("log.level error = %+v", e)
	}
	if _, ok := byKey["helper.timeoutMs"]; !ok {
		t.Fatalf("missing helper.timeoutMs error: %+v", out.Errors)
	}
	if got := get(out.Values, "log", "level"); got != "info" {
		t.Fatalf("log.level = %v", got)
	}
	if got := get(out.Values, "theme", "preset"); got != "tokyo-night" {
		t.Fatalf("valid keys must be kept: %v", got)
	}
}

func TestResolveBrokenFilesGiveDefaultsAndStayUntouched(t *testing.T) {
	for name, content := range map[string]string{
		"empty":    "",
		"spaces":   "  \n",
		"invalid":  "{not json",
		"array":    "[1,2]",
		"trailing": `{"log":{}} extra`,
	} {
		t.Run(name, func(t *testing.T) {
			p := writeFile(t, content)
			out := resolveConfig(t, p, "")
			if len(out.Errors) != 1 || out.Errors[0].Key != "" || out.Errors[0].Problem == "" {
				t.Fatalf("errors = %+v", out.Errors)
			}
			if got := get(out.Values, "log", "level"); got != "info" {
				t.Fatalf("defaults must apply: %v", got)
			}
			b, _ := os.ReadFile(p)
			if string(b) != content {
				t.Fatalf("file was modified: %q", b)
			}
		})
	}
}

func TestResolveUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	p := writeFile(t, `{"log":{"level":"debug"}}`)
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	out := resolveConfig(t, p, "")
	if len(out.Errors) != 1 || out.Errors[0].Key != "" {
		t.Fatalf("errors = %+v", out.Errors)
	}
}

func TestResolveNonObjectParent(t *testing.T) {
	out := resolveConfig(t, writeFile(t, `{"theme":5}`), "")
	if len(out.Errors) != 1 || out.Errors[0].Key != "theme" {
		t.Fatalf("errors = %+v", out.Errors)
	}
}

func TestResolveTempTheme(t *testing.T) {
	p := writeFile(t, `{"theme":{"preset":"tokyo-night"}}`)
	out := resolveConfig(t, p, "catppuccin-latte")
	if got := get(out.Values, "theme", "preset"); got != "catppuccin-latte" || out.Sources["theme.preset"] != "temp" {
		t.Fatalf("preset = %v source = %s", got, out.Sources["theme.preset"])
	}
	if got := get(out.Values, "theme", "mode"); got != "light" {
		t.Fatalf("mode = %v", got)
	}
	out = resolveConfig(t, p, "wallpaper")
	if got := get(out.Values, "theme", "source"); got != "wallpaper" || out.Sources["theme.source"] != "temp" {
		t.Fatalf("source = %v", got)
	}
	b, _ := os.ReadFile(p)
	if string(b) != `{"theme":{"preset":"tokyo-night"}}` {
		t.Fatalf("temp theme must not write the file: %q", b)
	}
}

func TestResolveUnknownTempTheme(t *testing.T) {
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": filepath.Join(t.TempDir(), "c.json"), "dataDir": dataDir(), "themeOverride": "nope",
	}), nil, "resolve")
	e := parseEnvelope(t, r)
	if e.OK || e.Error == nil || e.Error.Code != "invalid_theme" {
		t.Fatalf("response = %+v", e)
	}
}

func TestResolveKeepsUnknownKeysOutOfValuesButFileIntact(t *testing.T) {
	content := `{"custom":{"x":1},"log":{"level":"warn"}}`
	p := writeFile(t, content)
	out := resolveConfig(t, p, "")
	if len(out.Errors) != 0 {
		t.Fatalf("unknown keys are not errors: %+v", out.Errors)
	}
	if b, _ := os.ReadFile(p); string(b) != content {
		t.Fatalf("file changed: %q", b)
	}
}

func TestHelperContractBadUsage(t *testing.T) {
	if r := runProgram("config", "{}", nil); r.code != 2 {
		t.Fatalf("no subcommand: exit %d", r.code)
	}
	if r := runProgram("config", "{}", nil, "frobnicate"); r.code != 2 {
		t.Fatalf("unknown subcommand: exit %d", r.code)
	}
	r := runProgram("config", "not json", nil, "resolve")
	e := parseEnvelope(t, r)
	if e.OK || e.Error.Code != "bad_input" || r.code != 0 {
		t.Fatalf("bad input: %+v exit %d", e, r.code)
	}
}
