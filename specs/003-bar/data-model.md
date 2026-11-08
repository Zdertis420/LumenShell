# Data Model: Bar

Additions to [Modules' data model](../002-modules/data-model.md). Everything not listed here is unchanged.

## Manifest additions (interface 1.1)

See [contracts/module-manifest-1.1.md](contracts/module-manifest-1.1.md). Summary:

| Field | Type | Notes |
|-------|------|-------|
| `extensionPoints` | object[] | `{id, description}`; up to 8; `id` is `^[a-z][a-z0-9-]{0,31}$`, unique in the module. Declares a place others may contribute to |
| `contributions` | object[] | `{id, name, target: {module, point}, file}`; up to 16; `id` unique in the module and same pattern as point ids; `name` is the display name (1 to 64 characters); `file` is a `.qml` file inside the module directory |
| `commands` | string[] | Names a module handles through `api.commands`; up to 16; `^[a-z][a-z0-9-]{0,31}$` |
| `requires` | any | Still reserved (accepted, ignored, warned) |
| `permissions` | string[] | Gains the form `contribute:<module-id>` |
| setting `type` | | Gains `stringlist` (`default` an array of strings, optional `maxItems`) |
| setting rule | | `string` settings gain optional `minLength` and `maxLength` |

## Extension point

| Field | Type | Notes |
|-------|------|-------|
| `module` | string | The host module id |
| `id` | string | Point id, unique in the host |
| `description` | string | Up to 160 characters |
| `qualified` | string | `<module>/<id>`, for example `bar/items` |

## Contribution

| Field | Type | Notes |
|-------|------|-------|
| `module` | string | The contributor's module id |
| `id` | string | Item id, unique in the contributor |
| `name` | string | Display name |
| `target` | `{module, point}` | The extension point it contributes to |
| `file` | string | Absolute path of the visual part (after validation) |
| `key` | string | `<module>/<id>`; the long item name |
| `short` | string or empty | The contributor's module id when this is its first-declared contribution to the bar, otherwise empty |
| `state` | see below | Derived, never stored in a file |
| `reason` | string | Why it is waiting, rejected, or failed |

### Contribution state

| State | Meaning |
|-------|---------|
| `shown` | The host has created it and it is on screen (or in the host's layout) |
| `available` | The contributor and host are loaded and the item is valid, but no list names it ("available but not placed") |
| `waiting` | The host is missing, disabled, not loaded, or failed, or the contributor is disabled or not yet loaded |
| `rejected` | The host does not declare the extension point, the contributor lacks the `contribute:<host>` permission, or the host refused it with `api.extensions.reject` (limits; the bar refuses items beyond 32 in total) |
| `failed` | Its file did not load or it raised an early error storm; the contributor module is `failed` too |

```text
(contributor disabled) ──waiting──▶ enable ──▶ (host loaded?) ──no──▶ waiting
                                                    │yes
                                                    ▼
                          (named in a list?) ──no──▶ available
                                                    │yes
                                                    ▼
                                   create ──ok──▶ shown ──unload either side / unnamed──▶ available or waiting
                                      │
                                      └─error──▶ failed (contributor failed)
```

## Slot (what a host gives an item)

Defined by the host. For the bar, see [contracts/bar-slot.md](contracts/bar-slot.md): `edge`, `thickness`, `screenName`, `maxWidth`, `focused`, and the `popup` object (`open`, `close`, `isOpen`).

## Bar settings (module `bar`)

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `edge` | string | `top` | `top` or `bottom` |
| `screens` | stringlist | `["all"]` | Output names, or `all`; empty list shows no bar |
| `left` | stringlist | `[]` | Item names, in order; at most 64 |
| `center` | stringlist | `["bar-clock"]` | |
| `right` | stringlist | `["bar-sound", "bar-battery"]` | |
| `focusScreen` | string | empty | Output name for the keyboard focus command; empty means the first bar |

Stored as `modules.bar.<key>` in the user file, override-only.

## Item settings (built-in item modules)

| Module | Key | Type | Default | Limits |
|--------|-----|------|---------|--------|
| `bar-clock` | `format` | string | `ddd d MMM  HH:mm` | 1 to 64 characters |
| `bar-clock` | `seconds` | boolean | `false` | |
| `bar-sound` | `step` | integer | `5` | 1 to 25 |
| `bar-sound` | `mixer` | string | empty (automatic) | up to 128 characters |
| `bar-battery` | `lowLevel` | integer | `20` | 1 to 99 |
| `bar-battery` | `criticalLevel` | integer | `10` | 1 to 98 |

## Resolved item list

Result of `modules items` (see [contracts/helper-bar.md](contracts/helper-bar.md)):

| Field | Type | Notes |
|-------|------|-------|
| `left`, `center`, `right` | `[{name, key, module, item}]` | `name` is what the user wrote, `key` the long form |
| `warnings` | `[{place, entry, problem}]` | Unknown name, duplicate, module not enabled (with the command to enable it), module failed |
| `available` | `[{key, short, module, item, name, state}]` | Every contribution to the bar's point with its state, for `lumen modules show` |

## Fitting result (QML, per screen)

`fit(widths, available, gap)` returns `{left: n, center: n, right: n, dropped: [{place, index}]}`, the number of items kept per list and the dropped ones in drop order.

## Module command

| Field | Type | Notes |
|-------|------|-------|
| `module` | string | Id of the module handling it |
| `name` | string | One of the module's declared `commands` |
| `result` | string | The handler's return value as text, or an error |

## Registry additions (Modules singleton)

| Property / function | Notes |
|---------------------|-------|
| `apiVersion` | `"1.1"` |
| `contributions` | Array of contribution records, replaced as a whole on change |
| `extensionPoints` | Array of extension point records |
| `contributionsTo(module, point)` | Contributions targeting one point, with states |
| `rejectItem(hostId, point, key, reason)` | Marks a contribution `rejected` until the host's next `resolveItems`; destroys the item if it exists |
| `createItem(hostId, point, key, parent, slot)` | Creates a contributed item; returns the object or null with a reason |
| `resolveItems(hostId, point, lists, callback)` | Runs `modules items` with the host's contributions and their module states; calls back with the resolved lists, warnings and availability |
| `callCommand(id, name)` | Forwards to the module's command handler |
| Per-module `items` | Items the module has contributed, destroyed on unload |
