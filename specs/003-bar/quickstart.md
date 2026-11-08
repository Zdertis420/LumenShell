# Quickstart: Validating the Bar

Run these once the feature is implemented. Contracts: [module-manifest-1.1.md](contracts/module-manifest-1.1.md), [module-api-1.1.md](contracts/module-api-1.1.md), [bar-config.md](contracts/bar-config.md), [bar-slot.md](contracts/bar-slot.md), [cli-ipc-bar.md](contracts/cli-ipc-bar.md), [helper-bar.md](contracts/helper-bar.md).

## Prerequisites

- Modules validated (`specs/002-modules/quickstart.md`), a niri session, Go, Quickshell 0.3+, Qt 6 tools (including `qmltestrunner`), `make`
- A scratch user modules directory and a scratch config: `export LUMEN_USER_MODULES_DIR=$(mktemp -d)` and `export LUMEN_CONFIG_FILE=$(mktemp -d)/config.json`
- Optional: EasyEffects or `pavucontrol` for step 7, and `notify-send` (libnotify) with any notification daemon for step 3

## 0. Spikes (before building on them)

Record the results in `research.md` (R8a, R9a, R11a, R14a): panel space reservation on both edges, how a click outside a popup is seen on niri, the scale of the battery percentage (also on a machine without a battery), and whether exclusive keyboard focus on the panel hands focus back to the previous window. Each has a fallback in the research notes.

## 1. Build and gates

```text
make build
make fmt lint test
```

Expected: everything passes with zero warnings, including the new fitting test (`qmltestrunner scripts/tests/bar_layout_test.qml`) and every Modules fixture (SC-007).

## 2. The default bar (US1, SC-001, SC-012)

```text
scripts/bin/lumen start
scripts/bin/lumen modules list
```

Expected: a bar on every screen within 1 s, with the clock in the center and sound and battery on the right (no battery item on a desktop machine, and no error); `bar`, `bar-clock`, `bar-sound`, `bar-battery` are listed as `bundled`, `loaded`. A maximized window stops at the bar's edge. Unplug or disable a screen: its bar goes within 1 s and the other bar is untouched; plug it back in: the bar returns.

## 3. Arrange the bar in the file (US2, SC-004, SC-006, SC-009)

Edit `$LUMEN_CONFIG_FILE` while the shell runs, one step at a time, and watch the bar:

1. `{"modules":{"bar":{"edge":"bottom"}}}`: the bar moves to the bottom within 1 s.
2. Add `"left": ["bar-clock"], "center": []`: the clock moves to the left.
3. Add `"right": ["bar-sound", "bar-sound/sound", "nope", 42]`: one sound item; a warning for the duplicate and one for `nope`; the entry `42` is dropped on its own: the log shows `right[3]: 42 is not a string; entry dropped` and one desktop notification appears (with `notify-send` installed; without it only the log line); the other entries keep working. Then set `"right": "bar-sound"` (not a list): the whole `right` key falls back to its default and the error names `modules.bar.right`.
4. Set `"screens": ["DP-9"]` (an output that does not exist): no bar, no error. Set `[]`: no bar.
5. Remove every `modules.bar` key: the file holds no bar override, and the default bar returns.

Expected: `lumen modules show bar` lists the warnings with the place and entry; the file only ever holds the keys you wrote, and unknown keys stay.

## 4. Add a third-party item (US3, SC-003, SC-004)

```text
cp -r examples/hello-bar-item "$LUMEN_USER_MODULES_DIR/"
scripts/bin/lumen modules rescan
scripts/bin/lumen modules show hello-bar-item     # contributes to bar/items; permission contribute:bar; state: waiting (not enabled)
scripts/bin/lumen modules enable hello-bar-item   # loaded, item state: available (no list names it)
```

Then add `"left": ["hello-bar-item"]` to `modules.bar`. Expected: the item appears in the left region within 1 s, follows the theme, and its setting changes live. `lumen modules disable hello-bar-item` removes it within 1 s with nothing left behind. Name the item while the module is **not** enabled: it is not shown, the module stays disabled, and the warning says to run `lumen modules enable hello-bar-item`; enabling it makes the item appear without touching the lists (FR-012a).

## 5. Theme and keyboard (US4, SC-002, SC-008)

- `lumen theme tokyo-night`, `lumen theme catppuccin-latte`, a wallpaper palette in light and dark, and the fallback palette: every bar and item color follows within 1 s with no flicker.
- Bind `lumen modules call bar focus` to a niri key. Press it: focus enters the bar; Left and Right move between interactive items in list order; Enter, arrow keys and `M` work on the sound item; Escape returns focus.

## 6. Broken items (US5, SC-005, SC-006)

Copy each contributor fixture from `scripts/tests/fixtures/modules/` into `$LUMEN_USER_MODULES_DIR` (bad manifest, target module missing, visual error, runtime error storm, failing helper, hanging helper), enable them, and name their items in the lists.

Expected: the bar, all good items and `lumen version` keep working; each failure is listed with its module id and reason; the error-storm item is unloaded after three errors within 5 s and its place is released; `lumen modules reload <id>` brings a fixed one back. Disable the `bar` module: contributors stay loaded and show `waiting`; enable it: the items return.

## 7. The sound item (FR-016a, FR-016b)

- Scroll over the item: volume changes in 5% steps and stops at 0% and 100%. Right click and `M`: mute toggles. Left click and Enter: EasyEffects (or `pavucontrol`) opens.
- Uninstall or hide both mixers (or set `modules.bar-sound.mixer` to a missing program): left click logs `no mixer found`, and level, volume and mute still work.
- Stop the sound server: the item is hidden, and every other item works.

## 8. Popups (US6, SC-014)

Enable the example item and activate it: the popup appears next to the item within 1 s, never covers the item or leaves the screen, follows a theme change while open, and closes on Escape, on a click outside, and when another popup opens. Disable the module while it is open: the popup closes.

## 9. Overflow (FR-015a, SC-006)

Add many items to the right list (copy the example under several ids) or shrink the screen: items are dropped from the end of the right list first, then the center, then the left; none overlap; `lumen modules show bar` names the dropped ones; they return when there is room.

## 10. Resource checks (SC-010, SC-011)

- `scripts/tests/smoke.sh --bar-cycles 100` loads and unloads a contributor 100 times, reloads the bar 100 times and opens and closes a popup 100 times: memory within 10% of the use after the first cycle (12% for the bar reload, see SC-010), no duplicated or leftover item.
- With the bar and the three built-in items on one screen, idle 5 minutes: average CPU under 1%.

## 11. Scripted run

```text
make smoke
```

Runs the Foundation and Modules checks plus the bar checks (steps 2 to 4, 6 and 9) against a real shell with a temporary config and module directory.
