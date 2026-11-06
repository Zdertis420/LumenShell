# Research: Foundation

Verified locally: Go 1.27.3, Quickshell 0.3.2 (`qs -n`, `qs list --json`, `qs ipc call` all exist), `qmllint`/`qmlformat` present, matugen not installed, golangci-lint not installed.

## R1. Controller-to-shell transport

- **Decision**: The controller runs `qs -p <root> ipc call lumen <function> [args]` against a `Quickshell.Io.IpcHandler` in the shell. One IPC target named `lumen` exposes a small set of functions (see [contracts/ipc-protocol.md](contracts/ipc-protocol.md)). The instance is selected by the config path, so no pid lookup is needed: Lumen is the only Quickshell instance of its config in the session.
- **Rationale**: Quickshell has this built in, so no socket server, no daemon, and no dependency. The controller then needs only `os/exec`, `encoding/json`, and the standard library.
- **Result reporting**: A Quickshell `IpcHandler` function returns synchronously, so it cannot wait for an async helper without blocking the UI thread. Mutating handlers (`setTheme`, `resetTheme`) therefore validate, queue the work, and return at once. The controller then polls a `status()` function (every 50 ms, 2 s limit) until the queued command is `done` or `failed`, so the exit code reflects the real outcome. A spike task (T012a) confirmed this behaviour on Quickshell 0.3.2 (2026-11-06): `qs -p <dir> ipc call lumen run` returned `queued 1` in about 20 ms while a 1 s `Process` ran in the background, `status()` showed `pending 1` and then `done 1: hi`, and `qs -p <dir> list --json` listed the instance with its `pid` and `config_path`.
- **Instantiation**: QML singletons are created lazily, so `shell.qml` references all five at startup; otherwise the `IpcHandler` would never register.
- **Alternatives**: Own Unix socket or D-Bus service in Go (needs a daemon, which the spec rejects). Reading a command file (no reply, slow).

## R2. Controller scope and single instance

- **Decision**: The controller is one stdlib-only file. It does two things: start the shell and call `qs ipc`. `lumen start` pre-checks `qs -p <root> list --json` for a live instance of this config to print "already running" and exit with a distinct code, then runs `qs -p <root> -n`. It does not read or write config. `lumen theme` passes the choice and the `persist` flag to the shell, and Backend runs the `config` helper to write the file.
- **Rationale**: Keeps the controller trivially small and free of shared code (user decision). Quickshell already refuses duplicates with `-n`, so the pre-check only improves the message (FR-021b). Config writes stay in one place (the `config` helper).
- **Consequence**: With no shell running, `lumen theme <theme>` fails with exit 3 and writes nothing. Persisting needs a running shell (recorded in the spec clarifications).
- **Preconditions**: `lumen start` also checks that `scripts/bin/config` and `scripts/bin/palette` exist and are executable, and exits 8 with "run `make build`" otherwise.
- **Error mapping**: Shell replies start with `error: invalid: ...` (bad input, exit 2) or `error: failed: ...` (runtime failure, exit 6).
- **Alternatives**: Controller writes the file itself (duplicates config logic, or needs shared code). A lock file (stale-lock risk).

## R3. Where logic runs

- **Decision**: Standalone programs in `scripts/src/<name>/`, built into `scripts/bin/<name>`. Each run is a fresh process: JSON on stdin, one JSON document on stdout, ndjson diagnostics on stderr. Backend owns timeouts (kill on expiry). Two helpers: `config` (subcommands `resolve`, `set`, `unset`) and `palette`.
- **Rationale**: Short-lived processes cannot leak memory across reloads (SC-009) and are testable from a terminal (FR-021a). Grouping the three config operations in one program avoids sharing code between programs.
- **Alternatives**: One helper per operation (shared merge and validation code would need a module). A long-running helper (a hidden daemon, which the spec forbids).

## R4. Build without a Go module

- **Decision**: No `go.mod`. Build with `go build -o scripts/bin/<name> scripts/src/<name>/*.go`. `go vet` and `gofmt -l` run over the same files. Tests are black-box programs in `scripts/tests/` that run the built binaries. `go:embed` is not used because it cannot reach `data/` outside a program's directory.
- **Rationale**: Matches the user's structure and keeps the project dependency-free. Black-box tests also check the real JSON contract.
- **Consequence**: Presets and the schema are read from `data/` at runtime. Backend passes `dataDir` in each helper request. The fallback palette and the config defaults are compiled into the programs so a missing `data/` cannot break fallback behavior.
- **Fallback**: If a dependency becomes unavoidable, add one `go.mod` in `scripts/` and record why in the plan.

