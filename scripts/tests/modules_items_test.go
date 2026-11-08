package tests

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Black-box tests of `modules items`: resolving the user's lists of item names against the
// available contributions. Contract: specs/003-bar/contracts/helper-bar.md and bar-config.md.

type itemRef struct {
	Name   string `json:"name"`
	Key    string `json:"key"`
	Module string `json:"module"`
	Item   string `json:"item"`
}

type itemWarning struct {
	Place   string `json:"place"`
	Index   int    `json:"index"`
	Entry   any    `json:"entry"`
	Problem string `json:"problem"`
}

type itemAvail struct {
	Key   string `json:"key"`
	Short string `json:"short"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type itemsResult struct {
	Left      []itemRef     `json:"left"`
	Center    []itemRef     `json:"center"`
	Right     []itemRef     `json:"right"`
	Warnings  []itemWarning `json:"warnings"`
	Available []itemAvail   `json:"available"`
}

func contribution(module, item string, short bool, state string, enabled bool, reason string) map[string]any {
	c := map[string]any{
		"key": module + "/" + item, "module": module, "item": item, "name": item,
		"moduleState": state, "enabled": enabled, "reason": reason,
	}
	if short {
		c["short"] = module
	} else {
		c["short"] = ""
	}
	return c
}

func baseContribs() []any {
	return []any{
		contribution("bar-clock", "clock", true, "loaded", true, ""),
		contribution("bar-sound", "sound", true, "loaded", true, ""),
		contribution("bar-battery", "battery", true, "loaded", true, ""),
		contribution("two-items", "a", true, "loaded", true, ""),
		contribution("two-items", "b", false, "loaded", true, ""),
		contribution("off-mod", "x", true, "disabled", false, ""),
		contribution("bad-mod", "y", true, "failed", true, "Item.qml:3: syntax error"),
	}
}

func runItems(t *testing.T, left, center, right []any, contribs []any) itemsResult {
	t.Helper()
	e, r := runModules(t, "items", map[string]any{
		"point":         map[string]any{"module": "bar", "point": "items"},
		"lists":         map[string]any{"left": left, "center": center, "right": right},
		"contributions": contribs,
	})
	if !e.OK {
		t.Fatalf("items failed: %+v\nstderr=%s", e.Error, r.stderr)
	}
	var out itemsResult
	if err := json.Unmarshal(e.Data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func keysOf(list []itemRef) []string {
	out := []string{}
	for _, x := range list {
		out = append(out, x.Key)
	}
	return out
}

func warningFor(res itemsResult, place string, index int) *itemWarning {
	for i := range res.Warnings {
		if res.Warnings[i].Place == place && res.Warnings[i].Index == index {
			return &res.Warnings[i]
		}
	}
	return nil
}

func TestItemsDocumentedExample(t *testing.T) {
	contribs := []any{
		contribution("bar-clock", "clock", true, "loaded", true, ""),
		contribution("bar-sound", "sound", true, "loaded", true, ""),
	}
	res := runItems(t, []any{}, []any{"bar-clock"}, []any{"bar-sound", "nope", "bar-sound/sound"}, contribs)
	if len(res.Left) != 0 {
		t.Errorf("left = %+v", res.Left)
	}
	if len(res.Center) != 1 || res.Center[0] != (itemRef{"bar-clock", "bar-clock/clock", "bar-clock", "clock"}) {
		t.Errorf("center = %+v", res.Center)
	}
	if len(res.Right) != 1 || res.Right[0] != (itemRef{"bar-sound", "bar-sound/sound", "bar-sound", "sound"}) {
		t.Errorf("right = %+v", res.Right)
	}
	if w := warningFor(res, "right", 1); w == nil || w.Entry != "nope" || w.Problem != "no such item" {
		t.Errorf("warning for nope = %+v (all: %+v)", w, res.Warnings)
	}
	if w := warningFor(res, "right", 2); w == nil || w.Problem != "duplicate of right entry 0" {
		t.Errorf("warning for the duplicate = %+v", w)
	}
	if len(res.Warnings) != 2 {
		t.Errorf("warnings = %+v", res.Warnings)
	}
	states := map[string]string{}
	for _, a := range res.Available {
		states[a.Key] = a.State
	}
	if states["bar-clock/clock"] != "placed" || states["bar-sound/sound"] != "placed" {
		t.Errorf("available = %+v", res.Available)
	}
}

func TestItemsShortAndLongNamesAreOneItem(t *testing.T) {
	// Same list, then across lists (left is read before center before right).
	res := runItems(t, []any{"bar-clock"}, []any{"bar-clock/clock"}, []any{"bar-clock"}, baseContribs())
	if !reflect.DeepEqual(keysOf(res.Left), []string{"bar-clock/clock"}) || len(res.Center) != 0 || len(res.Right) != 0 {
		t.Fatalf("left=%v center=%v right=%v", keysOf(res.Left), keysOf(res.Center), keysOf(res.Right))
	}
	if w := warningFor(res, "center", 0); w == nil || w.Problem != "duplicate of left entry 0" {
		t.Errorf("center warning = %+v", w)
	}
	if w := warningFor(res, "right", 0); w == nil || w.Problem != "duplicate of left entry 0" {
		t.Errorf("right warning = %+v", w)
	}

	// Within one list.
	res = runItems(t, []any{"bar-clock/clock", "bar-clock"}, []any{}, []any{}, baseContribs())
	if len(res.Left) != 1 || warningFor(res, "left", 1) == nil {
		t.Fatalf("left = %+v warnings = %+v", res.Left, res.Warnings)
	}
}

func TestItemsShortFormIsOnlyForTheFirstItem(t *testing.T) {
	res := runItems(t, []any{"two-items", "two-items/b"}, []any{}, []any{}, baseContribs())
	if !reflect.DeepEqual(keysOf(res.Left), []string{"two-items/a", "two-items/b"}) {
		t.Fatalf("left = %v (warnings %+v)", keysOf(res.Left), res.Warnings)
	}
	// "two-items/c" does not exist, and the short name of b is empty.
	res = runItems(t, []any{"two-items/c"}, []any{}, []any{}, baseContribs())
	if len(res.Left) != 0 || warningFor(res, "left", 0) == nil || warningFor(res, "left", 0).Problem != "no such item" {
		t.Fatalf("warnings = %+v", res.Warnings)
	}
}

func TestItemsBadEntriesNeverStopOthers(t *testing.T) {
	entries := []any{"bar-clock", "Bad", "a/b/c", "", strings.Repeat("a", 66), 42, nil, "nope", "bar-sound"}
	res := runItems(t, entries, []any{}, []any{}, baseContribs())
	if !reflect.DeepEqual(keysOf(res.Left), []string{"bar-clock/clock", "bar-sound/sound"}) {
		t.Fatalf("good entries must stay: %v", keysOf(res.Left))
	}
	for _, c := range []struct {
		index int
		want  string
	}{
		{1, "not a valid item name"}, {2, "not a valid item name"}, {3, "not a valid item name"},
		{4, "not a valid item name"}, {5, "not a string"}, {6, "not a string"}, {7, "no such item"},
	} {
		w := warningFor(res, "left", c.index)
		if w == nil || w.Problem != c.want {
			t.Errorf("entry %d: want %q, got %+v", c.index, c.want, w)
		}
	}
	if len(res.Warnings) != 7 {
		t.Errorf("warnings = %+v", res.Warnings)
	}
}

func TestItemsLongestNameResolves(t *testing.T) {
	module := strings.Repeat("m", 32)
	item := strings.Repeat("i", 32)
	name := module + "/" + item
	if len(name) != 65 {
		t.Fatal("test setup")
	}
	res := runItems(t, []any{name}, []any{}, []any{}, []any{contribution(module, item, false, "loaded", true, "")})
	if len(res.Left) != 1 || res.Left[0].Key != name || len(res.Warnings) != 0 {
		t.Fatalf("left = %+v warnings = %+v", res.Left, res.Warnings)
	}
}

func TestItemsModuleStates(t *testing.T) {
	res := runItems(t, []any{"off-mod", "bad-mod", "bar-clock"}, []any{}, []any{}, baseContribs())
	if !reflect.DeepEqual(keysOf(res.Left), []string{"bar-clock/clock"}) {
		t.Fatalf("neither a disabled nor a failed module is placed: %v", keysOf(res.Left))
	}
	if w := warningFor(res, "left", 0); w == nil || w.Problem != "module off-mod is not enabled; run: lumen modules enable off-mod" {
		t.Errorf("disabled warning = %+v", w)
	}
	if w := warningFor(res, "left", 1); w == nil || w.Problem != "module bad-mod failed: Item.qml:3: syntax error" {
		t.Errorf("failed warning = %+v", w)
	}

	// A module that is not enabled yet is also reported when the state is not "disabled".
	notEnabled := []any{contribution("later", "z", true, "loaded", false, "")}
	res = runItems(t, []any{"later"}, []any{}, []any{}, notEnabled)
	if len(res.Left) != 0 || warningFor(res, "left", 0) == nil || !strings.Contains(warningFor(res, "left", 0).Problem, "is not enabled") {
		t.Errorf("enabled=false must not be placed: %+v", res)
	}
}

func TestItemsOrderAndEmptyLists(t *testing.T) {
	res := runItems(t, []any{"bar-sound", "bar-clock"}, []any{"bar-battery"}, []any{}, baseContribs())
	if !reflect.DeepEqual(keysOf(res.Left), []string{"bar-sound/sound", "bar-clock/clock"}) {
		t.Errorf("order must be kept: %v", keysOf(res.Left))
	}
	if !reflect.DeepEqual(keysOf(res.Center), []string{"bar-battery/battery"}) || len(res.Right) != 0 {
		t.Errorf("center=%v right=%v", keysOf(res.Center), keysOf(res.Right))
	}

	empty := runItems(t, []any{}, []any{}, []any{}, []any{})
	if len(empty.Left)+len(empty.Center)+len(empty.Right)+len(empty.Warnings)+len(empty.Available) != 0 {
		t.Errorf("empty input must give empty output: %+v", empty)
	}
	// Empty lists are arrays, not null.
	e, _ := runModules(t, "items", map[string]any{
		"point": map[string]any{"module": "bar", "point": "items"},
		"lists": map[string]any{"left": []any{}, "center": []any{}, "right": []any{}}, "contributions": []any{},
	})
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(e.Data, &raw)
	for _, k := range []string{"left", "center", "right", "warnings", "available"} {
		if string(raw[k]) != "[]" {
			t.Errorf("%s = %s, want []", k, raw[k])
		}
	}
}

func TestItemsAvailableStates(t *testing.T) {
	res := runItems(t, []any{"bar-clock"}, []any{}, []any{}, baseContribs())
	states := map[string]string{}
	for _, a := range res.Available {
		states[a.Key] = a.State
	}
	if states["bar-clock/clock"] != "placed" {
		t.Errorf("clock = %q", states["bar-clock/clock"])
	}
	if states["bar-sound/sound"] != "available" || states["two-items/b"] != "available" {
		t.Errorf("loaded but not named must be available: %v", states)
	}
	if states["off-mod/x"] == "placed" || states["off-mod/x"] == "available" || states["bad-mod/y"] == "available" {
		t.Errorf("a disabled or failed module is neither placed nor available: %v", states)
	}
	if len(res.Available) != len(baseContribs()) {
		t.Errorf("every contribution must be listed: %+v", res.Available)
	}
}

func TestItemsIgnoresOtherTargets(t *testing.T) {
	other := contribution("elsewhere", "w", true, "loaded", true, "")
	other["target"] = map[string]any{"module": "launcher", "point": "providers"}
	mine := contribution("here", "w", true, "loaded", true, "")
	mine["target"] = map[string]any{"module": "bar", "point": "items"}
	res := runItems(t, []any{"elsewhere", "here"}, []any{}, []any{}, []any{other, mine})
	if !reflect.DeepEqual(keysOf(res.Left), []string{"here/w"}) {
		t.Fatalf("left = %v", keysOf(res.Left))
	}
	if w := warningFor(res, "left", 0); w == nil || w.Problem != "no such item" {
		t.Errorf("warning = %+v", w)
	}
	for _, a := range res.Available {
		if a.Key == "elsewhere/w" {
			t.Errorf("a contribution for another point must not be listed: %+v", res.Available)
		}
	}
}

func TestItemsTwoModulesWithShortFormsDoNotCollide(t *testing.T) {
	res := runItems(t, []any{"bar-clock", "bar-sound"}, []any{}, []any{}, baseContribs())
	if !reflect.DeepEqual(keysOf(res.Left), []string{"bar-clock/clock", "bar-sound/sound"}) || len(res.Warnings) != 0 {
		t.Fatalf("left = %v warnings = %+v", keysOf(res.Left), res.Warnings)
	}
}

func TestItemsInvalidInput(t *testing.T) {
	many := make([]any, 65)
	for i := range many {
		many[i] = fmt.Sprintf("m%02d", i)
	}
	contribs := make([]any, 513)
	for i := range contribs {
		contribs[i] = contribution(fmt.Sprintf("m%03d", i), "x", true, "loaded", true, "")
	}
	cases := map[string]map[string]any{
		"more than 64 entries":      {"lists": map[string]any{"left": many, "center": []any{}, "right": []any{}}, "contributions": []any{}},
		"more than 512 contribs":    {"lists": map[string]any{"left": []any{}, "center": []any{}, "right": []any{}}, "contributions": contribs},
		"a list that is not a list": {"lists": map[string]any{"left": "bar-clock", "center": []any{}, "right": []any{}}, "contributions": []any{}},
		"no lists":                  {"contributions": []any{}},
		"contributions not a list":  {"lists": map[string]any{"left": []any{}, "center": []any{}, "right": []any{}}, "contributions": "x"},
	}
	for name, in := range cases {
		in["point"] = map[string]any{"module": "bar", "point": "items"}
		e, _ := runModules(t, "items", in)
		if e.OK || e.Error == nil || e.Error.Code != "invalid_input" {
			t.Errorf("%s: want invalid_input, got %+v", name, e)
		}
	}
}
