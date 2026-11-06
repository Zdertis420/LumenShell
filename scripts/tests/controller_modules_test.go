package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const listJSON = `{"attribution":"active","modules":[
 {"id":"test-surface","name":"Test surface","version":"1.0.0","origin":"bundled","state":"loaded","enabled":true,"defaultEnabled":true,"permissions":["config.read","log.read"],"runtimeErrors":0,"lastError":"","warnings":[]},
 {"id":"hello","name":"Hello","version":"1.0.0","origin":"user","state":"disabled","enabled":false,"defaultEnabled":false,"permissions":["helpers"],"runtimeErrors":0,"lastError":"","warnings":[]},
 {"id":"broken","name":"Broken","version":"0.1.0","origin":"user","state":"failed","enabled":true,"defaultEnabled":false,"permissions":[],"runtimeErrors":0,"lastError":"api: requires interface 2.0, Lumen provides 1.0","warnings":[]},
 {"id":"noisy","name":"Noisy","version":"2.0.0","origin":"user","state":"loaded","enabled":true,"defaultEnabled":false,"permissions":[],"runtimeErrors":2,"lastError":"TypeError","warnings":["contributions is not supported in module interface 1.0"]}
]}`

func detailJSON(id, state, lastError string) string {
	return `{"id":"` + id + `","name":"Name of ` + id + `","version":"1.2.3","origin":"user","state":"` + state + `","enabled":true,"defaultEnabled":false,` +
		`"permissions":["helpers"],"runtimeErrors":1,"lastError":"` + lastError + `",` +
		`"warnings":[{"field":"contributions","problem":"contributions is not supported in module interface 1.0"}],` +
		`"description":"A test module","dir":"/x/` + id + `","apiRequired":"1.0","apiProvided":"1.0",` +
		`"errors":[{"field":"api","problem":"bad"}],` +
		`"settings":{"size":{"type":"integer","default":3,"min":1,"max":5}},"settingValues":{"size":4},` +
		`"settingErrors":[{"key":"greeting","problem":"must be a string","fallback":"Hello"}],"helpers":["hello"]}`
}

// modulesEnv makes a directory with detail files and returns the environment for the fake qs.
func modulesEnv(t *testing.T, details map[string]string, extra ...string) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	for id, body := range details {
		if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	qsDir, env := fakeQS(t, append([]string{"FAKE_QS_MODULES_DIR=" + dir, "FAKE_QS_MODULES_LIST=" + strings.ReplaceAll(listJSON, "\n", "")}, extra...)...)
	return qsDir, env
}

func TestModulesListPrintsOneLinePerModule(t *testing.T) {
	_, env := modulesEnv(t, nil)
	r, took := lumen(env, "modules", "list")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	lines := strings.Split(strings.TrimSpace(r.stdout), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected a header and 4 modules, got:\n%s", r.stdout)
	}
	for _, want := range []string{"ID", "VERSION", "ORIGIN", "STATE", "NOTES"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header lacks %s: %q", want, lines[0])
		}
	}
	checks := map[string][]string{
		"test-surface": {"1.0.0", "bundled", "loaded"},
		"hello":        {"user", "disabled", "permissions: helpers"},
		"broken":       {"failed", "api: requires interface 2.0, Lumen provides 1.0"},
		"noisy":        {"loaded", "2 runtime errors", "1 warning"},
	}
	for id, wants := range checks {
		found := false
		for _, l := range lines[1:] {
			if strings.HasPrefix(l, id+" ") {
				found = true
				for _, w := range wants {
					if !strings.Contains(l, w) {
						t.Errorf("line for %s lacks %q: %q", id, w, l)
					}
				}
			}
		}
		if !found {
			t.Errorf("no line for %s in:\n%s", id, r.stdout)
		}
	}
	if took > time.Second {
		t.Errorf("took %s", took)
	}
}

