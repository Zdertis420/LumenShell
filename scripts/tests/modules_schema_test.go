package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func runSchemaCmd(t *testing.T, bundled, user string) map[string]any {
	t.Helper()
	e, r := runModules(t, "schema", map[string]any{
		"bundledDir": bundled, "userDir": user, "apiVersion": "1.0", "dataDir": dataDir(),
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
	return out.Schema
}

func TestMergedSchemaKeepsCoreAndAddsValidModules(t *testing.T) {
	user := stage(t, "good-full", "good-min", "err-bad-api", "err-bad-json")
	schema := runSchemaCmd(t, "", user)

	// The core schema is unchanged.
	b, _ := os.ReadFile(filepath.Join(dataDir(), "schema", "config.schema.json"))
	var core map[string]any
	_ = json.Unmarshal(b, &core)
	props := schema["properties"].(map[string]any)
	for _, key := range []string{"theme", "log", "helper"} {
		if !reflect.DeepEqual(props[key], core["properties"].(map[string]any)[key]) {
			t.Errorf("core property %s changed", key)
		}
	}

	// Valid modules are added under properties.modules.properties.<id>.
	mods := props["modules"].(map[string]any)["properties"].(map[string]any)
	if _, ok := mods["good-full"]; !ok {
		t.Fatalf("good-full missing: %v", mods)
	}
	if _, ok := mods["good-min"]; !ok {
		t.Errorf("good-min missing: %v", mods)
	}
	for _, id := range []string{"err-bad-api", "dir:err-bad-json"} {
		if _, ok := mods[id]; ok {
			t.Errorf("invalid module %s must not contribute", id)
		}
	}

	full := mods["good-full"].(map[string]any)["properties"].(map[string]any)
	if full["enabled"].(map[string]any)["type"] != "boolean" || full["enabled"].(map[string]any)["default"] != false {
		t.Errorf("enabled = %v", full["enabled"])
	}
	mode := full["mode"].(map[string]any)
	if mode["type"] != "string" || !reflect.DeepEqual(mode["enum"], []any{"one", "two"}) || mode["default"] != "one" || mode["description"] != "A choice" {
		t.Errorf("mode = %v", mode)
	}
	count := full["count"].(map[string]any)
	if count["type"] != "integer" || count["minimum"].(float64) != 1 || count["maximum"].(float64) != 5 || count["default"].(float64) != 3 {
		t.Errorf("count = %v", count)
	}
	if ratio := full["ratio"].(map[string]any); ratio["type"] != "number" || ratio["maximum"].(float64) != 1 {
		t.Errorf("ratio = %v", ratio)
	}
	if flag := full["flag"].(map[string]any); flag["type"] != "boolean" || flag["default"] != false {
		t.Errorf("flag = %v", flag)
	}
}

func TestMergedSchemaChecksRealConfigs(t *testing.T) {
	schema := runSchemaCmd(t, stage(t, "good-min"), stage(t, "good-full"))
	cases := []struct {
		name   string
		config string
		valid  bool
	}{
		{"empty", `{}`, true},
		{"core key", `{"log":{"level":"debug"}}`, true},
		{"module setting", `{"modules":{"good-full":{"count":4,"mode":"two","flag":true}}}`, true},
		{"module enabled", `{"modules":{"good-min":{"enabled":false}}}`, true},
		{"unknown module", `{"modules":{"gone":{"x":1}}}`, true},
		{"count too big", `{"modules":{"good-full":{"count":9}}}`, false},
		{"bad enum", `{"modules":{"good-full":{"mode":"three"}}}`, false},
		{"bad enabled", `{"modules":{"good-min":{"enabled":"yes"}}}`, false},
		{"bad core key", `{"log":{"level":"loud"}}`, false},
	}
	for _, c := range cases {
		var v any
		if err := json.Unmarshal([]byte(c.config), &v); err != nil {
			t.Fatal(err)
		}
		errs := checkSchema(schema, v, c.name)
		if (len(errs) == 0) != c.valid {
			t.Errorf("%s: valid=%v, schema errors %v", c.name, c.valid, errs)
		}
	}
}

func TestSchemaWithoutModulesIsTheCoreSchema(t *testing.T) {
	schema := runSchemaCmd(t, "", "")
	mods := schema["properties"].(map[string]any)["modules"].(map[string]any)["properties"].(map[string]any)
	if len(mods) != 0 {
		t.Errorf("no modules expected: %v", mods)
	}
}

func TestSchemaNeedsTheDataDir(t *testing.T) {
	e, _ := runModules(t, "schema", map[string]any{"bundledDir": "", "userDir": "", "apiVersion": "1.0", "dataDir": filepath.Join(t.TempDir(), "nope")})
	if e.OK || e.Error.Code != "no_schema" {
		t.Fatalf("response = %+v", e)
	}
}
