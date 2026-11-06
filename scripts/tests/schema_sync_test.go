package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type schemaProp struct {
	Type       string                `json:"type"`
	Enum       []string              `json:"enum"`
	Minimum    *float64              `json:"minimum"`
	Maximum    *float64              `json:"maximum"`
	Default    any                   `json:"default"`
	Properties map[string]schemaProp `json:"properties"`
}

func loadSchema(t *testing.T) schemaProp {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "data", "schema", "config.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s schemaProp
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	return s
}

func flattenSchema(prefix string, p schemaProp, out map[string]schemaProp) {
	for name, sub := range p.Properties {
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		if sub.Type == "object" {
			flattenSchema(key, sub, out)
		} else {
			out[key] = sub
		}
	}
}

// The compiled defaults and rules in the config helper match the published schema.
func TestCompiledDefaultsMatchSchema(t *testing.T) {
	leaves := map[string]schemaProp{}
	flattenSchema("", loadSchema(t), leaves)
	if len(leaves) == 0 {
		t.Fatal("schema has no keys")
	}

	out := resolveConfig(t, filepath.Join(t.TempDir(), "none.json"), "")
	if len(out.Errors) != 0 {
		t.Fatalf("errors: %+v", out.Errors)
	}
	// The default preset (catppuccin-mocha) is dark, the same as the default mode,
	// so the resolved values equal the pure defaults.
	defaults := out.Values
	for key, prop := range leaves {
		parts := strings.SplitN(key, ".", 2)
		got := defaults[parts[0]].(map[string]any)[parts[1]]
		want := prop.Default
		if f, ok := got.(float64); ok {
			got = f
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: helper default %v (%T), schema default %v (%T)", key, got, got, want, want)
		}
	}
	// Every helper key is in the schema.
	for group, v := range defaults {
		for k := range v.(map[string]any) {
			if _, ok := leaves[group+"."+k]; !ok {
				t.Errorf("helper key %s.%s is missing from the schema", group, k)
			}
		}
	}
}

// Every enum value in the schema is accepted by the helper, and a value outside is rejected.
func TestSchemaEnumsAreEnforced(t *testing.T) {
	leaves := map[string]schemaProp{}
	flattenSchema("", loadSchema(t), leaves)
	for key, prop := range leaves {
		p := filepath.Join(t.TempDir(), "c.json")
		for _, v := range prop.Enum {
			if e := runSet(t, p, change(key, v)); !e.OK {
				t.Errorf("%s=%q should be valid: %+v", key, v, e.Error)
			}
		}
		if len(prop.Enum) > 0 {
			if e := runSet(t, p, change(key, "definitely-not-allowed")); e.OK {
				t.Errorf("%s accepted a value outside its enum", key)
			}
		}
		if prop.Minimum != nil {
			if e := runSet(t, p, change(key, *prop.Minimum-1)); e.OK {
				t.Errorf("%s accepted a value below its minimum", key)
			}
			if e := runSet(t, p, change(key, *prop.Maximum+1)); e.OK {
				t.Errorf("%s accepted a value above its maximum", key)
			}
			if e := runSet(t, p, change(key, *prop.Minimum)); !e.OK {
				t.Errorf("%s rejected its minimum", key)
			}
		}
	}
}

// The preset ids in the schema are exactly the files in data/presets.
func TestPresetListMatchesFiles(t *testing.T) {
	leaves := map[string]schemaProp{}
	flattenSchema("", loadSchema(t), leaves)
	files, _ := filepath.Glob(filepath.Join(repoRoot, "data", "presets", "*.json"))
	var ids []string
	for _, f := range files {
		ids = append(ids, strings.TrimSuffix(filepath.Base(f), ".json"))
	}
	if !reflect.DeepEqual(sorted(ids), sorted(leaves["theme.preset"].Enum)) {
		t.Fatalf("schema presets %v, files %v", leaves["theme.preset"].Enum, ids)
	}
}

// Every key in the contract table (specs/.../config-keys.md) is in the schema.
func TestContractTableMatchesSchema(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "specs", "001-foundation", "contracts", "config-keys.md"))
	if err != nil {
		t.Skip("contract file not present")
	}
	leaves := map[string]schemaProp{}
	flattenSchema("", loadSchema(t), leaves)
	re := regexp.MustCompile("(?m)^\\| `([a-z]+\\.[A-Za-z]+)` \\|")
	found := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		found[m[1]] = true
		if _, ok := leaves[m[1]]; !ok {
			t.Errorf("contract key %s is missing from the schema", m[1])
		}
	}
	for key := range leaves {
		if !found[key] {
			t.Errorf("schema key %s is missing from the contract table", key)
		}
	}
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
