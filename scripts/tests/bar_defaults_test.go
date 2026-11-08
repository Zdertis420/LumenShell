package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The bundled bar and its built-in items, scanned as shipped, give the default bar without
// any warning. Contract: specs/003-bar/contracts/bar-config.md and data-model.md.

func scanBundled(t *testing.T) map[string]contribModule {
	t.Helper()
	e, r := runModules(t, "scan", map[string]any{
		"bundledDir": filepath.Join(repoRoot, "modules"), "userDir": "", "apiVersion": "1.1",
	})
	if !e.OK {
		t.Fatalf("scan failed: %+v\nstderr=%s", e.Error, r.stderr)
	}
	var out struct {
		Modules []contribModule `json:"modules"`
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	m := map[string]contribModule{}
	for _, x := range out.Modules {
		m[x.ID] = x
	}
	return m
}

func manifestSettings(t *testing.T, m contribModule) map[string]map[string]any {
	t.Helper()
	raw, _ := m.Manifest["settings"].(map[string]any)
	out := map[string]map[string]any{}
	for k, v := range raw {
		out[k], _ = v.(map[string]any)
	}
	return out
}

func TestBundledBarModulesAreValid(t *testing.T) {
	mods := scanBundled(t)
	for _, id := range []string{"bar", "bar-clock", "bar-sound", "bar-battery"} {
		m, ok := mods[id]
		if !ok {
			t.Fatalf("%s is not bundled", id)
		}
		if !m.Valid || len(m.Errors) != 0 || len(m.Warnings) != 0 || m.Origin != "bundled" || !m.DefaultEnabled {
			t.Errorf("%s: valid=%v errors=%+v warnings=%+v origin=%s default=%v", id, m.Valid, m.Errors, m.Warnings, m.Origin, m.DefaultEnabled)
		}
	}
}

func TestBarDeclaresItsPointCommandAndDefaults(t *testing.T) {
	bar := scanBundled(t)["bar"]
	if len(bar.ExtensionPoints) != 1 || bar.ExtensionPoints[0].ID != "items" {
		t.Errorf("extensionPoints = %+v", bar.ExtensionPoints)
	}
	if !reflect.DeepEqual(bar.Commands, []string{"focus"}) {
		t.Errorf("commands = %v", bar.Commands)
	}
	s := manifestSettings(t, bar)
	want := map[string]any{
		"edge": "top", "screens": []any{"all"}, "left": []any{}, "center": []any{"bar-clock"},
		"right": []any{"bar-sound", "bar-battery"}, "focusScreen": "",
	}
	for k, def := range want {
		if !reflect.DeepEqual(s[k]["default"], def) {
			t.Errorf("default of %s = %#v, want %#v", k, s[k]["default"], def)
		}
	}
	for _, k := range []string{"screens", "left", "center", "right"} {
		if s[k]["type"] != "stringlist" {
			t.Errorf("%s must be a stringlist, got %v", k, s[k]["type"])
		}
	}
	if !reflect.DeepEqual(s["edge"]["enum"], []any{"top", "bottom"}) {
		t.Errorf("edge enum = %v", s["edge"]["enum"])
	}
	// The bar's notification helper is declared and shipped.
	perms, _ := bar.Manifest["permissions"].([]any)
	if !reflect.DeepEqual(perms, []any{"helpers"}) {
		t.Errorf("bar permissions = %v", perms)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "modules", "bar", "bin", "notify")); err != nil {
		t.Errorf("the notify helper is missing: %v", err)
	}
}

