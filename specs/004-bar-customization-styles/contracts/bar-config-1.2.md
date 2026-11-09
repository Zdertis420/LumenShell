# Contract: bar configuration, interface 1.2

Extends [bar-config.md](../../003-bar/contracts/bar-config.md). The rules from Foundation and Modules apply to every key: layered resolution, override-only writes, unknown keys preserved, validation with an error that names `modules.bar.<key>`, live reload within 1 s, and reset removing the override. A bad value falls back for that key only.

## Example

```json
{
  "modules": {
    "bar": {
      "style": "islands",
      "edge": "left",
      "margin": 8,
      "radius": 14,
      "fontFamily": "JetBrains Mono",
      "fontSize": 14,
      "backgroundRole": "surfaceVariant",
      "islandOpacity": 0.85,
      "left": ["hello-bar-item"],
      "center": ["bar-clock"],
      "right": ["bar-sound", "bar-battery"]
    },
    "bar-clock": {"fontWeight": "bold"}
  }
}
```

## Keys changed

| Key | Type | Default | Rule |
|-----|------|---------|------|
| `edge` | string | `top` | `top`, `bottom`, `left` or `right` |

## Keys added

"unset" means the key has `default: null` and the value in the last column is used. Sizes are in logical pixels.

| Key | Type | Default | Range or values | When unset |
|-----|------|---------|-----------------|-----------|
| `style` | string | `solid` | `solid`, `islands`, `frame` | n/a |
| `thickness` | integer | unset | 16 to 256 | font size + 2 x `spacing.md` |
| `margin` | integer | unset | 0 to 128 | style default (solid 0, islands `spacing.sm`, frame 0) |
| `padding` | integer | unset | 0 to 128 | style default (solid `spacing.sm`, islands `spacing.md`, frame `spacing.sm`) |
| `itemGap` | integer | unset | 0 to 128 | `spacing.sm` |
| `placeGap` | integer | unset | 0 to 256 | `spacing.sm` |
| `itemPadding` | integer | unset | 0 to 64 | `spacing.md` |
| `itemRadius` | integer | unset | 0 to 64 | `radius.sm` |
| `radius` | integer | unset | 0 to 256 | style default (solid 0, islands `radius.full`, frame 0) |
| `frameRadius` | integer | unset | 0 to 256 | `radius.lg` (frame only) |
| `frameBorder` | integer | unset | 0 to 128 | `spacing.sm` (frame only) |
| `fontFamily` | string | unset | up to 128 characters | `api.style.font.family` |
| `fontSize` | integer | unset | 1 to 96 | `api.style.font.body` |
| `fontWeight` | string | unset | `thin`, `light`, `regular`, `medium`, `semibold`, `bold`, `black` | `regular` |
| `fontItalic` | boolean | `false` | | n/a |
| `backgroundRole` | string | `surface` | one of the 16 roles | n/a |
| `backgroundOpacity` | number | `1` | 0 to 1 | n/a |
| `islandRole` | string | `surface` | one of the 16 roles | n/a |
| `islandOpacity` | number | `1` | 0 to 1 | n/a |
| `frameRole` | string | `surface` | one of the 16 roles | n/a |
| `frameOpacity` | number | `1` | 0 to 1 | n/a |
| `textRole` | string | `onSurface` | one of the 16 roles | n/a |
| `focusRole` | string | `primary` | one of the 16 roles | n/a |
| `popupRadius` | integer | unset | 0 to 64 | `radius.md` |
| `popupPadding` | integer | `0` | 0 to 64 | n/a |
| `popupBorderWidth` | integer | `1` | 0 to 16 | n/a |
| `popupBackground` | string | `surfaceContainer` | one of the 16 roles | n/a |
| `popupBorder` | string | `outline` | one of the 16 roles | n/a |

The 16 roles are `primary`, `onPrimary`, `secondary`, `onSecondary`, `tertiary`, `onTertiary`, `error`, `onError`, `background`, `onBackground`, `surface`, `onSurface`, `surfaceVariant`, `onSurfaceVariant`, `surfaceContainer`, `outline`. A test keeps this list equal to `Theme.qml`.

The defaults of the solid style on the top edge give the bar of Feature III: thickness 37 (13 + 2 x 12), no margin, padding and gaps of 8, no radius, `surface` background, `onSurface` text (SC-002).

## Which keys apply to which style

| Key | solid | islands | frame |
|-----|-------|---------|-------|
| `backgroundRole`, `backgroundOpacity` | strip | not used | not used |
| `islandRole`, `islandOpacity` | not used | each island | not used |
| `frameRole`, `frameOpacity` | not used | not used | the strip, borders and corners |
| `radius` | strip | each island | not used |
| `placeGap` | between the three places | between islands | between the three places |
| `frameRadius`, `frameBorder` | not used | not used | used |

A key that does not apply to the current style is still validated and kept; it takes effect when the style changes.

## Limits and warnings

- A value outside its range, a value of the wrong type, a name that is not one of the allowed values: an error `modules.bar.<key>: <problem>`, and that key falls back (to the default, or to unset). Every bad key is reported, every good key is applied.
- A role name is checked against the enum by the config helper. There is no role the active theme lacks, because every palette maps all 16.
- A font family that is not installed: one warning `fontFamily: "<name>" is not installed; using <default>`, and the family falls back.
- A value that is valid but does not fit the screen is limited, with one warning that names the key and the limit used. The rule: along each axis, the space the bar and frame take (`thickness + margin`, and `frameBorder` on the opposite side) is at most half of the screen's length on that axis; `frameRadius` is at most half of the smaller side of the area inside the frame.
- A radius above half of the smaller side of its shape is drawn fully round and is **not** reported.
- Warnings are listed by `lumen modules show bar` (one line each, with the key), and written to the log once per change.

## Items

Each bundled item module has the settings listed in [data-model.md](../data-model.md), all unset by default. The effective value in an item is: its own setting if set, else the bar's value from `slot` (`slot.font`, `slot.itemPadding`, `slot.itemRadius`, `slot.textRole`), else the design token when the item runs without a bar. The clock adds `verticalFormat` (string, default `HH` + newline + `mm`, 1 to 64 characters), used while `slot.vertical` is true.
