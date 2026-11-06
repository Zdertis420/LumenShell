---

description: "Task list for the Modules feature"
---

# Tasks: Modules

**Input**: Design documents from `/specs/002-modules/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md. Foundation (`specs/001-foundation/`) is implemented.

**Tests**: Included. The constitution requires automated tests for backend logic, the plan defines black-box Go tests and a smoke test, and several requirements (isolation, memory, permissions) can only be shown against a real shell. Test tasks come before the implementation they cover and should fail first.

**Organization**: Grouped by user story. The shared pieces every story needs (fixtures, the published manifest schema, module support in the `config` helper, cancellable helper runs) are in Phase 2.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on unfinished tasks)
- **[Story]**: Which user story the task belongs to (US1–US5)

## Path Conventions

Paths are relative to the repository root (see plan.md "Source Code"). Go programs are standalone `package main` under `scripts/src/<name>/` with no `go.mod` and no shared code between programs; they build to `scripts/bin/<name>`. Tests are black-box and live in `scripts/tests/`; run them with `make test`. QML rules from Foundation apply: no `: void` or `: var` annotations (use `: variant`), no `sed -i` or shell redirection to edit files, no color or spacing literals in module QML.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Wire a fourth Go program into the build

- [X] T001 Add `modules` to `PROGRAMS` in `Makefile` (build, lint, vet loops) and create the stub `scripts/src/modules/main.go` (`package main`, empty `main`) so `make build` produces `scripts/bin/modules`
- [X] T002 Update `scripts/tests/helpers_test.go` so `TestMain` also builds the `modules` program, and add a helper `runModules(t, subcommand, input)` that returns the parsed envelope and the raw result (same style as `runProgram` + `parseEnvelope`)
- [X] T003 [P] Create the directories `modules/test-surface/`, `examples/hello-module/bin/`, and `scripts/tests/fixtures/modules/` (add a one-line `README.md` in `scripts/tests/fixtures/modules/` explaining that each subdirectory is a module fixture)
- [X] T003a Change the `QML` variable in `Makefile` so `make fmt` and `make lint` skip `./scripts/tests/fixtures/*` (several fixtures contain deliberately broken QML that `qmllint` rejects and `qmlformat -i` would rewrite); verify with a temporary broken file there that `make lint` still passes, then remove it

**Checkpoint**: `make build` and `make lint` pass with the stub.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Things every user story needs

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 [P] Create manifest-level fixtures under `scripts/tests/fixtures/modules/` (each a directory with a `module.json` and, where valid, a `Main.qml` containing `import QtQuick`, `import Quickshell`, and `Scope { required property var api }`): `good-min` (only required fields), `good-full` (settings of all four types with enum/min/max, permissions `helpers` and `config.read`, one helper `hello` at `bin/hello` marked executable, description), `warn-reserved` (valid, plus `extensionPoints`, `contributions`, `requires`, and one unknown field), `err-no-manifest` (directory without `module.json`), `err-bad-json` (`{not json`), `err-bad-id` (id `Bad Id`), `err-missing-field` (no `version`), `err-bad-api` (api `"one"`), `err-newer-minor` (api `1.9`), `err-other-major` (api `2.0`), `err-missing-main` (names `Nope.qml`), `err-main-escape` (main `../Main.qml`), `err-bad-permission` (permission `root`), `err-dup-permission` (same permission twice), `err-bad-setting-default` (integer setting with default `"x"`), `err-reserved-setting` (a setting named `enabled`), `err-helper-escape` (helper path `../x`), `err-helper-notexec` (helper file without the executable bit), `err-helper-absolute` (helper path `/bin/true`); plus `dup-a` and `dup-b` (both with id `dup`, for the duplicate-id tests), and a directory named `bad name!` with no `module.json` (to check the sanitized `dir:bad_name_` id). After creating the files, run `chmod +x` on `good-full/bin/hello` only; `err-helper-notexec` must stay non-executable
- [X] T005 [P] Create behavior fixtures under `scripts/tests/fixtures/modules/` for the real-shell tests (valid manifests, `api` `1.0`): `qml-syntax-error` (`Main.qml` with a syntax error), `qml-missing-import` (imports a missing directory), `qml-runtime-once` (one runtime error after 300 ms, then quiet), `qml-error-storm` (a timer that throws every 100 ms), `qml-good-window` (a small `FloatingWindow` whose color is `api.theme.background`, logs `fixture good-window loaded` through `api.log.info` on load), `helper-fails` and `helper-hangs` (each declares permission `helpers` and a helper `h` pointing to `scripts/tests/fixtures/helper-fail.sh`/`helper-hang.sh` copied into the module's `bin/` (then `chmod +x` them), and calls `api.run("h", {}, 1500, cb)` on load, logging the request state), `api-probe` (permissions `helpers`, `config.read`, `log.read`; on load logs one line per documented API member it reads or calls, with the result, and logs `paletteId` again every time `api.theme.paletteId` changes), `api-denied` (declares no permissions; on load calls `api.run`, `api.config.value`, `api.config.set`, `api.log.recent`, and `api.modules.settingsOf`, logging each result)
- [X] T006 [P] Create `data/schema/module.schema.json`: draft-07 JSON Schema for the manifest in `specs/002-modules/contracts/module-manifest.md` (required `id`, `name`, `version`, `api`; patterns and limits as specified; `permissions` enum; `settings` rules; `helpers`; the three reserved fields typed as any; `additionalProperties: true`)
- [X] T007 [P] Write `scripts/tests/config_modules_test.go` (black-box, against `config`): with `modules` definitions in the input, `resolve` returns `values.modules.<id>` with `enabled` (default from `defaultEnabled`) and every declared setting with defaults filled in and matching `sources`; user overrides win; an invalid value (wrong type, outside enum, below min, above max, non-integer for integer) is reported as `{key: "modules.<id>.<key>", problem, fallback}` and only that key falls back; settings of ids not in the list are neither validated nor reported and are preserved by `set`/`unset`; `set` accepts declared keys and `modules.<id>.enabled` in one atomic write and refuses unknown ids or undeclared keys with `invalid_key` and a changed-nothing file; `unset` removes keys and empty `modules.<id>` parents; types `number` and `boolean` work; input without `modules` behaves exactly like Foundation (rerun a few Foundation assertions); plus a randomized sequence test (fixed seed, 40 operations of `set`/`unset` over declared settings and `modules.<id>.enabled`, with an undeclared entry for an uninstalled module in the file from the start) after which the file holds exactly the values the test set, no empty objects, and the uninstalled module's entry intact (SC-006)
- [X] T008 Extend `scripts/src/config/schema.go`: add rule kinds `number` and `boolean`, a type for module definitions (`id`, `defaultEnabled`, `settings` map of rule fields), and `rulesFor(modules)` that returns the core rules plus a rule per `modules.<id>.<key>` and the boolean `modules.<id>.enabled`; make `findRule`, `check`, and `defaultsTree` work from that combined set instead of the package-level list
- [X] T009 Extend `scripts/src/config/resolve.go` to read the optional `modules` input, validate `modules.<id>.*` per key like core keys (dropping and reporting invalid values with the module default as fallback), skip unknown ids and undeclared keys, and return `values.modules.<id>` and `sources`; core behavior unchanged
- [X] T010 Extend `scripts/src/config/write.go` so `set` and `unset` accept `modules.<id>.<key>` and `modules.<id>.enabled` for ids in the `modules` input (validated like core keys, refused with `invalid_key` otherwise), keeping the single atomic write and empty-parent removal
- [X] T011 Extend `singletons/Backend.qml`: `run(...)` accepts an optional tag (module id) stored on the request and in error messages; the request gets `cancel()` (sends SIGKILL to the process, marks it finished as `cancelled` without calling the callback or emitting `helperFinished`); `lastErrors` is keyed by `<tag>:<helper>` when a tag is set so a module's failures do not overwrite another's; the new `cancelled` outcome is silent (no callback, no `helperFinished`) and is documented in `docs/module-api.md` (T034) as an addition to Foundation's helper-run states
- [X] T012 Run `make fmt lint test` and fix everything so Foundation's tests and the new config tests pass

**Checkpoint**: The `config` helper understands module settings; fixtures, the manifest schema, and cancellable helper runs exist.

---

## Phase 3: User Story 1 - Write a module and see it load (Priority: P1) 🎯 MVP

**Goal**: Modules are discovered in the bundled and user directories, validated, listed, and loaded when enabled; broken ones are listed with a clear reason.

**Independent Test**: Copy `examples/hello-module` into the user modules directory, start the shell, confirm it is listed as disabled with its permissions, enable it with `{"modules":{"hello":{"enabled":true}}}` in the config file, and confirm it loads and its window appears (quickstart step 3).

### Tests for User Story 1

- [X] T013 [P] [US1] Write `scripts/tests/modules_scan_test.go` (black-box, against `modules scan`): `good-min` and `good-full` are valid with `defaultEnabled` true for bundled and false for user; every `err-*` fixture is invalid with an error naming the right field (`id`, `version`, `api`, `main`, `permissions`, `settings.<key>`, `helpers[n].path`, and field `""` for missing or broken files) and the module's id (or `dir:<name>`); `err-newer-minor` and `err-other-major` give the required/available message; `warn-reserved` is valid with one warning per reserved field (`<field> is not supported in module interface 1.0`) and one `unknown field` warning; `dup-a`/`dup-b` in the same directory (first by directory name wins) and one bundled + one user copy (bundled wins) produce a loser with an error naming both directories; a directory named `bad name!` without a usable manifest is listed as `dir:bad_name_` with an error; ordering is bundled first, then user, each sorted by directory name; a missing, empty, or nonexistent `userDir` is not an error and nothing is created; a manifest over 64 KiB (generated in the test), over 64 settings, over 16 permissions, and over 8 helpers are rejected; the scan never changes any file (compare directory listings before and after)
- [X] T014 [P] [US1] Write `scripts/tests/modules_manifest_schema_test.go`: every fixture that `modules scan` calls valid also validates against `data/schema/module.schema.json` (implement a tiny checker in the test for the schema keywords actually used: type, required, pattern, enum, minimum, maximum, maxLength, maxItems), and a few of the invalid ones fail it; every permission name accepted by the program is in the schema enum and in the table in `specs/002-modules/data-model.md`

### Implementation for User Story 1

- [X] T015 [US1] Implement `scripts/src/modules/main.go`: dispatch (`scan`, later `schema`), JSON stdin/stdout envelope (`ok`, `protocol`, `data`/`error`), ndjson logging to stderr with `source: "helper:modules"` filtered by `LUMEN_LOG_LEVEL`, bad-usage exit 2 for a missing or unknown subcommand
- [X] T016 [US1] Implement `scripts/src/modules/manifest.go`: the manifest and setting-rule types; parsing with `json.Decoder` (numbers kept exact); every rule from `specs/002-modules/contracts/module-manifest.md` (id/name/version/api/main/permissions/settings/helpers limits, reserved and unknown fields as warnings, setting defaults must satisfy their rule, reserved setting key `enabled`, interface compatibility from an `apiVersion` argument); returns `[]problem{field, problem}` for errors and warnings
- [X] T017 [US1] Implement `scripts/src/modules/scan.go`: read `bundledDir` and `userDir` (skip hidden entries and plain files, tolerate a missing `userDir`, cap at 256 directories), read each `module.json` (reject over 64 KiB before parsing), validate through `manifest.go`, check that the entry file exists inside the module directory, check each helper path (relative, no `..`, regular file, executable bit), apply the duplicate-id rule (bundled first, then lower directory name), set `defaultEnabled` from the origin, set `id` to `dir:` plus the directory name with every character outside `[A-Za-z0-9._-]` replaced by `_` and cut to 64 characters when the manifest id is unusable (with an error when the name had to change), and return the list per `specs/002-modules/contracts/helper-modules.md`; never writes
- [X] T018 [US1] Create `singletons/ModuleApi.qml` as a normal type (not a singleton; register it in `singletons/qmldir` as `ModuleApi 1.0 ModuleApi.qml`) with the core members only: `id`, `name`, `version`, `apiVersion`, `theme` (the Theme singleton), `style` (the Style singleton), and `log` with `debug/info/warn/error(msg, fields)` that call `Log.write(level, "module:" + id, msg, fields)`
- [X] T019 [US1] Create `singletons/Modules.qml` (`pragma Singleton`): properties `apiVersion` (`"1.0"`), `list` (records per `specs/002-modules/data-model.md`), `ready`; `scan()` runs `Backend.run("modules", ["scan"], {bundledDir: Quickshell.shellDir + "/modules", userDir: <LUMEN_USER_MODULES_DIR or $XDG_DATA_HOME/lumen/modules>, apiVersion}, …)` and builds the records (state `failed` for invalid ones); after a scan it gives `Config` the definitions of the valid modules, and on every `Config.reloaded` it applies `modules.<id>.enabled` (loads newly enabled modules, unloads newly disabled ones, sets state `disabled` or `loaded`); `load(record)` uses `Qt.createComponent("file://" + main + "?r=" + counter)`, on `Component.Error` sets state `failed` with the trimmed `errorString()`, otherwise creates the object with `{api: <ModuleApi for the record>}` (state `failed` if `createObject` returns null), keeps the component, object, and API for `unload(record)`, which destroys all three
- [X] T020 [US1] Extend `singletons/Config.qml`: property `moduleDefs` (array), function `setModules(defs)` that stores them and calls `resolve()`; include `"modules": moduleDefs` in the input of every `config` helper call (`resolve`, `set`, `unset`); keep `values.modules` in `values`
- [X] T021 [US1] Register `Modules` in `singletons/qmldir`; in `shell.qml`, reference `Modules` in `Component.onCompleted` (so it exists at startup) and call `Modules.scan()` once the startup checks pass (in the same place that today sets `checksPassed`); keep the existing test surface loading unchanged for now
- [X] T022 [P] [US1] Create the first version of the example module: `examples/hello-module/module.json` (id `hello`, version `1.0.0`, api `1.0`, permissions `["helpers"]`, a `greeting` string setting and a `size` integer setting 1–5, helper `hello` at `bin/hello`) and `examples/hello-module/Main.qml` (a small `FloatingWindow` titled "Hello" whose background is `api.theme.background`, text `api.theme.onBackground`, spacing and font from `api.style`, text "Hello", and an `api.log.info` line on load); `examples/hello-module/bin/hello` as a shell script (then `chmod +x`) that reads stdin and prints `{"ok":true,"protocol":1,"data":{"message":"hello from the helper"}}`
- [X] T023 [US1] Add a module section to `scripts/tests/smoke.sh` (functions that use `LUMEN_USER_MODULES_DIR` and `LUMEN_CONFIG_FILE` in a temp dir): copy `examples/hello-module` there, start the shell, assert the shell log shows a `modules` scan with `hello` not loaded (no hello window log line), write `{"modules":{"hello":{"enabled":true}}}` to the config file, and assert within 1 s that the module's load log line appears; remove the override and assert it unloads
- [X] T024 [US1] Run `make fmt lint test smoke` and quickstart step 3 manually (rescan and enable by editing the config file); fix everything until it passes with zero warnings

**Checkpoint**: US1 works on its own: discovery, validation, listing (log and registry), enabled loading and unloading.

---

## Phase 4: User Story 2 - Use the shared services through the public interface (Priority: P1)

**Goal**: A module reaches Theme, Style, its settings, Log, helper runs, and the module list only through `api`; permissions are checked; Foundation's test surface becomes a bundled module.

**Independent Test**: Load `examples/hello-module`, switch themes and watch it follow live; run `scripts/tests/smoke.sh` module checks for `api-probe` and `api-denied`; `make test` shows the test-surface module uses only documented API members (quickstart steps 2 and 4).

### Tests for User Story 2

- [X] T025 [P] [US2] Write `scripts/tests/module_api_usage_test.go`: parse `modules/test-surface/Main.qml` and every `Main.qml` under `examples/` for `api\.[A-Za-z_][A-Za-z0-9_.]*` references, normalize them (drop call parentheses and indexes), and assert each begins with a member documented in `specs/002-modules/contracts/module-api.md` (SC-010); fail if the bundled module references `Backend`, `Config`, `Modules`, `Log.`, or any singleton name directly
- [X] T026 [P] [US2] Extend `scripts/tests/smoke.sh` with the API checks: enable `api-probe` and assert the probe log lines show each member working (`theme`, `style`, `settings.values`, `modules.list` with manifest, `log.recent`, `config.value`, a helper run with state `ok`); enable `api-denied` and assert each refused call is logged as `module api-denied was refused <call>: permission <name> not declared` with source `module:api-denied`, that the module is still `loaded`, and that its `api.run` result had state `denied`; assert module log lines carry source `module:<id>`; run `lumen theme tokyo-night` and then `lumen theme catppuccin-latte` and assert `api-probe` logs the new `paletteId` within 1 s each time (SC-004)
- [X] T027 [P] [US2] Update `scripts/tests/no_hardcoded_colors_test.go`, `scripts/tests/no_magic_values_test.go`, and `scripts/tests/no_network_test.go` so they scan `modules/*/*.qml` and `examples/*/*.qml` instead of `testsurface/*.qml` (keep their other file lists)

### Implementation for User Story 2

- [X] T028 [US2] Extend `singletons/ModuleApi.qml` with the remaining members from `specs/002-modules/contracts/module-api.md`: `settings` (`values`, `errors`, `set`, `reset` bound to `Config.values.modules[id]`, `Config.errors` filtered by `modules.<id>.`), `modules.list` (read-only copies without internals, including `manifest`), `config` (`values` without the `modules` subtree, `errors`, `value(key)`, `set`, `reset`, guarded by `config.read` and `config.write`), `log.recent()` (guarded by `log.read`), `modules.settingsOf(id)` (guarded by `modules.read-settings`), and a private `denied(call, permission)` that logs `module <id> was refused <call>: permission <permission> not declared` at warn level with source `module:<id>` and returns the harmless result
- [X] T029 [US2] Add `run(helper, input, timeoutMs, callback)` to `singletons/ModuleApi.qml`: requires permission `helpers` and a `helpers` entry in the manifest with that name (otherwise calls `callback` with a request-like object `{state: "denied", error: {...}}` and logs the refusal), resolves the path from the module directory, calls `Backend.run(path, [], input, timeoutMs, wrapped, moduleId)`, records the live request so the API object can `cancel()` all of them when the module is unloaded, and ensures helper failures are reported with both the module id and the helper name
- [X] T030 [US2] In `singletons/Modules.qml` unload path: call the API object's `cancelAll()` before destroying the object, so pending helper runs are killed and their callbacks never fire after unload
- [X] T031 [US2] Create `modules/test-surface/module.json` (id `test-surface`, name "Test surface", version `1.0.0`, api `1.0`, permissions `["config.read", "log.read"]`) and `modules/test-surface/Main.qml` by moving the content of `testsurface/TestSurface.qml` into a `Scope { required property var api … FloatingWindow { … } }` that uses only `api.*` (theme roles from `api.theme`, tokens from `api.style`, config values and errors from `api.config`, the log from `api.log.recent()`); replace its Backend section with a Modules section that lists `api.modules.list` with state and last error
- [X] T032 [US2] Delete `testsurface/TestSurface.qml` (and the empty `testsurface/` directory) and remove its loader from `shell.qml`; keep the startup checks; run `qmllint`/`qmlformat` over the new files via `make fmt lint`
- [X] T033 [US2] Extend `examples/hello-module/Main.qml` to use the helper (`api.run("hello", {}, 1500, cb)` on load, showing `data.message` or the failure state), `api.settings.values.greeting` and `size` (window text and size from them via `api.style`), and `api.theme` live; keep every color from `api.theme` and every size from `api.style`
- [X] T034 [US2] Write `docs/module-api.md`: the reference for interface 1.0 (copy the member table, the permission table, helper run behavior, rules for authors, the trust model, the reload limit for secondary QML files, and a migration-note template for interface changes); also state the `cancelled` helper-run outcome, and that path checks for `main` and helper paths are lexical (a symlink inside a module directory can point outside, which the trust model already covers) with a link to the contract
- [X] T035 [US2] Run `make fmt lint test smoke` and quickstart steps 2 and 4 manually (themes follow live in both the test surface and hello); fix everything until it passes with zero warnings

**Checkpoint**: US1 and US2 work: modules look and behave like Lumen, built-ins use the same interface.

---

## Phase 5: User Story 3 - Module settings in the user's config file (Priority: P2)

**Goal**: Module settings are declared in the manifest, merged into a published schema, validated, applied live, and reset through the same file as everything else.

**Independent Test**: With hello enabled, edit its `greeting` in the config file and watch it update; set an invalid `size`; remove the override (quickstart step 5).

### Tests for User Story 3

- [X] T036 [P] [US3] Write `scripts/tests/modules_schema_test.go` (against `modules schema`): the output contains the core schema keys unchanged and `properties.modules.properties.<id>` for every valid module (boolean `enabled` plus each declared setting with type, enum, min/max, default, description); invalid modules contribute nothing; the result is valid JSON Schema as far as the test's tiny checker can tell
- [X] T037 [P] [US3] Write `scripts/tests/module_rules_agree_test.go`: for a table of rules (every type, enum, min/max, boundary values) and candidate values (good and bad), assert that `modules scan` accepts exactly the manifests whose `default` satisfies the rule, and that `config resolve` accepts exactly the same values for that rule (generated into temp module directories), so the two implementations cannot drift
- [X] T038 [P] [US3] Extend `scripts/tests/smoke.sh` with the settings checks: edit `modules.hello.greeting` in the config file and assert the module log shows the new value within 1 s; set `modules.hello.size` to 99 and assert a warning naming `modules.hello.size`, that hello stays loaded, and that another loaded module is unaffected; reset through the controller-independent path (edit the file) and assert the fallback to the default; assert the file never gains keys the test did not write and that an unrelated `modules.gone.x` entry survives; assert the file is unchanged by `rescan`/`reload`

### Implementation for User Story 3

- [X] T039 [US3] Implement the `schema` subcommand in `scripts/src/modules/main.go` and a new `scripts/src/modules/schema.go`: read `dataDir/schema/config.schema.json`, run the same scan as `scan`, and add `properties.modules.properties.<id>` objects for every valid module (`additionalProperties: true`), mapping setting rules to JSON Schema keywords; return `{schema}`
- [X] T040 [US3] Make `ModuleApi.settings` live and complete in `singletons/ModuleApi.qml`: `values` re-evaluates on `Config.reloaded`, `errors` lists only this module's settings problems (key without the prefix, problem, fallback), `set`/`reset` call `Config.set`/`Config.reset` with the full `modules.<id>.<key>` and reject undeclared keys with a logged refusal (no permission needed for own settings)
- [X] T041 [US3] Make Config errors for module keys visible: in `singletons/Modules.qml` expose per-record `settingErrors` from `Config.errors` entries whose key starts with `modules.<id>.`, and show them in the module details (used by `lumen modules show` in US5 and by the test-surface Modules section)
- [X] T042 [US3] Add a "Settings" section to `docs/module-api.md`: declaration in the manifest, namespacing under `modules.<id>`, the reserved `enabled` key, reset semantics, unknown keys preserved, and how to print the merged schema for editors by hand (`echo '{"bundledDir":"modules","userDir":"","apiVersion":"1.0","dataDir":"data"}' | scripts/bin/modules schema`)
- [X] T043 [US3] Run `make fmt lint test smoke` and quickstart step 5 manually; fix everything until it passes with zero warnings

**Checkpoint**: US1–US3 work: modules are configurable like everything else.

---

## Phase 6: User Story 4 - A broken module only breaks itself (Priority: P2)

**Goal**: Every kind of module failure is contained, reported with the module id, and recoverable by reload; runtime errors are attributed and an error storm unloads the module.

**Independent Test**: Enable the whole set of broken fixtures next to a good one and confirm the shell and the good module keep answering while each failure is listed with a clear reason (quickstart step 6).

### Tests for User Story 4

- [X] T044 [P] [US4] Extend `scripts/tests/smoke.sh` with the isolation matrix: copy `qml-good-window` and the failure fixtures into the user directory, enable all of them, and assert for each of the eight failure kinds (`err-bad-json`, `err-missing-main`, `err-newer-minor`, `dup-a`+`dup-b`, `qml-syntax-error`, `qml-runtime-once`, `helper-fails`, `helper-hangs`) that: its log entry names the module id and a clear reason, the good `qml-good-window` module is `loaded`, and `lumen version` answers in under 500 ms throughout; assert the hanging helper is killed within its limit and named, the runtime-once module stays `loaded` with `runtimeErrors` 1, and `qml-error-storm` is unloaded after three errors within 5 s and stays unloaded for 10 s
- [X] T045 [P] [US4] Extend `scripts/tests/smoke.sh` with the recovery and memory checks: after fixing a copied failed module (replace its `Main.qml`) a reload brings it to `loaded` without a shell restart (use the `moduleReload` path through `qs ipc call` directly until US5 adds the command); and a `--modules-cycles N` option that enables and disables the `qml-good-window` module N times and asserts resident memory after the last cycle is within 10% of the memory after the first (SC-007)

### Implementation for User Story 4

- [X] T046 [US4] In `singletons/Modules.qml`, own one log follower: a `Process` running `qs -p <Quickshell.shellDir> log --follow` started before the first module loads and restarted (after a short delay) if it exits while the shell runs; parse each line, strip colors, accept only lines of the exact form `WARN|ERROR scene: file://<dir>/…` where `<dir>` is the directory of a known module (longest prefix match), ignore everything else (including the shell's own `qml:` lines, to avoid a feedback loop), and for each match increment that record's `runtimeErrors`, set `lastError` to the message, and write a Log entry (level warn, source `module:<id>`, message including the file and line). If the follower cannot start (for example `qs` is not in PATH) or exits three times in a row within 10 s, log one warning, stop restarting it, set `Modules.attribution` to `"degraded"` (otherwise `"active"`), and carry on: load errors and helper failures are still attributed, runtime errors are not (Principle IV)
- [X] T047 [US4] Add the error-storm rule in `singletons/Modules.qml`: if a module's third attributed error arrives within 5 s of its `loadedAt`, unload it, set state `failed` with `lastError` "3 runtime errors within 5 s of loading; unloaded", log it at error level, and never load it again until `reload` or `enable` is requested (store a `blocked` flag cleared by those actions)
- [X] T048 [US4] Make failure handling complete in `singletons/Modules.qml`: any exception while loading a module (component error, `createObject` returning null, a thrown error in `load`) is caught, recorded as `lastError`, logged with the module id, and leaves every other record untouched; helper failures and timeouts reported through `ModuleApi.run` include `module <id>` and the helper name in the logged message; a failed record keeps its previous object destroyed (no half-loaded state)
- [X] T049 [US4] Implement the reload path in `singletons/Modules.qml`: `reload(id)` cancels and destroys the old object and API, runs `modules scan` for all directories, replaces only that record from the new result (keeping `runtimeErrors` reset and `blocked` cleared), and loads it again if it is enabled and valid; unknown id returns a result the IPC layer can report. `reload` completes (calls its completion callback) only after the module's state has settled to `loaded`, `failed`, or `disabled`. If the module's directory is gone from the new scan, the record is marked `failed` with `lastError` "module directory is missing" and is removed from the list on the next rescan
- [X] T050 [US4] Run `make fmt lint test smoke` and quickstart step 6 manually; fix everything until it passes with zero warnings

**Checkpoint**: US1–US4 work: a broken module can only break itself.

---

## Phase 7: User Story 5 - See and control modules from the terminal (Priority: P3)

**Goal**: `lumen modules list|show|enable|disable|reload|rescan` against a running shell, with clear exit codes.

**Independent Test**: Run each command against a running shell with good and broken modules and check output and exit codes (quickstart step 7).

### Tests for User Story 5

- [X] T051 [P] [US5] Extend `scripts/tests/fixtures/qs-fake.sh` with replies for `modulesList`, `moduleShow` (a known id returns a detail JSON from `$FAKE_QS_MODULES_DIR/<id>.json`, an unknown id returns `error: unknown: unknown module '<id>'`), `moduleEnable`, `moduleDisable`, `moduleReload`, and `modulesRescan` (reply `queued 7`, then the existing status sequence), keeping every existing behavior
- [X] T052 [P] [US5] Write `scripts/tests/controller_modules_test.go` against the fake `qs`: `list` prints one line per module with id, version, origin, state, and the first error or the permissions for disabled ones; `show <id>` prints the detail fields; `enable`/`reload` exit 0 for a loaded result and 6 for a failed result (the controller calls `moduleShow` after `done`); `disable`, `rescan` exit 0; unknown module exits 7 for every command that takes an id; a bad id (including `dir:` with characters outside `[A-Za-z0-9._-]`) or a missing argument or an unknown subcommand exits 2 without contacting `qs`; no shell exits 3; protocol mismatch exits 5; `error: failed:` exits 6; all finish in under 1 s; the controller source still has no non-stdlib imports and never mentions configuration files

### Implementation for User Story 5

- [X] T053 [US5] Extend `singletons/Backend.qml`'s `IpcHandler` with `modulesList()`, `moduleShow(id)`, `moduleEnable(id)`, `moduleDisable(id)`, `moduleReload(id)`, `modulesRescan()` per `specs/002-modules/contracts/ipc-modules.md`: read functions answer from `Modules.list` immediately (JSON text; unknown id → `error: unknown: unknown module '<id>'`; unsafe id → `error: invalid: …`), queued functions use `beginCommand`/`endCommand` and return `queued <seq>`; map `unknown:` and `invalid:` prefixes in `endCommand`; the id check accepts the manifest id pattern and `dir:` followed by `[A-Za-z0-9._-]{1,64}` (the same pattern as the controller); `modulesList` also returns `"attribution"` (`active` or `degraded`, from `Modules.attribution`) and `moduleShow` includes `settingErrors`
- [X] T054 [US5] Implement `enable`, `disable`, and the enabled-override rule in `singletons/Modules.qml`: `setEnabled(id, bool)` computes whether the choice differs from `defaultEnabled`, then calls `Config.set("modules.<id>.enabled", value)` or `Config.reset(...)` accordingly, loads or unloads on the resulting `Config.reloaded`, and completes the queued command only after the module's state has settled (`loaded`, `failed`, or `disabled`), never merely when the config write finishes; `rescan()` runs `scan()` without writing anything, completes after the scan result is applied, and keeps loaded modules loaded (a loaded module whose directory is gone stays listed as `loaded` with the warning `directory missing` until reload or restart; any other module whose directory is gone is removed from the list)
- [X] T055 [US5] Extend `scripts/src/lumen/main.go` with the `modules` command group: parse and validate arguments first (exit 2), then connect as for `theme`; `list`/`show` call the read functions and print the formatted table or detail text from the JSON reply; `enable`, `disable`, `reload`, `rescan` call the queued functions and wait with the existing status polling; after `enable`/`reload` call `moduleShow` and print `<id>: <state>` (exit 6 with the error text if the state is `failed`); map `error: unknown:` and a `failed … unknown:` status to exit 7; print a note when `modulesList` reports `attribution: degraded`; validate ids with the pattern from the contract (manifest id, or `dir:` followed by `[A-Za-z0-9._-]{1,64}`); update the usage line; stdlib only
- [X] T056 [US5] Extend `scripts/tests/smoke.sh` with the command checks against a real shell: `list` shows the bundled test surface as loaded and a copied user module as disabled with its permissions; `enable hello` exits 0 in under 1 s and the file gains `modules.hello.enabled: true`; `disable hello` removes it again (default off); disabling a bundled module writes `enabled: false` and enabling it removes the override; `rescan` picks up a newly copied module without writing the file; `reload` of a failed copied module after fixing it exits 0; unknown id exits 7; bad id exits 2; with the shell stopped, `list` exits 3; the state read with `show` immediately after `enable`/`reload` returns is already final (never mid-load); a module directory deleted while loaded stays `loaded` with `directory missing` until `reload`, which then marks it failed
- [X] T057 [US5] Update `specs/001-foundation/contracts/cli.md`'s exit-code table to mention code 7 (unknown module) with a pointer to `specs/002-modules/contracts/cli-modules.md`, and add the new functions to a short "Module commands" note in `specs/001-foundation/contracts/ipc-protocol.md` pointing at `specs/002-modules/contracts/ipc-modules.md`
- [X] T058 [US5] Run `make fmt lint test smoke` and quickstart step 7 manually; fix everything until it passes with zero warnings

**Checkpoint**: All five stories work.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T059 [P] Write `docs/writing-a-module.md`: the author guide that walks through `examples/hello-module/` step by step (directory, manifest fields, entry file, settings, helper program, enabling, reloading, reading errors in `lumen modules show`), the rules for authors, the trust model, and the secondary-file reload limit; link it from `README.md`
- [X] T060 [P] Update `docs/layout.md` (where modules, examples, manifests, and the `modules` program go; the sixth singleton), `docs/dependencies.md` (`LUMEN_USER_MODULES_DIR`, `qs log` as a requirement of the Modules singleton), `docs/development.md` (the module interface and manifest versioning rules, the migration-note template, how to run `modules scan` and `modules schema` by hand), and `README.md` (module commands and the new layout entries)
- [X] T061 [P] Add `scripts/tests/no_telemetry_modules_test.go`: the `modules` program and the module QML files (`singletons/Modules.qml`, `singletons/ModuleApi.qml`, `modules/*/Main.qml`, `examples/*/Main.qml`) contain no network imports or URLs (reuse the Foundation patterns), and `modules scan` never creates `userDir`
- [X] T062 Measure and record in `docs/development.md`: the time from `lumen modules enable hello` to the log line "loaded" and the time from `lumen start` to that line for an already-enabled module (SC-002), a 100-cycle run of `scripts/tests/smoke.sh --modules-cycles 100` (SC-007), and average CPU over 5 minutes with 10 idle copies of the example module enabled under different ids (SC-009; script the copying with a loop in a temp user directory)
- [X] T063 Time SC-001 as far as it can be tested: starting from an empty temp user directory and only `docs/writing-a-module.md` plus `examples/hello-module/`, create a new module from the guide (not a copy-paste of the finished example), enable it, and confirm it appears; record any gaps in the guide and fix them
- [X] T064 Run the whole `specs/002-modules/quickstart.md`, then `make fmt lint test smoke`, and confirm the Foundation quickstart (`specs/001-foundation/quickstart.md`) still passes; fix remaining issues

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies.
- **Foundational (Phase 2)**: depends on Setup; blocks all stories.
- **US1 (Phase 3)**: after Foundational.
- **US2 (Phase 4)**: after US1 (needs `Modules.qml`, `ModuleApi.qml`, and loading).
- **US3 (Phase 5)**: after US2 (extends `ModuleApi.settings`); `modules schema` (T039) only needs US1.
- **US4 (Phase 6)**: after US1; the helper-failure parts need US2's `api.run`.
- **US5 (Phase 7)**: after US1; `enable`/`disable` write config, so it also needs Foundational's config support. It can run in parallel with US3 and US4.
- **Polish (Phase 8)**: after the stories you want to ship.

### Within Each Story

- Tests first, and they should fail before implementation.
- Go program first, then the QML that calls it, then the smoke checks, then the manual quickstart step.

### Parallel Opportunities

- T003 in Setup.
- T004–T007 in Foundational (different files); T008–T010 are sequential edits to the `config` program.
- T013 and T014 (US1 tests), T025–T027 (US2 tests), T036–T038 (US3 tests), T044 and T045 (US4 tests), T051 and T052 (US5 tests).
- T022 (example module) can be written while T015–T017 are in progress.
- T059–T061 in Polish.
- `scripts/src/modules/` (US1, US3) and the QML singletons can be built by different people since they meet only at the contract.

## Parallel Example: User Story 1

```bash
# Tests together:
Task: "scripts/tests/modules_scan_test.go"
Task: "scripts/tests/modules_manifest_schema_test.go"

# Go files of the modules program together, once main.go exists:
Task: "scripts/src/modules/manifest.go"
Task: "scripts/src/modules/scan.go"
```

## Implementation Strategy

### MVP First

1. Phase 1 Setup, Phase 2 Foundational.
2. US1: discovery, validation, loading by enabling in the config file. **Stop and validate** (quickstart step 3).
3. US2 (also P1): the public interface and the converted test surface. US1 and US2 together are the MVP: modules can be written and behave like Lumen.

### Incremental Delivery

1. Add US3 (settings), validate quickstart step 5.
2. Add US4 (isolation), validate step 6.
3. Add US5 (terminal commands), validate step 7.
4. Polish and measurements.

## Notes

- No shared Go code between programs. The `modules` and `config` programs each carry their own setting-rule check, and `scripts/tests/module_rules_agree_test.go` keeps them in step.
- The controller stays stdlib-only and only calls `qs`.
- The IPC protocol stays at 1: the new functions are additive.
- Spike results in `specs/002-modules/research.md` (R1–R4) are facts about Quickshell 0.3.2, not assumptions: loading with `Qt.createComponent`, cache-busting with `?r=<n>`, and error attribution through `qs log --follow` all work as described there.
- Commit after each task or logical group, using conventional-commit prefixes.
