# Prism

Prism is Lumen's Spotlight-like search window: one input, a list of results below it, and a short prefix to pick what to search. **This version is the window only.** It is opened and closed, placed, themed and customized, and it has an input and a results area, but it searches nothing yet. Searching apps, files and commands, running commands, web search and grep in a file come later, as modules that supply the rows (see [Sources](#sources)).

The formal contracts are in `specs/005-prism-window/contracts/`.

## Opening it

Prism has four commands. None has a default key: you bind the ones you want in your compositor.

| Command | What it does |
|---------|--------------|
| `lumen modules call prism toggle` | Opens the window, or closes it when it is open |
| `lumen modules call prism open` | Opens the window. When it is already open it is re-created on the screen that has the focus, and keeps its text |
| `lumen modules call prism close` | Closes the window |
| `lumen modules call prism clear` | Empties the input |

An example binding for niri (`~/.config/niri/config.kdl`):

```kdl
binds {
    Mod+Space { spawn "lumen" "modules" "call" "prism" "toggle"; }
}
```

## Keyboard

Prism is a search box, so it takes the keyboard when a command opens it (Principle VIII of the constitution allows this for a module that exists for keyboard input). It never takes the keyboard when Lumen starts, when the pointer passes over it, or by itself, and it gives the keyboard back to the window you were in when it closes. While it is open every key goes to Prism, so other windows get none until it closes.

| Keys | Effect |
|------|--------|
| Text keys, Backspace, Delete, arrows Left and Right, Ctrl+A, Ctrl+X, Ctrl+C, Ctrl+V | Edit the input (input methods work as the toolkit allows) |
| Escape | Closes at once, with or without text |
| Down, Up | Move the selection one row (wraps if `wrapSelection` is on) |
| Page Down, Page Up | Move by the rows that are shown |
| Home, End | Select the first or last row while the input is empty; in a text they move the text cursor |
| Enter | Choose the selected row |

The only popup Prism opens is its own window; it closes on Escape. A click outside it closes it too (`closeOnOutside`, on by default).

## Modes: the prefix

A mode is chosen by a prefix at the very start of the input: `@` and one letter, then a space (or the end of the text).

| Prefix | Mode | Meant for (a later feature) |
|--------|------|-----------------------------|
| `@a` | `apps` | Applications |
| `@f` | `files` | Files |
| `@w` | `web` | Web search |
| `@s` | `commands` | Commands to run |
| `@g` | `grep` | Searching inside a file |
| none | `default` | Everything else |

`@f report` is the mode `files` with the query `report`. `@fx`, `@x report` and `@@f` are not prefixes: they are plain text in the `default` mode. A recognized prefix is highlighted behind its letters. The letters are settings (`prefixApps`, `prefixFiles`, `prefixWeb`, `prefixCommands`, `prefixGrep`), each one lowercase letter or digit; a letter that another mode already uses falls back to the default letter with an error that names the key. Modules always receive the mode name (`files`), not your letter. The window recognizes the prefix and passes it on; it does not act on it.

## Where it goes

`anchor` puts the window at one of nine places of the usable area of the screen (the area the bar leaves): `top-left`, `top`, `top-right`, `left`, `center`, `right`, `bottom-left`, `bottom`, `bottom-right`. The default, `top`, puts the top at one fifth of the usable height, like Spotlight. `offsetX` (positive moves right) and `offsetY` (positive moves down) shift it, `margin` keeps a gap to the edges, and `width` and `maxHeight` (or `widthShare` and `maxHeightShare`, shares of the usable area) set the size. A size that does not fit is limited and an offset that would push the window out is reduced, each with one warning that names the key. The edge you anchored to stays fixed when the list grows: it grows downward at the top anchors and upward at the bottom ones, and with `inputPosition: "bottom"` the input sits at the bottom of the window.

With `screen` empty the window opens on the screen that has the focus; a name opens it on that output (an absent name falls back to the focused screen with a warning). Changing any placement setting while the window is open moves it within a second.

## Settings

All settings are under `modules.prism` in the configuration file and apply live. Unset (`null`) values follow the design tokens. The full table, with defaults and ranges, is `specs/005-prism-window/contracts/prism-config.md`; the groups are:

