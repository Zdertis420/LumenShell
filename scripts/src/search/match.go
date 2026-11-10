package main

import (
	"sort"
	"strings"
	"unicode"
)

// Name matching (spec FR-006, research R8): every word of the query must appear in the name, in
// any order, ignoring case and Latin accents. Ranking uses no history.

var accentPairs = []string{
	"aàáâãäåāăą", "cçćĉċč", "dďđ", "eèéêëēĕėęě", "gĝğġģ", "hĥħ", "iìíîïĩīĭįı", "jĵ",
	"kķ", "lĺļľŀł", "nñńņňŉ", "oòóôõöøōŏő", "rŕŗř", "sśŝşš", "tţťŧ", "uùúûüũūŭůűų",
	"wŵ", "yýÿŷ", "zźżž",
}

var accentMap = func() map[rune]rune {
	m := map[rune]rune{}
	for _, group := range accentPairs {
		rs := []rune(group)
		for _, r := range rs[1:] {
			m[r] = rs[0]
		}
	}
	return m
}()

// fold lowercases and removes Latin accents. Letters of other scripts only change case.
func fold(s string) string {
	var b strings.Builder
	for _, r := range s {
		r = unicode.ToLower(r)
		if base, ok := accentMap[r]; ok {
			r = base
		}
		b.WriteRune(r)
	}
	return b.String()
}

// queryWords splits a query into folded words.
func queryWords(q string) []string {
	return strings.Fields(fold(q))
}

// rankKey orders matches; smaller is better.
type rankKey struct {
	tier   int // 0: every word starts a word of the name; 1: only contained; 2: matched outside the name
	pos    int // position of the first word found, in runes
	length int // length of the name in runes
}

func (a rankKey) less(b rankKey) bool {
	if a.tier != b.tier {
		return a.tier < b.tier
	}
	if a.pos != b.pos {
		return a.pos < b.pos
	}
	return a.length < b.length
}

func isWordChar(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// matchName reports whether the folded name contains every word, and how well.
func matchName(words []string, name string) (bool, rankKey) {
	folded := fold(name)
	key := rankKey{length: len([]rune(folded))}
	if len(words) == 0 {
		return true, rankKey{} // no words: every name is equal, so the list is alphabetical
	}
	allPrefix := true
	first := -1
	for _, w := range words {
		idx := strings.Index(folded, w)
		if idx < 0 {
			return false, key
		}
		rpos := len([]rune(folded[:idx]))
		if first < 0 || rpos < first {
			first = rpos
		}
		if !startsWord(folded, w) {
			allPrefix = false
		}
	}
	key.pos = first
	if !allPrefix {
		key.tier = 1
	}
	return true, key
}

// startsWord reports whether w begins a word of s (at the start, or after a non-letter).
func startsWord(s, w string) bool {
	from := 0
	for {
		i := strings.Index(s[from:], w)
		if i < 0 {
			return false
		}
		i += from
		if i == 0 {
			return true
		}
		prev := []rune(s[:i])
		if !isWordChar(prev[len(prev)-1]) {
			return true
		}
		from = i + 1
		if from >= len(s) {
			return false
		}
	}
}

// matchFields matches words against the name first; a word missing from the name may be found
// in the extra text (generic name, keywords), which ranks lower.
func matchFields(words []string, name, extra string) (bool, rankKey) {
	if ok, key := matchName(words, name); ok {
		return true, key
	}
	combined := fold(name + " " + extra)
	for _, w := range words {
		if !strings.Contains(combined, w) {
			return false, rankKey{}
		}
	}
	return true, rankKey{tier: 2, length: len([]rune(name))}
}

type scored struct {
	r    row
	key  rankKey
	sort string
}

// sortScored orders rows by rank, then by the sort string (name or path).
func sortScored(list []scored) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].key != list[j].key {
			return list[i].key.less(list[j].key)
		}
		return list[i].sort < list[j].sort
	})
}
