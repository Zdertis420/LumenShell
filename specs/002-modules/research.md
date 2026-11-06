# Research: Modules

Verified locally on Quickshell 0.3.2 with throwaway spikes (2026-11-06). Results marked **Spike** below.

## R1. How a module is loaded

- **Decision**: A module's entry file (default `Main.qml`) is loaded with `Qt.createComponent("file://<abs path>/Main.qml?r=<n>")`, then `createObject(parent, {api: <api object>})`. The entry file's root type is `Scope`, and it declares `required property var api`. The Modules singleton keeps the component and the object, and unloads by `object.destroy()`, `component.destroy()`, and destroying the API object.
- **Spike**: A good file loads (`status Ready`). A syntax error, a missing import directory, and a missing file each give `status Error` with `errorString()` carrying the file, line, and column (`.../Main.qml:3:1: "./does-not-exist": no such directory`). A module with runtime errors still loads, and its errors only reach the log.
- **Rationale**: Windows, timers, and processes are all `QObject`s in a `Scope`, so a module can contain several windows and still be one object that is destroyed at once. `required property` makes a missing API a visible load error.
- **Alternatives**: Quickshell `LazyLoader` or `Loader` (fine for a fixed path, but give less error text and no handle for cache-busting). A Quickshell config per module (separate process, rejected in Complexity Tracking).

## R2. Reloading an edited module

- **Spike**: `Qt.createComponent` caches by URL. After the file was edited, the same URL still produced the old version, while `…Main.qml?r=2` produced the new one.
- **Decision**: Each load uses `?r=<counter>` with a counter that increases on every load. Reload therefore always reads the file again.
- **Known limit**: Other QML files that the entry file uses by relative type name stay cached under their own URLs. Editing only a secondary file needs a shell restart. The author guide says so.
- **Alternatives**: Staging a copy of the module to a unique directory on each reload (busts the cache for all files, costs a copy and a cleanup path). Not worth it for the first version; revisit if authors hit the limit.

## R3. Memory across load and unload

- **Spike**: 300 create/destroy cycles added about 2.4 to 3.7 MB each round, both with a cached URL and with unique URLs, on a 320 MB process. That is about 10 kB per cycle, about 1 MB per 100 cycles (0.3%).
- **Decision**: Use unique URLs freely. SC-007 (within 10% after 100 cycles) holds with a wide margin. The smoke test verifies it with the real singleton.

## R4. Seeing runtime errors of a module

- **Spike**: A failing handler or binding does not stop anything. It prints `WARN scene: file:///…/modules/<dir>/Main.qml[line:-1]: TypeError: …` to the shell's log. QML code cannot intercept it. A `Process` running `qs -p <root> log --follow` receives those lines, and the file path identifies the module. The follower also re-reads lines that the shell writes about itself, so it must ignore everything that does not match the exact form `WARN|ERROR scene: file://<module dir>/…`.
- **Decision**: The Modules singleton owns one `qs log --follow` process (restarted if it ends, started before the first module loads). For each matching line it finds the module by the longest matching directory prefix, increments that module's `runtimeErrors`, stores `lastError`, and writes a Log entry with the module id as source. Three matching lines within 5 seconds of a module's load unload it and mark it failed (FR-019).
- **Degradation**: If the follower cannot start (for example `qs` is not in PATH) or exits three times in a row within 10 seconds, Modules logs one warning, stops restarting it, and sets `attribution` to `degraded`. Load errors and helper failures are still attributed; only counting of runtime errors (and so the error-storm rule) is lost. `lumen modules list` and `show` say so.
- **Rationale**: This is the only way to attribute errors in code the shell does not call. It adds one idle child process, supervised by the shell, which keeps Foundation's rule that Lumen starts no resident process of its own beyond the shell and what the shell supervises.
- **Alternatives**: Wrapping every callback in try/catch (only covers callbacks that go through the API, not bindings or timers). Reading the log file directly (binary format). No attribution (fails FR-017/018 for runtime errors).

## R5. Automatic retry

- **Decision**: Lumen never retries a module by itself. Load errors are deterministic, so retrying cannot help. The "error storm" rule replaces the original "fails three times in a row" wording; the spec was adjusted to match (FR-019, User Story 4 scenario 5). The user reloads with `lumen modules reload <id>`.

## R6. Where manifests are read and validated