| Group | Keys |
|-------|------|
| Placement and size | `anchor`, `offsetX`, `offsetY`, `margin`, `width`, `widthShare`, `maxHeight`, `maxHeightShare`, `screen` |
| Behavior | `closeOnOutside`, `keepText`, `wrapSelection`, `maxRows`, `rowHeight`, `iconSize`, `inputPosition`, `preview` |
| Content | `placeholder`, `emptyNoSource`, `emptyNoMatch`, `showFooter`, `footerHints`, `showSubtitle`, `showIcon`, `showHint` |
| Prefix letters | `prefixApps`, `prefixFiles`, `prefixWeb`, `prefixCommands`, `prefixGrep` |
| Shape and spacing | `radius`, `inputRadius`, `rowRadius`, `borderWidth`, `padding`, `inputPadding`, `rowPadding`, `gap`, `rowGap`, `separator`, `separatorWidth`, `shadow`, `shadowBlur`, `shadowOffset` |
| Backdrop and prefix highlight | `backdrop`, `prefixOpacity` |
| Motion | `animation` (`fade`, `scale`, `slide`, `none`), `animationDuration`, `animationScale` |
| Colors | a role (one of the 16 theme roles) for each part: `windowRole`, `borderRole`, `inputRole`, `inputTextRole`, `placeholderRole`, `prefixRole`, `titleRole`, `subtitleRole`, `hintRole`, `selectedRole`, `selectedTextRole`, `hoverRole`, `separatorRole`, `footerRole`, `emptyRole`, `shadowRole`, `backdropRole`; and an opacity for `window`, `border`, `input`, `selected`, `hover`, `shadow` and `backdrop` (`windowOpacity` and so on) |
| Fonts | `fontFamily`, `fontSize`, `fontWeight`, `fontItalic` for the whole window, and the same four with a role prefix (`inputFont...`, `placeholderFont...`, `titleFont...`, `subtitleFont...`, `hintFont...`, `footerFont...`, `emptyFont...`) |

Every color is a role of the active theme, so the window follows theme and wallpaper changes live. A role the theme lacks falls back to that part's default with an error; a pair of roles whose text contrast against its background is below 4.5:1 is reported as a warning (your choice is kept). A font family that is not installed falls back to the design token family with a warning. Some spacings have no key of their own and follow a neighbor key (the icon-to-text gap follows `rowPadding`, the footer padding follows `inputPadding`, and so on; the table is in the contract).

Examples:

```json
{"modules": {"prism": {"anchor": "center", "width": 720, "radius": 24}}}
{"modules": {"prism": {"windowRole": "surface", "windowOpacity": 0.9, "backdrop": true, "animation": "scale"}}}
{"modules": {"prism": {"prefixFiles": "p", "inputPosition": "bottom", "anchor": "bottom"}}}
```

`lumen modules show prism` lists the state, the settings errors, the sources and every warning (limits, fonts, contrast, an absent screen).

## Preview mode

With `"preview": true` the results area shows 24 fixed sample rows (with and without icon, subtitle and hint, and one very long title). Typing filters them by a case-insensitive "contains" on the title (after the prefix), and no match shows `emptyNoMatch`. Use it to judge the look while setting colors, fonts and shapes. The sample rows are not results: choosing one does nothing. It is off by default.

## Sources

A source is a module that supplies rows to Prism. Prism declares the extension point `sources`; a source contributes to it with the permission `contribute:prism`:

```json
{
  "id": "my-source",
  "name": "My source",
  "version": "1.0.0",
  "api": "1.3",
  "permissions": ["contribute:prism"],
  "contributions": [
    {"id": "files", "name": "Files", "target": {"module": "prism", "point": "sources"}, "file": "Source.qml"}
  ]
}
```

The source file's root is an `Item` with `required property var api` and `property var slot`. It draws nothing; the slot is what Prism tells it:

| Member | Description |
|--------|-------------|
| `slot.mode` | `default`, `apps`, `files`, `web`, `commands` or `grep` (fixed names) |
| `slot.query` | The text after the prefix and one space (the whole text for `default`) |
| `slot.text` | The whole input text |
| `slot.open` | True while the window is shown |
| `slot.selectedId` | The id of the selected row if it is this source's, else empty |
| `slot.rowLimit` | 1000: rows beyond it are dropped |
| `slot.setRows(rows)` | Replaces this source's rows. A row is `{id, title, subtitle?, hint?, icon?}` (title, subtitle and hint are cut at 200 characters; ids are unique per source) |
| `slot.close()` | Asks Prism to close |
| `chosen(id)` | Signal: the user chose a row of this source (Enter or double-click). Prism does nothing else with it |

The query and mode reach the source within the same event-loop turn as the key. Slow work belongs in a helper (`api.run`); call `setRows` when it answers. A source that fails or is unloaded loses only its rows. A source cannot move, resize or restyle the window.

A working example is `examples/hello-prism-source/`: copy it to `~/.local/share/lumen/modules/` and enable it (`lumen modules enable hello-prism-source`). Type some text in Prism and three rows appear; type `@f text` and it shows the mode and the query it received; choosing a row closes Prism.

The slot may gain members in a minor version of Prism; a removal or a change of meaning is a major version of Prism with a migration note here.

## Known limits

- Search, ranking and running anything are not part of this version.
- Input method composition depends on the compositor's text-input support; it is checked by hand.
- The backdrop dims only; there is no blur (blur depends on the compositor).
- The shadow is drawn behind the whole panel, so with a translucent window background it shows through.
