---

description: "Task list for the Bar Customization and Styles feature"
---

# Tasks: Bar Customization and Styles

**Input**: Design documents from `/specs/004-bar-customization-styles/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md. Foundation, Modules and Bar (`specs/001-foundation/`, `specs/002-modules/`, `specs/003-bar/`) are implemented and complete.

**Tests**: Included, as in Feature III. The constitution requires automated tests for backend logic (the unset-default rule in both Go programs), the plan defines `qmltestrunner` tests for the pure QML functions, static scans for fixed visual values (FR-017), and a smoke test against a real shell for the surfaces, the 12 style and edge combinations and the resource checks.

**Organization**: Grouped by user story. The shared pieces every story needs (interface 1.2 in the Go programs and the Modules singleton, the bar's 34 settings, the axis-neutral layout, the geometry function, the resolved `look`) are in Phase 2. Font settings are resolved by the bar in US3 and used by the built-in items in US4, so the font scenario of US3 is finished by US4.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on unfinished tasks)
- **[Story]**: Which user story the task belongs to (US1 to US5)

## Path Conventions

Paths are relative to the repository root (see plan.md "Source Code"). Go programs are standalone `package main` under `scripts/src/<name>/` with no shared code between programs; they build to `scripts/bin/<name>`. Tests are black-box and live in `scripts/tests/`; fixtures in `scripts/tests/fixtures/modules/`. Write and change files with the Write and Edit tools only (the user's `CLAUDE.md`); use Bash only to build, test and run.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Settle the facts the plan could not read from type descriptions. Each spike runs in a throwaway directory outside the repository and records its result and chosen branch in `specs/004-bar-customization-styles/research.md` under "Measurements".

- [X] T001 (Done; (c) click-through not verifiable here, by hand later, see research.md) Spike R3 (panel margins, live anchors, input mask): in a throwaway Quickshell config, create a layer-shell panel with `exclusionMode: Auto` and `margins` set, check on niri (with `niri msg layers` and by maximizing a window) whether the reserved space is `thickness + margin` or only the panel's size; change its anchors live from top to left, bottom and right and check the panel moves and the reservation moves with it without recreating the window; set a `mask` made of three `Region` rectangles and check that a click in a gap reaches the window below. Also record the exact text `niri msg layers` prints for the panel, including its `Keyboard interactivity` line, with and without keyboard focus given. Record the result and the fallback used for each of (a), (b) and (c) in `research.md`
- [X] T002 (Done; seams, arcs and stacking are by-hand checks, see research.md) Spike R4 (frame surfaces): in the same throwaway config, build the frame from a strip panel, three border panels and four corner pieces (`QtQuick.Shapes`, empty mask, anchored to a corner with margins) on all four edges; check on niri for seams at output scale 1.0, 1.25 and 1.5, that the corner pieces are above windows and below fullscreen windows, that plugging an output in and out keeps the layout, and what `niri msg layers` shows for each surface, including each one's `Keyboard interactivity` line (it must be `none` for the border and corner surfaces). Record the result, and switch the plan to the one-overlay fallback in `research.md` if it fails
- [X] T003 (Done; `default: null` works, no sentinel needed) Spike R5 (unset settings): write a fixture manifest with `"default": null` settings of type `string`, `integer`, `number` and `boolean`, and trace by hand how the current `modules` program, the `config` helper, the merged schema (`scripts/bin/modules schema`) and `Modules.qml` treat it; list what must change. Record the result in `research.md`; if the pipeline cannot represent "unset", switch the plan to the `-1` sentinel fallback there and in `contracts/module-manifest-1.2.md` before T006
- [X] T004 [P] (Done; check works, generic names whitelisted) Spike R6 (installed fonts): in the throwaway config print `Qt.fontFamilies()` length and time, and test an installed family, a missing family, and the aliases `sans-serif` and `monospace`; test that a weight the family lacks renders with the nearest face. Record the result, and the fallback (no installed check) if the call is missing or slow, in `research.md`
- [X] T005 [P] (Done; forms decided, muted sound shows `off`) Spike R8 (vertical forms): in the throwaway config render, at the default thickness of 37 px and font 13, the clock as `HH` over `mm`, the sound item as `Vol` over `45%` and `Vol off`, and the battery as `Bat` over `85%` and `Chg` over `85%`; measure the widest line and decide the vertical forms or the fallback (a larger default thickness on vertical bars, or percent without a label). Record the result in `research.md` and update the vertical text in `contracts/bar-slot-1.2.md` if it changes

**Checkpoint**: The five spike results and their chosen branches are in `research.md`, and the plan and contracts match them.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Interface 1.2, the bar's settings, the axis-neutral layout and the geometry function, which every story builds on

**Warning**: No user story work can begin until this phase is complete

- [X] T006 [P] Update `data/schema/module.schema.json` to interface 1.2: allow `"default": null` for `string`, `integer`, `number` and `boolean` setting rules (not for `stringlist`), keep `api` pattern accepting `1.2`, and update the description of the version
- [X] T007 Update the `modules` program (`scripts/src/modules/manifest.go`, `scripts/src/modules/schema.go`, `scripts/src/modules/main.go`): the interface version constant becomes 1.2; a setting rule with `default: null` is valid for the four scalar types and invalid for `stringlist`; `min`, `max`, `enum`, `minLength` and `maxLength` still apply to user values; the merged schema (`modules schema`) leaves out `default` for such a key
- [X] T008 [P] (No code change was needed: the helper already carries a null default and rejects a null user value with its normal type message, covered by the T009 tests.) Update the `config` helper (`scripts/src/config/schema.go`, `scripts/src/config/resolve.go`): a key whose rule has `default: null` has no effective value unless the user sets one; a user value of `null` is an error `modules.<id>.<key>: null is not a <type>` and the key stays unset; reset removes the override and returns the key to unset; unknown keys are still preserved
- [X] T009 [P] (The `err-null-default-enum-value` case is covered inside `modules_null_default_test.go` instead of a fixture.) Add fixtures `null-default-good`, `err-null-default-stringlist`, `err-null-default-enum-value` and `api12-item` under `scripts/tests/fixtures/modules/` (contracts/module-manifest-1.2.md "Fixtures"), and write `scripts/tests/modules_null_default_test.go` covering: the manifests are valid or invalid as described; the merged schema has no `default`; the config helper returns the key only when set; a user `null` is an error; reset returns to unset; a 1.2 module is refused with both versions when the test sets the shell limit to 1.1. Extend `scripts/tests/module_rules_agree_test.go` so both programs agree on the new rule
- [X] T010 (An unset setting reaches `api.settings.values` as `null`, and `lumen modules show` prints `(unset)`. `ModuleApi.qml` needed no change.) Update `singletons/Modules.qml` and `singletons/ModuleApi.qml`: accept interface 1.2; an unset setting is absent from `api.settings.values` (reads as `undefined`) and is listed with `(unset)` where `lumen modules show` prints values; no new member is added to `api`
- [X] T011 Replace the `edge` rule and add the 28 new keys in `modules/bar/module.json` exactly as in `contracts/bar-config-1.2.md` (names, types, defaults, ranges, enums, descriptions), set `"api": "1.2"`, and keep the existing keys unchanged
- [X] T012 [P] Write `scripts/tests/bar_settings_test.go`: the bar manifest is valid and has exactly the keys, types, defaults and ranges of `contracts/bar-config-1.2.md`; every role enum equals the role names parsed from `singletons/Theme.qml`; `edge` accepts the four values and rejects `middle`; the merged schema marks the unset keys without a default; every key has a description. (The item settings agreement is added in T043.)
- [X] T013 [P] Make `modules/bar/layout.js` axis-neutral: `fit(lengths, available, opts)` with `opts = {itemGap, placeGap, placePadding}`, where a number as the third argument still means `{itemGap: n, placeGap: n, placePadding: 0}`; a place that holds a shown item adds `2 * placePadding`; keep the drop order and the center rule. Extend `scripts/tests/bar_layout_test.qml` (two gaps, place padding, the old call shape, a vertical-length example, dropping order unchanged)
- [X] T014 [P] Write `modules/bar/geometry.js` (pure functions, no QML types) and `scripts/tests/bar_geometry_test.qml`: `resolve(settings, tokens, screen)` returns the sizes of `data-model.md` and a list of warnings `{key, problem}`; defaults per style as in research R10 (the solid column equals Feature III: thickness 37 from font 13 and spacing 12, margin 0, padding 8, gaps 8, radius 0); the limits of `contracts/bar-surfaces.md` (half of the axis, `frameRadius` at most half of the inner area's smaller side); radii above half a shape are not reported; tests cover each style, each unset key, each limit and a very small screen
- [X] T015 (Also added a `lookWarnings` report field to `singletons/Modules.qml` and a "Look warnings" section to `lumen modules show` in `scripts/src/lumen/main.go`.) Update `modules/bar/Main.qml`: build one `look` object from `api.settings.values`, `api.style` and `api.theme` (edge and `vertical`, the style, the font with the installed-family check from T004, the seven role colors with opacity as theme-following bindings, the item facts), call `geometry.js` per screen for sizes and warnings, merge the warnings into the list printed by `lumen modules show bar` and log them once per change, and pass `look` and the per-screen sizes to every window. The edge falls back to `top` and the style to `solid` with an error naming the key when the value is not allowed
- [X] T016 (Passes: lint, tests, and `make smoke` 4 of 4 runs with 0 failures. Two Feature III flaky checks turned out to be real bugs and were fixed: the dropped-items oscillation in `BarWindow.qml`, and the log follower restarting with `--tail 1` before it had seen any output in `Modules.qml`. The smoke test's expected interface version moved to 1.2.) Run `make fmt lint test` and fix everything until it passes with zero warnings; confirm the existing Feature III smoke checks still pass with `make smoke` (the bar still looks the same: nothing in the windows uses `look` yet beyond the defaults)

**Checkpoint**: Interface 1.2 works end to end in both Go programs and the Modules singleton, the bar declares its 34 settings, and `geometry.js` and `layout.js` are tested. Feature III behavior is unchanged.

---

## Phase 3: User Story 1 - Choose how the bar looks: solid, islands or frame (Priority: P1) MVP

**Goal**: The `style` setting switches the bar between a solid strip, three islands and a frame, live, on the top edge.

**Independent Test**: Change only `modules.bar.style` between its three values and check the result on screen and with `niri msg layers` (quickstart steps 2 and 3).

### Tests for User Story 1

- [X] T017 [P] [US1] (Done as section 16 of `smoke.sh`; surfaces are told apart by the namespaces `lumen-bar`, `lumen-bar-border-<side>`, `lumen-bar-corner-<corner>`.) Extend `scripts/tests/smoke.sh` with a "styles" section (top edge): for `solid`, `islands` and `frame` set the style in the scratch config and check, with `niri msg layers` and the shell log, that the number of bar surfaces per output matches the style (1 for solid and islands; for frame, 1 plus the borders that have a width plus 4 corner pieces), that no surface remains after going back to `solid`, that an unknown style value logs an error naming `modules.bar.style` and shows solid, that `"frameBorder": 0` removes only the three border surfaces, that a theme change (`lumen theme tokyo-night --temp`) is logged as followed by every surface (each surface logs a debug line when its color follows the theme, added in T022), that in the islands style with all three lists empty (`"left": [], "center": [], "right": []`) no bar surface is listed and a maximized window takes the full screen, and that there are no QML errors
- [X] T018 [P] [US1] Add a smoke check that the bar window and its items are not recreated by a style change: count the `created item` and `removed item` log lines across solid, islands and frame and assert each item is created once per output (FR-005)

### Implementation for User Story 1

- [X] T019 [US1] Update `modules/bar/BarWindow.qml` for the solid style: draw the strip as a rectangle inset by `margin`, with `radius`, the `backgroundRole` color and `backgroundOpacity`; use the padding, `itemGap` and `placeGap` from the sizes (replacing the fixed `gap` and `margin` properties); keep `exclusionMode: Auto` and the reservation of `thickness + margin` (as chosen in T001); make the window background transparent so shape and opacity are drawn by the rectangle
- [X] T020 [US1] (Item holders wait in a parentless staging item while the panel is unmapped, because an item in an unmapped window reads as not visible and the panel would never come back.) Update `modules/bar/BarWindow.qml` for the islands style: draw one rounded rectangle per place that holds at least one shown item, using `islandRole`, `islandOpacity`, `radius` and the place padding passed to `layout.js`; draw nothing for an empty place; when no place holds a shown item on a screen, do not map the strip window at all (no surface, no reserved space) while keeping the window object and its items, so the island and the space return when an item is shown with nothing recreated (spec Clarifications, `contracts/bar-surfaces.md`); set the window's `mask` to the union of the island rectangles so the gaps are click-through (or use the fallback chosen in T001)
- [X] T021 [P] [US1] Create `modules/bar/FrameBorders.qml`: border panels (one per side other than the bar's, each `frameBorder` wide, `frameRole` color and `frameOpacity`, reserving its own space, empty mask) and four corner pieces (`frameRadius` square, anchored to a corner with margins, `QtQuick.Shapes` area between the corner and a quarter circle, same color, empty mask), all behind `LazyLoader`s that are active only when needed (or the one-overlay fallback chosen in T002)
- [X] T022 [US1] Update `modules/bar/Main.qml` and `modules/bar/BarWindow.qml`: in the frame style color the strip with `frameRole` and `frameOpacity`; instantiate `FrameBorders.qml` per shown screen with the sizes from `geometry.js`; make sure a style change only changes drawing, masks and the lazy surfaces, never the items; make the strip, border and corner surfaces each log one debug line when their color follows a theme change (used by T017); a screen that `screens` excludes gets no surface at all
- [X] T023 [US1] (Checked with a probe window on DP-2: solid 1013, islands 1005, frame 1874x1005 high, and the left and right edges, all as the sizes say. Margins are set only on the anchored sides, because the automatic reservation counts a margin on an unanchored side too.) Make the reserved space follow the style: windows must not go under the strip, the island window's margin or the frame (maximize a window in each style on this machine's three outputs); fix what the spike's chosen fallbacks require
- [X] T024 [US1] (Automated gates pass. The by-hand looks (seams, arcs, island click-through, themes on screen) are left to the quickstart.) Run `make fmt lint test smoke` and quickstart steps 2 and 3 by hand (the default bar unchanged; solid, islands and frame; theme changes in light, dark, two presets and the fallback); fix everything until it passes with zero warnings

**Checkpoint**: The three styles work on the top edge. This is the MVP: the visible half of the feature.

---

## Phase 4: User Story 2 - Put the bar on any of the four edges (Priority: P1)

**Goal**: `edge` accepts `top`, `bottom`, `left` and `right` with every style; a vertical bar lays items out top to bottom. Keyboard handling is not changed (spec FR-010).

**Independent Test**: Set the edge to each value with each style and check position, reserved space and item order (quickstart step 4).

### Tests for User Story 2

- [X] T025 [P] [US2] Extend `scripts/tests/bar_popup_test.qml` with the four edges: below for top, above for bottom, right of the item for left, left of the item for right; never overlapping the item, shifted to stay on screen at both ends, limited to half the screen and to the room on its side
- [X] T026 [P] [US2] (Done in section 16. The reserved space is not asserted in the smoke test, because no window is guaranteed to exist there; it was checked with a probe window during T023.) Extend `scripts/tests/smoke.sh` with the "edges" section: for each of the 12 style and edge combinations set the config and check the surfaces with `niri msg layers`, that each reserves space on the right side (maximized window geometry), that `lumen modules show bar` lists the dropped items by screen for an overflowing vertical bar (use a short output, `HDMI-A-1` is 1080x1920), that an invalid edge value falls back to `top` with an error naming `modules.bar.edge`, that opening the example popup and then changing the edge or the style closes it (log `popup closed`), that two edge or style changes written in quick succession end with the last value shown and each item created once, and that there are no QML errors

### Implementation for User Story 2

- [X] T027 [US2] Update `modules/bar/BarWindow.qml` for four edges: anchors from the edge (top or bottom: left and right plus the edge; left or right: top and bottom plus the edge), `margins` on the right sides, `implicitWidth` or `implicitHeight` from the thickness, the along-axis length for `layout.js`; change the edge without recreating the items (or recreate once if T001 chose that fallback)
- [X] T028 [US2] Update `modules/bar/BarWindow.qml` item holders for both axes: along-length from `implicitWidth` (horizontal) or `implicitHeight` (vertical) limited to `maxLength`, cross size `thickness`, items centered across; on a vertical bar place left items from the top, center items in the middle, right items from the bottom; report dropped items by length as before
- [X] T029 [US2] Update the slot in `modules/bar/BarWindow.qml`: `edge` takes four values, add `vertical` and `maxLength`, and make `maxWidth` equal `maxLength` on a horizontal bar and `thickness` on a vertical one (`contracts/bar-slot-1.2.md`)
- [X] T030 [US2] Keep keyboard handling unchanged in `modules/bar/BarWindow.qml` and `modules/bar/Main.qml` (FR-010, FR-011, spec Clarifications): add no key handling; check that the focus command, its keys from Feature III, Escape and `focusScreen` still work on every edge after the anchor and holder changes; and make an edge or style change close an open popup (`closePopup()` before re-anchoring)
- [X] T031 [US2] Update `modules/bar/popup.js` and `modules/bar/PopupHost.qml` to place the popup on the side facing into the screen for each of the four edges, using the item's rectangle in screen coordinates for any edge (replace the top/bottom-only origin computed in `BarWindow.qml` `openPopup`)
- [X] T032 [US2] Update `modules/bar/FrameBorders.qml` so the three border panels and the corner pieces follow the bar's edge (the strip is on the chosen edge, the borders on the other three), and check each of the four edges with the frame style
- [X] T033 [US2] (Automated gates pass; the rotated output and by-hand looks are left to the quickstart.) Run `make fmt lint test smoke` and quickstart step 4 by hand (all 12 combinations, the rotated output, a Feature III item fixture clipped on a vertical bar); fix everything until it passes with zero warnings

**Checkpoint**: Three styles on four edges, with popups working on all of them and keyboard behavior as in Feature III. US1 and US2 together are the new look of the bar.

---

## Phase 5: User Story 3 - Change every visual detail of the bar from the file (Priority: P1)

**Goal**: Every size, radius, gap, border, thickness, color role and opacity of the customization surface is a setting that applies live. The bar resolves the font settings into the slot (items use it in US4).

**Independent Test**: For each key of the surface, change it in the file and check the effect and the one-second rule, then reset it (quickstart step 5).

### Tests for User Story 3

- [X] T034 [P] [US3] (The look is logged as `key=value` pairs, so the smoke test can grep them.) Extend `scripts/tests/smoke.sh` with a "settings" section: for each size key (`thickness`, `margin`, `padding`, `itemGap`, `placeGap`, `itemPadding`, `itemRadius`, `radius`, `frameRadius`, `frameBorder`), each role key (`backgroundRole`, `islandRole`, `frameRole`, `textRole`, `focusRole`), each opacity key (`backgroundOpacity`, `islandOpacity`, `frameOpacity`) and each popup key set a valid value in the scratch config, check the shell logs the applied value within 1 s (the bar logs a debug line with its resolved sizes, roles and opacities per change) and no QML errors, then delete the key and check the default returns; the font keys are added in T049 once items use them; check that the config file holds only the keys written and keeps an unknown key after a sequence of sets and resets (SC-012)
- [X] T035 [P] [US3] Extend `scripts/tests/bar_defaults_test.go`: the bar manifest's role defaults are `surface`, `surface`, `surface`, `onSurface`, `primary`, `surfaceContainer`, `outline` and the opacities are 1, so the default look keeps Feature III's colors (SC-002); the unset size keys are all unset

### Implementation for User Story 3

- [X] T036 [US3] Wire every size of `geometry.js` into `modules/bar/BarWindow.qml` and `modules/bar/FrameBorders.qml`: thickness, margin, padding, `itemGap`, `placeGap`, radius, `frameRadius`, `frameBorder`; a radius above half of the shape's smaller side is drawn fully round (use `Math.min`, no warning)
- [X] T037 [US3] Wire the roles: the strip, islands, frame, focus indicator and popup colors come from `look.roles` (`api.theme` roles chosen by name with the opacity as alpha) in `modules/bar/BarWindow.qml`, `modules/bar/FrameBorders.qml`, `modules/bar/PopupHost.qml`; an unknown role (not expected, the enum rejects it) falls back to the part's default role
- [X] T038 [US3] Resolve the font in `modules/bar/Main.qml` and give it to items as `slot.font`, `slot.textRole`, `slot.focusRole`, `slot.itemPadding` and `slot.itemRadius` in the slot in `modules/bar/BarWindow.qml`; each follows its setting live
- [X] T039 [US3] Make the thickness adapt: an unset `thickness` follows the effective font size (`fontSize + 2 * spacing.md`), a set one is used as given within the geometry limits; the bar logs one debug line per change with its resolved sizes, roles with opacities, popup values and font (used by T034 and T049)
- [X] T040 [US3] (Automated gates pass; the by-hand pass over every key is left to the quickstart.) Run `make fmt lint test smoke` and quickstart step 5 by hand for every key of the customization surface; fix everything until it passes with zero warnings

**Checkpoint**: The bar and frame are fully customizable from the file. The font settings reach items in US4.

---

## Phase 6: User Story 4 - Items follow the bar and can be customized on their own (Priority: P2)

**Goal**: The built-in items and a third-party item follow the bar's font, padding, radius and roles; each built-in item has its own settings and a vertical form; the popup surface is customizable.

**Independent Test**: Change the bar's font and padding and check that the clock, sound, battery and the example item follow; then set a font on the clock only (quickstart step 7).

### Tests for User Story 4

- [X] T041 [P] [US4] (The smoke test (section 17) checks that the first is still shown on a vertical bar with every other item; the clipping itself is a by-hand look. The second reports its facts through a `facts` command.) Add fixtures `bar-item-ignores-direction` (a Feature III style item that uses `implicitHeight = slot.thickness` and a long text) and `bar-item-uses-slot-1-2` (declares 1.2 and reads each new slot member) under `scripts/tests/fixtures/modules/`, and extend `scripts/tests/smoke.sh` to enable them on a vertical bar and check: the first is shown and clipped to the thickness with no other item disturbed, the second reads the facts (it logs them) and follows a font change
- [X] T042 [P] [US4] (The slot members are documented in `docs/bar.md` now; the existing test also scans every QML file of the item modules, `ItemLook.qml` included.) Extend `scripts/tests/module_api_usage_test.go`: every `slot.` member used by a bundled module or example (including `vertical`, `maxLength`, `font`, `itemPadding`, `itemRadius`, `textRole`, `focusRole`) is documented in `docs/bar.md` and listed in `contracts/bar-slot-1.2.md`; no bundled item reads anything but `api` and `slot`
- [X] T043 [P] [US4] (Also checks that the three `ItemLook.qml` copies are identical.) Extend `scripts/tests/bar_settings_test.go`: the nine shared item settings are declared identically in `bar-clock`, `bar-sound` and `bar-battery` (names, types, ranges, enum, `default: null`), the role lists equal `Theme.qml`, and `bar-clock` has `verticalFormat` with the documented default

### Implementation for User Story 4

- [X] T044 [P] [US4] (The shared logic is in a new `ItemLook.qml` in each item module.) Update `modules/bar-clock/module.json` and `modules/bar-clock/Item.qml`: `"api": "1.2"`, the nine shared settings plus `verticalFormat`; the effective font, padding, radius and text role are the item's own setting, else the slot's value, else the token when there is no slot; draw the item's own background (`backgroundRole`, `backgroundOpacity`, radius) when set; use `verticalFormat` while `slot.vertical`; size along `implicitWidth` or `implicitHeight` as the direction requires and never exceed `slot.maxWidth`; keep the focus indicator on `slot.focusRole`
- [X] T045 [P] [US4] Update `modules/bar-sound/module.json` and `modules/bar-sound/Item.qml`: the same nine settings and rules, a vertical form (`Vol` over the percent, or the form chosen in T005), all key handling unchanged (Up and Down, Enter and `M` as in Feature III), mouse actions unchanged; the focus indicator on `slot.focusRole`
- [X] T046 [P] [US4] Update `modules/bar-battery/module.json` and `modules/bar-battery/Item.qml`: the same nine settings and rules, a vertical form (`Bat` or `Chg` over the percent, or the form chosen in T005), low and critical colors still from theme roles, hidden without a battery as before
- [X] T047 [US4] Update `examples/hello-bar-item/module.json` and `examples/hello-bar-item/Item.qml` (and `Main.qml` if needed): declare 1.2, read `slot.vertical`, `slot.font`, `slot.itemPadding`, `slot.textRole` and adapt to both directions, add an optional setting with `"default": null` that falls back to a token (for example `fontSize`), and keep the popup and the F12 test key
- [X] T048 [US4] (Done earlier with the frame work, in `PopupHost.qml`; the smoke test checks the popup values in the bar's look line.) Update `modules/bar/PopupHost.qml` and `modules/bar/BarWindow.qml` for the popup surface: the frame takes `popupRadius` (unset: `radius.md`), `popupPadding` (content inset), `popupBorderWidth` and the roles `popupBackground` and `popupBorder` from `look`, following them live; keep placement, size limit, one popup at a time and closing (Escape, outside click) as in Feature III
- [X] T049 [US4] (Done as section 17. The font keys of the bar are applied in the loop of section 16; the built-in items log a `look` line, and the fixture reports the slot facts. The popup's Escape closing is by hand.) Extend `scripts/tests/smoke.sh` with the items checks: set each font key (`fontFamily`, `fontSize`, `fontWeight`, `fontItalic`) on the bar, check the bar logs the applied value within 1 s (extending the loop of T034) and that the three built-in items and the example log the new font, then delete the key and check the default returns; set `modules.bar-clock.fontSize` and check only the clock changes; open the example popup with `popupRadius` and `popupPadding` set and check it logs the applied values and closes on Escape
- [X] T050 [US4] (Automated gates pass; the by-hand pass is left to the quickstart.) Run `make fmt lint test smoke` and quickstart step 7 by hand; fix everything until it passes with zero warnings

**Checkpoint**: Items and popups follow the bar and can be customized individually. The font scenario of US3 is complete.

---

## Phase 7: User Story 5 - A bad choice degrades only itself (Priority: P2)

**Goal**: Every kind of bad value falls back for its key only, is reported with the module and key, and leaves the bar working.

**Independent Test**: Put one bad value at a time into each kind of setting and check the fallback and the report (quickstart step 6).

### Tests for User Story 5

- [X] T051 [P] [US5] Write `scripts/tests/config_bar_values_test.go`: for the bar's manifest, a table of bad values (unknown style, unknown edge, negative size, oversize value, wrong type, unknown role, opacity above 1, a string for `fontSize`, a `null`) each yields an error `modules.bar.<key>: ...` for that key only, the key falls back (to its default or to unset), and every other key keeps its value; all errors are listed when there are several (SC-004)
- [X] T052 [P] [US5] (In section 16. A valid value that is too big for a screen cannot be produced here: every setting's maximum fits half of these screens, so the limit warnings are covered by the unit tests of `geometry.js` instead.) Extend `scripts/tests/smoke.sh` with the "bad values" section for the nine values of quickstart step 6: for each, the log and `lumen modules show bar` name the key and the reason, an oversize value is limited with one warning, the bar and the other settings keep working, and a font that is not installed gives one warning that names the key and the family

### Implementation for User Story 5

- [X] T053 [US5] (Done with T015: the config errors were already printed by `lumen modules show` as "setting errors"; the font and limit warnings are the new "Look warnings" section.) Make the bar report every issue: in `modules/bar/Main.qml` merge the config errors for `modules.bar.*` (`api.settings.errors`), the geometry warnings and the font warning into the warnings shown by `lumen modules show bar`, one line each with the key, and log them once per change; confirm `scripts/src/lumen/main.go` prints them (change it only if a line is missing)
- [X] T054 [US5] Apply the limits of `geometry.js` in the windows: reduced `thickness`, `margin`, `frameBorder` and `frameRadius` are used for drawing and reserving space, and each reduction produces one warning naming the key and the limit used (no warning for a radius above half of its shape); a combination that leaves no room for items still shows the bar with items dropped by the fitting rule
- [X] T055 [US5] (Automated gates pass: lint, tests, and smoke with 290 checks.) Run `make fmt lint test smoke` and quickstart step 6 by hand; fix everything until it passes with zero warnings

**Checkpoint**: All five user stories work.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Static checks for fixed values, documentation, resource measurements, final validation

- [X] T056 [P] (Added `TestNoFixedVisualValuesInModules`. The only fixed values it found were the 1 px focus ring of the sound item and the example, now `api.style.border.thin`, a new design token documented in `docs/module-api.md`.) Extend `scripts/tests/no_magic_values_test.go` (FR-017, SC-006): reject numeric literals assigned to `radius`, the margin properties, `spacing`, padding properties, `font.pixelSize`, `border.width` and size properties in every `*.qml` under `modules/*/` and `examples/*/` that is not a token, a declared setting or a named documented constant; fix every literal it reports in the new and changed code (including `BarWindow.qml`, `FrameBorders.qml`, `PopupHost.qml` and the three `Item.qml` files) by moving it to a token, a setting or a named constant with a comment
- [X] T057 [P] Write the bar section of `docs/bar.md`: the three styles with a configuration example for each and the look of the frame, the four edges and the vertical bar, a table of every bar setting with default and range copied from `contracts/bar-config-1.2.md`, which keys apply to which style, the limits and warnings, the item settings, the slot reference (all members of `contracts/bar-slot-1.2.md`), and a keyboard section that follows Principle VIII: the bar's one keyboard command (`focus`) with its example niri binding as in Feature III, that the clock, sound and battery items have no keyboard command, that nothing takes keyboard focus until the command runs, and the popups of the example item, which close on Escape
- [X] T058 [P] Update `docs/module-api.md` to interface 1.2 (the unset default `"default": null` and what a module must document for it, the new slot members, the version note "additive, no migration") and `docs/writing-a-module.md` (how an item follows the bar's font, padding and roles and adapts to a vertical bar, with the example item)
- [X] T059 [P] Update `README.md` (the Bar section: styles, edges, a configuration example for islands), `modules/README.md` (the changed bundled modules), `docs/layout.md` (`FrameBorders.qml`, `geometry.js`), and `docs/development.md` (running the new QML tests and the new smoke options)
- [X] T060 (The baseline is taken after a warm-up pass through all 12 combinations, because the toolkit keeps what it allocated for the first frame surfaces; see research.md "Measurements".) Add `--style-cycles N` to `scripts/tests/smoke.sh` (default 100, 10 in CI): change the style through all three values and the edge through all four values N times, assert memory within 12% of the use after the first cycle, no duplicated or leftover item, no leftover surface in `niri msg layers`, and add the idle CPU check in each of the three styles over the same window as the existing check
- [X] T061 (Numbers are in research.md "Measurements". Idle CPU used a 2-minute window per style, not 5. The one-second timings (SC-001, SC-003) are by hand and were not taken. A real leak of the frame's border panels was found and fixed.) Run the resource checks of T060 and the existing `--bar-cycles 100 --cpu-minutes 5` on a real session, and record the numbers under "Measurements" in `research.md` (memory over 100 style and edge cycles, idle CPU per style); fix anything outside SC-010, and measure by hand the one-second claims of SC-001 and SC-003 on this machine's three outputs
- [X] T064 (Done inside section 16 of `smoke.sh`: every bar surface reports no keyboard interactivity in all 12 combinations and before the focus command, the focus command reaches the bar, an open popup holds the keyboard and a closed one leaves no surface. Leaving with Escape stays by hand.) Add the keyboard check of Principle VIII (spec FR-010 and SC-008) to `scripts/tests/smoke.sh` as a "keyboard" section (the ID continues the sequence after the refinement for constitution 2.0.0; run it with T061): for each of the 12 style and edge combinations, read `niri msg layers` and assert that every surface of the bar (strip, borders, corner pieces) reports `Keyboard interactivity: none` before any command; run `lumen modules call bar focus` and assert that at most the strip surface reports a different value; open the example item's popup (driven by the fixture's module command, as in Feature III, because key events cannot be injected here) and assert the popup surface reports exclusive keyboard focus only while open and `none` after it closes; adapt the matching to the text recorded in T001. Leaving the focus session with Escape, and closing a popup with Escape, stay by-hand checks in the quickstart (step 4 and step 7)
- [ ] T062 (OPEN: waits for the owner to do or accept the by-hand checks listed under "Still by hand" in research.md, as was done for Feature III.) Set `**Status**: Complete` in `specs/004-bar-customization-styles/spec.md` once its own checks pass
- [ ] T063 (PARTLY DONE 2026-11-08: `make clean build fmt lint test smoke` passes from a clean build with 290 smoke checks and no lint warnings. OPEN: the by-hand steps of the quickstart (2 to 7 looks, click-through, frame seams, Escape) and a final `/speckit-analyze`.) Run the complete `specs/004-bar-customization-styles/quickstart.md` from a clean checkout (`make clean build`, scratch config and module directory), `make fmt lint test smoke`, and `/speckit-analyze` on the feature; fix everything until it passes with zero warnings

---

## Phase 10: Refinement of 2026-11-08, frame corners

**Purpose**: The frame style left gaps at the four screen corners (spec FR-004, scenario 4a, research R4a). The corner pieces of T021 become corner blocks; the layout of the other panels is not changed.

- [X] T065 [P] [US1] (The block carries the L-shaped `band`, the spandrel points `p0`, `p1`, `p2` and the arc direction instead of a disc center and an odd-even cut; see research R4a.) Add `cornerBlocks(sizes)` to `modules/bar/geometry.js`: for each of the four screen corners return `{id, atTop, atLeft, width, height, radius, cx, cy}` where `width` is the reserved space of the corner's side plus `frameRadius`, `height` is the reserved space at its end plus `frameRadius`, and `(cx, cy)` is the center of the quarter disc that stays transparent (the block corner farthest from the screen corner); leave out a block whose width or height is 0; with a radius of 0 the block is a plain rectangle. Extend `scripts/tests/bar_geometry_test.qml` with the sizes for each edge, for a frame radius of 0, for a border of 0, for both at 0 (no block next to a bare screen edge), and with the odd-even cut at the right corner of each block
- [X] T066 [US1] (Drawn as two non-overlapping `ShapePath`s, a band and a spandrel, not with the odd-even fill: that left the cut-out filled at the top-left corner.) Update `modules/bar/FrameBorders.qml`: replace the four corner pieces by corner blocks from `Geometry.cornerBlocks(sizes)`: each block is a transparent panel anchored to its screen corner with no margin and the size from the block, `ExclusionMode.Ignore`, an empty input mask, the namespace `lumen-bar-corner-<tl|tr|bl|br>`, and a `Shape` with one `ShapePath` using `OddEvenFill`: the block rectangle, then (when the radius is above 0) the quarter disc cut-out with the arc direction of its corner. The blocks exist whenever they have a size, also for a radius of 0, and keep the theme-follow log line
- [X] T067 [P] [US1] (Done as a loop over the four edges with a radius of 0, plus the case with both at 0; the 12-combination loop is unchanged.) Update `scripts/tests/smoke.sh` section 16: `expect_surfaces` expects the corner blocks whenever their size is above 0 (not only when the radius is above 0): with the defaults all four; with `"frameRadius":0` still all four on a frame with borders; with `"frameBorder":0` all four while the radius is above 0; add a check for `"frameRadius":0` and one for both 0 (only the strip, no blocks); and extend the 12-combination loop so each combination is also run with `"frameRadius":0`
- [X] T068 (Gates pass: lint, test, smoke 295 checks. Instead of looking by hand, the four corners were checked by pixel with an in-process capture on all four edges, a radius of 0 and a border of 3 with a radius of 30: no gaps, and the arc is right at every corner. Seams at scale 1.25 and 1.5 are still by hand.) Run `make fmt lint test smoke` and look at all four screen corners of the frame by hand on each edge, and with a frame radius of 0 (quickstart step 3, item 2); if any gap or seam is left, record what the screenshot shows in `research.md` R4a and fix it

## Phase 11: Refinement of 2026-11-09, frame gaps along the sides

**Purpose**: Gaps remained in the frame next to the corner blocks (spec FR-004, scenario 4b, research R4b): the border panels' `margins` counted the strip's space twice.

- [X] T069 [US1] Remove the `margins` block of the border panels in `modules/bar/FrameBorders.qml` (the compositor already lays a panel out inside earlier reservations) and update the file's header comment; keep the anchors and implicit sizes
- [X] T070 [P] [US1] Add a check to `scripts/tests/smoke.sh` section 16 that no border panel in `FrameBorders.qml` sets `margins` (a grep guard, since layer geometry cannot be read from `niri msg layers`)
- [ ] T071 [US1] (PARTLY DONE 2026-11-09: `make fmt lint test` pass. OPEN: `make smoke` and the pixel capture along the sides need a live niri session, and a look at the owner's screen.) Run `make fmt lint test smoke`; verify by pixel capture (as R4a) along every side, on all four edges, and record the result in `research.md` R4b

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies. T001 to T005 can run in any order; T003 decides whether T006 to T010 use `default: null` or the sentinel fallback, so do it before them.
- **Foundational (Phase 2)**: needs Phase 1. Inside it: T006 with T007 and T008 (T007 and T008 can be written in parallel, both need T003's result); T009 needs T007 and T008; T010 needs T007; T011 needs T006 to T008; T012 needs T011; T013 and T014 are independent of the Go work; T015 needs T010, T011, T013 and T014; T016 closes the phase.
- **US1 (Phase 3)** needs Phase 2 and the spike results T001 and T002. **US2** needs US1 (it changes `BarWindow.qml` and `FrameBorders.qml`) and T013. **US3** needs US1 (the windows it wires) and T014. **US4** needs US2 (the slot and axis) and US3 (the font and role facts). **US5** needs US3.
- **Refinement (Phase 10)**: T066 needs T065 (the block sizes); T067 needs T066; T068 comes last and needs T067. It changes only the frame's corner surfaces, so no other task is reopened.
- **Polish (Phase 8)** needs all stories; T056 and T057 need the code and slot documented by US4; T064 needs T001 (the recorded `niri msg layers` text), T026 and the example popup of T047 and T048.

### User Story Dependencies

- US1: foundation only. US2: US1. US3: US1. US4: US2 and US3. US5: US3.
- US2 and US3 touch different parts of the windows after US1 (edge and axes against sizes, roles and the slot), so they can be built by different people if `BarWindow.qml` changes are merged in turn.

### Parallel Opportunities

- T004 and T005 with T001 to T003 (different spike directories).
- T006, T008 (and T007 after it is designed) together; T013 and T014 beside the Go work; T012 while T011 is finished.
- T017 and T018 (US1 tests); T025 and T026 (US2 tests); T034 and T035 (US3 tests); T041, T042 and T043 (US4 tests); T044, T045 and T046 (three different module directories); T051 and T052 (US5 tests).
- T056 to T059 in Polish.

## Parallel Example: User Story 4

```bash
# Tests together:
Task: "scripts/tests/module_api_usage_test.go"
Task: "scripts/tests/bar_settings_test.go"

