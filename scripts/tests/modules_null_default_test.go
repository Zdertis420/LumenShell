package tests

import (
	"encoding/json"
	"strings"
	"testing"
)

// Interface 1.2 (specs/004-bar-customization-styles/contracts/module-manifest-1.2.md):
// a setting rule may have `"default": null`, meaning "unset". The module falls back to a
// design token. The rule is implemented in both Go programs and is covered here.

func nullDefaultModule() map[string]any {
	return map[string]any{
		"id":             "nd",
		"defaultEnabled": true,
		"settings": map[string]any{
			"size":   map[string]any{"type": "integer", "default": nil, "min": 1, "max": 96},
			"ratio":  map[string]any{"type": "number", "default": nil, "min": 0, "max": 1},
			"family": map[string]any{"type": "string", "default": nil, "maxLength": 32},
			"role":   map[string]any{"type": "string", "default": nil, "enum": []any{"primary", "surface"}},
			"italic": map[string]any{"type": "boolean", "default": nil},
			"fixed":  map[string]any{"type": "integer", "default": 5},
		},
	}
}

func resolveNullDefaults(t *testing.T, user string) listResolved {
	t.Helper()
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": writeUser(t, user), "dataDir": dataDir(), "modules": []any{nullDefaultModule()},
	}), nil, "resolve")
	e := parseEnvelope(t, r)
	if !e.OK {
		t.Fatalf("resolve failed: %+v", e.Error)
	}
	var out listResolved
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNullDefaultManifestIsValidForScalarTypes(t *testing.T) {
	m := byID(scanWithAPI(t, "", stage(t, "null-default-good"), "1.2"))["null-default-good"]
	if !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("null defaults must be valid: %+v", m.Errors)
	}
}

func TestNullDefaultIsRefusedForAStringlist(t *testing.T) {
	m := byID(scanWithAPI(t, "", stage(t, "err-null-default-stringlist"), "1.2"))["err-null-default-stringlist"]
	if m.Valid || !hasProblem(m.Errors, "settings.names.default", "null is not allowed") {
		t.Fatalf("a stringlist with a null default must be invalid: valid=%v %+v", m.Valid, m.Errors)
	}
}

func TestNullDefaultStillChecksTheRuleItself(t *testing.T) {
	// min greater than max is an error whatever the default is.
	dir := t.TempDir()
	writeModule(t, dir, "nd-bad", `{"id":"nd-bad","name":"x","version":"1.0.0","api":"1.2",
		"settings":{"s":{"type":"string","default":null,"minLength":5,"maxLength":2}}}`)
	m := byID(scanWithAPI(t, "", dir, "1.2"))["nd-bad"]
	if m.Valid || !hasProblem(m.Errors, "settings.s", "minLength must not be greater than maxLength") {
		t.Fatalf("rule checks must still run: valid=%v %+v", m.Valid, m.Errors)
	}
}

func TestMergedSchemaHasNoDefaultForUnsetKeys(t *testing.T) {
	e, r := runModules(t, "schema", map[string]any{
		"bundledDir": "", "userDir": stage(t, "null-default-good"), "apiVersion": "1.2", "dataDir": dataDir(),
	})
	if !e.OK {
		t.Fatalf("schema failed: %+v\nstderr=%s", e.Error, r.stderr)
	}
	var out struct {
		Schema map[string]any `json:"schema"`
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	props := out.Schema["properties"].(map[string]any)["modules"].(map[string]any)["properties"].(map[string]any)
	mod := props["null-default-good"].(map[string]any)["properties"].(map[string]any)
	for _, k := range []string{"size", "ratio", "family", "role", "italic"} {
		prop, ok := mod[k].(map[string]any)
		if !ok {
			t.Fatalf("%s missing from the merged schema", k)
		}
		if _, has := prop["default"]; has {
			t.Errorf("%s must have no default in the schema: %v", k, prop)
		}
		if prop["type"] == nil {
			t.Errorf("%s must keep its type", k)
		}
	}
	if mod["fixed"].(map[string]any)["default"] == nil {
		t.Errorf("a key with a real default keeps it")
	}
}

func TestConfigReturnsNullForUnsetKeysAndTheValueWhenSet(t *testing.T) {
	res := resolveNullDefaults(t, `{}`)
	nd := modulesOf(res.Values, "nd")
	for _, k := range []string{"size", "ratio", "family", "role", "italic"} {
		v, present := nd[k]
		if !present || v != nil {
			t.Errorf("%s must be present and null when unset, got %#v (present=%v)", k, v, present)
		}
	}
	if nd["fixed"] == nil {
		t.Errorf("fixed must keep its default")
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", res.Errors)
	}

	res = resolveNullDefaults(t, `{"modules":{"nd":{"size":14,"role":"primary","italic":true}}}`)
	nd = modulesOf(res.Values, "nd")
	if nd["size"] != float64(14) || nd["role"] != "primary" || nd["italic"] != true {
		t.Fatalf("user values must win: %v", nd)
	}
	if nd["ratio"] != nil {
		t.Errorf("ratio must stay unset: %v", nd["ratio"])
	}
	if res.Sources["modules.nd.size"] != "user" || res.Sources["modules.nd.ratio"] != "default" {
		t.Errorf("sources = %v", res.Sources)
	}
}

func TestConfigRejectsANullUserValueAndKeepsTheKeyUnset(t *testing.T) {
	for key, want := range map[string]string{"size": "integer", "ratio": "number", "family": "string", "italic": "true or false"} {
		res := resolveNullDefaults(t, `{"modules":{"nd":{"`+key+`":null}}}`)
		e := errorFor(res, "modules.nd."+key)
		if e == nil || !strings.Contains(e.Problem, want) {
			t.Errorf("%s: a null user value must be an error mentioning %q, got %+v", key, want, e)
		}
		if v := modulesOf(res.Values, "nd")[key]; v != nil {
			t.Errorf("%s must stay unset after the error, got %#v", key, v)
		}
	}
}

func TestConfigChecksUserValuesAgainstTheRuleOfAnUnsetKey(t *testing.T) {
	cases := map[string]string{
		"size":   `{"modules":{"nd":{"size":0}}}`,
		"ratio":  `{"modules":{"nd":{"ratio":1.5}}}`,
		"family": `{"modules":{"nd":{"family":"` + strings.Repeat("x", 40) + `"}}}`,
		"role":   `{"modules":{"nd":{"role":"sparkles"}}}`,
	}
	for key, user := range cases {
		res := resolveNullDefaults(t, user)
		if errorFor(res, "modules.nd."+key) == nil {
			t.Errorf("%s: a bad value must be an error: %s", key, user)
		}
		if v := modulesOf(res.Values, "nd")[key]; v != nil {
			t.Errorf("%s must fall back to unset, got %#v", key, v)
		}
	}
}

func TestInterface12ModuleIsRefusedByAShellProvidingOnlyOneOne(t *testing.T) {
	dir := stage(t, "api12-item")
	m := byID(scanWithAPI(t, "", dir, "1.1"))["api12-item"]
	if m.Valid || !hasProblem(m.Errors, "api", "1.2") || !hasProblem(m.Errors, "api", "1.1") {
		t.Fatalf("must be refused with both versions: valid=%v %+v", m.Valid, m.Errors)
	}
	m = byID(scanWithAPI(t, "", dir, "1.2"))["api12-item"]
	if !m.Valid {
		t.Fatalf("must load on 1.2: %+v", m.Errors)
	}
}
