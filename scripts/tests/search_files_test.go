package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Files (spec FR-012 to FR-018): the stages, their order, the limits and the walk rules.

func filesStage(e *searchEnv, stage, name string, extra map[string]any) searchData {
	req := map[string]any{"op": "files", "stage": stage, "name": name, "limit": 50, "budgetMs": 5000}
	for k, v := range extra {
		req[k] = v
	}
	return e.run(req)
}

func rel(e *searchEnv, rows []searchRow) []string {
	var out []string
	for _, r := range rows {
		out = append(out, strings.TrimPrefix(r.Key, e.home+"/"))
	}
	return out
}

func fixtureHome(e *searchEnv) {
	for _, p := range []string{
		"Desktop/report-a.txt", "Downloads/report-b.txt", "Documents/report-c.txt", "Documents/sub/deep/report-d.txt",
		"report-e.txt", "notes/report-f.txt", "projects/x/report-g.txt",
		".config/app/report-h.txt", ".cache/report-i.txt", ".local/share/report-j.txt",
		".secret/report-k.txt", ".hiddenfile-report", "Documents/.hidden-report",
	} {
		e.file(p, "x")
	}
}

func TestFilesPriorityStageOrder(t *testing.T) {
	e := newSearchEnv(t)
	fixtureHome(e)
	d := filesStage(e, "priority", "report", nil)
	want := []string{"Desktop/report-a.txt", "Downloads/report-b.txt", "Documents/report-c.txt", "Documents/sub/deep/report-d.txt"}
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if d.Total != 4 || d.Cut {
		t.Fatalf("total=%d cut=%v", d.Total, d.Cut)
	}
}

