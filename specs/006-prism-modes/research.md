# Research: Prism Modes

Decisions that the plan rests on. "Spike" marks a fact about the system that reading cannot settle; each has a fallback, so none blocks the plan.

## R1. Where the modes live: one bundled source module with one Go helper

- **Decision**: A new bundled module `modules/prism-search/` contributes one source (`Source.qml`) to `prism/sources`. All six modes are handled in that one source. The work is done by one Go program, `scripts/src/search/`, with one operation per job (`scope`, `apps`, `files`, `commands`, `grep`, `launch`, `open`, `web`). The module uses only `api` and runs the program through `api.run` (permission `helpers`).
- **Rationale**: Feature V made search a source on purpose (FR-001, no window change). One source can order the default mode's four parts itself (apps, home files, system files, web); with several sources the order would depend on contribution order. Matching, ranking, walking, parsing and launching are heavy or error-prone and belong in Go (Principle I).
- **Alternatives**: One module per mode (five manifests, five helpers; the default mode would need a sixth to merge them, or rely on contribution order). Rejected. Doing the walk in QML (`FileView`, `Process`): blocks nothing but cannot be unit tested and breaks Principle I.

## R2. Shipping a compiled helper with a bundled module

- **Decision**: The Makefile builds `scripts/src/search` into `modules/prism-search/bin/search` (new variable `MODULE_PROGRAMS`, output gitignored). The manifest declares `{"name": "search", "path": "bin/search"}`.
- **Rationale**: `api.run` runs only helpers declared by the module, by a path inside its own directory (lexical check). Existing helpers are shell scripts committed in `bin/`; a Go binary is built and cannot be committed. The programs in `scripts/bin` belong to the shell, not to a module.
- **Failure**: On a fresh clone without `make build` the file is missing; the run fails with `failed`, the source shows one error row ("search helper not built, run make build") and the rest of Prism works (Principle IV). `make build` is already the documented first step.
- **Alternative**: A shell wrapper in `bin/` that execs `scripts/bin/search`. Rejected: it hides a path that leaves the module directory.

## R3. Staged parallel runs, ordered release, cancel

- **Decision**: `api.run` answers once, so "results as found" is done by splitting one query into stages, run in parallel as separate helper processes, each with its own time budget. Stages: apps; files in `Desktop`, `Downloads`, `Documents`; files directly in `~`; other directories in `~`; `~/.cache`, `~/.local`, `~/.config`; system files. A new query cancels the runs of the old one (`request.cancel()` kills the process) and ignores any late answer by a query number.
- **Ordered release**: The source keeps the stages in their fixed order and publishes the longest ready prefix (stage 2 is shown only when stage 1 has answered). This means rows only ever appear below rows already shown, so the selection never moves (FR-031) and groups never appear in the wrong order (FR-012, FR-030).
- **Rationale**: No change to `api.run` and no streaming protocol; a stage that is slow delays only the stages after it, which is the order the user wants anyway. The web row is appended at the end, always (it needs no helper).
- **Alternative**: A streaming helper over a socket. Rejected: a new interface for one feature.

## R4. Keeping the selection while rows arrive (a small Prism change)

- **Finding**: Feature V resets the selection to the first row whenever a source sets rows (data-model.md, `selectedIndex`). With staged results the user's arrow-key position would jump back. Expanding the system group has the same problem.
- **Decision**: Prism 1.1.0 adds an optional second argument, `slot.setRows(rows, keepSelection)`. When true, Prism keeps the selected row by its id if it is still there, and otherwise the nearest index. The source passes `true` for every update of the same query and `false` when the query changed. Old sources are unaffected. **Refined 2026-11-10 (FR-031a):** Prism honors `true` only after the user moved the selection since the text last changed (the window keeps a `navigated` flag, reset on every text change and on close). Before that the selection stays on the first row. Reason: the first publish can be only the web row, which is "selected" (row 0), and plain keeping then followed it to the last position. Alternative rejected: the source avoids keep for the first N publishes (fragile, every source would need the rule).

## R5. Highlight, messages and errors need three additions to Prism

- **Finding**: The spec requires highlighted grep matches, hints ("type a pattern"), and errors that keep Prism open (`EDITOR` not set). Feature V has a row look without highlights and an empty-state text only.
- **Decision** (Prism 1.1.0, all additive, documented in `contracts/prism-source-slot-1.1.md`):
  1. A row may have `marks`: a list of `[start, end]` ranges in `title`; the window draws them in the color role `matchRole`.
  2. `slot.setMessage(text, kind)` with `kind` `info` or `error`; Prism shows one line between the input and the results, cleared when the query changes or when the source sets an empty text. Roles: `messageRole`, `errorRole`.
  3. Three new settings of Prism: `matchRole` (default `primary`), `messageRole` (default `onSurfaceVariant`), `errorRole` (default `error`), all with the existing role rules. This takes Prism from 105 to 108 settings, under the limit of 128.
