# Quickstart: validating the Prism window

Prerequisites: the repository built (`make build`), Quickshell 0.3.2 on niri, a config file at `$XDG_CONFIG_HOME/lumen/config.json` (it may be empty), and a niri binding:

```kdl
Mod+Space { spawn "lumen" "modules" "call" "prism" "toggle"; }
```

Contracts: [prism-config.md](contracts/prism-config.md), [prism-surface.md](contracts/prism-surface.md), [prism-source-slot.md](contracts/prism-source-slot.md), [module-manifest-1.3.md](contracts/module-manifest-1.3.md).

## 1. Automated checks

```text
make fmt && make lint && make test
```

Expected: gofmt, vet, qmllint and qmlformat clean; Go tests pass (including `prism_settings_test.go` with 105 keys and the 128 limit test); QML tests pass (`prism_geometry_test.qml`, `prism_prefix_test.qml`); the static scans (no hardcoded colors, no magic values, interface use) pass on `modules/prism/` and `examples/hello-prism-source/`.

## 2. Hidden by default (US1, SC-004, SC-009)

1. Start the shell with an empty config. Run `lumen modules show prism`: state running, no warnings.
2. Check that no Prism surface exists (`niri msg layers` has no `lumen-prism`) and that the focused window still gets the keys.

## 3. Open, type, close (US1, FR-005)

1. Press the bound key. The window appears within 200 ms, horizontally centered, top at about one fifth of the screen, the cursor in the input, the placeholder shown, the empty state "No source is installed".
2. Type `hello`, then Backspace; check the edit works. Press Escape: the window closes and the window you were in has the keyboard (type into it).
3. Open, type text, click outside: it closes. Set `closeOnOutside: false`: the click does nothing.
4. Press the key twice quickly: the final state is closed, never two windows.
5. Run `lumen modules call prism open` from a second screen with a focused window: the window appears on that screen only.
6. Open on each screen while another is focused: never a window on two screens.

Measuring the 200 ms target (SC-001): record the screen at 60 fps or more while pressing the key, and count the frames from the key press to the first frame with the input shown (at most 12 frames at 60 fps); do the same for Escape.

## 4. Placement (US2, SC-002)

For each of the nine `anchor` values, with and without `offsetX` and `offsetY`, open the window and check its position against the table in `prism-surface.md` within 1 px, at 100% and at a fractional display scale. Check the bar's reserved space is respected (research R1). Then: set `width: 5000` (limited with a warning that names the key), `offsetY: 4000` (moved back, one warning), `screen: "NOPE"` (opens on the focused screen, one warning). Change `anchor` while the window is open: it moves within 1 s without closing. Set `anchor: bottom`, `inputPosition: bottom`, type a prefix that makes rows appear in preview mode, and check the bottom edge stays fixed and the results grow upward.

## 5. Theming and customization (US3, SC-003, SC-005)

With `preview: true`, change each of the following in the file, one at a time, with the window open, and check it applies within 1 second: every `*Role` and `*Opacity` key (spot-check one per part: window, border, input, selected, hover, shadow, backdrop), the global font and one role font, `radius`, `borderWidth`, `padding`, `gap`, `rowHeight`, `shadow`, `shadowBlur`, `backdrop`, `animation`, `animationDuration`. Set `fontFamily` to a family that is not installed: the default family is used and one warning names the key. Set `windowRole: "nope"`: that part falls back with an error that names the key and the role.

Theme check (Principle II quality gate): repeat with two presets (for example Tokyo Night and Catppuccin Mocha), a light and a dark wallpaper-derived palette, and the fallback palette. The window stays readable and `lumen modules show prism` lists no contrast warning for the default roles.

## 6. Prefix (US5, FR-010 prefix)

Open with `preview: true`. Type `@f`, `@f x`, `@fx`, `@x y`, `@@f`: the chip appears behind `@f` only in the first two. Set `prefixFiles: "p"`: `@p` gets the chip and `@f` does not. Set `prefixWeb: "a"` (a duplicate): an error names `prefixWeb` and it falls back to `w`.

