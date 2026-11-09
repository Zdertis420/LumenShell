# Contract: the bar's slot, interface 1.2

Extends [bar-slot.md](../../003-bar/contracts/bar-slot.md). All members are read-only and reactive. The slot is still the only thing an item gets from the bar. Documented in `docs/bar.md`; a test fails if a member used by a bundled module or example is not documented there.

## Members changed

| Member | Type | Meaning now |
|--------|------|-------------|
| `slot.edge` | string | `top`, `bottom`, `left` or `right` |
| `slot.thickness` | int | The bar's size across its direction: its height on a horizontal bar, its width on a vertical one |
| `slot.maxWidth` | int | The most **width** the item can use. On a horizontal bar this is the most length the bar gives it (as before). On a vertical bar it is the thickness. An item written for Feature III that elides its text at `maxWidth` therefore shrinks to the thickness on a vertical bar |

## Members added

| Member | Type | Meaning |
|--------|------|---------|
| `slot.vertical` | bool | True when the bar is on the left or right edge. Items stack from top to bottom |
| `slot.maxLength` | int | The most length along the bar that the bar gives this item. Longer items are clipped |
| `slot.font` | object | The bar's resolved font: `{family, pixelSize, weight, italic}`. An item that sets no font of its own SHOULD use it |
| `slot.itemPadding` | int | The bar's padding for an item along the bar |
| `slot.itemRadius` | int | The bar's corner radius for an item's own shapes |
| `slot.textRole` | string | The name of the theme role for text in the bar (read the color from `api.theme[slot.textRole]`) |
| `slot.focusRole` | string | The name of the theme role for the focus indicator |

## How the bar sizes an item

| | Horizontal bar | Vertical bar |
|--|----------------|--------------|
| The item's length along the bar | `implicitWidth`, limited to `maxLength` | `implicitHeight`, limited to `maxLength` |
| The size across the bar | `thickness` (the item is centered in it) | `thickness` (the item is centered in it) |
| Clipping | The holder clips to the along-length and the thickness | Same |

An item that ignores the direction is still shown, clipped to the bar's thickness. It cannot move, resize or restyle the bar, other items or other modules' contributions (FR-024).

## Keyboard

This interface version changes no key handling. Keyboard use stays explicit: the bar reacts to the keyboard only after the user's own binding runs `lumen modules call bar focus`, which works on every edge with the keys it has in [Feature III](../../003-bar/contracts/bar-slot.md). The bundled items' keys do not change. A popup closes on Escape. This follows Principle VIII of the constitution (explicit keyboard interactivity); no surface of the bar takes keyboard focus until that command runs.

## Popup

`slot.popup` is unchanged in use. The popup appears on the side of the item that faces into the screen for each of the four edges, with the same rules (never over the item, never off screen, at most half of the screen, one at a time). Its radius, padding, border and colors come from the bar's popup settings ([bar-config-1.2.md](bar-config-1.2.md)).

## Interface version and legacy items

New members exist whatever `api` the item declares, so an item declaring 1.1 keeps working and may ignore them. An item that reads a new member MUST declare `"api": "1.2"`; a 1.1 shell then refuses it with the required and available versions.
