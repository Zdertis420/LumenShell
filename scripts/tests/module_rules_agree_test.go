package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// The `modules` program (validating a manifest's default) and the `config` program
// (validating a user's value) each have their own implementation of a setting rule.
// They must give the same verdict for the same value.
func TestSettingRulesAgreeBetweenModulesAndConfig(t *testing.T) {
	type tc struct {
		rule   map[string]any // type, enum, min, max
		values []any
		valid  []bool
	}
	table := []tc{
		{map[string]any{"type": "string"}, []any{"x", "", 5.0, true}, []bool{true, true, false, false}},
		{map[string]any{"type": "string", "enum": []any{"a", "b"}}, []any{"a", "b", "c", "", 1.0}, []bool{true, true, false, false, false}},
		{map[string]any{"type": "integer", "min": 1, "max": 5}, []any{1.0, 3.0, 5.0, 0.0, 6.0, 2.5, "3", true}, []bool{true, true, true, false, false, false, false, false}},
		{map[string]any{"type": "integer"}, []any{7.0, -7.0, 7.5}, []bool{true, true, false}},
		{map[string]any{"type": "integer", "min": 10}, []any{10.0, 99.0, 9.0}, []bool{true, true, false}},
		{map[string]any{"type": "number", "min": 0, "max": 1}, []any{0.0, 0.5, 1.0, -0.1, 1.1, "1", false}, []bool{true, true, true, false, false, false, false}},
		{map[string]any{"type": "number"}, []any{-3.5, 0.0, 1e6, "x"}, []bool{true, true, true, false}},
		// (null is left out on purpose: since interface 1.2 a null default is valid, meaning
		// "unset", while a null user value is not. modules_null_default_test.go covers both.)
		{map[string]any{"type": "boolean"}, []any{true, false, "true", 1.0}, []bool{true, true, false, false}},
		// Interface 1.1: string length limits (counted in characters) ...
		{map[string]any{"type": "string", "minLength": 1, "maxLength": 3}, []any{"a", "abc", "ééé", "", "abcd", "éééé", 5.0},
			[]bool{true, true, true, false, false, false, false}},
		{map[string]any{"type": "string", "maxLength": 2}, []any{"", "ab", "abc"}, []bool{true, true, false}},
		// ... and the stringlist type. A value is valid only if the list and every entry are
		// valid: the modules program refuses a bad default, and the config helper reports each
		// dropped entry as an error, so both give the same verdict.
		{map[string]any{"type": "stringlist", "maxItems": 3},
			[]any{[]any{"a", "b"}, []any{}, []any{"a", "b", "c"}, []any{"a", "b", "c", "d"}, "a", 7.0, []any{"a", 7.0}, []any{""},
				[]any{strings.Repeat("x", 80)}, []any{strings.Repeat("x", 81)}},
			[]bool{true, true, true, false, false, false, false, false, true, false}},
		{map[string]any{"type": "stringlist"}, []any{[]any{"ok"}, []any{"éé"}}, []bool{true, true}},
	}

	user := t.TempDir()
	type check struct {
		id    string
		rule  map[string]any
		value any
		valid bool
		label string
	}
	var checks []check
	n := 0
	for ti, c := range table {
		if len(c.values) != len(c.valid) {
			t.Fatalf("table %d is malformed", ti)
		}
		for vi, v := range c.values {
			n++
			id := fmt.Sprintf("rule-%d", n)
			rule := map[string]any{}
			for k, x := range c.rule {
				rule[k] = x
			}
			rule["default"] = v
			manifest := map[string]any{
				"id": id, "name": "rule", "version": "1.0.0", "api": "1.0",
				"settings": map[string]any{"value": rule},
			}
			b, _ := json.Marshal(manifest)
			writeModule(t, user, id, string(b))
			checks = append(checks, check{id, c.rule, v, c.valid[vi], fmt.Sprintf("%v %v", c.rule, v)})
		}
	}

	scanned := byID(scanDirs(t, "", user))
	for _, c := range checks {
		// 1. The modules program accepts the value as a default exactly when it is valid.
		if got := scanned[c.id].Valid; got != c.valid {
			t.Errorf("scan: rule %v default %v: valid=%v, want %v (errors %+v)", c.rule, c.value, got, c.valid, scanned[c.id].Errors)
		}

		// 2. The config program accepts the value as a user override exactly when it is valid.
		rule := map[string]any{}
		for k, x := range c.rule {
			rule[k] = x
		}
		var okDefault any
		switch c.rule["type"] {
		case "string":
			okDefault = "a"
		case "integer":
			okDefault = 10.0
			if _, hasMax := c.rule["max"]; hasMax {
				okDefault = c.rule["min"]
			}
		case "number":
			okDefault = 0.0
		case "boolean":
			okDefault = false
		case "stringlist":
			okDefault = []any{}
		}
		if c.rule["type"] == "string" {
			if _, hasMin := c.rule["minLength"]; hasMin {
				okDefault = "a"
			} else if _, hasMax := c.rule["maxLength"]; hasMax {
				okDefault = ""
			}
		}
		rule["default"] = okDefault
		def := map[string]any{"id": c.id, "defaultEnabled": false, "settings": map[string]any{"value": rule}}
		file := writeFile(t, mustJSON(map[string]any{"modules": map[string]any{c.id: map[string]any{"value": c.value}}}))
		out := resolveWithModules(t, file, def)
		gotValid := len(out.Errors) == 0
		if gotValid != c.valid {
			t.Errorf("config: rule %v value %v: valid=%v, want %v (errors %+v)", c.rule, c.value, gotValid, c.valid, out.Errors)
		}
	}
}
