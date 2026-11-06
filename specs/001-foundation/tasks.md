---

description: "Task list for the Foundation feature"
---

# Tasks: Foundation

**Input**: Design documents from `/specs/001-foundation/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included. The constitution requires automated tests for backend logic (config merge, theme processing), and the plan defines black-box tests in `scripts/tests/`. Test tasks come before the implementation they cover and should fail first.

**Organization**: Grouped by user story. Foundational work that every story needs (the Backend helper runner, Log core, Style tokens) sits in Phase 2, so Config and Theme can use it before US4 and US5 add their own pieces.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on unfinished tasks)
- **[Story]**: Which user story the task belongs to (US1–US5)

## Path Conventions

Paths are relative to the repository root (see plan.md "Source Code"). Go programs live in `scripts/src/<name>/`, are standalone `package main` with no `go.mod` and no shared code between programs, and build to `scripts/bin/<name>`. Tests are black-box and live in `scripts/tests/`. Run them with `go test scripts/tests/*_test.go`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Repository skeleton and build tooling

- [X] T001 Create the directory skeleton from plan.md: `singletons/`, `testsurface/`, `modules/`, `data/presets/`, `data/schema/`, `scripts/src/lumen/`, `scripts/src/config/`, `scripts/src/palette/`, `scripts/tests/fixtures/`, `docs/`; add a placeholder `README.md` in `modules/` stating it is reserved for later features
- [X] T002 Update `.gitignore` to ignore `scripts/bin/`
- [X] T003 Create `Makefile` with targets `build` (loop `go build -o scripts/bin/<name> scripts/src/<name>/*.go` for `lumen`, `config`, `palette`), `run` (`scripts/bin/lumen start`), `fmt` (`gofmt -w scripts/src scripts/tests`, `qmlformat -i` on all `.qml`), `lint` (`gofmt -l`, `go vet` per program, `qmllint` on all `.qml`), `test` (`go test scripts/tests/*_test.go`), `smoke` (`scripts/tests/smoke.sh`)
- [X] T004 [P] Create a minimal compiling stub `main.go` for each program in `scripts/src/lumen/`, `scripts/src/config/`, `scripts/src/palette/` so `make build` succeeds
- [X] T005 [P] Create `scripts/tests/helpers_test.go`: build the three programs once into a temp dir (`TestMain`) and provide `runProgram(name, stdin, args...)` returning stdout, stderr, exit code

**Checkpoint**: `make build` and `make lint` run on the empty skeleton.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Pieces that every story depends on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T006 [P] Create `data/schema/config.schema.json` for the keys in `specs/001-foundation/contracts/config-keys.md` (draft-07 JSON Schema; unknown keys allowed)
- [X] T007 [P] Create `data/presets/tokyo-night.json`, `data/presets/catppuccin-latte.json`, `data/presets/catppuccin-frappe.json`, `data/presets/catppuccin-macchiato.json`, `data/presets/catppuccin-mocha.json` as `{id, name, mode, roles}` with the 16 roles from `specs/001-foundation/data-model.md`
- [X] T008 [P] Create `data/fallback-palette.json` (dark palette, id `fallback`, all 16 roles, text contrast ≥ 4.5:1)
- [X] T009 [P] Create `singletons/Style.qml` (`pragma Singleton`) with spacing (`xs`–`xl`), radius (`sm`, `md`, `lg`, `full`), typography (`family`, `caption`, `body`, `title`, `headline`, weights) and motion (`fast`, `normal`, `slow`, easing) tokens per data-model.md
- [X] T010 [P] Create `singletons/Log.qml` (`pragma Singleton`): `debug/info/warn/error(msg, fields)` that writes one ndjson line (`time`, `level`, `source: "qml"`, `msg`, fields) to the shell's stderr via `console.*`, with a `level` property (default `info`) filtering output, and a `forward(line)` function that writes an already-formatted line unchanged
- [X] T011 Create `singletons/Backend.qml` (`pragma Singleton`) with the helper runner `run(program, args, input, timeoutMs)`: starts a `Quickshell.Io.Process`, writes JSON to stdin, collects stdout, parses one JSON document, forwards stderr lines to `Log.forward`, starts a timer and kills the process on expiry; returns a request object with `state` (`running`/`ok`/`failed`/`timeout`), `helper`, `durationMs`, `data`/`error`, and a `finished` signal. A program that does not exist or is not executable is `failed` with the program name. Never blocks the UI thread (contract: `contracts/helper-protocol.md`, "Backend behavior")
- [X] T012 Create `singletons/qmldir` registering `Config`, `Theme`, `Style`, `Backend`, `Log` as singletons
- [X] T012a Spike (throwaway, not committed): in a scratch Quickshell config under the scratchpad, add an `IpcHandler` whose function returns a value immediately while a `Process` runs in the background, and confirm with `qs -p <dir> ipc call` that (a) the return value is synchronous, and (b) a second `status()` function can report the background result. Record the outcome in `specs/001-foundation/research.md` R1. If (a) is false, simplify `contracts/ipc-protocol.md` to a direct reply
- [X] T012b [P] Create the stub `singletons/Config.qml` (`pragma Singleton`, `ready: true`, `values: {}`, `errors: []`) so the shell loads all five singletons from US1 on; US2 replaces it
- [X] T012c [P] Create the stub `singletons/Theme.qml` (`pragma Singleton`) with the 16 color roles loaded from `data/fallback-palette.json` through a `FileView` (no color literals in QML); US3 replaces it
- [X] T013 Create `shell.qml` as the Quickshell entry point. It MUST reference all five singletons at startup (for example in `Component.onCompleted`: `Backend.ping()`, `Config.ready`, `Theme.paletteId`, `Style.spacing`, `Log.info(...)`) because QML singletons are created lazily and the `IpcHandler` must exist before the first call. It loads `testsurface/TestSurface.qml` (empty `FloatingWindow`/`PanelWindow` placeholder for now); create that placeholder too

**Checkpoint**: `qs -p .` starts without errors and `make lint` passes.

---

## Phase 3: User Story 1 - Clear project structure that builds and runs (Priority: P1) 🎯 MVP

**Goal**: A documented layout and a build/run/format/lint/test loop that passes on the skeleton, starting the shell with a themed test surface.

**Independent Test**: On a fresh clone follow only the README to build and start the shell, see the test surface, and run `make fmt lint test` clean (quickstart steps 1–2).

- [X] T014 [P] [US1] Write `README.md`: what Lumen is, the layout tree, build/run/format/lint/test commands, a dependency table (Quickshell required; Go required to build; matugen optional with "wallpaper themes use the fallback palette" when absent; Qt tools for lint), and a link to `docs/` (FR-002, FR-003)
- [X] T015 [P] [US1] Write `docs/layout.md`: where a new component (`modules/`, `testsurface/`), helper program (`scripts/src/<name>/`), preset (`data/presets/`), schema change, test (`scripts/tests/`) and doc belongs, with the rule "no shared code between programs" (FR-001, SC-010)
- [X] T015a [P] [US1] Add a "Compositor-specific code" section to `docs/layout.md`: it lives in `checkSession` in `scripts/src/lumen/main.go`, `detectWallpaper` in `scripts/src/palette/wallpaper.go`, and the session check in `shell.qml`; explain how to add another compositor, and state that this is isolation, not a full abstraction layer (FR-022). Also state that "helper programs" lives under `scripts/`
- [X] T016 [P] [US1] Write `docs/dependencies.md` listing each dependency, whether it is optional, and how the system degrades without it (FR-003)
- [X] T017 [P] [US1] Write `docs/development.md`: build without a `go.mod`, how to run a helper alone from a terminal, how to run tests and the smoke test, the rule that QML holds only presentation and light glue
- [X] T018 [US1] Implement `testsurface/TestSurface.qml` as a window showing a section per singleton (placeholders for Config/Theme/Backend status that later stories fill in), with colors taken only from `Theme` roles and sizes from `Style` (the Config and Theme stubs from T012b and T012c already exist)
- [X] T019 [US1] Add the "outside niri" check to `shell.qml` as one clearly named function: if `NIRI_SOCKET` is unset, log an error through `Log` and quit cleanly (spec edge case). Also check at startup that `scripts/bin/config` and `scripts/bin/palette` exist and are executable; if not, log "run `make build`" and quit cleanly
- [X] T020 [US1] Run `make build fmt lint test` and `qs -p .`; fix all warnings so the commands pass with zero warnings (SC-002)

**Checkpoint**: US1 independently works: the skeleton builds, starts, and passes lint/format/test.

---

## Phase 4: User Story 2 - Single-owner Config singleton (Priority: P1)

**Goal**: Layered config with live reload, schema validation, override-only writes, and reset.

**Independent Test**: Quickstart step 3: read a value, edit the file for a live update, introduce an invalid value, reset the override.

### Tests for User Story 2

- [X] T021 [P] [US2] Write `scripts/tests/config_resolve_test.go`: no file gives defaults and creates no file; one override merges; with `theme.source` set to `preset`, different presets change the effective `theme.mode` with source `preset`, and a user `theme.mode` still wins; invalid value reports `{key, problem, fallback}` and only that key falls back; empty, unreadable, and invalid-JSON files give defaults plus a whole-file error and leave the file unchanged; `themeOverride` gives source `temp`
- [X] T022 [P] [US2] Write `scripts/tests/config_write_test.go`: `set` with a list of changes writes only the overrides in one write and preserves unknown keys; a batch with one invalid value (including an unknown preset id, checked against `dataDir`) or an invalid file refuses and leaves the file byte-identical; `unset` with a list of keys removes them and empty parents, returns `written:false` when none was present, and never creates the file; random sequences of set/unset never leave a value the test did not set (SC-004); atomic write leaves no temp file
- [X] T023 [P] [US2] Write `scripts/tests/schema_sync_test.go`: keys and defaults reported by `config resolve` match `data/schema/config.schema.json` and the table in `specs/001-foundation/contracts/config-keys.md`

### Implementation for User Story 2

- [X] T024 [US2] Implement `scripts/src/config/main.go`: argument dispatch (`resolve`, `set`, `unset`), JSON stdin/stdout response envelope (`ok`, `protocol`, `data`/`error`), ndjson logging to stderr with `source: "helper:config"`, filtered by `LUMEN_LOG_LEVEL`
- [X] T025 [US2] Implement the compiled defaults and the key rules (type, enum, range, preset id list read from `dataDir/presets`) in `scripts/src/config/schema.go`
- [X] T026 [US2] Implement `resolve` in `scripts/src/config/resolve.go`: read user file, validate per key, merge defaults → preset (its palette and `mode` only) → user → temp, return `values`, `errors`, `sources`; never write
- [X] T027 [US2] Implement `set` (input `changes: [{key, value}]`, `dataDir`) and `unset` (input `keys: [...]`, `dataDir`) in `scripts/src/config/write.go`: validate all changes first, then generic JSON tree edit in a single atomic write, unknown keys preserved, atomic temp-file-and-rename write, creates only Lumen's config directory, refuses on invalid file or value
- [X] T028 [US2] Replace the stub with the real `singletons/Config.qml` (`pragma Singleton`), passing `dataDir` (from `Quickshell.shellDir + "/data"`) to every helper call: `FileView` with `watchChanges` on `$XDG_CONFIG_HOME/lumen/config.json` (fallback `~/.config`), ~100 ms debounce, calls `Backend.run` for `config resolve`, run counter drops stale results, exposes `values`, `errors`, `ready`, `value(key)`, `set(key, v)`, `reset(key)`, and a `reloaded` signal; sets `Log.level` from `log.level`
- [X] T029 [US2] Show Config values and `errors` on `testsurface/TestSurface.qml` (key, problem, fallback for each error)
- [X] T030 [US2] Run `make test` and the quickstart step 3 manually (live update < 1 s, invalid key falls back alone, invalid JSON file untouched, reset removes the key, unknown keys preserved)

**Checkpoint**: US2 works on its own with the test surface.

---

## Phase 5: User Story 3 - Theme singleton with wallpaper and preset sources (Priority: P1)

**Goal**: Semantic color roles from a wallpaper-generated palette or a bundled preset, live updates, fallback, and contrast compliance.

**Independent Test**: Quickstart step 4: switch light and dark wallpaper palettes, two presets, and the fallback; confirm live update and contrast.

### Tests for User Story 3

- [X] T031 [P] [US3] Write `scripts/tests/palette_test.go`: every preset returns `status:"ok"` with all 16 roles; every returned palette from all sources has text contrast ≥ 4.5:1 (WCAG AA); a preset with a violating role is adjusted and reported in `adjusted`; an unadjustable palette gives `fallback` (or `stale` when `hasPrevious`); unknown preset gives `fallback` with a reason; missing matugen (empty `PATH`) gives `fallback` with a reason; the compiled fallback equals `data/fallback-palette.json`
- [X] T032 [P] [US3] Create a fake matugen script `scripts/tests/fixtures/matugen-fake.sh` that prints a valid light or dark JSON copied from the real matugen output recorded in T035a (chosen by an argument), a failing variant, and a violating-contrast variant; use them (through `LUMEN_MATUGEN`) in `scripts/tests/palette_wallpaper_test.go` for the wallpaper source (light, dark, failure, bad wallpaper path → `stale`/`fallback`)

### Implementation for User Story 3

- [X] T033 [US3] Implement `scripts/src/palette/main.go`: read the JSON request, dispatch by `source`, emit the response envelope, log to stderr with `source: "helper:palette"` filtered by `LUMEN_LOG_LEVEL`
- [X] T034 [US3] Implement role set, hex parsing, the compiled fallback palette, WCAG relative luminance and contrast, and the adjust-or-reject logic in `scripts/src/palette/contrast.go`
- [X] T035 [US3] Implement preset loading from `dataDir/presets/<id>.json` with validation of all 16 roles in `scripts/src/palette/preset.go`
- [X] T035a [US3] Verify matugen: install it (or ask the user to if it cannot be installed here), record `matugen --version`, the exact command line for a wallpaper image with scheme and mode, and its real JSON output in `docs/development.md`; correct the "Matugen Role Mapping" table in `specs/001-foundation/data-model.md` if the color names differ. If matugen cannot be installed, mark the mapping as unverified in both files and keep T036 behind that note
- [X] T036 [US3] Implement the wallpaper source in `scripts/src/palette/wallpaper.go`: `detectWallpaper` (the only compositor/session-specific function, see T015a) takes the path from the request or `awww query`/`swww query`; run matugen with the command line recorded in T035a (executable located through `PATH`, overridable by `LUMEN_MATUGEN` for tests) with a timeout shorter than the call limit, map output to the 16 roles per `data-model.md`, and return `fallback`/`stale` with a reason on any failure
- [X] T037 [US3] Replace the stub with the real `singletons/Theme.qml` (`pragma Singleton`), keeping the initial colors loaded from `data/fallback-palette.json` and the previous palette shown until a new one has arrived: 16 color properties, `mode`, `paletteId`, `status` (`ok`/`fallback`/`stale` plus `reason`); on `Config.reloaded` or a wallpaper change call `Backend.run` for `palette` (passing `hasPrevious`); apply the full palette in one assignment; on `stale` keep the previous palette; log the reason through `Log`
- [X] T038 [US3] Add wallpaper change detection: `Theme.qml` re-runs `palette` when the wallpaper file at the resolved path changes (`FileView` `watchChanges`), debounced, last result wins
- [X] T039 [US3] Show palette swatches, `paletteId`, and `status`/reason on `testsurface/TestSurface.qml`, and make sure no color literal exists anywhere in `testsurface/` or `singletons/` except the Theme-owned defaults (add a check in `scripts/tests/no_hardcoded_colors_test.go` that greps `testsurface/*.qml` and `shell.qml` for color literals)
- [X] T040 [US3] Run `make test` and quickstart step 4 (light and dark wallpaper palettes, two presets, fallback with matugen hidden; preset/fallback update < 1 s, wallpaper update < 3 s with the previous palette shown meanwhile, no flicker). Record the measured matugen time in `docs/development.md`

**Checkpoint**: US1–US3 work; the MVP set (structure, config, theme) is complete.

---

## Phase 6: User Story 4 - Backend singleton for controller commands and helper runs (Priority: P2)

**Goal**: The `lumen` controller and the Backend IPC handler; robust helper failure and hang handling.

**Independent Test**: Quickstart step 5: run `lumen start`, valid and invalid commands, a failing and a hanging helper.

### Tests for User Story 4

- [X] T041 [P] [US4] Create fixtures `scripts/tests/fixtures/helper-fail.sh` (exits non-zero), `scripts/tests/fixtures/helper-badjson.sh` (prints invalid output), and `scripts/tests/fixtures/helper-hang.sh` (sleeps forever)
- [X] T042 [P] [US4] Write `scripts/tests/controller_test.go`: with a fake `qs` script on `PATH` (`scripts/tests/fixtures/qs-fake.sh`) assert exit codes from `specs/001-foundation/contracts/cli.md`: unknown command → 2, bad arguments → 2, `error: invalid:` reply → 2, no instance → 3, already running on `start` → 4, protocol mismatch → 5, `error: failed:` reply or a `failed` status → 6, missing `qs` or missing helper binaries → 8, hung call or a status that stays `pending` → 9 within the timeout, `queued` followed by `pending` then `done` → 0; each failure prints one clear stderr line and each run finishes in under 1 s
- [X] T043 [P] [US4] Write `scripts/tests/smoke.sh`: start the shell headlessly with a temporary `XDG_CONFIG_HOME`, run quickstart step 5 commands, make a helper fail and hang through a debug IPC hook or env override, and assert, first of all, that `lumen version` answers right after `lumen start` (proves the singletons were instantiated), and that the shell stays responsive (`version` call) and the hung helper is killed within its limit; support `--reload N` for the SC-009 memory check

### Implementation for User Story 4

- [X] T044 [US4] Implement `scripts/src/lumen/main.go` (single file, stdlib only): subcommands `start`, `theme <theme> [--temp|--reset]`, `version`; find `<root>` from its own path (`scripts/bin/../..`) or `LUMEN_ROOT`; `start` checks that `scripts/bin/config` and `scripts/bin/palette` exist (exit 8, "run `make build`"), pre-checks `qs -p <root> list --json`, and then runs `qs -p <root> -n`; other commands call `qs -p <root> ipc call lumen <function> [args]` after checking `version()`; for `queued <seq>` replies poll `status()` every 50 ms until `done`/`failed`; map `error: invalid:` to exit 2 and `error: failed:` / `failed` to exit 6; exit codes and one-line stderr messages per `contracts/cli.md`; 5 s timeout; niri-specific session check (`NIRI_SOCKET`) in one isolated function named `checkSession`; forward its own log entries through the `log` IPC function when a shell is running, otherwise to its own stderr
- [X] T045 [US4] Add the `IpcHandler` (`target: "lumen"`) to `singletons/Backend.qml` with `version`, `ping`, `setTheme`, `resetTheme`, `status`, `log` per `contracts/ipc-protocol.md`; `setTheme`/`resetTheme` validate, queue one `config set`/`unset` run with both theme keys, and return `queued <seq>` at once (never wait); `status()` reports `pending`/`done`/`failed` for the last queued command; unknown theme ids return `error: invalid: unknown theme '<id>'`; `--temp` passes `themeOverride` to `config resolve` without writing
- [X] T046 [US4] Wire theme overrides in `singletons/Config.qml`: a `tempTheme` property passed as `themeOverride` and cleared by `resetTheme`, a new `setTheme` or restart
- [X] T047 [US4] Harden `Backend.run` in `singletons/Backend.qml`: distinguish `failed` (non-zero exit, invalid JSON) from `timeout`, report errors with the helper name, never throw, and expose the last error per helper so only the calling feature degrades; show Backend status and last helper errors on `testsurface/TestSurface.qml`
- [X] T048 [US4] Run `make test smoke` and quickstart step 5 manually (second `lumen start` → exit 4; no shell → exit 3 and nothing written; commands effective in < 1 s)

**Checkpoint**: US4 works; the shell is controllable and robust to helper failures.

---

## Phase 7: User Story 5 - Shared design tokens and logging (Priority: P3)

**Goal**: Tokens drive every test-surface element; logging is structured, leveled, and in one stream from QML, the controller, and helpers.

**Independent Test**: Quickstart step 6.

### Tests for User Story 5

- [X] T049 [P] [US5] Write `scripts/tests/log_format_test.go`: every program writes stderr diagnostics as ndjson with `time`, `level`, `source`, `msg`; unknown fields are tolerated; the output of a failing helper still parses line by line
- [X] T050 [P] [US5] Write `scripts/tests/no_magic_values_test.go`: `testsurface/*.qml` contains no numeric literal for spacing, radius, font size, or duration outside `Style` references (simple grep; the allow-list is the literals `0` and `1` and any number inside a `Style.` expression. If the test proves flaky, drop it and rely on review)

### Implementation for User Story 5

- [X] T051 [US5] Make every test-surface element in `testsurface/TestSurface.qml` take spacing, radius, font, and animation duration only from `Style`, and add a token showcase section
- [X] T052 [US5] Complete `singletons/Log.qml`: `source` field, level filtering driven by `Config` `log.level`, a bounded in-memory ring buffer of the last 100 entries shown on the test surface, and routing of the controller's `log` IPC calls through the same formatter
- [X] T053 [US5] Make the `config` and `palette` programs honor a `LUMEN_LOG_LEVEL` environment variable passed by `Backend.run` (from `log.level`) so helper debug lines are filtered at the source
- [X] T054 [US5] Run quickstart step 6 manually: change a token and confirm all surface elements follow; emit logs from QML, the controller, and a helper and confirm they appear in one stream filtered by level

**Checkpoint**: All five stories work.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T055 [P] Add no-network and no-telemetry checks: `scripts/tests/no_network_test.go` greps `scripts/src` for `net/http`, `net.Dial`, and similar imports (FR-024)
- [X] T056 [P] Document the IPC protocol and helper contract versioning rules and add a migration-note template in `docs/development.md`, referencing `specs/001-foundation/contracts/`
- [X] T057 Measure idle CPU over 5 minutes (target < 1%, SC-008) and run `scripts/tests/smoke.sh --reload 100` for memory stability (SC-009); record results in `docs/development.md`
- [ ] T058 Verify a fresh clone reaches the running test surface in under 10 minutes using only `README.md` (SC-001) and that the layout answers "where does X go" in under 2 minutes (SC-010); fix any README gaps
- [X] T059 Run the full `specs/001-foundation/quickstart.md` validation and fix remaining issues; confirm `make fmt lint test smoke` pass with zero warnings

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies.
- **Foundational (Phase 2)**: depends on Setup; blocks all stories.
- **US1 (Phase 3)**: after Foundational.
- **US2 (Phase 4)**: after Foundational (uses Backend.run and Log from Phase 2).
- **US3 (Phase 5)**: after Foundational; reads Config, so it needs T028 for its keys (`theme.*`).
- **US4 (Phase 6)**: after US2 (the IPC handler runs `config set`/`unset`) and US3 (`setTheme` applies palettes). The T012a spike must be done first, because the IPC contract depends on its result.
- **US5 (Phase 7)**: after US2 (log level from config); can overlap with US3 and US4.
- **Polish (Phase 8)**: after the stories you want to ship.

### Within Each Story

- Tests first, and they should fail before implementation.
- Go program first, then the QML singleton that calls it, then the test-surface display, then the manual quickstart check.

### Parallel Opportunities

- T004 and T005 in Setup.
- T006–T010 in Foundational (different files).
- T014–T017 (documentation) in US1.
- T021–T023 (tests) in US2, T031 and T032 in US3, T041–T043 in US4, T049 and T050 in US5.
- `scripts/src/config/` (US2) and `scripts/src/palette/` (US3) can be built in parallel by different developers, since the programs share no code.

## Parallel Example: User Story 2

```bash
# Tests together:
Task: "scripts/tests/config_resolve_test.go"
Task: "scripts/tests/config_write_test.go"
Task: "scripts/tests/schema_sync_test.go"
```

## Implementation Strategy

### MVP First

1. Phase 1 Setup, Phase 2 Foundational.
2. US1: the skeleton builds, starts, and lints clean. **Stop and validate.**
3. US2 and US3 (both P1): config and theme. The MVP set is US1–US3.

### Incremental Delivery

1. Add US4: controller and IPC. Validate with quickstart step 5.
2. Add US5: token and log polish. Validate with quickstart step 6.
3. Polish and measurements.

## Notes

- No shared Go code between programs. Each of `lumen`, `config`, and `palette` carries its own small copies of the response envelope and ndjson logger.
- The controller is stdlib-only and calls only `qs`. It never touches config.
- If a Go dependency becomes unavoidable, add one `go.mod` in `scripts/`, record the reason in `plan.md`, and update `Makefile` accordingly.
- Task IDs with a letter suffix (T012a, T015a, T035a) were added after the first analysis pass; they are ordered where they appear.
- Commit after each task or logical group, using conventional-commit prefixes.
