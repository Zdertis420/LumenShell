package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type configError struct {
	Key      string `json:"key"`
	Problem  string `json:"problem"`
	Fallback any    `json:"fallback"`
}

// readUserFile loads the user's override file.
// A missing file is not an error. Any other problem returns a message and an empty tree.
func readUserFile(path string) (tree map[string]any, problem string) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, ""
	}
	if err != nil {
		return map[string]any{}, "cannot read " + path + ": " + err.Error()
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return map[string]any{}, "file is empty: " + path
	}
	v, err := decodeJSON(b)
	if err != nil {
		return map[string]any{}, "invalid JSON in " + path + ": " + err.Error()
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}, "top level of " + path + " must be a JSON object"
	}
	return obj, ""
}

// loadPresetMode reads the "mode" of a bundled preset ("" when unavailable).
func loadPresetMode(dataDir, id string) string {
	b, err := os.ReadFile(filepath.Join(dataDir, "presets", id+".json"))
	if err != nil {
		return ""
	}
	var p struct {
		Mode string `json:"mode"`
	}
	if json.Unmarshal(b, &p) != nil {
		return ""
	}
	return p.Mode
}

func runResolve(in map[string]any) {
	userFile, _ := in["userFile"].(string)
	dataDir, _ := in["dataDir"].(string)
	themeOverride, _ := in["themeOverride"].(string)
	if userFile == "" {
		fail("bad_input", "userFile is required")
		return
	}

	rs := newRuleSet(in["modules"])
	errs := []configError{}
	user, problem := readUserFile(userFile)
	if problem != "" {
		errs = append(errs, configError{Key: "", Problem: problem, Fallback: "defaults"})
		logf("warn", problem, nil)
	}

	// Validate the user layer key by key. Invalid keys are dropped and reported,
	// so they fall back to the layer below.
	valid := map[string]any{}
	for _, r := range rs.rules {
		v, present := getPath(user, r.key)
		if !present {
			// A non-object parent hides the key; report the parent once.
			continue
		}
		if p := r.check(v, dataDir); p != "" {
			errs = append(errs, configError{Key: r.key, Problem: p, Fallback: r.def})
			logf("warn", r.key+": "+p, map[string]any{"key": r.key})
			continue
		}
		setPath(valid, r.key, v)
	}
	parents := []string{"theme", "log", "helper"}
	if len(rs.modules) > 0 {
		parents = append(parents, "modules")
	}
	for _, parent := range parents {
		if v, present := user[parent]; present {
			if _, isObj := v.(map[string]any); !isObj {
				errs = append(errs, configError{Key: parent, Problem: "must be an object", Fallback: "defaults"})
				logf("warn", parent+": must be an object", map[string]any{"key": parent})
			}
		}
	}
	// A known module's entry must be an object, too.
	for _, m := range rs.modules {
		if v, present := getPath(user, "modules."+m.id); present {
			if _, isObj := v.(map[string]any); !isObj {
				key := "modules." + m.id
				errs = append(errs, configError{Key: key, Problem: "must be an object", Fallback: "defaults"})
				logf("warn", key+": must be an object", map[string]any{"key": key})
			}
		}
	}

	sources := map[string]string{}
	overrides := map[string]string{} // key -> "user" or "temp"
	for _, r := range rs.rules {
		sources[r.key] = "default"
		if _, ok := getPath(valid, r.key); ok {
			overrides[r.key] = "user"
		}
	}

	// A temporary theme replaces the user's theme source and preset for this resolve only.
	if themeOverride != "" {
		if themeOverride == "wallpaper" {
			setPath(valid, "theme.source", "wallpaper")
		} else {
			if p := rs.find("theme.preset").check(themeOverride, dataDir); p != "" {
				fail("invalid_theme", "unknown theme %q: %s", themeOverride, p)
				return
			}
			setPath(valid, "theme.source", "preset")
			setPath(valid, "theme.preset", themeOverride)
			overrides["theme.preset"] = "temp"
		}
		overrides["theme.source"] = "temp"
	}

	values := rs.defaultsTree()

	// Layer 2: the selected preset supplies its mode.
	source := getString(values, valid, "theme.source")
	if source == "preset" {
		if mode := loadPresetMode(dataDir, getString(values, valid, "theme.preset")); mode != "" {
			setPath(values, "theme.mode", mode)
			sources["theme.mode"] = "preset"
		}
	}

	// Layer 3: user (and temp) overrides.
	for key, label := range overrides {
		v, _ := getPath(valid, key)
		setPath(values, key, v)
		sources[key] = label
	}

	succeed(map[string]any{"values": values, "errors": errs, "sources": sources})
}

// getString returns the effective string for key: the valid override, else the default.
func getString(defaults, overrides map[string]any, key string) string {
	if v, ok := getPath(overrides, key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	v, _ := getPath(defaults, key)
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
