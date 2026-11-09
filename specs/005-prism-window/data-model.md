# Data Model: Prism Window (Skeleton)

Nothing is stored on disk except the user's overrides in `modules.prism.*`. All state below lives in memory in the Prism module and is lost when it unloads.

## Window state

| Field | Type | Rules |
|-------|------|-------|
| `shown` | bool | `false` at load. Set by `toggle` and `open`, cleared by `close`, `toggle`, Escape, a click outside (if `closeOnOutside`), a source's `close()`, and module unload. |
| `closing` | bool | The window exists while `shown` or `closing` (the closing animation). |
| `screenName` | string | The output the window was created on (for `lumen modules show prism`); empty when the compositor chose it. |
| `text` | string | The whole input text. Empty on open unless `keepText` is on and the last text exists. |
| `mode`, `query`, `prefixLength` | string, string, int | Derived from `text` by `prefix.js`. `mode` is one of `default`, `apps`, `files`, `web`, `commands`, `grep`. |
| `selectedIndex` | int | `-1` when no rows; otherwise within the row list. Reset to `0` when the rows or the text change (so the best match is selected); kept when only the pointer hovers. |
| `hoverIndex` | int | The row under the pointer, or `-1`. Never changes `selectedIndex`. |
| `rows` | list | The merged rows of all sources (or the preview rows), after the limits below. |

Transitions: `hidden → shown` (create surface, take keyboard) `→ closing → hidden` (destroy surface, release keyboard). `shown → shown` on an `open` command from another screen is a destroy and create. Two commands in quick succession end in the state of the last one (the last command sets `shown` and the Loader follows).

## Row

| Field | Type | Rules |
|-------|------|-------|
| `id` | string | Required, 1 to 200 characters, unique inside its source; a duplicate drops the later row with a warning. |
| `title` | string | Required, cut to 200 characters. |
| `subtitle` | string | Optional, cut to 200 characters. |
| `hint` | string | Optional (the right side), cut to 200 characters. |
| `icon` | string | Optional icon name or absolute path; a missing icon shows the neutral placeholder. |
| `source` | string | Set by Prism: the contribution that supplied it (not supplied by the source). |

A row with no `id` or `title` is dropped with a warning that names the source. A source supplies at most 1000 rows; the rest are dropped with one warning per change. Unknown fields are ignored.

## Source

| Field | Type | Rules |
|-------|------|-------|
| `name` | string | The contribution's name (from the manifest of the module that offers it). |
| `state` | `shown`, `available`, `waiting`, `rejected`, `failed` | As for any contribution (interface 1.1). |
| `slot` | object | Created by Prism per source; see `contracts/prism-source-slot.md`. |
| `rows` | list | What the source last set. Cleared when the source fails or is unloaded. |

## Mode

Six fixed names; five have a configurable prefix letter.

| Mode name | Setting for its letter | Default letter |
|-----------|------------------------|----------------|
| `apps` | `prefixApps` | `a` |
| `files` | `prefixFiles` | `f` |
| `web` | `prefixWeb` | `w` |
| `commands` | `prefixCommands` | `s` |
| `grep` | `prefixGrep` | `g` |
| `default` | none (no prefix) | |

Parsing (`prefix.js`): text starts with `@`, then exactly one configured letter, then a space or the end of the text. `prefixLength` counts the `@` and the letter (so the chip covers them). The query is the text after the prefix and one space. An unknown `@x`, an `@` alone, `@@f` and `@fx` are `default` with the whole text as the query. Letters are case-sensitive.

## Look (resolved settings)

`Main.qml` resolves the 105 settings once into one object given to the window:

| Part | Content |
|------|---------|
| `place` | anchor, offsets (null kept as "default"), margin, width, widthShare, maxHeight, maxHeightShare, screen |
| `layout` | inputPosition, maxRows, rowHeight, iconSize, wrapSelection, gaps, paddings, radii, borderWidth, separator, shadow |
| `colors` | the resolved colors (role chosen by name, opacity applied) for every colored part |
| `fonts` | global font and the seven role fonts, each with an unset field falling back to the global one, then to the tokens |
| `texts` | placeholder, empty-state texts, footer hints, which row parts are shown |
| `prefixLetters` | the five letters, after the duplicate check |
| `motion` | kind, duration |
| `tokens` | the design tokens that unset settings fall back to |

An unset (`null`) setting takes the documented token (see `contracts/prism-config.md`). A role name the theme lacks falls back to that part's default role with an error that names the key and the role (the Go rules already reject unknown role names; the theme check is a second guard).

## Sizes (output of `geometry.js`)

`resolve(settings, tokens, area)` with `area = {width, height}` of the surface (the usable area) returns `{sizes, panel, warnings}`:

- `sizes`: width (px), maxHeight (px), margin, offsetX, offsetY, rowHeight, iconSize, inputHeight, visibleRows.
- `panel`: `{x, y, width, height}` for a given current content height (the function `place(contentHeight)`), so the window can ask again as rows change without a new resolve. The fixed edge follows R8 of research.md.
- `warnings`: `[{key, problem}]`, one per value that was limited (size larger than the area, offset pushing the window out, a radius above half the smaller side, a named screen that is absent, a font that is not installed, a contrast pair below 4.5).

## Contrast pairs

Input text on input background (over the window background), row title on window background, subtitle and hint on window background, selected row text on selected row background, placeholder on input background, footer on window background, empty-state text on window background. Each below 4.5 gives one warning naming both keys; the check runs again when the theme changes.
