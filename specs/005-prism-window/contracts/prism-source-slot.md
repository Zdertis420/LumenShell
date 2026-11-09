# Contract: Prism source slot

How a later feature (or any module) supplies rows to Prism. The window itself is not reachable from a source: it can neither move, resize nor restyle it, and it receives nothing private (FR-029).

## Offering a source

In the source module's manifest:

```json
{
  "id": "my-source",
  "api": "1.3",
  "permissions": ["contribute:prism"],
  "contributions": [
    {"id": "files", "name": "Files", "target": {"module": "prism", "point": "sources"}, "file": "Source.qml"}
  ]
}
```

(`api` is `1.2` if the module uses nothing newer; the target point exists in both.)

## The source file

The root is an `Item` with `required property var api` and `property var slot`, as for any contributed item. It draws nothing. It is created when Prism and the module are loaded and lives until either unloads.

```qml
import QtQuick

Item {
    required property var api
    property var slot

    Connections {
        target: slot
        function onQueryChanged() {
            if (slot.mode === "default")
                slot.setRows([{id: "1", title: "You typed: " + slot.query}])
        }
        function onChosen(id) {
            slot.close()
        }
    }
}
```

## `slot`

| Member | Type | Description |
|--------|------|-------------|
| `slot.mode` | string | `default`, `apps`, `files`, `web`, `commands` or `grep`. Fixed names, not the user's letters |
| `slot.query` | string | The text after the prefix and one space (the whole text for `default`) |
| `slot.text` | string | The whole input text as typed |
| `slot.open` | bool | True while the window is shown |
| `slot.selectedId` | string | The id of the selected row of this source, or empty |
| `slot.setRows(rows)` | function | Replaces this source's rows. `rows` is a list of `{id, title, subtitle?, hint?, icon?}`. Rows that break the rules of data-model.md are dropped with a warning that names the source |
| `slot.close()` | function | Asks Prism to close |
| `chosen(id)` | signal | The user chose a row of this source (Enter or double-click). Prism does nothing else with the choice |
| `slot.rowLimit` | int | `1000`: rows beyond it are dropped |

Notes:

- `mode`, `query` and `text` change as the user types; the change reaches the source within 50 ms of the key. A source decides for itself what to do with each mode and may ignore all but its own.
- Prism never calls a source with work to do; the source reacts to the change signals. A slow source must use its own helper (`api.run`) and call `setRows` when it has results, so the window never waits.
- `setRows` can be called at any time, also while the window is hidden; the rows are kept until replaced. When the window opens, rows stay as they are; a source that wants fresh results reacts to `slot.open`.
- The rows of all sources are shown in the order of the contributions as listed in `lumen modules show prism`.

## Failure

A source whose file fails to load, throws repeatedly (three runtime errors within 5 seconds of loading unload the module, as for any module), or is unloaded loses its rows; the window keeps working, other sources are unchanged, and the failure is logged with the source's path. A source that stays silent shows nothing and causes no error.

## What a source must not do

It must not take keyboard focus, show its own window over Prism, or read the settings of Prism (it needs the `modules.read-settings` permission for that, and it should use the facts it is given). Keyboard rules of Principle VIII apply to it as to every module.

## Versioning

The slot is part of the Prism contract: new members may be added in a minor version of Prism (`module.json` `version`), existing ones keep their meaning; a removal or change of meaning is a major version of Prism with a migration note in `docs/prism.md`.
