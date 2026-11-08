package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Black-box tests of the `config` helper for the setting types added in module interface
// 1.1: `stringlist`, and the minLength/maxLength limits of `string`.
// Contract: specs/003-bar/contracts/helper-bar.md ("config helper changes").

type listError struct {
	Key      string `json:"key"`
	Problem  string `json:"problem"`
	Fallback any    `json:"fallback"`
	Dropped  bool   `json:"dropped"`
	Value    any    `json:"value"`
}

type listResolved struct {
	Values  map[string]any    `json:"values"`
	Errors  []listError       `json:"errors"`
	Sources map[string]string `json:"sources"`
}

func listModule() map[string]any {
	return map[string]any{
		"id":             "lists",
		"defaultEnabled": true,
		"settings": map[string]any{
			"names": map[string]any{"type": "stringlist", "default": []any{"a", "b"}, "maxItems": 3},
			"wide":  map[string]any{"type": "stringlist", "default": []any{}},
			"word":  map[string]any{"type": "string", "default": "ok", "minLength": 1, "maxLength": 8},
		},
	}
}

func writeUser(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func resolveLists(t *testing.T, userFile string) listResolved {
	t.Helper()
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": userFile, "dataDir": dataDir(), "modules": []any{listModule()},
	}), nil, "resolve")
	e := parseEnvelope(t, r)
	if !e.OK {
		t.Fatalf("resolve failed: %+v", e.Error)
	}
	var out listResolved
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func listOf(t *testing.T, res listResolved, key string) []any {
	t.Helper()
	got, ok := modulesOf(res.Values, "lists")[key].([]any)
	if !ok {
		t.Fatalf("modules.lists.%s is not a list: %#v", key, modulesOf(res.Values, "lists")[key])
	}
	return got
}

func errorFor(res listResolved, key string) *listError {
	for i := range res.Errors {
		if res.Errors[i].Key == key {
			return &res.Errors[i]
		}
	}
	return nil
}

func TestStringlistDefaultAndOverride(t *testing.T) {
	res := resolveLists(t, writeUser(t, `{}`))
	if got := listOf(t, res, "names"); !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Fatalf("default = %v", got)
	}
	if res.Sources["modules.lists.names"] != "default" {
		t.Fatalf("source = %q", res.Sources["modules.lists.names"])
	}

	res = resolveLists(t, writeUser(t, `{"modules":{"lists":{"names":["z","y","x"]}}}`))
	if got := listOf(t, res, "names"); !reflect.DeepEqual(got, []any{"z", "y", "x"}) {
		t.Fatalf("override must keep its order: %v", got)
	}
	if res.Sources["modules.lists.names"] != "user" {
		t.Fatalf("source = %q", res.Sources["modules.lists.names"])
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}
}

func TestStringlistEmptyListIsValid(t *testing.T) {
	res := resolveLists(t, writeUser(t, `{"modules":{"lists":{"names":[]}}}`))
	if got := listOf(t, res, "names"); len(got) != 0 {
		t.Fatalf("an empty list must stay empty, got %v", got)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}
}

