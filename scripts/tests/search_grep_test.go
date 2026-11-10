package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Grep (spec FR-022 to FR-026, research R12).

func grepRun(e *searchEnv, query string, extra map[string]any) searchData {
	req := map[string]any{"op": "grep", "query": query, "limit": 100, "budgetMs": 5000}
	for k, v := range extra {
		req[k] = v
	}
	return e.run(req)
}

func TestGrepListsLinesWithNumbersAndMarks(t *testing.T) {
	e := newSearchEnv(t)
	a := e.file("a.md", "TODO one\nnothing\n  fix the FIXME here\n")
	b := e.file("b.md", "todo lower\n")
	d := grepRun(e, `"TODO|FIXME" `+a+" "+b, nil)
	var got []string
	for _, r := range d.Rows {
		got = append(got, r.Subtitle+"="+r.Title)
	}
	want := []string{"a.md:1=TODO one", "a.md:3=fix the FIXME here"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !reflect.DeepEqual(d.Rows[0].Marks, [][2]int{{0, 4}}) || !reflect.DeepEqual(d.Rows[1].Marks, [][2]int{{8, 13}}) {
		t.Fatalf("marks = %v %v", d.Rows[0].Marks, d.Rows[1].Marks)
	}
}

func TestGrepSmartCase(t *testing.T) {
	e := newSearchEnv(t)
	f := e.file("a.txt", "Todo\ntodo\nTODO\n")
	if d := grepRun(e, "todo "+f, nil); len(d.Rows) != 3 {
		t.Errorf("no uppercase: want 3, got %d", len(d.Rows))
	}
	if d := grepRun(e, "Todo "+f, nil); len(d.Rows) != 1 {
		t.Errorf("with uppercase: want 1, got %d", len(d.Rows))
	}
	if d := grepRun(e, "todo "+f, map[string]any{"caseRule": "match"}); len(d.Rows) != 1 {
		t.Errorf("match rule: want 1, got %d", len(d.Rows))
	}
	if d := grepRun(e, "Todo "+f, map[string]any{"caseRule": "ignore"}); len(d.Rows) != 3 {
		t.Errorf("ignore rule: want 3, got %d", len(d.Rows))
	}
}

func TestGrepHintsAndInvalidPattern(t *testing.T) {
	e := newSearchEnv(t)
	f := e.file("a.txt", "x\n")
	for _, q := range []string{"", "pattern", "pattern   "} {
		d := grepRun(e, q, nil)
		if len(d.Rows) != 0 || len(d.Messages) != 1 || d.Messages[0].Kind != "info" {
			t.Errorf("%q: %+v", q, d)
		}
	}
	d := grepRun(e, `"(" `+f, nil)
	if len(d.Rows) != 0 || len(d.Messages) != 1 || d.Messages[0].Kind != "error" || !contains(d.Messages[0].Text, "not valid") {
		t.Errorf("invalid pattern: %+v", d)
	}
}

func TestGrepFileProblemsDoNotStopOthers(t *testing.T) {
	e := newSearchEnv(t)
	good := e.file("good.txt", "needle\n")
	bin := e.file("bin.dat", "needle\x00more\n")
	dir := filepath.Join(e.home, "adir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d := grepRun(e, "needle "+filepath.Join(e.home, "missing.txt")+" "+dir+" "+bin+" "+good, nil)
	if len(d.Rows) != 1 || d.Rows[0].Subtitle != "good.txt:1" {
		t.Fatalf("rows = %+v", d.Rows)
	}
	var texts []string
	for _, m := range d.Messages {
		texts = append(texts, m.Text)
	}
	joined := strings.Join(texts, "|")
	for _, want := range []string{"File not found: missing.txt", "Is a directory: adir", "Binary file skipped: bin.dat"} {
		if !contains(joined, want) {
			t.Errorf("missing message %q in %q", want, joined)
		}
	}
}

func TestGrepUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read everything")
	}
	e := newSearchEnv(t)
	f := e.file("secret.txt", "needle\n")
	if err := os.Chmod(f, 0); err != nil {
		t.Fatal(err)
	}
	d := grepRun(e, "needle "+f, nil)
	if len(d.Rows) != 0 || len(d.Messages) != 1 || !contains(d.Messages[0].Text, "Cannot read: secret.txt") {
		t.Fatalf("got %+v", d)
	}
}

func TestGrepPathsFromHome(t *testing.T) {
	e := newSearchEnv(t)
	e.file("notes/a.txt", "needle\n")
	for _, p := range []string{"~/notes/a.txt", "notes/a.txt"} {
		if d := grepRun(e, "needle "+p, nil); len(d.Rows) != 1 {
			t.Errorf("%s: %+v", p, d)
		}
	}
}

func TestGrepStopsAtLimit(t *testing.T) {
	e := newSearchEnv(t)
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("hit " + strconv.Itoa(i) + "\n")
	}
	f := e.file("a.txt", b.String())
	d := grepRun(e, "hit "+f, map[string]any{"limit": 10})
	if len(d.Rows) != 10 || !d.Cut {
		t.Fatalf("rows=%d cut=%v", len(d.Rows), d.Cut)
	}
	last := d.Messages[len(d.Messages)-1].Text
	if !contains(last, "first 10 matches") {
		t.Fatalf("message = %q", last)
	}
}

func TestGrepLongLineAndMultibyteMarks(t *testing.T) {
	e := newSearchEnv(t)
	long := strings.Repeat("x", 70000) + " needle"
	f := e.file("long.txt", long+"\nпривет мир needle\n")
	d := grepRun(e, "needle "+f, nil)
	if len(d.Rows) != 1 {
		t.Fatalf("the long line is cut at 64 KiB before the match; rows = %d", len(d.Rows))
	}
	r := d.Rows[0]
	if r.Title != "привет мир needle" || !reflect.DeepEqual(r.Marks, [][2]int{{11, 17}}) {
		t.Fatalf("row = %+v", r)
	}
}

func TestGrepMarksAfterTrimAndTitleCut(t *testing.T) {
	e := newSearchEnv(t)
	f := e.file("a.txt", "\t\t   lead needle\n"+strings.Repeat("a", 190)+"needle tail\n")
	d := grepRun(e, "needle "+f, nil)
	if len(d.Rows) != 2 {
		t.Fatalf("rows = %d", len(d.Rows))
	}
	if d.Rows[0].Title != "lead needle" || !reflect.DeepEqual(d.Rows[0].Marks, [][2]int{{5, 11}}) {
		t.Errorf("trimmed row = %+v", d.Rows[0])
	}
	if n := len([]rune(d.Rows[1].Title)); n != 200 {
		t.Errorf("title length = %d, want 200", n)
	}
	if !reflect.DeepEqual(d.Rows[1].Marks, [][2]int{{190, 196}}) {
		t.Errorf("marks = %v", d.Rows[1].Marks)
	}
}

func TestGrepMatchesReferenceSearch(t *testing.T) {
	ref, err := exec.LookPath("grep")
	if err != nil {
		t.Skip("grep not installed")
	}
	e := newSearchEnv(t)
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("line " + strconv.Itoa(i) + " alpha beta\n")
		if i%7 == 0 {
			b.WriteString("seven " + strconv.Itoa(i) + " Alpha\n")
		}
	}
	f := e.file("ref.txt", b.String())
	out, _ := exec.Command(ref, "-n", "-E", "alpha|seven", f).Output()
	want := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
	d := grepRun(e, `"alpha|seven" `+f, map[string]any{"limit": 1000})
	if len(d.Rows) != want {
		t.Fatalf("rows = %d, grep says %d", len(d.Rows), want)
	}
}
