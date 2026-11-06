# Data Model: Foundation

## Palette

A flat set of semantic color roles plus a mode.

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Preset id (`tokyo-night`, `catppuccin-mocha`, ...), `wallpaper`, or `fallback` |
| `mode` | `light` \| `dark` | |
| `roles` | map role → `#RRGGBB` | See role set below |

**Role set (v1)**: `primary`, `onPrimary`, `secondary`, `onSecondary`, `tertiary`, `onTertiary`, `error`, `onError`, `background`, `onBackground`, `surface`, `onSurface`, `surfaceVariant`, `onSurfaceVariant`, `surfaceContainer`, `outline`.

**Validation**: every role present and a valid hex color. Contrast pairs `onX` against `X` (and `onBackground`/`onSurface`/`onSurfaceVariant` against their surfaces) are at least 4.5:1. Violations are adjusted, or the palette is rejected.

## Preset

A Palette stored as `data/presets/<id>.json` with `{id, name, mode, roles}`. Read at runtime from `dataDir`.

## Fallback Palette

A built-in dark Palette (`id: fallback`) compiled into the `palette` helper and mirrored in `data/fallback-palette.json`. `Theme.qml` loads its initial colors from that file, so QML holds no copy. It always passes the contrast check (verified by a test, along with a check that the compiled copy and the file match).

## Matugen Role Mapping (verified with matugen 4.2.0 in T035a)

| Role | matugen color |
|------|---------------|
| `primary`, `onPrimary` | `primary`, `on_primary` |
| `secondary`, `onSecondary` | `secondary`, `on_secondary` |
| `tertiary`, `onTertiary` | `tertiary`, `on_tertiary` |
| `error`, `onError` | `error`, `on_error` |
| `background`, `onBackground` | `background`, `on_background` |
| `surface`, `onSurface` | `surface`, `on_surface` |
| `surfaceVariant`, `onSurfaceVariant` | `surface_variant`, `on_surface_variant` |
| `surfaceContainer` | `surface_container` |
| `outline` | `outline` |

All names exist in matugen 4.2.0's `colors` object, each as `{dark|default|light: {color: "#rrggbb"}}`. See `docs/development.md`.

## Configuration Layers

Resolved in order. Later layers override earlier ones key by key:

1. Built-in defaults (compiled into the `config` helper).
2. Selected preset values (when `theme.source = preset`): the preset's palette and its `mode`. No other keys.
3. User override file.

**Resolved config** (returned by `config resolve`): `{values, errors, sources}`. `values` is the effective nested object. `errors` is a list of `{key, problem, fallback}`. `sources` maps each key to the layer that set it.

## User Override File

`$XDG_CONFIG_HOME/lumen/config.json`. A JSON object holding only explicit overrides.

- Missing file means defaults, and no file is created on read.
- Unknown keys are preserved on rewrite.
- Invalid JSON means defaults plus an error; the file is never modified.
- `config set` and `config unset` write atomically; `unset` deletes the key and removes empty parent objects.

## Config Schema

`data/schema/config.schema.json`, derived from [contracts/config-keys.md](contracts/config-keys.md). Published. The `config` helper validates with its own compiled rules, and a test keeps both in sync.

## Design Tokens (Style)

| Group | Tokens |
|-------|--------|
| Spacing | `xs`, `sm`, `md`, `lg`, `xl` (px) |
| Radius | `sm`, `md`, `lg`, `full` |
| Typography | `family`, size steps `caption`, `body`, `title`, `headline`; weights |
| Motion | duration `fast`, `normal`, `slow`; easing names |

Tokens are constants in `singletons/Style.qml`. Runtime overrides from config are not part of Foundation.

## Singletons (state owned)

| Singleton | Owns | Reads | Interface (summary) |
|-----------|------|-------|---------------------|
| Config | Resolved config, `errors`, `ready` | user file, `config resolve` via Backend | `value(key)`, `errors`, `set(key, v)`, `reset(key)`, `reloaded` signal |
| Theme | Active Palette, `source`, `status` (ok / fallback / stale + reason) | Config `theme.*`, `palette` helper via Backend | color role properties, `mode`, `paletteId`, `status` |
| Style | Design tokens | none | `spacing.*`, `radius.*`, `font.*`, `motion.*` |
| Backend | IPC handler, helper runs, status of the last queued command | none | `run(program, input, timeoutMs)` returning a request with `result` or `error`; IPC functions (including `status()`) in [contracts/ipc-protocol.md](contracts/ipc-protocol.md) |
| Log | Level filter | Config `log.level` | `debug/info/warn/error(msg, fields)` |

## Helper Run (Backend)

States: `running` → `ok` | `failed` (non-zero exit or invalid JSON) | `timeout` (killed). The result carries `{helper, durationMs, data | error}`. Failure is reported with the helper name (FR-021).

## Theme Status

- `ok`: active palette is the requested one.
- `fallback(reason)`: fallback palette in use (matugen missing, failed, or palette rejected, and no previous good palette).
- `stale(reason)`: previous good palette kept (bad or missing wallpaper, rejected palette).

## Protocol Version

Integer `protocol` (starts at 1). Returned by IPC `version`. The controller checks it before sending commands and fails with a distinct exit code on mismatch. Helpers include `"protocol": 1` in every response.
