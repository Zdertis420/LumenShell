package main

import (
	"os"
	"path/filepath"
	"sort"
)

// settingSchema maps one manifest setting rule to JSON Schema keywords.
func settingSchema(rule map[string]any) map[string]any {
	out := map[string]any{}
	if typ, ok := rule["type"].(string); ok {
		out["type"] = typ
		if typ == "stringlist" {
			// A stringlist is a JSON array of non-empty strings (module interface 1.1).
			out["type"] = "array"
			out["items"] = map[string]any{"type": "string", "minLength": 1, "maxLength": maxListEntry}
			maxItems := any(maxListItems)
			if v, ok := rule["maxItems"]; ok {
				maxItems = v
			}
			out["maxItems"] = maxItems
		}
	}
	for _, k := range []string{"default", "enum", "description", "minLength", "maxLength"} {
		if v, ok := rule[k]; ok {
			out[k] = v
		}
	}
	if v, ok := rule["min"]; ok {
		out["minimum"] = v
	}
	if v, ok := rule["max"]; ok {
		out["maximum"] = v
	}
	return out
}

// runSchema prints the core configuration schema extended with the settings of every
// valid module, for editors. It never writes anything.
func runSchema(in map[string]any) {
	bundledDir, _ := in["bundledDir"].(string)
	userDir, _ := in["userDir"].(string)
	dataDir, _ := in["dataDir"].(string)
	apiStr, _ := in["apiVersion"].(string)
	lumen, ok := parseAPI(apiStr)
	if !ok {
		fail("bad_input", "apiVersion must look like 1.0")
		return
	}

	path := filepath.Join(dataDir, "schema", "config.schema.json")
	b, err := os.ReadFile(path)
	if err != nil {
		fail("no_schema", "cannot read the core schema %s: %v", path, err)
		return
	}
	v, err := decodeJSON(b)
	schema, isObj := v.(map[string]any)
	if err != nil || !isObj {
		fail("no_schema", "the core schema %s is not a JSON object", path)
		return
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
		schema["properties"] = props
	}

	moduleProps := map[string]any{}
	for _, m := range scanAll(bundledDir, userDir, lumen) {
		if !m.Valid {
			continue
		}
		fields := map[string]any{
			"enabled": map[string]any{
				"type":        "boolean",
				"default":     m.DefaultEnabled,
				"description": "Load this module (bundled modules default to on, user modules to off)",
			},
		}
		settings, _ := m.Manifest["settings"].(map[string]any)
		keys := make([]string, 0, len(settings))
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if rule, ok := settings[k].(map[string]any); ok {
				fields[k] = settingSchema(rule)
			}
		}
		desc, _ := m.Manifest["name"].(string)
		moduleProps[m.ID] = map[string]any{
			"type":                 "object",
			"description":          desc,
			"additionalProperties": true,
			"properties":           fields,
		}
	}
	props["modules"] = map[string]any{
		"type":                 "object",
		"description":          "Per-module settings, keyed by module id",
		"additionalProperties": true,
		"properties":           moduleProps,
	}
	succeed(map[string]any{"schema": schema})
}
