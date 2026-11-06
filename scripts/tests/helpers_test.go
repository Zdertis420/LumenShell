package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	repoRoot string
	binDir   string
)

// TestMain builds the three programs once into a temp directory.
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot = filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))

	tmp, err := os.MkdirTemp("", "lumen-test-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binDir = tmp
	for _, name := range []string{"lumen", "config", "palette"} {
		srcs, _ := filepath.Glob(filepath.Join(repoRoot, "scripts", "src", name, "*.go"))
		args := append([]string{"build", "-o", filepath.Join(binDir, name)}, srcs...)
		cmd := exec.Command("go", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "build %s failed: %v\n%s", name, err, out)
			os.RemoveAll(tmp)
			os.Exit(1)
		}
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

type result struct {
	stdout string
	stderr string
	code   int
}

// runProgram runs a built program with stdin and extra environment (KEY=VALUE).
func runProgram(name, stdin string, env []string, args ...string) result {
	cmd := exec.Command(filepath.Join(binDir, name), args...)
	cmd.Stdin = strings.NewReader(stdin)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	cmd.Env = append(os.Environ(), env...)
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return result{stderr: err.Error(), code: -1}
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				code = -1
			}
		}
		return result{so.String(), se.String(), code}
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		return result{so.String(), se.String() + "\ntest harness: timeout", -2}
	}
}

// envelope is the common helper response.
type envelope struct {
	OK       bool            `json:"ok"`
	Protocol int             `json:"protocol"`
	Data     json.RawMessage `json:"data"`
	Error    *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func parseEnvelope(t *testing.T, r result) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal([]byte(r.stdout), &e); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\nstdout=%q\nstderr=%q", err, r.stdout, r.stderr)
	}
	if e.Protocol != 1 {
		t.Fatalf("protocol = %d, want 1", e.Protocol)
	}
	return e
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func dataDir() string { return filepath.Join(repoRoot, "data") }
