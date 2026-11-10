package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Applications (spec FR-008 to FR-011, FR-010a): visibility, precedence, matching, the cache
// and the launch.

func appsQuery(e *searchEnv, q string) []string {
	return titles(e.run(map[string]any{"op": "apps", "query": q, "limit": 50}).Rows)
}

func TestAppsVisibility(t *testing.T) {
	e := newSearchEnv(t)
	app := func(where, id, name string, extra ...string) {
		e.desktop(where, id, append([]string{"Type=Application", "Name=" + name, "Exec=" + name}, extra...)...)
	}
	app("sys", "shown.desktop", "Shown")
	app("sys", "hidden.desktop", "Hidden One", "Hidden=true")
	app("sys", "nodisplay.desktop", "NoDisplay One", "NoDisplay=true")
	app("sys", "onlykde.desktop", "Only KDE", "OnlyShowIn=KDE;")
	app("sys", "onlyniri.desktop", "Only Niri", "OnlyShowIn=GNOME;niri;")
	app("sys", "notniri.desktop", "Not Niri", "NotShowIn=niri;")
	app("sys", "tryexec.desktop", "Missing Binary", "TryExec=/no/such/program")
	e.file(filepath.Join(e.sys, "applications", "link.desktop"), "[Desktop Entry]\nType=Link\nName=A Link\nURL=x\n")
	e.file(filepath.Join(e.sys, "applications", "notes.txt"), "Name=Not an entry\n")
	e.file(filepath.Join(e.sys, "applications", "broken.desktop"), "this is not a desktop entry\n")
	got := appsQuery(e, "")
	want := []string{"Only Niri", "Shown"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAppsUserEntryWinsAndHiddenUserEntryHides(t *testing.T) {
	e := newSearchEnv(t)
	e.desktop("sys", "fox.desktop", "Type=Application", "Name=Fox System", "Exec=fox")
	e.desktop("user", "fox.desktop", "Type=Application", "Name=Fox User", "Exec=fox")
	e.desktop("sys", "gone.desktop", "Type=Application", "Name=Gone", "Exec=gone")
	e.desktop("user", "gone.desktop", "Type=Application", "Name=Gone", "Exec=gone", "Hidden=true")
	got := appsQuery(e, "fox")
	if !reflect.DeepEqual(got, []string{"Fox User"}) {
		t.Fatalf("got %v", got)
	}
	if got := appsQuery(e, "gone"); len(got) != 0 {
		t.Fatalf("a hidden user entry must hide the system one, got %v", got)
	}
}

func TestAppsMatchNameGenericAndKeywords(t *testing.T) {
	e := newSearchEnv(t)
	e.desktop("sys", "a.desktop", "Type=Application", "Name=Waterfox", "GenericName=Web Browser", "Keywords=internet;www;", "Exec=w", "Comment=Browse the web")
	if got := appsQuery(e, "browser"); len(got) != 1 {
		t.Errorf("generic name: %v", got)
	}
	if got := appsQuery(e, "internet"); len(got) != 1 {
		t.Errorf("keyword: %v", got)
	}
	if got := appsQuery(e, "fox water"); len(got) != 1 {
		t.Errorf("words in any order: %v", got)
	}
	if got := appsQuery(e, "water nothere"); len(got) != 0 {
		t.Errorf("each word must be found: %v", got)
	}
	d := e.run(map[string]any{"op": "apps", "query": "water", "limit": 5})
	if d.Rows[0].Subtitle != "Browse the web" || d.Rows[0].Key != "a.desktop" {
		t.Errorf("row = %+v", d.Rows[0])
	}
}

func TestAppsLocalizedName(t *testing.T) {
	e := newSearchEnv(t)
	e.desktop("sys", "a.desktop", "Type=Application", "Name=Files", "Name[ru]=Файлы", "Exec=f")
	e.set("LANG=ru_RU.UTF-8", "LC_MESSAGES=")
	if got := appsQuery(e, ""); !reflect.DeepEqual(got, []string{"Файлы"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAppsNestedIdAndLimit(t *testing.T) {
	e := newSearchEnv(t)
	e.desktop("sys", "kde/dolphin.desktop", "Type=Application", "Name=Dolphin", "Exec=d")
	for i := 0; i < 5; i++ {
		e.desktop("sys", "x"+string(rune('a'+i))+".desktop", "Type=Application", "Name=Extra "+string(rune('a'+i)), "Exec=x")
	}
	d := e.run(map[string]any{"op": "apps", "query": "", "limit": 3})
	if len(d.Rows) != 3 || d.Total != 6 || !d.Cut {
		t.Fatalf("limit: rows=%d total=%d cut=%v", len(d.Rows), d.Total, d.Cut)
	}
	d = e.run(map[string]any{"op": "apps", "query": "dolphin", "limit": 3})
	if d.Rows[0].Key != "kde-dolphin.desktop" {
		t.Fatalf("id = %q", d.Rows[0].Key)
	}
}

func TestAppsCacheFollowsChanges(t *testing.T) {
	e := newSearchEnv(t)
	e.desktop("sys", "a.desktop", "Type=Application", "Name=Alpha", "Exec=a")
	if got := appsQuery(e, ""); !reflect.DeepEqual(got, []string{"Alpha"}) {
		t.Fatalf("first: %v", got)
	}
	if _, err := os.Stat(filepath.Join(e.cache, "lumen", "prism-apps.json")); err != nil {
		t.Fatalf("no cache written: %v", err)
	}
	e.desktop("sys", "b.desktop", "Type=Application", "Name=Beta", "Exec=b")
	if got := appsQuery(e, ""); !reflect.DeepEqual(got, []string{"Alpha", "Beta"}) {
		t.Fatalf("after adding: %v", got)
	}
	if err := os.Remove(filepath.Join(e.sys, "applications", "a.desktop")); err != nil {
		t.Fatal(err)
	}
	if got := appsQuery(e, ""); !reflect.DeepEqual(got, []string{"Beta"}) {
		t.Fatalf("after removing: %v", got)
	}
}

func TestAppsUnreadableCacheIsIgnored(t *testing.T) {
	e := newSearchEnv(t)
	e.desktop("sys", "a.desktop", "Type=Application", "Name=Alpha", "Exec=a")
	e.file(filepath.Join(e.cache, "lumen", "prism-apps.json"), "{not json")
	if got := appsQuery(e, ""); !reflect.DeepEqual(got, []string{"Alpha"}) {
		t.Fatalf("got %v", got)
	}
}

func TestLaunchFieldCodesAndPath(t *testing.T) {
	e := newSearchEnv(t)
	rec := e.fake("myapp")
	e.desktop("sys", "a.desktop", "Type=Application", "Name=My App", "Icon=myicon", "Exec=myapp --flag %f %U %i --pct=100%% --name %c", "Path="+e.home)
	d := e.run(map[string]any{"op": "launch", "id": "a.desktop"})
	if d.Started != "My App" {
		t.Fatalf("started = %q", d.Started)
	}
	got := e.args(rec)
	want := []string{"--flag", "--icon", "myicon", "--pct=100%", "--name", "My App"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLaunchTerminalEntry(t *testing.T) {
	e := newSearchEnv(t)
	rec := e.fake("foot")
	e.desktop("sys", "t.desktop", "Type=Application", "Name=Top", "Exec=htop -d 5", "Terminal=true")
	e.set("TERMINAL=foot")
	e.run(map[string]any{"op": "launch", "id": "t.desktop"})
	if got := e.args(rec); !reflect.DeepEqual(got, []string{"-e", "htop", "-d", "5"}) {
		t.Fatalf("got %v", got)
	}
	// Without a terminal the answer is an error that names the setting and the variable.
	e2 := newSearchEnv(t)
	e2.desktop("sys", "t.desktop", "Type=Application", "Name=Top", "Exec=htop", "Terminal=true")
	code, msg := e2.fail(map[string]any{"op": "launch", "id": "t.desktop"})
	if code != "no_terminal" || !contains(msg, "TERMINAL") || !contains(msg, "modules.prism-search.terminal") {
		t.Fatalf("code=%q msg=%q", code, msg)
	}
}

func TestLaunchMissingAndBroken(t *testing.T) {
	e := newSearchEnv(t)
	if code, _ := e.fail(map[string]any{"op": "launch", "id": "none.desktop"}); code != "not_found" {
		t.Fatalf("code = %q", code)
	}
	e.desktop("sys", "bad.desktop", "Type=Application", "Name=Bad App", "Exec=/no/such/program-xyz")
	code, msg := e.fail(map[string]any{"op": "launch", "id": "bad.desktop"})
	if code != "start_failed" || !contains(msg, "Bad App") {
		t.Fatalf("code=%q msg=%q", code, msg)
	}
}
