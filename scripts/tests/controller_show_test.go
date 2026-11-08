package tests

import (
	"strings"
	"testing"
)

// `lumen modules list` and `show` print the interface 1.1 information (extension points,
// contributions, commands, dropped items, item warnings). Contract:
// specs/003-bar/contracts/cli-ipc-bar.md.

const barDetailJSON = `{"id":"bar","name":"Bar","version":"1.0.0","origin":"bundled","state":"loaded","enabled":true,"defaultEnabled":true,` +
	`"permissions":["helpers"],"runtimeErrors":0,"lastError":"","warnings":[],` +
	`"description":"A bar","dir":"/x/bar","apiRequired":"1.1","apiProvided":"1.1","errors":[],` +
	`"settings":{},"settingValues":{},"settingErrors":[],"helpers":["notify"],` +
	`"commands":["focus"],` +
	`"extensionPoints":[{"id":"items","description":"Items shown in the bar","contributions":[` +
	`{"key":"bar-clock/clock","short":"bar-clock","name":"Clock","state":"shown","reason":""},` +
	`{"key":"hello/greeting","short":"hello","name":"Greeting","state":"waiting","reason":"module hello is not enabled"},` +
	`{"key":"x/y","short":"","name":"Y","state":"rejected","reason":"bar item limit (32) reached"}]}],` +
	`"contributions":[],` +
	`"dropped":[{"screen":"DP-1","items":["bar-battery","bar-sound"]}],` +
	`"itemWarnings":[{"place":"right","index":2,"entry":"weather","problem":"no such item"},{"place":"left","index":0,"entry":"hello","problem":"module hello is not enabled; run: lumen modules enable hello"}]}`

const itemDetailJSON = `{"id":"bar-clock","name":"Clock","version":"1.0.0","origin":"bundled","state":"loaded","enabled":true,"defaultEnabled":true,` +
	`"permissions":["contribute:bar"],"runtimeErrors":0,"lastError":"","warnings":[],` +
	`"description":"Clock","dir":"/x/bar-clock","apiRequired":"1.1","apiProvided":"1.1","errors":[],` +
	`"settings":{},"settingValues":{},"settingErrors":[],"helpers":[],` +
	`"commands":[],"extensionPoints":[],` +
	`"contributions":[{"key":"bar-clock/clock","short":"bar-clock","name":"Clock","target":{"module":"bar","point":"items"},"state":"shown","reason":""}],` +
	`"dropped":[],"itemWarnings":[]}`

const listWithNotesJSON = `{"attribution":"active","modules":[
 {"id":"bar","name":"Bar","version":"1.0.0","origin":"bundled","state":"loaded","enabled":true,"defaultEnabled":true,"permissions":["helpers"],"runtimeErrors":0,"lastError":"","warnings":[],"notes":["hosts: items"]},
 {"id":"bar-clock","name":"Clock","version":"1.0.0","origin":"bundled","state":"loaded","enabled":true,"defaultEnabled":true,"permissions":["contribute:bar"],"runtimeErrors":0,"lastError":"","warnings":[],"notes":["contributes to bar"]},
 {"id":"hello-bar-item","name":"Hello","version":"1.0.0","origin":"user","state":"disabled","enabled":false,"defaultEnabled":false,"permissions":["contribute:bar"],"runtimeErrors":0,"lastError":"","warnings":[],"notes":["contributes to bar"]}
]}`

func TestShowPrintsExtensionPointsCommandsDroppedAndWarnings(t *testing.T) {
	_, env := modulesEnv(t, map[string]string{"bar": barDetailJSON})
	r, _ := lumen(env, "modules", "show", "bar")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	for _, want := range []string{
		"Extension points", "items", "Items shown in the bar", "contributions: 3",
		"bar-clock/clock", "bar-clock", "shown",
		"hello/greeting", "waiting", "module hello is not enabled",
		"x/y", "rejected", "bar item limit (32) reached",
		"Commands", "focus",
		"Dropped items (screen DP-1)", "bar-battery", "bar-sound",
		"right[2]", "weather", "no such item",
		"left[0]", "run: lumen modules enable hello",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("show output lacks %q:\n%s", want, r.stdout)
		}
	}
}

func TestShowPrintsTheItemsOfAContributor(t *testing.T) {
	_, env := modulesEnv(t, map[string]string{"bar-clock": itemDetailJSON})
	r, _ := lumen(env, "modules", "show", "bar-clock")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	for _, want := range []string{"Contributions", "ITEM NAME", "SHORT NAME", "TARGET", "STATE", "bar-clock/clock", "bar/items", "shown"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("show output lacks %q:\n%s", want, r.stdout)
		}
	}
	// A module without extension points or commands prints neither section.
	for _, not := range []string{"Extension points", "Commands", "Dropped items"} {
		if strings.Contains(r.stdout, not) {
			t.Errorf("show output has %q for a module without it:\n%s", not, r.stdout)
		}
	}
}

func TestShowOfOlderRepliesStillPrintsAsBefore(t *testing.T) {
	// detailJSON has none of the new fields (as sent by an older shell).
	_, env := modulesEnv(t, map[string]string{"hello": detailJSON("hello", "disabled", "")})
	r, _ := lumen(env, "modules", "show", "hello")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	for _, not := range []string{"Extension points", "Contributions", "Commands", "Dropped items"} {
		if strings.Contains(r.stdout, not) {
			t.Errorf("an old reply must not print %q:\n%s", not, r.stdout)
		}
	}
	if !strings.Contains(r.stdout, "runtime errors: 1") {
		t.Errorf("the usual output is missing:\n%s", r.stdout)
	}
}

func TestListPrintsContributionNotes(t *testing.T) {
	_, env := modulesEnv(t, nil, "FAKE_QS_MODULES_LIST="+strings.ReplaceAll(listWithNotesJSON, "\n", ""))
	r, _ := lumen(env, "modules", "list")
	if r.code != 0 {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
	want := map[string][]string{
		"bar":            {"loaded", "hosts: items"},
		"bar-clock":      {"loaded", "contributes to bar"},
		"hello-bar-item": {"disabled", "permissions: contribute:bar", "contributes to bar"},
	}
	for id, parts := range want {
		found := false
		for _, l := range strings.Split(r.stdout, "\n") {
			if strings.HasPrefix(l, id+" ") {
				found = true
				for _, p := range parts {
					if !strings.Contains(l, p) {
						t.Errorf("line for %s lacks %q: %q", id, p, l)
					}
				}
			}
		}
		if !found {
			t.Errorf("no line for %s:\n%s", id, r.stdout)
		}
	}
}
