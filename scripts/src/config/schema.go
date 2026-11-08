package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Limits of the "stringlist" setting type (module interface 1.1, specs/003-bar).
const (
	defaultMaxListItems = 64
	maxListEntryChars   = 80
)

// rule describes one known config key. These rules are compiled in so a missing
// data directory cannot break validation. They must match
// data/schema/config.schema.json (a test checks this).
type rule struct {
	key       string
	kind      string   // "string", "int", "number", "boolean", or "stringlist"
	enum      []string // allowed strings; nil means any string
	min       float64
	max       float64
	hasMin    bool
	hasMax    bool
	minLen    int // "string" only: limits in characters
	maxLen    int
	hasMinLen bool
	hasMaxLen bool
	maxItems  int // "stringlist" only
	def       any
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
	{key: "helper.timeoutMs", kind: "int", min: 500, max: 30000, hasMin: true, hasMax: true, def: int64(3000)},
}

// moduleDef is a module as the caller describes it in the "modules" input: the rules for
// its settings plus whether it is enabled by default.
type moduleDef struct {
	id             string
	defaultEnabled bool
}

// ruleSet is the core rules plus the rules of the modules named in the input.
type ruleSet struct {
	rules   []rule
	modules []moduleDef
}

// newRuleSet builds the rules for one request. raw is the optional "modules" input.
// Malformed module entries are skipped: the caller (the Modules singleton) only passes
// definitions that the modules program already validated.
func newRuleSet(raw any) *ruleSet {
	s := &ruleSet{rules: append([]rule(nil), rules...)}
	list, _ := raw.([]any)
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			continue
		}
		enabled, _ := m["defaultEnabled"].(bool)
		s.modules = append(s.modules, moduleDef{id: id, defaultEnabled: enabled})
		s.rules = append(s.rules, rule{key: "modules." + id + ".enabled", kind: "boolean", def: enabled})
		settings, _ := m["settings"].(map[string]any)
		keys := make([]string, 0, len(settings))
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if r, ok := moduleRule("modules."+id+"."+k, settings[k]); ok {
				s.rules = append(s.rules, r)
			}
		}
	}
	return s
}

// moduleRule converts one setting rule from a module definition.
func moduleRule(key string, raw any) (rule, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return rule{}, false
	}
	typ, _ := m["type"].(string)
	r := rule{key: key, def: m["default"]}
	switch typ {
	case "string":
		r.kind = "string"
		if list, ok := m["enum"].([]any); ok {
			for _, e := range list {
				if s, ok := e.(string); ok {
					r.enum = append(r.enum, s)
				}
			}
		}
		if n, ok := m["minLength"].(json.Number); ok {
			if f, err := n.Int64(); err == nil {
				r.minLen, r.hasMinLen = int(f), true
			}
		}
		if n, ok := m["maxLength"].(json.Number); ok {
			if f, err := n.Int64(); err == nil {
				r.maxLen, r.hasMaxLen = int(f), true
			}
		}
	case "stringlist":
		r.kind = "stringlist"
		r.maxItems = defaultMaxListItems
		if n, ok := m["maxItems"].(json.Number); ok {
			if f, err := n.Int64(); err == nil && f >= 1 && f <= defaultMaxListItems {
				r.maxItems = int(f)
			}
		}
	case "integer":
		r.kind = "int"
	case "number":
		r.kind = "number"
	case "boolean":
		r.kind = "boolean"
	default:
		return rule{}, false
	}
	if r.kind == "int" || r.kind == "number" {
		if n, ok := m["min"].(json.Number); ok {
			if f, err := n.Float64(); err == nil {
				r.min, r.hasMin = f, true
			}
		}
		if n, ok := m["max"].(json.Number); ok {
			if f, err := n.Float64(); err == nil {
				r.max, r.hasMax = f, true
			}
		}
	}
	return r, true
}

func (s *ruleSet) find(key string) *rule {
	for i := range s.rules {
		if s.rules[i].key == key {
			return &s.rules[i]
		}
	}
	return nil
}

func (s *ruleSet) defaultsTree() map[string]any {
	t := map[string]any{}
	for _, r := range s.rules {
		setPath(t, r.key, r.def)
	}
	return t
}

func (s *ruleSet) hasModule(id string) bool {
	for _, m := range s.modules {
		if m.id == id {
			return true
		}
	}
	return false
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
		f, err := n.Float64()
		if !ok || err != nil || f != float64(int64(f)) || !r.inRange(f) {
			return "must be an integer" + r.rangeText()
		}
	case "number":
		n, ok := v.(json.Number)
		f, err := n.Float64()
		if !ok || err != nil || !r.inRange(f) {
			return "must be a number" + r.rangeText()
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return "must be true or false"
		}
	case "stringlist":
		// Only the list as a whole is checked here. Single bad entries are dropped on read
		// (filterList), so a write never fails because of them.
		list, ok := v.([]any)
		if !ok {
			return "must be a list of strings"
		}
		if len(list) > r.maxItems {
			return fmt.Sprintf("must have at most %d entries", r.maxItems)
		}
	default:
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if n := utf8.RuneCountInString(s); (r.hasMinLen && n < r.minLen) || (r.hasMaxLen && n > r.maxLen) {
			return "must be a string" + r.lengthText()
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

func (r *rule) lengthText() string {
	switch {
	case r.hasMinLen && r.hasMaxLen:
		return fmt.Sprintf(" of %d to %d characters", r.minLen, r.maxLen)
	case r.hasMinLen:
		return fmt.Sprintf(" of at least %d characters", r.minLen)
	case r.hasMaxLen:
		return fmt.Sprintf(" of at most %d characters", r.maxLen)
	}
	return ""
}

// droppedEntry is one entry of a stringlist value that was left out.
type droppedEntry struct {
	index   int
	problem string
	value   any
}

// filterList keeps the valid entries of a stringlist value (non-empty strings of at most
// 80 characters), in order, and reports the others. The result is never nil.
func (r *rule) filterList(v any) ([]any, []droppedEntry) {
	list, _ := v.([]any)
	kept := make([]any, 0, len(list))
	var dropped []droppedEntry
	for i, item := range list {
		s, ok := item.(string)
		switch {
		case !ok:
			dropped = append(dropped, droppedEntry{i, "is not a string; entry dropped", cutValue(item)})
		case s == "":
			dropped = append(dropped, droppedEntry{i, "is empty; entry dropped", s})
		case utf8.RuneCountInString(s) > maxListEntryChars:
			dropped = append(dropped, droppedEntry{i, fmt.Sprintf("is longer than %d characters; entry dropped", maxListEntryChars), cutValue(s)})
		default:
			kept = append(kept, s)
		}
	}
	return kept, dropped
}

// cutValue shortens a reported value to 80 characters. Non-strings are reported as written.
func cutValue(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	if utf8.RuneCountInString(s) <= maxListEntryChars {
		return s
	}
	return string([]rune(s)[:maxListEntryChars])
}

func (r *rule) inRange(f float64) bool {
	return (!r.hasMin || f >= r.min) && (!r.hasMax || f <= r.max)
}

func (r *rule) rangeText() string {
	switch {
	case r.hasMin && r.hasMax:
		return fmt.Sprintf(" between %g and %g", r.min, r.max)
	case r.hasMin:
		return fmt.Sprintf(" of at least %g", r.min)
	case r.hasMax:
		return fmt.Sprintf(" of at most %g", r.max)
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
