package tests

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type scanProblem struct {
	Field   string `json:"field"`
	Problem string `json:"problem"`
}

type scanModule struct {
	ID             string         `json:"id"`
	Dir            string         `json:"dir"`
	DirName        string         `json:"dirName"`
	Origin         string         `json:"origin"`
	Valid          bool           `json:"valid"`
	DefaultEnabled bool           `json:"defaultEnabled"`
	Main           string         `json:"main"`
	Manifest       map[string]any `json:"manifest"`
	Errors         []scanProblem  `json:"errors"`
	Warnings       []scanProblem  `json:"warnings"`
}

func fixturesDir() string { return filepath.Join(repoRoot, "scripts", "tests", "fixtures", "modules") }

// copyTree copies a directory, keeping file modes.
func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		dst := filepath.Join(to, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, b, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

// stage copies the named fixture directories into a fresh directory and returns it.
func stage(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		copyTree(t, filepath.Join(fixturesDir(), n), filepath.Join(dir, n))
	}
	return dir
}

func scanDirs(t *testing.T, bundled, user string) []scanModule {
	t.Helper()
	return scanWithAPI(t, bundled, user, "1.0")
}

func scanWithAPI(t *testing.T, bundled, user, api string) []scanModule {
	t.Helper()
	e, r := runModules(t, "scan", map[string]any{"bundledDir": bundled, "userDir": user, "apiVersion": api})
	if !e.OK {
		t.Fatalf("scan failed: %+v\nstderr=%s", e.Error, r.stderr)
	}
	var out struct {
		APIVersion string       `json:"apiVersion"`
		Modules    []scanModule `json:"modules"`
	}
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	if out.APIVersion != api {
		t.Fatalf("apiVersion = %q", out.APIVersion)
	}
	return out.Modules
}

func byID(mods []scanModule) map[string]scanModule {
	m := map[string]scanModule{}
	for _, x := range mods {
		m[x.ID] = x
	}
	return m
}

func hasProblem(list []scanProblem, fieldPrefix, contains string) bool {
	for _, p := range list {
		if strings.HasPrefix(p.Field, fieldPrefix) && strings.Contains(p.Problem, contains) {
			return true
		}
	}
	return false
}

func TestScanValidModulesAndDefaults(t *testing.T) {
	bundled := stage(t, "good-min", "good-full")
	for _, m := range scanDirs(t, bundled, "") {
		if !m.Valid || len(m.Errors) != 0 || !m.DefaultEnabled || m.Origin != "bundled" {
			t.Errorf("bundled %s: %+v", m.ID, m)
		}
		if !strings.HasSuffix(m.Main, "/Main.qml") || !filepath.IsAbs(m.Main) {
			t.Errorf("%s: main = %q", m.ID, m.Main)
		}
	}
	for _, m := range scanDirs(t, "", stage(t, "good-min", "good-full")) {
		if !m.Valid || m.DefaultEnabled || m.Origin != "user" {
			t.Errorf("user %s: %+v", m.ID, m)
		}
	}
	full := byID(scanDirs(t, stage(t, "good-full"), ""))["good-full"]
	if full.Manifest["description"] != "Uses every manifest field." || full.Manifest["version"] != "1.2.3" {
		t.Errorf("manifest not returned: %v", full.Manifest)
	}
}