func TestModulesListNotesDegradedAttribution(t *testing.T) {
	_, env := modulesEnv(t, nil, `FAKE_QS_MODULES_LIST={"attribution":"degraded","modules":[]}`)
	r, _ := lumen(env, "modules", "list")
	if r.code != 0 || !strings.Contains(r.stdout, "runtime errors of modules are not being counted") {
		t.Fatalf("exit %d stdout %q", r.code, r.stdout)
	}
}

func TestModulesShowPrintsDetails(t *testing.T) {
	_, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "disabled", "")})
	r, _ := lumen(env, "modules", "show", "hello")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	for _, want := range []string{
		"hello", "Name of hello", "1.2.3", "A test module", "user", "/x/hello", "disabled",
		"requires 1.0", "provides 1.0", "permissions: helpers", "helpers: hello",
		"size", "integer", "4", "greeting", "must be a string", "api: bad", "contributions is not supported",
		"runtime errors: 1",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("show output lacks %q:\n%s", want, r.stdout)
		}
	}
}

func TestModulesStateChangingCommands(t *testing.T) {
	details := map[string]string{
		"hello":  detailJSON("hello", "loaded", ""),
		"broken": detailJSON("broken", "failed", "api: requires interface 2.0"),
		"off":    detailJSON("off", "disabled", ""),
	}
	cases := []struct {
		name string
		args []string
		code int
		out  string
		err  string
	}{
		{"enable loaded", []string{"modules", "enable", "hello"}, 0, "hello: loaded", ""},
		{"enable failed", []string{"modules", "enable", "broken"}, 6, "", "api: requires interface 2.0"},
		{"reload loaded", []string{"modules", "reload", "hello"}, 0, "hello: loaded", ""},
		{"reload failed", []string{"modules", "reload", "broken"}, 6, "", "api: requires interface 2.0"},
		{"disable", []string{"modules", "disable", "off"}, 0, "off: disabled", ""},
		{"rescan", []string{"modules", "rescan"}, 0, "rescanned", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, env := modulesEnv(t, details, "FAKE_QS_STATUS=pending 7,pending 7,done 7")
			r, took := lumen(env, c.args...)
			if r.code != c.code {
				t.Fatalf("exit %d, want %d\nstdout=%q\nstderr=%q", r.code, c.code, r.stdout, r.stderr)
			}
			if c.out != "" && !strings.Contains(r.stdout, c.out) {
				t.Errorf("stdout %q lacks %q", r.stdout, c.out)
			}
			if c.err != "" && !strings.Contains(r.stderr, c.err) {
				t.Errorf("stderr %q lacks %q", r.stderr, c.err)
			}
			if took > time.Second {
				t.Errorf("took %s", took)
			}
			// The state is read only after the command has settled (done), never before.
			log := calls(t, dir)
			if c.args[1] != "rescan" && c.args[1] != "disable" && strings.Index(log, "moduleShow") < strings.LastIndex(log, "status") {
				t.Errorf("moduleShow must come after the last status poll:\n%s", log)
			}
		})
	}
}

func TestModulesCallTheRightFunctions(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"modules", "list"}, "modulesList"},
		{[]string{"modules", "show", "hello"}, "moduleShow hello"},
		{[]string{"modules", "enable", "hello"}, "moduleEnable hello"},
		{[]string{"modules", "disable", "hello"}, "moduleDisable hello"},
		{[]string{"modules", "reload", "hello"}, "moduleReload hello"},
		{[]string{"modules", "rescan"}, "modulesRescan"},
	} {
		dir, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "loaded", "")})
		if r, _ := lumen(env, tc.args...); r.code != 0 {
			t.Fatalf("%v: exit %d %q", tc.args, r.code, r.stderr)
		}
		if !strings.Contains(calls(t, dir), tc.want) {
			t.Errorf("%v: calls = %q, want %q", tc.args, calls(t, dir), tc.want)
		}
	}
}

