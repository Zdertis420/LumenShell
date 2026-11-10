# Data Model: Prism Modes

Nothing is stored for the user. In memory: the source's query state. On disk: only the derived applications cache in `$XDG_CACHE_HOME/lumen/prism-apps.json` (safe to delete).

## Query state (in `Source.qml`)

| Field | Type | Rules |
|-------|------|-------|
| `seq` | int | Number of the current query; every started run carries it and a late answer with an older number is ignored. |
| `mode`, `query` | string | From Prism's slot (`default`, `apps`, `files`, `commands`, `grep`, `web`). |
| `runs` | list | The helper requests of this query; all are cancelled when the query changes, the window closes or the module unloads. |
| `stages` | list | Ordered stages of this query (below), each `waiting`, `ready` or `failed`, with its rows. |
| `actions` | map | Short row id to action (`{kind, ...}`), rebuilt per query. |
| `expanded` | bool | The system group is expanded; `false` at every new query. |

## Stage

| Mode | Stages, in order |
|------|------------------|
| `apps` | `apps` |
| `files` (no directory) | `priority`, `home`, `homeDirs`, `dotDirs`, `system` |
| `files` (directory) | `dir` |
| `commands` | `commands` |
| `grep` | `grep` |
| `web` | none (the row is built locally) |
| `default` | `apps`, then the five file stages, then the local web row (always last) |

Published rows are the rows of the longest ready prefix of stages. A failed stage is treated as ready and empty and sets a message when it is the only one.

## Row kinds (the source's side of a Prism row)

| Kind | Prism row | Action when chosen |
|------|-----------|--------------------|
| `app` | icon, name, description | helper `launch` with the entry id |
| `file` | icon of kind, name, folder as subtitle | helper `open` with the path |
| `grepmatch` | title: the line with `marks`; subtitle `file:line`; hint: none | helper `open` with path and line |
| `command` | name, full path as subtitle | none (reported only) |
| `group` | "System files (N)" / "(N+)", hint "expand" or "collapse" | toggles `expanded` |
| `more` | "N more", only in a full group | message "refine your search" |
| `web` | "Search the web for «query»" | helper `web`, then close |

## Helper messages (data crossing the module boundary)

Defined in `contracts/search-helper.md`. Entities there: **Scope** (directory and name words), **AppEntry**, **FileMatch**, **Command**, **GrepMatch**, **Failure** (`code`, `message`).

### AppEntry (helper result)

| Field | Rules |
|-------|-------|
| `id` | The desktop-file id (`org.mozilla.firefox.desktop`); unique; a user entry wins over a system one. |
| `name`, `description`, `icon` | From the entry (localized name); `description` is Comment, else GenericName. |
| `sourcePath` | Path of the file, for the launch. |

Visibility: see research R7. Sort: rank tuple of R8; an empty query sorts by name.

### FileMatch

| Field | Rules |
|-------|-------|
| `path` | Absolute path; one row per file (device and inode). |
| `name`, `dir` | The base name and the folder, the folder shortened with `~` for the home. |
| `kind` | `file`, `directory`; the icon follows it. |
| `group` | One of the stage names. |

Sort inside a stage: rank tuple of R8, then path. A stage shows at most `rowsPerGroup` rows and reports `total` and `cut`.

### Command

`name`, `path` (full path of the first match). One per name.

### GrepMatch

`file`, `line` (1-based), `text` (trimmed, at most 200 characters), `marks` (ranges in `text`, in runes, after the trim). Order: files in the order given, then line order.

### Completion (helper result, refinement 2026-11-10)

`{query, cursor, changed, message?}`: `query` is the whole query with its last word replaced, `cursor` is a rune index in `query` (where the text cursor goes), `changed` says whether the text differs, `message` is `{text, kind}` or absent. In `Source.qml` the module also keeps a `note` `{text, kind, forText}`: the message to show while the window text equals `forText`.

## Settings (module `prism-search`)

See `contracts/search-settings.md`. Layered and override-only as every module setting; `lumen modules show prism-search` prints their state.

## Validation rules carried from the spec

- An empty or whitespace query produces no run and a hint message (FR-004).
- A stage never returns more than 1000 rows (Prism's limit); `rowsPerGroup` is at most 200.
- Texts shown (hints, errors) are the helper's `message` strings or fixed strings of the source; none contains the user's query text except the web row's title (FR-029).
