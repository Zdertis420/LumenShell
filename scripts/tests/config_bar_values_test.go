package tests

import (
	"reflect"
	"testing"
)

// Bad values in the bar's settings (specs/004-bar-customization-styles, SC-004, FR-026): each
// is an error for that key only, the key falls back, and every other key keeps its value.
// The rules are the real ones, read from modules/bar/module.json.

// barModuleDefinition is the bar's settings as the Modules singleton passes them to the helper.
func barModuleDefinition(t *testing.T) map[string]any {
	t.Helper()
	m := readModuleJSON(t, "modules/bar")
	settings := map[string]any{}
	for key, r := range m.Settings {
		rule := map[string]any{"type": r.Type, "default": r.Default}
		if len(r.Enum) > 0 {
			enum := make([]any, len(r.Enum))
			for i, e := range r.Enum {
				enum[i] = e
			}
			rule["enum"] = enum
		}
		if r.Min != nil {
			rule["min"] = *r.Min
		}
		if r.Max != nil {
			rule["max"] = *r.Max
		}
		if r.MaxLength != nil {
			rule["maxLength"] = *r.MaxLength
		}
		settings[key] = rule
	}
	return map[string]any{"id": "bar", "defaultEnabled": true, "settings": settings}
}

func TestEachBadBarValueFallsBackForItsKeyOnly(t *testing.T) {
	def := barModuleDefinition(t)
	good := `"margin": 6, "radius": 14, "frameRadius": 20, "backgroundOpacity": 0.5, "textRole": "error", "fontSize": 15`
	cases := []struct {
		key      string
		value    string
		fallback any // the value after the error: the default, or nil for an unset key
	}{
		{"style", `"pills"`, "solid"},
		{"edge", `"middle"`, "top"},
		{"thickness", `-4`, nil},
		{"thickness", `4`, nil},
		{"thickness", `1000`, nil},
		{"margin", `5000`, nil},
		{"padding", `-1`, nil},
		{"itemGap", `"wide"`, nil},
		{"radius", `true`, nil},
		{"frameBorder", `900`, nil},
		{"fontSize", `0`, nil},
		{"fontSize", `"big"`, nil},
		{"fontWeight", `"extra"`, nil},
		{"fontItalic", `"yes"`, false},
		{"backgroundRole", `"sparkles"`, "surface"},
		{"islandRole", `7`, "surface"},
		{"frameRole", `"nope"`, "surface"},
		{"textRole", `""`, "onSurface"},
		{"backgroundOpacity", `3`, float64(1)},
		{"islandOpacity", `-0.1`, float64(1)},
		{"frameOpacity", `"half"`, float64(1)},
		{"popupBorderWidth", `99`, float64(1)},
		{"popupBackground", `"glass"`, "surfaceContainer"},
		{"screens", `"all"`, []any{"all"}},
		{"left", `7`, []any{}},
	}
	for _, c := range cases {
		user := writeFile(t, `{"modules":{"bar":{`+good+`, "`+c.key+`": `+c.value+`}}}`)
		out := resolveWithModules(t, user, def)
		bar := modulesOf(out.Values, "bar")
		var errs []string
		for _, e := range out.Errors {
			errs = append(errs, e.Key)
		}
		want := "modules.bar." + c.key
		if len(errs) != 1 || errs[0] != want {
			t.Errorf("%s = %s: errors %v, want exactly [%s]", c.key, c.value, errs, want)
			continue
		}
		if got := bar[c.key]; !reflect.DeepEqual(got, c.fallback) {
			t.Errorf("%s = %s: fell back to %#v, want %#v", c.key, c.value, got, c.fallback)
		}
		// Every other key keeps the value the user gave it.
		for key, want := range map[string]any{"margin": float64(6), "radius": float64(14), "frameRadius": float64(20), "backgroundOpacity": 0.5, "textRole": "error", "fontSize": float64(15)} {
			if key == c.key {
				continue
			}
			if bar[key] != want {
				t.Errorf("%s = %s: %s is %#v, want %#v", c.key, c.value, key, bar[key], want)
			}
		}
	}
}

func TestSeveralBadBarValuesAreAllReported(t *testing.T) {
	def := barModuleDefinition(t)
	user := writeFile(t, `{"modules":{"bar":{"thickness": -4, "margin": 5000, "fontSize": "big", "backgroundRole": "sparkles", "islandOpacity": 3, "radius": 12}}}`)
	out := resolveWithModules(t, user, def)
	got := map[string]bool{}
	for _, e := range out.Errors {
		got[e.Key] = true
	}
	for _, key := range []string{"thickness", "margin", "fontSize", "backgroundRole", "islandOpacity"} {
		if !got["modules.bar."+key] {
			t.Errorf("a bad %s is not reported: %v", key, got)
		}
	}
	if got["modules.bar.radius"] {
		t.Errorf("a good radius must not be reported")
	}
	if v := modulesOf(out.Values, "bar")["radius"]; v != float64(12) {
		t.Errorf("the good radius must be applied, got %#v", v)
	}
}

func TestEveryBarKeyAcceptsItsDocumentedExtremes(t *testing.T) {
	def := barModuleDefinition(t)
	m := readModuleJSON(t, "modules/bar")
	for key, r := range m.Settings {
		if r.Type != "integer" || r.Min == nil || r.Max == nil {
			continue
		}
		for _, v := range []float64{*r.Min, *r.Max} {
			user := writeFile(t, `{"modules":{"bar":{"`+key+`": `+formatNumber(v)+`}}}`)
			out := resolveWithModules(t, user, def)
			if len(out.Errors) != 0 {
				t.Errorf("%s = %v is inside its range but was refused: %+v", key, v, out.Errors)
			}
		}
	}
}

func formatNumber(v float64) string {
	return string(mustJSONBytes(v))
}

func mustJSONBytes(v any) []byte { return []byte(mustJSON(v)) }
