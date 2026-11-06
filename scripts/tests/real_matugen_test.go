package tests

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// writeTestImage writes a PNG with several distinct color regions, so matugen finds
// multiple candidate source colors (which is when it needs --prefer).
func writeTestImage(t *testing.T, name string, colors []color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	for y := 0; y < 96; y++ {
		for x := 0; x < 96; x++ {
			img.Set(x, y, colors[(x/32+3*(y/32))%len(colors)])
		}
	}
	p := filepath.Join(t.TempDir(), name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

// Runs the real matugen (skipped when it is not installed): every scheme, light and
// dark, on a dark and a light image, must give a compliant palette.
func TestRealMatugenAllSchemesAreCompliant(t *testing.T) {
	if _, err := exec.LookPath("matugen"); err != nil {
		t.Skip("matugen is not installed")
	}
	images := map[string]string{
		"colorful": writeTestImage(t, "colorful.png", []color.RGBA{{200, 40, 40, 255}, {40, 160, 60, 255}, {40, 90, 200, 255}, {240, 200, 40, 255}}),
		"dark":     writeTestImage(t, "dark.png", []color.RGBA{{10, 12, 20, 255}, {30, 25, 50, 255}, {15, 40, 35, 255}}),
		"light":    writeTestImage(t, "light.png", []color.RGBA{{245, 240, 230, 255}, {230, 235, 250, 255}, {250, 225, 225, 255}}),
	}
	schemes := []string{"scheme-content", "scheme-expressive", "scheme-fidelity", "scheme-fruit-salad", "scheme-monochrome",
		"scheme-neutral", "scheme-rainbow", "scheme-tonal-spot", "scheme-vibrant", "scheme-smart"}
	var slowest time.Duration
	for imgName, imgPath := range images {
		for _, scheme := range schemes {
			for _, mode := range []string{"light", "dark"} {
				start := time.Now()
				p, _ := runPalette(t, nil, map[string]any{
					"source": "wallpaper", "wallpaper": imgPath, "scheme": scheme, "mode": mode, "matugenTimeoutMs": 5000,
				})
				if d := time.Since(start); d > slowest {
					slowest = d
				}
				label := imgName + "/" + scheme + "/" + mode
				if p.Status != "ok" || p.Palette.Mode != mode {
					t.Errorf("%s: status %q mode %q reason %q", label, p.Status, p.Palette.Mode, p.Reason)
					continue
				}
				assertCompliant(t, label, p)
			}
		}
	}
	t.Logf("slowest wallpaper palette (matugen included): %s", slowest)
}
