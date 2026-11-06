# Contract: `lumen` controller

One-shot. Exits after the command completes. Never stays resident. Single file, standard library only. It runs `qs` and nothing else, and it does not read or write config.

## Commands

| Command | Behavior |
|---------|----------|
| `lumen start` | Starts the shell in the foreground (`qs -p <root> -n`). Intended for a niri startup entry or a systemd user unit. Refuses when a shell is already running. |
| `lumen theme <theme>` | Asks the running shell to set the theme and save it as an override in the user file. |
| `lumen theme <theme> --temp` | Asks the shell to apply the theme live only; the file is not written. |
| `lumen theme --reset` | Asks the shell to remove the theme override and apply the result live. |
| `lumen version` | Prints the controller version and, if a shell is running, its protocol version. |

`<theme>` is a preset id (`tokyo-night`, `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `catppuccin-mocha`) or `wallpaper`. Wallpaper scheme and mode are set through config keys (see [config-keys.md](config-keys.md)).

`<root>` is the directory holding `shell.qml`. The controller finds it from its own location (`scripts/bin/../..`) or from `LUMEN_ROOT`.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Unknown command or invalid arguments (including an unknown theme id: shell replies starting `error: invalid:` map here) |
| 3 | Shell is not running |
| 4 | `start`: a shell is already running |
| 5 | Protocol version mismatch with the running shell |
| 6 | The shell failed the command (replies starting `error: failed:` or a `failed` status; message on stderr) |
| 8 | Environment problem (not under niri, `qs` missing, or `scripts/bin/config` / `scripts/bin/palette` missing: "run `make build`") |
| 9 | Command timed out (default 5 s, including the status poll) |

All errors print one clear line on stderr. A valid command finishes in under 1 s (SC-007).

## Behavior notes

- Persisting a theme is done by the shell (Backend runs the `config` helper). With no shell running, `lumen theme` exits 3 and writes nothing.
- Commands that queue work in the shell (`theme`) poll `status()` until the work is `done` or `failed`; see [ipc-protocol.md](ipc-protocol.md).
- The controller logs its own events as ndjson. When a shell is running it forwards them through the `log` IPC function; otherwise it writes to its own stderr.
