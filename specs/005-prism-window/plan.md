# Implementation Plan: Prism Window (Skeleton)

**Branch**: `005-prism-window` | **Date**: 2026-11-09 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/005-prism-window/spec.md`

## Summary

Prism is a new bundled module, `modules/prism/`, that owns one floating, keyboard-driven search window (the skeleton of a Spotlight-like multi-menu). It builds only the window: open and close by command, nine placements, theme roles with opacity, a complete customization surface, an input with a mode prefix (`@a`, `@f`, `@w`, `@s`, `@g`), a results area with a row look and an empty state, an optional preview mode with sample rows, and a small public surface through which later features supply rows. The pieces:

1. **One module, no new singleton, no new program.** `modules/prism/` has `module.json`, `Main.qml` (settings resolved into one `look`, commands, sources, state), `PrismWindow.qml` (the layer-shell surface and its parts), `geometry.js` (pure placement, sizes and limits, as in the bar), `prefix.js` (pure prefix parsing) and `sample.js` (the preview rows). It uses only `api`.
2. **Module interface 1.3 (additive, one number).** The only interface change is that the per-module settings limit rises from 64 to 128 (the customization surface needs 105 keys). A module that declares `"api": "1.3"` needs a shell that provides it; every older module loads unchanged. Everything else (rows, query, chosen row) travels through Prism's own **source slot**, an extension point of the Prism module, in the same way the bar's slot works. No new `api` member.
3. **The window** is one transparent layer-shell surface on the overlay layer that covers the usable area of one screen while open, and is destroyed when closed (a `Loader`), so a hidden Prism has no surface, no focus and no cost (FR-002, FR-004). Inside it sit the optional backdrop, the panel (positioned by `geometry.js`) and the shadow. A click outside the panel closes it; Escape closes it; the surface asks for exclusive keyboard focus only while shown, and leaving it returns the focus to the previous window (the compositor's rule, the same one the bar's popup relies on).
4. **Prefix** parsing is a pure function in `prefix.js` on the keystroke path (it must meet the 50 ms rule without an IPC round trip). The five letters are settings; modules receive fixed mode names (`apps`, `files`, `web`, `commands`, `grep`, `default`).
5. **Sources.** Prism declares the extension point `sources`. A source is a contributed invisible item that gets a slot with `mode`, `query`, `text`, `selectedId`, `open`, `setRows(rows)`, `close()` and the signal `chosen(id)`. Prism merges the rows of all sources and routes `chosen` to the source that supplied the row. A failing source loses only its rows.
6. **Customization surface**: 105 flat keys under `modules.prism` (placement, behavior, content texts, prefix letters, shape and spacing, fonts per role, colors by role with opacity, backdrop, motion), validated by the existing Go setting rules, with `default: null` meaning "from the design tokens" (interface 1.2).
7. **Checks and docs**: QML unit tests for the two pure files, Go tests for the settings and the raised limit, the existing static scans extended to the new module, a smoke section, `docs/prism.md`, updates to `docs/module-api.md`, `docs/layout.md`, `README.md`, and an example source module.

No new dependency, no new resident process, IPC protocol stays at 1.

## Technical Context

**Language/Version**: Go 1.27 (stdlib only; only the limit constant changes), QML/Qt 6.12 (Quickshell 0.3.2), no new shell helpers.

**Primary Dependencies**: Quickshell's layer-shell `PanelWindow` (`WlrLayershell.layer`, `keyboardFocus`, `namespace`, `exclusionMode`) as the bar and its popup already use; `QtQuick.Effects` (`MultiEffect`) for the shadow, loaded only while the shadow is on. No new runtime programs.

**Storage**: Files only: `modules.prism.*` in `$XDG_CONFIG_HOME/lumen/config.json`. `data/schema/module.schema.json` changes one number (`maxProperties` 128). Optionally the last text (`keepText`) is held in memory only, never written.

**Testing**: QML unit tests with `qmltestrunner` for `geometry.js` and `prefix.js` (the Makefile glob is widened from `bar_*_test.qml` to the bar and prism files); Go black-box tests for the manifest, the 105 settings (defaults, ranges, role list agreement with `Theme.qml`, the 128 limit and the 129th key refused); the existing static scans (no hardcoded colors, no magic values, no network, interface usage) extended to `modules/prism/`; smoke checks on a real Quickshell instance on niri.

**Target Platform**: Linux, Wayland, niri only (as in Features I to IV).

**Project Type**: Desktop shell (QML UI + small Go programs).

**Performance Goals**: Open or close within 200 ms (SC-001); a setting change shows within 1 s (SC-003); mode and query reach a source within 50 ms of a key (SC-006); refresh-rate frames while opening, typing and scrolling 1000 rows (SC-009); idle CPU near zero while hidden because no surface exists.

**Constraints**: The UI thread never blocks (no helper is run). A row list is virtualized (a `ListView`), so 1000 rows cost only the visible delegates. Sizes are logical pixels. No network, no telemetry. At most 1000 rows per source and 200 characters per title, subtitle and hint are used; the rest is cut with a documented warning.

**Scale/Scope**: 105 settings (limit 128 after this feature), 1 new module, 1 extension point, 4 commands, 5 modes.

No NEEDS CLARIFICATION remain; see [research.md](research.md). Four spikes (R1, R2, R4, R5 there) are facts about Quickshell and niri that reading type descriptions cannot settle, and each has a documented fallback, so none blocks the plan.

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.* Constitution version 2.0.0.

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Thin QML, heavy logic in Go | PASS with a note | Validation of the 105 settings (types, ranges, enums, role names) is done by the existing Go rules. In QML remain presentation, the placement arithmetic and the prefix split. The prefix split is a few lines on the keystroke path, where an asynchronous round trip would only add latency (FR-029 asks 50 ms); it is a pure function with unit tests. All search and ranking logic, which would be heavy, is left to later features and belongs in Go helpers there. |
| II. Matugen theming | PASS | Every color is a role of `api.theme` chosen by name from the 16 roles, with an opacity where it is a background, shadow or backdrop. A test keeps the role list in `module.json` equal to `Theme.qml`. The existing no-hardcoded-colors scan covers the new files. The contrast rule is checked in `geometry.js` for the default and chosen roles (see data-model.md, "contrast"). |
| III. Modular components | PASS | Prism is a module on the public interface; sources reach it through a declared extension point and a documented slot, the way bar items do. The interface changes by one raised limit (1.3, additive, no migration). The usage test is extended to `modules/prism/`. A source cannot move, resize or restyle the window (it receives no window object). |
| IV. Reliability and degradation | PASS | A bad value falls back for its key; a missing font, screen or icon falls back with one warning; a failing source loses its rows only; a destroyed window leaves nothing behind; the window is created lazily, so a broken style cannot affect the shell at start. |
| V. Layered, override-only config | PASS | Same layers, validation, hot reload and reset as the bar; unset keys write nothing. |
| VI. Consistent, accessible UX | PASS | Defaults come from Style and Theme tokens; Prism consumes the central style module like every component. |
| VII. Total customizability | PASS | Nine placements (the launcher minimum), colors, fonts per role, radii, margins, paddings and gaps, all named settings; a magic-values test covers `modules/prism/`. The prefix letters are settings too. |
| VIII. Explicit keyboard interactivity | PASS | Prism is a keyboard-input module (a search box). It takes no focus at load, on hover or on any event the user did not ask for, ships no binding and registers no key; the commands `toggle`, `open`, `close`, `clear` are bound by the user in niri; the window takes the keyboard only when a command opens it, declares that in `module.json` (`keyboard` note in the description, and in `docs/prism.md`), closes on Escape and returns the focus. |
| Technical constraints | PASS | niri first class, XDG paths, no network, no new dependency. Choosing the focused output is left to the compositor (research R2) so that no compositor-specific code enters the module; if the spike fails, the fallback adds one documented isolated place. |
| Quality gates | PASS | gofmt, vet, tests, qmllint, qmlformat; the theme check (two presets, light and dark wallpaper palettes, fallback) is in the quickstart. The customization surface and keyboard commands are listed in the spec and in `docs/prism.md`, with live-change and Escape checks in the quickstart. |

**Post-design re-check (after Phase 1)**: unchanged. The design adds one module and one number to the manifest rule, no singleton, no resident process and no dependency. No violations to justify.

### Complexity Tracking

No constitution violations. Two choices add complexity on purpose:

| Choice | Why | Cheaper alternative and when it applies |
|--------|-----|-----------------------------------------|
| Raise the settings limit from 64 to 128 (interface 1.3) | FR-014 to FR-024 of the spec need 105 named settings (24 for colors, 32 for fonts per role); the limit protects the merged schema from abuse and 128 keeps that protection | Fewer keys: share one font across the rows (drop 20 keys) and one opacity for all backgrounds (drop 5). Used if the limit change proves hard to keep in step between `modules`, `config` and the schema. |
| Create the window on open and destroy it on close (a `Loader`) | A hidden Prism then has no surface, no input region and no focus (FR-002, SC-009), and an open from another screen is a re-create, which is also how the screen is moved | Keep one hidden surface and set `visible`. Used if spike R2 shows that the output cannot be chosen by the compositor when the surface is created. |

## Project Structure

### Documentation (this feature)

```text
specs/005-prism-window/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── module-manifest-1.3.md   # the raised settings limit, interface 1.3
│   ├── prism-config.md          # all 105 settings, defaults, limits, warnings
│   ├── prism-source-slot.md     # what a source gets and may do; rows, chosen, limits
│   └── prism-surface.md         # surface, layer, focus, input region, placement rules, commands
├── checklists/requirements.md
└── tasks.md                     # created later by /speckit-tasks
```

### Source Code (repository root)

```text
modules/
└── prism/                       # NEW
    ├── module.json              # 105 settings, point `sources`, commands toggle/open/close/clear
    ├── Main.qml                 # resolves settings into one `look`, commands, sources, state, warnings
    ├── PrismWindow.qml          # layer-shell surface: backdrop, shadow, panel, input, results, footer
    ├── ResultRow.qml            # one row (icon, title, subtitle, hint; selected and hovered looks)
    ├── geometry.js              # pure: anchors, offsets, size limits, usable-area clamp, contrast check, warnings
    ├── prefix.js                # pure: split `@x query` into {mode, query, prefixLength}
    └── sample.js                # pure: the fixed preview rows and their contains match
