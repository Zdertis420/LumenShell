# Contract: the bar's slot (what an item receives)

The `slot` given to every item that the bar creates through `api.extensions.create` (see [module-api-1.1.md](module-api-1.1.md)). It is the only thing an item gets from the bar. Documented in `docs/bar.md`.

## Properties (read-only, reactive)

| Member | Type | Meaning |
|--------|------|---------|
| `slot.edge` | string | `top` or `bottom` |
| `slot.thickness` | int | The bar's height in pixels (an item should size itself to it) |
| `slot.maxWidth` | int | The most width the bar will give this item; wider items are clipped |
| `slot.screenName` | string | The compositor output name of the screen this instance is on |
| `slot.place` | string | `left`, `center` or `right` |
| `slot.focused` | bool | True while keyboard focus is on this item (see below) |

The bar creates one instance of an item per bar (per screen). An item that holds shared state should keep it in its module's entry file (`Main.qml`) or in its own settings; each instance reads it from `api`.

## Popup

| Member | Meaning |
|--------|---------|
| `slot.popup.open(component, options)` | Opens a popup next to this item and creates `component` (a QML `Component`) inside it. `options` is `{width, height}` (the content's wished size). Returns `true` if it opened. Any popup that was open is closed first |
| `slot.popup.close()` | Closes this item's popup. Does nothing if it is not open |
| `slot.popup.isOpen` | True while this item's popup is open |
| `slot.popup.content` | The created content object (or `null`) |

Rules the bar enforces (FR-037 to FR-043):

- The popup is on the item's screen, next to the item, and never covers the item or leaves the screen: it is moved to fit and its size is limited to a maximum (documented in `docs/bar.md`, at most half of the screen's width and height).
- The content gets the same `api` as the item (it is created in the item's context) and takes colors and sizes from Theme and Style, so it follows theme changes live.
- At most one popup is open at a time across all bars.
- The popup closes on Escape, on a click outside it, when another popup opens, and when the item, its module or the bar is unloaded, disabled, reloaded or fails. Keyboard focus returns to the item that opened it.
- `open` for an item that is not currently shown is refused and logged.
- A runtime error in the content is a runtime error of the contributing module (same unload rule).

## Keyboard

- `lumen modules call bar focus` puts keyboard focus into the bar. While focused:
  - Left and Right move between the interactive items, in list order, left list first, then center, then right;
  - Escape leaves the bar and gives focus back;
  - every other key goes to the focused item, which handles it (Enter and its own keys).
- An item is interactive when its root sets `property bool interactive: true`. Items without it are skipped by the arrow keys.
- The focused item has `slot.focused` set and SHOULD draw a visible focus indicator from theme roles.
- Pointer and keyboard follow the same rules for a given action. For example, the sound item changes volume on scroll and on Up and Down, toggles mute on right click and `M`, and starts the mixer on left click and Enter.

## What a slot does not give

No access to other items, the bar's windows, the lists, or other screens. An item that needs to know about other modules uses the read-only module list in `api`.
