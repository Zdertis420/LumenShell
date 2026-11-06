package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
)

// palette is a set of semantic color roles plus a mode.
type palette struct {
	ID    string            `json:"id"`
	Name  string            `json:"name,omitempty"`
	Mode  string            `json:"mode"`
	Roles map[string]string `json:"roles"`
}

var roleNames = []string{
	"primary", "onPrimary", "secondary", "onSecondary", "tertiary", "onTertiary",
	"error", "onError", "background", "onBackground", "surface", "onSurface",
	"surfaceVariant", "onSurfaceVariant", "surfaceContainer", "outline",
}

// textPairs are (text role, surface role) pairs that must reach minContrast.
var textPairs = [][2]string{
	{"onPrimary", "primary"},
	{"onSecondary", "secondary"},
	{"onTertiary", "tertiary"},
	{"onError", "error"},
	{"onBackground", "background"},
	{"onSurface", "surface"},
	{"onSurfaceVariant", "surfaceVariant"},
	{"onSurface", "surfaceContainer"},
}

// minContrast is the WCAG AA minimum for normal text.
const minContrast = 4.5

// fallbackPalette is the built-in palette. It mirrors data/fallback-palette.json
// (a test checks that both match).
func fallbackPalette() palette {
	return palette{
		ID:   "fallback",
		Name: "Fallback",
		Mode: "dark",
		Roles: map[string]string{
			"primary":          "#a8c8ff",
			"onPrimary":        "#00315e",
			"secondary":        "#bcc7dc",
			"onSecondary":      "#263141",
			"tertiary":         "#dabce2",
			"onTertiary":       "#3d2846",
			"error":            "#ffb4ab",
			"onError":          "#690005",
			"background":       "#121316",
			"onBackground":     "#e3e2e6",
			"surface":          "#121316",
			"onSurface":        "#e3e2e6",
			"surfaceVariant":   "#44474e",
			"onSurfaceVariant": "#c4c6d0",
			"surfaceContainer": "#1e2022",
			"outline":          "#8e9099",
		},
	}
}

type rgb struct{ r, g, b float64 } // 0..1

var hexPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func parseHex(s string) (rgb, bool) {
	if !hexPattern.MatchString(s) {
		return rgb{}, false
	}
	v, _ := strconv.ParseUint(s[1:], 16, 32)
	return rgb{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}, true
}

func (c rgb) hex() string {
	to := func(f float64) int { return int(math.Round(math.Max(0, math.Min(1, f)) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", to(c.r), to(c.g), to(c.b))
}

func channel(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// luminance is the WCAG relative luminance.
func luminance(c rgb) float64 {
	return 0.2126*channel(c.r) + 0.7152*channel(c.g) + 0.0722*channel(c.b)
}

func contrast(a, b rgb) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func mix(c, to rgb, t float64) rgb {
	return rgb{c.r + (to.r-c.r)*t, c.g + (to.g-c.g)*t, c.b + (to.b-c.b)*t}
}

// validate checks the mode and that every role is present as #rrggbb, and normalizes
// the colors to lower case. Extra roles are dropped.
func validate(p *palette) error {
	if p.Mode != "light" && p.Mode != "dark" {
		return fmt.Errorf("mode must be light or dark, got %q", p.Mode)
	}
	clean := make(map[string]string, len(roleNames))
	for _, name := range roleNames {
		v, ok := p.Roles[name]
		if !ok {
			return fmt.Errorf("role %s is missing", name)
		}
		c, ok := parseHex(v)
		if !ok {
			return fmt.Errorf("role %s has an invalid color %q", name, v)
		}
		clean[name] = c.hex()
	}
	p.Roles = clean
	return nil
}

// enforceContrast adjusts text roles until every text pair reaches the WCAG AA minimum.
// It returns the roles it changed. A validated palette can always be made compliant,
// because black or white text reaches at least 4.58:1 on any background.
func enforceContrast(p *palette) []string {
	changed := map[string]bool{}
	white, black := rgb{1, 1, 1}, rgb{0, 0, 0}
	// A little headroom above the minimum so rounding to 8 bits cannot undercut it.
	const target = minContrast + 0.05
	for pass := 0; pass < 4; pass++ {
		clean := true
		for _, pair := range textPairs {
			text, _ := parseHex(p.Roles[pair[0]])
			surface, _ := parseHex(p.Roles[pair[1]])
			if contrast(text, surface) >= minContrast {
				continue
			}
			clean = false
			fixed, ok := adjustText(text, surface, white, black, target)
			if !ok {
				continue
			}
			p.Roles[pair[0]] = fixed.hex()
			changed[pair[0]] = true
		}
		if clean {
			break
		}
	}
	var out []string
	for _, name := range roleNames {
		if changed[name] {
			out = append(out, name)
		}
	}
	return out
}

// adjustText moves text toward white or black, whichever needs the smaller change,
// until it contrasts with surface by at least target.
func adjustText(text, surface, white, black rgb, target float64) (rgb, bool) {
	best := rgb{}
	bestStep := math.MaxInt32
	found := false
	for _, end := range []rgb{white, black} {
		for step := 1; step <= 100; step++ {
			c := mix(text, end, float64(step)/100)
			c = rgb{math.Round(c.r*255) / 255, math.Round(c.g*255) / 255, math.Round(c.b*255) / 255}
			if contrast(c, surface) >= target {
				if step < bestStep {
					best, bestStep, found = c, step, true
				}
				break
			}
		}
	}
	return best, found
}
