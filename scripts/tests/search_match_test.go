package tests

import (
	"reflect"
	"testing"
)

// Name matching (spec FR-006), tested through the commands operation, which matches names and
// nothing else: every word must appear, in any order, ignoring case and Latin accents.

func matchTitles(t *testing.T, names []string, query string) []string {
	t.Helper()
	e := newSearchEnv(t)
	for _, n := range names {
		p := e.file("cmds/"+n, "#!/bin/sh\n")
		_ = chmodExec(p)
	}
	d := e.run(map[string]any{"op": "commands", "query": query, "limit": 50, "dirs": []string{e.home + "/cmds"}})
	return titles(d.Rows)
}

func TestMatchAllWordsAnyOrder(t *testing.T) {
	got := matchTitles(t, []string{"final_report.pdf", "report.txt", "other"}, "report final")
	if !reflect.DeepEqual(got, []string{"final_report.pdf"}) {
		t.Fatalf("got %v", got)
	}
}

func TestMatchIgnoresCaseAndAccents(t *testing.T) {
	got := matchTitles(t, []string{"Café", "cafe-au-lait", "tea"}, "CAFÉ")
	if len(got) != 2 {
		t.Fatalf("got %v, want the two cafe names", got)
	}
}

func TestMatchCyrillicCase(t *testing.T) {
	got := matchTitles(t, []string{"Отчёт", "отчет", "план"}, "ОТЧЁТ")
	if !reflect.DeepEqual(got, []string{"Отчёт"}) {
		t.Fatalf("ё must stay a separate letter from е; got %v", got)
	}
}

func TestMatchRanking(t *testing.T) {
	// Prefix matches first, then contained ones; shorter before longer; then alphabetical.
	got := matchTitles(t, []string{"xreportx", "report-long-name", "report", "areport"}, "report")
	want := []string{"report", "report-long-name", "areport", "xreportx"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMatchEmptyQueryIsAlphabetical(t *testing.T) {
	got := matchTitles(t, []string{"b", "a", "ccc"}, "")
	if !reflect.DeepEqual(got, []string{"a", "b", "ccc"}) {
		t.Fatalf("got %v", got)
	}
}
