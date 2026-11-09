# The bar

The bar is a bundled module (`bar`). It is a frame with three places, **left**, **center** and **right**, on every screen you choose. It contains no items of its own: every item is supplied by another module (the clock, the sound level and the battery are `bar-clock`, `bar-sound` and `bar-battery`), and you decide which items appear, and in which order, in the configuration file.

The bar can sit on any of the four edges of the screen and has three looks: **solid** (one strip), **islands** (three separate pills) and **frame** (a border around the whole screen). Colors, fonts, sizes, corner radii and spacing are all settings, so you never have to touch the code. See [Styles](#styles), [The look](#the-look) and [Vertical bars](#vertical-bars).

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
| `edge` | `top` | `top`, `bottom`, `left` or `right` |
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

## Styles

`style` chooses how the bar is drawn. You can change it while Lumen runs.

| Style | Look |
|-------|------|
| `solid` (the default) | One continuous strip along the edge of the screen, as before |
| `islands` | Each of the three places that holds an item is its own rounded shape, with the desktop visible between them; a place with no item draws nothing, and with no item at all there is no bar on that screen. The gaps between the islands let mouse clicks through |
| `frame` | A border of one color around the whole screen. The items sit in the strip on the bar's edge, a thinner border runs along the other three sides, and the area inside the frame has rounded corners. Windows never go under the frame |

```json
{"modules": {"bar": {"style": "islands", "margin": 8, "radius": 14, "islandOpacity": 0.9}}}
```

```json
{"modules": {"bar": {"style": "frame", "edge": "left", "frameBorder": 10, "frameRadius": 24}}}
```

A key that does not apply to the current style is still checked and kept; it takes effect when you switch style.

| Key | Used by |
|-----|---------|
| `backgroundRole`, `backgroundOpacity` | solid |
| `islandRole`, `islandOpacity` | islands |
| `frameRole`, `frameOpacity`, `frameRadius`, `frameBorder` | frame |
| `radius` | the strip (solid) or each island (islands) |

## The look

Every size is in logical pixels, so the proportions are the same at any display scale. A key that is not set uses a design token or the default of the style (the "When unset" column), and deleting a key from the file returns it to that. A bad value falls back for that key only and `lumen modules show bar` names the key.

| Key | Range | When unset |
|-----|-------|-----------|
| `style` | `solid`, `islands`, `frame` | `solid` |
| `thickness` | 16 to 256 | the font size plus twice the medium spacing (37 with the default font) |
| `margin` | 0 to 128 | solid 0, islands the small spacing, frame 0. The gap between the bar and the screen edge |
| `padding` | 0 to 128 | the small spacing (islands: the medium). Space at the ends of the strip, or inside each island |
| `itemGap` | 0 to 128 | the small spacing. Space between the items of a place |
| `placeGap` | 0 to 256 | the small spacing. Space between the three places |
| `itemPadding` | 0 to 64 | the medium spacing. Padding the bar gives its items along the bar |
| `itemRadius` | 0 to 64 | the small radius. Corner radius the bar gives the shapes of its items |
| `radius` | 0 to 256 | solid 0, islands fully round. Corner radius of the strip or of each island |
| `frameRadius` | 0 to 256 | the large radius. Radius of the rounded inner corners of the frame |
| `frameBorder` | 0 to 128 | the small spacing. Width of the frame on the three sides without the bar; 0 removes those sides and keeps the rounded corners |
| `fontFamily` | up to 128 characters | the design font |
| `fontSize` | 1 to 96 | the body text size |
| `fontWeight` | `thin`, `light`, `regular`, `medium`, `semibold`, `bold`, `black` | `regular` |
| `fontItalic` | `true` or `false` | `false` |
| `backgroundRole`, `islandRole`, `frameRole` | a theme role | `surface` |
| `backgroundOpacity`, `islandOpacity`, `frameOpacity` | 0 to 1 | `1` |
| `textRole` | a theme role | `onSurface` |
| `focusRole` | a theme role | `primary` |
| `popupRadius` | 0 to 64 | the medium radius |
| `popupPadding` | 0 to 64 | `0` |
| `popupBorderWidth` | 0 to 16 | `1` |
| `popupBackground` | a theme role | `surfaceContainer` |
| `popupBorder` | a theme role | `outline` |

The theme roles are `primary`, `onPrimary`, `secondary`, `onSecondary`, `tertiary`, `onTertiary`, `error`, `onError`, `background`, `onBackground`, `surface`, `onSurface`, `surfaceVariant`, `onSurfaceVariant`, `surfaceContainer` and `outline`. A color is always a role of the active theme, so it follows theme changes; to use a color the theme does not have, write a theme.

- A radius larger than half of the shape's smaller side is drawn fully round and is not an error.
- On a small screen the bar and the frame together may take at most half of the screen across the bar and the margins at most half along it; a value that does not fit is reduced, with one warning that names the key (see "Look warnings" in `lumen modules show bar`).
- A font that is not installed falls back to the design font with one warning that names the key and the family (the generic names `sans-serif`, `serif`, `monospace`, `cursive`, `fantasy` and `system-ui` need no check).
- The built-in items follow the bar's font, padding, radius and text role. Each also has its own keys to override them for that item alone (`fontFamily`, `fontSize`, `fontWeight`, `fontItalic`, `padding`, `radius`, `textRole`, `backgroundRole`, `backgroundOpacity`) in its own module, for example `{"modules": {"bar-clock": {"fontSize": 16, "fontWeight": "bold"}}}`. Unset, they follow the bar; an item draws a background of its own only when `backgroundRole` is set.

## Vertical bars

With `edge` set to `left` or `right` the bar is vertical: the left place is at the top, the center in the middle and the right place at the bottom, each in list order, and the dropping rule works along the height. The built-in items adapt: the clock shows hours above minutes (`verticalFormat`, default two lines), the sound item shows `Vol` above its level (`Vol` over `off` while muted) and the battery item shows `Bat` (or `Chg` while charging) above its level. An item written for a horizontal bar is still shown, clipped to the bar's thickness.

## Keyboard

The bar never takes the keyboard on its own: nothing in it, in any style or on any edge, holds keyboard focus until you run the command below, and a popup holds it only while it is open and closes on Escape. Bind the focus command to a key in the compositor. In niri:

```kdl
Mod+B { spawn "lumen" "modules" "call" "bar" "focus"; }
```

While the bar has the keyboard focus, Left and Right move between the interactive items (in list order), Escape leaves the bar, and every other key goes to the focused item. The focused item draws an outline in the primary color. The sound item uses Enter (open the mixer), Up and Down (volume) and `M` (mute). If no item is interactive, the command answers `no interactive items` and the focus stays where it was.

## The sound item

Left click (or Enter) opens an audio mixer, scrolling (or Up and Down) changes the volume by the `step` setting (default 5, in percent) and stops at 0% and 100%, and right click (or `M`) toggles mute. `mixer` is the program to open; empty means EasyEffects if it is installed, otherwise `pavucontrol`. Without a mixer the other actions still work and the click is logged as `mixer: no audio mixer found`. When there is no sound output the item is hidden.

## The clock and the battery

The clock follows `format` (a Qt date-time format, default `ddd d MMM  HH:mm`) and `seconds` (update every second); on a vertical bar it uses `verticalFormat` (default: hours, a new line, minutes). The battery item appears only on a machine with a battery. It shows the level and a `+` while charging, and uses the tertiary and error colors at or below `lowLevel` (default 20) and `criticalLevel` (default 10).

## Writing an item

See section 8 of [writing-a-module.md](writing-a-module.md) and `examples/hello-bar-item/`. An item receives its own `api` (see [module-api.md](module-api.md)) and a `slot`, described here.

### The slot

| Member | Meaning |
|--------|---------|
| `slot.edge` | `top`, `bottom`, `left` or `right` |
| `slot.vertical` | True when the bar is on the left or right edge: items stack from top to bottom |
| `slot.thickness` | The bar's size across its direction: its height on a horizontal bar, its width on a vertical one; size the item to it |
| `slot.maxWidth` | The most width the item can use: `slot.maxLength` on a horizontal bar, `slot.thickness` on a vertical one. Wider items are clipped |
| `slot.maxLength` | The most length along the bar the bar gives this item; longer items are clipped |
| `slot.font` | The bar's resolved font `{family, pixelSize, weight, italic}`; an item that sets no font of its own should use it |
| `slot.itemPadding` | The bar's padding for an item along the bar |
| `slot.itemRadius` | The bar's corner radius for an item's own shapes |
| `slot.textRole` | The name of the theme role for text in the bar; read the color from `api.theme.roles[slot.textRole]` |
| `slot.focusRole` | The name of the theme role for the focus indicator |
| `slot.screenName` | The output this instance is on |
| `slot.place` | `left`, `center` or `right` |
| `slot.focused` | True while the keyboard focus is on this item |
| `slot.popup` | Opens a popup next to the item, see below |

The slot facts follow the bar's settings live. An item that reads `slot.vertical`, `slot.maxLength`, `slot.font`, `slot.itemPadding`, `slot.itemRadius`, `slot.textRole` or `slot.focusRole` must declare `"api": "1.2"`.

The bar creates one instance of an item per bar. An item is interactive when its root sets `property bool interactive: true`; it then receives the keys the bar does not use (Enter and its own keys) while it is focused.

### Popups

`slot.popup.open(component, options)` creates `component` in a popup next to the item and returns `true` when it opened. `options` is `{width, height}`; 0 or missing means the content's own size. `slot.popup.close()` closes it, `slot.popup.isOpen` tells whether it is open, and `slot.popup.content` is the created object.

- The popup is on the item's screen, below a top bar, above a bottom bar, right of a left bar and left of a right bar, and never covers the item or leaves the screen. It takes at most half of the screen's width and height.
- It takes its colors from the theme and its radius, padding and border from the bar's popup keys (`popupRadius`, `popupPadding`, `popupBorderWidth`, `popupBackground`, `popupBorder`), so it follows theme and setting changes while open.
- It closes when the bar's style or edge changes.
- Only one popup is open at a time across all bars: opening one closes the other.
- It closes on Escape and when you click outside it, and when its item or module is removed, disabled, reloaded or fails.
- Opening a popup for an item that is not currently shown is refused and logged.

## Optional programs

| Program | Used by | Without it |
|---------|---------|------------|
| EasyEffects or `pavucontrol` | Left click on the sound item | The click logs `no audio mixer found`; everything else works |
| `notify-send` (libnotify) and a notification daemon | The notification for dropped list entries | Only the log entry is written |
