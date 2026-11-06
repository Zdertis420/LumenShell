# Contract: Shell IPC (protocol version 1)

Transport: Quickshell IPC. One `IpcHandler` in `singletons/Backend.qml` with `target: "lumen"`. The controller calls `qs -p <root> ipc call lumen <function> [args...]`. The instance is chosen by the config path; no pid or instance id is used.

`shell.qml` references every singleton at startup, because QML singletons are created lazily and the handler must exist before the first call.

All functions return a string immediately. A Quickshell `IpcHandler` function returns synchronously, so handlers never wait for async work: they validate, queue the work, and return. Results of queued work are read with `status()`.

## Functions

| Function | Arguments | Returns | Notes |
|----------|-----------|---------|-------|
| `version()` | none | `"1"` | Protocol version. The controller checks it first. |
| `ping()` | none | `"ok"` | Liveness. |
| `setTheme(theme: string, persist: string)` | preset id or `wallpaper`; `"true"`/`"false"` | `"queued <seq>"` or `"error: invalid: <message>"` | `persist=true`: Backend runs one `config set` with both theme keys, then re-resolves. `persist=false` (`--temp`): the shell applies the theme without touching the file. |
| `resetTheme()` | none | `"queued <seq>"` or `"error: failed: <message>"` | Backend runs one `config unset` for the theme keys, clears any temporary theme, and re-resolves. |
| `status()` | none | `"idle"`, `"pending <seq>"`, `"done <seq>"`, or `"failed <seq>: <message>"` | State of the last queued command. `seq` is an integer that increases per queued command. |
| `log(level, source, msg)` | strings | `"ok"` | Controller log forwarding. |

## Controller flow for queued commands

1. Call the function and read `queued <seq>`.
2. Poll `status()` every 50 ms for up to 2 s until it reports `done <seq>` or `failed <seq>: ...`.
3. Map the outcome to an exit code (see [cli.md](cli.md)). A poll timeout is exit 9.

## Module commands

Feature 002 adds the functions `modulesList`, `moduleShow`, `moduleEnable`, `moduleDisable`, `moduleReload`, and `modulesRescan` to this target, and the error prefix `error: unknown:`. Adding functions is a compatible change, so `version()` still returns `"1"`. See `specs/002-modules/contracts/ipc-modules.md`.

## Rules

- Error replies are `error: invalid: <message>` for bad input (controller exit 2) and `error: failed: <message>` for runtime failures (controller exit 6). A `failed <seq>: ...` status maps to exit 6.
- Unknown function: Quickshell returns its own error; the controller maps it to exit 6.
- Unknown theme id: `"error: invalid: unknown theme '<id>'"`.
- Adding a function is backward compatible. Removing or changing the meaning of one bumps `protocol` and needs a migration note in `docs/`.
- A temporary theme (`--temp`) lasts until the shell restarts, `resetTheme` runs, or another `setTheme` runs.
