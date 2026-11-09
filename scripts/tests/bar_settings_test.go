package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"
	"unicode/utf8"
)

// The bar's settings (specs/004-bar-customization-styles/contracts/bar-config-1.2.md) and the
// settings shared by the bundled item modules. These tests keep the manifests, the theme's
// role list and the contract in step.

type settingRule struct {
	Type        string   `json:"type"`
	Default     any      `json:"default"`
	Enum        []string `json:"enum"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	MaxLength   *int     `json:"maxLength"`
	Description string   `json:"description"`
}

type moduleManifest struct {
	ID       string                 `json:"id"`
	API      string                 `json:"api"`
	Settings map[string]settingRule `json:"settings"`
}

func readModuleJSON(t *testing.T, dir string) moduleManifest {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, dir, "module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m moduleManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("%s/module.json: %v", dir, err)
	}
	return m
}

// themeRoles returns the role names of singletons/Theme.qml (its `property color <name>` lines).
func themeRoles(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "singletons", "Theme.qml"))
	if err != nil {
		t.Fatal(err)
	}
	var roles []string
	for _, m := range regexp.MustCompile(`(?m)^\s*property color (\w+):`).FindAllStringSubmatch(string(b), -1) {
		roles = append(roles, m[1])
	}
	sort.Strings(roles)
	if len(roles) == 0 {
		t.Fatal("no roles found in Theme.qml")
	}
	return roles
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func f(v float64) *float64 { return &v }

// barSettings is the contract table: key, type, default (nil means unset), range.
var barSettings = map[string]settingRule{
	"edge":              {Type: "string", Default: "top", Enum: []string{"top", "bottom", "left", "right"}},
	"screens":           {Type: "stringlist", Default: []any{"all"}},
	"left":              {Type: "stringlist", Default: []any{}},
	"center":            {Type: "stringlist", Default: []any{"bar-clock"}},
	"right":             {Type: "stringlist", Default: []any{"bar-sound", "bar-battery"}},
	"focusScreen":       {Type: "string", Default: ""},
	"style":             {Type: "string", Default: "solid", Enum: []string{"solid", "islands", "frame"}},
	"thickness":         {Type: "integer", Default: nil, Min: f(16), Max: f(256)},
	"margin":            {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
	"padding":           {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
	"itemGap":           {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
	"placeGap":          {Type: "integer", Default: nil, Min: f(0), Max: f(256)},
	"itemPadding":       {Type: "integer", Default: nil, Min: f(0), Max: f(64)},
	"itemRadius":        {Type: "integer", Default: nil, Min: f(0), Max: f(64)},
	"radius":            {Type: "integer", Default: nil, Min: f(0), Max: f(256)},
	"frameRadius":       {Type: "integer", Default: nil, Min: f(0), Max: f(256)},
	"frameBorder":       {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
	"fontFamily":        {Type: "string", Default: nil},
	"fontSize":          {Type: "integer", Default: nil, Min: f(1), Max: f(96)},
	"fontWeight":        {Type: "string", Default: nil, Enum: []string{"thin", "light", "regular", "medium", "semibold", "bold", "black"}},
	"fontItalic":        {Type: "boolean", Default: false},
	"backgroundRole":    {Type: "string", Default: "surface", Enum: []string{"@roles"}},
	"backgroundOpacity": {Type: "number", Default: float64(1), Min: f(0), Max: f(1)},
	"islandRole":        {Type: "string", Default: "surface", Enum: []string{"@roles"}},
	"islandOpacity":     {Type: "number", Default: float64(1), Min: f(0), Max: f(1)},
	"frameRole":         {Type: "string", Default: "surface", Enum: []string{"@roles"}},
	"frameOpacity":      {Type: "number", Default: float64(1), Min: f(0), Max: f(1)},
	"textRole":          {Type: "string", Default: "onSurface", Enum: []string{"@roles"}},
	"focusRole":         {Type: "string", Default: "primary", Enum: []string{"@roles"}},
	"popupRadius":       {Type: "integer", Default: nil, Min: f(0), Max: f(64)},
	"popupPadding":      {Type: "integer", Default: float64(0), Min: f(0), Max: f(64)},
	"popupBorderWidth":  {Type: "integer", Default: float64(1), Min: f(0), Max: f(16)},
	"popupBackground":   {Type: "string", Default: "surfaceContainer", Enum: []string{"@roles"}},
	"popupBorder":       {Type: "string", Default: "outline", Enum: []string{"@roles"}},
}

func equalFloat(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestBarManifestMatchesTheContract(t *testing.T) {
	m := readModuleJSON(t, "modules/bar")
	if m.API != "1.2" {
		t.Errorf("api = %q, want 1.2", m.API)
	}
	roles := themeRoles(t)
	if len(m.Settings) != len(barSettings) {
		var have, want []string
		for k := range m.Settings {
			have = append(have, k)
		}
		for k := range barSettings {
			want = append(want, k)
		}
		t.Errorf("settings differ:\nmanifest: %v\ncontract: %v", sortedCopy(have), sortedCopy(want))
	}
	for key, want := range barSettings {
		got, ok := m.Settings[key]
		if !ok {
			t.Errorf("%s: missing from the manifest", key)
			continue
		}
		if got.Type != want.Type {
			t.Errorf("%s: type %q, want %q", key, got.Type, want.Type)
		}
		if !reflect.DeepEqual(got.Default, want.Default) {
			t.Errorf("%s: default %#v, want %#v", key, got.Default, want.Default)
		}
		wantEnum := want.Enum
		if len(wantEnum) == 1 && wantEnum[0] == "@roles" {
			wantEnum = roles
			if !reflect.DeepEqual(sortedCopy(got.Enum), roles) {
				t.Errorf("%s: role list %v differs from Theme.qml %v", key, sortedCopy(got.Enum), roles)
			}
		} else if !reflect.DeepEqual(got.Enum, wantEnum) {
			t.Errorf("%s: enum %v, want %v", key, got.Enum, wantEnum)
		}
		if !equalFloat(got.Min, want.Min) || !equalFloat(got.Max, want.Max) {
			t.Errorf("%s: range [%v, %v], want [%v, %v]", key, got.Min, got.Max, want.Min, want.Max)
		}
		if got.Description == "" || utf8.RuneCountInString(got.Description) > 160 {
			t.Errorf("%s: description must be 1 to 160 characters", key)
		}
	}
}

// The nine settings that the three bundled item modules share (data-model.md "Item settings").
var itemSettings = []string{"fontFamily", "fontSize", "fontWeight", "fontItalic", "padding", "radius", "backgroundRole", "backgroundOpacity", "textRole"}

func TestItemModulesDeclareTheSameSharedSettings(t *testing.T) {
	roles := themeRoles(t)
	clock := readModuleJSON(t, "modules/bar-clock")
	for _, dir := range []string{"modules/bar-clock", "modules/bar-sound", "modules/bar-battery"} {
		m := readModuleJSON(t, dir)
		if m.API != "1.2" {
			t.Errorf("%s: api %q, want 1.2", dir, m.API)
		}
		for _, key := range itemSettings {
			got, ok := m.Settings[key]
			if !ok {
				t.Errorf("%s: missing the shared setting %s", dir, key)
				continue
			}
			if got.Default != nil {
				t.Errorf("%s: %s must be unset by default, got %#v", dir, key, got.Default)
			}
			want := clock.Settings[key]
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %s differs from bar-clock's: %+v vs %+v", dir, key, got, want)
			}
			if key == "backgroundRole" || key == "textRole" {
				if !reflect.DeepEqual(sortedCopy(got.Enum), roles) {
					t.Errorf("%s: %s role list differs from Theme.qml", dir, key)
				}
			}
		}
	}
	// The clock adds a format for a vertical bar: hours above minutes by default.
	vf, ok := clock.Settings["verticalFormat"]
	if !ok || vf.Default != "HH\nmm" {
		t.Errorf("bar-clock verticalFormat = %#v", vf.Default)
	}
}

func TestItemLookIsTheSameInEveryItemModule(t *testing.T) {
	first, err := os.ReadFile(filepath.Join(repoRoot, "modules", "bar-clock", "ItemLook.qml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"bar-sound", "bar-battery"} {
		other, err := os.ReadFile(filepath.Join(repoRoot, "modules", id, "ItemLook.qml"))
		if err != nil {
			t.Fatal(err)
		}
		if string(other) != string(first) {
			t.Errorf("modules/%s/ItemLook.qml differs from modules/bar-clock/ItemLook.qml", id)
		}
	}
}

func TestBarManifestIsValidForTheModulesProgram(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot, "modules", "bar"), filepath.Join(dir, "bar"))
	mods := byID(scanWithAPI(t, dir, "", "1.2"))
	if m := mods["bar"]; !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("bar manifest: valid=%v errors=%+v", m.Valid, m.Errors)
	}
}

func TestBarEdgeAcceptsFourValuesAndRejectsOthers(t *testing.T) {
	m := readModuleJSON(t, "modules/bar")
	rule := map[string]any{}
	for k, v := range m.Settings {
		rule[k] = map[string]any{"type": v.Type, "default": v.Default, "enum": v.Enum}
	}
	def := map[string]any{"id": "bar", "defaultEnabled": true, "settings": map[string]any{"edge": rule["edge"]}}
	for value, valid := range map[string]bool{"top": true, "bottom": true, "left": true, "right": true, "middle": false, "": false} {
		out := resolveWithModules(t, writeFile(t, mustJSON(map[string]any{"modules": map[string]any{"bar": map[string]any{"edge": value}}})), def)
		if got := len(out.Errors) == 0; got != valid {
			t.Errorf("edge %q: valid=%v, want %v (%+v)", value, got, valid, out.Errors)
		}
	}
}

func TestBarUnsetKeysHaveNoDefaultInTheMergedSchema(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot, "modules", "bar"), filepath.Join(dir, "bar"))
	e, r := runModules(t, "schema", map[string]any{"bundledDir": dir, "userDir": "", "apiVersion": "1.2", "dataDir": dataDir()})
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
	bar := props["bar"].(map[string]any)["properties"].(map[string]any)
	for key, want := range barSettings {
		prop, ok := bar[key].(map[string]any)
		if !ok {
			t.Errorf("%s missing from the merged schema", key)
			continue
		}
		_, has := prop["default"]
		if want.Default == nil && has {
			t.Errorf("%s is unset by default and must have no schema default: %v", key, prop)
		}
		if want.Default != nil && !has {
			t.Errorf("%s must keep its schema default", key)
		}
	}
}