- **Rationale**: These are presentation, so they belong in the window; the source stays unable to style anything (Feature V FR-029).
- **Alternative**: Put hints and errors as unselectable rows. Rejected: they would be ranked, scrolled and counted like results, and need a "not selectable" flag anyway.

## R6. Row identifiers

Row `id` is limited to 200 characters and a path can be 4096. The source keeps a table from a short id (`f12`) to the full action (`{kind: "file", path}`), rebuilt for each query. `chosen(id)` looks it up. Nothing leaves the source.

## R7. Applications

- **Decision**: The helper reads `*.desktop` files from `$XDG_DATA_HOME/applications` and each `$XDG_DATA_DIRS/applications` (defaults `~/.local/share` and `/usr/local/share:/usr/share`), recursing into subdirectories (id is the path with `-` for `/`). The first entry for an id wins, in the order user, then system, which gives the user's precedence (FR-009). Skipped: `Type` other than `Application`, `Hidden=true`, `NoDisplay=true`, an `OnlyShowIn` that does not include the current desktop (`XDG_CURRENT_DESKTOP`) or a `NotShowIn` that does, and a `TryExec` that does not resolve. Names use `Name[ll]` by `LANG`, then `Name`. Matching fields: Name, GenericName, Keywords.
- **Cache**: parsed entries are kept in `$XDG_CACHE_HOME/lumen/prism-apps.json`, valid while the modification times of all application directories (and of the files in them) are unchanged. A stat of ~500 files costs a few ms; this meets SC-001 (100 ms) and FR-011 (a change shows on the next search, at most 5 seconds later because the check runs per query). If the cache cannot be read or written the helper parses directly.
- **Launch**: Go, per the Desktop Entry spec: strip the field codes (`%f %F %u %U %i %c %k`, `%%` to `%`), honor `Path`, run `Terminal=true` entries inside the terminal of R10, start detached (new session, no inherited input) so the app outlives Prism. Chosen over Quickshell's `DesktopEntries.execute` because the module may use only `api` and the helper is testable.
- **Spike S1**: confirm on niri that a process started by the helper (a child of the shell) survives a shell reload. Fallback: start through `systemd-run --user --scope` when it is installed, else `setsid` only.

## R8. Name matching

- **Decision**: Fold the case with `unicode.ToLower` and remove Latin accents with a small table (the Latin-1 and Latin Extended-A letters, about 120 entries) instead of a Unicode normalization package (stdlib only). Cyrillic letters are case-folded; `й`/`ё` are not merged with `и`/`е` because they are separate letters. Words are split on spaces; every word must be a substring of the folded name (spec FR-006).
- **Rank**: lowest tuple first: (all words are prefixes of a name word? 0 : 1; position of the first match; name length; name). Stable and testable; no history.
- **Alternative**: golang.org/x/text/unicode/norm. Rejected for now (dependency); revisit if the table proves too narrow.

## R9. Files

- **Scope parsing** (`scope` op, one fast call): split the query into words (quotes allowed); a word that is `~`, or starts with `/`, `~/` or `./`, and resolves to an existing readable directory is the directory (a `./` path is taken from `~`, as the spec says); the rest is the name. If a path-like word names a missing directory, it stays in the name and a hint says "directory not found". The default mode never parses a directory.
- **Walk**: `filepath.WalkDir` does not follow links; each walk keeps a set of visited directories by device and inode (guards bind mounts) and files by device and inode (a hard link or a second path counts once); unreadable directories are skipped; the walk stops at the stage's `maxFiles` or `searchTimeMs`, and says it was cut. A hung mount is handled by running the walk in a goroutine and returning at the deadline; the process then exits and drops the stuck goroutine.
- **Groups** (FR-012), as stages: (1) `~/Desktop`, `~/Downloads`, `~/Documents` in that order and at any depth; (2) files directly in `~` (depth 1, no recursion); (3) non-hidden directories of `~` other than those three, at any depth; (4) `~/.cache`, `~/.local`, `~/.config`; (5) system roots. Hidden entries inside a searched directory are skipped unless `hiddenFiles` is on; hidden home directories other than the three of stage 4 are searched only if `hiddenHomeDirs` is on (they join stage 3).
- **System roots**: default `/etc`, `/usr`, `/opt`, `/var`, `/srv` (a stringlist setting). `/proc`, `/sys`, `/dev`, `/run` and `/tmp` are never walked, even if listed. The stage returns the number of matches (up to `maxFiles`, shown as `N+` when cut) and only the first `rowsPerGroup` rows, which are held back until the user expands the group.
- **Group rows**: the source builds a title row ("System files (N)") with the id of the group; Enter or a click on it expands it (`setRows(rows, true)`); the "N more" row of a full group is a message row that does nothing except show "refine the search". A new query collapses all.
- **Spike S2**: measure stage 1 and stage 3 on a real home with 200,000 files to confirm SC-002 (200 ms for the first rows). Fallback: walk stage 1 and 2 first and start the others 50 ms later.

