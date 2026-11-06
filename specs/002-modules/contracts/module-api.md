# Contract: module interface 1.0

This is everything a module is meant to use. Built-in modules use exactly this, and nothing more.

## The entry file

The entry file's root type is Quickshell's `Scope`. Lumen creates it and sets one property:

```qml
import QtQuick
import Quickshell

Scope {
    required property var api

    FloatingWindow {
        color: api.theme.background
        // windows, timers, processes: anything a Scope can hold
    }
}
```

When the module is unloaded, this object and everything inside it are destroyed. Keep state in the object, not in globals.

## The `api` object

| Member | Needs permission | Description |
|--------|------------------|-------------|
| `api.id`, `api.name`, `api.version` | none | From the manifest |
| `api.apiVersion` | none | The interface version Lumen provides, `"1.0"` |
| `api.theme` | none | Theme: the 16 color roles (`primary`, `onPrimary`, …), `mode`, `paletteId`, `status`, `reason`. Colors follow theme changes live; bind to them, never copy hex values |
| `api.style` | none | Style tokens: `spacing`, `radius`, `font`, `motion` |
| `api.log` | none | `debug(msg, fields)`, `info`, `warn`, `error`. Entries carry the source `module:<id>` |
| `api.settings.values` | none | This module's resolved settings (declared keys with defaults filled in). Reactive |
| `api.settings.errors` | none | Problems with this module's settings: `[{key, problem, fallback}]` |
| `api.settings.set(key, value, callback)` | none | Saves an override for one of the module's own keys. `callback(request)` is optional |
| `api.settings.reset(key, callback)` | none | Removes the override |
| `api.modules.list` | none | Read-only array of `{id, name, version, origin, state, enabled, manifest, permissions, errors, warnings, settingErrors, runtimeErrors, lastError}`. `manifest` is the parsed manifest, reserved fields included, so a later host module can discover what others declare. Reactive |
| `api.run(helper, input, timeoutMs, callback)` | `helpers` | Runs one of the module's own declared helper programs. See below |
| `api.config.values`, `api.config.errors` | `config.read` | Core configuration (everything except the `modules` subtree). Reactive |
| `api.config.value(key)` | `config.read` | One core value by dotted key, for example `theme.preset` |
| `api.config.set(key, value, callback)`, `api.config.reset(key, callback)` | `config.write` | Core keys only; validated like any other write |
| `api.log.recent()` | `log.read` | The recent shared log entries |
| `api.modules.settingsOf(id)` | `modules.read-settings` | Another module's resolved settings |

## Mediated calls and refusals

Reading the properties `api.config.values` and `api.config.errors` without `config.read` gives an empty value and is not logged (they are evaluated when the module is created, whether or not it looks at them). A function call that needs a permission the manifest did not declare is refused: nothing happens, Lumen logs `module <id> was refused <call>: permission <name> not declared` (level warn, source `module:<id>`), and the call returns a harmless result (`undefined`, `false`, or, for `api.run`, a request with `state: "denied"`). The module keeps running. Declared permissions are shown to the user in `lumen modules list` and `lumen modules show`.

## Helper runs

`api.run(name, input, timeoutMs, callback)`:

- `name` must be a `helpers` entry in the manifest; otherwise the run is denied.
- The program is run as a separate process with `input` as JSON on stdin; it writes one response document to stdout, as in Foundation's helper contract (`specs/001-foundation/contracts/helper-protocol.md`): `{ok, protocol, data | error}`.
- The run is asynchronous and time-limited (`timeoutMs`, or the default from `helper.timeoutMs`). On a failure or a timeout, the request has `state: "failed"` or `"timeout"`, `error` names the module and the helper, and nothing else is affected.
- `callback(request)` is called once. `request.state` is `"ok"`, `"failed"`, `"timeout"`, or `"denied"`; `request.data` is the helper's `data` when `ok`.
- When the module is unloaded, its pending runs are cancelled (outcome `cancelled`: the process is killed and the callback is not called).
- Path checks for `main` and helper paths are lexical. A symlink inside a module directory can point outside it; this falls under the trust model below.

## Rules for module authors

- Take every color from `api.theme` and every spacing, radius, font size, and duration from `api.style`. Do not hardcode them.
- Do heavy work (parsing, filtering large data) in a helper program, not in QML.
- Do not block the display thread. Lumen cannot interrupt a module that does.
- Make interactive elements operable from the keyboard.
- Runtime errors are written to the shell log and attributed to your module by file path. Three within 5 seconds of loading unload the module and mark it failed.
- Trust model: a module runs inside the shell with the shell's privileges. Permissions limit what the `api` object does for you; they are not a sandbox, and QML can reach system facilities directly. Install only modules you trust.

## Versioning

The interface version is `major.minor`. Adding a capability raises the minor number; a removal or a change of meaning raises the major number and comes with a migration note in `docs/module-api.md`. The reserved manifest fields `extensionPoints`, `contributions`, and `requires` are not part of 1.0.