## 7. Preview mode and keys (US4)

With `preview: true`: the 24 sample rows show, the first is selected, a very long title is cut with an ellipsis. Use Down, Up, Page Down, Page Up, Home, End (with an empty input); hover a row (the selection does not change) and click it (it does). Type text that matches nothing: the "Nothing matched" state. Enter on a row does nothing. Set `wrapSelection: true` and check the wrap. Scroll a list of 1000 rows (the example source with a high count) and type at a fast rate: no dropped characters, no visible stutter (SC-006, SC-009).

Measuring SC-006: type 200 characters into the open window with `wtype` and compare the input text with the sent text (it must be identical); the example source prints the query it last received, which must equal the text. Measuring SC-009: record the screen while scrolling the 1000 rows and while opening, and check no frame interval exceeds twice the refresh interval.

## 8. A source (US5, SC-007)

Install `examples/hello-prism-source` into the user modules directory and reload. Open Prism, type `abc`: three rows appear from the query with the window's look, and `lumen modules show prism` lists the source and 3 rows. Type `@f abc`: the source sees mode `files` and query `abc` (it prints it in a row). Choose a row: the source gets `chosen` and closes the window. Break the source (throw an error): its rows vanish, the window keeps working, and the log names the source file.

## 9. Keyboard rules (Principle VIII, SC-004)

1. With Prism loaded and the window closed, type in another window: Prism gets nothing, and hovering over where the window would be changes nothing.
2. Open it with the command: it holds the keyboard; close it: the previous window has it back. Repeat 20 times.
3. Check `docs/prism.md` lists the commands, the niri binding example, the keys while open, and says the module takes the keyboard when opened because it is a search box.

## 10. Failure and cleanup (US6, SC-008)

Put ten wrong values in the file (wrong types, out of range, bad enum, long strings): the window opens and each is named in `lumen modules show prism`. Disable the module: its commands say it is not running and no surface is left. Reload it while open: the window closes and the focus returns. Run 100 open and close cycles: memory stays within 12% and idle CPU is near zero with the window closed.

## Result of the run on 2026-11-09 (niri, Quickshell 0.3.2, three outputs)

Checked here, by the automated checks (`make fmt lint test`, `make smoke`, which includes section 18 and `make smoke SECTION=18c` with `--prism-cycles 100`: memory +0.3%) and by hand with a dev shell and screenshots:

- Sections 1 and 2: no surface and no keyboard at load; `toggle`, `open`, `close` and `clear` answer as the contract says; quick command sequences end in the state of the last one.
- Section 3: the window opens at the default place, takes the keyboard (`niri msg layers` shows exclusive interactivity), takes typed text (`wtype`), and Escape closes it. The unit tests check the nine anchors to the pixel; smoke checks that each opens.
- Section 4: `width: 5000`, `offsetY: 4000` and `screen: "NOPE"` give one warning each, naming the key. A bottom anchor with the input at the bottom keeps its bottom edge and grows upward (screenshot).
- Section 5: roles, backdrop, radius, a missing font family and low-contrast role pairs are applied and reported (screenshot and `lumen modules show prism`).
- Section 6: `@f xyz` is mode `files` with query `xyz`, with the highlight behind `@f`.
- Section 7: the 24 sample rows, selection with the arrow keys, the footer and the separator (screenshot).
- Section 8: the example source gives three rows from the query, receives mode and query, and `chosen` closes the window.
- Sections 9 and 10: ten wrong values are all named and the window still opens; disabling the module while open removes the surface.

Not checked here, left for a person: that the keyboard returns to the previous window (type into it after a close), the placement at a fractional display scale and a live anchor change while open, input method composition (research R5), the theme check over the two presets, a light and a dark wallpaper palette and the fallback palette, hover versus click, 1000 rows with fast typing, and the frame-rate and response-time readings (SC-001, SC-006, SC-009).
