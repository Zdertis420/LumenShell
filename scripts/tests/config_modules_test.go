package tests

import (
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"testing"
)

// helloModule is a module definition as the Modules singleton passes it to the helper.
func helloModule() map[string]any {
	return map[string]any{
		"id":             "hello",
		"defaultEnabled": false,
		"settings": map[string]any{
			"greeting": map[string]any{"type": "string", "default": "Hello"},
			"mode":     map[string]any{"type": "string", "default": "one", "enum": []any{"one", "two"}},
			"size":     map[string]any{"type": "integer", "default": 3, "min": 1, "max": 5},
			"ratio":    map[string]any{"type": "number", "default": 0.5, "min": 0, "max": 1},
			"flag":     map[string]any{"type": "boolean", "default": false},
		},
	}
}

func bundledModule() map[string]any {
	return map[string]any{"id": "panel", "defaultEnabled": true, "settings": map[string]any{}}
}

func resolveWithModules(t *testing.T, userFile string, mods ...map[string]any) resolved {
	t.Helper()
	list := make([]any, len(mods))
	for i, m := range mods {
		list[i] = m
	}
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": userFile, "dataDir": dataDir(), "modules": list,
	}), nil, "resolve")
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

func modulesOf(values map[string]any, id string) map[string]any {
	mods, _ := values["modules"].(map[string]any)
	m, _ := mods[id].(map[string]any)
	return m
}

func setModuleKeys(t *testing.T, userFile string, mods []map[string]any, changes ...map[string]any) envelope {
	t.Helper()
	list := make([]any, len(mods))
	for i, m := range mods {
		list[i] = m
	}
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": userFile, "dataDir": dataDir(), "modules": list, "changes": changes,
	}), nil, "set")
	return parseEnvelope(t, r)
}

func unsetModuleKeys(t *testing.T, userFile string, mods []map[string]any, keys ...string) envelope {
	t.Helper()
	list := make([]any, len(mods))
	for i, m := range mods {
		list[i] = m
	}
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": userFile, "dataDir": dataDir(), "modules": list, "keys": keys,
	}), nil, "unset")
	return parseEnvelope(t, r)
}

func TestModuleDefaultsAndEnabled(t *testing.T) {
	p := writeFile(t, `{}`)
	out := resolveWithModules(t, p, helloModule(), bundledModule())
	if len(out.Errors) != 0 {
		t.Fatalf("errors: %+v", out.Errors)
	}
	hello := modulesOf(out.Values, "hello")
	if hello["enabled"] != false || hello["greeting"] != "Hello" || hello["mode"] != "one" || hello["flag"] != false {
		t.Fatalf("hello = %v", hello)
	}
	if hello["size"].(float64) != 3 || hello["ratio"].(float64) != 0.5 {
		t.Fatalf("hello numbers = %v", hello)
	}
	if panel := modulesOf(out.Values, "panel"); panel["enabled"] != true {
		t.Fatalf("bundled module must default to enabled: %v", panel)
	}
	for _, key := range []string{"modules.hello.enabled", "modules.hello.greeting", "modules.panel.enabled"} {
		if out.Sources[key] != "default" {
			t.Errorf("source of %s = %q", key, out.Sources[key])
		}
	}
}

func TestModuleUserOverridesWin(t *testing.T) {
	p := writeFile(t, `{"modules":{"hello":{"enabled":true,"greeting":"Hi","size":5,"flag":true,"ratio":1,"mode":"two"}}}`)
	out := resolveWithModules(t, p, helloModule())
	if len(out.Errors) != 0 {
		t.Fatalf("errors: %+v", out.Errors)
	}
	hello := modulesOf(out.Values, "hello")
	if hello["enabled"] != true || hello["greeting"] != "Hi" || hello["flag"] != true || hello["mode"] != "two" {
		t.Fatalf("hello = %v", hello)
	}
	if hello["size"].(float64) != 5 {
		t.Fatalf("size = %v", hello["size"])
	}
	if out.Sources["modules.hello.greeting"] != "user" || out.Sources["modules.hello.enabled"] != "user" {
		t.Fatalf("sources = %v", out.Sources)
	}
}

