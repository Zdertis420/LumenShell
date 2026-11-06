// Command lumen is the one-shot controller for the Lumen shell.
//
//	lumen start                      start the shell (foreground)
//	lumen theme <theme> [--temp]     set the theme (saved, or this shell only with --temp)
//	lumen theme --reset              remove the saved theme choice
//	lumen version                    print versions
//
// It only runs `qs` (Quickshell) and never reads or writes configuration.
// Standard library only. See specs/001-foundation/contracts/cli.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	controllerVersion = "0.1.0"
	protocolVersion   = "1"

	exitOK             = 0
	exitUsage          = 2
	exitNotRunning     = 3
	exitAlreadyRunning = 4
	exitProtocol       = 5
	exitShellFailed    = 6
	exitEnv            = 8
	exitTimeout        = 9

	commandTimeout = 5 * time.Second
	pollInterval   = 50 * time.Millisecond
	pollTimeout    = 2 * time.Second
)

var themePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// failure carries a user-facing message and the exit code to use.
type failure struct {
	code int
	msg  string
}

func (f *failure) Error() string { return f.msg }

func fail(code int, format string, args ...any) *failure {
	return &failure{code: code, msg: fmt.Sprintf(format, args...)}
}

type controller struct {
	root string // directory holding shell.qml
	qs   string // path of the qs executable
	ctx  context.Context
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	code := run(ctx, os.Args[1:])
	cancel()
	os.Exit(code)
}

func run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage())
		return exitUsage
	}
	cmd, rest := args[0], args[1:]

	// Validate arguments before touching the system.
	var theme string
	var temp, reset bool
	switch cmd {
	case "start", "version":
		if len(rest) != 0 {
			return report(fail(exitUsage, "%s takes no arguments", cmd))
		}
	case "theme":
		var f *failure
		theme, temp, reset, f = parseTheme(rest)
		if f != nil {
			return report(f)
		}
	case "-h", "--help", "help":
		fmt.Println(usage())
		return exitOK
	default:
		return report(fail(exitUsage, "unknown command %q\n%s", cmd, usage()))
	}

	c, f := newController(ctx)
	if f != nil {
		return report(f)
	}

	switch cmd {
	case "start":
		f = c.start()
	case "version":
		f = c.version()
	case "theme":
		f = c.theme(theme, temp, reset)
	}
	return report(f)
}

func usage() string {
	return "usage: lumen start | lumen theme <theme> [--temp] | lumen theme --reset | lumen version"
}

func report(f *failure) int {
	if f == nil {
		return exitOK
	}
	fmt.Fprintln(os.Stderr, "lumen: "+f.msg)
	return f.code
}

func parseTheme(args []string) (theme string, temp, reset bool, f *failure) {
	for _, a := range args {
		switch {
		case a == "--temp":
			temp = true
		case a == "--reset":
			reset = true
		case strings.HasPrefix(a, "-"):
			return "", false, false, fail(exitUsage, "unknown option %q", a)
		case theme == "":
			theme = a
		default:
			return "", false, false, fail(exitUsage, "unexpected argument %q", a)
		}
	}
	switch {
	case reset && (theme != "" || temp):
		return "", false, false, fail(exitUsage, "--reset cannot be combined with a theme or --temp")
	case reset:
		return "", false, true, nil
	case theme == "":
		return "", false, false, fail(exitUsage, "theme needs a theme name or --reset")
	case !themePattern.MatchString(theme):
		return "", false, false, fail(exitUsage, "invalid theme name %q", theme)
	}
	return theme, temp, false, nil
}

// ---- environment ----

// checkSession reports whether a supported compositor session is running.
// niri is the only supported compositor; this is the one place the controller asks.
func checkSession() *failure {
	if os.Getenv("LUMEN_SKIP_SESSION_CHECK") == "1" {
		return nil
	}
	if os.Getenv("NIRI_SOCKET") == "" {
		return fail(exitEnv, "not running under niri (NIRI_SOCKET is not set)")
	}
	return nil
}