func TestFilesHomeStageIsDirectChildrenOnly(t *testing.T) {
	e := newSearchEnv(t)
	fixtureHome(e)
	e.file("reports/inside.txt", "x") // a folder named like the query is a match, its content is not in this stage
	d := filesStage(e, "home", "report", nil)
	got := rel(e, d.Rows)
	want := []string{"reports", "report-e.txt"} // the shorter name ranks first
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFilesHomeDirsStageSkipsPriorityAndHidden(t *testing.T) {
	e := newSearchEnv(t)
	fixtureHome(e)
	d := filesStage(e, "homeDirs", "report", nil)
	want := []string{"notes/report-f.txt", "projects/x/report-g.txt"}
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	d = filesStage(e, "homeDirs", "report", map[string]any{"hiddenHomeDirs": true})
	if got := rel(e, d.Rows); !contains(strings.Join(got, ","), ".secret/report-k.txt") {
		t.Fatalf("hiddenHomeDirs should add .secret: %v", got)
	}
	if got := rel(e, d.Rows); contains(strings.Join(got, ","), ".config") {
		t.Fatalf(".config belongs to the dotDirs stage: %v", got)
	}
}

func TestFilesDotDirsStage(t *testing.T) {
	e := newSearchEnv(t)
	fixtureHome(e)
	d := filesStage(e, "dotDirs", "report", nil)
	want := []string{".cache/report-i.txt", ".local/share/report-j.txt", ".config/app/report-h.txt"}
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFilesHiddenFilesSetting(t *testing.T) {
	e := newSearchEnv(t)
	fixtureHome(e)
	d := filesStage(e, "priority", "hidden", nil)
	if len(d.Rows) != 0 {
		t.Fatalf("hidden files must be skipped by default: %v", rel(e, d.Rows))
	}
	d = filesStage(e, "priority", "hidden", map[string]any{"hidden": true})
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, []string{"Documents/.hidden-report"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilesSystemStage(t *testing.T) {
	e := newSearchEnv(t)
	// The test folders live under /tmp, which is never walked, so the stand-in system root is
	// made next to the developer's home instead.
	root, err := os.MkdirTemp(os.Getenv("HOME"), ".lumen-search-test-")
	if err != nil {
		t.Skipf("cannot create a folder in HOME: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	e.file(filepath.Join(root, "etc", "report.conf"), "x")
	e.file(filepath.Join(root, "usr", "share", "report.dat"), "x")
	d := filesStage(e, "system", "report", map[string]any{"systemRoots": []string{root}, "limit": 1})
	if d.Total != 2 || len(d.Rows) != 1 {
		t.Fatalf("total=%d rows=%d", d.Total, len(d.Rows))
	}
}

func TestFilesPseudoRootsAreRefused(t *testing.T) {
	e := newSearchEnv(t)
	d := filesStage(e, "system", "a", map[string]any{"systemRoots": []string{"/proc", "/sys", "/dev", "/run", "/tmp", "/tmp/sub", "relative"}})
	if d.Total != 0 || len(d.Rows) != 0 {
		t.Fatalf("pseudo roots must not be walked: %+v", d)
	}
}

func TestFilesDirStageSearchesOnlyThatDirectory(t *testing.T) {
	e := newSearchEnv(t)
	fixtureHome(e)
	dir := filepath.Join(e.home, "projects")
	d := filesStage(e, "dir", "report", map[string]any{"dir": dir})
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, []string{"projects/x/report-g.txt"}) {
		t.Fatalf("got %v", got)
	}
	// A named hidden directory is searched, although hidden entries are normally skipped.
	d = filesStage(e, "dir", "report", map[string]any{"dir": filepath.Join(e.home, ".secret")})
	if len(d.Rows) != 1 {
		t.Fatalf("got %v", rel(e, d.Rows))
	}
}

func TestFilesEmptyNameGivesHint(t *testing.T) {
	e := newSearchEnv(t)
	d := filesStage(e, "priority", "  ", nil)
	if len(d.Rows) != 0 || d.Hint == "" {
		t.Fatalf("got %+v", d)
	}
}

func TestFilesSymlinksAreNotFollowed(t *testing.T) {
	e := newSearchEnv(t)
	e.file("outside/report-out.txt", "x")
	e.file("Documents/real/report-in.txt", "x")
	if err := os.Symlink(e.home+"/outside", filepath.Join(e.home, "Documents", "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(e.home+"/Documents", filepath.Join(e.home, "Documents", "loop")); err != nil {
		t.Fatal(err)
	}
	d := filesStage(e, "priority", "report", nil)
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, []string{"Documents/real/report-in.txt"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilesHardLinkCountsOnce(t *testing.T) {
	e := newSearchEnv(t)
	p := e.file("Documents/report-a.txt", "x")
	if err := os.Link(p, filepath.Join(e.home, "Documents", "report-b.txt")); err != nil {
		t.Fatal(err)
	}
	d := filesStage(e, "priority", "report", nil)
	if d.Total != 1 {
		t.Fatalf("total = %d, want 1", d.Total)
	}
}

func TestFilesUnreadableFolderIsSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read everything")
	}
	e := newSearchEnv(t)
	e.file("Documents/open/report-ok.txt", "x")
	locked := filepath.Join(e.home, "Documents", "locked")
	e.file("Documents/locked/report-no.txt", "x")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	d := filesStage(e, "priority", "report", nil)
	if got := rel(e, d.Rows); !reflect.DeepEqual(got, []string{"Documents/open/report-ok.txt"}) {
		t.Fatalf("got %v", got)
	}
}

func TestFilesLimitTotalAndCuts(t *testing.T) {
	e := newSearchEnv(t)
	for i := 0; i < 30; i++ {
		e.file("Documents/report-"+string(rune('a'+i%26))+string(rune('a'+i/26))+".txt", "x")
	}
	d := filesStage(e, "priority", "report", map[string]any{"limit": 5})
	if len(d.Rows) != 5 || d.Total != 30 || d.Cut {
		t.Fatalf("rows=%d total=%d cut=%v", len(d.Rows), d.Total, d.Cut)
	}
	d = filesStage(e, "priority", "report", map[string]any{"limit": 5, "maxFiles": 10})
	if !d.Cut || d.Total > 10 {
		t.Fatalf("maxFiles must cut: total=%d cut=%v", d.Total, d.Cut)
	}
}

func TestFilesUnusualNames(t *testing.T) {
	e := newSearchEnv(t)
	e.file("Documents/my report final.txt", "x")
	e.file("Documents/line\nbreak report.txt", "x")
	long := strings.Repeat("r", 249) + "report" // 255 bytes, the longest name a file system allows
	e.file("Documents/"+long, "x")
	d := filesStage(e, "priority", "report final", nil)
	if len(d.Rows) != 1 || d.Rows[0].Title != "my report final.txt" {
		t.Fatalf("got %v", titles(d.Rows))
	}
	d = filesStage(e, "priority", "report", nil)
	if d.Total != 3 {
		t.Fatalf("total = %d, want 3 (spaces, a newline and a 256-byte name)", d.Total)
	}
}

func TestFilesRowShape(t *testing.T) {
	e := newSearchEnv(t)
	e.file("Documents/report.txt", "x")
	e.file("Documents/reports/readme", "x")
	d := filesStage(e, "priority", "report", nil)
	for _, r := range d.Rows {
		switch r.Title {
		case "report.txt":
			if r.Subtitle != "~/Documents" || r.Icon != "text-x-generic" {
				t.Errorf("file row = %+v", r)
			}
		case "reports":
			if r.Icon != "folder" {
				t.Errorf("directory row = %+v", r)
			}
		default:
			t.Errorf("unexpected row %+v", r)
		}
	}
}
