package tests

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var roleList = []string{
	"primary", "onPrimary", "secondary", "onSecondary", "tertiary", "onTertiary",
	"error", "onError", "background", "onBackground", "surface", "onSurface",
	"surfaceVariant", "onSurfaceVariant", "surfaceContainer", "outline",
}

var textPairs = [][2]string{
	{"onPrimary", "primary"}, {"onSecondary", "secondary"}, {"onTertiary", "tertiary"},
	{"onError", "error"}, {"onBackground", "background"}, {"onSurface", "surface"},
	{"onSurfaceVariant", "surfaceVariant"}, {"onSurface", "surfaceContainer"},
}

type paletteResult struct {
	Palette struct {
		ID    string            `json:"id"`
		Mode  string            `json:"mode"`
		Roles map[string]string `json:"roles"`
	} `json:"palette"`
	Status   string   `json:"status"`
	Reason   string   `json:"reason"`
	Adjusted []string `json:"adjusted"`
}

func runPalette(t *testing.T, env []string, in map[string]any) (paletteResult, result) {
	t.Helper()
	if _, ok := in["dataDir"]; !ok {
		in["dataDir"] = dataDir()
	}
	r := runProgram("palette", mustJSON(in), env)
	e := parseEnvelope(t, r)
	if !e.OK {
		t.Fatalf("palette failed: %+v", e.Error)
	}
	var out paletteResult
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out, r
}

