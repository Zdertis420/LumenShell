# Contract: bar configuration

The bar is the module `bar`. Its settings live in the user's single configuration file under `modules.bar`, with the rules from Foundation and Modules: layered resolution, override-only writes, unknown keys preserved, validation with clear errors, live reload, and reset removing the override. The schema comes from the module's manifest and appears in the merged schema (`modules schema`).

## Example (waybar style)

```json
{
  "modules": {
    "bar": {
      "edge": "bottom",
      "screens": ["DP-1"],
      "left": ["hello-bar-item"],
      "center": ["bar-clock"],
      "right": ["bar-sound", "bar-battery"]
    },
    "bar-clock": {"format": "HH:mm"}
  }
}
```

## Keys

| Key | Type | Default | Rule |
|-----|------|---------|------|
| `modules.bar.edge` | string | `top` | `top` or `bottom` |
| `modules.bar.screens` | stringlist | `["all"]` | Each entry is `all` or a compositor output name (for example `DP-1`). `all` matches every screen. An empty list shows no bar. A name that matches no connected screen is ignored without error, and the bar appears when that output is connected |
| `modules.bar.left` | stringlist | `[]` | Item names in order, left to right |
| `modules.bar.center` | stringlist | `["bar-clock"]` | Item names in order |
| `modules.bar.right` | stringlist | `["bar-sound", "bar-battery"]` | Item names in order, left to right (the first listed is the leftmost of the right region) |
| `modules.bar.focusScreen` | string | empty | Output name that receives keyboard focus from `lumen modules call bar focus`; empty means the first bar |

Each list holds at most 64 entries of at most 80 characters. Invalid values:

- A value that is not a list, a list with more than 64 entries, or an edge that is not `top` or `bottom` is an error `modules.bar.<key>: <problem>`, and only that key falls back to its default.
- An entry of a list that is not a string (or is empty, or is longer than 80 characters) is **dropped on its own**: the other entries keep working. The bar writes an error to the log (`right[2]: 42 is not a string; entry dropped`, source `module:bar`) and sends one desktop notification per change (by the helper `modules/bar/bin/notify`, which needs `notify-send` from libnotify; without it only the log entry is written).

## Limits of the bar

The bar accepts at most 32 items in total across its three lists. Items beyond that, in list order (left, center, right), are refused by the bar with the reason `bar item limit (32) reached`, shown as `rejected` in `lumen modules show`.

## Item names

- Long form: `module-id/item-id` (for example `hello-bar-item/greeting`). It names exactly one contribution and stays valid when the module adds or reorders items.
- Short form: `module-id` (for example `bar-clock`). It names that module's only or first-declared contribution to the bar's extension point (`bar/items`).
- Both forms are always valid. The short and long form of the same item are the same item.
- `lumen modules show <id>` prints both forms for every item the module offers.

## Resolution rules

Done by `modules items` (see [helper-bar.md](helper-bar.md)), in this order for every entry of every list, left, then center, then right:

1. Not a name of the allowed form: skipped, warning `not a valid item name`.
2. Already placed (same long form, by either name form): skipped, warning `duplicate of <place> entry <n>`.
3. No contribution matches: skipped, warning `no such item`.
4. The module is installed but `disabled`: skipped, warning `module <id> is not enabled; run: lumen modules enable <id>`.
5. The module is `failed`: skipped, warning `module <id> failed: <reason>`.
6. Otherwise placed.

Every warning names the entry and the place. A skipped entry never stops the others and leaves no gap. Naming an item never enables its module and never writes to the file. When the module is enabled later, the item appears with no change to the lists.

## Layout rules

- Three regions on each bar: left items from the left edge, right items to the right edge, center items centered on the screen when they fit there, otherwise between the other two.
- Items are never overlapped. When they do not fit, items are dropped from the end of the right list first, then from the end of the center list, then from the end of the left list, one at a time, until the rest fit. Dropped items come back when the width allows. Which items were dropped is logged (source `module:bar`, once per change) and shown by `lumen modules show bar`.
- The bar's thickness comes from the design tokens (font size plus twice the spacing token). Items get the bar's thickness and a maximum width (`slot.maxWidth`); an item that asks for more is clipped to it.

## Live changes

Any change to the lists, the edge or the screens takes effect without a restart and without recreating items that did not change. Enabling, disabling, reloading or failing a module that supplies a listed item adds or removes just that item.