## R10. Opening files, the editor and the terminal

- **Decision**: `open` reads `EDITOR` (split into words with quote rules, so `code -w` works) and the terminal (setting `terminal`, else `TERMINAL`); it starts `<terminal> <exec-flag> <editor words…> <line args> <file>` detached. Missing `EDITOR`: error `no_editor`; missing terminal: error `no_terminal`. Both messages name the variable, say that a shell startup file such as `.bashrc` is not read by Lumen, and show the fix for niri (`environment { EDITOR "nvim" }` in `config.kdl`, which applies to programs niri starts, or a user environment file). Nothing is guessed (FR-005a, FR-005b).
- **Exec flag** by terminal program name, overridable by the `terminalExecFlag` setting: default `-e`; `wezterm` uses `start --`; `gnome-terminal` and `kgx` use `--`; `kitty` uses none (it takes the command as arguments). **Spike S3**: confirm `-e`, `--` and none against the installed terminals (foot, alacritty, kitty, wezterm); the table is data in the helper, so a correction is one line.
- **Line argument** by editor program name (a table in the helper): `+N file` for vi, vim, nvim, view, nano, emacs, emacsclient, kak, joe, mcedit, jed; `file:N` for hx, helix, micro, zed, zeditor; `-g file:N` for code, codium, cursor; unknown editors get the file alone. Grep matches use the line; plain files use none.
- **Directories**: a file row that is a directory (a symlink to one) opens nothing and gives the message "only files are opened".

## R11. Commands

- **Decision**: Directories, in the order the shell would resolve them: `~/.local/bin`, `~/bin`, `/usr/local/bin`, `/usr/local/sbin`, `/usr/bin`, `/usr/sbin`, `/bin`, `/sbin`, `/opt/bin`, then the user's `commandDirs`. A directory that resolves to the same real path as an earlier one (merged `/usr`) is searched once. A name is listed once (first wins). Only regular files or links to them with an execute bit for the user (`unix.Access` is not stdlib; use the mode bits and the owner/group/other test with the process ids).
- The mode searches only; choosing a row reports it and does nothing (spec FR-021).

## R12. Grep

- **Decision**: Go `regexp` (RE2): no look-ahead or back-references; documented, with the invalid-pattern message quoting the parser's error. Words are split with quote rules (single or double quotes; `\` escapes inside double quotes). Smart case: add `(?i)` when the pattern has no uppercase letter (setting `grepCase`: `smart`, `ignore`, `match`). Files are read through a buffered reader, line by line, never whole; a line over 64 KiB is cut; a NUL byte in the first 8 KiB marks a file as binary and skips it. The output row title is the line text trimmed to 200 characters (leading blanks removed, `marks` shifted), subtitle `file:line`.
- Stops at `grepMaxMatches` (default 500) and says so. `~/` is expanded; relative paths are taken from the home directory (Prism has no working directory).

## R13. Web

- **Decision**: The engine is a setting `searchUrl` holding a URL template with `%s` (default `https://duckduckgo.com/?q=%s`); it must be `http` or `https` and contain `%s` once, else the key falls back with an error. The helper percent-encodes the query as a query component and refuses a URL over 2000 characters. `BROWSER` follows its convention: colon-separated alternatives, each possibly with `%s` for the URL; the first whose program exists is used; with no `%s` the URL is the last argument. Started detached, no terminal. Missing: error `no_browser` with the same guidance as `no_editor`. Nothing touches the network before Enter (FR-027); Prism itself never opens a socket (a static scan already exists for the repository and is extended).
- **Privacy**: the query is not stored; helper logs name the operation and the error code, never the text (FR-029).

## R14. Environment seen by the helper

The helper inherits the environment of the shell process (the Quickshell started by niri). A variable set only in an interactive shell file does not reach it. This is why the error states the cause and the fix. **Spike S4**: confirm that `environment { … }` in niri's config reaches the processes the shell starts. Fallback text: set it in `~/.config/environment.d/*.conf` (systemd user environment) and restart the session.

