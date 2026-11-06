# Contract: Config keys (v1)

Source for `data/schema/config.schema.json`. The user file holds nested JSON. Keys below use dot paths. Unknown keys are allowed and preserved.

| Key | Type | Default | Constraint | Meaning |
|-----|------|---------|------------|---------|
| `theme.source` | string | `"preset"` | `wallpaper` \| `preset` | Palette source |
| `theme.preset` | string | `"catppuccin-mocha"` | one of the bundled preset ids | Used when source is `preset` |
| `theme.scheme` | string | `"scheme-tonal-spot"` | a matugen scheme type | Used when source is `wallpaper` |
| `theme.mode` | string | `"dark"` | `light` \| `dark` | Used for wallpaper palettes |
| `theme.wallpaper` | string | `""` | file path, empty means auto-detect | Wallpaper path override |
| `log.level` | string | `"info"` | `debug` \| `info` \| `warn` \| `error` | Minimum log level |
| `helper.timeoutMs` | integer | `3000` | 500–30000 | Default helper time limit |

## Resolution

1. Defaults (above, compiled into the `config` helper).
2. Preset values: when `theme.source = preset`, the preset supplies the palette and its `mode` (reported as `theme.mode` with source `preset`). No other keys are changed by presets in Foundation.
3. User file.

## `lumen theme` mapping

Each of these is one `config set` / `config unset` call, so the file watcher never sees a half-applied theme.

- `lumen theme <preset-id>` sets `theme.source = "preset"` and `theme.preset = <id>`.
- `lumen theme wallpaper` sets `theme.source = "wallpaper"`.
- `--reset` removes both keys.

## Error report

Each invalid value yields `{key, problem, fallback}`, for example `{"key":"log.level","problem":"must be one of debug, info, warn, error","fallback":"info"}`. Only that key falls back.

## Change rules

Adding a key is compatible. Renaming or removing a key needs a migration note in `docs/`. A test keeps this table, the compiled defaults, and the schema file in sync.
