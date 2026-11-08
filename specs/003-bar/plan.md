# Implementation Plan: Bar

**Branch**: `003-bar` | **Date**: 2026-11-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-bar/spec.md`

## Summary

The bar is a bundled module (`bar`) that owns three places (left, center, right) and fills them only from the user's lists. Everything it shows comes from other modules through a new, compatible module interface, **1.1**. The pieces:

1. **Extension points in the module system (interface 1.1).** Manifests may declare `extensionPoints` and `contributions` (the reserved fields from Modules now take effect) and `commands`. The `modules` Go program validates them at scan time. The Modules singleton keeps a registry of contributions and their states, and gives a host module an `api.extensions` object to read them and to create a contributed item. `requires` stays reserved.
2. **A `stringlist` setting type** (and optional `minLength`/`maxLength` on `string` settings) so that module settings can hold the three item lists and the screen list. Both the `modules` and `config` programs learn it. Wrong-type entries in a list are dropped on their own, logged, and announced with one desktop notification through a small `notify` helper of the bar.
3. **A new `modules items` subcommand** that resolves the user's lists against the available contributions (short and long names, unknown names, duplicates, modules not enabled) and returns the result with warnings. This is the logic of FR-009 to FR-012a, kept in Go and covered by black-box tests (Principle I).
4. **The `bar` module** (QML): one layer-shell panel per selected screen, three regions, the resolved items created through `api.extensions.create`, a pure JavaScript fitting function for the overflow rule, a **slot** object given to every item (placement facts, popup, focus), and keyboard focus.
5. **Three built-in item modules**, each a plain module that contributes to the bar: `bar-clock`, `bar-sound` (Pipewire, with a small shell helper that starts the mixer program), `bar-battery` (UPower).
6. **Module commands**: a manifest field `commands` plus `lumen modules call <id> <command>`, which the bar uses for `focus`. The shell IPC gains one function and stays at protocol 1.
7. **Docs and an example**: `examples/hello-bar-item/` (an item with a popup), updates to `docs/module-api.md` and `docs/writing-a-module.md`, and a new `docs/bar.md`.

No new dependencies (Pipewire and UPower are Quickshell services; EasyEffects or a PulseAudio-style mixer is an optional runtime program). No process per module. The shell IPC stays at protocol 1.

## Technical Context

**Language/Version**: Go 1.27 (stdlib only), QML/Qt 6 (Quickshell 0.3.2), POSIX `sh` for one module helper

**Primary Dependencies**: Quickshell (required) with its `Quickshell.Services.Pipewire` and `Quickshell.Services.UPower` services and the layer-shell window type (all present in the installed 0.3.2); matugen stays optional. Optional at runtime: EasyEffects or a PulseAudio-style mixer (`pavucontrol`), and `notify-send` (libnotify) for the notification about dropped list entries; both degrade to a log line. No Go dependencies.

**Storage**: Files only. Bar and item settings: `modules.<id>.*` in `$XDG_CONFIG_HOME/lumen/config.json` (the single user file). Bundled modules in `modules/`. Manifest schema `data/schema/module.schema.json` (updated to interface 1.1).

**Testing**: Black-box Go tests in `scripts/tests/` for `modules scan`/`items`, the `config` helper (`stringlist`), and the controller (`modules call`, `modules show`) with fixtures for good and broken contributors; QML unit tests for the fitting function with `qmltestrunner` (installed with Qt 6); an interface-usage test extended to every bundled module (SC-013); `scripts/tests/smoke.sh` extended with a bar section against a real shell; `qmllint`, `qmlformat`, `gofmt`, `go vet` as before.

**Target Platform**: Linux, Wayland, niri only.

**Project Type**: Desktop shell (QML UI + small Go programs), as in Foundation and Modules.

**Performance Goals**: Bar on every screen within 1 s of start (SC-001). Theme change within 1 s (SC-002). List change, enable, reload or disable reflected in under 1 s (SC-004). Popup within 1 s (SC-014). Screen hot-plug within 1 s (SC-012). Idle CPU under 1% with the bar and three items (SC-011).

**Constraints**: UI thread never blocks: list resolution runs in the `modules` helper through `Backend.run`; the only QML-side computation is the width fitting, which is a single pass over at most 64 items. No network, no telemetry. Memory flat over 100 cycles of contributor load/unload, bar reload and popup open/close (SC-010, SC-014).

**Scale/Scope**: At most 64 items per list, 8 extension points and 16 contributions per module, 16 commands per module, item names up to 65 characters (`module-id/item-id`). Five new bundled or example modules (`bar`, `bar-clock`, `bar-sound`, `bar-battery`, `hello-bar-item`) plus test fixtures, four Go programs (unchanged count), no new singleton.

No NEEDS CLARIFICATION remain; see [research.md](research.md). Four spikes (R8a panel space reservation, R9a outside-click handling for popups, R11a battery scale, R14a keyboard focus hand-back) are listed there as the first tasks, because they are facts about Quickshell 0.3.2 that cannot be settled by reading its type descriptions.

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Thin QML, heavy logic in Go | PASS with notes | Parsing and validating the new manifest fields, resolving the item lists (names, duplicates, unknowns, enabled state), and the new setting type are in Go. QML creates windows and items and presents. Two small exceptions are kept in QML, both presentation: the overflow fitting (needs the measured item widths, a pure function over at most 64 numbers) and the popup placement. The mixer launcher and the notification launcher are 10-line `sh` scripts, because each only picks an installed program and detaches it. No UI-thread blocking: all helper calls are asynchronous with time limits. Sound and battery use Quickshell services directly, which the principle allows when they are cheap. |
| II. Matugen theming | PASS | The bar, the popup and every built-in item take colors from `api.theme` and sizes from `api.style`. The no-hardcoded-colors test already covers all `*.qml` files outside fixtures, so it covers the new modules. |
| III. Modular components | PASS | The bar is a module using only the public interface (a usage test checks it, SC-013). Built-in items reach it through the same contribution mechanism a third party uses. The public interface gains a version (1.1) and is documented, with an example module. Single owner of the registry stays the Modules singleton. |
| IV. Reliability and degradation | PASS | A failing item is a failing module: it is unloaded after three early runtime errors, and only its place is released. A missing sound server or battery hides one item. The bar can fail without stopping contributors, which wait. A hanging item cannot be interrupted (stated in Modules' Assumptions and unchanged). |
| V. Layered override-only config | PASS | The lists and bar settings live under `modules.bar`, item settings under their own ids, with the same layers, validation, hot reload and reset. The new `stringlist` type follows the same rules. Defaults are never written to the file. |
| VI. Consistent, accessible UX | PASS | One design language through Style and Theme. All interactive items are keyboard operable through the bar's focus mode and the item key convention (see the bar-slot contract). |
| Technical constraints | PASS | niri first-class, XDG paths, no network, no new dependencies, optional ones degrade gracefully. Compositor-specific code: none in this feature (workspaces are deferred). |
| Quality gates | PASS | gofmt, vet, tests, qmllint, qmlformat. Theme changes verified on the bar with two presets, a light and a dark wallpaper palette, and the fallback palette. Interface changes are versioned (1.1, additive, so no migration note) and documented. |

**Post-design re-check (after Phase 1)**: unchanged. The design adds no new singleton, no new resident process, and no dependency. No violations to justify.

### Complexity Tracking

No constitution violations. The two QML-side computations (overflow fitting and popup placement) are noted in the table above as presentation logic that needs live measurements; the alternative of round-tripping widths through a helper would add latency to every resize for no testing gain, because the fitting function is a pure function that is unit tested with `qmltestrunner`.

## Project Structure

### Documentation (this feature)

```text
specs/003-bar/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── module-manifest-1.1.md   # extensionPoints, contributions, commands, stringlist, contribute permission
│   ├── module-api-1.1.md        # api.extensions, api.commands, host and contributor sides
│   ├── bar-config.md            # settings, item names, resolution rules, warnings, defaults
│   ├── bar-slot.md              # what the bar gives an item: placement facts, popup, focus, key convention
│   ├── cli-ipc-bar.md           # `lumen modules call`, show output, IPC addition (protocol stays 1)
│   └── helper-bar.md            # `modules items`, scan additions, config `stringlist`
├── checklists/requirements.md
└── tasks.md                     # created later by /speckit-tasks
```

### Source Code (repository root)

```text
singletons/
├── Modules.qml                  # extended: contribution registry, create/destroy of items, commands, version 1.1
├── ModuleApi.qml                # extended: api.extensions, api.commands
├── Backend.qml                  # unchanged
├── Config.qml                   # unchanged except it passes stringlist settings through the helper
└── Style.qml                    # unchanged
modules/
├── bar/                         # NEW: module.json, Main.qml, BarWindow.qml, BarRegion.qml, PopupHost.qml, layout.js, popup.js, bin/notify
├── bar-clock/                   # NEW: module.json, Main.qml (the item)
├── bar-sound/                   # NEW: module.json, Main.qml, bin/open-mixer
├── bar-battery/                 # NEW: module.json, Main.qml
├── test-surface/                # unchanged
└── README.md                    # updated
examples/
├── hello-module/                # unchanged
└── hello-bar-item/              # NEW: an item with a setting and a popup
data/schema/
├── module.schema.json           # updated to interface 1.1
└── config.schema.json           # unchanged (module settings are merged at runtime)
scripts/
├── src/
│   ├── modules/                 # extended: contributions/commands validation, `items` subcommand, stringlist
│   ├── config/                  # extended: stringlist rule kind
│   ├── lumen/                   # extended: `modules call`, `modules show` prints contributions
│   └── palette/                 # unchanged
└── tests/
    ├── fixtures/modules/        # NEW: bar and contributor fixtures (good and broken)
    ├── bar_layout_test.qml      # NEW: qmltestrunner test for layout.js
    ├── modules_contrib_test.go, modules_items_test.go, config_stringlist_test.go,
    │   controller_call_test.go, bar_api_usage_test.go            # NEW
    └── smoke.sh                 # extended with a bar section
docs/
├── bar.md                       # NEW: configuring the bar, finding item names, writing an item
├── module-api.md                # updated: interface 1.1
├── writing-a-module.md          # updated: contributing to the bar
└── layout.md                    # updated
README.md                        # updated: the bar, optional dependencies
```

**Structure Decision**: Same rules as Foundation and Modules: standalone Go programs under `scripts/src/<name>/`, no shared Go code between programs (the setting-rule check is duplicated in `modules` and `config` and kept in step by a fixture test, now with `stringlist`), the Modules singleton as the single owner of the registry, and a per-module API object so that a module sees only documented members. The bar and its items are ordinary directories under `modules/`, so the "built-ins use the public interface" rule is checked by the same test as for the test surface.