## R5. Config layering, validation, and writes

- **Decision**: `config resolve` takes the user file path and `dataDir`, merges defaults, the selected preset, and the user layer, and validates the user layer by hand against a small schema subset (type, enum, range). It returns the effective values plus errors `{key, problem, fallback}`. A bad key falls back alone. `config set` and `config unset` take a list of changes and a `dataDir`, so one `lumen theme` call is a single atomic write (a theme change touches two keys, and the file watcher must never see a half-applied state). They edit a generic JSON tree (unknown keys preserved) and write atomically (temp file, then rename). An invalid or unparsable file is never overwritten: writes refuse and report.
- **Rationale**: Meets FR-008–FR-013 and the edge case that an invalid file is never discarded. A test keeps compiled defaults, `data/schema/config.schema.json`, and [contracts/config-keys.md](contracts/config-keys.md) in sync.
- **Alternatives**: A third-party JSON Schema library (a dependency for a tiny schema). Validation in QML (violates Principle I).

## R6. Live reload

- **Decision**: In QML, `FileView` with `watchChanges: true` watches the user file. Changes are debounced (about 100 ms) and trigger `config resolve`. A run counter drops stale results, so the last saved state wins.
- **Rationale**: Native, cheap, no polling (SC-008).
- **Alternatives**: inotify in a daemon (no daemon). Polling (CPU).

## R7. Palette generation and fallback

- **Decision**: The `palette` helper runs `matugen` (when on PATH) for wallpaper sources and maps its output to the semantic role set. The exact matugen flags and the role mapping are verified against a real matugen in task T035a before the wallpaper code is written; the expected mapping is in [data-model.md](data-model.md). Theme loads its initial colors from `data/fallback-palette.json`, so QML holds no color literals. Presets are JSON files with the same role set. Every palette goes through a WCAG AA contrast check (4.5:1 for text roles against their surfaces). Violating text roles are adjusted by shifting lightness; if that fails, the palette is rejected. On a missing matugen, failure, or rejection, Theme uses the compiled-in fallback palette, or keeps the last good palette when only the wallpaper file is bad, and reports the reason through Log and Theme `status`.
- **Rationale**: FR-014–FR-018 and the edge cases. The role set is defined once ([data-model.md](data-model.md)).
- **Alternatives**: Computing the palette in Go without matugen (breaks Principle II's source of truth).

## R8. Live theme application without flicker

- **Decision**: Theme exposes color roles as properties on one QML singleton. A change assigns a full new palette in one JS turn, and consumers bind to those properties.
- **Rationale**: Bindings update every consumer in one frame: no restart, no flicker. The previous palette stays shown until the new one has arrived, which also covers slow matugen runs.

## R9. Wallpaper discovery

- **Decision**: The path comes from `theme.wallpaper` when set. Otherwise the `palette` helper asks a small wallpaper provider function (first try: `awww`/`swww query`). If nothing is found, it reports "no wallpaper" and Theme keeps the last good or fallback palette. Wallpaper management stays out of scope.
- **Rationale**: niri has no built-in wallpaper setting, and the spec assumes the session supplies one. An explicit key keeps tests deterministic.

## R10. Logging

- **Decision**: The stream is ndjson on the shell's stderr (journald under a systemd user unit). Fields: `time`, `level`, `source` (`qml`, `controller`, `helper:<name>`), `msg`, optional `key` and `err`. QML `Log` writes one formatted line per entry. Backend forwards helper stderr lines to `Log` unchanged. The controller sends its entries to the running shell with the `log` IPC function; with no shell running, it writes to its own stderr. The `log.level` config key filters (default `info`).
- **Rationale**: One place without a writable log file from QML (QML cannot append). Meets FR-023.

## R11. Build, lint, and test commands

- **Decision**: `Makefile` targets: `build`, `run` (`scripts/bin/lumen start`), `fmt` (gofmt and qmlformat), `lint` (`go vet`, `gofmt -l`, `qmllint`), `test`, `smoke`.
- **Rationale**: Matches the constitution's gates.

## R12. Compositor abstraction

- **Decision**: FR-022 is met by keeping compositor-specific code in one named function per place, with niri as the only implementation: `checkSession` in `scripts/src/lumen/main.go`, `detectWallpaper` in `scripts/src/palette/wallpaper.go`, and one session check in `shell.qml`. `docs/layout.md` says where these live and how to add another compositor. A QML Compositor singleton is deferred.
- **Rationale**: Minimal surface (YAGNI) and no shared code. This is isolation, not a full abstraction layer, and the docs say so.
