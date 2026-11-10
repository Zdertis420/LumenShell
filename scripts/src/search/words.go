package main

import "strings"

// splitWords splits text into words on spaces. A word in single or double quotes is one word;
// inside double quotes a backslash escapes the next character. An unbalanced quote is taken as
// a literal character.
func splitWords(s string) []string {
	var out []string
	for _, sp := range splitSpans(s) {
		out = append(out, sp.Value)
	}
	return out
}

// span is one word: its value (quotes removed) and where it sits in the raw text, as rune
// offsets [Start, End).
type span struct {
	Value      string
	Start, End int
}

// splitSpans is splitWords with the raw position of every word, so a word can be replaced in
// the text (Tab completion).
func splitSpans(s string) []span {
	var out []span
	var cur strings.Builder
	has := false
	start := 0
	rs := []rune(s)
	begin := func(i int) {
		if !has {
			start = i
			has = true
		}
	}
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == ' ' || r == '\t' || r == '\n':
			if has {
				out = append(out, span{cur.String(), start, i})
				cur.Reset()
				has = false
			}
		case r == '\'' || r == '"':
			end := closingQuote(rs, i)
			begin(i)
			if end < 0 {
				cur.WriteRune(r)
				continue
			}
			for j := i + 1; j < end; j++ {
				if r == '"' && rs[j] == '\\' && j+1 < end {
					j++
				}
				cur.WriteRune(rs[j])
			}
			i = end
		default:
			begin(i)
			cur.WriteRune(r)
		}
	}
	if has {
		out = append(out, span{cur.String(), start, len(rs)})
	}
	return out
}

// closingQuote returns the index of the quote that closes the one at rs[start], or -1.
func closingQuote(rs []rune, start int) int {
	q := rs[start]
	for j := start + 1; j < len(rs); j++ {
		if q == '"' && rs[j] == '\\' {
			j++
			continue
		}
		if rs[j] == q {
			return j
		}
	}
	return -1
}

// expandHome turns "~" and "~/x" into paths under the user's home.
func expandHome(p string) string {
	if p == "~" {
		return userHome
	}
	if strings.HasPrefix(p, "~/") {
		return userHome + p[1:]
	}
	return p
}
