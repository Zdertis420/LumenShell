# Contract: the `search` helper

A Go program (`scripts/src/search/`, standard library only) built to `modules/prism-search/bin/search` and run by `api.run("search", input, timeoutMs, callback)`. It follows the module helper contract: JSON on stdin, one JSON document on stdout, diagnostics as ndjson on stderr, `protocol: 1`.

```json
{"ok": true, "protocol": 1, "data": { ... }}
{"ok": false, "protocol": 1, "error": {"code": "...", "message": "..."}}
```

`op` selects the job. Every run has `budgetMs`: the program answers by then with what it has (`cut: true`) and exits, so a hung mount cannot keep a process alive past the module's own timeout (which is `budgetMs` + 500 ms). Logs never contain a query, a pattern or a path typed by the user; they name the op and the error code.

## Common types

`Row` (what the source turns into a Prism row): `{key, title, subtitle?, hint?, icon?, marks?}` where `key` is an opaque string the source maps to an action, and `marks` is `[[start,end],…]` in runes of `title`.

## `scope`

Input: `{op:"scope", query, home}` (home is optional, for tests). Output:

```json
{"dir": "/home/u/Documents/work", "name": "report", "hint": ""}
```

`dir` is empty when no directory was named. `hint` is "directory not found: <path>" when a path-like word names a missing directory (the word then stays in `name`). Cheap: one stat per path-like word.

## `apps`

Input: `{op:"apps", query, limit, budgetMs}`. Output: `{rows:[{key (desktop id), title, subtitle (description), icon}], total, cut}`. Empty query lists all, sorted by name, up to `limit`.

## `files`

Input:

```json
{"op":"files", "stage":"priority|home|homeDirs|dotDirs|system|dir",
 "name":"report", "dir":"", "limit":20, "maxFiles":200000, "budgetMs":1500,
 "hidden":false, "hiddenHomeDirs":false, "systemRoots":["/etc","/usr","/opt","/var","/srv"]}
```

Output: `{rows:[{key (path), title (name), subtitle (shortened folder), icon (kind)}], total, cut}`. `total` is the number of matches found within the limits; with `cut: true` the number is a lower bound. For `system`, rows are always returned (the source decides to show them only when expanded). The stage `dir` walks only `dir`; the other stages ignore `dir`. Empty `name`: `{rows:[], total:0}` and `hint` "type a file name".

## `commands`

Input: `{op:"commands", query, limit, extraDirs:[…]}`. Output: `{rows:[{key (path), title (name), subtitle (path)}], total, cut}`.

## `grep`

Input: `{op:"grep", query, limit (max matches), caseRule:"smart|ignore|match", budgetMs}`. The program splits `query` itself. Output: `{rows:[{key ("<path>\u0000<line>"), title (line text), subtitle ("<file>:<line>"), marks}], total, cut, messages:[{kind:"error|info", text}]}`. `messages` carries per-file problems (missing, unreadable, directory, binary) and a pattern error; the source shows the first as the message line. With no pattern or no file: `{rows:[], messages:[{kind:"info", text:"type a pattern and at least one file"}]}`.

## `launch`

Input: `{op:"launch", id}` (desktop-file id). Starts the application detached and answers at once. Errors: `not_found`, `start_failed`, `no_terminal` (a `Terminal=true` entry and no terminal), each with a message naming the application.

## `open`

Input: `{op:"open", path, line?, terminal?, terminalExecFlag?}`. Starts `<terminal> <flag> <EDITOR…> [line args] <path>` detached (research R10). Errors and their messages:

| Code | Message (shown on the message line) |
|------|--------------------------------------|
| `no_editor` | `EDITOR is not set. Set it in the environment that starts Lumen (a shell file such as .bashrc is not read by the shell). For niri: environment { EDITOR "nvim" } in config.kdl, then restart the session.` |
| `no_terminal` | `No terminal: set modules.prism-search.terminal or the TERMINAL variable (see EDITOR note).` |
| `is_directory` | `Only files are opened.` |
| `not_found` | `The file no longer exists: <name>` (the base name only) |
| `start_failed` | `Could not start <program>.` |

## `web`

Input: `{op:"web", query, searchUrl}`. Builds the URL (research R13) and starts `BROWSER` detached. Errors: `no_browser` (message as `no_editor`, with `BROWSER`), `bad_engine` (the template is invalid; cannot normally happen because the setting is validated), `too_long`, `start_failed`.

## `complete` (refinement 2026-11-10)

Request: `{op: "complete", mode: "files"|"grep", query, budgetMs?}`. Completes the last raw word of `query` (the text after the mode prefix).

- Files: only when the last word is path-like (`~`, `/…`, `~/…`, `./…`); candidates are directories. Grep: only when there is more than one word (the first is the pattern); candidates are files and directories. A query ending in a space, or any other case, answers `changed: false` with no message.
- The directory is the word up to its last `/` (`~` and `./` as in `scope`; other relative words from the home); only that directory is listed (at most 5,000 entries). Prefix matching is exact in case, else ignoring case; hidden names only when the typed rest starts with `.`.
- Answer `data`: `{query, cursor, changed, message?}`. One candidate: the full name (directory with `/`). Several: the longest common start, and `message` `{text: "N matches: a, b, c…", kind: "info"}` (at most 8 names). None: unchanged, `message` "no match". A missing or unreadable directory: unchanged, `message` kind `error` ("directory not found: …" or "cannot read: …"). A name with a space or single quote is written in double quotes, `cursor` before the closing quote for a directory; a name containing a double quote is not offered.
- Logs counts only (never names): `helper:search` debug "complete mode=… candidates=N".

## Exit and failure

Exit status is 0 whenever a JSON document was written. A panic or a crash is turned into a `failed` run by the module system; the source shows a message "search failed" (no text) and the other stages continue.
