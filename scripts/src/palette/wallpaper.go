package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// matugenRoles maps our semantic roles to matugen's color names
// (verified against matugen 4.2.0, task T035a).
var matugenRoles = map[string]string{
	"primary":          "primary",
	"onPrimary":        "on_primary",
	"secondary":        "secondary",
	"onSecondary":      "on_secondary",
	"tertiary":         "tertiary",
	"onTertiary":       "on_tertiary",
	"error":            "error",
	"onError":          "on_error",
	"background":       "background",
	"onBackground":     "on_background",
	"surface":          "surface",
	"onSurface":        "on_surface",
	"surfaceVariant":   "surface_variant",
	"onSurfaceVariant": "on_surface_variant",
	"surfaceContainer": "surface_container",
	"outline":          "outline",
}

var wallpaperLine = regexp.MustCompile(`currently displaying: image: (.+)$`)

// detectWallpaper asks the session which wallpaper is set. This is compositor/session
// specific (niri has no wallpaper of its own): it tries awww, then swww.
// Returns "" when nothing is found.
func detectWallpaper() string {
	for _, tool := range []string{"awww", "swww"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		out, err := exec.CommandContext(ctx, path, "query").Output()
		cancel()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if m := wallpaperLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				return strings.TrimSpace(m[1])
			}
		}
	}
	return ""
}

func findMatugen() string {
	if p := os.Getenv("LUMEN_MATUGEN"); p != "" {
		return p
	}
	p, err := exec.LookPath("matugen")
	if err != nil {
		return ""
	}
	return p
}

// currentWallpaper returns the wallpaper path from the request or the session.
func currentWallpaper(req request) string {
	if req.Wallpaper != "" {
		return req.Wallpaper
	}
	return detectWallpaper()
}

// mtime returns the modification time of path in unix seconds (0 when unavailable).
func mtime(path string) int64 {
	if path == "" {
		return 0
	}
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.ModTime().Unix()
}

// fromWallpaper builds a palette from the current wallpaper and reports which file
// and modification time it used, so the shell can tell when the wallpaper changes.
func fromWallpaper(req request) result {
	wp := currentWallpaper(req)
	res := buildWallpaper(req, wp)
	res.Wallpaper = wp
	res.WallpaperMtime = mtime(wp)
	return res
}

func buildWallpaper(req request, wp string) result {
	mode := req.Mode
	if mode == "" {
		mode = "dark"
	}
	scheme := req.Scheme
	if scheme == "" {
		scheme = "scheme-tonal-spot"
	}
	if mode != "light" && mode != "dark" {
		return degrade(req, "invalid mode "+quote(mode))
	}

	matugen := findMatugen()
	if matugen == "" {
		return degrade(req, "matugen is not installed")
	}

	if wp == "" {
		return degrade(req, "no wallpaper found (set theme.wallpaper or install awww/swww)")
	}
	if st, err := os.Stat(wp); err != nil || st.IsDir() {
		return degrade(req, "wallpaper file is missing or unreadable: "+wp)
	}

	limit := time.Duration(req.MatugenTimeoutMs) * time.Millisecond
	if limit <= 0 {
		limit = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	// --dry-run keeps matugen from writing templates or running hooks from the user's setup.
	// --prefer is required: without it matugen asks on a terminal when an image has several
	// candidate source colors, and fails when none is attached.
	cmd := exec.CommandContext(ctx, matugen, "image", wp, "--json", "hex", "--type", scheme, "--mode", mode, "--prefer", "saturation", "--dry-run")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// Kill matugen's whole process group on timeout, and do not wait for pipes that a
	// stray child might keep open.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 200 * time.Millisecond
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return degrade(req, fmt.Sprintf("matugen did not finish within %d ms", limit.Milliseconds()))
	}
	if err != nil {
		return degrade(req, "matugen failed: "+err.Error()+": "+strings.TrimSpace(stderr.String()))
	}

	p, err := paletteFromMatugen(out, mode)
	if err != nil {
		return degrade(req, "matugen output rejected: "+err.Error())
	}
	adjusted := enforceContrast(&p)
	if adjusted == nil {
		adjusted = []string{}
	} else {
		logf("info", "adjusted text colors to meet contrast", map[string]any{"roles": adjusted})
	}
	return result{Palette: p, Status: "ok", Adjusted: adjusted}
}

// paletteFromMatugen maps matugen's JSON output to our roles. The shape (verified with
// matugen 4.2.0, `--json hex`) is:
//
//	{"colors": {"on_primary": {"dark": {"color": "#192e60"}, "default": {...}, "light": {...}}, ...}}
func paletteFromMatugen(raw []byte, mode string) (palette, error) {
	var doc struct {
		Colors map[string]json.RawMessage `json:"colors"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return palette{}, fmt.Errorf("not valid JSON: %v", err)
	}
	if len(doc.Colors) == 0 {
		return palette{}, fmt.Errorf("no colors in output")
	}

	pick := func(matugenName string) (string, bool) {
		rawName, ok := doc.Colors[matugenName]
		if !ok {
			return "", false
		}
		var byMode map[string]json.RawMessage
		if json.Unmarshal(rawName, &byMode) != nil {
			return "", false
		}
		return colorString(byMode[mode])
	}

	p := palette{ID: "wallpaper", Name: "Wallpaper", Mode: mode, Roles: map[string]string{}}
	for role, name := range matugenRoles {
		s, ok := pick(name)
		if !ok {
			return palette{}, fmt.Errorf("color %s (%s) not found", name, role)
		}
		p.Roles[role] = s
	}
	if err := validate(&p); err != nil {
		return palette{}, err
	}
	return p, nil
}

// colorString reads either "#rrggbb" or {"color": "#rrggbb"}.
func colorString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, true
	}
	var obj struct {
		Color string `json:"color"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Color != "" {
		return obj.Color, true
	}
	return "", false
}
