# Implementation Plan: Modules

**Branch**: `002-modules` | **Date**: 2026-11-06 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-modules/spec.md`

## Summary

Modules adds a sixth singleton, **Modules**, that owns a registry of modules. A module is a directory with a `module.json` manifest and a QML entry file. The pieces:

1. **A new Go program, `modules`** (`scripts/src/modules/`). It scans the bundled and user module directories, validates every manifest against a published schema, checks the interface version, and reports problems per module. QML never parses manifests.
2. **The `config` helper learns module settings.** Callers pass the module definitions, and the helper validates `modules.<id>.*` the same way as core settings (override-only, unknown keys kept, bad keys fall back alone). `modules.<id>.enabled` is built in.
3. **The Modules singleton** (QML) loads and unloads enabled modules with `Qt.createComponent`, injects a per-module **API object** (the public module interface, version 1.0) as a required property, and tracks state, errors, and runtime-error counts. It follows the shell's own log (`qs log --follow`) to attribute runtime errors to modules by file path.
4. **The API object** is the only door a module needs: Theme, Style, its own settings, Log (tagged with the module id), helper runs through Backend, and a read-only module list. Where Lumen mediates access, it checks the declared permissions.
5. **Controller and IPC**: `lumen modules list|show|enable|disable|reload|rescan`, built on the existing channel. The controller stays stdlib-only and calls only `qs`.
6. **Foundation's test surface becomes the first bundled module** (`modules/test-surface/`), and an example module lives in `examples/hello-module/`.

No new dependencies. No process per module.

## Technical Context

**Language/Version**: Go 1.27 (stdlib only), QML/Qt 6 (Quickshell 0.3.x)

**Primary Dependencies**: Quickshell (required, including `qs log`); matugen stays optional. No Go dependencies.

**Storage**: Files only. Bundled modules: `modules/` in the repository. User modules: `$XDG_DATA_HOME/lumen/modules/` (default `~/.local/share/lumen/modules/`), never created or written by Lumen. Module settings: `modules.<id>.*` in `$XDG_CONFIG_HOME/lumen/config.json`. Manifest schema: `data/schema/module.schema.json`.

**Testing**: Black-box Go tests in `scripts/tests/` for the `modules` and `config` programs (fixture module directories under `scripts/tests/fixtures/modules/`), a fake-`qs` controller test for the new commands, and `scripts/tests/smoke.sh` extended with a module section that runs against a real shell. `qmllint`, `qmlformat`, `gofmt`, `go vet` as before.

**Target Platform**: Linux, Wayland, niri only.

**Project Type**: Desktop shell (QML UI + small Go programs), as in Foundation.

**Performance Goals**: An enabled valid module on screen within 1 s of start, enable, reload, or rescan (SC-002). Module commands in <1 s (SC-008). Idle CPU <1% with 10 idle modules (SC-009).

**Constraints**: UI thread never blocks: scanning and validation run in the `modules` helper through `Backend.run`; loading a component is the only QML-side work. No network, no telemetry. Memory flat over 100 load/unload cycles (spike: about 10 kB per cycle, SC-007).

**Scale/Scope**: Up to 256 modules scanned, manifests up to 64 KiB, up to 64 settings and 16 permissions per module. Six singletons, four Go programs, one bundled module, one example module.

No NEEDS CLARIFICATION remain; see [research.md](research.md).

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Thin QML, heavy logic in Go | PASS | Scanning, manifest validation, version checks, setting validation, and merging are in Go programs. QML holds the registry state, creates and destroys components, and presents. All helper calls are asynchronous with time limits. |
| II. Matugen theming | PASS | Modules take every color from `api.theme`. A test keeps color literals out of bundled module QML. |
| III. Modular components | PASS with one note | Manifest with id, version, required API version, permissions, settings schema; access to theme, config, and a backend surface (helper runs); discovery from a user directory; independent load and unload; versioned, documented interface; an example module; built-ins use the same interface. The note: "a crashing module MUST NOT take down the shell" cannot be met for a module that blocks the display thread or kills the process by its own code. See Complexity Tracking. |
| IV. Reliability and degradation | PASS | Manifest, version, and load errors are caught per module. Helper failures and hangs degrade one module. Runtime errors are attributed and counted, and an error storm unloads the module. |
| V. Layered override-only config | PASS | Module settings use the same layers, schema validation, live reload, reset, and preserved unknown keys, under `modules.<id>`. |
| VI. Consistent, accessible UX | PASS | Modules consume Style and Theme. Keyboard operability is a rule in the author guide; the bundled test-surface module has no interactive controls. |
| Technical constraints | PASS | XDG paths, no network by default, no new dependencies, niri-only. |
| Quality gates | PASS | gofmt, vet, tests, qmllint, qmlformat; theme changes verified on a module; IPC protocol and module interface changes versioned and documented. |

### Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|--------------------------------------|
| Principle III: containment of a module that blocks the display thread or crashes the process is not guaranteed | Modules are QML objects inside the shell process. The Quickshell/Qt scene cannot be split across processes, and a thread blocked by user code cannot be interrupted from the same process. Native plugins are not allowed, so a hard crash needs a bug in Qt itself. | Running each module in its own Quickshell process would give isolation, but each module would then be a separate shell with its own windows, config, IPC handler, and theme state, and cross-module features (bar items, launchers) would become inter-process. That is a different product and is out of proportion for this feature. The spec states the limit in its Assumptions. |

## Project Structure

### Documentation (this feature)

```text
specs/002-modules/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── module-manifest.md     # module.json fields, limits, permissions, reserved fields
│   ├── module-api.md          # interface 1.0: what a module can use
│   ├── ipc-modules.md         # additions to the shell IPC (still protocol 1)
│   ├── cli-modules.md         # `lumen modules ...` commands and exit codes
│   └── helper-modules.md      # the `modules` program, and the config helper changes
└── tasks.md                   # created later by /speckit-tasks
```

### Source Code (repository root)

```text
singletons/
├── Modules.qml                # NEW: registry owner (6th singleton)
├── ModuleApi.qml              # NEW: per-module API object (a normal type, not a singleton)
├── Backend.qml                # extended: module IPC functions, cancellable helper runs
├── Config.qml                 # extended: knows module definitions, passes them to the helper
└── qmldir                     # registers Modules and ModuleApi
modules/
├── README.md
└── test-surface/              # NEW: Foundation's test surface, rewritten as a module
    ├── module.json
    └── Main.qml