examples/
└── hello-prism-source/          # NEW: a source that supplies three rows from the query
singletons/Modules.qml           # apiVersion "1.3"
scripts/
├── src/modules/manifest.go      # maxSettings 128
└── tests/
    ├── prism_geometry_test.qml  # NEW
    ├── prism_prefix_test.qml    # NEW
    ├── prism_settings_test.go   # NEW: 105 keys, defaults, ranges, role list agrees with Theme.qml
    ├── modules_settings_limit_test.go  # NEW: 128 accepted, 129 refused; 1.3 on 1.2 rules refused
    ├── no_magic_values_test.go  # extended: modules/prism/
    ├── module_api_usage_test.go # extended: modules/prism/ and the example use only api
    ├── fixtures/modules/        # NEW: settings-128-good, settings-129-bad
    └── smoke.sh                 # extended: nine anchors, open and close, bad values, 100 open/close cycles
data/schema/module.schema.json   # maxProperties 128, interface 1.3
docs/
├── prism.md                     # NEW: commands, niri binding, prefixes, every setting, source slot, example
├── module-api.md                # updated: interface 1.3 (limit)
├── writing-a-module.md          # updated: sources for Prism
└── layout.md                    # updated: where Prism files go
Makefile                         # qmltest glob widened to prism_*_test.qml
README.md, modules/README.md     # updated
```

**Structure Decision**: Same rules as before. Prism is a bundled module that uses only `api`; the pure `.js` files hold the arithmetic and are unit tested without a shell; Go programs stay standalone, and the only Go change is one constant (with its schema twin, kept in step by the existing agreement test). Later features add search as sources (modules plus Go helpers) without changing the window.
