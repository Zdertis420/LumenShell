package tests

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Black-box tests of the bar's notification helper modules/bar/bin/notify, with a stub
// notify-send on a controlled PATH. Contract: specs/003-bar/contracts/helper-bar.md
// ("The notify helper").

func runNotify(t *testing.T, pathDir, markerDir string, input map[string]any) (mixerAnswer, time.Duration) {
	t.Helper()
	script := filepath.Join(repoRoot, "modules", "bar", "bin", "notify")
	cmd := exec.Command(script)
	cmd.Stdin = strings.NewReader(mustJSON(input))
	cmd.Env = []string{"PATH=" + pathDir, "STUB_MARKER_DIR=" + markerDir}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	start := time.Now()
	if err := cmd.Run(); err != nil {
		t.Fatalf("notify failed: %v\nstdout=%q stderr=%q", err, out.String(), errOut.String())
	}
	took := time.Since(start)
	var ans mixerAnswer
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &ans); err != nil {
		t.Fatalf("stdout must be exactly one JSON document: %v\nstdout=%q", err, out.String())
	}
	if ans.Protocol != 1 {
		t.Errorf("protocol = %d", ans.Protocol)
	}
	return ans, took
}

func TestNotifyRunsNotifySendWithBothTexts(t *testing.T) {
	markers := t.TempDir()
	ans, took := runNotify(t, pathWith(t, "notify-send"), markers, map[string]any{"summary": "S", "body": "B"})
	if !ans.OK {
		t.Fatalf("answer = %+v", ans)
	}
	if got := waitForMarker(t, markers, "notify-send"); got != "notify-send -- S B" {
		t.Errorf("notify-send arguments = %q", got)
	}
	// It answers at once, although notify-send (the stub) keeps running.
	if took > time.Second {
		t.Errorf("helper took %v", took)
	}
}

func TestNotifyCutsLongTexts(t *testing.T) {
	markers := t.TempDir()
	long := strings.Repeat("x", 300)
	ans, _ := runNotify(t, pathWith(t, "notify-send"), markers, map[string]any{"summary": long, "body": long})
	if !ans.OK {
		t.Fatalf("answer = %+v", ans)
	}
	got := waitForMarker(t, markers, "notify-send")
	want := "notify-send -- " + strings.Repeat("x", 200) + " " + strings.Repeat("x", 200)
	if got != want {
		t.Errorf("texts must be cut to 200 characters; got %d characters", len(got))
	}
}

func TestNotifyWithoutNotifySend(t *testing.T) {
	ans, _ := runNotify(t, pathWith(t), t.TempDir(), map[string]any{"summary": "S", "body": "B"})
	if ans.OK || ans.Error == nil || ans.Error.Code != "no_notifier" {
		t.Fatalf("answer = %+v", ans)
	}
	if want := "notify-send is not installed; install libnotify to get notifications"; ans.Error.Message != want {
		t.Errorf("message = %q", ans.Error.Message)
	}
}
