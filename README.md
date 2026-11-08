# Lumen

Lumen Shell is a desktop environment shell for [niri](https://github.com/YaLTeR/niri), built with [Quickshell](https://quickshell.org).

This repository has the project layout, the core singletons (Config, Theme, Style, Backend, Log, and the Modules registry), a one-shot `lumen` controller, the module system, a minimal test surface (which is itself a module), and the first end-user component: the [bar](docs/bar.md).

## Quick start

```text
make build          # builds scripts/bin/{lumen,config,palette,modules}
scripts/bin/lumen start
```

A window titled "Lumen test surface" appears. It shows the active palette, the resolved configuration, helper status, design tokens, and recent log lines.

Change the theme from another terminal:

```text
scripts/bin/lumen theme tokyo-night          # saved to ~/.config/lumen/config.json
scripts/bin/lumen theme catppuccin-latte --temp   # this shell only, nothing written
scripts/bin/lumen theme --reset              # remove the saved choice
```

To start Lumen with your session, run `lumen start` from a niri startup entry or a systemd user unit. Lumen does not restart a crashed shell; your session does.

## Modules

Every part of Lumen is a module: a directory with a `module.json` manifest and a QML entry file. Bundled modules live in `modules/` and load automatically. Modules you add yourself go in `~/.local/share/lumen/modules/`; they are listed with their permissions but stay off until you enable them.

```text
scripts/bin/lumen modules list            # what is there, and in which state
scripts/bin/lumen modules show hello      # details, permissions, settings, errors
scripts/bin/lumen modules enable hello    # load it now and after restarts
scripts/bin/lumen modules disable hello
scripts/bin/lumen modules reload hello    # re-read a module after you edited it
scripts/bin/lumen modules rescan          # look for new module directories
```

```text
scripts/bin/lumen modules call bar focus  # run a module command, here: put keyboard focus in the bar
```

## Bar

The bar has three places (left, center, right) on every screen you choose, and shows the items that you list in the configuration file, like waybar. Items come from modules: the clock, the sound level and the battery are bundled, and any module can add its own. See [docs/bar.md](docs/bar.md) for the settings, the keyboard use, and how to write an item.

```json
{"modules": {"bar": {"edge": "bottom", "left": ["hello-bar-item"], "center": ["bar-clock"], "right": ["bar-sound", "bar-battery"]}}}
```

A broken module only breaks itself: it is listed as `failed` with the reason. Modules run inside the shell with its privileges; permissions are shown, not a sandbox. To write one, follow [docs/writing-a-module.md](docs/writing-a-module.md) and copy `examples/hello-module/`. The interface is described in [docs/module-api.md](docs/module-api.md).

## Commands

| Command | What it does |
|---------|--------------|
| `make build` | Build the Go programs into `scripts/bin/` |
| `make run` | Build, then `lumen start` |
| `make fmt` | `gofmt` and `qmlformat` |
| `make lint` | `gofmt -l`, `go vet`, `qmllint` |
| `make test` | Black-box tests in `scripts/tests/` |
| `make smoke` | End-to-end test against a real shell (needs a niri session) |

## Layout

```text
shell.qml          Quickshell entry point (qs -p .)
singletons/        Config, Theme, Style, Backend, Log, Modules (QML singletons)
modules/           Bundled modules: bar, bar-clock, bar-sound, bar-battery, test-surface
examples/          Example module to copy (hello-module)
data/              Presets, fallback palette, config and manifest schemas
scripts/
├── src/           One directory per Go program (lumen, config, palette, modules)
├── bin/           Build output (git-ignored)
└── tests/         Black-box tests, smoke test, fixtures
docs/              Layout, dependencies, development notes
specs/             Spec Kit specifications, plans, and tasks
```

See [docs/layout.md](docs/layout.md) for where new things belong.

## Dependencies

| Dependency | Needed for | Without it |
|------------|------------|------------|
| niri | The only supported compositor | Lumen reports the problem and exits |
| Quickshell 0.3+ | The shell | Required |
| Go 1.27+ | Building the helper programs | Required to build |
| Qt 6 tools (`qmllint`, `qmlformat`) | `make lint` and `make fmt` | Those targets fail; the shell still runs |
| matugen | Wallpaper-derived palettes | Wallpaper themes use the fallback palette, and the reason is logged |
| EasyEffects or `pavucontrol` | The click on the bar's sound item | The click logs that no mixer was found; the rest of the item works |
| `notify-send` (libnotify) | The notification about dropped bar list entries | Only a log entry is written |
| `awww` or `swww` | Detecting the current wallpaper | Set `theme.wallpaper` yourself, or wallpaper themes use the fallback palette |

Details are in [docs/dependencies.md](docs/dependencies.md).

## Configuration

User overrides live in `$XDG_CONFIG_HOME/lumen/config.json` (usually `~/.config/lumen/config.json`). The file holds only values you set. Defaults are never written into it, and unknown keys are kept. Edits apply live. The schema is `data/schema/config.schema.json`.

## Privacy

Lumen uses no network and sends no telemetry.