func TestScanErrorsNameTheRightField(t *testing.T) {
	cases := []struct {
		dir      string
		id       string
		field    string
		contains string
	}{
		{"err-no-manifest", "dir:err-no-manifest", "", "module.json"},
		{"err-bad-json", "dir:err-bad-json", "", "JSON"},
		{"err-bad-id", "dir:err-bad-id", "id", ""},
		{"err-missing-field", "err-missing-field", "version", "required"},
		{"err-bad-api", "err-bad-api", "api", ""},
		{"err-newer-minor", "err-newer-minor", "api", "1.9"},
		{"err-other-major", "err-other-major", "api", "2.0"},
		{"err-missing-main", "err-missing-main", "main", "Nope.qml"},
		{"err-main-escape", "err-main-escape", "main", ""},
		{"err-bad-permission", "err-bad-permission", "permissions", "root"},
		{"err-dup-permission", "err-dup-permission", "permissions", "helpers"},
		{"err-bad-setting-default", "err-bad-setting-default", "settings.n", "default"},
		{"err-reserved-setting", "err-reserved-setting", "settings.enabled", "reserved"},
		{"err-helper-escape", "err-helper-escape", "helpers[0].path", ""},
		{"err-helper-notexec", "err-helper-notexec", "helpers[0].path", "executable"},
		{"err-helper-absolute", "err-helper-absolute", "helpers[0].path", ""},
	}
	for _, c := range cases {
		t.Run(c.dir, func(t *testing.T) {
			mods := scanDirs(t, "", stage(t, c.dir))
			if len(mods) != 1 {
				t.Fatalf("expected one module, got %d", len(mods))
			}
			m := mods[0]
			if m.Valid || m.ID != c.id {
				t.Fatalf("valid=%v id=%q want invalid %q", m.Valid, m.ID, c.id)
			}
			if !hasProblem(m.Errors, c.field, c.contains) {
				t.Fatalf("errors %+v lack field %q containing %q", m.Errors, c.field, c.contains)
			}
			for _, p := range m.Errors {
				if p.Problem == "" {
					t.Errorf("empty problem text: %+v", p)
				}
			}
		})
	}
}

func TestScanRequiredAndAvailableVersionsAreNamed(t *testing.T) {
	m := scanDirs(t, "", stage(t, "err-newer-minor"))[0]
	if !hasProblem(m.Errors, "api", "requires interface 1.9, Lumen provides 1.0") {
		t.Fatalf("errors = %+v", m.Errors)
	}
}

func TestScanInterfaceCompatibilityRule(t *testing.T) {
	user := t.TempDir()
	for _, api := range []string{"1.0", "1.2", "1.3", "0.9", "2.0"} {
		dir := filepath.Join(user, "m-"+strings.ReplaceAll(api, ".", "-"))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(`{"id":"m-%s","name":"m","version":"1.0.0","api":"%s"}`, strings.ReplaceAll(api, ".", "-"), api)
		_ = os.WriteFile(filepath.Join(dir, "module.json"), []byte(manifest), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("import QtQuick\nItem {}\n"), 0o644)
	}
	got := byID(scanWithAPI(t, "", user, "1.2"))
	want := map[string]bool{"m-1-0": true, "m-1-2": true, "m-1-3": false, "m-0-9": false, "m-2-0": false}
	for id, valid := range want {
		if got[id].Valid != valid {
			t.Errorf("%s: valid=%v, want %v (errors %+v)", id, got[id].Valid, valid, got[id].Errors)
		}
	}
}

func TestScanReservedAndUnknownFieldsAreWarnings(t *testing.T) {
	m := scanDirs(t, "", stage(t, "warn-reserved"))[0]
	if !m.Valid || len(m.Errors) != 0 {
		t.Fatalf("must be valid: %+v", m)
	}
	// From interface 1.1 on, extensionPoints and contributions take effect (see
	// modules_contrib_test.go); only `requires` stays reserved.
	if !hasProblem(m.Warnings, "requires", "requires is not supported in module interface 1.0") {
		t.Errorf("missing reserved warning for requires: %+v", m.Warnings)
	}
	if !hasProblem(m.Warnings, "somethingFromTheFuture", "unknown field") {
		t.Errorf("missing unknown-field warning: %+v", m.Warnings)
	}
	if len(m.Warnings) != 2 {
		t.Errorf("warnings = %+v", m.Warnings)
	}
	if _, ok := m.Manifest["requires"]; !ok {
		t.Errorf("reserved fields must stay in the returned manifest: %v", m.Manifest)
	}
}