examples/
└── hello-module/              # NEW: the documented example (setting + helper program)
    ├── module.json
    ├── Main.qml
    └── bin/hello              # tiny helper script
data/schema/
├── config.schema.json
└── module.schema.json         # NEW: published manifest schema
scripts/
├── src/
│   ├── lumen/                 # extended: `modules` subcommands
│   ├── config/                # extended: module settings and enabled flags
│   ├── palette/
│   └── modules/               # NEW: scan, validate, merged schema
└── tests/
    ├── fixtures/modules/      # NEW: good and broken module directories
    └── modules_*_test.go, config_modules_test.go, controller_modules_test.go, smoke.sh (extended)
docs/
├── writing-a-module.md        # NEW: author guide using examples/hello-module
├── module-api.md              # NEW: reference for interface 1.0 (mirrors the contract)
└── layout.md                  # updated
shell.qml                      # extended: starts Modules after the startup checks
testsurface/                   # REMOVED (moved into modules/test-surface/)
```

**Structure Decision**: Same rules as Foundation: standalone Go programs under `scripts/src/<name>/`, no `go.mod`, no shared Go code between programs. The `modules` program and the `config` program each carry their own small copy of the setting-rule check; a test feeds both the same fixtures so they cannot drift. The Modules singleton owns the registry and the loading; the per-module API object is its own file so a module sees only that object, never the singletons' internals.