- **Decision**: A new Go program, `modules`, with the subcommand `scan`. Input: the two directories and the interface version Lumen provides. Output: one entry per module directory with the parsed manifest, `errors` and `warnings` (module id, field, problem), the origin, whether it is valid, and its default-enabled value. It checks size, JSON validity, field types and limits, id characters, duplicate ids (bundled wins), interface compatibility, the entry file, declared helper paths (inside the module directory, executable), setting rules and defaults, and permission names.
- **Rationale**: Principle I. The same code path serves startup, `reload`, and `rescan`. A second subcommand, `schema`, emits the merged schema (core plus every valid module's settings) for editors.
- **Alternatives**: Validating in QML (violates Principle I). Validating in the `config` helper (mixes concerns).

## R7. Interface version

- **Decision**: The interface version is `"major.minor"`, starting at `1.0`. The source of truth is a property in `Modules.qml`; it is passed to `scan`, which applies the rule (same major, not-higher minor). Manifests use the field `api`. The reserved field names `extensionPoints`, `contributions`, and `requires` produce a warning and nothing else (FR-005a).
- **Rationale**: The name `api` avoids a clash with the reserved `requires`.

## R8. Module settings

- **Decision**: Manifest `settings` is an object of key → rule. A rule has `type` (`string`, `integer`, `number`, `boolean`), `default`, and optional `enum`, `min`, `max`, `description`. Keys are flat (no nesting) and cannot be `enabled`. The `config` helper takes the module definitions as part of every request (`modules: [{id, defaultEnabled, settings}]`), validates `modules.<id>.*` with them, and returns resolved values under `values.modules.<id>`. Settings of modules that are not installed are not validated and are preserved.
- **Rationale**: The helper stays stateless (Foundation rule), and it never rescans disk. The Modules singleton scans first, gives Config the definitions, then Config resolves.
- **Alternatives**: The helper scanning directories itself (couples two programs to the directory layout and costs a scan per resolve). Storing module settings in per-module files (breaks "one file").
- **Consequence**: The `config` helper's rule kinds grow from string and integer to include number and boolean. The core keys do not change.

## R9. Enabled state

- **Decision**: `modules.<id>.enabled` (boolean). The default is `true` for bundled modules and `false` for user modules. Enable and disable write an override only when the choice differs from the default, and remove it when it matches again (FR-023). Both are one `config set` or `config unset` call through `Config`.

## R10. The API object and permissions

- **Decision**: Each loaded module gets its own `ModuleApi` object. Reads that need no permission: Theme roles, Style tokens, the module's own settings, its own `log`, and the module list. Mediated and checked: helper runs (`helpers`, and the helper must be declared), core config reads and writes (`config.read`, `config.write`, core keys only), the log buffer (`log.read`), and other modules' settings (`modules.read-settings`). A refused call logs a warning with the module id, returns a clear result (`state: "denied"` for runs, `undefined` or `false` for others), and the module keeps running. `network` is declarable and shown, but Lumen offers no network API, so it is documentation only.
- **Rationale**: Honest scope (spec Clarifications): the API is mediated; QML can still `import` singletons by path, so this is not a sandbox, and the author guide says so.
- **Alternatives**: A sandbox via separate processes (rejected, see Complexity Tracking). Declaring no permissions (fails the constitution's manifest requirement).

## R11. Module helper programs

- **Decision**: A manifest may list `helpers: [{name, path}]` with paths inside the module directory. `api.run(name, input, timeoutMs, callback)` resolves the name to the declared path and calls `Backend.run` with the usual rules (separate process, JSON on stdin and stdout, time limit, failures named). The helper contract is Foundation's. Runs are tagged with the module id, and `Backend.run` returns a request that can be cancelled; unloading cancels a module's pending runs (the process gets a kill signal and its callback is not called). The outcome is `cancelled`, a state added to Foundation's `ok`, `failed`, `timeout`.

## R12. IPC and the controller

- **Decision**: The shell IPC stays protocol 1: the new functions are an additive, compatible change (`modulesList`, `moduleShow`, `moduleEnable`, `moduleDisable`, `moduleReload`, `modulesRescan`). Queued commands report through `status()` as before. A new error prefix `error: unknown:` maps to a new controller exit code, 7, for "unknown module". `lumen modules list|show|enable|disable|reload|rescan` are added to the stdlib-only controller; after `enable` or `reload` it reads the module back and exits 6 if the module ended up failed, printing the error.
- **Rationale**: Keeps the controller a thin caller of `qs`.

## R13. Discovery directories and timing

- **Decision**: Bundled directory: `<root>/modules` (next to `shell.qml`). User directory: `$XDG_DATA_HOME/lumen/modules`, overridable with `LUMEN_USER_MODULES_DIR` (tests and the example). Scans run at start and on `rescan`/`reload`. There is no directory watching (spec Assumptions). The user directory is never created.

## R14. Converting the test surface

- **Decision**: `testsurface/TestSurface.qml` becomes `modules/test-surface/Main.qml` with a manifest, using only `api.*`. The Backend panel (helper run counts and last errors) is replaced by a Modules panel (the module list with states and errors) because helper internals are not part of the public interface. A test lists the `api.` members that the file uses and checks that each one is documented (SC-010). `shell.qml` stops loading the surface directly and starts the Modules singleton after the startup checks.
- **Rationale**: The built-in module proves the interface is sufficient for a real component.

## R15. Documentation and the example

- **Decision**: `docs/writing-a-module.md` walks through `examples/hello-module/` (a manifest with a setting and a helper, a Main.qml with a small window that follows the theme, the setting, and the helper result). `docs/module-api.md` is the reference. Both state the trust model (no sandbox), the single-file reload limit (R2), and the migration-note template for interface changes.