func TestScanOddDirectoryNameIsSanitized(t *testing.T) {
	mods := scanDirs(t, "", stage(t, "bad name!"))
	if len(mods) != 1 || mods[0].ID != "dir:bad_name_" || mods[0].Valid {
		t.Fatalf("mods = %+v", mods)
	}
	if mods[0].DirName != "bad name!" || !hasProblem(mods[0].Errors, "", "bad_name_") {
		t.Fatalf("errors = %+v", mods[0].Errors)
	}
}

func TestScanDuplicateIDs(t *testing.T) {
	// Same directory: the first by directory name wins.
	mods := byID(scanDirs(t, "", stage(t, "dup-a", "dup-b")))
	var valid, invalid []scanModule
	all := scanDirs(t, "", stage(t, "dup-a", "dup-b"))
	for _, m := range all {
		if m.Valid {
			valid = append(valid, m)
		} else {
			invalid = append(invalid, m)
		}
	}
	if len(valid) != 1 || valid[0].DirName != "dup-a" || len(invalid) != 1 || invalid[0].DirName != "dup-b" {
		t.Fatalf("valid %+v invalid %+v (%v)", valid, invalid, mods)
	}
	if !hasProblem(invalid[0].Errors, "id", "duplicate") || !hasProblem(invalid[0].Errors, "id", "dup-a") || !hasProblem(invalid[0].Errors, "id", "dup-b") {
		t.Fatalf("the error must name both directories: %+v", invalid[0].Errors)
	}

	// Bundled wins over user, whatever the directory names.
	bundled, user := stage(t, "dup-b"), stage(t, "dup-a")
	for _, m := range scanDirs(t, bundled, user) {
		if m.Origin == "bundled" && !m.Valid {
			t.Errorf("the bundled module must win: %+v", m)
		}
		if m.Origin == "user" && m.Valid {
			t.Errorf("the user module must lose: %+v", m)
		}
	}
}

func TestScanOrderIsBundledThenUserSortedByDirectoryName(t *testing.T) {
	bundled := stage(t, "good-min", "warn-reserved")
	user := stage(t, "good-full", "err-bad-json", "err-bad-api")
	var got []string
	for _, m := range scanDirs(t, bundled, user) {
		got = append(got, m.Origin+":"+m.DirName)
	}
	want := []string{"bundled:good-min", "bundled:warn-reserved", "user:err-bad-api", "user:err-bad-json", "user:good-full"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order %v, want %v", got, want)
	}
}

func TestScanMissingUserDirIsNotAnErrorAndNothingIsCreated(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does", "not", "exist")
	if mods := scanDirs(t, stage(t, "good-min"), missing); len(mods) != 1 {
		t.Fatalf("mods = %+v", mods)
	}
	if _, err := os.Stat(filepath.Join(missing, "..", "..")); !os.IsNotExist(err) {
		t.Fatalf("the user directory (or its parents) must not be created: %v", err)
	}
	if mods := scanDirs(t, stage(t, "good-min"), ""); len(mods) != 1 {
		t.Fatalf("an empty userDir must work: %+v", mods)
	}
	if mods := scanDirs(t, "", t.TempDir()); len(mods) != 0 {
		t.Fatalf("an empty directory has no modules: %+v", mods)
	}
}

func TestScanHiddenEntriesAndFilesAreSkipped(t *testing.T) {
	user := stage(t, "good-min")
	_ = os.MkdirAll(filepath.Join(user, ".hidden"), 0o755)
	_ = os.WriteFile(filepath.Join(user, ".hidden", "module.json"), []byte("{}"), 0o644)
	_ = os.WriteFile(filepath.Join(user, "notes.txt"), []byte("x"), 0o644)
	mods := scanDirs(t, "", user)
	if len(mods) != 1 || mods[0].ID != "good-min" {
		t.Fatalf("mods = %+v", mods)
	}
}

func writeModule(t *testing.T, root, name, manifest string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "module.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "Main.qml"), []byte("import QtQuick\nItem {}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "bin", "h"), []byte("#!/bin/sh\n"), 0o755)
}