func TestStringlistWholeKeyFallsBack(t *testing.T) {
	for name, file := range map[string]string{
		"not a list":     `{"modules":{"lists":{"names":"a"}}}`,
		"object":         `{"modules":{"lists":{"names":{"a":1}}}}`,
		"too many items": `{"modules":{"lists":{"names":["a","b","c","d"]}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			res := resolveLists(t, writeUser(t, file))
			if got := listOf(t, res, "names"); !reflect.DeepEqual(got, []any{"a", "b"}) {
				t.Fatalf("the key must fall back to its default, got %v", got)
			}
			e := errorFor(res, "modules.lists.names")
			if e == nil || e.Dropped {
				t.Fatalf("want a whole-key error, got %+v", res.Errors)
			}
			if !reflect.DeepEqual(e.Fallback, []any{"a", "b"}) {
				t.Fatalf("fallback = %v", e.Fallback)
			}
			// Other keys are not affected.
			if got := listOf(t, res, "wide"); len(got) != 0 {
				t.Fatalf("other keys must keep their defaults, got %v", got)
			}
		})
	}
}

func TestStringlistDropsBadEntriesOnTheirOwn(t *testing.T) {
	long := strings.Repeat("x", 81)
	ok80 := strings.Repeat("y", 80)
	file := `{"modules":{"lists":{"wide":["a", 42, "", null, {"k":1}, "` + long + `", "` + ok80 + `", "b"]}}}`
	res := resolveLists(t, writeUser(t, file))

	got := listOf(t, res, "wide")
	if !reflect.DeepEqual(got, []any{"a", ok80, "b"}) {
		t.Fatalf("good entries must stay in order, got %v", got)
	}
	wantIdx := map[string]bool{
		"modules.lists.wide[1]": false, "modules.lists.wide[2]": false, "modules.lists.wide[3]": false,
		"modules.lists.wide[4]": false, "modules.lists.wide[5]": false,
	}
	for _, e := range res.Errors {
		if _, ok := wantIdx[e.Key]; ok {
			wantIdx[e.Key] = true
			if !e.Dropped || e.Problem == "" {
				t.Errorf("%s: want dropped=true and a problem, got %+v", e.Key, e)
			}
		}
	}
	for k, seen := range wantIdx {
		if !seen {
			t.Errorf("no error for %s: %+v", k, res.Errors)
		}
	}
	if len(res.Errors) != len(wantIdx) {
		t.Errorf("want %d errors, got %+v", len(wantIdx), res.Errors)
	}
	// The value is reported as written (cut to 80 characters).
	e := errorFor(res, "modules.lists.wide[1]")
	if e == nil || e.Value == nil {
		t.Fatalf("the dropped value must be reported: %+v", e)
	}
	if s, ok := errorFor(res, "modules.lists.wide[5]").Value.(string); !ok || len([]rune(s)) > 80 {
		t.Errorf("a long value must be cut to 80 characters, got %q", s)
	}
}

func TestStringLengthLimits(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{`"a"`, true}, {`"12345678"`, true}, {`""`, false}, {`"123456789"`, false},
		{`"ééééééé"`, true},    // 7 characters, 14 bytes
		{`"éééééééé"`, true},   // 8 characters, 16 bytes: counted in characters
		{`"ééééééééé"`, false}, // 9 characters
	}
	for _, c := range cases {
		res := resolveLists(t, writeUser(t, `{"modules":{"lists":{"word":`+c.value+`}}}`))
		e := errorFor(res, "modules.lists.word")
		if c.ok && e != nil {
			t.Errorf("%s: unexpected error %+v", c.value, e)
		}
		if !c.ok {
			if e == nil || e.Fallback != "ok" {
				t.Errorf("%s: want an error with fallback \"ok\", got %+v", c.value, res.Errors)
			}
			if got := modulesOf(res.Values, "lists")["word"]; got != "ok" {
				t.Errorf("%s: value = %v, want the default", c.value, got)
			}
		}
	}
}

func TestStringlistSet(t *testing.T) {
	mods := []map[string]any{listModule()}
	file := filepath.Join(t.TempDir(), "config.json")

	e := setModuleKeys(t, file, mods, map[string]any{"key": "modules.lists.names", "value": []any{"p", "q"}})
	if !e.OK {
		t.Fatalf("set failed: %+v", e.Error)
	}
	res := resolveLists(t, file)
	if got := listOf(t, res, "names"); !reflect.DeepEqual(got, []any{"p", "q"}) {
		t.Fatalf("saved list = %v", got)
	}

	// Refused writes must leave the file untouched.
	before, _ := os.ReadFile(file)
	for name, v := range map[string]any{
		"not a list":     "a",
		"too many items": []any{"a", "b", "c", "d"},
	} {
		e := setModuleKeys(t, file, mods, map[string]any{"key": "modules.lists.names", "value": v})
		if e.OK || e.Error == nil || e.Error.Code != "invalid_value" {
			t.Errorf("%s: want invalid_value, got %+v", name, e)
		}
		after, _ := os.ReadFile(file)
		if string(after) != string(before) {
			t.Errorf("%s: the file changed after a refused write", name)
		}
	}

	// An entry that would be dropped on read does not make a write fail, so a hand edit and
	// `set` behave the same.
	e = setModuleKeys(t, file, mods, map[string]any{"key": "modules.lists.wide", "value": []any{"a", 7}})
	if !e.OK {
		t.Fatalf("set with a wrong-type entry must succeed: %+v", e.Error)
	}
	res = resolveLists(t, file)
	if got := listOf(t, res, "wide"); !reflect.DeepEqual(got, []any{"a"}) {
		t.Fatalf("the wrong-type entry must be dropped on read, got %v", got)
	}

	// Unset removes the override and the empty parent.
	e = unsetModuleKeys(t, file, mods, "modules.lists.names", "modules.lists.wide")
	if !e.OK {
		t.Fatalf("unset failed: %+v", e.Error)
	}
	b, _ := os.ReadFile(file)
	var tree map[string]any
	if err := json.Unmarshal(b, &tree); err != nil {
		t.Fatal(err)
	}
	if _, has := tree["modules"]; has {
		t.Fatalf("an empty modules object must be removed, file = %s", b)
	}
}

func TestStringlistKeepsUnknownKeys(t *testing.T) {
	mods := []map[string]any{listModule()}
	file := writeUser(t, `{"keep":{"me":1},"modules":{"gone":{"x":[1,2]}}}`)
	e := setModuleKeys(t, file, mods, map[string]any{"key": "modules.lists.names", "value": []any{"p"}})
	if !e.OK {
		t.Fatalf("set failed: %+v", e.Error)
	}
	e = unsetModuleKeys(t, file, mods, "modules.lists.names")
	if !e.OK {
		t.Fatalf("unset failed: %+v", e.Error)
	}
	b, _ := os.ReadFile(file)
	var tree map[string]any
	_ = json.Unmarshal(b, &tree)
	if _, ok := tree["keep"]; !ok {
		t.Errorf("unknown top-level key lost: %s", b)
	}
	if got := modulesOf(tree, "gone"); got == nil || got["x"] == nil {
		t.Errorf("entry of an uninstalled module lost: %s", b)
	}
}

func TestResolveWithoutModulesIsUnchangedBySetting(t *testing.T) {
	// Input without `modules` still behaves like Foundation: module keys are not validated.
	res := resolveConfig(t, writeUser(t, `{"modules":{"lists":{"names":"nonsense"}}}`), "")
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors without module definitions: %+v", res.Errors)
	}
}