func TestModulesUnknownIDExits7(t *testing.T) {
	for _, sub := range []string{"show", "enable", "disable", "reload"} {
		_, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "loaded", "")})
		r, _ := lumen(env, "modules", sub, "nope-module")
		if r.code != 7 || !strings.Contains(r.stderr, "unknown module") {
			t.Errorf("%s: exit %d stderr %q", sub, r.code, r.stderr)
		}
	}
	// An unknown module reported through the status poll maps to 7 as well.
	_, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "loaded", "")}, "FAKE_QS_STATUS=pending 7,failed 7: unknown: unknown module 'hello'")
	if r, _ := lumen(env, "modules", "enable", "hello"); r.code != 7 {
		t.Errorf("status-reported unknown module: exit %d stderr %q", r.code, r.stderr)
	}
}

func TestModulesBadArgumentsExit2WithoutContactingQS(t *testing.T) {
	cases := [][]string{
		{"modules"},
		{"modules", "frobnicate"},
		{"modules", "list", "extra"},
		{"modules", "rescan", "extra"},
		{"modules", "show"},
		{"modules", "enable"},
		{"modules", "enable", "a", "b"},
		{"modules", "enable", "Bad Id"},
		{"modules", "enable", "UPPER"},
		{"modules", "show", "dir:has space"},
		{"modules", "show", "dir:"},
		{"modules", "reload", "x"},
	}
	for _, args := range cases {
		dir, env := modulesEnv(t, nil)
		r, _ := lumen(env, args...)
		if r.code != 2 {
			t.Errorf("%v: exit %d stderr %q", args, r.code, r.stderr)
		}
		if strings.Contains(calls(t, dir), "ipc call") {
			t.Errorf("%v: qs was contacted: %q", args, calls(t, dir))
		}
	}
	// Both id forms are accepted.
	for _, id := range []string{"hello", "dir:bad_name_", "dir:My.Dir-1"} {
		_, env := modulesEnv(t, map[string]string{id: detailJSON(id, "loaded", "")})
		if r, _ := lumen(env, "modules", "show", id); r.code != 0 {
			t.Errorf("%s: exit %d stderr %q", id, r.code, r.stderr)
		}
	}
}

func TestModulesWithoutShellOrOnMismatch(t *testing.T) {
	for _, sub := range [][]string{{"list"}, {"show", "hello"}, {"enable", "hello"}, {"rescan"}} {
		_, env := modulesEnv(t, nil, "FAKE_QS_RUNNING=0")
		if r, _ := lumen(env, append([]string{"modules"}, sub...)...); r.code != 3 {
			t.Errorf("%v with no shell: exit %d stderr %q", sub, r.code, r.stderr)
		}
		_, env = modulesEnv(t, nil, "FAKE_QS_VERSION=2")
		if r, _ := lumen(env, append([]string{"modules"}, sub...)...); r.code != 5 {
			t.Errorf("%v with a protocol mismatch: exit %d stderr %q", sub, r.code, r.stderr)
		}
	}
}

func TestModulesShellFailureExits6(t *testing.T) {
	_, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "loaded", "")}, "FAKE_QS_REPLY=error: failed: disk on fire")
	r, _ := lumen(env, "modules", "enable", "hello")
	if r.code != 6 || !strings.Contains(r.stderr, "disk on fire") {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	_, env = modulesEnv(t, map[string]string{"hello": detailJSON("hello", "loaded", "")}, "FAKE_QS_STATUS=pending 7,failed 7: cannot write")
	r, _ = lumen(env, "modules", "disable", "hello")
	if r.code != 6 || !strings.Contains(r.stderr, "cannot write") {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
}

func TestModulesTimeOut(t *testing.T) {
	_, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "loaded", "")}, "FAKE_QS_HANG=1")
	r, took := lumen(env, "modules", "enable", "hello")
	if r.code != 9 || took > 8*time.Second {
		t.Fatalf("exit %d after %s", r.code, took)
	}
}
