package tests

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// checkNDJSON asserts every non-empty stderr line is a structured log entry.
func checkNDJSON(t *testing.T, label, stderr, wantSource string) int {
	t.Helper()
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
		if line == "" {
			continue
		}
		count++
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Errorf("%s: stderr line is not JSON: %q", label, line)
			continue
		}
		ts, _ := e["time"].(string)
		if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
			t.Errorf("%s: bad time %q", label, ts)
		}
		if lvl, _ := e["level"].(string); lvl != "debug" && lvl != "info" && lvl != "warn" && lvl != "error" {
			t.Errorf("%s: bad level %v", label, e["level"])
		}
		if src, _ := e["source"].(string); src != wantSource {
			t.Errorf("%s: source %v, want %s", label, e["source"], wantSource)
		}
		if msg, _ := e["msg"].(string); msg == "" {
			t.Errorf("%s: empty msg", label)
		}
	}
	return count
}

func TestHelperDiagnosticsAreStructured(t *testing.T) {
	bad := writeFile(t, `{"log":{"level":"loud"}}`)
	r := runProgram("config", mustJSON(map[string]any{"userFile": bad, "dataDir": dataDir()}), nil, "resolve")
	if n := checkNDJSON(t, "config", r.stderr, "helper:config"); n == 0 {
		t.Error("config: expected a warning for the invalid value")
	}

	r = runProgram("palette", mustJSON(map[string]any{"source": "preset", "preset": "missing", "dataDir": dataDir()}), nil)
	if n := checkNDJSON(t, "palette", r.stderr, "helper:palette"); n == 0 {
		t.Error("palette: expected a warning for the missing preset")
	}

	r = runProgram("config", "not json", nil, "resolve")
	checkNDJSON(t, "config bad input", r.stderr, "helper:config")
}

func TestLogLevelFiltersAtTheSource(t *testing.T) {
	bad := writeFile(t, `{"log":{"level":"loud"}}`)
	in := mustJSON(map[string]any{"userFile": bad, "dataDir": dataDir()})

	if r := runProgram("config", in, []string{"LUMEN_LOG_LEVEL=info"}, "resolve"); strings.TrimSpace(r.stderr) == "" {
		t.Error("level info should show the warning")
	}
	if r := runProgram("config", in, []string{"LUMEN_LOG_LEVEL=error"}, "resolve"); strings.TrimSpace(r.stderr) != "" {
		t.Errorf("level error should hide warnings, got %q", r.stderr)
	}
	// stdout is unaffected by the log level.
	r := runProgram("config", in, []string{"LUMEN_LOG_LEVEL=error"}, "resolve")
	if e := parseEnvelope(t, r); !e.OK {
		t.Errorf("response = %+v", e)
	}
}

func TestSetLogsAnInfoLineWithTheSource(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	r := runProgram("config", mustJSON(map[string]any{
		"userFile": p, "dataDir": dataDir(),
		"changes": []map[string]any{change("log.level", "debug")},
	}), []string{"LUMEN_LOG_LEVEL=info"}, "set")
	if n := checkNDJSON(t, "set", r.stderr, "helper:config"); n != 1 {
		t.Errorf("expected exactly one info line, got %d: %q", n, r.stderr)
	}
}
