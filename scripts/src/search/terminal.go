package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// The terminal in which the editor and Terminal=true applications run (research R10, FR-005b).
// The terminal is the "terminal" setting, else the TERMINAL variable. Nothing is guessed.

const noTerminalMessage = "No terminal: set modules.prism-search.terminal or the TERMINAL variable " +
	"(see the EDITOR note: a shell startup file such as .bashrc is not read by the shell)."

// execFlags is the flag that precedes the command, by terminal program name. A terminal that
// is not listed gets "-e". An empty list means the command follows directly.
var execFlags = map[string][]string{
	"kitty":          {},
	"wezterm":        {"start", "--"},
	"gnome-terminal": {"--"},
	"kgx":            {"--"},
}

// resolveTerminal returns the terminal program and the arguments that come before the command.
func resolveTerminal(setting, flagSetting string) (string, []string, *failure) {
	words := splitWords(setting)
	if len(words) == 0 {
		words = splitWords(os.Getenv("TERMINAL"))
	}
	if len(words) == 0 {
		return "", nil, fail("no_terminal", "%s", noTerminalMessage)
	}
	prog := words[0]
	args := append([]string{}, words[1:]...)
	if flagSetting != "" {
		return prog, append(args, flagSetting), nil
	}
	if flags, ok := execFlags[filepath.Base(prog)]; ok {
		return prog, append(args, flags...), nil
	}
	return prog, append(args, "-e"), nil
}

// startDetached starts a program in a new session with no input and no output, and does not
// wait for it, so it outlives this helper and Prism. It reports only a failure to start.
func startDetached(prog string, args []string, dir string) error {
	path, err := exec.LookPath(prog)
	if err != nil {
		return err
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
