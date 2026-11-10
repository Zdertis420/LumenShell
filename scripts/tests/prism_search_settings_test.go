package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// The settings of the prism-search module (specs/006-prism-modes/contracts/search-settings.md):
// 13 keys with their types, defaults and ranges, the manifest's other promises, and the
// per-key fallback of a bad value.

const searchSettingCount = 13

func searchSettings() map[string]settingRule {
	return map[string]settingRule{
		"rowsPerGroup":     {Type: "integer", Default: float64(20), Min: f(1), Max: f(200)},
		"maxFiles":         {Type: "integer", Default: float64(200000), Min: f(1000), Max: f(5000000)},
		"searchTimeMs":     {Type: "integer", Default: float64(1500), Min: f(100), Max: f(30000)},
		"hiddenFiles":      {Type: "boolean", Default: false},
		"hiddenHomeDirs":   {Type: "boolean", Default: false},
		"systemGroup":      {Type: "boolean", Default: true},
		"systemRoots":      {Type: "stringlist", Default: []any{"/etc", "/usr", "/opt", "/var", "/srv"}},
		"commandDirs":      {Type: "stringlist", Default: []any{}},
		"grepMaxMatches":   {Type: "integer", Default: float64(500), Min: f(10), Max: f(10000)},
		"grepCase":         {Type: "string", Default: "smart", Enum: []string{"smart", "ignore", "match"}},
		"searchUrl":        {Type: "string", Default: "https://duckduckgo.com/?q=%s"},
		"terminal":         {Type: "string", Default: ""},
		"terminalExecFlag": {Type: "string", Default: nil},
	}
}

func TestSearchManifestMatchesTheContract(t *testing.T) {
	m := readModuleJSON(t, "modules/prism-search")
	if m.API != "1.3" {
		t.Errorf("api = %q, want 1.3", m.API)
	}
	want := searchSettings()
	if len(want) != searchSettingCount || len(m.Settings) != searchSettingCount {
		t.Fatalf("table has %d keys, manifest %d, want %d", len(want), len(m.Settings), searchSettingCount)
	}
	for key, w := range want {
		got, ok := m.Settings[key]
		if !ok {
			t.Errorf("%s: missing from the manifest", key)
			continue
		}
		if got.Type != w.Type {
			t.Errorf("%s: type %q, want %q", key, got.Type, w.Type)
		}
		if !reflect.DeepEqual(got.Default, w.Default) {
			t.Errorf("%s: default %#v, want %#v", key, got.Default, w.Default)
		}
		if !reflect.DeepEqual(got.Enum, w.Enum) {
			t.Errorf("%s: enum %v, want %v", key, got.Enum, w.Enum)
		}
		if !equalFloat(got.Min, w.Min) || !equalFloat(got.Max, w.Max) {
			t.Errorf("%s: range [%v, %v], want [%v, %v]", key, got.Min, got.Max, w.Min, w.Max)
		}
		if got.Description == "" || utf8.RuneCountInString(got.Description) > 200 {
			t.Errorf("%s: description must be 1 to 200 characters", key)
		}
	}
}