## Spike results (2026-11-10, checked on the developer's niri session)

- **S1 (detached start)**: not run against a shell reload in this pass (it needs a person to reload the shell). The helper starts processes with a new session (`Setsid`), which is the standard way to outlive the parent; the `systemd-run --user --scope` branch stays as the recorded fallback. A by-hand check stays in quickstart scenario 1 (launch an app, run `lumen modules reload prism-search`, the app must stay).
- **S2 (scan speed)**: on this home (28,348 files in `Desktop`, `Downloads`, `Documents`; 102 files directly in `~`; 306,144 non-hidden files in total) a warm `find` took 60 ms for stage 1, 3 ms for stage 2 and 880 ms for the whole non-hidden home. Decision: stages 1 and 2 give the first rows well inside the 200 ms of SC-002; stage 3 is the slow one and is bounded by `searchTimeMs`/`maxFiles`. The stages run in parallel as planned; no delay between them is needed.
- **S3 (terminal flags)**: only `kitty` is installed here. Its own help says "program to run inside kitty as normal arguments", so kitty takes the command with no flag, as in the table. The other entries (`-e` for foot and alacritty, `start --` for wezterm, `--` for gnome-terminal and kgx) come from those programs' documented usage and are untested here; they are data in `terminal.go` and the `terminalExecFlag` setting overrides them.
- **S4 (environment)**: the running `niri` process already has the login shell's `EDITOR=zeditor` and `BROWSER=waterfox` in its environment, so variables exported by the login shell do reach niri and the programs it starts. `TERMINAL` is not set on this machine, so file opening needs `modules.prism-search.terminal` (or `TERMINAL`) here. The niri `environment { … }` recipe is niri's documented way to set variables for its children and is kept as the documented fix; it was not exercised here because that needs a session restart.

## Findings during implementation (2026-11-10)

- **Web row timing (changes R3)**: the web row is not held back until the other stages answer. It is always the last row of the rows released so far, so the user can reach it at once; the selection is kept by id when rows arrive above it. Rows from the other stages still only appear below rows already shown.
- **Measured on the developer's machine** (helper run by hand, warm cache): apps 2 ms with the cache; first file stage (`priority`, 28,348 files) 53 ms; so SC-001 and SC-002 hold with a wide margin. The slow stage is the non-hidden rest of the home (about 0.9 s for 306,144 files), which is bounded by `searchTimeMs` and does not delay the earlier stages.
- **Prism fix found by the real-session test**: `PrismWindow.select()` used the `rowCount` binding from `onRowsChanged`, which is not yet updated when the rows go from none to one, so no row was selected and Enter did nothing. It now reads `rows.length`. This was latent in Feature V.
- **Helper log lines** need `time`, `level`, `source` and `msg` (the helper log contract); a line without `time` broke the `test-surface` module's log view.
- **A walk that starts above `/proc`, `/sys`, `/dev`, `/run` or `/tmp`** (for example a partly typed `/` in the query) does not enter them.
- **Default colors**: `primary` and `error` on `surfaceContainer` give 4.3:1 and 3.9:1 in the dark test palette, and some presets give less for the colors Feature V already used. The check reports such pairs as warnings and keeps the user's choice, so the three new roles follow the same rule.
- **Not verified here**: the look of the marked text, the message line and the collapsed group (by eye), the hung-mount case (SC-011), the search settings applied live to a running window beyond the wrong-value report, and the missing-directory warning of `systemRoots` and `commandDirs` (T048, T049, T050, T055 and T056 stay open).

## Refinement 2026-11-10: selection on the first row (FR-031a)

Fix: `navigated` flag in `PrismWindow.qml` (set by `userSelect`, reset on text change), mirrored in `Main.qml`, which passes keep only when it is true. Result: `make lint` and `make test` pass; smoke sections 18 and 19 pass (69 checks); the new smoke check fails without the fix (2 failures) and passes with it. Not verified: the Down-then-update case (manual, quickstart 3.9) and how it looks.

## Refinement 2026-11-10: a lone `@`

- **Decision**: in `modules/prism-search/stages.js` `plan()`, a default-mode query that is exactly `@` after trimming is planned like an empty query (no stages, no web row, no hint). Prism's prefix parser (Feature V) is unchanged: it still reports mode `default` with query `@`; only the search module decides not to search it.
- **Rationale**: `@` is the start of a prefix, so the user is about to type a letter; searching it listed every name containing `@` (scoped package folders). Longer text such as `@swc` stays a normal query so such names remain findable.
- **Result**: `make lint` and `make test` pass; smoke sections 18 and 19 pass (70 checks, including "a lone @ lists nothing"). Not verified by eye.
- **Alternatives rejected**: changing Prism's parser to a new "pending prefix" mode (a Prism change for one source's concern); showing a hint with the mode letters (the letters are settings, the search module does not know them).

