# Data Model: Modules

## Manifest (`module.json`)

See [contracts/module-manifest.md](contracts/module-manifest.md) for every field. Summary:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `id` | string | yes | `^[a-z][a-z0-9-]{1,31}$` |
| `name` | string | yes | Display name, up to 64 characters |
| `version` | string | yes | The module's own version, `MAJOR.MINOR.PATCH` |
| `api` | string | yes | Interface version it was built for, `MAJOR.MINOR` |
| `description` | string | no | Up to 256 characters |
| `main` | string | no | Entry file inside the module directory, default `Main.qml` |
| `permissions` | string[] | no | From the documented list |
| `settings` | object | no | key → setting rule |
| `helpers` | object[] | no | `{name, path}` helper programs shipped with the module |
| `extensionPoints`, `contributions`, `requires` | any | no | Reserved; accepted, ignored, warned about |

Limits: manifest up to 64 KiB, up to 64 settings, up to 16 permissions, up to 8 helpers.

## Setting rule

`{type, default, enum?, min?, max?, description?}`

| Field | Meaning |
|-------|---------|
| `type` | `string`, `integer`, `number`, or `boolean` |
| `default` | Required. Must satisfy the rule |
| `enum` | Allowed values (strings only) |
| `min`, `max` | Inclusive limits (integer and number only) |
| `description` | Up to 160 characters |

Keys: `^[a-z][A-Za-z0-9]{0,31}$`. The key `enabled` is reserved.

## Permission

| Name | Meaning | Enforced by Lumen? |
|------|---------|--------------------|
| `helpers` | Run the helper programs declared in the manifest | Yes |
| `config.read` | Read core configuration values (not the `modules` subtree) | Yes |
| `config.write` | Set and reset core configuration keys | Yes |
| `log.read` | Read the recent shared log entries | Yes |
| `modules.read-settings` | Read other modules' settings | Yes |
| `network` | The module intends to use the network | No (declaration only; Lumen offers no network API) |

Own settings, own log writing, Theme, Style, and the module list are always allowed.

## Module record (registry entry)

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | From the manifest; `dir:<name>` when the manifest has no usable id |
| `dir` | string | Absolute path of the module directory |
| `origin` | `bundled` \| `user` | |
| `manifest` | object | The parsed manifest (empty when unreadable) |
| `valid` | bool | No manifest errors |
| `errors` | `[{field, problem}]` | Manifest-level problems |
| `warnings` | `[{field, problem}]` | Reserved fields used, unknown fields |
| `defaultEnabled` | bool | `true` for bundled, `false` for user |
| `enabled` | bool | Resolved from `modules.<id>.enabled` |
| `state` | `loaded` \| `disabled` \| `failed` | See below |
| `lastError` | string | Load error text or the last attributed runtime error |
| `runtimeErrors` | int | Count since the last load |
| `loadedAt` | timestamp | Used for the 5-second error-storm window |

## State transitions

```text
discovered ──invalid manifest / version / duplicate──────────────▶ failed
discovered ──valid, enabled = false──────────────────────────────▶ disabled
discovered ──valid, enabled = true──▶ loading ──component error──▶ failed
                                         │
                                         └──object created──────▶ loaded
loaded ──disable / enabled becomes false──────────────────────────▶ disabled
loaded ──3 runtime errors within 5 s of load───────────────────────▶ failed (unloaded)
loaded ──reload──▶ (unload, rescan this module) ──▶ discovered ──▶ …
failed ──reload / enable──▶ (rescan this module) ──▶ discovered ──▶ …
```

A loaded module whose directory disappears stays `loaded` (with the warning `directory missing`) until `reload` turns it into `failed` ("module directory is missing") or the shell restarts. A module that is not loaded and whose directory disappears is dropped from the list on the next scan.

A module that is invalid stays `failed` whether or not it is enabled, so errors are visible before enabling. `disabled` covers both "the user disabled it" and "a user module nobody enabled yet". Nothing moves a module out of `failed` except an explicit user action.

## Module settings values

For each known module, `values.modules.<id>` holds `enabled` plus every declared setting (defaults filled in), and `sources` has an entry per key (`default` or `user`). Errors use the key `modules.<id>.<key>` with the module's default as the fallback. The user file holds only overrides under `modules.<id>`.

## Module definition (passed to the `config` helper)

`{id, defaultEnabled, settings: {key: rule}}` for every valid module. Invalid modules are not passed, so their stored settings are treated like those of uninstalled modules (preserved, not validated).

## Registry (Modules singleton)

| Property | Notes |
|----------|-------|
| `list` | Array of module records (read-only to modules) |
| `apiVersion` | `{major: 1, minor: 0}` |
| `ready` | True after the first scan and apply |
| `attribution` | `active` while the shell follows its own log to count module runtime errors; `degraded` when it could not (`qs` missing or the follower keeps failing). Load errors and helper failures are attributed either way |
| `scan()` / `rescan()` | Runs the `modules` helper, updates records, hands definitions to Config |
| `load(id)` / `unload(id)` / `reload(id)` | One module at a time |

## API object (per module)

Described in [contracts/module-api.md](contracts/module-api.md). One instance per loaded module; it holds the module id and manifest, so every mediated call can be checked and attributed.

## Helper run (addition to Foundation)

Foundation's helper runs end as `ok`, `failed`, or `timeout`. Module runs add `denied` (permission or declaration missing; the helper never starts) and `cancelled` (the module was unloaded; the process is killed and the callback is not called).

## Interface Version

`{major, minor}`, starts at `1.0`. A module loads if `module.major == lumen.major` and `module.minor <= lumen.minor`.
