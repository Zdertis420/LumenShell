# Quickstart: Validating Bar Customization and Styles

Run these once the feature is implemented. Contracts: [module-manifest-1.2.md](contracts/module-manifest-1.2.md), [bar-config-1.2.md](contracts/bar-config-1.2.md), [bar-slot-1.2.md](contracts/bar-slot-1.2.md), [bar-surfaces.md](contracts/bar-surfaces.md). Earlier steps of [Feature III's quickstart](../003-bar/quickstart.md) must still pass.

## Prerequisites

- Feature III validated, a niri session (three outputs help), Go, Quickshell 0.3+, Qt 6 tools including `qmltestrunner`, `make`
- A scratch config and module directory: `export LUMEN_USER_MODULES_DIR=$(mktemp -d)` and `export LUMEN_CONFIG_FILE=$(mktemp -d)/config.json`
- `niri msg layers` for counting the bar's surfaces

## 0. Spikes (before building on them)

Record the results in `research.md` (R3, R4, R5, R6, R8): space reservation with margins and live anchor changes, input mask, frame seams and stacking, unset settings through the pipeline, the installed-font check, and the vertical forms at the default thickness. Each has a fallback in the research notes.

## 1. Build and gates

```text
make build
make fmt lint test
```

Expected: everything passes with zero warnings, including the extended `layout.js` tests, the new geometry and popup tests, the unset-default tests, the role and item-settings agreement tests and the extended static scans (SC-006, SC-007).

## 2. The default bar is unchanged (SC-002)

```text
scripts/bin/lumen start
```

Expected: a solid bar on the top edge that looks as it did after Feature III (compare the thickness, padding, colors and item positions); `lumen modules show bar` lists no warnings and prints `(unset)` for the unset keys.

## 3. The three styles (US1, SC-001)

Change `modules.bar.style` in `$LUMEN_CONFIG_FILE` one value at a time while the shell runs:

1. `"islands"`: three separate rounded shapes (left, center, right) with the desktop between them; click in a gap and the click reaches the window below; empty places (set `"left": []`) draw nothing.
2. `"frame"`: a border around the screen, items in the strip on the edge, the area inside has rounded corners; **look at all four corners of the screen on each edge: the band must be unbroken, with no gap and no seam, also with `"frameRadius": 0`**; set `"frameBorder": 0` and the three side borders go while the strip and the rounded corners stay; `niri msg layers` shows the surfaces appear and disappear with the style.
3. `"solid"`: back to one continuous strip; `niri msg layers` shows one bar surface per screen and nothing left over.

Expected: each change shows within 1 s, windows never go under the bar or frame, and a theme change (`lumen theme tokyo-night`, `lumen theme catppuccin-latte`, a wallpaper palette in light and dark, the fallback) changes every part within 1 s in all three styles (SC-005).

## 4. The four edges (US2, SC-001, SC-008, SC-009)

For each style set `edge` to `top`, `bottom`, `left` and `right`.

Expected: the bar moves within 1 s and its reserved space moves with it; on a vertical bar the left place is at the top and the right place at the bottom; the clock shows hours above minutes; the sound and battery items are readable at the default thickness; the focus command (`lumen modules call bar focus`) works as in Feature III (this feature changes no keys). With `niri msg layers`, check in every style and edge that every bar surface reports `Keyboard interactivity: none` until the focus command runs, and `none` again after Escape (Principle VIII). Try the rotated output (1080 x 1920) too. Add the example item and confirm it adapts to both directions; copy a Feature III item fixture and confirm it is shown and clipped to the thickness on a vertical bar.

## 5. Customize everything (US3, SC-003)

For each key of the customization surface in [bar-config-1.2.md](contracts/bar-config-1.2.md) change the value in the file and watch the bar: sizes, radii, gaps, thickness, margin, the font (family, size, weight, italic), the roles and the opacities. Then reset each key (delete it) and confirm the default returns.

Expected: every change shows within 1 s; the file holds only the keys you set and keeps unknown keys (SC-012); a radius above half of the shape's smaller side is drawn fully round with no message.

## 6. Bad values (US5, SC-004)

Put one bad value at a time: `"style": "pills"`, `"edge": "middle"`, `"thickness": -4`, `"margin": 5000`, `"fontFamily": "No Such Font"`, `"backgroundRole": "sparkles"`, `"islandOpacity": 3`, `"fontSize": "big"`, `"frameBorder": 900`.

Expected: only that key falls back, the log and `lumen modules show bar` name the key and the reason, an oversize value is limited with one warning, and the bar and every other setting keep working.

## 7. Items follow the bar (US4)

Set `fontFamily`, `fontSize` and `itemPadding` on the bar: the clock, sound, battery and the example item follow. Then set `modules.bar-clock.fontSize` only: only the clock changes. Give an item a `backgroundRole` and `backgroundOpacity`. Open the example popup with `popupRadius`, `popupPadding`, `popupBackground` set: it follows, on all four edges, never covering the item.

## 8. Resource checks (SC-010)

```text
scripts/tests/smoke.sh --bar-cycles 100
```

Expected: 100 changes of style and edge in a row stay within 12% memory of the first, no item is duplicated or left behind, no extra surfaces remain in `niri msg layers`, and the idle CPU over 5 minutes is under 1% in each style.

## 9. Scripted run

```text
make smoke
```

Runs the Feature I to III checks plus the new section (steps 2 to 4 and 6, and the resource check) against a real shell with a temporary config and module directory.
