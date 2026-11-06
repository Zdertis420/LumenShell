# Implementation Plan: Foundation

**Branch**: `001-foundation` | **Date**: 2026-11-06 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-foundation/spec.md`

## Summary

Foundation creates the repository skeleton and five core QML singletons (Config, Theme, Style, Backend, Log). There is no daemon. The shell is the only long-running process. Heavy logic lives in small standalone Go programs under `scripts/`, each of which takes input, does its work, and gives output:

1. `lumen`, the one-shot controller. It is a single stdlib-only file whose only job is to call `qs` (start the shell, `qs ipc call` into it).
2. `config` and `palette`, short-lived helpers that Backend runs from QML as a `Process`. They exchange JSON over stdin/stdout.

There is no Go module and no third-party dependency. A module is added only if a dependency becomes unavoidable. QML stays thin: it holds state and presents it. A minimal test surface shows that all five singletons work.

## Technical Context

**Language/Version**: Go 1.27 (stdlib only), QML/Qt 6 (Quickshell 0.3.x)

**Primary Dependencies**: Quickshell (required). matugen (optional; the fallback palette is used without it). No Go dependencies.

**Storage**: Files only. Config: `$XDG_CONFIG_HOME/lumen/config.json` (user overrides). State: `$XDG_STATE_HOME/lumen/`. Cache: `$XDG_CACHE_HOME/lumen/`. Bundled data: `data/presets/*.json`, `data/schema/config.schema.json`, read at runtime from `data/`. The fallback palette and the config defaults are also compiled into the helpers, so failure of the data directory cannot break them. State and cache directories are reserved and unused in Foundation.

**Testing**: `go vet` and `gofmt -l` on `scripts/src`, black-box Go tests in `scripts/tests/` (they build the programs and drive them over stdin/stdout/exit codes), `qmllint`, `qmlformat`, and `scripts/tests/smoke.sh` (starts the shell and drives it with `lumen`). `golangci-lint` is not installed, so it is not part of the gate.

**Target Platform**: Linux, Wayland, niri only. Compositor access sits behind an abstraction (FR-022).

**Project Type**: Desktop shell (QML UI + small Go programs)

**Performance Goals**: Config and theme changes visible in <1 s (SC-003, SC-005). `lumen` command effective in <1 s (SC-007). Idle CPU <1% (SC-008).

**Constraints**: UI thread never blocks (all Process calls async, with timeouts). No network. No telemetry. Stable memory over 100 reloads (SC-009).

**Scale/Scope**: 5 singletons, 3 Go programs, 5 presets (Tokyo Night plus four Catppuccin flavors), one test surface.

No NEEDS CLARIFICATION remain; see [research.md](research.md).

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Thin QML, heavy logic in Go | PASS | Merge, validation, palette work, contrast checks, and file writes are in Go. The constitution allows "daemon or helper binaries"; Foundation uses helper binaries plus a one-shot controller (spec clarification). The IPC is Quickshell's own, with a versioned, documented command set ([contracts/ipc-protocol.md](contracts/ipc-protocol.md)). All calls are async. |
| II. Matugen theming, presets | PASS | The Theme singleton is the only color source. Wallpaper (matugen via the `palette` helper) and presets map to the same roles. The fallback palette and the WCAG AA check are in Go. Exporting to GTK/Qt/terminals is a SHOULD deferred by the spec. |
| III. Modular components | PASS (scoped) | The module system is out of scope. Singletons have one owner each and a documented interface, and the layout reserves `modules/`. |
| IV. Reliability, degradation | PASS with note | Helper failure and timeout degrade only the affected feature. The principle's "UI reconnects to a restarted backend" does not apply because there is no daemon. The spec says the session restarts a crashed shell. |
| V. Layered override-only config | PASS | Defaults, then preset, then user file. Only overrides are written, unknown keys are kept, and the file is validated against a published schema, hot-reloaded, and resettable. |
| VI. Consistent, accessible UX | PASS | The Style singleton holds the tokens. Keyboard operability applies to later components. |
| Technical constraints | PASS | XDG paths, no network by default, no dependencies, and a compositor abstraction (niri only at first). |
| Quality gates | PASS | gofmt, vet, tests, qmllint, qmlformat, and a theme check with a light and a dark wallpaper palette, two presets, and the fallback. |

**Complexity Tracking**: no violations needing justification.

## Project Structure

### Documentation (this feature)

```text
specs/001-foundation/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md             # `lumen` controller commands and exit codes
│   ├── ipc-protocol.md    # Quickshell IPC target, functions, version
│   ├── helper-protocol.md # `config` and `palette` programs: JSON formats
│   └── config-keys.md     # Config key reference (source for config.schema.json)
└── tasks.md               # created later by /speckit-tasks
```

### Source Code (repository root)

```text
Makefile                    # build, run, fmt, lint, test, smoke targets
README.md                   # layout, dependencies, degradation, quickstart
shell.qml                   # Quickshell entry point (`qs -p .`); shows the test surface
singletons/                 # Config.qml, Theme.qml, Style.qml, Backend.qml, Log.qml, qmldir
testsurface/                # TestSurface.qml (verification only)
modules/                    # reserved for later features (README only)
data/
├── presets/                # tokyo-night.json, catppuccin-{latte,frappe,macchiato,mocha}.json
├── fallback-palette.json   # mirrors the palette compiled into the palette helper
└── schema/config.schema.json
scripts/
├── src/                    # one directory per program, each a standalone `package main`
│   ├── lumen/              # controller: stdlib only, calls `qs` and nothing else
│   ├── config/             # helper: resolve, set, unset
│   └── palette/            # helper: presets, matugen, fallback, contrast check
├── bin/                    # build output (git-ignored)
└── tests/                  # black-box Go tests, smoke.sh, fixtures (failing and hanging helpers)
docs/
├── layout.md               # where new components, helpers, presets, docs go
├── dependencies.md
└── development.md
```

**Structure Decision**: The QML config sits at the repository root, so Quickshell starts it with `qs -p .`. Each program under `scripts/src/` is built on its own (`go build -o scripts/bin/<name> scripts/src/<name>/*.go`), so no `go.mod` is needed while everything stays stdlib-only. If a dependency ever becomes necessary, a single `go.mod` is added in `scripts/` and the reason is recorded in the plan. Tests in `scripts/tests/` are black-box: they run the built programs and check stdout, stderr, and exit codes, which also proves each helper works alone from a terminal (FR-021a). Presets and the schema are read from `data/` at runtime. The controller locates the repository root from its own path or `LUMEN_ROOT`.
