# Implementation Plan: Bar Customization and Styles

**Branch**: `004-bar-customization-styles` | **Date**: 2026-11-08 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-bar-customization-styles/spec.md`

## Summary

The bar keeps its design from Feature III (one layer-shell panel per screen, three places, items created through the public module interface) and gains three looks, four edges and a complete customization surface. The pieces:

1. **Module interface 1.2 (additive).** Two changes, both optional for existing modules. (a) A setting may declare `"default": null`, meaning "unset": the module then falls back to a shared design token. This is how FR-016 is met without copying token values into manifests, so a later global override of Style reaches the bar unchanged. (b) The bar's slot gains placement facts: `vertical`, `maxLength`, the resolved `font`, `itemPadding`, `itemRadius`, `textRole` and `focusRole`. The `modules` and `config` Go programs learn `default: null`; the version constant moves to 1.2. A 1.1 module loads unchanged.
2. **The bar's settings** grow from 6 to 34 flat keys under `modules.bar`: the style, the four edges, sizes (thickness, margin, padding, gaps, radii, frame border), fonts, theme roles with opacity for each colored part, and the popup surface. Validation (ranges, enums, role names) is done by the existing setting rules in Go. Only the size limits that depend on the screen are computed in QML, in a pure, unit-tested JavaScript function.
3. **One panel per screen stays the owner of the items.** A style or edge change only re-draws and re-anchors that panel, so items are not recreated (FR-005). Extra surfaces exist only for the frame style (up to three border panels and four corner blocks that cover the screen corners and draw the rounded inner corners) and are created and destroyed lazily.
4. **Vertical layout.** `layout.js` becomes axis-neutral (lengths along the bar, an item gap and a place gap, padding per island), keyboard handling is left as in Feature III, and `popup.js` places a popup next to the item on any of four edges.
5. **Built-in items** (`bar-clock`, `bar-sound`, `bar-battery`) gain the same family of own settings (font, padding, radius, background role and opacity, text role), a vertical form, and follow the bar's font by default. The example item shows both directions.
6. **Checks and docs.** A test that finds visual values fixed in code (FR-017), tests that keep the role lists, the item settings and the docs in step, a smoke section that drives all 12 style and edge combinations, and updates to `docs/bar.md`, `docs/module-api.md` and the example.

No new dependency, no new singleton, no new resident process, IPC protocol stays at 1.

## Technical Context

**Language/Version**: Go 1.27 (stdlib only), QML/Qt 6.12 (Quickshell 0.3.2), POSIX `sh` (unchanged helpers)

**Primary Dependencies**: Quickshell with its layer-shell `PanelWindow` (its `margins`, `mask`, `exclusionMode`, `exclusiveZone` and `aboveWindows` properties exist in the installed 0.3.2 type descriptions) and `QtQuick.Shapes` for the frame's rounded corners. No new runtime programs.

**Storage**: Files only. All new settings live in `modules.bar.*` and `modules.<item>.*` of `$XDG_CONFIG_HOME/lumen/config.json`. Manifest schema `data/schema/module.schema.json` is updated to interface 1.2 (`default: null`).

**Testing**: Black-box Go tests (`scripts/tests/`) for manifests, the setting rules, the unset default and the role list agreement; QML tests with `qmltestrunner` for `layout.js`, a new `geometry.js` and `popup.js` on four edges; the existing static scans (colors, magic values, network, interface usage) extended; `smoke.sh` extended with the 12 style and edge combinations against a real shell, using `niri msg layers` to see the surfaces; `gofmt`, `go vet`, `qmllint`, `qmlformat`.

**Target Platform**: Linux, Wayland, niri only.

**Project Type**: Desktop shell (QML UI + small Go programs), as in Features I to III.

**Performance Goals**: A setting change shows within 1 s (SC-003). Idle CPU under 1% in each style (SC-010). 100 style or edge changes within 12% memory (SC-010).

**Constraints**: The UI thread never blocks (no helper calls are added). Frame style costs up to 8 small surfaces per screen, so each extra surface must be created only while the style needs it. All sizes are logical pixels. No network, no telemetry.

**Scale/Scope**: 34 bar settings (limit 64 per module), 9 settings per built-in item, 3 styles x 4 edges = 12 layouts, 3 bundled item modules plus the example changed. No new module.

No NEEDS CLARIFICATION remain; see [research.md](research.md). Five spikes (R3, R4, R5, R6, R8 there) are facts about Quickshell and niri that cannot be settled by reading type descriptions, and each has a fallback, so none blocks the plan.

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.* Constitution version 2.0.0 (re-checked 2026-11-08 after Principle VIII was added and Principle VI's keyboard rule moved to it).

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Thin QML, heavy logic in Go | PASS with notes | Value validation (ranges, enums, role names, opacity, text length) uses the existing Go setting rules in both programs, and `default: null` is validated in Go. In QML only presentation remains: screen-dependent limits and the three styles' geometry in a pure function (`geometry.js`), the fitting in `layout.js`, popup placement in `popup.js`. All three are pure functions with unit tests, and need the live screen size, which a helper cannot give without a round trip on every resize. No UI-thread blocking is added. |
| II. Matugen theming | PASS | Every color is still a role of `api.theme`, now chosen by name from the 16 roles with an opacity. The role list in the manifests is checked against `Theme.qml` by a test. The existing no-hardcoded-colors test covers all new files. |
| III. Modular components | PASS | The bar stays a module on the public interface. The new slot facts and `default: null` are public, documented and additive (interface 1.2). Built-in items use only `api` and `slot`. The usage test (SC-013 of Feature III) is extended to the new members. |
| IV. Reliability and degradation | PASS | A bad value falls back for that key, a missing font falls back with one warning, a value that cannot fit is limited with one warning, and no case stops the bar. Extra frame surfaces are lazy and released when the style changes. |
| V. Layered, override-only config | PASS | New keys follow the same layers, validation, hot reload and reset; unset keys write nothing, and `default: null` makes "no value" a first-class state instead of a sentinel number. |
| VI. Consistent, accessible UX | PASS | One design language through Style and Theme tokens (a central style module that items consume). Principle VI no longer carries a keyboard rule; it defers to VIII. |
| VIII. Explicit keyboard interactivity | PASS | This feature adds no key handling. The only keyboard command among its modules is the bar's `focus` command, unchanged from Feature III (bound by the user in niri; the clock, sound and battery items and the example item have none). Nothing this feature adds takes keyboard focus on its own: the strip, island, border and corner surfaces are created with keyboard interactivity `none`, the bar takes it only while the `focus` session is active, and a popup only while open and it closes on Escape (unchanged from Feature III). The workflow gate is met by the smoke check of T064 (the compositor's layer list shows each surface's keyboard interactivity) and by the example item's popup test. No module of this feature needs implicit keyboard focus. The keyboard commands and the niri binding stay documented in `docs/bar.md`. |
| VII. Total customizability | PASS with a note | The customization surface of the spec lists colors, fonts, radii, margins, paddings and placement for the bar, items and popup, and tests enforce that nothing is fixed in code (FR-017). Two points are narrower than the principle's wording, and are justified here: the bar has one text role (body), so font settings are per bar plus per built-in item instead of per text role; and motion durations, which the principle does not list, stay design tokens. The global parts of VII (user themes, global token overrides) are other features, as the spec's Clarifications say. |
| Technical constraints | PASS | niri first class, XDG paths, no network, no new dependency, compositor-specific code limited to layer-shell use inside the bar module. |
| Quality gates | PASS | gofmt, vet, tests, qmllint, qmlformat. The theme check (two presets, light and dark wallpaper palettes, fallback) is in the quickstart for all three styles. The interface change is versioned (1.2, additive, "no migration") and documented. The customization surface is listed in the spec, as the workflow gate requires. |

**Post-design re-check (after Phase 1)**: unchanged. The design adds surfaces only for the frame style, creates them lazily, and adds no singleton, resident process or dependency. No violations to justify.

### Complexity Tracking

No constitution violations. Two choices add complexity on purpose, and each has a cheaper fallback if its spike fails:

| Choice | Why | Cheaper alternative and when it applies |
|--------|-----|-----------------------------------------|
| `default: null` in the setting rules (touches `modules` and `config` programs) | Token-derived defaults stay true to FR-016 and a value of `0` stays usable for sizes | Sentinel `-1` meaning "from tokens", with `min: -1`. Used if spike R5 finds that null defaults break the merged schema or the settings pipeline. |
| Frame as one strip panel plus three border panels plus four corner pieces | Each border panel reserves its own space, and corner pieces stay small | One full-screen, input-less overlay drawing the whole frame, with the strip panel reserving space. Used if spike R4 shows seams or stacking problems. |

## Project Structure

### Documentation (this feature)

```text
specs/004-bar-customization-styles/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── module-manifest-1.2.md   # default null, interface 1.2, shared item settings
│   ├── bar-config-1.2.md        # all bar settings, defaults per style, limits and warnings
│   ├── bar-slot-1.2.md          # new slot facts, legacy behaviour on a vertical bar
│   └── bar-surfaces.md          # surfaces per style, space reservation, input regions, geometry rules, popup edges
├── checklists/requirements.md
└── tasks.md                     # created later by /speckit-tasks
```

### Source Code (repository root)

```text
singletons/
├── Modules.qml                  # accepts interface 1.2; unset (null) settings reach api.settings.values as undefined
├── ModuleApi.qml                # unchanged members; documented facts only
└── Style.qml                    # unchanged (tokens remain the defaults)
modules/
├── bar/
│   ├── module.json              # 34 settings
│   ├── Main.qml                 # resolves settings into one `look` object, warnings, per-screen windows
│   ├── BarWindow.qml            # edge-generic anchors, margins, mask, drawing for solid and islands, item holders on both axes
│   ├── FrameBorders.qml         # NEW: lazy border panels and corner pieces of the frame style
│   ├── PopupHost.qml            # four edges, popup radius, padding, border and role settings
│   ├── layout.js                # axis-neutral fitting: item gap, place gap, place padding
│   ├── geometry.js              # NEW: pure defaults per style, limits for small screens, resolved sizes and warnings
│   └── popup.js                 # placement on four edges
├── bar-clock/    module.json, Item.qml   # own settings, vertical form
├── bar-sound/    module.json, Item.qml   # own settings, vertical form (keys unchanged)
└── bar-battery/  module.json, Item.qml   # own settings, vertical form
examples/hello-bar-item/         # reads the new facts and adapts to both directions
data/schema/module.schema.json   # interface 1.2, default null
scripts/
├── src/
│   ├── modules/                 # accepts api up to 1.2 and default null in a setting rule
│   └── config/                  # effective value of a key with a null default is "unset"
└── tests/
    ├── bar_layout_test.qml      # extended: two gaps, place padding, vertical lengths
    ├── bar_geometry_test.qml    # NEW
    ├── bar_popup_test.qml       # extended: four edges
    ├── bar_settings_test.go     # NEW: keys, defaults, ranges, role list agrees with Theme.qml, item settings agree
    ├── modules_null_default_test.go  # NEW
    ├── no_magic_values_test.go  # extended: radius, margins, spacing, font sizes as numeric literals in bar files
    ├── module_api_usage_test.go # extended: new slot members documented in docs/bar.md
    ├── fixtures/modules/        # NEW: null-default fixtures, a 1.2 item on 1.1 rules, an item that ignores the direction
    └── smoke.sh                 # extended: 12 style and edge combinations, bad values, 100 style changes
docs/
├── bar.md                       # updated: styles, edges, every setting, vertical use, example per style
├── module-api.md                # updated: interface 1.2
└── writing-a-module.md          # updated: unset defaults, adapting to the bar's direction
README.md, modules/README.md, docs/layout.md   # updated
```

**Structure Decision**: Same rules as before: Go programs stay standalone under `scripts/src/<name>/` with no shared code (the null-default rule is implemented in both `modules` and `config` and kept in step by the existing agreement test), the Modules singleton stays the owner of the registry, and the bar and its items stay ordinary directories under `modules/` so the "built-ins use the public interface" test covers them. The three item modules repeat one block of settings and one small block of QML; a test keeps the declarations equal, because modules cannot import code from each other.