func TestSearchManifestDeclaresItsPromises(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "modules", "prism-search", "module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Permissions   []string `json:"permissions"`
		Commands      []string `json:"commands"`
		Helpers       []struct{ Name, Path string }
		Contributions []struct {
			Target struct{ Module, Point string }
			File   string
		} `json:"contributions"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(raw.Permissions, []string{"contribute:prism", "helpers"}) {
		t.Errorf("permissions = %v", raw.Permissions)
	}
	if len(raw.Commands) != 0 {
		t.Errorf("Feature VI adds no command (Principle VIII), got %v", raw.Commands)
	}
	if len(raw.Helpers) != 1 || raw.Helpers[0].Name != "search" || raw.Helpers[0].Path != "bin/search" {
		t.Errorf("helpers = %+v", raw.Helpers)
	}
	if len(raw.Contributions) != 1 || raw.Contributions[0].Target.Module != "prism" || raw.Contributions[0].Target.Point != "sources" || raw.Contributions[0].File != "Source.qml" {
		t.Errorf("contributions = %+v", raw.Contributions)
	}
	if !strings.Contains(strings.ToLower(raw.Description), "never takes the keyboard") {
		t.Errorf("the description must say that the module never takes the keyboard: %q", raw.Description)
	}
	for _, name := range []string{"Source.qml", "Main.qml", "stages.js"} {
		if _, err := os.Stat(filepath.Join(repoRoot, "modules", "prism-search", name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSearchManifestIsValidForTheModulesProgram(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot, "modules", "prism-search"), filepath.Join(dir, "prism-search"))
	mods := byID(scanWithAPI(t, dir, "", "1.3"))
	if m := mods["prism-search"]; !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("prism-search manifest: valid=%v errors=%+v", m.Valid, m.Errors)
	}
}

func TestSearchDefaultSearchURLIsValidForTheHelper(t *testing.T) {
	m := readModuleJSON(t, "modules/prism-search")
	url, _ := m.Settings["searchUrl"].Default.(string)
	if !(strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) || strings.Count(url, "%s") != 1 {
		t.Fatalf("the default engine %q would be refused by the helper", url)
	}
}

// A wrong value falls back for that key only, and the error names the module and the key.
func TestSearchWrongValuesFallBackPerKey(t *testing.T) {
	m := readModuleJSON(t, "modules/prism-search")
	rules := map[string]any{}
	for k, v := range m.Settings {
		r := map[string]any{"type": v.Type, "default": v.Default}
		if v.Enum != nil {
			r["enum"] = v.Enum
		}
		if v.Min != nil {
			r["min"] = *v.Min
		}
		if v.Max != nil {
			r["max"] = *v.Max
		}
		if v.MaxLength != nil {
			r["maxLength"] = *v.MaxLength
		}
		rules[k] = r
	}
	def := map[string]any{"id": "prism-search", "defaultEnabled": true, "settings": rules}
	user := map[string]any{"modules": map[string]any{"prism-search": map[string]any{
		"rowsPerGroup":   0,        // below the minimum
		"maxFiles":       "many",   // wrong type
		"grepCase":       "loose",  // not in the list
		"hiddenFiles":    "yes",    // wrong type
		"grepMaxMatches": 99999999, // above the maximum
		"systemGroup":    false,    // valid
		"searchTimeMs":   2000,     // valid
	}}}
	out := resolveWithModules(t, writeFile(t, mustJSON(user)), def)
	bad := map[string]bool{}
	for _, e := range out.Errors {
		for _, key := range []string{"rowsPerGroup", "maxFiles", "grepCase", "hiddenFiles", "grepMaxMatches", "systemGroup", "searchTimeMs"} {
			if e.Key == "modules.prism-search."+key {
				bad[key] = true
			}
		}
	}
	for _, key := range []string{"rowsPerGroup", "maxFiles", "grepCase", "hiddenFiles", "grepMaxMatches"} {
		if !bad[key] {
			t.Errorf("%s: no error naming the key (errors: %+v)", key, out.Errors)
		}
	}
	for _, key := range []string{"systemGroup", "searchTimeMs"} {
		if bad[key] {
			t.Errorf("%s is valid but was reported", key)
		}
	}
}

func TestSearchUnsetFlagHasNoDefaultInTheMergedSchema(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot, "modules", "prism-search"), filepath.Join(dir, "prism-search"))
	e, r := runModules(t, "schema", map[string]any{"bundledDir": dir, "userDir": "", "apiVersion": "1.3", "dataDir": dataDir()})
	if !e.OK {
		t.Fatalf("schema failed: %+v\nstderr=%s", e.Error, r.stderr)
	}
	var out struct {
		Schema map[string]any `json:"schema"`
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	props := out.Schema["properties"].(map[string]any)["modules"].(map[string]any)["properties"].(map[string]any)
	search, ok := props["prism-search"].(map[string]any)["properties"].(map[string]any)
	if !ok {
		t.Fatal("prism-search is missing from the merged schema")
	}
	for key, want := range searchSettings() {
		prop, ok := search[key].(map[string]any)
		if !ok {
			t.Errorf("%s missing from the merged schema", key)
			continue
		}
		_, hasDefault := prop["default"]
		if want.Default == nil && hasDefault {
			t.Errorf("%s: an unset key must have no default in the merged schema", key)
		}
		if want.Default != nil && !hasDefault {
			t.Errorf("%s: expected a default in the merged schema", key)
		}
	}
}