# The three built-in item modules together:
Task: "modules/bar-clock/Item.qml"
Task: "modules/bar-sound/Item.qml"
Task: "modules/bar-battery/Item.qml"
```

## Implementation Strategy

### MVP First

1. Phase 1 Setup, Phase 2 Foundational.
2. US1: the three styles on the top edge. **Stop and validate** (quickstart steps 2 and 3).
3. US2 (also P1): four edges and the vertical bar. US3 (also P1): every size, radius and color role from the file. US1 to US3 together are the MVP of the spec: the new looks, all edges and a customizable bar.

### Incremental Delivery

1. Add US2, validate quickstart step 4.
2. Add US3, validate step 5.
3. Add US4, validate step 7 (and finish the font scenario of step 5).
4. Add US5, validate step 6.
5. Polish and measurements (steps 8 and 9).

## Notes

- No shared Go code between programs. The `modules` and `config` programs each carry their own rule check, and `scripts/tests/module_rules_agree_test.go` keeps them in step (now with the unset default).
- The shell IPC stays at protocol 1. The interface becomes 1.2 and is additive; every Feature III module and fixture must still load (SC-007), so no fixture of Features II and III is changed.
- The five spikes (T001 to T005) decide the margin and anchor details, the frame surfaces, the null-default pipeline, the font check and the vertical forms. Each has a documented fallback, so none blocks the plan.
- Commit after each task or logical group, using conventional-commit prefixes.