func TestBuiltInItemsContributeToTheBar(t *testing.T) {
	mods := scanBundled(t)
	for id, item := range map[string]string{"bar-clock": "clock", "bar-sound": "sound", "bar-battery": "battery"} {
		m := mods[id]
		if len(m.Contributions) != 1 {
			t.Fatalf("%s: contributions = %+v", id, m.Contributions)
		}
		c := m.Contributions[0]
		if c.ID != item || c.Target.Module != "bar" || c.Target.Point != "items" || c.Short != id || c.Key != id+"/"+item {
			t.Errorf("%s: contribution = %+v", id, c)
		}
		if m.Manifest["api"] != "1.1" {
			t.Errorf("%s: api = %v", id, m.Manifest["api"])
		}
		perms, _ := m.Manifest["permissions"].([]any)
		found := false
		for _, p := range perms {
			if p == "contribute:bar" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: permission contribute:bar missing: %v", id, perms)
		}
	}
	// bar-sound runs the mixer helper.
	sound := mods["bar-sound"]
	perms, _ := sound.Manifest["permissions"].([]any)
	hasHelpers := false
	for _, p := range perms {
		if p == "helpers" {
			hasHelpers = true
		}
	}
	helpers, _ := sound.Manifest["helpers"].([]any)
	if !hasHelpers || len(helpers) != 1 {
		t.Errorf("bar-sound must declare the helpers permission and open-mixer: %v %v", perms, helpers)
	}
}

func TestItemSettingsMatchTheDataModel(t *testing.T) {
	mods := scanBundled(t)
	type rule struct {
		typ      string
		def      any
		min, max any
	}
	want := map[string]map[string]rule{
		"bar-clock":   {"format": {"string", "ddd d MMM  HH:mm", nil, nil}, "seconds": {"boolean", false, nil, nil}},
		"bar-sound":   {"step": {"integer", float64(5), float64(1), float64(25)}, "mixer": {"string", "", nil, nil}},
		"bar-battery": {"lowLevel": {"integer", float64(20), float64(1), float64(99)}, "criticalLevel": {"integer", float64(10), float64(1), float64(98)}},
	}
	for id, rules := range want {
		s := manifestSettings(t, mods[id])
		for key, r := range rules {
			got := s[key]
			if got == nil {
				t.Errorf("%s.%s is missing", id, key)
				continue
			}
			if got["type"] != r.typ || !reflect.DeepEqual(got["default"], r.def) {
				t.Errorf("%s.%s = %v %#v, want %s %#v", id, key, got["type"], got["default"], r.typ, r.def)
			}
			if r.min != nil && !reflect.DeepEqual(got["min"], r.min) || r.max != nil && !reflect.DeepEqual(got["max"], r.max) {
				t.Errorf("%s.%s range = %v..%v", id, key, got["min"], got["max"])
			}
		}
	}
	// The clock format has a length limit (1 to 64) and the mixer command at most 128.
	clock := manifestSettings(t, mods["bar-clock"])["format"]
	if clock["minLength"] != float64(1) || clock["maxLength"] != float64(64) {
		t.Errorf("format limits = %v %v", clock["minLength"], clock["maxLength"])
	}
	if mixer := manifestSettings(t, mods["bar-sound"])["mixer"]; mixer["maxLength"] != float64(128) {
		t.Errorf("mixer maxLength = %v", mixer["maxLength"])
	}
}

// The default lists resolve to the default bar: the clock in the center, sound and battery on
// the right, nothing on the left, and no warning.
func TestDefaultListsGiveTheDefaultBar(t *testing.T) {
	mods := scanBundled(t)
	var contribs []any
	for _, id := range []string{"bar-clock", "bar-sound", "bar-battery"} {
		for _, c := range mods[id].Contributions {
			contribs = append(contribs, map[string]any{
				"key": c.Key, "short": c.Short, "module": id, "item": c.ID, "name": c.Name,
				"moduleState": "loaded", "enabled": true, "reason": "",
				"target": map[string]any{"module": c.Target.Module, "point": c.Target.Point},
			})
		}
	}
	s := manifestSettings(t, mods["bar"])
	res := runItems(t, s["left"]["default"].([]any), s["center"]["default"].([]any), s["right"]["default"].([]any), contribs)
	if len(res.Left) != 0 || len(res.Warnings) != 0 {
		t.Fatalf("left=%+v warnings=%+v", res.Left, res.Warnings)
	}
	if !reflect.DeepEqual(keysOf(res.Center), []string{"bar-clock/clock"}) {
		t.Errorf("center = %v", keysOf(res.Center))
	}
	if !reflect.DeepEqual(keysOf(res.Right), []string{"bar-sound/sound", "bar-battery/battery"}) {
		t.Errorf("right = %v", keysOf(res.Right))
	}
}
