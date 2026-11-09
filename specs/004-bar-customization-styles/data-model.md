# Data Model: Bar Customization and Styles

Additions to [Feature III's data model](../003-bar/data-model.md). Everything not listed here is unchanged.

## Manifest addition (interface 1.2)

| Field | Change |
|-------|--------|
| setting `default` | May be `null` for `string`, `integer`, `number` and `boolean` rules: the setting is **unset** until the user gives a value. `stringlist` keeps a required array default. See [contracts/module-manifest-1.2.md](contracts/module-manifest-1.2.md) |
| `api` | Up to `1.2` is accepted |

No other manifest field changes.

## Settings entity

| Field | Notes |
|-------|-------|
| `key` | `^[a-z][A-Za-z0-9]{0,31}$` |
| `rule` | `type`, optional `enum`, `min`, `max`, `minLength`, `maxLength`, `maxItems`, `description` |
| `default` | A value of the type, or `null` (unset) |
| `value` | The effective value: the user's value, else the default; `undefined` when both are absent. Never written to the user file unless the user set it |

## Look (the bar's resolved settings)

Computed in the bar module from `api.settings.values` and the design tokens; not stored. One object per bar module, read by every screen's window.

| Field | Source | Notes |
|-------|--------|-------|
| `style` | `style` | `solid`, `islands`, `frame` |
| `edge`, `vertical` | `edge` | `vertical` is true for `left` and `right` |
| `font` | `fontFamily`, `fontSize`, `fontWeight`, `fontItalic` and the tokens | `{family, pixelSize, weight, italic}` |
| `roles` | the seven role settings and `Theme` | Each part's color with its alpha, bindings that follow the theme |
| `sizes` | the size settings, the tokens, the style defaults and the screen | Output of `geometry.js`, one per screen |
| `warnings` | `geometry.js`, the font check and the config errors | Printed by `lumen modules show bar`, logged once per change |

## Sizes (output of geometry.js, per screen)

| Field | Meaning |
|-------|---------|
| `thickness` | The bar's size across its direction |
| `margin` | The gap between the bar and the screen edge |
| `padding` | Space at the ends of the strip, or inside each island |
| `itemGap`, `placeGap` | Space between items in a place, and between places |
| `itemPadding`, `itemRadius` | The bar's values handed to items |
| `radius` | Corner radius of the strip or of each island |
| `frameRadius`, `frameBorder` | Inner corner radius and the border width on the other three sides |
| `reserved` | The space each edge gives up: `{top, bottom, left, right}` |

## Surfaces per screen

| Surface | Exists when | Reserves space | Input |
|---------|-------------|----------------|-------|
| Strip panel | bar shown on the screen (in the islands style, only while at least one item is shown) | on the bar's edge: thickness + margin | whole panel (solid, frame); the three islands only (islands, by mask) |
| Border panel (0 to 3) | frame style, and `frameBorder` above 0 | on its side: `frameBorder` | none needed (empty mask) |
| Corner block (0 to 4) | frame style, one per screen corner whose block (reserved width of its side + `frameRadius` by reserved height at its end + `frameRadius`) has a size, also for a radius of 0. It starts at the screen corner and ignores other surfaces' zones | none | none (empty mask) |
| Popup host | while a popup is open | none | whole screen while open (as in Feature III) |

## Slot (interface 1.2)

Read-only and reactive. Existing members are unchanged ([bar-slot.md](../003-bar/contracts/bar-slot.md)); new members are in [contracts/bar-slot-1.2.md](contracts/bar-slot-1.2.md): `vertical`, `maxLength`, `font`, `itemPadding`, `itemRadius`, `textRole`, `focusRole`, and `edge` now takes four values.

## Item settings (each built-in item)

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `fontFamily` | string | unset | Own font family; unset follows the bar |
| `fontSize` | integer 1 to 96 | unset | Own size; unset follows the bar |
| `fontWeight` | string enum | unset | Own weight; unset follows the bar |
| `fontItalic` | boolean | unset | Own italic; unset follows the bar |
| `padding` | integer 0 to 64 | unset | Own padding along the bar; unset uses `slot.itemPadding` |
| `radius` | integer 0 to 64 | unset | Own corner radius; unset uses `slot.itemRadius` |
| `backgroundRole` | string enum (the 16 roles) | unset | Own background; unset draws none |
| `backgroundOpacity` | number 0 to 1 | unset | Opacity of the own background; unset means 1 |
| `textRole` | string enum (the 16 roles) | unset | Own text color; unset uses `slot.textRole` |

Plus the item's own existing settings (clock: `format`, `seconds`, new `verticalFormat`; sound: `step`, `mixer`; battery: `lowLevel`, `criticalLevel`).

## State transitions

```text
style:  solid ⇄ islands ⇄ frame   (the strip panel and the items stay; border and corner surfaces are created or released)
edge:   top ⇄ bottom ⇄ left ⇄ right   (the strip panel is re-anchored; items are re-laid out, not recreated)
value:  unset ──user sets──▶ set ──user resets──▶ unset
        set to a bad value ──▶ error for that key, token or style default used
```
