# Contract: module interface 1.1 (additions)

Additions to [specs/002-modules/contracts/module-api.md](../../002-modules/contracts/module-api.md). `api.apiVersion` is `"1.1"`. Every 1.0 member keeps its meaning. `docs/module-api.md` mirrors this contract.

## `api.extensions` (hosts and contributors)

| Member | Needs permission | Description |
|--------|------------------|-------------|
| `api.extensions.points` | none | This module's declared extension points: `[{id, description}]` |
| `api.extensions.contributions(pointId)` | none | Reactive list of contributions that target this module's point `pointId`: `[{key, short, module, item, name, state, reason}]`. Only the host of a point may ask for it; a call for another module's point returns `[]` |
| `api.extensions.create(pointId, itemRef, parent, slot)` | none for the host; the contributor must have declared `contribute:<this module>` | Creates the contributed item named by `itemRef` (a `key`, `module/item`) as a child of `parent` (an `Item`), with `slot` given to it. Returns the object, or `null` with the reason logged and the contribution's state set to `rejected` or `failed` |
| `api.extensions.resolve(pointId, lists, callback)` | none | Host only. Resolves the user's lists of item names for one of this module's points against the available contributions and their module states (the `modules items` rules, see [helper-bar.md](helper-bar.md)). `lists` is `{left: [...], center: [...], right: [...]}` (a host may use other list names). `callback({lists, warnings, available})` is called once, asynchronously; the call is time-limited and a failure calls back with `{error}` and logs it |
| `api.extensions.reject(pointId, itemRef, reason)` | none | Host only. Refuses a contribution that goes beyond the limits the host documents. The contribution's state becomes `rejected` with `reason` (shown by the module list and `lumen modules show`), and `create` for it returns `null` until the mark is cleared. The mark is cleared by the host's next `resolve` for that point. An item that is already created is destroyed. Ignored (and logged) for a point the module does not declare or an unknown item |
| `api.extensions.destroyItem(item)` | none | Destroys an item created by this module's `create`. Items are also destroyed automatically in the cases below |
| `api.extensions.report(pointId, info)` | none | The host publishes the state of its point so that `lumen modules show <host>` can print it: `info` is `{warnings: [{place, index, entry, problem}], dropped: [{screen, items: [name, ...]}]}`. Replaces the previous report for that point. Read-only for everybody else (it appears in the host's detail view) |

The object `create` returns is the contributed file's root (an `Item`). It is destroyed when:

- the host or the contributor is unloaded, disabled, reloaded, or fails;
- the host calls `destroyItem(item)` (not named `destroy`, which QML's built-in object method would shadow);
- the host's own object is destroyed.

Nothing the item created (timers, processes, windows) outlives it, because the item is a normal QML object. Pending helper runs of the contributor are cancelled when the contributor unloads (unchanged).

## What a contributed file receives

The contributed file's root declares:

```qml
import QtQuick

Item {
    required property var api       // the contributor's own api object (same members as any module)
    property var slot               // set by the host: its placement facts and services
}
```

- `api` is the contributor's own API object, so theme, style, log, its own settings and its own helpers work exactly as in a module's entry file, and its log entries carry its own module id.
- `slot` is defined by the host's contract; the bar's is [bar-slot.md](bar-slot.md). A contributed file MUST NOT assume members that its host does not document.
- The module's entry file (`Main.qml`) is still loaded when the module is enabled. A pure contributor may keep it nearly empty (a `Scope` with nothing in it) or use it for state shared by its items.

## `api.commands` (any module)

| Member | Description |
|--------|-------------|
| `api.commands.handler` | A function `(name, args) -> string` that the module assigns. Called when `lumen modules call <id> <name>` reaches the module. The name is always one of the module's declared `commands` |

- Calls for a name that is not declared are refused by Lumen (`unknown command`) and never reach the handler.
- A handler that throws is reported as a runtime error of the module; the call fails with the message.
- The return value (converted to text, up to 4 KiB) is printed by the controller.

## Version rules

- Interface 1.1 adds `api.extensions`, `api.commands`, the manifest fields and the permission form listed in [module-manifest-1.1.md](module-manifest-1.1.md). Nothing was removed or changed in meaning, so there is no migration note.
- A module that only uses 1.0 members declares `"api": "1.0"` and loads on 1.1. A module that uses a 1.1 member declares `"api": "1.1"` and is refused on a 1.0 shell.
- `api.settings.errors` entries keep `{key, problem, fallback}` and gain `dropped` (true when a single list entry was dropped) and `value` (the entry as written, cut to 80 characters). For a dropped entry `key` is indexed (`right[2]`), relative to the module like other keys, and there is no `fallback`. A module that does not read the new fields behaves as before.
- `api.modules.list` entries gain `extensionPoints`, `contributions` (declared, from the manifest) and `contributionStates` (`[{id, state, reason}]`), read-only.

## Rules the host and contributors follow

- The host owns the layout. A contribution cannot move, resize, restyle or remove the host or another contribution (it receives only its own item and its slot).
- A host MAY limit what it accepts and MUST then refuse the excess with a clear reason (`rejected`) without affecting other contributions. The bar's limits: item width up to `slot.maxWidth` (wider items are clipped), and at most 32 items in total across its three lists (the rest are refused with `reject`, see [bar-config.md](bar-config.md)).
- Contributions are created only while both modules are loaded and only while the host asks for them.
