# Lumen

Lumen Shell is a desktop environment shell for [niri](https://github.com/YaLTeR/niri), built with [Quickshell](https://quickshell.org).

This repository is at the **Foundation** stage: the project layout, the five core singletons (Config, Theme, Style, Backend, Log), a one-shot `lumen` controller, and a minimal test surface. It has no end-user components yet.

## Quick start

```text
make build          # builds scripts/bin/{lumen,config,palette}
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
singletons/        Config, Theme, Style, Backend, Log (QML singletons)
testsurface/       Minimal surface for verification only
modules/           Reserved for later features
data/              Presets, fallback palette, config schema
scripts/
├── src/           One directory per Go program (lumen, config, palette)
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
| `awww` or `swww` | Detecting the current wallpaper | Set `theme.wallpaper` yourself, or wallpaper themes use the fallback palette |

Details are in [docs/dependencies.md](docs/dependencies.md).

## Configuration

User overrides live in `$XDG_CONFIG_HOME/lumen/config.json` (usually `~/.config/lumen/config.json`). The file holds only values you set. Defaults are never written into it, and unknown keys are kept. Edits apply live. The schema is `data/schema/config.schema.json`.

## Privacy

Lumen uses no network and sends no telemetry.
