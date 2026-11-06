# Layout: where things go

| You are adding... | Put it in |
|-------------------|-----------|
| A shared service all components use | `singletons/` (and register it in `singletons/qmldir`). Foundation defines exactly five: Config, Theme, Style, Backend, Log |
| A shell component (bar, launcher, notifications, ...) | `modules/<name>/` (reserved; empty in Foundation) |
| A throwaway view for checking singletons | `testsurface/` |
| A helper that does heavy work (parsing, filtering, validation) | `scripts/src/<name>/` as a standalone Go `package main`; add `<name>` to `PROGRAMS` in the `Makefile` |
| A test for a helper | `scripts/tests/<name>_test.go` (black-box: run the built program) |
| A fixture used by tests | `scripts/tests/fixtures/` |
| A bundled preset | `data/presets/<id>.json` and add the id to `data/schema/config.schema.json` and `specs/001-foundation/contracts/config-keys.md` |
| A config key | `data/schema/config.schema.json`, the defaults and rules in `scripts/src/config/schema.go`, and `specs/001-foundation/contracts/config-keys.md` (a test keeps them in sync) |
| An IPC function | `singletons/Backend.qml` and `specs/001-foundation/contracts/ipc-protocol.md` (bump `protocol` for breaking changes) |
| A document | `docs/` |
| A specification or plan | `specs/<feature>/` |

## Rules

- QML holds presentation and light glue only. Parsing, validation, merging, and color work belong in a Go program.
- No shared Go code between programs. Each program under `scripts/src/` is built on its own, with only the standard library. If a dependency ever becomes unavoidable, add a single `go.mod` in `scripts/` and record why in the plan.
- The controller (`scripts/src/lumen`) only calls `qs`. It never reads or writes configuration.
- Components never hardcode colors, spacing, radii, font sizes, or durations. Use `Theme` and `Style`.
- Terminology: *helper programs* are the Go programs run by Backend (`config`, `palette`); `scripts/` is only the directory name.

## Compositor-specific code

niri is the only supported compositor. Compositor-specific code is isolated in three places, so adding another compositor means changing only these:

- `checkSession` in `scripts/src/lumen/main.go` (is a supported session running?)
- `detectWallpaper` in `scripts/src/palette/wallpaper.go` (which wallpaper is set?)
- `checkSession` in `shell.qml` (the shell's own session check)

This is isolation, not a full abstraction layer. A Compositor singleton (workspaces, windows, outputs) is left to a later feature.
