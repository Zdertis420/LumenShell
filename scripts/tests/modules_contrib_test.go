package tests

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// Black-box tests of `modules scan` for module interface 1.1: extension points,
// contributions, commands, and the permission form contribute:<module-id>.
// Contract: specs/003-bar/contracts/module-manifest-1.1.md and helper-bar.md.

type contribEntry struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Target struct {
		Module string `json:"module"`
		Point  string `json:"point"`
	} `json:"target"`
	File  string `json:"file"`
	Key   string `json:"key"`
	Short string `json:"short"`
}

type pointEntry struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type contribModule struct {
	scanModule
	ExtensionPoints []pointEntry   `json:"extensionPoints"`
	Contributions   []contribEntry `json:"contributions"`
	Commands        []string       `json:"commands"`
}

func scanContrib(t *testing.T, api string, fixtures ...string) map[string]contribModule {
	t.Helper()
	e, r := runModules(t, "scan", map[string]any{"bundledDir": stage(t, fixtures...), "userDir": "", "apiVersion": api})
	if !e.OK {
		t.Fatalf("scan failed: %+v\nstderr=%s", e.Error, r.stderr)
	}
	var out struct {
		Modules []contribModule `json:"modules"`
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	m := map[string]contribModule{}
	for _, x := range out.Modules {
		m[x.ID] = x
	}
	return m
}

func TestScanHostPointsAndCommands(t *testing.T) {
	m := scanContrib(t, "1.1", "host-points")["host-points"]
	if !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("host-points must be valid: %+v", m.Errors)
	}
	if len(m.ExtensionPoints) != 2 || m.ExtensionPoints[0].ID != "items" || m.ExtensionPoints[1].ID != "tools" {
		t.Errorf("extensionPoints = %+v", m.ExtensionPoints)
	}
	if strings.Join(m.Commands, ",") != "focus,reset" {
		t.Errorf("commands = %v", m.Commands)
	}
	if len(m.Warnings) != 0 {
		t.Errorf("1.1 fields must not warn: %+v", m.Warnings)
	}
}

func TestScanContributionsAndShortNames(t *testing.T) {
	mods := scanContrib(t, "1.1", "contrib-good", "contrib-two-hosts", "contrib-target-missing", "contrib-self")

	g := mods["contrib-good"]
	if !g.Valid || len(g.Errors) != 0 {
		t.Fatalf("contrib-good must be valid: %+v", g.Errors)
	}
	if len(g.Contributions) != 3 {
		t.Fatalf("contributions = %+v", g.Contributions)
	}
	for _, c := range g.Contributions {
		if !filepath.IsAbs(c.File) || filepath.Base(c.File) != "Item.qml" || !strings.HasPrefix(c.File, g.Dir) {
			t.Errorf("file must be absolute and inside the module: %s", c.File)
		}
		if c.Key != "contrib-good/"+c.ID {
			t.Errorf("key = %q for %s", c.Key, c.ID)
		}
	}
	// The short form is the first-declared contribution per target point: one (items) and
	// three (tools) have it, two does not.
	short := map[string]string{}
	for _, c := range g.Contributions {
		short[c.ID] = c.Short
	}
	if short["one"] != "contrib-good" || short["two"] != "" || short["three"] != "contrib-good" {
		t.Errorf("short names = %v", short)
	}

	// One short form per target host.
	two := mods["contrib-two-hosts"]
	if len(two.Contributions) != 2 || two.Contributions[0].Short != "contrib-two-hosts" || two.Contributions[1].Short != "contrib-two-hosts" {
		t.Errorf("contrib-two-hosts: %+v", two.Contributions)
	}

	// A target that is not installed is not a manifest error.
	if x := mods["contrib-target-missing"]; !x.Valid || len(x.Errors) != 0 {
		t.Errorf("contrib-target-missing must be valid: %+v", x.Errors)
	}
	// A module may contribute to its own point without a permission.
	if x := mods["contrib-self"]; !x.Valid || len(x.Errors) != 0 || len(x.Contributions) != 1 || len(x.ExtensionPoints) != 1 {
		t.Errorf("contrib-self: %+v", x)
	}
}

func TestScanInterfaceVersions(t *testing.T) {
	// api10-contrib declares api 1.0 and uses 1.1 fields: valid on both, and the fields take effect.
	for _, lumen := range []string{"1.0", "1.1"} {
		m := scanContrib(t, lumen, "api10-contrib")["api10-contrib"]
		if !m.Valid || len(m.Errors) != 0 {
			t.Errorf("api10-contrib on %s: %+v", lumen, m.Errors)
		}
		if len(m.Contributions) != 1 || len(m.ExtensionPoints) != 1 {
			t.Errorf("api10-contrib on %s: fields must take effect: %+v", lumen, m)
		}
		if len(m.Warnings) != 0 {
			t.Errorf("api10-contrib on %s: %+v", lumen, m.Warnings)
		}
	}
	// host-points needs 1.1: refused by a 1.0 shell with the required and available versions.
	m := scanContrib(t, "1.0", "host-points")["host-points"]
	if m.Valid || !hasProblem(m.Errors, "api", "requires interface 1.1, Lumen provides 1.0") {
		t.Errorf("host-points on 1.0: %+v", m.Errors)
	}
	if m := scanContrib(t, "1.1", "host-points")["host-points"]; !m.Valid {
		t.Errorf("host-points on 1.1: %+v", m.Errors)
	}
}

