package tests

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func change(key string, value any) map[string]any {
	return map[string]any{"key": key, "value": value}
}

func runSet(t *testing.T, userFile string, changes ...map[string]any) envelope {
	t.Helper()
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": userFile, "dataDir": dataDir(), "changes": changes,
	}), nil, "set")
	return parseEnvelope(t, r)
}

func runUnsetKeys(t *testing.T, userFile string, keys ...string) envelope {
	t.Helper()
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": userFile, "dataDir": dataDir(), "keys": keys,
	}), nil, "unset")
	return parseEnvelope(t, r)
}

func readTree(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("file is not valid JSON: %v\n%s", err, b)
	}
	return m
}

func TestSetWritesOnlyOverridesAndPreservesUnknownKeys(t *testing.T) {
	p := writeFile(t, `{"custom":{"keep":true},"theme":{"mode":"light","extra":"x"}}`)
	e := runSet(t, p, change("theme.source", "preset"), change("theme.preset", "tokyo-night"))
	if !e.OK {
		t.Fatalf("set failed: %+v", e.Error)
	}
	want := map[string]any{
		"custom": map[string]any{"keep": true},
		"theme":  map[string]any{"mode": "light", "extra": "x", "source": "preset", "preset": "tokyo-night"},
	}
	if got := readTree(t, p); !reflect.DeepEqual(got, want) {
		t.Fatalf("file = %v\nwant %v", got, want)
	}
}

func TestSetCreatesFileAndDirectoryWithOnlyTheOverride(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lumen", "config.json")
	if e := runSet(t, p, change("log.level", "debug")); !e.OK {
		t.Fatalf("set failed: %+v", e.Error)
	}
	want := map[string]any{"log": map[string]any{"level": "debug"}}
	if got := readTree(t, p); !reflect.DeepEqual(got, want) {
		t.Fatalf("file = %v", got)
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestSetBatchWithOneInvalidValueChangesNothing(t *testing.T) {
	content := `{"log":{"level":"warn"}}`
	for name, ch := range map[string][]map[string]any{
		"bad enum":       {change("theme.source", "preset"), change("log.level", "loud")},
		"unknown preset": {change("theme.preset", "nope")},
		"bad int":        {change("helper.timeoutMs", 5)},
		"float":          {change("helper.timeoutMs", 1000.5)},
		"wrong type":     {change("theme.wallpaper", 12)},
		"unknown key":    {change("nope.key", "x")},
	} {
		t.Run(name, func(t *testing.T) {
			p := writeFile(t, content)
			e := runSet(t, p, ch...)
			if e.OK || e.Error == nil || (e.Error.Code != "invalid_value" && e.Error.Code != "invalid_key") {
				t.Fatalf("response = %+v", e)
			}
			if b, _ := os.ReadFile(p); string(b) != content {
				t.Fatalf("file changed: %q", b)
			}
		})
	}
}

func TestSetRefusesInvalidFile(t *testing.T) {
	for _, content := range []string{"{broken", "", "[]"} {
		p := writeFile(t, content)
		e := runSet(t, p, change("log.level", "debug"))
		if e.OK || e.Error.Code != "file_invalid" {
			t.Fatalf("content %q: response = %+v", content, e)
		}
		if b, _ := os.ReadFile(p); string(b) != content {
			t.Fatalf("file changed: %q", b)
		}
	}
}

func TestUnsetRemovesKeysAndEmptyParents(t *testing.T) {
	p := writeFile(t, `{"custom":1,"theme":{"source":"preset","preset":"tokyo-night"},"log":{"level":"debug"}}`)
	e := runUnsetKeys(t, p, "theme.source", "theme.preset")
	if !e.OK || string(e.Data) != `{"written":true}` {
		t.Fatalf("response = %+v %s", e, e.Data)
	}
	want := map[string]any{"custom": json.Number("1"), "log": map[string]any{"level": "debug"}}
	got := readTree(t, p)
	if _, hasTheme := got["theme"]; hasTheme {
		t.Fatalf("empty parent not removed: %v", got)
	}
	if got["log"].(map[string]any)["level"] != "debug" || got["custom"].(float64) != 1 {
		t.Fatalf("file = %v (want %v)", got, want)
	}
}

func TestUnsetAbsentKeyDoesNotWrite(t *testing.T) {
	content := `{"log": {"level": "debug"}}`
	p := writeFile(t, content)
	e := runUnsetKeys(t, p, "theme.preset")
	if !e.OK || string(e.Data) != `{"written":false}` {
		t.Fatalf("response = %+v %s", e, e.Data)
	}
	if b, _ := os.ReadFile(p); string(b) != content {
		t.Fatalf("file rewritten: %q", b)
	}
}

func TestUnsetNeverCreatesTheFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lumen", "config.json")
	e := runUnsetKeys(t, p, "theme.preset")
	if !e.OK || string(e.Data) != `{"written":false}` {
		t.Fatalf("response = %+v", e)
	}
	if _, err := os.Stat(filepath.Dir(p)); !os.IsNotExist(err) {
		t.Fatalf("directory created: %v", err)
	}
}

func TestUnsetRefusesInvalidFile(t *testing.T) {
	p := writeFile(t, "{broken")
	e := runUnsetKeys(t, p, "log.level")
	if e.OK || e.Error.Code != "file_invalid" {
		t.Fatalf("response = %+v", e)
	}
	if b, _ := os.ReadFile(p); string(b) != "{broken" {
		t.Fatalf("file changed: %q", b)
	}
}

// SC-004: after any sequence of sets and unsets, the file holds exactly the values
// the test set explicitly (plus the unknown key it started with).
func TestRandomSequencesLeaveOnlyExplicitValues(t *testing.T) {
	type op struct {
		key   string
		value any
	}
	candidates := []op{
		{"theme.source", "wallpaper"}, {"theme.source", "preset"},
		{"theme.preset", "tokyo-night"}, {"theme.preset", "catppuccin-mocha"},
		{"theme.mode", "light"}, {"log.level", "debug"}, {"log.level", "error"},
		{"helper.timeoutMs", float64(1000)}, {"theme.wallpaper", "/tmp/w.png"},
	}
	rng := rand.New(rand.NewSource(7))
	p := writeFile(t, `{"custom":{"x":"y"}}`)
	expected := map[string]any{}
	for i := 0; i < 40; i++ {
		c := candidates[rng.Intn(len(candidates))]
		if rng.Intn(3) == 0 {
			runUnsetKeys(t, p, c.key)
			delete(expected, c.key)
		} else {
			if e := runSet(t, p, change(c.key, c.value)); !e.OK {
				t.Fatalf("set %s failed: %+v", c.key, e.Error)
			}
			expected[c.key] = c.value
		}
	}
	got := readTree(t, p)
	if !reflect.DeepEqual(got["custom"], map[string]any{"x": "y"}) {
		t.Fatalf("unknown key lost: %v", got)
	}
	delete(got, "custom")
	flat := map[string]any{}
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			if sub, ok := v.(map[string]any); ok {
				if len(sub) == 0 {
					t.Fatalf("empty object left behind at %s", key)
				}
				walk(key, sub)
			} else {
				flat[key] = v
			}
		}
	}
	walk("", got)
	if !reflect.DeepEqual(flat, expected) {
		t.Fatalf("file holds %v, test set %v", flat, expected)
	}
}