## Refinement 2026-11-10: Tab completion (US8, FR-038 to FR-042)

- **Decision (who does what)**: the window handles Tab and emits `completeRequested`; Main forwards it on every source slot; the source decides whether it applies. The search module asks the helper (`complete` op) only in files and grep modes, writes the answer with the new `slot.setText(text, cursor?)` and shows the helper's message. Prism stays ignorant of files (Principle I and III).
- **Decision (what is completed)**: the last raw word of the query. Files mode: only a path-like word (`~`, `/…`, `~/…`, `./…`), directories only. Grep: every word after the first (the pattern), files and directories. A text ending in a space completes nothing. Rationale: user answered "directory argument" and "files and grep file arguments".
- **Decision (candidates)**: one `os.ReadDir` of the directory named by the word up to its last `/` (relative from the home), at most 5,000 entries and a time budget; prefix match exact case first, else ignoring case; hidden entries only if the typed rest starts with `.`. One candidate: completed (dir gets `/`); several: longest common prefix plus a message with the count and up to 8 names; none: message. No walk, so no hung-tree risk beyond one listing; a slow listing is cut by the helper's run timeout.
- **Decision (quoting)**: names with spaces or quotes are written in double quotes (a name that contains a double quote is not offered for quoting and is skipped with a message, since the word rules have no escape outside quotes). For a directory the cursor is placed before the closing quote, by `setText`'s cursor argument.
- **Decision (word spans)**: `splitWords` becomes a thin wrapper over `splitSpans` returning raw start/end offsets, so the last word can be replaced in the raw text; behavior of `splitWords` is unchanged (its tests stay).
- **Decision (message survival)**: a search `publish()` resets the message line, so the source keeps a `note` (text, kind, the full text it belongs to) and `publish()` shows it while `slot.text` equals that text; typing anything drops it naturally.
- **Decision (Prism 1.2.0)**: additive. `setText` clips nothing (the window has its own limit) and moves the cursor to `cursor` (default: end) through a `cursorRequest {pos, n}` property of the window, applied after the text binding updated. A source that does not know the signal is unaffected.
- **Result (T077)**: `make lint` and `make test` pass; smoke sections 18 and 19 pass (74 checks, 4 new: Tab completes a grep file word on the real session and the editor opens the completed file, Tab logs a completion, Tab in the apps mode completes nothing). The span refactor is covered through the quoted-word cases of `search_complete_test.go` (spans are internal, tests are black-box). Not verified: the cursor position before a closing quote in the real input (set by `cursorRequest`, only the helper's `cursor` value is tested), the look of the message line with the candidates, and a Tab on a hung mount.
- **Alternatives rejected**: completing in Prism itself (would need file access in the window, against Principle I); a popup list of candidates (a new popup, more keyboard rules; the message line is enough here); the Right arrow as the key (conflicts with moving the text cursor); completing name words in the files mode (not asked, and ambiguous with searching).

## Refinement 2026-11-10: the prefix highlight was cropped

- **Cause**: in `PrismWindow.qml` the highlight rectangle starts `rowGap` left of the first letter and ends `rowGap` after the last, but its parent (the clip Item that hides the scrolled part of a long text) had exactly the size of the TextInput, so the left edge with its rounded corners (and the top and bottom of the outline) were cut.
- **Decision**: the clip Item grows by a margin `min(rowGap, inputPadding)` on all four sides (it never leaves the input box, because the margin is at most the input padding) and the rectangle is offset by that margin; the highlight is as tall as the text area plus the margin. Only the scroll of a long text is still clipped. No new setting: the margin follows the existing tokens (Principle VII).
- **Result (T080)**: `make lint` and `make test` pass; smoke sections 18 and 19 pass (74 checks), so the window still loads with no QML errors. **Not verified**: the look itself. No screenshot tool is installed here (`grim`), so nobody has seen the new outline; check quickstart 8c.
- **Alternatives rejected**: removing the clip (a long scrolled text would paint the highlight over the box edge); a new setting for the margin (the tokens already cover it).

## R15. Settings namespace

The search settings are the settings of the module `prism-search`, so they live at `modules.prism-search.*`, and errors name that module and key (spec FR-034 says `modules.prism`; it is corrected to the real namespace). Prism's own three new roles live at `modules.prism.*`. Both are validated by the existing Go rules and shown by `lumen modules show`.
