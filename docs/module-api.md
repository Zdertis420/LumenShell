# Module interface reference (version 1.1)

Everything a module is meant to use. The bundled modules use exactly this and nothing more; a test checks it (`scripts/tests/module_api_usage_test.go`). The formal contract is `specs/002-modules/contracts/module-api.md`; the manifest is described in `specs/002-modules/contracts/module-manifest.md`. For a walkthrough, read [writing-a-module.md](writing-a-module.md).

## The entry file

The entry file's root type is Quickshell's `Scope`. Lumen creates it and sets one property, `api`:

```qml
import QtQuick
import Quickshell

Scope {
    required property var api

    FloatingWindow {
        color: api.theme.background
    }
}
```

When the module is unloaded (disabled, reloaded, or on shutdown), this object and everything inside it are destroyed. Keep state in the object, not in globals.

## Members of `api`

| Member | Needs permission | Description |
|--------|------------------|-------------|
| `api.id`, `api.name`, `api.version` | none | From the manifest |
| `api.apiVersion` | none | The interface version Lumen provides, `"1.1"` |
| `api.theme` | none | The 16 color roles (`primary`, `onPrimary`, `surface`, ...), plus `mode`, `paletteId`, `status`, `reason`. Colors follow theme changes live; bind to them, never copy hex values |
| `api.style` | none | Design tokens: `spacing`, `radius`, `font`, `motion` |
| `api.log` | none | `debug(msg, fields)`, `info`, `warn`, `error`. Entries carry the source `module:<id>` |
| `api.settings.values` | none | This module's resolved settings, defaults filled in. Reactive |
| `api.settings.errors` | none | Problems with this module's settings: `[{key, problem, fallback}]`. Reactive |
| `api.settings.set(key, value, callback)` | none | Saves an override for one of the module's own declared settings |
| `api.settings.reset(key, callback)` | none | Removes that override |
| `api.modules.list` | none | Read-only list of `{id, name, version, origin, state, enabled, manifest, permissions, errors, warnings, settingErrors, runtimeErrors, lastError}`. Reactive |
| `api.run(helper, input, timeoutMs, callback)` | `helpers` | Runs one of the module's own declared helper programs (see below) |
| `api.config.values`, `api.config.errors` | `config.read` | Core configuration (everything except the `modules` subtree). Reactive |
| `api.config.value(key)` | `config.read` | One core value by dotted key, for example `theme.preset` |
| `api.config.set(key, value, callback)`, `api.config.reset(key, callback)` | `config.write` | Core keys only; validated like any other write |
| `api.log.recent()` | `log.read` | The recent shared log entries |
| `api.modules.settingsOf(id)` | `modules.read-settings` | Another module's resolved settings |

## Permissions

Declared in the manifest (`"permissions": ["helpers", ...]`). Own settings, own log writing, Theme, Style, and the module list never need one.

| Name | Meaning | Enforced by Lumen? |
|------|---------|--------------------|
| `helpers` | Run the helper programs declared in the manifest | Yes |
| `config.read` | Read core configuration values | Yes |
| `config.write` | Set and reset core configuration keys | Yes |
| `log.read` | Read the recent shared log entries | Yes |
| `modules.read-settings` | Read other modules' settings | Yes |
| `network` | The module intends to use the network | No: declaration only (Lumen offers no network API) |

Reading the properties `api.config.values` and `api.config.errors` without `config.read` gives an empty value and is not logged (they are evaluated when the module is created, whether or not it looks at them). A function call that needs a permission the manifest did not declare is refused. Nothing happens, Lumen logs a warning (`module <id> was refused <call>: permission <name> not declared`, source `module:<id>`), the call returns a harmless result (`undefined`, `false`, an empty value, or for `api.run` a request with `state: "denied"`), and the module keeps running. The declared permissions are shown by `lumen modules list` and `lumen modules show` before you enable a module.

## Helper runs

`api.run(name, input, timeoutMs, callback)`:

- `name` must be a `helpers` entry in the manifest, and the module must declare the `helpers` permission.
- The program is started as a separate process with `input` as JSON on stdin. It writes one response document on stdout, as in Foundation's helper contract: `{"ok": true, "protocol": 1, "data": {...}}` or `{"ok": false, "protocol": 1, "error": {"code": "...", "message": "..."}}`. Diagnostics go to stderr as ndjson lines and end up in the shared log.
- The run is asynchronous and time-limited (`timeoutMs`; `0` means the default from `helper.timeoutMs`). A failure or timeout is reported with the module id and the helper name, and nothing else is affected.
- `callback(request)` is called once. `request.state` is `"ok"`, `"failed"`, `"timeout"`, or `"denied"`; `request.data` is the helper's `data` when `ok`; `request.error` has `code` and `message` otherwise.
- When the module is unloaded, its pending runs are cancelled (outcome `cancelled`): the process is killed and the callback is not called. This outcome is new in the module interface; Foundation's helper runs only end as `ok`, `failed`, or `timeout`.

## Settings

Declared in the manifest under `settings`: a key and a rule `{type, default, enum?, min?, max?, description?}` (types `string`, `integer`, `number`, `boolean`). The user sets them in the same file as everything else:

