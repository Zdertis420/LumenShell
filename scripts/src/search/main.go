// search is the helper of the prism-search module (specs/006-prism-modes). It follows the
// module helper contract: one JSON request on stdin, one JSON document on stdout, ndjson
// diagnostics on stderr. Diagnostics never carry a query, a pattern or a path typed by the user.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	protocol         = 1
	defaultBudgetMs  = 1500
	defaultRowLimit  = 20
	defaultMaxFiles  = 200000
	maxRowsPerAnswer = 1000
)

// request holds every field of every operation (contracts/search-helper.md).
type request struct {
	Op               string   `json:"op"`
	Mode             string   `json:"mode"`
	Query            string   `json:"query"`
	Name             string   `json:"name"`
	Dir              string   `json:"dir"`
	Stage            string   `json:"stage"`
	Limit            int      `json:"limit"`
	MaxFiles         int      `json:"maxFiles"`
	BudgetMs         int      `json:"budgetMs"`
	Hidden           bool     `json:"hidden"`
	HiddenHomeDirs   bool     `json:"hiddenHomeDirs"`
	SystemRoots      []string `json:"systemRoots"`
	ExtraDirs        []string `json:"extraDirs"`
	Dirs             []string `json:"dirs"` // replaces the default command folders (tests)
	CaseRule         string   `json:"caseRule"`
	ID               string   `json:"id"`
	Path             string   `json:"path"`
	Line             int      `json:"line"`
	Terminal         string   `json:"terminal"`
	TerminalExecFlag string   `json:"terminalExecFlag"`
	SearchURL        string   `json:"searchUrl"`
	Home             string   `json:"home"` // replaces the user's home (tests)
}

// row is what the source turns into a Prism row. Key is opaque to Prism.
type row struct {
	Key      string   `json:"key"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle,omitempty"`
	Hint     string   `json:"hint,omitempty"`
	Icon     string   `json:"icon,omitempty"`
	Marks    [][2]int `json:"marks,omitempty"`
}

type message struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// failure is an error with a machine code and a message that may be shown to the user.
type failure struct {
	Code    string
	Message string
}

func (f *failure) Error() string { return f.Code + ": " + f.Message }

func fail(code, format string, args ...any) *failure {
	return &failure{Code: code, Message: fmt.Sprintf(format, args...)}
}

var userHome string

func main() {
	defer func() {
		if r := recover(); r != nil {
			logDiag("error", "panic", map[string]any{"value": fmt.Sprint(r)})
			writeError(fail("failed", "search failed"))
			os.Exit(0)
		}
	}()
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeError(fail("bad_request", "cannot read the request"))
		return
	}
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(fail("bad_request", "the request is not valid JSON"))
		return
	}
	userHome = req.Home
	if userHome == "" {
		userHome, _ = os.UserHomeDir()
	}
	data, ferr := dispatch(&req)
	if ferr != nil {
		logDiag("warn", "operation failed", map[string]any{"op": req.Op, "code": ferr.Code})
		writeError(ferr)
		return
	}
	writeData(data)
}

func dispatch(req *request) (any, *failure) {
	switch req.Op {
	case "scope":
		return opScope(req)
	case "apps":
		return opApps(req)
	case "launch":
		return opLaunch(req)
	case "files":
		return opFiles(req)
	case "commands":
		return opCommands(req)
	case "grep":
		return opGrep(req)
	case "open":
		return opOpen(req)
	case "web":
		return opWeb(req)
	case "complete":
		return opComplete(req)
	}
	return nil, fail("bad_request", "unknown operation")
}

func writeData(data any) {
	out, err := json.Marshal(map[string]any{"ok": true, "protocol": protocol, "data": data})
	if err != nil {
		writeError(fail("failed", "search failed"))
		return
	}
	fmt.Println(string(out))
}

func writeError(f *failure) {
	out, _ := json.Marshal(map[string]any{
		"ok": false, "protocol": protocol,
		"error": map[string]string{"code": f.Code, "message": f.Message},
	})
	fmt.Println(string(out))
}

var logLevels = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

// logDiag writes one ndjson diagnostic line (the helper log contract: time, level, source and
// msg), filtered by LUMEN_LOG_LEVEL. Callers pass counts and codes, never user text.
func logDiag(level, msg string, fields map[string]any) {
	min, ok := logLevels[os.Getenv("LUMEN_LOG_LEVEL")]
	if !ok {
		min = logLevels["info"]
	}
	if logLevels[level] < min {
		return
	}
	entry := map[string]any{
		"time":   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"level":  level,
		"source": "helper:search",
		"msg":    msg,
	}
	for k, v := range fields {
		if _, exists := entry[k]; !exists {
			entry[k] = v
		}
	}
	b, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(b))
}

func budgetOf(req *request) time.Duration {
	ms := req.BudgetMs
	if ms <= 0 {
		ms = defaultBudgetMs
	}
	return time.Duration(ms) * time.Millisecond
}

func limitOf(req *request) int {
	n := req.Limit
	if n <= 0 {
		n = defaultRowLimit
	}
	if n > maxRowsPerAnswer {
		n = maxRowsPerAnswer
	}
	return n
}

// shorten replaces the home prefix of a folder by "~".
func shorten(dir string) string {
	if userHome != "" {
		if dir == userHome {
			return "~"
		}
		if len(dir) > len(userHome) && dir[:len(userHome)] == userHome && dir[len(userHome)] == '/' {
			return "~" + dir[len(userHome):]
		}
	}
	return dir
}
