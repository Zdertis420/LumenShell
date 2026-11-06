package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// checkSchema is a tiny validator for the JSON Schema keywords that
// data/schema/module.schema.json actually uses. It returns one message per violation.
func checkSchema(schema map[string]any, v any, path string) []string {
	var errs []string
	add := func(format string, a ...any) { errs = append(errs, path+": "+fmt.Sprintf(format, a...)) }

	if typ, ok := schema["type"].(string); ok {
		okType := false
		switch typ {
		case "object":
			_, okType = v.(map[string]any)
		case "array":
			_, okType = v.([]any)
		case "string":
			_, okType = v.(string)
		case "number", "integer":
			_, okType = v.(float64)
		case "boolean":
			_, okType = v.(bool)
		}
		if !okType {
			add("must be %s", typ)
			return errs
		}
	}
	if enum, ok := schema["enum"].([]any); ok {
		found := false
		for _, e := range enum {
			if e == v {
				found = true
			}
		}
		if !found {
			add("%v is not in the enum", v)
		}
	}
	if s, ok := v.(string); ok {
		if p, ok := schema["pattern"].(string); ok && !regexp.MustCompile(p).MatchString(s) {
			add("%q does not match %s", s, p)
		}
		if n, ok := schema["minLength"].(float64); ok && float64(len([]rune(s))) < n {
			add("shorter than %v", n)
		}
		if n, ok := schema["maxLength"].(float64); ok && float64(len([]rune(s))) > n {
			add("longer than %v", n)
		}
	}
	if f, ok := v.(float64); ok {
		if n, ok := schema["minimum"].(float64); ok && f < n {
			add("below %v", n)
		}
		if n, ok := schema["maximum"].(float64); ok && f > n {
			add("above %v", n)
		}
	}
	if arr, ok := v.([]any); ok {
		if n, ok := schema["maxItems"].(float64); ok && float64(len(arr)) > n {
			add("more than %v items", n)
		}
		if schema["uniqueItems"] == true {
			seen := map[string]bool{}
			for _, x := range arr {
				k := fmt.Sprint(x)
				if seen[k] {
					add("duplicate item %s", k)
				}
				seen[k] = true
			}
		}
		if items, ok := schema["items"].(map[string]any); ok {
			for i, x := range arr {
				errs = append(errs, checkSchema(items, x, fmt.Sprintf("%s[%d]", path, i))...)
			}
		}
	}
	if obj, ok := v.(map[string]any); ok {
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				if _, present := obj[r.(string)]; !present {
					add("%s is required", r)
				}
			}
		}
		if n, ok := schema["maxProperties"].(float64); ok && float64(len(obj)) > n {
			add("more than %v properties", n)
		}
		props, _ := schema["properties"].(map[string]any)
		names, _ := schema["propertyNames"].(map[string]any)
		extra, _ := schema["additionalProperties"].(map[string]any)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if names != nil {
				if p, ok := names["pattern"].(string); ok && !regexp.MustCompile(p).MatchString(k) {
					add("property name %q does not match %s", k, p)
				}
			}
			if sub, ok := props[k].(map[string]any); ok {
				errs = append(errs, checkSchema(sub, obj[k], path+"."+k)...)
			} else if extra != nil {
				errs = append(errs, checkSchema(extra, obj[k], path+"."+k)...)
			}
		}
	}
	return errs
}

func loadModuleSchema(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "data", "schema", "module.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("module.schema.json is not valid JSON: %v", err)
	}
	return s
}

func readManifest(t *testing.T, fixture string) any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixturesDir(), fixture, "module.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("%s: %v", fixture, err)
	}
	return v
}

// Manifests the program accepts also satisfy the published schema.
func TestValidManifestsSatisfyTheSchema(t *testing.T) {
	schema := loadModuleSchema(t)
	for _, name := range []string{"good-min", "good-full", "warn-reserved", "dup-a", "dup-b"} {
		if errs := checkSchema(schema, readManifest(t, name), name); len(errs) != 0 {
			t.Errorf("%s should satisfy the schema: %v", name, errs)
		}
	}
	// Everything the program calls valid (staged from all fixtures) satisfies the schema.
	all := stage(t, "good-min", "good-full", "warn-reserved", "api-probe", "api-denied", "helper-fails", "helper-hangs", "qml-good-window")
	for _, m := range scanDirs(t, "", all) {
		if !m.Valid {
			t.Errorf("%s should be valid: %+v", m.ID, m.Errors)
			continue
		}
		if errs := checkSchema(schema, readManifest(t, m.DirName), m.ID); len(errs) != 0 {
			t.Errorf("%s: %v", m.ID, errs)
		}
	}
}

// Manifests the schema can express as wrong are wrong in the schema, too.
func TestSchemaRejectsWhatTheProgramRejects(t *testing.T) {
	schema := loadModuleSchema(t)
	for _, name := range []string{"err-bad-id", "err-missing-field", "err-bad-api", "err-bad-permission", "err-dup-permission", "err-bad-setting-default"} {
		errs := checkSchema(schema, readManifest(t, name), name)
		if name == "err-bad-setting-default" {
			// The schema cannot tie a default to its type; the program checks that.
			continue
		}
		if len(errs) == 0 {
			t.Errorf("%s should violate the schema", name)
		}
	}
}

// The permission names in the schema, the data model, and the program agree.
func TestPermissionListsAgree(t *testing.T) {
	schema := loadModuleSchema(t)
	perms := schema["properties"].(map[string]any)["permissions"].(map[string]any)["items"].(map[string]any)["enum"].([]any)
	var fromSchema []string
	for _, p := range perms {
		fromSchema = append(fromSchema, p.(string))
	}
	sort.Strings(fromSchema)

	b, err := os.ReadFile(filepath.Join(repoRoot, "specs", "002-modules", "data-model.md"))
	if err != nil {
		t.Skip("data model not present")
	}
	re := regexp.MustCompile("(?m)^\\| `([a-z.-]+)` \\| (Run|Read|Set|The module intends)")
	var fromDoc []string
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		fromDoc = append(fromDoc, m[1])
	}
	sort.Strings(fromDoc)
	if strings.Join(fromSchema, ",") != strings.Join(fromDoc, ",") {
		t.Fatalf("schema permissions %v, data model permissions %v", fromSchema, fromDoc)
	}

	// Each permission in the schema is accepted by the program.
	user := t.TempDir()
	for i, p := range fromSchema {
		id := fmt.Sprintf("perm-%d", i)
		writeModule(t, user, id, fmt.Sprintf(`{"id":"%s","name":"p","version":"1.0.0","api":"1.0","permissions":["%s"]}`, id, p))
	}
	for _, m := range scanDirs(t, "", user) {
		if !m.Valid {
			t.Errorf("%s: %+v", m.ID, m.Errors)
		}
	}
}
