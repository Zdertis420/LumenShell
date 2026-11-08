package tests

import (
	"strings"
	"testing"
)

// `lumen modules call <id> <command>`. Contract: specs/003-bar/contracts/cli-ipc-bar.md.

func TestCallPrintsTheAnswerAndCallsTheShell(t *testing.T) {
	dir, env := fakeQS(t, `FAKE_QS_CALL_REPLY={"ok":true,"result":"focus on DP-1"}`)
	r, _ := lumen(env, "modules", "call", "bar", "focus")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	if strings.TrimSpace(r.stdout) != "focus on DP-1" {
		t.Errorf("stdout = %q", r.stdout)
	}
	if got := calls(t, dir); !strings.Contains(got, "moduleCall") || !strings.Contains(got, "bar") || !strings.Contains(got, "focus") {
		t.Errorf("the shell was not asked to call bar focus: %q", got)
	}
}

func TestCallWithNoResultPrintsNothing(t *testing.T) {
	_, env := fakeQS(t, `FAKE_QS_CALL_REPLY={"ok":true,"result":""}`)
	r, _ := lumen(env, "modules", "call", "bar", "focus")
	if r.code != 0 || strings.TrimSpace(r.stdout) != "" {
		t.Errorf("exit %d stdout %q", r.code, r.stdout)
	}
}

func TestCallRejectsBadArgumentsBeforeContactingTheShell(t *testing.T) {
	cases := map[string][]string{
		"no command":         {"modules", "call", "bar"},
		"no arguments":       {"modules", "call"},
		"an extra argument":  {"modules", "call", "bar", "focus", "now"},
		"unsafe id":          {"modules", "call", "Bad Id", "focus"},
		"dir id":             {"modules", "call", "dir:x", "focus"},
		"unsafe command":     {"modules", "call", "bar", "Do It"},
		"command with slash": {"modules", "call", "bar", "a/b"},
		"empty command":      {"modules", "call", "bar", ""},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			dir, env := fakeQS(t)
			r, _ := lumen(env, args...)
			if r.code != 2 {
				t.Errorf("exit %d, want 2 (stderr %q)", r.code, r.stderr)
			}
			if got := calls(t, dir); got != "" {
				t.Errorf("the shell must not be contacted: %q", got)
			}
		})
	}
}

func TestCallExitCodes(t *testing.T) {
	cases := []struct {
		name  string
		reply string
		code  int
		err   string
	}{
		{"unknown module", "error: unknown: unknown module 'nope'", 7, "unknown module 'nope'"},
		{"handler failed", "error: failed: module bar is not loaded", 6, "module bar is not loaded"},
		{"unknown command", "error: failed: unknown command 'zap'", 6, "unknown command"},
		{"invalid", "error: invalid: bad command", 2, "bad command"},
		{"garbage", "what", 6, "unexpected reply"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, env := fakeQS(t, "FAKE_QS_CALL_REPLY="+c.reply)
			r, _ := lumen(env, "modules", "call", "bar", "focus")
			if r.code != c.code || !strings.Contains(r.stderr, c.err) {
				t.Errorf("exit %d stderr %q, want exit %d and %q", r.code, r.stderr, c.code, c.err)
			}
		})
	}
}

func TestCallWithNoShellExits3(t *testing.T) {
	_, env := fakeQS(t, "FAKE_QS_RUNNING=0")
	r, _ := lumen(env, "modules", "call", "bar", "focus")
	if r.code != 3 {
		t.Errorf("exit %d, want 3 (stderr %q)", r.code, r.stderr)
	}
}

func TestUsageListsCall(t *testing.T) {
	_, env := fakeQS(t)
	r, _ := lumen(env, "help")
	if !strings.Contains(r.stdout, "call <id> <command>") {
		t.Errorf("usage lacks call: %q", r.stdout)
	}
}
