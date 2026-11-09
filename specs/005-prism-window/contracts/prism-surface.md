# Contract: Prism surface, commands and keyboard

## Commands

Declared in the manifest (`"commands": ["toggle", "open", "close", "clear"]`), reached by `lumen modules call prism <name>`. The handler returns one line of text.

| Command | Effect | Answer |
|---------|--------|--------|
| `toggle` | Opens when hidden, closes when shown | `opened` or `closed` |
| `open` | Opens; when already open, re-creates the window (moves to the focused screen) and keeps the text | `opened` |
| `close` | Closes when shown | `closed`, or `already closed` |
| `clear` | Empties the input (the window stays open if it is) | `cleared` |

No command is bound by default. Example binding for niri (`~/.config/niri/config.kdl`):

```kdl
binds {
    Mod+Space { spawn "lumen" "modules" "call" "prism" "toggle"; }
}
```

A command to a disabled or unloaded Prism answers that the module is not running.

## The surface

- One layer-shell surface, on the overlay layer, namespace `lumen-prism`, anchored to all four edges, `exclusionMode: Normal` (no zone of its own), so the compositor shrinks it to the usable area (research R1).
- It exists only while the window is shown or closing. While it does not exist Prism has no surface, no input region and no keyboard focus.
- Input: the whole surface takes pointer input while shown. A click outside the panel closes the window when `closeOnOutside` is true; otherwise it does nothing.
- Keyboard: `keyboardFocus: Exclusive` while the surface exists. The input is focused on open. All keys go to Prism until it closes; other windows get none (the user's expectation, stated in the spec).
- Layers inside the surface, bottom to top: the backdrop (if on), the shadow (if on), the panel (background and border), then the input, the separator, the results area and the footer inside it.

## Placement rules

Let the surface have size `A = (aw, ah)`, the margin `m`, the width `w` and the largest height `H` (after limiting to `A` minus twice `m`), and the current content height `h` (input, gap, rows up to `maxRows`, footer, padding, border, never above `H`).

| Anchor | Horizontal | Vertical (fixed edge) |
|--------|------------|-----------------------|
| `top-left`, `left`, `bottom-left` | `x = m + offsetX` | see below |
| `top`, `center`, `bottom` | `x = (aw - w) / 2 + offsetX` | see below |
| `top-right`, `right`, `bottom-right` | `x = aw - m - w + offsetX` | see below |

Vertical: top row (`top-left`, `top`, `top-right`): `y = m + offsetY` (when `offsetY` is unset and the anchor is `top`: `y = ah / 5`); middle row (`left`, `center`, `right`): `y = (ah - H) / 2 + offsetY` (top fixed, so the panel does not move as `h` changes); bottom row: `y = ah - m - h + offsetY` (the bottom edge fixed). Offsets are screen directions: positive X is right and positive Y is down, for every anchor. The result is moved back inside `[m, aw - m] x [m, ah - m]` when it would stick out, with one warning for the offset that caused it.

`inputPosition: top` places the input at the top of the panel and the results below; `bottom` places the input at the bottom and the results above, growing upward. With the bottom row of anchors, `inputPosition` defaults in the same way (`top`) but may be set to `bottom` by the user (the default is not changed by the anchor).

## Keys while open

| Keys | Effect |
|------|--------|
| Text keys, Backspace, Delete, Left, Right, Ctrl+A, Ctrl+X, Ctrl+C, Ctrl+V, Shift+arrows | Edit the input (as `TextInput` does) |
| Escape | Close at once, with or without text |
| Down, Up | Move the selection by one row; at the ends stop, or wrap if `wrapSelection` |
| Page Down, Page Up | Move by the visible rows |
| Home, End | First or last row (when the input is empty or the selection has the focus) |
| Enter | Choose the selected row (tells its source via `chosen`; sample rows do nothing) |

Home and End move the cursor in a non-empty input; they move the selection only when the input is empty. Documented in `docs/prism.md`.

## Popups

Prism opens no popup other than the window itself. Escape closes it.

## Preview mode

With `preview: true`, a fixed list of 24 sample rows replaces the sources' rows: titles of different lengths (one over 100 characters), with and without subtitle, hint and icon, enough to scroll. The input (the text after a mode prefix, the query) filters them by a case-insensitive "contains" on the title; none left shows `emptyNoMatch`. Choosing one does nothing (the window stays open). With `preview: false` and no rows, the empty state shows `emptyNoSource` for an empty input and `emptyNoMatch` for a non-empty one.
