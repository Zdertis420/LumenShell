package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeQS makes a directory with a `qs` that is the fixture script, and returns the
// directory plus the base environment for running lumen against it.
func fakeQS(t *testing.T, extra ...string) (dir string, env []string) {
	t.Helper()
	dir = t.TempDir()
	script := filepath.Join(repoRoot, "scripts", "tests", "fixtures", "qs-fake.sh")
	if err := os.Symlink(script, filepath.Join(dir, "qs")); err != nil {
		t.Fatal(err)
	}
	env = []string{
		"PATH=" + dir + ":/usr/bin:/bin",
		"FAKE_QS_DIR=" + dir,
		"LUMEN_ROOT=" + repoRoot,
		"LUMEN_SKIP_SESSION_CHECK=1",
		"LUMEN_HELPER_DIR=" + t.TempDir(),
	}
	// Fake helper programs so `start` finds them.
	helperDir := env[len(env)-1][len("LUMEN_HELPER_DIR="):]
	for _, name := range []string{"config", "palette", "modules"} {
		if err := os.WriteFile(filepath.Join(helperDir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir, append(env, extra...)
}

func lumen(env []string, args ...string) (result, time.Duration) {
	start := time.Now()
	r := runProgram("lumen", "", env, args...)
	return r, time.Since(start)
}

func calls(t *testing.T, dir string) string {
	b, _ := os.ReadFile(filepath.Join(dir, "calls.log"))
	return string(b)
}

func TestControllerExitCodes(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		env     []string
		code    int
		message string // substring expected on stderr
	}{
		{"no arguments", nil, nil, 2, "usage"},
		{"unknown command", []string{"bogus"}, nil, 2, "unknown command"},
		{"start with arguments", []string{"start", "x"}, nil, 2, "no arguments"},
		{"theme without name", []string{"theme"}, nil, 2, "needs a theme"},
		{"theme bad name", []string{"theme", "Bad Name"}, nil, 2, "invalid theme name"},
		{"theme extra argument", []string{"theme", "a", "b"}, nil, 2, "unexpected argument"},
		{"theme unknown option", []string{"theme", "a", "--nope"}, nil, 2, "unknown option"},
		{"reset with theme", []string{"theme", "a", "--reset"}, nil, 2, "cannot be combined"},
		{"shell rejects theme", []string{"theme", "nope"}, []string{"FAKE_QS_REPLY=error: invalid: unknown theme 'nope'"}, 2, "unknown theme"},
		{"invalid reported by status", []string{"theme", "nope"}, []string{"FAKE_QS_STATUS=pending 7,failed 7: invalid: unknown theme"}, 2, "unknown theme"},
		{"no shell", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_RUNNING=0"}, 3, "no Lumen shell is running"},
		{"reset with no shell", []string{"theme", "--reset"}, []string{"FAKE_QS_RUNNING=0"}, 3, "no Lumen shell is running"},
		{"start with shell running", []string{"start"}, nil, 4, "already running"},
		{"protocol mismatch", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_VERSION=2"}, 5, "protocol"},
		{"shell failure reply", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_REPLY=error: failed: disk on fire"}, 6, "disk on fire"},
		{"shell failure status", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_STATUS=pending 7,failed 7: cannot write"}, 6, "cannot write"},
		{"unexpected reply", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_REPLY=banana"}, 6, "unexpected reply"},
		{"list fails", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_LIST_FAIL=1"}, 8, "qs list failed"},
		{"success", []string{"theme", "tokyo-night"}, nil, 0, ""},
		{"success after pending", []string{"theme", "tokyo-night", "--temp"}, []string{"FAKE_QS_STATUS=pending 7,pending 7,pending 7,done 7"}, 0, ""},
		{"later command superseded ours", []string{"theme", "tokyo-night"}, []string{"FAKE_QS_STATUS=done 9"}, 0, ""},
		{"reset success", []string{"theme", "--reset"}, []string{"FAKE_QS_REPLY=queued 3", "FAKE_QS_STATUS=done 3"}, 0, ""},
		{"start success", []string{"start"}, []string{"FAKE_QS_RUNNING=0"}, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, env := fakeQS(t, c.env...)
			r, took := lumen(env, c.args...)
			if r.code != c.code {
				t.Fatalf("exit %d, want %d\nstdout=%q\nstderr=%q", r.code, c.code, r.stdout, r.stderr)
			}
			if c.message != "" && !strings.Contains(r.stderr, c.message) {
				t.Fatalf("stderr %q does not contain %q", r.stderr, c.message)
			}
			if c.code != 0 && strings.Count(strings.TrimSpace(r.stderr), "\n") > 1 && c.name != "no arguments" && c.name != "unknown command" {
				t.Fatalf("expected one clear line, got %q", r.stderr)
			}
			if took > time.Second {
				t.Fatalf("took %s, want under 1 s", took)
			}
		})
	}
}

func TestControllerThemeCallsThePersistFlag(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"theme", "tokyo-night"}, "setTheme tokyo-night true"},
		{[]string{"theme", "tokyo-night", "--temp"}, "setTheme tokyo-night false"},
		{[]string{"theme", "--reset"}, "resetTheme"},
	} {
		dir, env := fakeQS(t)
		if r, _ := lumen(env, tc.args...); r.code != 0 {
			t.Fatalf("%v: exit %d %q", tc.args, r.code, r.stderr)
		}
		if !strings.Contains(calls(t, dir), tc.want) {
			t.Fatalf("%v: calls = %q, want %q", tc.args, calls(t, dir), tc.want)
		}
	}
}

func TestControllerStartRefusesToStartTwiceAndUsesNoDuplicate(t *testing.T) {
	dir, env := fakeQS(t, "FAKE_QS_RUNNING=0")
	r, _ := lumen(env, "start")
	if r.code != 0 || !strings.Contains(r.stdout, "STARTED") {
		t.Fatalf("exit %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	if c := calls(t, dir); !strings.Contains(c, "-n") {
		t.Fatalf("qs should be started with -n, calls = %q", c)
	}
}

func TestControllerMissingQS(t *testing.T) {
	r := runProgram("lumen", "", []string{"PATH=/nonexistent", "LUMEN_ROOT=" + repoRoot}, "version")
	if r.code != 8 || !strings.Contains(r.stderr, "qs") {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
}

func TestControllerStartNeedsNiriAndHelpers(t *testing.T) {
	_, env := fakeQS(t, "FAKE_QS_RUNNING=0", "LUMEN_SKIP_SESSION_CHECK=0", "NIRI_SOCKET=")
	r, _ := lumen(env, "start")
	if r.code != 8 || !strings.Contains(r.stderr, "niri") {
		t.Fatalf("outside niri: exit %d stderr %q", r.code, r.stderr)
	}

	_, env = fakeQS(t, "FAKE_QS_RUNNING=0", "LUMEN_HELPER_DIR="+t.TempDir())
	r, _ = lumen(env, "start")
	if r.code != 8 || !strings.Contains(r.stderr, "make build") {
		t.Fatalf("missing helpers: exit %d stderr %q", r.code, r.stderr)
	}
}

func TestControllerBadRoot(t *testing.T) {
	_, env := fakeQS(t, "LUMEN_ROOT="+t.TempDir())
	r, _ := lumen(env, "version")
	if r.code != 8 || !strings.Contains(r.stderr, "shell.qml") {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
}

func TestControllerTimesOutOnAHungShell(t *testing.T) {
	_, env := fakeQS(t, "FAKE_QS_HANG=1")
	r, took := lumen(env, "theme", "tokyo-night")
	if r.code != 9 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	if took > 8*time.Second {
		t.Fatalf("took %s", took)
	}
}

func TestControllerTimesOutWhenStatusStaysPending(t *testing.T) {
	_, env := fakeQS(t, "FAKE_QS_STATUS=pending 7")
	r, took := lumen(env, "theme", "tokyo-night")
	if r.code != 9 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	if took > 6*time.Second {
		t.Fatalf("took %s", took)
	}
}

func TestControllerVersion(t *testing.T) {
	_, env := fakeQS(t)
	r, _ := lumen(env, "version")
	if r.code != 0 || !strings.Contains(r.stdout, "protocol 1") || !strings.Contains(r.stdout, "running") {
		t.Fatalf("exit %d stdout %q", r.code, r.stdout)
	}
	_, env = fakeQS(t, "FAKE_QS_RUNNING=0")
	r, _ = lumen(env, "version")
	if r.code != 0 || !strings.Contains(r.stdout, "not running") {
		t.Fatalf("exit %d stdout %q", r.code, r.stdout)
	}
}

func TestControllerSourceHasNoThirdPartyImportsAndOnlyCallsQS(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "scripts", "src", "lumen", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"`) && strings.Contains(line, ".") && strings.Contains(line, "/") {
			t.Errorf("controller imports a non-stdlib package: %s", line)
		}
	}
	for _, forbidden := range []string{"config.json", "XDG_CONFIG_HOME", "net/http"} {
		if strings.Contains(src, forbidden) {
			t.Errorf("controller must not touch configuration or the network (found %q)", forbidden)
		}
	}
}
