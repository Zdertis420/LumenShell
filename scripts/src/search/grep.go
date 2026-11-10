package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Grep (spec FR-022 to FR-026, research R12): a regular expression and one or more files.

const (
	maxLineBytes = 64 * 1024 // documented constant: longer lines are cut
	binaryProbe  = 8 * 1024  // documented constant: bytes looked at for a NUL
	titleRunes   = 200       // documented constant: characters of a result line
)

func opGrep(req *request) (any, *failure) {
	words := splitWords(req.Query)
	if len(words) < 2 {
		return grepAnswer(nil, 0, false, []message{{Kind: "info", Text: "Type a pattern and at least one file."}}), nil
	}
	pattern := words[0]
	re, err := compilePattern(pattern, req.CaseRule)
	if err != nil {
		return grepAnswer(nil, 0, false, []message{{Kind: "error", Text: "The pattern is not valid: " + shortErr(err)}}), nil
	}
	limit := limitOf(req)
	deadline := time.Now().Add(budgetOf(req))
	var rows []row
	var msgs []message
	cut := false
	for _, f := range words[1:] {
		if cut {
			break
		}
		path := expandHome(f)
		if !filepath.IsAbs(path) {
			path = filepath.Join(userHome, path)
		}
		label := filepath.Base(path)
		info, err := os.Stat(path)
		switch {
		case err != nil && errors.Is(err, os.ErrNotExist):
			msgs = append(msgs, message{"error", "File not found: " + label})
			continue
		case err != nil:
			msgs = append(msgs, message{"error", "Cannot read: " + label})
			continue
		case info.IsDir():
			msgs = append(msgs, message{"error", "Is a directory: " + label})
			continue
		}
		fh, err := os.Open(path)
		if err != nil {
			msgs = append(msgs, message{"error", "Cannot read: " + label})
			continue
		}
		stopped, binary := grepFile(fh, path, re, limit, deadline, &rows)
		fh.Close()
		if binary {
			msgs = append(msgs, message{"info", "Binary file skipped: " + label})
		}
		if stopped {
			cut = true
		}
	}
	if cut {
		msgs = append(msgs, message{"info", fmt.Sprintf("Only the first %d matches are shown.", limit)})
	}
	return grepAnswer(rows, len(rows), cut, msgs), nil
}

func grepAnswer(rows []row, total int, cut bool, msgs []message) map[string]any {
	if rows == nil {
		rows = []row{}
	}
	if msgs == nil {
		msgs = []message{}
	}
	return map[string]any{"rows": rows, "total": total, "cut": cut, "messages": msgs}
}

func shortErr(err error) string {
	s := err.Error()
	s = strings.TrimPrefix(s, "error parsing regexp: ")
	return s
}

// compilePattern applies the case rule: smart case ignores case when the pattern has no
// uppercase letter.
func compilePattern(p, rule string) (*regexp.Regexp, error) {
	ignore := false
	switch rule {
	case "ignore":
		ignore = true
	case "match":
	default:
		ignore = true
		for _, r := range p {
			if unicode.IsUpper(r) {
				ignore = false
				break
			}
		}
	}
	if ignore {
		p = "(?i)" + p
	}
	return regexp.Compile(p)
}

// grepFile appends the matching lines of one file to rows. It reports whether the search must
// stop (the limit or the time) and whether the file looked binary.
func grepFile(fh *os.File, path string, re *regexp.Regexp, limit int, deadline time.Time, rows *[]row) (stopped, binary bool) {
	head := make([]byte, binaryProbe)
	n, _ := io.ReadFull(fh, head)
	if bytes.IndexByte(head[:n], 0) >= 0 {
		return false, true
	}
	if _, err := fh.Seek(0, io.SeekStart); err != nil {
		return false, false
	}
	r := bufio.NewReaderSize(fh, 64*1024)
	label := filepath.Base(path)
	for lineNo := 1; ; lineNo++ {
		line, ok := readLine(r)
		if !ok {
			return false, false
		}
		if lineNo%checkEvery == 0 && time.Now().After(deadline) {
			return true, false
		}
		text := strings.ToValidUTF8(line, "�")
		locs := re.FindAllStringIndex(text, -1)
		if len(locs) == 0 {
			continue
		}
		title, marks := trimLine(text, locs)
		*rows = append(*rows, row{
			Key:      path + "\x00" + fmt.Sprint(lineNo),
			Title:    title,
			Subtitle: fmt.Sprintf("%s:%d", label, lineNo),
			Marks:    marks,
		})
		if len(*rows) >= limit {
			return true, false
		}
	}
}

// readLine reads one line, cutting it at maxLineBytes. ok is false at the end of the file.
func readLine(r *bufio.Reader) (string, bool) {
	var buf []byte
	got := false
	for {
		part, prefix, err := r.ReadLine()
		if err != nil {
			return string(buf), got && len(buf) > 0
		}
		got = true
		if len(buf) < maxLineBytes {
			buf = append(buf, part...)
		}
		if !prefix {
			if len(buf) > maxLineBytes {
				buf = buf[:maxLineBytes]
			}
			return string(buf), true
		}
	}
}

// trimLine removes leading blanks, cuts the line to titleRunes and shifts the marks (rune
// offsets, end exclusive) to match. Empty matches are not marked.
func trimLine(text string, locs [][]int) (string, [][2]int) {
	trimmed := strings.TrimLeft(text, " \t")
	shift := len(text) - len(trimmed)
	var marks [][2]int
	for _, l := range locs {
		s, e := l[0]-shift, l[1]-shift
		if e <= s || s < 0 {
			continue
		}
		rs := utf8.RuneCountInString(trimmed[:s])
		re := rs + utf8.RuneCountInString(trimmed[s:e])
		if rs >= titleRunes {
			continue
		}
		if re > titleRunes {
			re = titleRunes
		}
		marks = append(marks, [2]int{rs, re})
	}
	runes := []rune(trimmed)
	if len(runes) > titleRunes {
		trimmed = string(runes[:titleRunes])
	}
	return trimmed, marks
}
