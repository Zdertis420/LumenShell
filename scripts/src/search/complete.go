package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	completeMaxEntries = 5000 // entries read from the one directory (FR-039)
	completeShownNames = 8    // names listed in the message (FR-040)
)

// opComplete completes the last word of a files or grep query (spec FR-038 to FR-042,
// contracts/search-helper.md "complete"). Only one directory is listed; nothing is walked.
func opComplete(req *request) (any, *failure) {
	rs := []rune(req.Query)
	unchanged := func(msg *message) (any, *failure) {
		out := map[string]any{"query": req.Query, "cursor": len(rs), "changed": false}
		if msg != nil {
			out["message"] = msg
		}
		return out, nil
	}
	spans := splitSpans(req.Query)
	if len(spans) == 0 || spans[len(spans)-1].End != len(rs) {
		return unchanged(nil)
	}
	last := spans[len(spans)-1]
	dirsOnly := false
	switch req.Mode {
	case "files":
		if !pathLike(last.Value) {
			return unchanged(nil)
		}
		dirsOnly = true
	case "grep":
		if len(spans) < 2 {
			return unchanged(nil)
		}
	default:
		return unchanged(nil)
	}

	typed := last.Value
	if typed == "~" { // "~" alone is the home: only the slash is missing
		return completed(rs, last, "~/", true, false)
	}
	cut := strings.LastIndex(typed, "/")
	dirPart, rest := typed[:cut+1], typed[cut+1:]
	dir := userHome
	switch {
	case dirPart == "":
	case pathLike(dirPart):
		dir = resolvePath(dirPart)
	default:
		dir = filepath.Join(userHome, dirPart)
	}

	names, skipped, ferr := candidates(dir, rest, dirsOnly)
	logDiag("debug", "complete", map[string]any{"mode": req.Mode, "candidates": len(names)})
	if ferr != nil {
		return unchanged(&message{Kind: "error", Text: ferr.Message})
	}
	left := ""
	if skipped > 0 {
		left = " (names with a double quote are not offered)"
	}
	switch len(names) {
	case 0:
		return unchanged(&message{Kind: "info", Text: "no match" + left})
	case 1:
		n := names[0]
		word := dirPart + strings.TrimSuffix(n, "/")
		if strings.HasSuffix(n, "/") {
			word += "/"
		}
		return completed(rs, last, word, strings.HasSuffix(n, "/"), false)
	}
	common := commonStart(names)
	if len([]rune(common)) < len([]rune(rest)) {
		common = rest // a case-insensitive match: keep what was typed
	}
	word := dirPart + common
	shown := names
	more := ""
	if len(shown) > completeShownNames {
		shown = shown[:completeShownNames]
		more = ", …"
	}
	msg := &message{Kind: "info", Text: itoa(len(names)) + " matches: " + strings.Join(shown, ", ") + more + left}
	return completed(rs, last, word, false, true, msg)
}

// completed replaces the last word of the query by word (quoted when needed) and places the
// cursor: at the end, or before the closing quote when the user is likely to keep typing.
func completed(rs []rune, last span, word string, keepTyping bool, partial bool, msg ...*message) (any, *failure) {
	quote := strings.ContainsAny(word, " \t\n'")
	text := word
	if quote {
		text = "\"" + word + "\""
	}
	query := string(rs[:last.Start]) + text
	cursor := len([]rune(query))
	if quote && (keepTyping || partial) {
		cursor--
	}
	changed := query != string(rs)
	out := map[string]any{"query": query, "cursor": cursor, "changed": changed}
	if len(msg) > 0 && msg[0] != nil {
		out["message"] = msg[0]
	}
	return out, nil
}

// candidates lists the entries of dir whose names start with rest: exact case first, else
// ignoring case. Directory names end with "/". skipped counts names that cannot be written.
func candidates(dir, rest string, dirsOnly bool) (names []string, skipped int, ferr *failure) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, 0, fail("not_found", "directory not found: %s", filepath.Base(dir))
	}
	f, err := os.Open(dir)
	if err != nil {
		return nil, 0, fail("unreadable", "cannot read: %s", filepath.Base(dir))
	}
	defer f.Close()
	entries, err := f.ReadDir(completeMaxEntries)
	if err != nil && !errors.Is(err, io.EOF) && len(entries) == 0 {
		return nil, 0, fail("unreadable", "cannot read: %s", filepath.Base(dir))
	}
	type item struct {
		name string
		dir  bool
	}
	var all []item
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(rest, ".") {
			continue
		}
		isDir := e.IsDir()
		if e.Type()&os.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(dir, name)); err == nil && st.IsDir() {
				isDir = true
			}
		}
		if dirsOnly && !isDir {
			continue
		}
		all = append(all, item{name, isDir})
	}
	pick := func(match func(string) bool) {
		for _, it := range all {
			if !match(it.name) {
				continue
			}
			if strings.Contains(it.name, "\"") {
				skipped++
				continue
			}
			n := it.name
			if it.dir {
				n += "/"
			}
			names = append(names, n)
		}
	}
	pick(func(n string) bool { return strings.HasPrefix(n, rest) })
	if len(names) == 0 && skipped == 0 {
		low := strings.ToLower(rest)
		pick(func(n string) bool { return strings.HasPrefix(strings.ToLower(n), low) })
	}
	sort.Strings(names)
	return names, skipped, nil
}

// commonStart is the longest start that all names share (by characters).
func commonStart(names []string) string {
	first := []rune(names[0])
	n := len(first)
	for _, s := range names[1:] {
		r := []rune(s)
		i := 0
		for i < n && i < len(r) && r[i] == first[i] {
			i++
		}
		n = i
	}
	return string(first[:n])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
