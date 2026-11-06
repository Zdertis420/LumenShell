package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// rule describes one known config key. These rules are compiled in so a missing
// data directory cannot break validation. They must match
// data/schema/config.schema.json (a test checks this).
type rule struct {
	key  string
	kind string   // "string" or "int"
	enum []string // allowed strings; nil means any string
	min  int64
	max  int64
	def  any
}

var schemes = []string{
	"scheme-content", "scheme-expressive", "scheme-fidelity", "scheme-fruit-salad",
	"scheme-monochrome", "scheme-neutral", "scheme-rainbow", "scheme-tonal-spot",
	"scheme-vibrant", "scheme-smart",
}

// builtinPresets is used only when the presets directory cannot be read.
var builtinPresets = []string{
	"tokyo-night", "catppuccin-latte", "catppuccin-frappe", "catppuccin-macchiato", "catppuccin-mocha",
}

var rules = []rule{
	{key: "theme.source", kind: "string", enum: []string{"wallpaper", "preset"}, def: "preset"},
	{key: "theme.preset", kind: "string", def: "catppuccin-mocha"}, // enum comes from the presets directory
	{key: "theme.scheme", kind: "string", enum: schemes, def: "scheme-tonal-spot"},
	{key: "theme.mode", kind: "string", enum: []string{"light", "dark"}, def: "dark"},
	{key: "theme.wallpaper", kind: "string", def: ""},
	{key: "log.level", kind: "string", enum: []string{"debug", "info", "warn", "error"}, def: "info"},
	{key: "helper.timeoutMs", kind: "int", min: 500, max: 30000, def: int64(3000)},
}

func findRule(key string) *rule {
	for i := range rules {
		if rules[i].key == key {
			return &rules[i]
		}
	}
	return nil
}

// presetIDs lists the preset ids found in dataDir/presets.
func presetIDs(dataDir string) []string {
	entries, err := os.ReadDir(filepath.Join(dataDir, "presets"))
	if err != nil {
		return builtinPresets
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasSuffix(name, ".json") {
			ids = append(ids, strings.TrimSuffix(name, ".json"))
		}
	}
	if len(ids) == 0 {
		return builtinPresets
	}
	sort.Strings(ids)
	return ids
}

func (r *rule) allowed(dataDir string) []string {
	if r.key == "theme.preset" {
		return presetIDs(dataDir)
	}
	return r.enum
}

// check returns "" when v is valid for the rule, otherwise a short problem text.
func (r *rule) check(v any, dataDir string) string {
	switch r.kind {
	case "int":
		n, ok := v.(json.Number)
		if !ok {
			return fmt.Sprintf("must be an integer between %d and %d", r.min, r.max)
		}
		f, err := n.Float64()
		if err != nil || f != float64(int64(f)) || int64(f) < r.min || int64(f) > r.max {
			return fmt.Sprintf("must be an integer between %d and %d", r.min, r.max)
		}
	default:
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if allowed := r.allowed(dataDir); allowed != nil {
			for _, a := range allowed {
				if a == s {
					return ""
				}
			}
			return "must be one of " + strings.Join(allowed, ", ")
		}
	}
	return ""
}

// ---- tree helpers (dot paths over map[string]any) ----

func splitKey(key string) []string { return strings.Split(key, ".") }

func getPath(tree map[string]any, key string) (any, bool) {
	var cur any = tree
	for _, part := range splitKey(key) {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// setPath sets key in tree, creating (or replacing non-object) parents.
func setPath(tree map[string]any, key string, v any) {
	parts := splitKey(key)
	cur := tree
	for _, part := range parts[:len(parts)-1] {
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[part] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = v
}

// deletePath removes key and then any parent objects that became empty.
// It reports whether the key existed.
func deletePath(tree map[string]any, key string) bool {
	parts := splitKey(key)
	chain := []map[string]any{tree}
	cur := tree
	for _, part := range parts[:len(parts)-1] {
		next, ok := cur[part].(map[string]any)
		if !ok {
			return false
		}
		chain = append(chain, next)
		cur = next
	}
	last := parts[len(parts)-1]
	if _, ok := cur[last]; !ok {
		return false
	}
	delete(cur, last)
	for i := len(parts) - 2; i >= 0; i-- {
		if len(chain[i+1]) > 0 {
			break
		}
		delete(chain[i], parts[i])
	}
	return true
}

func defaultsTree() map[string]any {
	t := map[string]any{}
	for _, r := range rules {
		setPath(t, r.key, r.def)
	}
	return t
}
