package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fakeMatugen(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot, "scripts", "tests", "fixtures", "matugen-fake.sh")
	if st, err := os.Stat(p); err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("fixture %s must exist and be executable", p)
	}
	return p
}

func fakeWallpaper(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wall.png")
	if err := os.WriteFile(p, []byte("not really an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func wallpaperRequest(t *testing.T, mode string, extra map[string]any) map[string]any {
	in := map[string]any{"source": "wallpaper", "wallpaper": fakeWallpaper(t), "mode": mode, "scheme": "scheme-tonal-spot"}
	for k, v := range extra {
		in[k] = v
	}
	return in
}

func TestWallpaperLightAndDarkAreCompliant(t *testing.T) {
	env := []string{"LUMEN_MATUGEN=" + fakeMatugen(t)}
	light, _ := runPalette(t, env, wallpaperRequest(t, "light", nil))
	dark, _ := runPalette(t, env, wallpaperRequest(t, "dark", nil))
	for label, p := range map[string]paletteResult{"light": light, "dark": dark} {
		if p.Status != "ok" || p.Palette.ID != "wallpaper" || p.Palette.Mode != label {
			t.Errorf("%s: status %q id %q mode %q reason %q", label, p.Status, p.Palette.ID, p.Palette.Mode, p.Reason)
		}
		assertCompliant(t, label, p)
	}
	if light.Palette.Roles["background"] == dark.Palette.Roles["background"] {
		t.Error("light and dark palettes should differ")
	}
	if lum(t, light.Palette.Roles["background"]) <= lum(t, dark.Palette.Roles["background"]) {
		t.Error("light background is not lighter than dark background")
	}
}

func TestWallpaperLowContrastIsAdjusted(t *testing.T) {
	env := []string{"LUMEN_MATUGEN=" + fakeMatugen(t), "FAKE_MATUGEN=lowcontrast"}
	for _, mode := range []string{"light", "dark"} {
		p, _ := runPalette(t, env, wallpaperRequest(t, mode, nil))
		if p.Status != "ok" || len(p.Adjusted) == 0 {
			t.Fatalf("%s: status %s adjusted %v", mode, p.Status, p.Adjusted)
		}
		assertCompliant(t, mode+" adjusted", p)
	}
}

func TestWallpaperFailuresDegrade(t *testing.T) {
	cases := map[string]map[string]any{
		"fail":    {},
		"badjson": {},
		"missing": {},
		"hang":    {"matugenTimeoutMs": 300},
	}
	for behavior, extra := range cases {
		t.Run(behavior, func(t *testing.T) {
			env := []string{"LUMEN_MATUGEN=" + fakeMatugen(t), "FAKE_MATUGEN=" + behavior}
			p, _ := runPalette(t, env, wallpaperRequest(t, "dark", extra))
			if p.Status != "fallback" || p.Reason == "" {
				t.Fatalf("status %q reason %q", p.Status, p.Reason)
			}
			assertCompliant(t, behavior, p)
			p, _ = runPalette(t, env, wallpaperRequest(t, "dark", map[string]any{"hasPrevious": true, "matugenTimeoutMs": 300}))
			if p.Status != "stale" {
				t.Fatalf("with previous: status %q", p.Status)
			}
		})
	}
}

func TestWallpaperMissingMatugen(t *testing.T) {
	env := []string{"PATH=/nonexistent", "LUMEN_MATUGEN="}
	p, _ := runPalette(t, env, wallpaperRequest(t, "dark", nil))
	if p.Status != "fallback" || p.Reason == "" {
		t.Fatalf("status %q reason %q", p.Status, p.Reason)
	}
	assertCompliant(t, "no matugen", p)
}

func TestWallpaperFileProblems(t *testing.T) {
	env := []string{"LUMEN_MATUGEN=" + fakeMatugen(t)}
	for name, wp := range map[string]string{
		"missing file": filepath.Join(t.TempDir(), "gone.png"),
		"directory":    t.TempDir(),
	} {
		in := wallpaperRequest(t, "dark", map[string]any{"wallpaper": wp, "hasPrevious": true})
		p, _ := runPalette(t, env, in)
		if p.Status != "stale" || p.Reason == "" {
			t.Errorf("%s: status %q reason %q", name, p.Status, p.Reason)
		}
	}
}

func TestWallpaperResultIdentifiesTheFile(t *testing.T) {
	env := []string{"LUMEN_MATUGEN=" + fakeMatugen(t)}
	in := wallpaperRequest(t, "dark", nil)
	r := runProgram("palette", mustJSON(withDataDir(in)), env)
	e := parseEnvelope(t, r)
	var out struct {
		Wallpaper      string `json:"wallpaper"`
		WallpaperMtime int64  `json:"wallpaperMtime"`
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Wallpaper != in["wallpaper"] || out.WallpaperMtime == 0 {
		t.Fatalf("wallpaper %q mtime %d", out.Wallpaper, out.WallpaperMtime)
	}
}

func TestDetectOnlyReportsPathAndMtimeWithoutMatugen(t *testing.T) {
	wp := fakeWallpaper(t)
	r := runProgram("palette", mustJSON(map[string]any{
		"dataDir": dataDir(), "source": "wallpaper", "wallpaper": wp, "detectOnly": true,
	}), []string{"PATH=/nonexistent"})
	e := parseEnvelope(t, r)
	if !e.OK {
		t.Fatalf("response = %+v", e)
	}
	var out struct {
		Wallpaper string `json:"wallpaper"`
		Mtime     int64  `json:"wallpaperMtime"`
	}
	_ = json.Unmarshal(e.Data, &out)
	if out.Wallpaper != wp || out.Mtime == 0 {
		t.Fatalf("data = %s", e.Data)
	}
}

func withDataDir(in map[string]any) map[string]any {
	in["dataDir"] = dataDir()
	return in
}

func TestWallpaperNoneDetected(t *testing.T) {
	env := []string{"LUMEN_MATUGEN=" + fakeMatugen(t), "PATH=" + filepath.Join(repoRoot, "scripts", "tests", "fixtures", "nothing-here")}
	in := map[string]any{"source": "wallpaper", "mode": "dark"}
	p, _ := runPalette(t, env, in)
	if p.Status != "fallback" || p.Reason == "" {
		t.Fatalf("status %q reason %q", p.Status, p.Reason)
	}
}
