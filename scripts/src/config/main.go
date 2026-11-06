// Command config resolves, sets, and unsets Lumen configuration.
//
// Usage: config resolve|set|unset   (one JSON object on stdin, one JSON document on stdout)
// See specs/001-foundation/contracts/helper-protocol.md.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const protocol = 1

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type response struct {
	OK       bool           `json:"ok"`
	Protocol int            `json:"protocol"`
	Data     any            `json:"data,omitempty"`
	Error    *responseError `json:"error,omitempty"`
}

var logLevels = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

// logf writes one ndjson diagnostic line to stderr, filtered by LUMEN_LOG_LEVEL.
func logf(level, msg string, fields map[string]any) {
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
		"source": "helper:config",
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

func writeResponse(r response) {
	r.Protocol = protocol
	b, err := json.Marshal(r)
	if err != nil {
		b = []byte(`{"ok":false,"protocol":1,"error":{"code":"internal","message":"cannot encode response"}}`)
	}
	fmt.Println(string(b))
}

func succeed(data any) { writeResponse(response{OK: true, Data: data}) }

func fail(code, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	logf("warn", msg, map[string]any{"code": code})
	writeResponse(response{OK: false, Error: &responseError{Code: code, Message: msg}})
}

// decodeJSON decodes one JSON document, keeping numbers as json.Number.
func decodeJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("unexpected data after the JSON document")
	}
	return v, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: config resolve|set|unset")
		os.Exit(2)
	}
	cmd := os.Args[1]
	if cmd != "resolve" && cmd != "set" && cmd != "unset" {
		fmt.Fprintf(os.Stderr, "unknown subcommand %q; use resolve, set, or unset\n", cmd)
		os.Exit(2)
	}

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail("bad_input", "cannot read stdin: %v", err)
		return
	}
	var in map[string]any
	if len(bytes.TrimSpace(raw)) == 0 {
		in = map[string]any{}
	} else {
		v, err := decodeJSON(raw)
		obj, isObj := v.(map[string]any)
		if err != nil || !isObj {
			fail("bad_input", "input must be one JSON object")
			return
		}
		in = obj
	}

	switch cmd {
	case "resolve":
		runResolve(in)
	case "set":
		runSet(in)
	case "unset":
		runUnset(in)
	}
}
