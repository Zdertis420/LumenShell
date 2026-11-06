# Contract: shell IPC additions for modules (protocol stays 1)

These functions are added to the existing `lumen` IPC target (see `specs/001-foundation/contracts/ipc-protocol.md`). Adding functions is a compatible change, so `version()` still returns `"1"`. Handlers return at once; work that needs a helper is queued and read back with `status()`, exactly as for `setTheme`.

## Read functions (answer immediately)

| Function | Arguments | Returns |
|----------|-----------|---------|
| `modulesList()` | none | JSON text: `{"attribution": "active" \| "degraded", "modules": [<summary>, ...]}` |
| `moduleShow(id)` | module id | JSON text: `<detail>`, or `error: unknown: unknown module '<id>'` |

`<summary>`: `{id, name, version, origin, state, enabled, defaultEnabled, permissions, runtimeErrors, lastError, warnings}` (`warnings` is a list of strings; `lastError` is a string or empty).

`<detail>`: the summary plus `{description, dir, apiRequired, apiProvided, errors, warnings, settings, settingValues, settingErrors, helpers}`: `errors` and `warnings` as `[{field, problem}]`, `settings` the declared rules, `settingValues` the resolved values, `settingErrors` as `[{key, problem, fallback}]` for this module's setting problems, `helpers` the declared helper names.

`attribution` is `degraded` when the shell could not follow its own log, so runtime errors of modules are not being counted (load errors and helper failures still are).

A module directory with an unusable manifest appears with id `dir:` plus the directory name, where every character outside `[A-Za-z0-9._-]` is replaced by `_` and the name is cut to 64 characters. Ids accepted by these functions are manifest ids or `dir:` followed by `[A-Za-z0-9._-]{1,64}`; anything else is `error: invalid:`.

## Queued functions

| Function | Arguments | Returns | Effect |
|----------|-----------|---------|--------|
| `moduleEnable(id)` | module id | `queued <seq>` | Makes the module enabled (writes or removes the override as needed, FR-023), loads it |
| `moduleDisable(id)` | module id | `queued <seq>` | Makes the module disabled and unloads it |
| `moduleReload(id)` | module id | `queued <seq>` | Unloads it, re-reads its manifest, loads it again if enabled and valid. Writes nothing |
| `modulesRescan()` | none | `queued <seq>` | Scans both directories again: new modules appear, removed ones disappear from the list (see "Directory removed" below), loaded modules stay loaded. Writes nothing |

`status()` reports `pending <seq>`, `done <seq>`, or `failed <seq>: <message>`.

**Completion rule**: `moduleEnable`, `moduleDisable`, and `moduleReload` report `done` only after the module's state has settled to `loaded`, `failed`, or `disabled`; they never report `done` merely because the configuration write finished. `modulesRescan` reports `done` after the scan result has been applied. A module that ends up failed after `moduleEnable` or `moduleReload` still completes the command with `done`; the controller then reads the final state with `moduleShow` and reports the error.

**Directory removed**: a loaded module whose directory has disappeared stays listed as `loaded` with the warning `directory missing` after `modulesRescan`, until `moduleReload` (which marks it `failed` with `module directory is missing`; the next rescan drops it) or a restart. A module that is not loaded and whose directory has disappeared is removed from the list.

## Errors

| Reply | Meaning | Controller exit |
|-------|---------|-----------------|
| `error: invalid: <message>` | Bad argument (for example an id with unsafe characters) | 2 |
| `error: unknown: unknown module '<id>'` | No module with this id | 7 |
| `error: failed: <message>` | Runtime failure | 6 |

`status()` may report `failed <seq>: unknown: …` or `failed <seq>: invalid: …` for the same cases when they are only detected after queuing; the controller maps the prefixes the same way.

## Notes

- Only the controller and tests are expected to call these functions. Modules cannot call IPC.
- Enabling a module whose manifest is invalid saves the choice and reports the module as failed.