func TestScanOnlyRequiresStaysReserved(t *testing.T) {
	m := scanContrib(t, "1.1", "warn-requires")["warn-requires"]
	if !m.Valid {
		t.Fatalf("must be valid: %+v", m.Errors)
	}
	if !hasProblem(m.Warnings, "requires", "requires is not supported in module interface 1.1") {
		t.Errorf("missing requires warning: %+v", m.Warnings)
	}
	if len(m.Warnings) != 2 { // requires + the unknown field
		t.Errorf("warnings = %+v", m.Warnings)
	}
}

func TestScanInvalidContributionFixtures(t *testing.T) {
	cases := []struct {
		fixture, field, contains string
	}{
		{"err-contrib-no-permission", "contributions[0].target.module", "contribute:host-points"},
		{"err-contrib-dup-id", "contributions[1].id", "dup"},
		{"err-contrib-file-missing", "contributions[0].file", "Nope.qml"},
		{"err-contrib-file-escape", "contributions[0].file", "inside the module directory"},
		{"err-contrib-bad-target", "contributions[0].target.module", ""},
		{"err-contrib-slash", "contributions[0].id", ""},
		{"err-point-dup", "extensionPoints[1].id", "items"},
		{"err-point-bad-id", "extensionPoints[0].id", ""},
		{"err-commands-bad", "commands[0]", ""},
		{"err-commands-dup", "commands[1]", "go"},
		{"err-stringlist-default", "settings.names.default", ""},
		{"err-stringlist-entry", "settings.names.default", ""},
		{"err-stringlist-too-many", "settings.names.default", ""},
		{"err-stringlist-enum", "settings.names.enum", ""},
		{"err-string-length", "settings.word.default", ""},
		{"err-string-length-type", "settings.size.minLength", ""},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			m := scanContrib(t, "1.1", c.fixture)[c.fixture]
			if m.Valid {
				t.Fatalf("%s must be invalid", c.fixture)
			}
			if !hasProblem(m.Errors, c.field, c.contains) {
				t.Errorf("want an error on %s containing %q, got %+v", c.field, c.contains, m.Errors)
			}
			// An invalid module registers nothing.
			if len(m.Contributions) != 0 || len(m.ExtensionPoints) != 0 {
				t.Errorf("an invalid module must not register points or contributions: %+v", m)
			}
		})
	}
}

func TestScanStringLengthFixtureIsValid(t *testing.T) {
	m := scanContrib(t, "1.1", "good-string-length")["good-string-length"]
	if !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("good-string-length must be valid: %+v", m.Errors)
	}
}

func TestScanContributePermissionForm(t *testing.T) {
	user := t.TempDir()
	good := func(perm string) string {
		return `{"id":"perm","name":"p","version":"1.0.0","api":"1.1","permissions":[` + perm + `]}`
	}
	cases := []struct {
		perm  string
		valid bool
	}{
		{`"contribute:bar"`, true},
		{`"contribute:host-points"`, true},
		{`"contribute:Bad Id"`, false},
		{`"contribute:"`, false},
		{`"contribute:a"`, false}, // module ids have at least 2 characters
		{`"contribute"`, false},
		{`"contribute:bar","contribute:bar"`, false}, // duplicate
	}
	for _, c := range cases {
		dir := filepath.Join(user, "case")
		writeModule(t, user, "case", good(c.perm))
		mods := scanWithAPI(t, "", user, "1.1")
		if len(mods) != 1 || mods[0].Valid != c.valid {
			t.Errorf("%s: valid=%v want %v (%+v)", c.perm, mods[0].Valid, c.valid, mods[0].Errors)
		}
		_ = dir
	}
}

func TestScanContribLimits(t *testing.T) {
	user := t.TempDir()
	mk := func(n int, one string) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = strings.ReplaceAll(one, "N", string(rune('a'+i%26))+string(rune('a'+i/26)))
		}
		return strings.Join(parts, ",")
	}
	points := `{"id":"pN","description":"d"}`
	contribs := `{"id":"cN","name":"c","target":{"module":"host-points","point":"items"},"file":"Item.qml"}`
	cmds := `"cmdN"`
	cases := []struct {
		name, manifest string
		field          string
	}{
		{"9 points", `{"id":"lim-a","name":"l","version":"1.0.0","api":"1.1","extensionPoints":[` + mk(9, points) + `]}`, "extensionPoints"},
		{"17 contributions", `{"id":"lim-b","name":"l","version":"1.0.0","api":"1.1","permissions":["contribute:host-points"],"contributions":[` + mk(17, contribs) + `]}`, "contributions"},
		{"17 commands", `{"id":"lim-c","name":"l","version":"1.0.0","api":"1.1","commands":[` + mk(17, cmds) + `]}`, "commands"},
	}
	for _, c := range cases {
		writeModule(t, user, strings.ReplaceAll(c.name, " ", "-"), c.manifest)
	}
	for _, m := range scanWithAPI(t, "", user, "1.1") {
		if m.Valid {
			t.Errorf("%s must be invalid", m.ID)
		}
	}
	// Exactly at the limits is fine.
	ok := t.TempDir()
	writeModule(t, ok, "at-limits", `{"id":"at-limits","name":"l","version":"1.0.0","api":"1.1","extensionPoints":[`+mk(8, points)+`],"commands":[`+mk(16, cmds)+`]}`)
	if m := scanWithAPI(t, "", ok, "1.1")[0]; !m.Valid {
		t.Errorf("8 points and 16 commands must be valid: %+v", m.Errors)
	}
}
