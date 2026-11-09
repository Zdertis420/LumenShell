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

// The Prism settings (specs/005-prism-window/contracts/prism-config.md). These tests keep the
// manifest, the theme's role list and the contract in step: 105 keys with their types,
// defaults and ranges.

const prismSettingCount = 105

var prismWeights = []string{"thin", "light", "regular", "medium", "semibold", "bold", "black"}

func prismLetters() []string {
	var out []string
	for c := 'a'; c <= 'z'; c++ {
		out = append(out, string(c))
	}
	for c := '0'; c <= '9'; c++ {
		out = append(out, string(c))
	}
	return out
}

// prismSettings builds the contract table. "@roles" stands for the role list of Theme.qml.
func prismSettings() map[string]settingRule {
	roles := []string{"@roles"}
	s := map[string]settingRule{
		// Placement and size (9)
		"anchor":         {Type: "string", Default: "top", Enum: []string{"top-left", "top", "top-right", "left", "center", "right", "bottom-left", "bottom", "bottom-right"}},
		"offsetX":        {Type: "integer", Default: float64(0), Min: f(-4096), Max: f(4096)},
		"offsetY":        {Type: "integer", Default: nil, Min: f(-4096), Max: f(4096)},
		"margin":         {Type: "integer", Default: nil, Min: f(0), Max: f(512)},
		"width":          {Type: "integer", Default: float64(640), Min: f(120), Max: f(8192)},
		"widthShare":     {Type: "number", Default: nil, Min: f(0.1), Max: f(1)},
		"maxHeight":      {Type: "integer", Default: float64(480), Min: f(60), Max: f(8192)},
		"maxHeightShare": {Type: "number", Default: nil, Min: f(0.1), Max: f(1)},
		"screen":         {Type: "string", Default: ""},
		// Behavior (8)
		"closeOnOutside": {Type: "boolean", Default: true},
		"keepText":       {Type: "boolean", Default: false},
		"wrapSelection":  {Type: "boolean", Default: false},
		"maxRows":        {Type: "integer", Default: float64(8), Min: f(1), Max: f(64)},
		"rowHeight":      {Type: "integer", Default: nil, Min: f(16), Max: f(128)},
		"iconSize":       {Type: "integer", Default: nil, Min: f(8), Max: f(128)},
		"inputPosition":  {Type: "string", Default: "top", Enum: []string{"top", "bottom"}},
		"preview":        {Type: "boolean", Default: false},
		// Content (8)
		"placeholder":   {Type: "string", Default: "Search"},
		"emptyNoSource": {Type: "string", Default: "No source is installed"},
		"emptyNoMatch":  {Type: "string", Default: "Nothing matched"},
		"showFooter":    {Type: "boolean", Default: false},
		"footerHints":   {Type: "stringlist", Default: []any{"Esc close", "↑↓ select", "Enter choose"}},
		"showSubtitle":  {Type: "boolean", Default: true},
		"showIcon":      {Type: "boolean", Default: true},
		"showHint":      {Type: "boolean", Default: true},
		// Prefix letters (5)
		"prefixApps":     {Type: "string", Default: "a", Enum: prismLetters()},
		"prefixFiles":    {Type: "string", Default: "f", Enum: prismLetters()},
		"prefixWeb":      {Type: "string", Default: "w", Enum: prismLetters()},
		"prefixCommands": {Type: "string", Default: "s", Enum: prismLetters()},
		"prefixGrep":     {Type: "string", Default: "g", Enum: prismLetters()},
		// Shape and spacing (14)
		"radius":         {Type: "integer", Default: nil, Min: f(0), Max: f(256)},
		"inputRadius":    {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"rowRadius":      {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"borderWidth":    {Type: "integer", Default: nil, Min: f(0), Max: f(32)},
		"padding":        {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"inputPadding":   {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"rowPadding":     {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"gap":            {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"rowGap":         {Type: "integer", Default: nil, Min: f(0), Max: f(64)},
		"separator":      {Type: "boolean", Default: false},
		"separatorWidth": {Type: "integer", Default: nil, Min: f(1), Max: f(16)},
		"shadow":         {Type: "boolean", Default: true},
		"shadowBlur":     {Type: "integer", Default: nil, Min: f(0), Max: f(128)},
		"shadowOffset":   {Type: "integer", Default: nil, Min: f(-64), Max: f(64)},
		// Backdrop and prefix chip (2)
		"backdrop":      {Type: "boolean", Default: false},
		"prefixOpacity": {Type: "number", Default: 0.25, Min: f(0), Max: f(1)},
		// Motion (3)
		"animation":         {Type: "string", Default: "fade", Enum: []string{"fade", "scale", "slide", "none"}},
		"animationDuration": {Type: "integer", Default: nil, Min: f(0), Max: f(2000)},
		"animationScale":    {Type: "number", Default: 0.96, Min: f(0.5), Max: f(1)},
		// Fonts, global (4)
		"fontFamily": {Type: "string", Default: nil},
		"fontSize":   {Type: "integer", Default: nil, Min: f(1), Max: f(96)},
		"fontWeight": {Type: "string", Default: nil, Enum: prismWeights},
		"fontItalic": {Type: "boolean", Default: false},
	}
	// Colors: 17 roles with their defaults, 7 opacities.
	for key, def := range map[string]string{
		"windowRole": "surfaceContainer", "borderRole": "outline", "inputRole": "surfaceVariant",
		"inputTextRole": "onSurface", "placeholderRole": "onSurfaceVariant", "prefixRole": "primary",
		"titleRole": "onSurface", "subtitleRole": "onSurfaceVariant", "hintRole": "onSurfaceVariant",
		"selectedRole": "primary", "selectedTextRole": "onPrimary", "hoverRole": "surfaceVariant",
		"separatorRole": "outline", "footerRole": "onSurfaceVariant", "emptyRole": "onSurfaceVariant",
		"shadowRole": "background", "backdropRole": "background",
	} {
		s[key] = settingRule{Type: "string", Default: def, Enum: roles}
	}
	for key, def := range map[string]float64{
		"windowOpacity": 1, "borderOpacity": 1, "inputOpacity": 1, "selectedOpacity": 1,
		"hoverOpacity": 1, "shadowOpacity": 0.35, "backdropOpacity": 0.4,
	} {
		s[key] = settingRule{Type: "number", Default: def, Min: f(0), Max: f(1)}
	}
	// Fonts per role: 7 roles by 4 keys, all unset by default.
	for _, role := range []string{"input", "placeholder", "title", "subtitle", "hint", "footer", "empty"} {
		s[role+"FontFamily"] = settingRule{Type: "string", Default: nil}
		s[role+"FontSize"] = settingRule{Type: "integer", Default: nil, Min: f(1), Max: f(96)}
		s[role+"FontWeight"] = settingRule{Type: "string", Default: nil, Enum: prismWeights}
		s[role+"FontItalic"] = settingRule{Type: "boolean", Default: nil}
	}
	return s
}

func TestPrismManifestMatchesTheContract(t *testing.T) {
	m := readModuleJSON(t, "modules/prism")
	if m.API != "1.3" {
		t.Errorf("api = %q, want 1.3", m.API)
	}
	want := prismSettings()
	if len(want) != prismSettingCount {
		t.Fatalf("the test table has %d keys, want %d", len(want), prismSettingCount)
	}
	if len(m.Settings) != prismSettingCount {
		var have, wantKeys []string
		for k := range m.Settings {
			have = append(have, k)
		}
		for k := range want {
			wantKeys = append(wantKeys, k)
		}
		t.Errorf("%d settings, want %d:\nmanifest: %v\ncontract: %v", len(m.Settings), prismSettingCount, sortedCopy(have), sortedCopy(wantKeys))
	}
	roles := themeRoles(t)
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
		if len(w.Enum) == 1 && w.Enum[0] == "@roles" {
			if !reflect.DeepEqual(sortedCopy(got.Enum), roles) {
				t.Errorf("%s: role list %v differs from Theme.qml %v", key, sortedCopy(got.Enum), roles)
			}
		} else if !reflect.DeepEqual(got.Enum, w.Enum) {
			t.Errorf("%s: enum %v, want %v", key, got.Enum, w.Enum)
		}
		if !equalFloat(got.Min, w.Min) || !equalFloat(got.Max, w.Max) {
			t.Errorf("%s: range [%v, %v], want [%v, %v]", key, got.Min, got.Max, w.Min, w.Max)
		}
		if got.Description == "" || utf8.RuneCountInString(got.Description) > 200 {
			t.Errorf("%s: description must be 1 to 200 characters", key)
		}
	}
	for key := range m.Settings {
		if _, ok := want[key]; !ok {
			t.Errorf("%s: in the manifest but not in the contract", key)
		}
	}
}

func TestPrismPrefixDefaultsAreDistinct(t *testing.T) {
	m := readModuleJSON(t, "modules/prism")
	seen := map[string]string{}
	for _, key := range []string{"prefixApps", "prefixFiles", "prefixWeb", "prefixCommands", "prefixGrep"} {
		letter, _ := m.Settings[key].Default.(string)
		if other, dup := seen[letter]; dup {
			t.Errorf("%s and %s share the default letter %q", key, other, letter)
		}
		seen[letter] = key
	}
	if len(seen) != 5 {
		t.Errorf("want 5 prefix keys, got %d", len(seen))
	}
}

func TestPrismManifestDeclaresItsPointAndCommands(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "modules", "prism", "module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Permissions     []string `json:"permissions"`
		Commands        []string `json:"commands"`
		ExtensionPoints []struct {
			ID string `json:"id"`
		} `json:"extensionPoints"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(raw.Commands, []string{"toggle", "open", "close", "clear"}) {
		t.Errorf("commands = %v", raw.Commands)
	}
	if len(raw.ExtensionPoints) != 1 || raw.ExtensionPoints[0].ID != "sources" {
		t.Errorf("extensionPoints = %+v", raw.ExtensionPoints)
	}
	if len(raw.Permissions) != 0 {
		t.Errorf("Prism needs no permission, got %v", raw.Permissions)
	}
	// Principle VIII: a module that takes the keyboard says so.
	if !strings.Contains(strings.ToLower(raw.Description), "keyboard") {
		t.Errorf("the description must say that Prism takes the keyboard: %q", raw.Description)
	}
}

func TestPrismManifestIsValidForTheModulesProgram(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot, "modules", "prism"), filepath.Join(dir, "prism"))
	mods := byID(scanWithAPI(t, dir, "", "1.3"))
	if m := mods["prism"]; !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("prism manifest: valid=%v errors=%+v", m.Valid, m.Errors)
	}
}

// A wrong value falls back for that key only, with an error that names the key.
func TestPrismWrongValuesFallBackPerKey(t *testing.T) {
	m := readModuleJSON(t, "modules/prism")
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
	rules["footerHints"] = map[string]any{"type": "stringlist", "default": []any{"Esc close"}}
	def := map[string]any{"id": "prism", "defaultEnabled": true, "settings": rules}
	user := map[string]any{"modules": map[string]any{"prism": map[string]any{
		"anchor":      "middle", // not an anchor
		"width":       10,       // below the minimum
		"windowRole":  "nope",   // not a role
		"prefixWeb":   "@",      // not a letter
		"maxRows":     "many",   // wrong type
		"placeholder": "Find",   // valid
		"radius":      12,       // valid
	}}}
	out := resolveWithModules(t, writeFile(t, mustJSON(user)), def)
	bad := map[string]bool{}
	for _, e := range out.Errors {
		for _, key := range []string{"anchor", "width", "windowRole", "prefixWeb", "maxRows", "placeholder", "radius"} {
			if e.Key == "modules.prism."+key {
				bad[key] = true
			}
		}
	}
	for _, key := range []string{"anchor", "width", "windowRole", "prefixWeb", "maxRows"} {
		if !bad[key] {
			t.Errorf("%s: no error naming the key (errors: %+v)", key, out.Errors)
		}
	}
	for _, key := range []string{"placeholder", "radius"} {
		if bad[key] {
			t.Errorf("%s is valid but was reported", key)
		}
	}
}

func TestPrismUnsetKeysHaveNoDefaultInTheMergedSchema(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot, "modules", "prism"), filepath.Join(dir, "prism"))
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
	prism, ok := props["prism"].(map[string]any)["properties"].(map[string]any)
	if !ok {
		t.Fatal("prism is missing from the merged schema")
	}
	for key, want := range prismSettings() {
		prop, ok := prism[key].(map[string]any)
		if !ok {
			t.Errorf("%s missing from the merged schema", key)
			continue
		}
		_, hasDefault := prop["default"]
		if want.Default == nil && hasDefault {
			t.Errorf("%s: unset keys must have no default in the merged schema", key)
		}
		if want.Default != nil && !hasDefault {
			t.Errorf("%s: expected a default in the merged schema", key)
		}
	}
}
