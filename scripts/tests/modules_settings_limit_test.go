package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Interface 1.3 (specs/005-prism-window/contracts/module-manifest-1.3.md): the limit of
// settings per module rises from 64 to 128. The manifest schema and the `modules` program
// must agree on the number.

func manyBooleanSettings(n int) string {
	var s []string
	for i := 0; i < n; i++ {
		s = append(s, fmt.Sprintf(`"s%c%c":{"type":"boolean","default":false}`, 'a'+rune(i/26), 'a'+rune(i%26)))
	}
	return strings.Join(s, ",")
}

func TestSettingsLimitIs128(t *testing.T) {
	user := t.TempDir()
	for _, n := range []int{64, 65, 128, 129} {
		id := fmt.Sprintf("n%d", n)
		writeModule(t, user, id, `{"id":"`+id+`","name":"m","version":"1.0.0","api":"1.3","settings":{`+manyBooleanSettings(n)+`}}`)
	}
	got := byID(scanWithAPI(t, "", user, "1.3"))
	for _, id := range []string{"n64", "n65", "n128"} {
		if !got[id].Valid {
			t.Errorf("%s must be valid: %+v", id, got[id].Errors)
		}
	}
	if m := got["n129"]; m.Valid || !hasProblem(m.Errors, "settings", "128") {
		t.Errorf("129 settings must be refused with the limit named: %+v", m)
	}
}

func TestInterface13IsRefusedByOlderRules(t *testing.T) {
	user := t.TempDir()
	writeModule(t, user, "new", `{"id":"new","name":"m","version":"1.0.0","api":"1.3"}`)
	if m := byID(scanWithAPI(t, "", user, "1.2"))["new"]; m.Valid {
		t.Errorf("a 1.3 module must be refused on 1.2: %+v", m)
	}
	if m := byID(scanWithAPI(t, "", user, "1.3"))["new"]; !m.Valid {
		t.Errorf("a 1.3 module must load on 1.3: %+v", m.Errors)
	}
}

// The manifest schema file and the program agree on the limit.
func TestSettingsLimitAgreesWithTheSchemaFile(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(dataDir(), "schema", "module.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			Settings struct {
				MaxProperties int `json:"maxProperties"`
			} `json:"settings"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	if s.Properties.Settings.MaxProperties != 128 {
		t.Errorf("schema maxProperties = %d, want 128", s.Properties.Settings.MaxProperties)
	}
}