func newController(ctx context.Context) (*controller, *failure) {
	qs, err := exec.LookPath("qs")
	if err != nil {
		return nil, fail(exitEnv, "qs (Quickshell) was not found in PATH")
	}
	root := os.Getenv("LUMEN_ROOT")
	if root == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fail(exitEnv, "cannot locate the Lumen directory: %v", err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		// scripts/bin/lumen -> repository root
		root = filepath.Join(filepath.Dir(exe), "..", "..")
	}
	root = filepath.Clean(root)
	if _, err := os.Stat(filepath.Join(root, "shell.qml")); err != nil {
		return nil, fail(exitEnv, "shell.qml not found in %s (set LUMEN_ROOT)", root)
	}
	return &controller{root: root, qs: qs, ctx: ctx}, nil
}

// qsRun runs `qs -p <root> args...` and returns trimmed stdout.
func (c *controller) qsRun(args ...string) (string, string, error) {
	cmd := exec.CommandContext(c.ctx, c.qs, append([]string{"-p", c.root}, args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = 200 * time.Millisecond
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// running reports whether a shell of this config is running.
func (c *controller) running() (bool, *failure) {
	out, errOut, err := c.qsRun("list", "--json")
	if c.ctx.Err() != nil {
		return false, fail(exitTimeout, "timed out asking Quickshell for running instances")
	}
	if err != nil {
		return false, fail(exitEnv, "qs list failed: %v %s", err, errOut)
	}
	var list []json.RawMessage
	// With no instance, qs prints a plain sentence ("No running instances for ...")
	// instead of a JSON array, and still exits 0.
	if !strings.HasPrefix(out, "[") {
		return false, nil
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return false, fail(exitEnv, "cannot read the output of qs list: %v", err)
	}
	return len(list) > 0, nil
}

// ---- commands ----

func (c *controller) start() *failure {
	if f := checkSession(); f != nil {
		return f
	}
	if f := checkHelpers(c.root); f != nil {
		return f
	}
	isRunning, f := c.running()
	if f != nil {
		return f
	}
	if isRunning {
		return fail(exitAlreadyRunning, "a Lumen shell is already running")
	}
	// Replace this process with the shell, so signals and the exit status are the shell's.
	args := []string{c.qs, "-p", c.root, "-n"}
	if err := syscall.Exec(c.qs, args, os.Environ()); err != nil {
		return fail(exitEnv, "cannot start qs: %v", err)
	}
	return nil
}

// checkHelpers verifies that the helper programs have been built.
func checkHelpers(root string) *failure {
	dir := os.Getenv("LUMEN_HELPER_DIR")
	if dir == "" {
		dir = filepath.Join(root, "scripts", "bin")
	}
	for _, name := range []string{"config", "palette"} {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
			return fail(exitEnv, "helper program %q is missing in %s; run `make build`", name, dir)
		}
	}
	return nil
}

func (c *controller) version() *failure {
	fmt.Printf("lumen %s (protocol %s)\n", controllerVersion, protocolVersion)
	isRunning, f := c.running()
	if f != nil || !isRunning {
		fmt.Println("shell: not running")
		return nil
	}
	v, _, err := c.call("version")
	if err != nil {
		fmt.Println("shell: running, protocol unknown")
		return nil
	}
	fmt.Printf("shell: running, protocol %s\n", v)
	return nil
}

// call runs one IPC function of the shell.
func (c *controller) call(fn string, args ...string) (string, string, error) {
	return c.qsRun(append([]string{"ipc", "call", "lumen", fn}, args...)...)
}

// connect checks that a shell is running and speaks our protocol.
func (c *controller) connect() *failure {
	isRunning, f := c.running()
	if f != nil {
		return f
	}
	if !isRunning {
		return fail(exitNotRunning, "no Lumen shell is running; start it with `lumen start`")
	}
	v, errOut, err := c.call("version")
	if c.ctx.Err() != nil {
		return fail(exitTimeout, "the shell did not answer in time")
	}
	if err != nil {
		return fail(exitShellFailed, "the shell did not answer: %v %s", err, errOut)
	}
	if v != protocolVersion {
		return fail(exitProtocol, "the shell speaks protocol %q, this lumen speaks %q", v, protocolVersion)
	}
	return nil
}

func (c *controller) theme(theme string, temp, reset bool) *failure {
	if f := c.connect(); f != nil {
		return f
	}
	var reply, errOut string
	var err error
	switch {
	case reset:
		reply, errOut, err = c.call("resetTheme")
	default:
		reply, errOut, err = c.call("setTheme", theme, strconv.FormatBool(!temp))
	}
	if c.ctx.Err() != nil {
		return fail(exitTimeout, "the shell did not answer in time")
	}
	if err != nil {
		return fail(exitShellFailed, "the shell rejected the command: %v %s", err, errOut)
	}
	f := c.finish(reply)
	what := "theme " + theme
	if reset {
		what = "theme reset"
	}
	if f != nil {
		c.forwardLog("error", what+" failed: "+f.msg)
		return f
	}
	c.forwardLog("info", what+" applied")
	return nil
}

// finish interprets the shell's reply and, for queued work, waits for the outcome.
func (c *controller) finish(reply string) *failure {
	switch {
	case strings.HasPrefix(reply, "error: invalid:"):
		return fail(exitUsage, "%s", strings.TrimSpace(strings.TrimPrefix(reply, "error: invalid:")))
	case strings.HasPrefix(reply, "error: failed:"):
		return fail(exitShellFailed, "%s", strings.TrimSpace(strings.TrimPrefix(reply, "error: failed:")))
	case strings.HasPrefix(reply, "error:"):
		return fail(exitShellFailed, "%s", strings.TrimSpace(strings.TrimPrefix(reply, "error:")))
	case strings.HasPrefix(reply, "queued "):
		seq, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(reply, "queued ")))
		if err != nil {
			return fail(exitShellFailed, "unexpected reply from the shell: %q", reply)
		}
		return c.wait(seq)
	case reply == "ok":
		return nil
	}
	return fail(exitShellFailed, "unexpected reply from the shell: %q", reply)
}