func TestModuleInvalidValuesFallBackAlone(t *testing.T) {
	cases := map[string]string{
		"wrong type":   `{"modules":{"hello":{"greeting":5}}}`,
		"outside enum": `{"modules":{"hello":{"mode":"three"}}}`,
		"below min":    `{"modules":{"hello":{"size":0}}}`,
		"above max":    `{"modules":{"hello":{"size":99}}}`,
		"not integer":  `{"modules":{"hello":{"size":2.5}}}`,
		"number range": `{"modules":{"hello":{"ratio":2}}}`,
		"bad boolean":  `{"modules":{"hello":{"flag":"yes"}}}`,
		"bad enabled":  `{"modules":{"hello":{"enabled":"yes"}}}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			p := writeFile(t, content)
			out := resolveWithModules(t, p, helloModule())
			if len(out.Errors) != 1 {
				t.Fatalf("errors = %+v", out.Errors)
			}
			e := out.Errors[0]
			if len(e.Key) < len("modules.hello.") || e.Key[:len("modules.hello.")] != "modules.hello." || e.Problem == "" {
				t.Fatalf("error = %+v", e)
			}
			hello := modulesOf(out.Values, "hello")
			key := e.Key[len("modules.hello."):]
			if !reflect.DeepEqual(hello[key], e.Fallback) {
				t.Fatalf("value %v, fallback reported %v", hello[key], e.Fallback)
			}
		})
	}
	// Only the bad key falls back; the good one next to it is kept.
	p := writeFile(t, `{"modules":{"hello":{"greeting":"Hi","size":99}}}`)
	out := resolveWithModules(t, p, helloModule())
	hello := modulesOf(out.Values, "hello")
	if hello["greeting"] != "Hi" || hello["size"].(float64) != 3 {
		t.Fatalf("hello = %v", hello)
	}
	// Other modules and core keys are unaffected.
	if got := get(out.Values, "log", "level"); got != "info" {
		t.Fatalf("log.level = %v", got)
	}
}

func TestModuleUnknownIdsAndKeysAreNotValidated(t *testing.T) {
	content := `{"modules":{"gone":{"x":1,"enabled":"maybe"},"hello":{"extra":true}},"log":{"level":"warn"}}`
	p := writeFile(t, content)
	out := resolveWithModules(t, p, helloModule())
	if len(out.Errors) != 0 {
		t.Fatalf("unknown ids and keys are not errors: %+v", out.Errors)
	}
	if modulesOf(out.Values, "gone") != nil {
		t.Fatalf("uninstalled module must not appear in values: %v", out.Values["modules"])
	}
	if b, _ := os.ReadFile(p); string(b) != content {
		t.Fatalf("resolve must not touch the file: %q", b)
	}
}

func TestModuleSetAcceptsDeclaredKeysAndEnabled(t *testing.T) {
	p := writeFile(t, `{"custom":1}`)
	mods := []map[string]any{helloModule(), bundledModule()}
	e := setModuleKeys(t, p, mods,
		change("modules.hello.enabled", true), change("modules.hello.greeting", "Hi"), change("modules.hello.size", 4),
		change("modules.hello.ratio", 0.25), change("modules.hello.flag", true), change("modules.panel.enabled", false))
	if !e.OK {
		t.Fatalf("set failed: %+v", e.Error)
	}
	got := readTree(t, p)
	hello := got["modules"].(map[string]any)["hello"].(map[string]any)
	if hello["enabled"] != true || hello["greeting"] != "Hi" || hello["flag"] != true || hello["size"].(float64) != 4 {
		t.Fatalf("file = %v", got)
	}
	if got["modules"].(map[string]any)["panel"].(map[string]any)["enabled"] != false || got["custom"].(float64) != 1 {
		t.Fatalf("file = %v", got)
	}
}

func TestModuleSetRefusesBadBatchesAndChangesNothing(t *testing.T) {
	content := `{"log":{"level":"warn"}}`
	mods := []map[string]any{helloModule()}
	for name, ch := range map[string][]map[string]any{
		"unknown module": {change("modules.nope.enabled", true)},
		"undeclared key": {change("modules.hello.colour", "red")},
		"bad value":      {change("modules.hello.greeting", "ok"), change("modules.hello.size", 99)},
		"bad enabled":    {change("modules.hello.enabled", "yes")},
		"bad enum":       {change("modules.hello.mode", "three")},
	} {
		t.Run(name, func(t *testing.T) {
			p := writeFile(t, content)
			e := setModuleKeys(t, p, mods, ch...)
			if e.OK || e.Error == nil || (e.Error.Code != "invalid_value" && e.Error.Code != "invalid_key") {
				t.Fatalf("response = %+v", e)
			}
			if b, _ := os.ReadFile(p); string(b) != content {
				t.Fatalf("file changed: %q", b)
			}
		})
	}
	// Without a module list the module keys are not known.
	p := writeFile(t, content)
	if e := runSet(t, p, change("modules.hello.enabled", true)); e.OK {
		t.Fatalf("module keys must be refused without module definitions")
	}
}

func TestModuleUnsetRemovesKeysAndEmptyParents(t *testing.T) {
	p := writeFile(t, `{"modules":{"hello":{"enabled":true,"greeting":"Hi"},"gone":{"x":1}},"log":{"level":"debug"}}`)
	mods := []map[string]any{helloModule()}
	e := unsetModuleKeys(t, p, mods, "modules.hello.enabled", "modules.hello.greeting")
	if !e.OK || string(e.Data) != `{"written":true}` {
		t.Fatalf("response = %+v %s", e, e.Data)
	}
	got := readTree(t, p)
	m := got["modules"].(map[string]any)
	if _, ok := m["hello"]; ok {
		t.Fatalf("empty module object must be removed: %v", got)
	}
	if m["gone"].(map[string]any)["x"].(float64) != 1 || got["log"].(map[string]any)["level"] != "debug" {
		t.Fatalf("other entries must survive: %v", got)
	}
	// Removing the last module entry removes the empty modules object too.
	p = writeFile(t, `{"modules":{"hello":{"enabled":true}}}`)
	if e := unsetModuleKeys(t, p, mods, "modules.hello.enabled"); !e.OK {
		t.Fatalf("response = %+v", e)
	}
	if got := readTree(t, p); len(got) != 0 {
		t.Fatalf("file = %v", got)
	}
	// An absent key writes nothing.
	p = writeFile(t, `{"log": {"level": "debug"}}`)
	e = unsetModuleKeys(t, p, mods, "modules.hello.enabled")
	if !e.OK || string(e.Data) != `{"written":false}` {
		t.Fatalf("response = %+v %s", e, e.Data)
	}
}

func TestModuleInputWithoutModulesBehavesLikeFoundation(t *testing.T) {
	p := writeFile(t, `{"theme":{"preset":"tokyo-night"},"modules":{"hello":{"enabled":true}}}`)
	out := resolveConfig(t, p, "")
	if len(out.Errors) != 0 {
		t.Fatalf("errors: %+v", out.Errors)
	}
	if got := get(out.Values, "theme", "preset"); got != "tokyo-night" {
		t.Fatalf("theme.preset = %v", got)
	}
	if _, ok := out.Values["modules"]; ok {
		t.Fatalf("no modules key expected without definitions: %v", out.Values["modules"])
	}
	if e := runSet(t, writeFile(t, `{}`), change("log.level", "debug")); !e.OK {
		t.Fatalf("core set must still work: %+v", e.Error)
	}
}

// SC-006: after any sequence of module sets and unsets, the file holds exactly the
// values the test set, no empty objects, and an uninstalled module's entry is intact.
func TestModuleRandomSequencesLeaveOnlyExplicitValues(t *testing.T) {
	type op struct {
		key   string
		value any
	}
	candidates := []op{
		{"modules.hello.enabled", true}, {"modules.hello.enabled", false},
		{"modules.hello.greeting", "Hi"}, {"modules.hello.greeting", "Yo"},
		{"modules.hello.size", float64(4)}, {"modules.hello.mode", "two"},
		{"modules.hello.flag", true}, {"modules.hello.ratio", 0.75},
		{"modules.panel.enabled", false},
	}
	mods := []map[string]any{helloModule(), bundledModule()}
	rng := rand.New(rand.NewSource(11))
	p := writeFile(t, `{"modules":{"gone":{"x":"y"}}}`)
	expected := map[string]any{}
	for i := 0; i < 40; i++ {
		c := candidates[rng.Intn(len(candidates))]
		if rng.Intn(3) == 0 {
			unsetModuleKeys(t, p, mods, c.key)
			delete(expected, c.key)
		} else {
			if e := setModuleKeys(t, p, mods, change(c.key, c.value)); !e.OK {
				t.Fatalf("set %s failed: %+v", c.key, e.Error)
			}
			expected[c.key] = c.value
		}
	}
	got := readTree(t, p)
	mod := got["modules"].(map[string]any)
	if !reflect.DeepEqual(mod["gone"], map[string]any{"x": "y"}) {
		t.Fatalf("uninstalled module entry lost: %v", got)
	}
	delete(mod, "gone")
	flat := map[string]any{}
	for id, entry := range mod {
		obj, ok := entry.(map[string]any)
		if !ok || len(obj) == 0 {
			t.Fatalf("empty or invalid module object left behind for %s: %v", id, entry)
		}
		for k, v := range obj {
			flat["modules."+id+"."+k] = v
		}
	}
	if !reflect.DeepEqual(flat, expected) {
		t.Fatalf("file holds %v, test set %v", flat, expected)
	}
}
