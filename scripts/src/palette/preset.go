package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// fromPreset loads a bundled preset and makes sure it meets the contrast minimum.
func fromPreset(req request) result {
	if !idPattern.MatchString(req.Preset) {
		return degrade(req, "invalid preset id "+quote(req.Preset))
	}
	b, err := os.ReadFile(filepath.Join(req.DataDir, "presets", req.Preset+".json"))
	if err != nil {
		return degrade(req, "preset "+req.Preset+" not found: "+err.Error())
	}
	var p palette
	if err := json.Unmarshal(b, &p); err != nil {
		return degrade(req, "preset "+req.Preset+" is not valid JSON: "+err.Error())
	}
	if err := validate(&p); err != nil {
		return degrade(req, "preset "+req.Preset+" rejected: "+err.Error())
	}
	p.ID = req.Preset
	adjusted := enforceContrast(&p)
	if adjusted == nil {
		adjusted = []string{}
	} else {
		logf("info", "adjusted text colors to meet contrast", map[string]any{"preset": req.Preset, "roles": adjusted})
	}
	return result{Palette: p, Status: "ok", Adjusted: adjusted}
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
