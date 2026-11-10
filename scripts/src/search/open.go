package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Opening files in the user's editor and the web search in the user's browser (spec FR-005,
// FR-005a, FR-018a, FR-026, FR-028; research R10, R13). The programs come from the environment
// that starts Lumen. Nothing is guessed.

const (
	noEditorMessage  = `EDITOR is not set. Set it in the environment that starts Lumen (a shell file such as .bashrc is not read by the shell). For niri: environment { EDITOR "nvim" } in config.kdl, then restart the session.`
	noBrowserMessage = `BROWSER is not set. Set it in the environment that starts Lumen (a shell file such as .bashrc is not read by the shell). For niri: environment { BROWSER "firefox" } in config.kdl, then restart the session.`
	maxURLLength     = 2000 // documented constant: longest address the web search builds
)

// lineStyle says how an editor is told to start at a line.
type lineStyle int

const (
	lineNone  lineStyle = iota
	linePlus            // +N file
	lineColon           // file:N
	lineGoto            // -g file:N
)

var editorStyles = map[string]lineStyle{
	"vi": linePlus, "vim": linePlus, "nvim": linePlus, "view": linePlus, "nano": linePlus,
	"emacs": linePlus, "emacsclient": linePlus, "kak": linePlus, "joe": linePlus,
	"mcedit": linePlus, "jed": linePlus,
	"hx": lineColon, "helix": lineColon, "micro": lineColon, "zed": lineColon, "zeditor": lineColon,
	"code": lineGoto, "codium": lineGoto, "cursor": lineGoto,
}

func opOpen(req *request) (any, *failure) {
	path := req.Path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fail("not_found", "The file no longer exists: %s", filepath.Base(path))
	}
	if info.IsDir() {
		return nil, fail("is_directory", "Only files are opened.")
	}
	editor := splitWords(os.Getenv("EDITOR"))
	if len(editor) == 0 {
		return nil, fail("no_editor", "%s", noEditorMessage)
	}
	term, pre, ferr := resolveTerminal(req.Terminal, req.TerminalExecFlag)
	if ferr != nil {
		return nil, ferr
	}
	args := append([]string{}, pre...)
	args = append(args, editor...)
	args = append(args, fileArgs(filepath.Base(editor[0]), path, req.Line)...)
	if err := startDetached(term, args, ""); err != nil {
		return nil, fail("start_failed", "Could not start %s.", filepath.Base(term))
	}
	return map[string]any{"started": filepath.Base(editor[0])}, nil
}

// fileArgs returns the arguments that name the file and, for known editors, the line.
func fileArgs(editor, path string, line int) []string {
	if line <= 0 {
		return []string{path}
	}
	n := strconv.Itoa(line)
	switch editorStyles[editor] {
	case linePlus:
		return []string{"+" + n, path}
	case lineColon:
		return []string{path + ":" + n}
	case lineGoto:
		return []string{"-g", path + ":" + n}
	}
	return []string{path}
}

func opWeb(req *request) (any, *failure) {
	tmpl := req.SearchURL
	if !(strings.HasPrefix(tmpl, "http://") || strings.HasPrefix(tmpl, "https://")) || strings.Count(tmpl, "%s") != 1 {
		return nil, fail("bad_engine", "The search engine address is not valid.")
	}
	url := strings.Replace(tmpl, "%s", percentEncode(strings.TrimSpace(req.Query)), 1)
	if len(url) > maxURLLength {
		return nil, fail("too_long", "The search text is too long for a web address.")
	}
	raw := os.Getenv("BROWSER")
	if strings.TrimSpace(raw) == "" {
		return nil, fail("no_browser", "%s", noBrowserMessage)
	}
	var tried string
	for _, alt := range strings.Split(raw, ":") {
		words := splitWords(alt)
		if len(words) == 0 {
			continue
		}
		args := browserArgs(words[1:], url)
		if err := startDetached(words[0], args, ""); err == nil {
			return map[string]any{"started": filepath.Base(words[0])}, nil
		}
		if tried == "" {
			tried = filepath.Base(words[0])
		}
	}
	return nil, fail("start_failed", "Could not start %s.", tried)
}

// browserArgs puts the address where %s is, or last when no argument holds %s.
func browserArgs(args []string, url string) []string {
	out := make([]string, 0, len(args)+1)
	used := false
	for _, a := range args {
		if strings.Contains(a, "%s") {
			a = strings.ReplaceAll(a, "%s", url)
			used = true
		}
		out = append(out, a)
	}
	if !used {
		out = append(out, url)
	}
	return out
}

// percentEncode encodes a query component: every byte except unreserved characters.
func percentEncode(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}
