# Contract: Prism settings (`modules.prism.*`)

Flat keys in the user's config file, declared in `modules/prism/module.json`. All follow the layered, override-only rules; a bad value falls back for that key only, with an error that names `modules.prism` and the key. `null` default means *unset* (interface 1.2) and falls back to the token named in the last column. Sizes are logical pixels. "Role" keys take one of the 16 theme roles (`primary`, `onPrimary`, `secondary`, `onSecondary`, `tertiary`, `onTertiary`, `error`, `onError`, `background`, `onBackground`, `surface`, `onSurface`, `surfaceVariant`, `onSurfaceVariant`, `surfaceContainer`, `outline`); a test keeps this list equal to `Theme.qml`.

Total: **105 keys**.

## Placement and size (9)

| Key | Type | Default | Range | Meaning |
|-----|------|---------|-------|---------|
| `anchor` | string | `top` | `top-left`, `top`, `top-right`, `left`, `center`, `right`, `bottom-left`, `bottom`, `bottom-right` | The nine placements |
| `offsetX` | integer | `0` | -4096..4096 | Horizontal offset from the anchor; positive moves right |
| `offsetY` | integer | null | -4096..4096 | Vertical offset from the anchor; positive moves down. Unset: the top edge at one fifth of the usable height for `top`, `0` for every other anchor |
| `margin` | integer | null | 0..512 | Gap kept to the edges of the usable area; unset is the large spacing token |
| `width` | integer | `640` | 120..8192 | Width of the window |
| `widthShare` | number | null | 0.1..1 | Share of the usable width; when set it replaces `width` |
| `maxHeight` | integer | `480` | 60..8192 | Largest height; the results area scrolls beyond it |
| `maxHeightShare` | number | null | 0.1..1 | Share of the usable height; when set it replaces `maxHeight` |
| `screen` | string | `""` | up to 64 chars | Output name; empty follows the focus (compositor's choice) |

Limits: width and height are limited to the usable area minus twice the margin, with one warning (`width`, `maxHeight`); an offset that would push the window out is reduced to fit, with one warning (`offsetX` or `offsetY`); a named `screen` that is absent opens on the focused screen with one warning.

## Behavior (8)

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `closeOnOutside` | boolean | `true` | A click outside the window closes it |
| `keepText` | boolean | `false` | The input keeps its last text on the next open (memory only) |
| `wrapSelection` | boolean | `false` | Up at the first row goes to the last, and down at the last goes to the first |
| `maxRows` | integer | `8`, 1..64 | Rows visible before the list scrolls; also bounds the height with `maxHeight` |
| `rowHeight` | integer | null, 16..128 | Height of a row; unset is the body and caption text sizes plus twice the medium spacing (room for a title and a subtitle) |
| `iconSize` | integer | null, 8..128 | Icon size in a row; unset is the row height minus twice the small spacing |
| `inputPosition` | string | `top` | `top` or `bottom`: where the input sits in the window (results grow away from it) |
| `preview` | boolean | `false` | Preview mode: fixed sample rows fill the results area |

## Content (8)

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `placeholder` | string | `Search` | Text shown in an empty input (0..120 chars) |
| `emptyNoSource` | string | `No source is installed` | Empty state when nothing supplies rows and the input is empty (0..160) |
| `emptyNoMatch` | string | `Nothing matched` | Empty state when the input has text and no row remains (0..160) |
| `showFooter` | boolean | `false` | Show the footer strip |
| `footerHints` | stringlist | `["Esc close", "↑↓ select", "Enter choose"]` | Footer text items, up to 8 of 1..80 chars |
| `showSubtitle` | boolean | `true` | Show the row subtitle |
| `showIcon` | boolean | `true` | Show the row icon |
| `showHint` | boolean | `true` | Show the row's right-side hint |

## Prefix letters (5)

| Key | Default | Mode |
|-----|---------|------|
| `prefixApps` | `a` | `apps` |
| `prefixFiles` | `f` | `files` |
| `prefixWeb` | `w` | `web` |
| `prefixCommands` | `s` | `commands` |
| `prefixGrep` | `g` | `grep` |

Each is one lowercase letter or digit (an `enum` of the 36 characters, since manifest rules have no pattern). A duplicate of another key's letter is an error that names the later key (in the order above) and falls that key back to its default; if that default letter is taken too, that mode has no prefix (the error says so). The `@` sign is fixed.

## Shape and spacing (14)

| Key | Type | Default | Range | Unset / default means |
|-----|------|---------|-------|-----------------------|
| `radius` | integer | null | 0..256 | Large radius token |
| `inputRadius` | integer | null | 0..128 | Medium radius token |
| `rowRadius` | integer | null | 0..128 | Medium radius token |
| `borderWidth` | integer | null | 0..32 | Thin border token |
| `padding` | integer | null | 0..128 | Large spacing token |
| `inputPadding` | integer | null | 0..128 | Medium spacing token |
| `rowPadding` | integer | null | 0..128 | Medium spacing token |
| `gap` | integer | null | 0..128 | Small spacing token (input to results) |
| `rowGap` | integer | null | 0..64 | Extra small spacing token |
| `separator` | boolean | `false` | | A line between the input and the results |
| `separatorWidth` | integer | null | 1..16 | Thin border token |
| `shadow` | boolean | `true` | | Draw the shadow |
| `shadowBlur` | integer | null | 0..128 | Extra large spacing token |
| `shadowOffset` | integer | null | -64..64 | Small spacing token (downward) |

A radius larger than half of the smaller side of its shape is limited with one warning.

## Backdrop and prefix chip (2)

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `backdrop` | boolean | `false` | Dim the rest of the screen |
| `prefixOpacity` | number | `0.25`, 0..1 | Opacity of the chip behind a recognized prefix |

## Motion (3)

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `animation` | string | `fade` | `fade`, `scale`, `slide` or `none` |
| `animationDuration` | integer | null, 0..2000 | Unset is the normal motion token; `0` shows and hides at once |
| `animationScale` | number | `0.96`, 0.5..1 | Starting scale of the `scale` animation; `1` means no scaling |

The `slide` animation moves the panel by the extra large spacing token (no own key).

## Values without their own key

Each of these follows a named key or token, so nothing is fixed in code (Principle VII). The magic-values test lists these mappings:

| Value | Follows |
|-------|---------|
| Corner radius of the prefix chip | `rowRadius` |
| Horizontal padding of the prefix chip | `rowGap` |
| Gap between a row's icon and its text | `rowPadding` |
| Gap between a row's title and subtitle | `rowGap` |
| Gap between the row's text and its hint | `rowPadding` |
| Padding of the footer strip | `inputPadding` |
| Padding around the empty-state text | `padding` |
| Gap between footer items | `rowPadding` |
| Slide distance | extra large spacing token |

## Colors (24)

Roles (17):

| Key | Default role | Part |
|-----|--------------|------|
| `windowRole` | `surfaceContainer` | Window background |
| `borderRole` | `outline` | Window border |
| `inputRole` | `surfaceVariant` | Input background |
| `inputTextRole` | `onSurface` | Input text |
| `placeholderRole` | `onSurfaceVariant` | Placeholder |
| `prefixRole` | `primary` | Prefix chip |
| `titleRole` | `onSurface` | Row title |
| `subtitleRole` | `onSurfaceVariant` | Row subtitle |
| `hintRole` | `onSurfaceVariant` | Row hint |
| `selectedRole` | `primary` | Selected row background |
| `selectedTextRole` | `onPrimary` | Selected row text (title, subtitle and hint) |
| `hoverRole` | `surfaceVariant` | Hovered row background |
| `separatorRole` | `outline` | Separator line |
| `footerRole` | `onSurfaceVariant` | Footer text |
| `emptyRole` | `onSurfaceVariant` | Empty-state text |
| `shadowRole` | `background` | Shadow |
| `backdropRole` | `background` | Backdrop |

Opacities (7), numbers 0..1: `windowOpacity` (default `1`), `borderOpacity` (`1`), `inputOpacity` (`1`), `selectedOpacity` (`1`), `hoverOpacity` (`1`), `shadowOpacity` (`0.35`), `backdropOpacity` (`0.4`).

## Fonts (32)

Global (4): `fontFamily` (string, null: token family), `fontSize` (integer 1..96, null: body token), `fontWeight` (`thin`, `light`, `regular`, `medium`, `semibold`, `bold`, `black`; null: regular), `fontItalic` (boolean, `false`).

Per role (7 roles by 4 keys): the roles are `input`, `placeholder`, `title` (row title), `subtitle`, `hint`, `footer` and `empty`. The keys are `<role>FontFamily` (null: follows the global family), `<role>FontSize` (integer 1..96, null: the role's default token), `<role>FontWeight` (null: follows the global weight) and `<role>FontItalic` (boolean, null: follows the global italic). All 28 role keys default to `null`.

An unset role size follows the global `fontSize` when that is set. When it is not, the role's token applies: input = title token (16), placeholder = input, title = body (13), subtitle = caption (11), hint = caption (11), footer = caption (11), empty = body (13). A family that is not installed falls back to the token family with one warning that names the key and the family.

## Warnings and the report

`lumen modules show prism` prints the state, the settings errors, the warnings (limits, fonts, contrast, absent screen) and, under "Extension points", each source with its state (as for every contribution); row problems of a source are printed as item warnings. A warning is printed once per cause until the cause changes.

## Applies live

Every key changes the open window within 1 second without closing it, except `screen` (applies at the next open) and `keepText` (applies at the next open).
