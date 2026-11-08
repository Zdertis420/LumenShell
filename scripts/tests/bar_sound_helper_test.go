package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Black-box tests of the sound item's helper modules/bar-sound/bin/open-mixer, with a
// controlled PATH that holds stub mixers. Contract: specs/003-bar/contracts/helper-bar.md
// ("The mixer helper").

type mixerAnswer struct {
	OK       bool `json:"ok"`
	Protocol int  `json:"protocol"`
	Data     struct {
		Started string `json:"started"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// pathWith returns a directory that holds the few system tools the script needs (as
// symlinks) plus the given stub programs, each a script that records its name and arguments
// in the file named by STUB_MARKER_<NAME>.
func pathWith(t *testing.T, stubs ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range []string{"cat", "sed", "setsid", "sleep", "sh", "tr"} {
		p, err := exec.LookPath(tool)
		if err != nil {
			t.Skipf("%s not found: %v", tool, err)
		}
		if err := os.Symlink(p, filepath.Join(dir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range stubs {
		body := "#!/bin/sh\necho \"" + name + " $*\" > \"$STUB_MARKER_DIR/" + name + "\"\nsleep 3\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runMixer(t *testing.T, pathDir, markerDir, command string) (mixerAnswer, time.Duration, string) {
	t.Helper()
	script := filepath.Join(repoRoot, "modules", "bar-sound", "bin", "open-mixer")
	cmd := exec.Command(script)
	cmd.Stdin = strings.NewReader(mustJSON(map[string]any{"command": command}))
	cmd.Env = []string{"PATH=" + pathDir, "STUB_MARKER_DIR=" + markerDir}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	start := time.Now()
	if err := cmd.Run(); err != nil {
		t.Fatalf("open-mixer failed: %v\nstdout=%q stderr=%q", err, out.String(), errOut.String())
	}
	took := time.Since(start)
	var ans mixerAnswer
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &ans); err != nil {
		t.Fatalf("stdout must be exactly one JSON document: %v\nstdout=%q", err, out.String())
	}
	if ans.Protocol != 1 {
		t.Errorf("protocol = %d", ans.Protocol)
	}
	return ans, took, out.String()
}

// waitForMarker waits until the stub has written its marker (it runs detached).
func waitForMarker(t *testing.T, markerDir, name string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(filepath.Join(markerDir, name)); err == nil && len(b) > 0 {
			return strings.TrimSpace(string(b))
		}
		time.Sleep(20 * time.Millisecond)
	}
	return ""
}

func noMarker(markerDir, name string) bool {
	_, err := os.Stat(filepath.Join(markerDir, name))
	return err != nil
}

func TestMixerPrefersEasyEffects(t *testing.T) {
	markers := t.TempDir()
	ans, took, _ := runMixer(t, pathWith(t, "easyeffects", "pavucontrol"), markers, "")
	if !ans.OK || ans.Data.Started != "easyeffects" {
		t.Fatalf("answer = %+v", ans)
	}
	if got := waitForMarker(t, markers, "easyeffects"); got != "easyeffects" {
		t.Errorf("easyeffects was not started: %q", got)
	}
	if !noMarker(markers, "pavucontrol") {
		t.Error("only one mixer may be started")
	}
	// The helper answers at once, although the started program keeps running.
	if took > time.Second {
		t.Errorf("helper took %v", took)
	}
}

func TestMixerFallsBackToPavucontrol(t *testing.T) {
	markers := t.TempDir()
	ans, _, _ := runMixer(t, pathWith(t, "pavucontrol"), markers, "")
	if !ans.OK || ans.Data.Started != "pavucontrol" {
		t.Fatalf("answer = %+v", ans)
	}
	if waitForMarker(t, markers, "pavucontrol") == "" {
		t.Error("pavucontrol was not started")
	}
}

func TestMixerRunsTheConfiguredCommandOnly(t *testing.T) {
	markers := t.TempDir()
	ans, _, _ := runMixer(t, pathWith(t, "easyeffects", "pavucontrol", "mymixer"), markers, "mymixer --flag 7")
	if !ans.OK || ans.Data.Started != "mymixer" {
		t.Fatalf("answer = %+v", ans)
	}
	if got := waitForMarker(t, markers, "mymixer"); got != "mymixer --flag 7" {
		t.Errorf("arguments must reach the program, got %q", got)
	}
	time.Sleep(100 * time.Millisecond)
	if !noMarker(markers, "easyeffects") || !noMarker(markers, "pavucontrol") {
		t.Error("a configured command replaces the automatic choice")
	}
}

func TestMixerNoneInstalled(t *testing.T) {
	ans, _, _ := runMixer(t, pathWith(t), t.TempDir(), "")
	if ans.OK || ans.Error == nil || ans.Error.Code != "no_mixer" {
		t.Fatalf("answer = %+v", ans)
	}
	want := "no audio mixer found (install EasyEffects or pavucontrol, or set modules.bar-sound.mixer)"
	if ans.Error.Message != want {
		t.Errorf("message = %q", ans.Error.Message)
	}
}

func TestMixerConfiguredProgramMissing(t *testing.T) {
	ans, _, _ := runMixer(t, pathWith(t, "easyeffects"), t.TempDir(), "no-such-mixer --x")
	if ans.OK || ans.Error == nil || ans.Error.Code != "start_failed" {
		t.Fatalf("answer = %+v", ans)
	}
	if !strings.Contains(ans.Error.Message, "no-such-mixer") {
		t.Errorf("the message must name the program: %q", ans.Error.Message)
	}
}