func lum(t *testing.T, hex string) float64 {
	t.Helper()
	if len(hex) != 7 || hex[0] != '#' {
		t.Fatalf("not a #rrggbb color: %q", hex)
	}
	var c [3]float64
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		if err != nil {
			t.Fatalf("bad color %q", hex)
		}
		f := float64(v) / 255
		if f <= 0.03928 {
			c[i] = f / 12.92
		} else {
			c[i] = math.Pow((f+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

func ratio(t *testing.T, a, b string) float64 {
	la, lb := lum(t, a), lum(t, b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// assertCompliant checks that all roles exist and every text pair reaches WCAG AA.
func assertCompliant(t *testing.T, label string, p paletteResult) {
	t.Helper()
	for _, name := range roleList {
		if _, ok := p.Palette.Roles[name]; !ok {
			t.Errorf("%s: role %s missing", label, name)
		}
	}
	for _, pair := range textPairs {
		if r := ratio(t, p.Palette.Roles[pair[0]], p.Palette.Roles[pair[1]]); r < 4.5 {
			t.Errorf("%s: %s on %s has contrast %.2f < 4.5", label, pair[0], pair[1], r)
		}
	}
}

func presetNames(t *testing.T) []string {
	files, _ := filepath.Glob(filepath.Join(dataDir(), "presets", "*.json"))
	if len(files) < 5 {
		t.Fatalf("expected at least 5 presets, found %d", len(files))
	}
	var ids []string
	for _, f := range files {
		ids = append(ids, f[len(filepath.Join(dataDir(), "presets"))+1:len(f)-len(".json")])
	}
	return ids
}

func TestEveryPresetIsCompliant(t *testing.T) {
	for _, id := range presetNames(t) {
		p, _ := runPalette(t, nil, map[string]any{"source": "preset", "preset": id})
		if p.Status != "ok" || p.Palette.ID != id {
			t.Errorf("%s: status %q id %q reason %q", id, p.Status, p.Palette.ID, p.Reason)
		}
		assertCompliant(t, id, p)
	}
}

func TestExpectedPresetsExist(t *testing.T) {
	have := map[string]bool{}
	for _, id := range presetNames(t) {
		have[id] = true
	}
	for _, want := range []string{"tokyo-night", "catppuccin-latte", "catppuccin-frappe", "catppuccin-macchiato", "catppuccin-mocha"} {
		if !have[want] {
			t.Errorf("preset %s is missing", want)
		}
	}
}

func TestViolatingPresetIsAdjusted(t *testing.T) {
	dir := t.TempDir()
	presets := filepath.Join(dir, "presets")
	if err := os.MkdirAll(presets, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dataDir(), "presets", "tokyo-night.json"))
	var doc map[string]any
	_ = json.Unmarshal(b, &doc)
	doc["roles"].(map[string]any)["onPrimary"] = "#7aa2f8" // nearly the same as primary
	b, _ = json.Marshal(doc)
	if err := os.WriteFile(filepath.Join(presets, "bad-contrast.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	p, _ := runPalette(t, nil, map[string]any{"source": "preset", "preset": "bad-contrast", "dataDir": dir})
	if p.Status != "ok" {
		t.Fatalf("status = %s (%s)", p.Status, p.Reason)
	}
	found := false
	for _, a := range p.Adjusted {
		if a == "onPrimary" {
			found = true
		}
	}
	if !found {
		t.Fatalf("onPrimary should be reported as adjusted: %v", p.Adjusted)
	}
	assertCompliant(t, "adjusted", p)
}

func TestBrokenPresetsFallBack(t *testing.T) {
	dir := t.TempDir()
	presets := filepath.Join(dir, "presets")
	_ = os.MkdirAll(presets, 0o755)
	_ = os.WriteFile(filepath.Join(presets, "notjson.json"), []byte("{oops"), 0o644)
	_ = os.WriteFile(filepath.Join(presets, "norole.json"), []byte(`{"id":"norole","mode":"dark","roles":{"primary":"#112233"}}`), 0o644)
	_ = os.WriteFile(filepath.Join(presets, "badcolor.json"), []byte(`{"id":"badcolor","mode":"dark","roles":{"primary":"red"}}`), 0o644)

	for _, id := range []string{"notjson", "norole", "badcolor", "missing", "../escape", ""} {
		p, _ := runPalette(t, nil, map[string]any{"source": "preset", "preset": id, "dataDir": dir})
		if p.Status != "fallback" || p.Reason == "" || p.Palette.ID != "fallback" {
			t.Errorf("%q: status %q reason %q id %q", id, p.Status, p.Reason, p.Palette.ID)
		}
		assertCompliant(t, "fallback for "+id, p)

		p, _ = runPalette(t, nil, map[string]any{"source": "preset", "preset": id, "dataDir": dir, "hasPrevious": true})
		if p.Status != "stale" || p.Reason == "" {
			t.Errorf("%q with previous: status %q", id, p.Status)
		}
	}
}

func TestFallbackMatchesDataFile(t *testing.T) {
	p, _ := runPalette(t, nil, map[string]any{"source": "preset", "preset": "missing"})
	b, err := os.ReadFile(filepath.Join(dataDir(), "fallback-palette.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Mode  string            `json:"mode"`
		Roles map[string]string `json:"roles"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		t.Fatal(err)
	}
	if file.Mode != p.Palette.Mode {
		t.Errorf("mode: file %s, compiled %s", file.Mode, p.Palette.Mode)
	}
	for _, name := range roleList {
		if file.Roles[name] != p.Palette.Roles[name] {
			t.Errorf("%s: file %s, compiled %s", name, file.Roles[name], p.Palette.Roles[name])
		}
	}
	assertCompliant(t, "fallback file", p)
}

func TestPaletteBadInput(t *testing.T) {
	r := runProgram("palette", `{"source":"nope"}`, nil)
	e := parseEnvelope(t, r)
	if e.OK || e.Error.Code != "bad_input" {
		t.Fatalf("response = %+v", e)
	}
	r = runProgram("palette", `not json`, nil)
	if e := parseEnvelope(t, r); e.OK {
		t.Fatalf("response = %+v", e)
	}
}

func TestPaletteDiagnosticsAreNDJSON(t *testing.T) {
	r := runProgram("palette", mustJSON(map[string]any{"source": "preset", "preset": "missing", "dataDir": dataDir()}), nil)
	if r.stderr == "" {
		t.Fatal("expected a diagnostic line on stderr")
	}
}