// wait polls status() until the queued command is done or failed.
func (c *controller) wait(seq int) *failure {
	deadline := time.Now().Add(pollTimeout)
	for {
		st, _, err := c.call("status")
		if c.ctx.Err() != nil {
			return fail(exitTimeout, "timed out waiting for the shell")
		}
		if err != nil {
			return fail(exitShellFailed, "the shell stopped answering: %v", err)
		}
		kind, n, msg := parseStatus(st)
		switch {
		case kind == "done" && n >= seq:
			return nil
		case kind == "failed" && n >= seq:
			if msg == "" {
				msg = "the command failed"
			}
			// "invalid: ..." marks bad input (for example an unknown theme), not a runtime failure.
			if rest, ok := strings.CutPrefix(msg, "invalid:"); ok {
				return fail(exitUsage, "%s", strings.TrimSpace(rest))
			}
			return fail(exitShellFailed, "%s", msg)
		case kind == "unknown":
			return fail(exitShellFailed, "unexpected status from the shell: %q", st)
		}
		if time.Now().After(deadline) {
			return fail(exitTimeout, "timed out waiting for the shell to finish the command")
		}
		select {
		case <-time.After(pollInterval):
		case <-c.ctx.Done():
			return fail(exitTimeout, "timed out waiting for the shell")
		}
	}
}

// parseStatus reads "idle", "pending N", "done N", or "failed N: message".
func parseStatus(s string) (kind string, seq int, msg string) {
	fields := strings.SplitN(strings.TrimSpace(s), " ", 2)
	kind = fields[0]
	switch kind {
	case "idle":
		return kind, 0, ""
	case "pending", "done", "failed":
		if len(fields) < 2 {
			return "unknown", 0, ""
		}
		num, rest, _ := strings.Cut(fields[1], ":")
		n, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil {
			return "unknown", 0, ""
		}
		return kind, n, strings.TrimSpace(rest)
	}
	return "unknown", 0, ""
}

// forwardLog sends one entry to the shell's log stream. Best effort: errors are ignored,
// and it never changes the outcome of the command.
func (c *controller) forwardLog(level, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.qs, "-p", c.root, "ipc", "call", "lumen", "log", level, "controller", msg)
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return
	}
}
