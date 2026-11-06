package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Limits from specs/002-modules/contracts/module-manifest.md.
const (
	maxManifestBytes = 64 * 1024
	maxSettings      = 64
	maxPermissions   = 16
	maxHelpers       = 8
)

var (
	idPattern      = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	apiPattern     = regexp.MustCompile(`^([0-9]+)\.([0-9]+)$`)
	settingKey     = regexp.MustCompile(`^[a-z][A-Za-z0-9]{0,31}$`)
	helperName     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
)

// permissions is the documented list. Keep in sync with data/schema/module.schema.json
// and specs/002-modules/data-model.md (a test checks this).
var permissions = []string{"helpers", "config.read", "config.write", "log.read", "modules.read-settings", "network"}

var reservedFields = []string{"extensionPoints", "contributions", "requires"}

var knownFields = map[string]bool{
	"id": true, "name": true, "version": true, "api": true, "description": true,
	"main": true, "permissions": true, "settings": true, "helpers": true,
	"extensionPoints": true, "contributions": true, "requires": true,
}

// problem is one finding about a manifest, tied to a field.
type problem struct {
	Field   string `json:"field"`
	Problem string `json:"problem"`
}

// apiVersion is a module interface version.
type apiVersion struct{ major, minor int }

func (v apiVersion) String() string { return fmt.Sprintf("%d.%d", v.major, v.minor) }