```json
{"modules": {"hello": {"greeting": "Hi", "size": 4}}}
```

- Namespace: `modules.<id>.<key>`, so settings of different modules and Lumen's own can never collide. The name `enabled` is reserved: `modules.<id>.enabled` switches the module on or off.
- The file holds only overrides. Defaults are never written into it, and keys of modules that are not installed are preserved.
- An invalid value produces an error that names the module and key (`modules.hello.size: must be an integer between 1 and 5`), and only that key falls back to the module's default. `api.settings.errors` shows it.
- Resetting a setting removes its override (`api.settings.reset(key)`).
- Edits apply live: `api.settings.values` updates within a second.
- To print the merged schema (core settings plus the settings of every valid module) for an editor, run the `modules` program by hand:

```text
echo '{"bundledDir":"modules","userDir":"","apiVersion":"1.0","dataDir":"data"}' | scripts/bin/modules schema
```

## Interface 1.1: extension points, contributions and commands

Version 1.1 adds ways for modules to work together. Everything is optional, so a 1.0 module loads unchanged. The formal contract is `specs/003-bar/contracts/module-api-1.1.md` and `module-manifest-1.1.md`.

**Manifest fields.** `extensionPoints` (`[{id, description}]`, up to 8) declares a place others may contribute to. `contributions` (`[{id, name, target: {module, point}, file}]`, up to 16) offers a visual part to another module's point. `commands` (up to 16 names) lists what `lumen modules call <id> <command>` may ask the module. A module that contributes to another declares the permission `contribute:<module-id>` for each target; it is shown with the other permissions. `requires` stays reserved: accepted, ignored, and reported as a warning.

**Settings.** The setting type `stringlist` holds a list of strings (`default` an array of strings of 1 to 80 characters, `maxItems` up to 64). Entries of a stored list that are not strings are dropped on their own and reported. `string` settings accept `minLength` and `maxLength`. The entries of `api.settings.errors` gain `dropped` and `value`; for a dropped list entry the key is indexed (`right[2]`) and there is no `fallback`.

**A contributed item** is a file whose root is an `Item` with `required property var api` (its own interface object) and `property var slot` (what the host tells it). The host decides where it goes. For the bar, see [bar.md](bar.md).

**For hosts** (modules that declare a point):

| Member | Description |
|--------|-------------|
| `api.extensions.points` | The module's declared points |
| `api.extensions.contributions(pointId)` | Reactive list of the contributions to one of the module's points, with their states (`shown`, `available`, `waiting`, `rejected`, `failed`) |
| `api.extensions.create(pointId, itemRef, parent, slot)` | Creates a contributed item (a long or short item name), or returns `null` |
| `api.extensions.destroyItem(item)` | Destroys an item the host created. Items are also destroyed when either module is unloaded |
| `api.extensions.reject(pointId, itemRef, reason)` | Refuses a contribution with a reason, until the host's next `resolve` |
| `api.extensions.report(pointId, info)` | Publishes `{warnings, dropped}` for `lumen modules show` |
| `api.extensions.resolve(pointId, lists, callback)` | Resolves lists of item names against the available contributions (the rules are in [bar.md](bar.md)) |

**Commands.** A module assigns `api.commands.handler`, a function `(name, args)` that returns text. It runs when `lumen modules call <id> <name>` reaches the module, for a name the manifest declares.

## Rules for module authors

- Take every color from `api.theme` and every spacing, radius, font size, and duration from `api.style`. Do not hardcode them.
- Do heavy work (parsing, filtering large data) in a helper program, not in QML.
- Do not block the display thread. Lumen cannot interrupt a module that does.
- Make interactive elements operable from the keyboard.
- Runtime errors are written to the shell log and attributed to your module by file path. Three within 5 seconds of loading unload the module and mark it failed.
- A module that fails is never loaded again automatically. After fixing it, run `lumen modules reload <id>`.
- Reloading re-reads the entry file. Other QML files of the module are cached by the engine: after editing only a secondary file, restart the shell.

## Trust model

A module runs inside the shell with the shell's own privileges. Permissions limit what the `api` object does for you; they are **not a sandbox**, and QML can reach system facilities directly. Modules cannot be isolated from a hang or a crash that comes from their own code. Path checks for `main` and helper paths are lexical: a symlink inside a module directory can point outside it. Install only modules you trust. Modules from the user directory stay off until you enable them, and their permissions are shown first.

## Versioning

The interface version is `major.minor`. A module loads if its `api` major number equals Lumen's and its minor number is not higher. Adding a capability raises the minor number. A removal or a change of meaning raises the major number and comes with a migration note here:

```text
### Migration: module interface N.M -> N+1.0 (date)
What changed:
What to do:
```

No migrations: version 1.1 only added things (extension points, contributions, commands, the `stringlist` setting type, string length limits, and the `contribute:<module-id>` permission), so no module needs to change. A module that needs a 1.1 member declares `"api": "1.1"`; a 1.0 shell refuses it with the required and available versions. `requires` is still reserved. It is accepted and ignored, with a warning.
