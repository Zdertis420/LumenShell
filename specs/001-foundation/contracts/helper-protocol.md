# Contract: helper programs

Standalone programs in `scripts/src/`, built to `scripts/bin/`. Each is short-lived and runnable alone from a terminal, for example:

```text
echo '{"userFile":"/tmp/c.json","dataDir":"data"}' | scripts/bin/config resolve
```

## Common format

- **Input**: one JSON object on stdin (may be `{}`).
- **Output**: exactly one JSON object on stdout: `{"ok": true, "protocol": 1, "data": {...}}` or `{"ok": false, "protocol": 1, "error": {"code": "...", "message": "..."}}`.
- **Diagnostics**: ndjson log lines on stderr (`time`, `level`, `source`, `msg`). Stdout carries nothing else.
- **Exit code**: 0 when a response was written (even `ok:false`); non-zero only for a crash or bad invocation. Backend treats a non-zero exit or unparsable stdout as `failed`.
- **Environment**: `LUMEN_LOG_LEVEL` (`debug`|`info`|`warn`|`error`, default `info`) filters stderr diagnostics at the source. Backend sets it from `log.level`.
- **Limits**: each call finishes within the time limit set by Backend (default from `helper.timeoutMs`). Backend kills overruns.

## `config` program

Subcommands: `resolve`, `set`, `unset`.

### `config resolve`

Input: `{"userFile": string, "dataDir": string, "themeOverride": string?}` (`themeOverride` is the `--temp` theme id).

Data: `{"values": object, "errors": [{"key","problem","fallback"}], "sources": {key: "default"|"preset"|"user"|"temp"}}`.

A missing file gives defaults and no error. Invalid JSON gives defaults plus a single error with key `""` (whole file). Never writes.

### `config set`

Input: `{"userFile": string, "dataDir": string, "changes": [{"key": string, "value": any}, ...]}`.

Data: `{"written": true}`. Validates every change first (preset ids are checked against `dataDir/presets`). If any change is invalid, or the file is invalid, it refuses (error code `invalid_value` or `file_invalid`) and changes nothing. All changes land in one atomic write, unknown keys preserved. Creates Lumen's own config directory if needed, and nothing else.

### `config unset`

Input: `{"userFile": string, "dataDir": string, "keys": [string, ...]}`.

Data: `{"written": bool}` (`false` when none of the keys was present). Removes the keys and any empty parents in one atomic write. Never creates the file.

## `palette` program

No subcommands.

Input: `{"dataDir": string, "source": "wallpaper"|"preset", "preset": string?, "wallpaper": string?, "scheme": string?, "mode": "light"|"dark"?, "hasPrevious": bool}`.

Data: `{"palette": {"id","mode","roles"}, "status": "ok"|"fallback"|"stale", "reason": string?, "adjusted": [role...]}`.

- Presets are read from `dataDir/presets/`.
- The wallpaper source runs matugen with a timeout shorter than the call limit. The executable is found through `PATH`, or through `LUMEN_MATUGEN` when set (used by tests). The command is `matugen image <path> --json hex --type <scheme> --mode <mode> --prefer saturation --dry-run`; the matugen-to-role mapping is in `data-model.md`, and both were verified with matugen 4.2.0 (`docs/development.md`). A request with `detectOnly: true` for the wallpaper source returns only `{wallpaper, wallpaperMtime}` and runs no matugen; wallpaper palettes also report these two fields.
- Every returned palette passes the contrast check. If a palette cannot comply, `status` is `fallback` with the reason, or `stale` when `hasPrevious` is true (Backend then keeps the previous palette and ignores the returned one).

## Backend behavior

- Runs each call as a separate `Process`, writes stdin, collects stdout, and starts a timer for the limit.
- On timeout it kills the process and reports `timeout` with the helper name.
- A program that does not exist or is not executable is reported as `failed` with the program name.
- It forwards stderr lines to `Log` unchanged.
- It never blocks the UI thread; callers get an async result (property change or signal).
- It can run any executable path, which the smoke test uses with fixtures in `scripts/tests/fixtures/` (one that fails, one that hangs).
