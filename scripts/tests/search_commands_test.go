package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// Commands (spec FR-019 to FR-021, research R11).

func exe(e *searchEnv, rel string) {
	p := e.file(rel, "#!/bin/sh\n")
	if err := os.Chmod(p, 0o755); err != nil {
		e.t.Fatal(err)
	}
}

func TestCommandsOnlyExecutablesOncePerName(t *testing.T) {
	e := newSearchEnv(t)
	exe(e, "b1/tool")
	exe(e, "b2/tool") // same name later in the order: not listed twice
	exe(e, "b2/other")
	e.file("b1/notexec", "x")
	if err := os.MkdirAll(filepath.Join(e.home, "b1", "adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/no/such/target", filepath.Join(e.home, "b1", "broken")); err != nil {
		t.Fatal(err)
	}
	d := e.run(map[string]any{"op": "commands", "query": "", "limit": 50, "dirs": []string{e.home + "/b1", e.home + "/b2"}})
	if got := titles(d.Rows); !reflect.DeepEqual(got, []string{"other", "tool"}) {
		t.Fatalf("got %v", got)
	}
	for _, r := range d.Rows {
		if r.Title == "tool" && r.Key != e.home+"/b1/tool" {
			t.Errorf("the first folder must win, got %q", r.Key)
		}
		if r.Subtitle != r.Key {
			t.Errorf("the subtitle must be the full path: %+v", r)
		}
	}
}

func TestCommandsLinkToExecutable(t *testing.T) {
	e := newSearchEnv(t)
	exe(e, "real/prog")
	if err := os.MkdirAll(filepath.Join(e.home, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(e.home+"/real/prog", filepath.Join(e.home, "b", "alias")); err != nil {
		t.Fatal(err)
	}
	d := e.run(map[string]any{"op": "commands", "query": "alias", "limit": 5, "dirs": []string{e.home + "/b"}})
	if len(d.Rows) != 1 {
		t.Fatalf("got %v", titles(d.Rows))
	}
}

func TestCommandsMergedUsrFolderSearchedOnce(t *testing.T) {
	e := newSearchEnv(t)
	exe(e, "usr/bin/tool")
	if err := os.Symlink(e.home+"/usr/bin", filepath.Join(e.home, "bin")); err != nil {
		t.Fatal(err)
	}
	d := e.run(map[string]any{"op": "commands", "query": "tool", "limit": 5, "dirs": []string{e.home + "/usr/bin", e.home + "/bin"}})
	if d.Total != 1 {
		t.Fatalf("total = %d, want 1", d.Total)
	}
}

func TestCommandsExtraFoldersAndMissingFolders(t *testing.T) {
	e := newSearchEnv(t)
	exe(e, "mine/mytool")
	d := e.run(map[string]any{"op": "commands", "query": "mytool", "limit": 5,
		"dirs": []string{e.home + "/does-not-exist"}, "extraDirs": []string{e.home + "/mine", "relative/ignored"}})
	if got := titles(d.Rows); !reflect.DeepEqual(got, []string{"mytool"}) {
		t.Fatalf("got %v", got)
	}
}

func TestCommandsMatchingAndLimit(t *testing.T) {
	e := newSearchEnv(t)
	for _, n := range []string{"ffmpeg", "obs-ffmpeg-mux", "ls", "ffprobe"} {
		exe(e, "b/"+n)
	}
	d := e.run(map[string]any{"op": "commands", "query": "ff", "limit": 2, "dirs": []string{e.home + "/b"}})
	if len(d.Rows) != 2 || d.Total != 3 || !d.Cut {
		t.Fatalf("rows=%d total=%d cut=%v", len(d.Rows), d.Total, d.Cut)
	}
	if d.Rows[0].Title != "ffmpeg" {
		t.Fatalf("prefix matches must come first: %v", titles(d.Rows))
	}
}

func TestCommandsDefaultFoldersIncludeUserLocalBin(t *testing.T) {
	e := newSearchEnv(t)
	exe(e, ".local/bin/my-local-tool")
	d := e.run(map[string]any{"op": "commands", "query": "my-local-tool", "limit": 5})
	if len(d.Rows) != 1 || d.Rows[0].Key != e.home+"/.local/bin/my-local-tool" {
		t.Fatalf("got %+v", d.Rows)
	}
}
