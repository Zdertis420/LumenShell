# Prism

Prism is Lumen's Spotlight-like search window: one input, a list of results below it, and a short prefix to pick what to search. The window (Prism, version 1.1) is opened and closed, placed, themed and customized; the six search modes ([below](#search-modes)) come from the bundled module `prism-search`, which supplies the rows through Prism's source interface ([Sources](#sources)). Running commands, usage ranking and searching inside file contents from the files mode are not part of this version.

The formal contracts are in `specs/005-prism-window/contracts/` (the window) and `specs/006-prism-modes/contracts/` (the modes and the 1.1 additions).

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
| Tab | Completes the last word of the text when a source can: in the files mode the directory word (`@f report ~/Doc`), in the grep mode the file words after the pattern (`@g TODO notes/re`). Nothing happens in other modes or after a trailing space |

The only popup Prism opens is its own window; it closes on Escape. A click outside it closes it too (`closeOnOutside`, on by default).

## Modes: the prefix

A mode is chosen by a prefix at the very start of the input: `@` and one letter, then a space (or the end of the text).

| Prefix | Mode | What it does (see [Search modes](#search-modes)) |
|--------|------|-----------------------------|
| `@a` | `apps` | Finds and launches applications |
| `@f` | `files` | Finds files by name and opens them in your editor |
| `@w` | `web` | Opens a web search in your browser |
| `@s` | `commands` | Finds commands (never runs them) |
| `@g` | `grep` | Lists matching lines of files and opens them in your editor |
| none | `default` | Applications, then files, then a web search suggestion |

`@f report` is the mode `files` with the query `report`. `@fx`, `@x report` and `@@f` are not prefixes: they are plain text in the `default` mode. A recognized prefix is highlighted behind its letters. The letters are settings (`prefixApps`, `prefixFiles`, `prefixWeb`, `prefixCommands`, `prefixGrep`), each one lowercase letter or digit; a letter that another mode already uses falls back to the default letter with an error that names the key. Modules always receive the mode name (`files`), not your letter. The window recognizes the prefix and passes it on; it does not act on it.

## Search modes

The bundled module `prism-search` gives Prism its six modes. All matching, walking and parsing is done by a Go helper (`modules/prism-search/bin/search`, built by `make build`), so typing never blocks the window. A word in a query must appear in the name, in any order, ignoring case and Latin accents (`report final` finds `final_report.pdf`); better matches come first (names whose words start with yours, then shorter names). An empty query shows a one-line hint of what the mode expects, in the default mode nothing. Tab completes paths, as in a shell: `@f report ~/Doc` becomes `~/Documents/` when that is the only match; several matches extend to their common start and a line of message shows how many there are and the first names; none says "no match". `@g TODO notes/re` completes the file (grep file words may be files or folders, relative to the home). Names with spaces are written in double quotes (a name that contains a double quote is not offered). Hidden names are offered only after you type a dot. Tab never searches or runs anything. A lone `@` also shows nothing: it is the start of a prefix, so type the mode letter (longer text without a prefix, such as `@swc`, is an ordinary query). No search keeps history.

| Mode | Query | Result and what Enter does |
|------|-------|----------------------------|
| `@a` apps | part of a name, generic name or keyword | Applications from the desktop entries of the system and of your user (yours wins). Hidden, `NoDisplay`, other-desktop and uninstalled entries are skipped. **Enter launches the application** (a `Terminal=true` entry runs in your terminal) |
| `@f` files | `name` or `name <directory>` | Files by name, in this order: `~/Desktop`, `~/Downloads`, `~/Documents`; files directly in `~`; other folders of `~`; `~/.cache`, `~/.local`, `~/.config`; and last a **collapsed** "System files (N)" row (Enter on it expands it). With a directory (a word that is `~`, or starts with `/`, `~/` or `./` and names an existing folder) only that folder is searched. **Enter opens a file in your editor** |
| `@s` commands | part of a name | Executables of `~/.local/bin`, `~/bin`, `/usr/local/bin`, `/usr/bin`, `/usr/sbin`, `/bin`, `/sbin`, `/opt/bin` and your `commandDirs`, one row per name, with the full path. **Enter does nothing: commands are never run** |
| `@g` grep | `pattern file [file ...]` | Matching lines, with the match highlighted. The pattern is a Go (RE2) regular expression (no look-ahead or back-references); quote it if it has spaces (`@g "TODO|FIXME" notes.md`). Smart case: lowercase patterns ignore case. Relative files are taken from `~`. **Enter opens the file at that line** when the editor supports it |
| `@w` web | the text | One row "Search the web for «text»". **Enter opens your browser** on the engine's page. Nothing is sent before Enter |
| default | the text | Applications, then home files (same groups as `@f`, no directory is taken from the text), the collapsed system files, and last the web row |

The search stops reading a group after `maxFiles` files or `searchTimeMs`; a full group ends with a "N more" row. Unreadable folders are skipped, symbolic links to folders are not followed, a file with several paths is shown once, and `/proc`, `/sys`, `/dev`, `/run` and `/tmp` are never searched.

### Editor, terminal and browser

Files and grep matches open in `$EDITOR`, started inside a terminal (`terminal` setting, else `$TERMINAL`), because many editors need one: `kitty nvim +12 file`. `$EDITOR` may have arguments (`code -w`). The line argument is known for vi, vim, nvim, nano, emacs, kak (`+N`), hx, helix, micro, zed (`file:N`) and code, codium (`-g file:N`); other editors get the file alone. The web row opens `$BROWSER` (colon-separated alternatives are tried in order; `%s` in its arguments is replaced by the address). Prism never guesses another program: when a variable is not set it stays open and shows an error that names it.

These variables are read from the environment that starts the shell. **A shell startup file such as `.bashrc` is not read by the shell**, so a variable set only there is not seen. Set them where the session starts: in a login shell file that your session runs, or, for niri, in `config.kdl`:

```kdl
environment {
    EDITOR "nvim"
    TERMINAL "kitty"
    BROWSER "firefox"
}
```

and restart the session.

### Settings of the modes

Under `modules.prism-search`; they apply to the next search.

| Key | Default | Meaning |
|-----|---------|---------|
| `rowsPerGroup` | 20 | Rows per group; the rest is a "more" row |
| `maxFiles`, `searchTimeMs` | 200000, 1500 | Limits of one stage of the file search |
| `hiddenFiles`, `hiddenHomeDirs` | false | Match hidden files and folders; search hidden folders of `~` beyond `.cache`, `.local`, `.config` |
| `systemGroup`, `systemRoots` | true, `/etc /usr /opt /var /srv` | Show the system group and where it looks |
| `commandDirs` | `[]` | Extra absolute folders for `@s` |
| `grepMaxMatches`, `grepCase` | 500, `smart` | Grep limit (at most 1000 rows are shown) and case rule (`smart`, `ignore`, `match`) |
| `searchUrl` | `https://duckduckgo.com/?q=%s` | Web engine: `http(s)` with one `%s` |
| `terminal`, `terminalExecFlag` | empty, unset | Terminal program, and the flag before the command (known for kitty: none; foot and alacritty: `-e`; wezterm: `start --`; gnome-terminal: `--`) |

Hints and error texts of the modes are fixed text of the module. Prism's own three new colors are `matchRole` (highlighted text), `messageRole` (hint line) and `errorRole` (error line).

No keyboard command is added: the modes are chosen by the prefix in the window opened by your binding, Enter and a click act on a row, and the keyboard rules above are unchanged.

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
| Colors | a role (one of the 16 theme roles) for each part: `windowRole`, `borderRole`, `inputRole`, `inputTextRole`, `placeholderRole`, `prefixRole`, `titleRole`, `subtitleRole`, `hintRole`, `selectedRole`, `selectedTextRole`, `hoverRole`, `separatorRole`, `footerRole`, `emptyRole`, `matchRole`, `messageRole`, `errorRole`, `shadowRole`, `backdropRole`; and an opacity for `window`, `border`, `input`, `selected`, `hover`, `shadow` and `backdrop` (`windowOpacity` and so on) |
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
| `slot.setRows(rows, keepSelection)` | Replaces this source's rows. A row is `{id, title, subtitle?, hint?, icon?, marks?}` (title, subtitle and hint are cut at 200 characters; ids are unique per source). `marks` (version 1.1) is a list of `[start, end]` ranges in the title, in characters, that the window draws in bold and in the `matchRole` color. With `keepSelection` true (version 1.1) the selected row stays selected if its id is still there, instead of the first row being selected: use it for updates of the same query. It counts only after the user moved the selection (arrow keys, paging, Home/End, a click) since the text last changed; before that the first row stays selected, so rows arriving in stages never pull the selection down |
| `slot.setMessage(text, kind)` | Version 1.1. Shows one line between the input and the results; `kind` is `info` or `error`. An empty text clears it. It is cleared when the text of the query changes |
| `slot.setText(text, cursor)` | Version 1.2. Replaces the text of the input, like typing (mode, query and selection follow, messages are cleared). `cursor` is optional: a character index, default the end. Use it only as the answer to `completeRequested` |
| `slot.close()` | Asks Prism to close |
| `completeRequested()` | Signal (version 1.2): the user pressed Tab. Prism does nothing else with the key; a source that can complete the text answers with `setText` |
| `chosen(id)` | Signal: the user chose a row of this source (Enter or double-click). Prism does nothing else with it |

The query and mode reach the source within the same event-loop turn as the key. Slow work belongs in a helper (`api.run`); call `setRows` when it answers. A source that fails or is unloaded loses only its rows. A source cannot move, resize or restyle the window.

A working example is `examples/hello-prism-source/`: copy it to `~/.local/share/lumen/modules/` and enable it (`lumen modules enable hello-prism-source`). Type some text in Prism and three rows appear; type `@f text` and it shows the mode and the query it received; choosing a row closes Prism.

The slot may gain members in a minor version of Prism; a removal or a change of meaning is a major version of Prism with a migration note here.

## Known limits

- Ranking uses no history; running commands is a later feature; the files mode searches names, not contents (use `@g` for that).
- Accent folding covers Latin letters; other scripts are only case-folded (Cyrillic `й` and `ё` are separate letters).
- Grep uses Go regular expressions (RE2).
- Input method composition depends on the compositor's text-input support; it is checked by hand.
- The backdrop dims only; there is no blur (blur depends on the compositor).
- The shadow is drawn behind the whole panel, so with a translucent window background it shows through.