func TestScanLimits(t *testing.T) {
	user := t.TempDir()

	huge := `{"id":"huge","name":"huge","version":"1.0.0","api":"1.0","description":"` + strings.Repeat("x", 70*1024) + `"}`
	writeModule(t, user, "huge", huge)

	var settings []string
	for i := 0; i < 65; i++ {
		settings = append(settings, fmt.Sprintf(`"s%c%c":{"type":"boolean","default":false}`, 'a'+rune(i/26), 'a'+rune(i%26)))
	}
	writeModule(t, user, "many-settings", `{"id":"many-settings","name":"m","version":"1.0.0","api":"1.0","settings":{`+strings.Join(settings, ",")+`}}`)

	perms := strings.Repeat(`"helpers",`, 16) + `"helpers"`
	writeModule(t, user, "many-permissions", `{"id":"many-permissions","name":"m","version":"1.0.0","api":"1.0","permissions":[`+perms+`]}`)

	var helpers []string
	for i := 1; i <= 9; i++ {
		helpers = append(helpers, fmt.Sprintf(`{"name":"h%d","path":"bin/h"}`, i))
	}
	writeModule(t, user, "many-helpers", `{"id":"many-helpers","name":"m","version":"1.0.0","api":"1.0","helpers":[`+strings.Join(helpers, ",")+`]}`)

	got := byID(scanDirs(t, "", user))
	if m := got["dir:huge"]; m.Valid || !hasProblem(m.Errors, "", "64 KiB") {
		t.Errorf("huge: %+v", m)
	}
	if m := got["many-settings"]; m.Valid || !hasProblem(m.Errors, "settings", "64") {
		t.Errorf("many-settings: %+v", m)
	}
	if m := got["many-permissions"]; m.Valid || !hasProblem(m.Errors, "permissions", "16") {
		t.Errorf("many-permissions: %+v", m)
	}
	if m := got["many-helpers"]; m.Valid || !hasProblem(m.Errors, "helpers", "8") {
		t.Errorf("many-helpers: %+v", m)
	}
}

func TestScanTooManyDirectories(t *testing.T) {
	user := t.TempDir()
	for i := 0; i < 260; i++ {
		writeModule(t, user, fmt.Sprintf("m%03d", i), fmt.Sprintf(`{"id":"m%03d","name":"m","version":"1.0.0","api":"1.0"}`, i))
	}
	mods := scanDirs(t, "", user)
	if len(mods) > 257 {
		t.Fatalf("%d entries; at most 256 directories are scanned plus one error entry", len(mods))
	}
	last := mods[len(mods)-1]
	if last.Valid || !hasProblem(last.Errors, "", "too many modules") {
		t.Fatalf("last entry = %+v", last)
	}
}

// snapshot hashes the names, modes, and contents of every file under dir.
func snapshot(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	var paths []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		paths = append(paths, p)
		return nil
	})
	sort.Strings(paths)
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		fmt.Fprintf(h, "%s %v %d %d\n", p, info.Mode(), info.Size(), info.ModTime().UnixNano())
		if info.Mode().IsRegular() {
			b, _ := os.ReadFile(p)
			h.Write(b)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestScanNeverWrites(t *testing.T) {
	bundled := stage(t, "good-min", "good-full", "warn-reserved")
	user := stage(t, "err-bad-json", "err-helper-notexec", "dup-a", "dup-b", "bad name!")
	before := snapshot(t, bundled) + snapshot(t, user)
	scanDirs(t, bundled, user)
	scanDirs(t, bundled, user)
	if after := snapshot(t, bundled) + snapshot(t, user); after != before {
		t.Fatal("scan changed files")
	}
}

func TestModulesBadUsage(t *testing.T) {
	if r := runProgram("modules", "{}", nil); r.code != 2 {
		t.Fatalf("no subcommand: exit %d", r.code)
	}
	if r := runProgram("modules", "{}", nil, "frobnicate"); r.code != 2 {
		t.Fatalf("unknown subcommand: exit %d", r.code)
	}
	r := runProgram("modules", "not json", nil, "scan")
	if e := parseEnvelope(t, r); e.OK || e.Error.Code != "bad_input" {
		t.Fatalf("response = %+v", e)
	}
}