func parseAPI(s string) (apiVersion, bool) {
	m := apiPattern.FindStringSubmatch(s)
	if m == nil {
		return apiVersion{}, false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return apiVersion{major, minor}, true
}

// helperDecl is a helper program declared in a manifest.
type helperDecl struct {
	Name string
	Path string
}

// manifest holds what the scan needs from a validated manifest.
type manifest struct {
	raw     map[string]any
	id      string // set only when the id field is usable
	main    string
	helpers []helperDecl
}

// parseManifest validates the manifest bytes against the rules. lumen is the interface
// version Lumen provides. It returns errors and warnings; the manifest is valid when
// there are no errors.
func parseManifest(b []byte, lumen apiVersion) (*manifest, []problem, []problem) {
	var errs, warns []problem
	addErr := func(field, format string, a ...any) {
		errs = append(errs, problem{field, fmt.Sprintf(format, a...)})
	}

	if len(b) > maxManifestBytes {
		addErr("", "manifest is larger than 64 KiB")
		return &manifest{raw: map[string]any{}}, errs, warns
	}
	v, err := decodeJSON(b)
	if err != nil {
		addErr("", "module.json is not valid JSON: %v", err)
		return &manifest{raw: map[string]any{}}, errs, warns
	}
	obj, ok := v.(map[string]any)
	if !ok {
		addErr("", "module.json must contain a JSON object")
		return &manifest{raw: map[string]any{}}, errs, warns
	}
	m := &manifest{raw: obj, main: "Main.qml"}

	// Required strings.
	str := func(field string, required bool) (string, bool) {
		v, present := obj[field]
		if !present {
			if required {
				addErr(field, "is required")
			}
			return "", false
		}
		s, ok := v.(string)
		if !ok {
			addErr(field, "must be a string")
			return "", false
		}
		return s, true
	}

	if id, ok := str("id", true); ok {
		if idPattern.MatchString(id) {
			m.id = id
		} else {
			addErr("id", "must start with a lower-case letter and use only lower-case letters, digits, and '-' (2 to 32 characters)")
		}
	}
	if name, ok := str("name", true); ok {
		if n := utf8.RuneCountInString(name); n < 1 || n > 64 {
			addErr("name", "must be 1 to 64 characters")
		}
	}
	if version, ok := str("version", true); ok && !versionPattern.MatchString(version) {
		addErr("version", "must look like 1.0.0")
	}
	if api, ok := str("api", true); ok {
		want, valid := parseAPI(api)
		switch {
		case !valid:
			addErr("api", "must look like 1.0 (interface major.minor)")
		case want.major != lumen.major || want.minor > lumen.minor:
			addErr("api", "requires interface %s, Lumen provides %s", want, lumen)
		}
	}
	if d, ok := str("description", false); ok && utf8.RuneCountInString(d) > 256 {
		addErr("description", "must be at most 256 characters")
	}
	if main, ok := str("main", false); ok {
		if !strings.HasSuffix(main, ".qml") || !filepath.IsLocal(main) {
			addErr("main", "must be a relative path inside the module directory that ends in .qml")
		} else {
			m.main = filepath.Clean(main)
		}
	}

	// Permissions.
	if raw, present := obj["permissions"]; present {
		list, ok := raw.([]any)
		if !ok {
			addErr("permissions", "must be a list of strings")
		} else {
			if len(list) > maxPermissions {
				addErr("permissions", "at most %d permissions are allowed", maxPermissions)
			}
			seen := map[string]bool{}
			for _, item := range list {
				s, ok := item.(string)
				switch {
				case !ok:
					addErr("permissions", "entries must be strings")
				case !contains(permissions, s):
					addErr("permissions", "unknown permission %q (allowed: %s)", s, strings.Join(permissions, ", "))
				case seen[s]:
					addErr("permissions", "duplicate permission %q", s)
				}
				if ok {
					seen[s] = true
				}
			}
		}
	}

	// Settings.
	if raw, present := obj["settings"]; present {
		settings, ok := raw.(map[string]any)
		if !ok {
			addErr("settings", "must be an object")
		} else {
			if len(settings) > maxSettings {
				addErr("settings", "at most %d settings are allowed", maxSettings)
			}
			keys := make([]string, 0, len(settings))
			for k := range settings {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				checkSetting(k, settings[k], addErr)
			}
		}
	}

	// Helpers.
	if raw, present := obj["helpers"]; present {
		list, ok := raw.([]any)
		if !ok {
			addErr("helpers", "must be a list of {name, path}")
		} else {
			if len(list) > maxHelpers {
				addErr("helpers", "at most %d helpers are allowed", maxHelpers)
			}
			names := map[string]bool{}
			for i, item := range list {
				h, ok := item.(map[string]any)
				field := fmt.Sprintf("helpers[%d]", i)
				if !ok {
					addErr(field, "must be an object with name and path")
					continue
				}
				name, _ := h["name"].(string)
				path, _ := h["path"].(string)
				switch {
				case !helperName.MatchString(name):
					addErr(field+".name", "must start with a lower-case letter and use only lower-case letters, digits, and '-' (1 to 32 characters)")
				case names[name]:
					addErr(field+".name", "duplicate helper name %q", name)
				}
				names[name] = true
				if path == "" || !filepath.IsLocal(path) {
					addErr(field+".path", "must be a relative path inside the module directory (no absolute paths, no ..)")
					continue
				}
				m.helpers = append(m.helpers, helperDecl{Name: name, Path: filepath.Clean(path)})
			}
		}
	}

	// Reserved and unknown fields are warnings.
	for _, f := range reservedFields {
		if _, present := obj[f]; present {
			warns = append(warns, problem{f, fmt.Sprintf("%s is not supported in module interface %s", f, lumen)})
		}
	}
	var unknown []string
	for k := range obj {
		if !knownFields[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		warns = append(warns, problem{k, "unknown field (ignored)"})
	}
	return m, errs, warns
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// checkSetting validates one setting rule from a manifest, including that its default
// satisfies the rule.
func checkSetting(key string, raw any, addErr func(field, format string, a ...any)) {
	field := "settings." + key
	if !settingKey.MatchString(key) {
		addErr(field, "setting names must start with a lower-case letter and use letters and digits only (1 to 32 characters)")
		return
	}
	if key == "enabled" {
		addErr(field, "the setting name 'enabled' is reserved")
		return
	}
	rule, ok := raw.(map[string]any)
	if !ok {
		addErr(field, "must be an object with type and default")
		return
	}
	typ, _ := rule["type"].(string)
	if typ != "string" && typ != "integer" && typ != "number" && typ != "boolean" {
		addErr(field+".type", "must be string, integer, number, or boolean")
		return
	}
	def, hasDef := rule["default"]
	if !hasDef {
		addErr(field+".default", "is required")
		return
	}
	if d, ok := rule["description"]; ok {
		if s, isStr := d.(string); !isStr || utf8.RuneCountInString(s) > 160 {
			addErr(field+".description", "must be a string of at most 160 characters")
		}
	}

	var enum []string
	if e, present := rule["enum"]; present {
		list, isList := e.([]any)
		if typ != "string" || !isList {
			addErr(field+".enum", "is only allowed for string settings and must be a list of strings")
		} else {
			for _, item := range list {
				if s, isStr := item.(string); isStr {
					enum = append(enum, s)
				} else {
					addErr(field+".enum", "entries must be strings")
				}
			}
		}
	}
	var min, max *float64
	for _, bound := range []struct {
		name string
		dst  **float64
	}{{"min", &min}, {"max", &max}} {
		v, present := rule[bound.name]
		if !present {
			continue
		}
		n, isNum := v.(json.Number)
		f, err := n.Float64()
		if (typ != "integer" && typ != "number") || !isNum || err != nil {
			addErr(field+"."+bound.name, "is only allowed for integer and number settings and must be a number")
			continue
		}
		*bound.dst = &f
	}
	if min != nil && max != nil && *min > *max {
		addErr(field, "min must not be greater than max")
	}
	if p := checkSettingValue(typ, enum, min, max, def); p != "" {
		addErr(field+".default", "%s", p)
	}
}

// checkSettingValue returns "" when v satisfies the rule, otherwise a short problem text.
func checkSettingValue(typ string, enum []string, min, max *float64, v any) string {
	inRange := func(f float64) bool {
		return (min == nil || f >= *min) && (max == nil || f <= *max)
	}
	rangeText := func() string {
		switch {
		case min != nil && max != nil:
			return fmt.Sprintf(" between %g and %g", *min, *max)
		case min != nil:
			return fmt.Sprintf(" of at least %g", *min)
		case max != nil:
			return fmt.Sprintf(" of at most %g", *max)
		}
		return ""
	}
	switch typ {
	case "integer":
		n, ok := v.(json.Number)
		f, err := n.Float64()
		if !ok || err != nil || f != float64(int64(f)) || !inRange(f) {
			return "default must be an integer" + rangeText()
		}
	case "number":
		n, ok := v.(json.Number)
		f, err := n.Float64()
		if !ok || err != nil || !inRange(f) {
			return "default must be a number" + rangeText()
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return "default must be true or false"
		}
	case "string":
		s, ok := v.(string)
		if !ok {
			return "default must be a string"
		}
		if enum != nil && !contains(enum, s) {
			return "default must be one of " + strings.Join(enum, ", ")
		}
	}
	return ""
}
