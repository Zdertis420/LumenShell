# The bar

The bar is a bundled module (`bar`). It is a frame with three places, **left**, **center** and **right**, on every screen you choose. It contains no items of its own: every item is supplied by another module (the clock, the sound level and the battery are `bar-clock`, `bar-sound` and `bar-battery`), and you decide which items appear, and in which order, in the configuration file.

## Configuring the bar

The bar's settings live under `modules.bar` in `~/.config/lumen/config.json`, like every other setting. They apply live.

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

| Key | Default | Meaning |
|-----|---------|---------|
| `edge` | `top` | `top` or `bottom` |
| `screens` | `["all"]` | Compositor output names (for example `DP-1`; list them with `niri msg outputs`), or the entry `all`. An empty list shows no bar. A name that matches no connected output is ignored, and the bar appears when that output is connected |
| `left`, `center`, `right` | `[]`, `["bar-clock"]`, `["bar-sound", "bar-battery"]` | The item names to show in each place, in order |
| `focusScreen` | empty | The output that receives keyboard focus from the focus command; empty means the first bar |

An invalid value falls back to its default for that key only, with an error that names the module and the key. A value that is not a list falls back as a whole. An entry of a list that is not a string is dropped on its own: it is written to the log as an error, one desktop notification is sent (through `notify-send`, if it is installed), and the rest of the list keeps working.

### Item names

An item has two names, and both always work:

- the long form `module-id/item-id`, which names exactly one item and stays valid when a module adds or reorders items;
- the short form `module-id`, which means that module's only or first-declared item for the bar.

`lumen modules show <module>` prints both forms for every item the module offers. Naming the short and the long form of the same item lists it twice, and the second is skipped with a warning.

### What the bar does with your lists

For every entry, in the order left, center, right, the bar places the item or skips it with a warning that names the place and the entry. You can read the warnings in `lumen modules show bar` and in the log.

| Situation | Warning |
|-----------|---------|
| No such item, or not a valid name | `no such item`, `not a valid item name` |
| The same item twice | `duplicate of <place> entry <n>` |
| The module is installed but not enabled | `module <id> is not enabled; run: lumen modules enable <id>` |
| The module failed | `module <id> failed: <reason>` |

Naming an item never enables its module. When you enable the module later, the item appears without any change to the lists.

### When the items do not fit

Items are never drawn over each other. If a screen is too narrow, items are dropped from the end of the right list first, then the end of the center list, then the end of the left list, one at a time, until the rest fit. The dropped items come back when there is room. `lumen modules show bar` names the dropped items per screen. The bar shows at most 32 items; the ones beyond that, in list order, are refused with a visible reason (`rejected` in `lumen modules show`).

## Keyboard

Bind the focus command to a key in the compositor. In niri:

```kdl
Mod+B { spawn "lumen" "modules" "call" "bar" "focus"; }
```

While the bar has the keyboard focus, Left and Right move between the interactive items (in list order), Escape leaves the bar, and every other key goes to the focused item. The focused item draws an outline in the primary color. The sound item uses Enter (open the mixer), Up and Down (volume) and `M` (mute). If no item is interactive, the command answers `no interactive items` and the focus stays where it was.

## The sound item

Left click (or Enter) opens an audio mixer, scrolling (or Up and Down) changes the volume by the `step` setting (default 5, in percent) and stops at 0% and 100%, and right click (or `M`) toggles mute. `mixer` is the program to open; empty means EasyEffects if it is installed, otherwise `pavucontrol`. Without a mixer the other actions still work and the click is logged as `mixer: no audio mixer found`. When there is no sound output the item is hidden.

## The clock and the battery

The clock follows `format` (a Qt date-time format, default `ddd d MMM  HH:mm`) and `seconds` (update every second). The battery item appears only on a machine with a battery. It shows the level and a `+` while charging, and uses the tertiary and error colors at or below `lowLevel` (default 20) and `criticalLevel` (default 10).

## Writing an item

See section 8 of [writing-a-module.md](writing-a-module.md) and `examples/hello-bar-item/`. An item receives its own `api` (see [module-api.md](module-api.md)) and a `slot`, described here.

### The slot

| Member | Meaning |
|--------|---------|
| `slot.edge` | `top` or `bottom` |
| `slot.thickness` | The bar's height in pixels; size the item to it |
| `slot.maxWidth` | The most width the bar gives this item; wider items are clipped |
| `slot.screenName` | The output this instance is on |
| `slot.place` | `left`, `center` or `right` |
| `slot.focused` | True while the keyboard focus is on this item |
| `slot.popup` | Opens a popup next to the item, see below |

The bar creates one instance of an item per bar. An item is interactive when its root sets `property bool interactive: true`; it then receives the keys the bar does not use (Enter and its own keys) while it is focused.

### Popups

`slot.popup.open(component, options)` creates `component` in a popup next to the item and returns `true` when it opened. `options` is `{width, height}`; 0 or missing means the content's own size. `slot.popup.close()` closes it, `slot.popup.isOpen` tells whether it is open, and `slot.popup.content` is the created object.

- The popup is on the item's screen, below a top bar and above a bottom bar, and never covers the item or leaves the screen. It takes at most half of the screen's width and height.
- It takes its colors and sizes from the theme and style like everything else, so it follows theme changes while open.
- Only one popup is open at a time across all bars: opening one closes the other.
- It closes on Escape and when you click outside it, and when its item or module is removed, disabled, reloaded or fails.
- Opening a popup for an item that is not currently shown is refused and logged.

## Optional programs

| Program | Used by | Without it |
|---------|---------|------------|
| EasyEffects or `pavucontrol` | Left click on the sound item | The click logs `no audio mixer found`; everything else works |
| `notify-send` (libnotify) and a notification daemon | The notification for dropped list entries | Only the log entry is written |
