# Dependencies

| Dependency | Required? | Used by | Behavior when absent |
|------------|-----------|---------|----------------------|
| niri | Required | Session | The shell logs "not running under niri" and exits. The controller exits with code 8 |
| Quickshell (`qs`) 0.3+ | Required | The shell and the controller. The Modules singleton also runs `qs log --follow` to count module runtime errors | The controller exits with code 8. Without a working `qs log`, runtime errors of modules are not counted (`attribution: degraded`); everything else works |
| Go 1.27+ | Required to build | `make build` | No helper programs; the shell logs "run `make build`" and exits |
| Qt 6 tools (`qmllint`, `qmlformat`) | Dev only | `make lint`, `make fmt` | Those targets fail; nothing else is affected |
| matugen | Optional | `palette` helper, wallpaper source | The fallback palette is used and the reason is logged and shown as Theme `status` |
| EasyEffects or `pavucontrol` | Optional | The sound item of the bar (left click) | The click logs `no audio mixer found`; level, volume and mute still work |
| `notify-send` (libnotify) and a notification daemon | Optional | The notification about dropped list entries of the bar | Only the log entry is written |
| Qt 6 `qmltestrunner` | Dev only | `make test` (QML unit tests of the bar) | `make test` fails with a clear message. On some systems `/usr/bin/qmltestrunner` is the Qt 5 runner, which prints nothing; the Makefile uses `/usr/lib/qt6/bin/qmltestrunner` when it exists |
| `awww` / `swww` | Optional | `palette` helper, wallpaper detection | Set `theme.wallpaper` to a file path, or the fallback palette is used |

Go dependencies: none. All Go programs use only the standard library.

Lumen makes no network connections and sends no telemetry.

## Environment variables

| Variable | Used by | Purpose |
|----------|---------|---------|
| `LUMEN_ROOT` | `lumen` | Directory containing `shell.qml` (default: two levels above the binary) |
| `LUMEN_HELPER_DIR` | shell | Directory with the helper programs (default `scripts/bin`); used by tests |
| `LUMEN_SKIP_SESSION_CHECK` | shell, `lumen` | `1` skips the niri session check (tests only) |
| `LUMEN_USER_MODULES_DIR` | shell | Directory with user modules (default `$XDG_DATA_HOME/lumen/modules`); never created by Lumen |
| `LUMEN_CONFIG_FILE` | shell | Path of the user override file (default `$XDG_CONFIG_HOME/lumen/config.json`); used by tests |
| `LUMEN_MATUGEN` | `palette` | Path to the matugen executable (tests) |
| `LUMEN_LOG_LEVEL` | helpers | Minimum log level (`debug`, `info`, `warn`, `error`) |
