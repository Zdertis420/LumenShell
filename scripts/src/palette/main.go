// Command palette produces a Lumen color palette from a bundled preset or a wallpaper.
//
// One JSON object on stdin, one JSON document on stdout.
// See specs/001-foundation/contracts/helper-protocol.md.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"
)

const protocol = 1

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type response struct {
	OK       bool           `json:"ok"`
	Protocol int            `json:"protocol"`
	Data     any            `json:"data,omitempty"`
	Error    *responseError `json:"error,omitempty"`
}

type request struct {
	DataDir     string `json:"dataDir"`
	Source      string `json:"source"`
	Preset      string `json:"preset"`
	Wallpaper   string `json:"wallpaper"`
	Scheme      string `json:"scheme"`
	Mode        string `json:"mode"`
	HasPrevious bool   `json:"hasPrevious"`
	// MatugenTimeoutMs bounds the matugen run (default 2000, below Backend's call limit).
	MatugenTimeoutMs int `json:"matugenTimeoutMs"`
	// DetectOnly (wallpaper source) only reports the current wallpaper path and mtime.
	DetectOnly bool `json:"detectOnly"`
}

type result struct {
	Palette  palette  `json:"palette"`
	Status   string   `json:"status"` // ok, fallback, stale
	Reason   string   `json:"reason,omitempty"`
	Adjusted []string `json:"adjusted"`
	// Wallpaper and WallpaperMtime identify the wallpaper a wallpaper palette came from.
	Wallpaper      string `json:"wallpaper,omitempty"`
	WallpaperMtime int64  `json:"wallpaperMtime,omitempty"`
}

var logLevels = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

// logf writes one ndjson diagnostic line to stderr, filtered by LUMEN_LOG_LEVEL.
func logf(level, msg string, fields map[string]any) {
	min, ok := logLevels[os.Getenv("LUMEN_LOG_LEVEL")]
	if !ok {
		min = logLevels["info"]
	}
	if logLevels[level] < min {
		return
	}
	entry := map[string]any{
		"time":   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"level":  level,
		"source": "helper:palette",
		"msg":    msg,
	}
	for k, v := range fields {
		if _, exists := entry[k]; !exists {
			entry[k] = v
		}
	}
	b, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(b))
}

func writeResponse(r response) {
	r.Protocol = protocol
	b, err := json.Marshal(r)
	if err != nil {
		b = []byte(`{"ok":false,"protocol":1,"error":{"code":"internal","message":"cannot encode response"}}`)
	}
	fmt.Println(string(b))
}

func fail(code, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	logf("warn", msg, map[string]any{"code": code})
	writeResponse(response{OK: false, Error: &responseError{Code: code, Message: msg}})
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// degrade builds the result used when the requested palette is unavailable.
// With a previous palette available, status is "stale" and Backend keeps the old colors.
func degrade(req request, reason string) result {
	status := "fallback"
	if req.HasPrevious {
		status = "stale"
	}
	logf("warn", "palette unavailable, using "+status+": "+reason, map[string]any{"reason": reason})
	return result{Palette: fallbackPalette(), Status: status, Reason: reason, Adjusted: []string{}}
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail("bad_input", "cannot read stdin: %v", err)
		return
	}
	var req request
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			fail("bad_input", "input must be one JSON object: %v", err)
			return
		}
	}

	if req.DetectOnly && req.Source == "wallpaper" {
		wp := currentWallpaper(req)
		writeResponse(response{OK: true, Data: map[string]any{"wallpaper": wp, "wallpaperMtime": mtime(wp)}})
		return
	}

	var res result
	switch req.Source {
	case "preset":
		res = fromPreset(req)
	case "wallpaper":
		res = fromWallpaper(req)
	default:
		fail("bad_input", "source must be \"preset\" or \"wallpaper\"")
		return
	}
	writeResponse(response{OK: true, Data: res})
}
