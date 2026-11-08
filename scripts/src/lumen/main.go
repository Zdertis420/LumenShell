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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
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
	exitUnknownModule  = 7
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
	var sub, modID, modArg string
	switch cmd {
	case "modules":
		var f *failure
		sub, modID, modArg, f = parseModules(rest)
		if f != nil {
			return report(f)
		}
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
	case "modules":
		f = c.modules(sub, modID, modArg)
	}
	return report(f)
}

func usage() string {
	return "usage: lumen start | lumen theme <theme> [--temp] | lumen theme --reset | lumen version |\n" +
		"       lumen modules list | show <id> | enable <id> | disable <id> | reload <id> | rescan | call <id> <command>"
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
	for _, name := range []string{"config", "palette", "modules"} {
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
	case strings.HasPrefix(reply, "error: unknown:"):
		return fail(exitUnknownModule, "%s", strings.TrimSpace(strings.TrimPrefix(reply, "error: unknown:")))
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
			if rest, ok := strings.CutPrefix(msg, "unknown:"); ok {
				return fail(exitUnknownModule, "%s", strings.TrimSpace(rest))
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

// ---- lumen modules ... ----

var (
	moduleIDPattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)
	moduleDirIDPattern = regexp.MustCompile(`^dir:[A-Za-z0-9._-]{1,64}$`)
)

const modulesUsage = "usage: lumen modules list | show <id> | enable <id> | disable <id> | reload <id> | rescan | call <id> <command>"

// commandPattern is the shape of a module command name (interface 1.1).
var commandPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// parseModules validates the arguments of `lumen modules ...` before anything is contacted.
// arg is the command name for `call`.
func parseModules(args []string) (sub, id, arg string, f *failure) {
	if len(args) == 0 {
		return "", "", "", fail(exitUsage, "modules needs a subcommand\n%s", modulesUsage)
	}
	sub = args[0]
	switch sub {
	case "list", "rescan":
		if len(args) != 1 {
			return "", "", "", fail(exitUsage, "modules %s takes no arguments", sub)
		}
		return sub, "", "", nil
	case "show", "enable", "disable", "reload":
		if len(args) != 2 {
			return "", "", "", fail(exitUsage, "modules %s needs exactly one module id", sub)
		}
		id = args[1]
		if !moduleIDPattern.MatchString(id) && !moduleDirIDPattern.MatchString(id) {
			return "", "", "", fail(exitUsage, "invalid module id %q", id)
		}
		return sub, id, "", nil
	case "call":
		if len(args) != 3 {
			return "", "", "", fail(exitUsage, "modules call needs a module id and a command\n%s", modulesUsage)
		}
		id, arg = args[1], args[2]
		if !moduleIDPattern.MatchString(id) {
			return "", "", "", fail(exitUsage, "invalid module id %q", id)
		}
		if !commandPattern.MatchString(arg) {
			return "", "", "", fail(exitUsage, "invalid command name %q", arg)
		}
		return sub, id, arg, nil
	}
	return "", "", "", fail(exitUsage, "unknown modules subcommand %q\n%s", sub, modulesUsage)
}

// modulesCall asks a module to run one of its declared commands and prints the answer.
func (c *controller) modulesCall(id, command string) *failure {
	reply, f := c.callChecked("moduleCall", id, command)
	if f != nil {
		return f
	}
	if strings.HasPrefix(reply, "error:") {
		return c.finish(reply)
	}
	var out struct {
		OK     bool   `json:"ok"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(reply), &out); err != nil || !out.OK {
		return fail(exitShellFailed, "unexpected reply from the shell: %q", reply)
	}
	if out.Result != "" {
		fmt.Println(out.Result)
	}
	return nil
}

func (c *controller) modules(sub, id, arg string) *failure {
	if f := c.connect(); f != nil {
		return f
	}
	switch sub {
	case "call":
		return c.modulesCall(id, arg)
	case "list":
		return c.modulesList()
	case "show":
		return c.modulesShow(id)
	case "rescan":
		reply, f := c.callChecked("modulesRescan")
		if f != nil {
			return f
		}
		if f := c.finish(reply); f != nil {
			return f
		}
		fmt.Println("rescanned the module directories")
		return nil
	}

	fn := map[string]string{"enable": "moduleEnable", "disable": "moduleDisable", "reload": "moduleReload"}[sub]
	reply, f := c.callChecked(fn, id)
	if f != nil {
		return f
	}
	if f := c.finish(reply); f != nil {
		return f
	}
	if sub == "disable" {
		fmt.Printf("%s: disabled\n", id)
		return nil
	}
	// The command reports "done" only after the module's state has settled, so this is final.
	detail, f := c.moduleDetail(id)
	if f != nil {
		return f
	}
	state, _ := detail["state"].(string)
	fmt.Printf("%s: %s\n", id, state)
	if state == "failed" {
		msg, _ := detail["lastError"].(string)
		if msg == "" {
			msg = "the module failed"
		}
		return fail(exitShellFailed, "%s: %s", id, msg)
	}
	return nil
}

// callChecked runs one IPC function and turns transport problems into failures.
func (c *controller) callChecked(fn string, args ...string) (string, *failure) {
	reply, errOut, err := c.call(fn, args...)
	if c.ctx.Err() != nil {
		return "", fail(exitTimeout, "the shell did not answer in time")
	}
	if err != nil {
		return "", fail(exitShellFailed, "the shell rejected the command: %v %s", err, errOut)
	}
	return reply, nil
}

func (c *controller) moduleDetail(id string) (map[string]any, *failure) {
	reply, f := c.callChecked("moduleShow", id)
	if f != nil {
		return nil, f
	}
	if strings.HasPrefix(reply, "error:") {
		return nil, c.finish(reply)
	}
	var detail map[string]any
	if err := json.Unmarshal([]byte(reply), &detail); err != nil {
		return nil, fail(exitShellFailed, "unexpected reply from the shell: %q", reply)
	}
	return detail, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (c *controller) modulesList() *failure {
	reply, f := c.callChecked("modulesList")
	if f != nil {
		return f
	}
	if strings.HasPrefix(reply, "error:") {
		return c.finish(reply)
	}
	var list struct {
		Attribution string `json:"attribution"`
		Modules     []struct {
			ID            string   `json:"id"`
			Version       string   `json:"version"`
			Origin        string   `json:"origin"`
			State         string   `json:"state"`
			Permissions   []string `json:"permissions"`
			RuntimeErrors int      `json:"runtimeErrors"`
			LastError     string   `json:"lastError"`
			Warnings      []any    `json:"warnings"`
			Notes         []string `json:"notes"` // interface 1.1: "contributes to bar", "hosts: items"
		} `json:"modules"`
	}
	if err := json.Unmarshal([]byte(reply), &list); err != nil {
		return fail(exitShellFailed, "unexpected reply from the shell: %q", reply)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVERSION\tORIGIN\tSTATE\tNOTES")
	for _, m := range list.Modules {
		var notes []string
		switch m.State {
		case "failed":
			if m.LastError != "" {
				notes = append(notes, m.LastError)
			}
		case "disabled":
			if len(m.Permissions) > 0 {
				notes = append(notes, "permissions: "+strings.Join(m.Permissions, ", "))
			}
		}
		if m.State != "failed" && m.RuntimeErrors > 0 {
			notes = append(notes, plural(m.RuntimeErrors, "runtime error", "runtime errors"))
		}
		if len(m.Warnings) > 0 {
			notes = append(notes, plural(len(m.Warnings), "warning", "warnings"))
		}
		notes = append(notes, m.Notes...)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", m.ID, m.Version, m.Origin, m.State, strings.Join(notes, "; "))
	}
	w.Flush()
	if list.Attribution == "degraded" {
		fmt.Println("note: runtime errors of modules are not being counted (the shell cannot follow its own log)")
	}
	return nil
}

// problemText formats a warning or error that is either a string or {field, problem}.
func problemText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case map[string]any:
		field, _ := x["field"].(string)
		problem, _ := x["problem"].(string)
		if field != "" {
			return field + ": " + problem
		}
		return problem
	}
	return fmt.Sprint(v)
}

func str(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func (c *controller) modulesShow(id string) *failure {
	d, f := c.moduleDetail(id)
	if f != nil {
		return f
	}
	fmt.Printf("id: %s\n", str(d, "id"))
	fmt.Printf("name: %s\n", str(d, "name"))
	fmt.Printf("version: %s\n", str(d, "version"))
	if desc := str(d, "description"); desc != "" {
		fmt.Printf("description: %s\n", desc)
	}
	fmt.Printf("origin: %s\n", str(d, "origin"))
	fmt.Printf("directory: %s\n", str(d, "dir"))
	enabled, _ := d["enabled"].(bool)
	def, _ := d["defaultEnabled"].(bool)
	fmt.Printf("state: %s (enabled: %v, default: %v)\n", str(d, "state"), enabled, def)
	fmt.Printf("interface: requires %s, Lumen provides %s\n", str(d, "apiRequired"), str(d, "apiProvided"))
	perms := []string{}
	for _, p := range anyList(d["permissions"]) {
		perms = append(perms, fmt.Sprint(p))
	}
	if len(perms) == 0 {
		fmt.Println("permissions: none")
	} else {
		fmt.Printf("permissions: %s\n", strings.Join(perms, ", "))
	}
	helpers := []string{}
	for _, h := range anyList(d["helpers"]) {
		helpers = append(helpers, fmt.Sprint(h))
	}
	if len(helpers) > 0 {
		fmt.Printf("helpers: %s\n", strings.Join(helpers, ", "))
	}

	settings, _ := d["settings"].(map[string]any)
	values, _ := d["settingValues"].(map[string]any)
	if len(settings) > 0 {
		fmt.Println("settings:")
		keys := make([]string, 0, len(settings))
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rule, _ := settings[k].(map[string]any)
			value, ok := values[k]
			if !ok {
				// Not resolved yet (for example the module is not loaded): the default applies.
				value = rule["default"]
			}
			fmt.Printf("  %s: %v (%s, default %v)\n", k, value, str(rule, "type"), rule["default"])
		}
	}
	printList := func(title string, items []any, format func(any) string) {
		if len(items) == 0 {
			return
		}
		fmt.Println(title + ":")
		for _, it := range items {
			fmt.Println("  " + format(it))
		}
	}
	printList("errors", anyList(d["errors"]), problemText)
	printList("warnings", anyList(d["warnings"]), problemText)
	printList("setting errors", anyList(d["settingErrors"]), func(v any) string {
		m, _ := v.(map[string]any)
		return fmt.Sprintf("%s: %s (fallback: %v)", str(m, "key"), str(m, "problem"), m["fallback"])
	})
	printInterface11(d)
	runtime, _ := d["runtimeErrors"].(float64)
	fmt.Printf("runtime errors: %d\n", int(runtime))
	if last := str(d, "lastError"); last != "" {
		fmt.Printf("last error: %s\n", last)
	}
	return nil
}

func anyList(v any) []any {
	l, _ := v.([]any)
	return l
}

// stateText is "state" or "state (reason)".
func stateText(m map[string]any) string {
	if reason := str(m, "reason"); reason != "" {
		return str(m, "state") + " (" + reason + ")"
	}
	return str(m, "state")
}

// printInterface11 prints what interface 1.1 adds to `lumen modules show`: the module's own
// contributions, its extension points with the contributions to them, its commands, and what
// a host reported (dropped items and item warnings). A section is printed only when the reply
// has it, so an older shell's reply prints as before.
func printInterface11(d map[string]any) {
	if list := anyList(d["contributions"]); len(list) > 0 {
		fmt.Println("Contributions:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  ITEM NAME\tSHORT NAME\tTARGET\tSTATE")
		for _, it := range list {
			c, _ := it.(map[string]any)
			target, _ := c["target"].(map[string]any)
			fmt.Fprintf(w, "  %s\t%s\t%s/%s\t%s\n", str(c, "key"), str(c, "short"), str(target, "module"), str(target, "point"), stateText(c))
		}
		w.Flush()
	}
	if list := anyList(d["extensionPoints"]); len(list) > 0 {
		fmt.Println("Extension points:")
		for _, it := range list {
			p, _ := it.(map[string]any)
			contribs := anyList(p["contributions"])
			shown := 0
			for _, ci := range contribs {
				if c, _ := ci.(map[string]any); str(c, "state") == "shown" {
					shown++
				}
			}
			fmt.Printf("  %s  %s  contributions: %d (shown %d)\n", str(p, "id"), str(p, "description"), len(contribs), shown)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, ci := range contribs {
				c, _ := ci.(map[string]any)
				fmt.Fprintf(w, "    %s\t%s\t%s\n", str(c, "key"), str(c, "short"), stateText(c))
			}
			w.Flush()
		}
	}
	if list := anyList(d["commands"]); len(list) > 0 {
		fmt.Println("Commands:")
		for _, c := range list {
			fmt.Println("  " + fmt.Sprint(c))
		}
	}
	for _, it := range anyList(d["dropped"]) {
		m, _ := it.(map[string]any)
		names := []string{}
		for _, n := range anyList(m["items"]) {
			names = append(names, fmt.Sprint(n))
		}
		if len(names) > 0 {
			fmt.Printf("Dropped items (screen %s): %s\n", str(m, "screen"), strings.Join(names, ", "))
		}
	}
	if list := anyList(d["itemWarnings"]); len(list) > 0 {
		fmt.Println("Item warnings:")
		for _, it := range list {
			m, _ := it.(map[string]any)
			entry, _ := json.Marshal(m["entry"])
			index, _ := m["index"].(float64)
			fmt.Printf("  %s[%d]: %s: %s\n", str(m, "place"), int(index), entry, str(m, "problem"))
		}
	}
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
