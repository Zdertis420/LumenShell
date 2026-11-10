---

description: "Task list for the Prism Modes feature"
---

# Tasks: Prism Modes

**Input**: Design documents from `/specs/006-prism-modes/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md. Features I to V (`specs/001-foundation/` to `specs/005-prism-window/`) are implemented. Module interface 1.3 and the Prism 1.0.0 window are in place.

**Tests**: Included, as in earlier features. The constitution requires automated tests for backend logic (search is backend logic) and a check for each setting, so each mode gets black-box Go tests of its helper operation, the pure QML files get `qmltestrunner` tests, the static scans are extended, and the rest is exercised in the quickstart on a real session.

**Organization**: As requested, **each mode is a phase**: Apps (US1), Files (US2), Commands (US3), Grep (US4), Web (US5), Default (US6). US7 (settings) is the last phase before polish. The shared pieces (helper skeleton, match rules, Prism 1.1.0, source module shell) are in Phase 2. `Source.qml` and `stages.js` are touched by every mode phase, so those tasks are in order and not marked [P]; each mode's Go file and Go test file are separate, so they are [P] inside their phase.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on unfinished tasks)
- **[Story]**: Which user story the task belongs to (US1 to US7)

## Path Conventions

Paths are relative to the repository root (see plan.md "Source Code"). Search code is `scripts/src/search/` (standalone Go `package main`, stdlib only); the source module is `modules/prism-search/`; Prism changes are in `modules/prism/`; tests are in `scripts/tests/`; contracts are in `specs/006-prism-modes/contracts/`. Every file must pass `make fmt`, `make lint` and `make test`.

---

## Phase 1: Setup (Spikes)

**Purpose**: Settle the four system facts the plan could not read from documentation. Each runs outside the repository and records its result and chosen branch in `specs/006-prism-modes/research.md`; the plan and contracts are adjusted if a fallback is taken.

- [X] T001 Spike S4 (environment): add `environment { EDITOR "nvim"; TERMINAL "foot"; BROWSER "firefox" }` to a test niri config, restart the session, and record from a throwaway Quickshell `Process` whether the variables reach the shell and the processes it starts; record the working recipe (niri block, or `~/.config/environment.d/*.conf`) in `research.md` (R14) and in the error text of `specs/006-prism-modes/contracts/search-helper.md`
- [X] T002 [P] Spike S3 (terminal flags): for each installed terminal (foot, alacritty, kitty, wezterm, gnome-terminal) run `<terminal> <flag> sh -c 'echo ok; sleep 1'` with the candidate flags; record the working flag in `research.md` (R10) as the table the helper will use
- [X] T003 [P] Spike S1 (detached start): write a throwaway helper that starts `sleep 300` detached (new session) and run it through a throwaway module with `api.run`; reload the module and the shell and record whether the process survives; if not, test `systemd-run --user --scope` and record the chosen branch in `research.md` (R7)
- [X] T004 [P] Spike S2 (home scan speed): on the real home (or a generated tree with 200,000 files) time a Go `filepath.WalkDir` with a name filter over `~/Documents`, over `~` (depth 1) and over the non-hidden folders of `~`, cold and warm cache; record the times and whether stage parallelism helps in `research.md` (R9)

**Checkpoint**: Four results and chosen branches are in `research.md`; plan and contracts match them.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The helper skeleton, the shared match rules, the Prism 1.1.0 additions and the source module shell. No mode can start before this phase is done.

**⚠️ CRITICAL**: No mode phase can begin until this phase is complete.

- [X] T005 Create the Go program `scripts/src/search/main.go`: read one JSON request from stdin, dispatch on `op`, write one `{"ok","protocol":1,"data"}` or `{"ok":false,"protocol":1,"error":{"code","message"}}` document to stdout, ndjson diagnostics to stderr (never the query, pattern or typed path), a panic guard that turns a panic into an error document, and a deadline helper that returns what it has at `budgetMs` (contracts/search-helper.md "Common types" and "Exit and failure")
- [X] T006 [P] Create `scripts/src/search/match.go`: case fold with `unicode.ToLower`, Latin accent removal with a small table (Latin-1 and Latin Extended-A), split of the query into words, `Match(words, name)` requiring every word as a substring, and the rank tuple (all words are prefixes of a name word, position of first match, name length, name) per research R8
- [X] T007 [P] Create `scripts/tests/search_match_test.go`: tests for folding (case, accents, Cyrillic case, no `й`/`ё` merge), all-words in any order (`report final` finds `final_report.pdf`), empty word lists, and the ranking order (spec FR-006)
- [X] T008 [P] Create `scripts/src/search/words.go`: a quote-aware word splitter used by `scope`, `grep`, `EDITOR` and `BROWSER` parsing (single and double quotes, `\` inside double quotes, unbalanced quote treated as literal) and a `~` expander; create `scripts/src/search/terminal.go` with the shared terminal resolver (setting, then `TERMINAL`; exec-flag table from spike S3; error `no_terminal` with the exact message) used by both `launch` and `open`; create `scripts/tests/search_words_test.go` with the cases for both files
- [X] T009 Add `MODULE_PROGRAMS := search` to `Makefile` so that `make build` also builds `scripts/src/search/*.go` into `modules/prism-search/bin/search`, extend the `lint` loop to `go vet` it, and add `modules/prism-search/bin/search` to `.gitignore` (research R2)
- [X] T010 Create `modules/prism/rows.js` (pure): row validation moved out of `Main.qml` plus the new `marks` rule (list of `[start,end]` rune ranges inside the title, invalid ranges dropped with a warning) and the keep-selection rule (keep the selected row by id, else the nearest index) per `contracts/prism-source-slot-1.1.md`; create `scripts/tests/prism_slot_1_1_test.qml` testing both rules; widen the `qmltest` glob in `Makefile` if needed
- [X] T011 Update `modules/prism/Main.qml` to resolve `matchRole`, `messageRole` and `errorRole` into `look.colors` with the same role-fallback rules and error naming as the other color settings (the `baseColor` path), to use `rows.js`, accept `slot.setRows(rows, keepSelection)`, add `slot.setMessage(text, kind)` (one message per source, last set wins, cleared when the query text changes, cut to 200 characters), and expose `message` and `messageKind` to the window
- [X] T012 Update `modules/prism/module.json` to version `1.1.0` and add the settings `matchRole` (default `primary`), `messageRole` (`onSurfaceVariant`) and `errorRole` (`error`) with the same 16-role enum as the other role settings (contracts/search-settings.md, Prism section); update `scripts/tests/prism_settings_test.go` so the setting count and the role-list agreement with `Theme.qml` cover the three new keys; extend the WCAG AA contrast check in `modules/prism/geometry.js` to the pairs (`matchRole`, `messageRole`, `errorRole`) against the row, selected-row and window backgrounds (a violating choice is corrected or reported with a warning naming the key, per Principle II) and add the cases to `scripts/tests/prism_geometry_test.qml`
- [X] T013 [P] Update `modules/prism/ResultRow.qml` to draw the title with the `marks` ranges in the `matchRole` color (rich text built from validated ranges, text escaped), keeping the row height and the ellipsis
- [X] T014 [P] Update `modules/prism/PrismWindow.qml` to show the one-line message between the input and the results, using the empty-state font and `messageRole` / `errorRole`, hidden when empty, not selectable and not part of the row list
- [X] T015 Create the source module shell: `modules/prism-search/module.json` (id `prism-search`, `api` `1.3`, permissions `contribute:prism` and `helpers`, helper `search` at `bin/search`, contribution `search` to `prism/sources` with file `Source.qml`, the 13 settings of contracts/search-settings.md with defaults, ranges and descriptions), `modules/prism-search/Main.qml` (an empty `Scope` that sets nothing), and a first `modules/prism-search/Source.qml` (root `Item` with `required property var api`, `property var slot`, reacts to `slot.mode` and `slot.query`, keeps the query number, cancels earlier `api.run` requests, shows the "search helper not built" message when a run fails to start)
- [X] T016 [P] Create `modules/prism-search/stages.js` (pure) with the stage table of data-model.md per mode, `release(stages)` returning the rows of the longest ready prefix, the "stale answer" check by query number, and the row builder that maps a helper row to a Prism row with a short id (`a1`, `f12`, …) and records the action in an `actions` map (research R6)
- [X] T017 [P] Create `scripts/tests/search_stages_test.qml`: tests for ordered release (a later stage ready first is held back), stale answers ignored, failed stage treated as ready and empty, short ids unique and mapped, and that the web row is never released before the other stages of the default mode
- [X] T018 [P] Create `scripts/tests/prism_search_settings_test.go`: the 13 settings (defaults, ranges, `systemRoots` and `commandDirs` absolute paths only, `searchUrl` with exactly one `%s` and `http(s)`, `grepCase` enum, unset `terminalExecFlag`), a bad value falls back for its key only and the error names `modules.prism-search` and the key
- [X] T019 Extend `scripts/tests/no_magic_values_test.go`, `scripts/tests/module_api_usage_test.go` and the no-network scan to cover `modules/prism-search/` and `modules/prism/rows.js`; add `scripts/tests/fixtures/search/` builders (temp `HOME`, `XDG_DATA_HOME`, `XDG_DATA_DIRS`, `PATH`, and a fake-program writer that records its arguments to a file) used by the mode tests

**Checkpoint**: `make build` produces the helper; `make lint` and `make test` pass; Prism 1.1.0 shows a message and marks in a hand-made source; `prism-search` loads and is listed by `lumen modules show prism-search`.

---

## Phase 3: Apps mode `@a` (User Story 1, Priority: P1) 🎯 MVP

**Goal**: `@a fire` lists matching applications from the desktop entries (system and user), once each, and Enter launches the chosen one.

**Independent Test**: With a fixture set of desktop entries, type `@a` plus part of a name; the matching visible entries are listed once, user entry first; Enter launches and closes; a failing launch keeps Prism open with an error (quickstart scenario 1).

- [X] T020 [P] [US1] Create `scripts/tests/search_apps_test.go`: fixture entries for visible, `Hidden=true`, `NoDisplay=true`, `Type=Link`, `OnlyShowIn`/`NotShowIn` with `XDG_CURRENT_DESKTOP`, a failing `TryExec`, a malformed file, the same id in user and system directories (user wins), a localized `Name[ru]`, a nested directory id, a non-`.desktop` file; test matching on Name, GenericName and Keywords, empty query sorted by name, cache invalidation on adding and removing an entry, an unreadable cache file; and `launch` of an entry with field codes (`%f`, `%U`, `%%`), `Path=`, `Terminal=true` with and without a terminal, and a missing id, using a fake program that records its arguments
- [X] T021 [US1] Create `scripts/src/search/apps.go` (using the terminal resolver of T008): directory discovery from `XDG_DATA_HOME` and `XDG_DATA_DIRS`, a Desktop Entry parser (main group only, localized keys), the visibility rules of research R7, the derived cache `$XDG_CACHE_HOME/lumen/prism-apps.json` keyed by directory and file modification times (falls back to direct parsing on any cache error), `apps` op (query, limit, budget) and `launch` op (field-code stripping, `Path`, `Terminal=true` through the shared terminal resolver of `terminal.go`, detached start per spike S1, error codes `not_found`, `start_failed`, `no_terminal` with messages naming the application)
- [X] T022 [US1] Extend `modules/prism-search/Source.qml` and `modules/prism-search/stages.js` for mode `apps`: one `apps` run per query with `limit` from `rowsPerGroup`, rows built with icon, name and description, an empty-query listing sorted by name, a hint message for the empty state of the mode, and the `chosen` handler that runs `launch` and closes Prism on success or shows the helper's error message (kind `error`) and stays open
- [X] T023 [US1] Add the apps cases to `scripts/tests/search_stages_test.qml` (single stage, hint for empty query) and run `make test`; then run quickstart scenario 1 on a real session and record the first-match time against SC-001 in `specs/006-prism-modes/research.md`

**Checkpoint**: `@a` finds and launches applications; this is a usable MVP.

---

## Phase 4: Files mode `@f` (User Story 2, Priority: P1)

**Goal**: `@f report` shows files in the strict group order with a collapsed system group; `@f report ~/dir` searches only that directory; Enter opens the file in `EDITOR`.

**Independent Test**: With files named alike in each group location, check the order, the collapsed system group and its count, the directory form, the limits, and opening in the editor with and without `EDITOR` (quickstart scenarios 2 to 4).

- [X] T024 [P] [US2] Create `scripts/tests/search_scope_test.go` with cases for the `scope` op: `~`, `/abs`, `~/x`, `./x`, a path with quotes and spaces, a missing directory (stays in the name, hint returned), the default-mode style query with no directory, and words after the directory
- [X] T025 [P] [US2] Create `scripts/tests/search_files_test.go`: a fixture home with matches in `Desktop`, `Downloads`, `Documents`, `~`, `~/notes`, `~/.config`, a hidden `~/.secret` and a system-root directory given through `systemRoots`; test stage contents and order, the system stage returning a count and rows, `dir` stage scope only, hidden rules (`hidden`, `hiddenHomeDirs`), symlink to a directory not followed, a link loop, a hard link counted once, an unreadable directory skipped, `maxFiles` and `budgetMs` cutting with `cut: true`, `limit` and `total`, empty name returning a hint, and that `/proc`, `/sys`, `/dev`, `/run`, `/tmp` are refused as roots; names with spaces, a newline and a 255-byte name are listed and shown correctly; each skipped unreadable directory is counted and logged as one warning per stage (FR-007)
- [X] T026 [P] [US2] Create `scripts/tests/search_open_test.go` cases for the `open` op in files form: `EDITOR` with arguments (`code -w`), the terminal from the setting then from `TERMINAL`, the exec flag table (`-e`, `wezterm start --`, `--`, none), a missing `EDITOR` and a missing terminal giving the exact messages of contracts/search-helper.md, a directory giving `is_directory`, a vanished file giving `not_found`, and that the editor and terminal are started detached; uses fake programs that record arguments
- [X] T027 [US2] Create `scripts/src/search/scope.go`: the `scope` op (research R9): quote-aware words, path-like word detection, existing-readable-directory test, `./` taken from the home, `hint` for a missing directory
- [X] T028 [US2] Create `scripts/src/search/files.go`: the `files` op with the stages `priority`, `home`, `homeDirs`, `dotDirs`, `system`, `dir`; `filepath.WalkDir` without following links, visited-directory and file identity sets by device and inode, unreadable-directory skip, hidden rules, refused pseudo roots, `maxFiles` and `budgetMs` cut (walk in a goroutine, answer at the deadline), `total` and `cut`, sorting by the rank of `match.go`, `limit`, kind and shortened folder (`~`) for the row
- [X] T029 [US2] Create `scripts/src/search/open.go`: the `open` op (research R10): word-split `EDITOR`, terminal resolution (setting, then `TERMINAL`), the line-argument table by editor name, detached start, the error codes and exact messages; use the terminal resolver of `terminal.go` (T008)
- [X] T030 [US2] Extend `modules/prism-search/stages.js` and `Source.qml` for mode `files`: call `scope` first, then start the five stages in parallel (or the single `dir` stage) with settings from `api.settings.values`, publish by ordered release with `keepSelection` true for updates of the same query, build the collapsed "System files (N)" group row (hint "expand"/"collapse"; `N+` when cut), expand on `chosen` with `setRows(rows, true)` and collapse on every new query, the "N more" row of a full group, the `hint` message from `scope`, and the `chosen` handler for a file that runs `open` and closes Prism or shows the error message
- [X] T031 [US2] Add the files cases to `scripts/tests/search_stages_test.qml` (group order, collapsed group, expand keeps the selection, directory form has no system group) and run `make test`; run quickstart scenarios 2 to 4 and record the first-rows time against SC-002 (spike S2 branch) in `research.md`

**Checkpoint**: `@f` works with groups, directory form and opening in the editor; apps still work.

---

## Phase 5: Commands mode `@s` (User Story 3, Priority: P2)

**Goal**: `@s ffm` lists executables from the default command folders, once per name, with the full path; choosing does nothing.

**Independent Test**: An executable, a non-executable file and a duplicate name in different command folders: only executables are listed, one per name, first in resolution order (quickstart scenario 5).

- [X] T032 [P] [US3] Create `scripts/tests/search_commands_test.go`: fixture folders standing in for `~/.local/bin`, `~/bin`, `/usr/local/bin`, `/usr/bin` (through the helper's test override of the folder list), a merged-usr symlink folder searched once, a non-executable file, a directory, a broken link, a duplicate name (first wins), extra folders from `extraDirs`, a missing folder skipped silently, and the all-words matching and ranking
- [X] T033 [US3] Create `scripts/src/search/commands.go`: the `commands` op with the folder order of research R11, real-path de-duplication of folders, executability by mode bits against the process user and groups, one row per name with the full path as subtitle, `limit`, `total`, `cut`
- [X] T034 [US3] Extend `modules/prism-search/stages.js` and `Source.qml` for mode `commands`: one `commands` run with `extraDirs` from `commandDirs`; `chosen` on a command does nothing except keep the row selected (the choice is only reported by Prism, FR-021); hint for an empty query
- [X] T035 [US3] Add the commands cases to `scripts/tests/search_stages_test.qml` (choosing a command never starts a run) and run `make test`; run quickstart scenario 5

**Checkpoint**: `@s` lists commands and never runs one.

---

## Phase 6: Grep mode `@g` (User Story 4, Priority: P2)

**Goal**: `@g <regex> <file>...` lists matching lines with highlights, and Enter opens the file on that line in `EDITOR`.

**Independent Test**: Two small files with known lines: listed lines, numbers and highlights match a reference search; invalid pattern, missing file and binary file each give a message (quickstart scenario 6).

- [X] T036 [P] [US4] Create `scripts/tests/search_grep_test.go`: pattern and several files; quoted pattern with spaces; `~/` file; relative file taken from the home; smart case (no uppercase ignores case, any uppercase is case-sensitive) and the `ignore` / `match` rules; invalid pattern message; no pattern and no file hint; a missing, unreadable and directory file named in a message while others are searched; a binary file skipped with a message; a line over 64 KiB cut; the stop at `limit` with `cut: true`; `marks` offsets in runes for multibyte text after trimming; one row per line with all matches marked; results compared with a reference search for the same fixture (SC-006)
- [X] T037 [P] [US4] Add grep cases to `scripts/tests/search_open_test.go`: `open` with `line` for each editor family of the table (`+N file`, `file:N`, `-g file:N`, unknown editor gets the file alone) and the same `EDITOR`/terminal errors
- [X] T038 [US4] Create `scripts/src/search/grep.go`: the `grep` op per research R12 (word split, `regexp` with `(?i)` by case rule, buffered line reader never loading a whole file, NUL test in the first 8 KiB, 64 KiB line cut, trimmed title with shifted `marks`, subtitle `file:line`, key `path\u0000line`, `messages` list, `limit`, `budgetMs`)
- [X] T039 [US4] Extend `modules/prism-search/stages.js` and `Source.qml` for mode `grep`: one `grep` run with `grepCase` and `grepMaxMatches`, rows with `marks`, the first helper message shown through `setMessage` (kind from the helper), the "results cut" message, and the `chosen` handler that runs `open` with the path and line from the action map and closes Prism or shows the error
- [X] T040 [US4] Add the grep cases to `scripts/tests/search_stages_test.qml` (messages, marks pass through, choose maps to path and line) and run `make test`; run quickstart scenario 6

**Checkpoint**: `@g` searches inside files and opens matches on their line.

---

## Phase 7: Web mode `@w` (User Story 5, Priority: P2)

**Goal**: `@w <text>` offers one row; Enter opens `BROWSER` on the configured engine with the exact text.

**Independent Test**: A query with spaces, accents and symbols reaches a fake browser byte-exact; nothing starts before Enter; unset `BROWSER` gives an error and Prism stays open (quickstart scenario 7).

- [X] T041 [P] [US5] Add web cases to `scripts/tests/search_open_test.go`: URL encoding of spaces, accents, `&`, `#`, `%`, quotes, emoji; `BROWSER` as one program, as colon-separated alternatives (first existing used), with `%s` and without (URL appended); a template without `%s` or with two refused as `bad_engine`; a URL over 2000 characters refused as `too_long`; missing `BROWSER` giving the exact `no_browser` message; the browser started detached without a terminal; that the helper logs no query text
- [X] T042 [US5] Add the `web` op to `scripts/src/search/open.go` (research R13): template check, query-component encoding, length limit, `BROWSER` parsing, detached start, error codes and messages
- [X] T043 [US5] Extend `modules/prism-search/stages.js` and `Source.qml` for mode `web`: build the single "Search the web for «query»" row locally (no run until Enter, FR-027), no row and a hint for an empty query, and the `chosen` handler that runs `web` with `searchUrl` and closes Prism on success or shows the error message and stays open
- [X] T044 [US5] Add the web cases to `scripts/tests/search_stages_test.qml` and run `make test`; run quickstart scenario 7 and check with a fake `BROWSER` that logs each call that it is never called while typing and exactly once after Enter

**Checkpoint**: `@w` opens the browser search; all five prefixed modes work.

---

## Phase 8: Default mode (User Story 6, Priority: P1)

**Goal**: With no prefix, results come in the order apps, home files, collapsed system files, and last the web search suggestion.

**Independent Test**: With matches in every source, a word with no prefix shows the four parts in order; each is absent when empty; the web row is always last and always present for a non-empty query (quickstart scenarios 8 and 9).

- [X] T045 [US6] Extend `modules/prism-search/stages.js` and `Source.qml` for mode `default`: stages `apps`, then the five file stages without calling `scope` (a path in the text is plain text), then the local web row appended last whenever the query is non-empty; no commands or grep rows; no results and no message for an empty query so the empty state of Prism applies; reuse the apps and files stage code unchanged (FR-032)
- [X] T046 [US6] Add the default-mode cases to `scripts/tests/search_stages_test.qml`: final order (apps, home files by group, system group, web), absent parts skipped, web row present when everything else is empty, apps published before the files answer, rows already shown never move while later stages arrive, and a late answer of the previous query is ignored
- [X] T047 [US6] Update `modules/prism-search/Source.qml` so that every update of the same query passes `keepSelection` true and a changed query passes false, and cancel all runs when the window closes (`slot.open` false) or the module unloads; run quickstart scenarios 8 and 9

**Checkpoint**: The default mode behaves as specified; all six modes work.

---

> **Open after the first implementation pass**: T048, T049, T050, T055 and T056 are not done (see "Findings during implementation" in `research.md`). Section 19 of `scripts/tests/smoke.sh` covers the modes end to end with `wtype`, and the wrong-value report; it does not yet cover the live setting changes, the 100-cycle memory check, a stalled mount, or the missing-directory warning.

## Phase 9: Search settings (User Story 7, Priority: P3)

**Goal**: Limits, scopes, engine and terminal follow the settings live, bad values fall back by key, and the report names them.

**Independent Test**: Change each setting in the file while Prism is open; the next search follows within a second; ten wrong values leave all modes working and each is named (quickstart scenario 10 and 11).

- [ ] T048 [US7] In `modules/prism-search/Source.qml` report missing directories of `systemRoots` and `commandDirs` with `api.extensions.report`/`api.log.warn` as one warning each (FR-034), and verify that every setting is read from `api.settings.values` at each query (not cached at load), including `systemGroup` (hides the group and skips the system stage), `rowsPerGroup`, `maxFiles`, `searchTimeMs`, `hiddenFiles`, `hiddenHomeDirs`, `systemRoots`, `commandDirs`, `grepMaxMatches`, `grepCase`, `searchUrl`, `terminal` and `terminalExecFlag`; fix any setting that is only read once
- [ ] T049 [P] [US7] Add scenarios to `scripts/tests/smoke.sh`: change `rowsPerGroup`, `searchUrl`, `matchRole`, `messageRole` and `errorRole` live and check each applies within a second, check that a missing directory in `systemRoots` or `commandDirs` gives one warning in `lumen modules show prism-search`, write ten wrong values and check each is named by `lumen modules show prism-search`, rename the helper and check that Prism and `@w` still work with a message, `chmod 000` a folder and check the search continues, and run 100 open/search/close cycles checking for no process or memory growth
- [ ] T050 [US7] Run quickstart scenarios 10 and 11 and the hung-mount check (SC-011) on a real session and record the results in `specs/006-prism-modes/research.md`

**Checkpoint**: All user stories are done and independently verified.

---

## Phase 10: Polish & Cross-Cutting Concerns

- [X] T051 [P] Update `docs/prism.md`: the six modes and their rules, the query forms of files (`@f name ~/dir`) and grep, what choosing each kind of result does (commands are not run), the `EDITOR` / `TERMINAL` / `BROWSER` requirement with the niri `environment` recipe from spike S4, the setting tables (13 + 3), the slot 1.1 additions with an example, the note that no keyboard command is added (Principle VIII) and the list of limits (RE2 regex, Latin accent folding, no usage ranking)
- [X] T052 [P] Update `docs/layout.md` (where search code goes), `docs/dependencies.md` (optional `EDITOR`, `TERMINAL`, `BROWSER`), `docs/module-api.md` (a pointer only; interface unchanged), `README.md` and `modules/README.md`
- [X] T053 [P] Update `examples/hello-prism-source/Source.qml` to show `setRows(rows, true)`, `setMessage` and `marks` in a few lines each, and keep `module_api_usage_test.go` passing
- [X] T054 Run `make fmt`, `make lint`, `make test` and fix every warning; run `qmllint` and `qmlformat` on all new QML
- [ ] T055 Run the theme check of the quickstart for the new roles: the message line, error line and highlighted text readable in two presets, a light and a dark wallpaper palette, and the fallback palette; record the result in `research.md`
- [ ] T056 Run the complete `quickstart.md` on a real session, tick the manual scenarios, and confirm the constitution check of `plan.md` still holds (no new keyboard command, no network by Prism, no hardcoded values)

---

## Phase 11: Refinement 2026-11-10 (selection stays on the first row, FR-031a, US6)

**Goal**: with all results shown the selection is on the first row until the user moves it; after that the chosen row stays. **Cause**: the first publish can be only the web row; keepSelection then followed it to the end.

- [X] T057 [US6] Add a `navigated` flag to `modules/prism/PrismWindow.qml`: set true when the user moves the selection (Up/Down/PageUp/PageDown/Home/End and a row click or hover select), reset to false in `onTextChanged` and when the window hides; emit a `navigatedChanged` signal (the property itself) for Main
- [X] T058 [US6] In `modules/prism/Main.qml` add a `property bool navigated` fed by the window's `onNavigatedChanged`, reset in `setText` and in the window's `onHidden`; in `setSourceRows` pass `keep === true && root.navigated` to `Rows.selectionAfter` so keepSelection is honored only after the user moved the selection
- [X] T059 [P] [US6] Cover the rule with tests. There is no window-level QML test harness (the window needs a layer shell), so the "no navigation" case is covered by the smoke check T060 (verified to fail without the fix); `Rows.selectionAfter` keep-rules stay covered in `scripts/tests/prism_slot_1_1_test.qml`. Not covered automatically: a Down key followed by a keep=true update (check by hand, quickstart 3.9)
- [X] T060 [US6] Smoke check in `scripts/tests/smoke.sh` section 19: type a default-mode query without touching keys, wait for the published log line with done=true, and confirm the selected row is the first (read from the debug log or a selection log line, counts and kinds only)
- [X] T061 [P] [US6] Update `docs/prism.md` (keepSelection applies only after the user moved the selection) and the "Selection while results arrive" wording where it appears; mention in `examples/hello-prism-source/Source.qml` comment
- [X] T062 Run `make lint`, `make test` and smoke sections 18 and 19; record the result in `research.md` "Findings during implementation"

## Phase 12: Refinement 2026-11-10 (lone `@`, US6)

**Goal**: the text `@` alone is an empty query in the default mode: no search, no rows, no web row, no hint. Longer text such as `@swc` stays a normal query.

- [X] T063 [US6] In `modules/prism-search/stages.js` `plan()`, treat a default-mode query that is exactly `@` after trimming like an empty query (no stages, `web` false, empty hint)
- [X] T064 [P] [US6] Add tests to `scripts/tests/search_stages_test.qml`: `plan("default", "@", ...)` and `plan("default", " @ ", ...)` have no stages, no web row and no hint; `plan("default", "@swc", ...)` still has stages and the web row; the files mode with `@` is unchanged
- [X] T065 [US6] Smoke check in `scripts/tests/smoke.sh` section 19: type `@` into Prism and confirm the last published line reports `rows=0` (the `@s` part is already covered by the commands check)
- [X] T066 [P] [US6] Note in `docs/prism.md` (Search modes) that `@` alone shows nothing until a mode letter follows
- [X] T067 Run `make lint`, `make test` and smoke sections 18 and 19; record the result in `research.md`

## Phase 13: Refinement 2026-11-10 (Tab completion, US8, Priority: P2)

**Goal**: Tab completes the last directory word (files mode) or file word (grep mode). **Independent test**: the Go black-box tests with a fixture home, then quickstart 8b.

- [X] T068 [US8] Refactor `splitWords` in `scripts/src/search/words.go` into `splitSpans(s) []span` (value plus raw rune start/end of each word) and make `splitWords` a wrapper; the behavior of `splitWords` is unchanged (its tests in `scripts/tests/search_words_test.go` stay green); add a test for the spans (quoted word, unbalanced quote, trailing space)
- [X] T069 [P] [US8] Write Go black-box tests `scripts/tests/search_complete_test.go` (fixture home from `search_fixture_test.go`): single candidate (dir gets `/`), several (common prefix and message with count and at most 8 names), none, hidden only after `.`, exact case then ignoring case, name with a space (double quotes, cursor before the closing quote for a directory), a name with a double quote is not offered, grep pattern word never completed, files-mode non-path word unchanged, trailing space unchanged, missing and unreadable directory (message, kind error), a 5,000+ entry directory answers within the budget, relative words from the home, no names in the helper's log
- [X] T070 [US8] Implement op `complete` in `scripts/src/search/complete.go` per `contracts/search-helper.md` (one `os.ReadDir`, limits, rules of FR-038 to FR-042) and register it in the dispatch of `scripts/src/search/main.go`
- [X] T071 [P] [US8] Prism 1.2.0 window: in `modules/prism/PrismWindow.qml` handle `Qt.Key_Tab` (emit `completeRequested()`, accept the event) and add the `cursorRequest {pos, n}` property that places `input.cursorPosition` (clamped) after the text binding updated (`Qt.callLater`); set version `1.2.0` in `modules/prism/module.json`
- [X] T072 [US8] In `modules/prism/Main.qml` forward the window's `completeRequested` to every source slot (new `signal completeRequested()` of the slot component) and add `slot.setText(text, cursor)` (string check, then `setText`, then set `cursorRequest`), per `contracts/prism-source-slot-1.2.md`
- [X] T073 [P] [US8] Add `completable(mode)` (true for `files` and `grep`) to `modules/prism-search/stages.js` and a test in `scripts/tests/search_stages_test.qml`
- [X] T074 [US8] In `modules/prism-search/Source.qml` handle `onCompleteRequested`: only when `Stages.completable(mode)`, call helper `complete` (stale-guarded by `seq` and by the text), apply with `slot.setText(prefix + answer.query, prefix.length + answer.cursor)` where the prefix is the text before `slot.query`, keep a `note {text, kind, forText}` that `publish()` shows while `slot.text === note.forText` and no stage message exists, show an unchanged-text message at once; debug log counts only ("search completed changed=B count=N")
- [X] T075 [US8] Smoke check in `scripts/tests/smoke.sh` section 19: `@g needle notes/rep` then Tab publishes one grep row from `report-note.md`; Tab in `@a` changes nothing (no new published line); no QML errors
- [X] T076 [P] [US8] Docs: `docs/prism.md` (the Tab key in the keys list, the completion rules with examples, `setText` and `completeRequested` in the sources table), `docs/module-api.md` (Prism's slot gained them; interface unchanged), `README.md` and `modules/README.md` if they list keys, a comment in `examples/hello-prism-source/Source.qml`
- [X] T077 [US8] Run `make lint`, `make test` and smoke sections 18 and 19; record the result and what was not verified in `research.md`

## Phase 14: Refinement 2026-11-10 (prefix highlight cropped)

**Goal**: the highlight behind a recognized prefix (`@s`) shows its whole rounded outline.

- [X] T078 In `modules/prism/PrismWindow.qml` widen the clip Item around the prefix highlight by `margin = Math.min(win.sizes.rowGap, win.sizes.inputPadding)` on all four sides (x and y shifted by `-margin`, width and height grown by `2 * margin`) and place the Rectangle inside it with the same margin on every side: x = `from.x` (that is `from.x - margin` in the input's space), y = 0, width = `to.x - from.x + 2 * margin`, height = the Item's height, so only the scroll of a long text is clipped
- [X] T079 Check on the real session: no screenshot tool (`grim`) is installed here, so confirm that Prism still loads with no QML errors (smoke 18 and 19) and record that the look is unverified by eye; the user checks quickstart 8c
- [X] T080 Run `make lint`, `make test` and smoke sections 18 and 19; record the result and what was not verified in `research.md`

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (spikes)**: no dependencies; T002 to T004 are parallel. T001's result is needed by the error text in T029.
- **Phase 2 (foundational)**: needs Phase 1 results only where a task cites a spike (T029 later). Blocks all mode phases.
- **Phases 3 to 8 (one per mode)**: each needs Phase 2. Their Go files and Go tests are independent, but `Source.qml` and `stages.js` are shared, so the phases are done one after another in the order written. Apps (Phase 3) must come first: it introduces the terminal resolver used by `launch`, and `open.go` (Phase 4) shares it. Web (Phase 7) extends `open.go` from Phase 4. Default (Phase 8) needs Phases 3, 4 and 7.
- **Phase 9 (settings)** needs Phases 3 to 8. **Phase 10 (polish)** needs everything.

### Within a mode phase

Tests (Go black-box, then the stage tests) → Go operation → `stages.js` / `Source.qml` → real-session check. Tests are written first and must fail before the operation exists.

### Parallel opportunities

- Phase 1: T002, T003, T004.
- Phase 2: T006 and T007 and T008; T013 and T014; T016, T017, T018 (after T015 for T018's module path).
- Inside Phases 3 to 7: the Go test files named [P] (T020; T024 to T026; T032; T036 and T037; T041) can be written at the same time as each other, and, with two people, the Go work of Commands (T032, T033), Grep (T036, T038) and Web (T041, T042) can proceed in parallel with the QML of an earlier phase, as they touch only their own Go files.
- Phase 10: T051 to T053.

### Parallel example: Phase 4 (Files)

```text
T024  scope tests          scripts/tests/search_scope_test.go      [P]
T025  files tests          scripts/tests/search_files_test.go      [P]
T026  open tests           scripts/tests/search_open_test.go       [P]
then T027 scope.go, T028 files.go, T029 open.go (separate files),
then T030 Source.qml / stages.js (shared, in order)
```

## Implementation Strategy

### MVP first

Phases 1, 2 and 3: apps mode with launching. Prism already becomes a launcher. Stop and validate with quickstart scenario 1.

### Incremental delivery

Add one mode per phase, validating each with its quickstart scenarios: Files (P1) next, since the default mode needs it; then Commands, Grep, Web (P2); then Default (P1, which assembles apps, files and web); then settings (P3). Each phase leaves the shell working and the earlier modes unchanged.

### Notes

- Do not mark the Prism 1.1.0 tasks (T010 to T014) as done until a hand-made source shows a message and marks on a real session; every later phase depends on them.
- The helper binary is built, not committed; run `make build` after changing `scripts/src/search/`.
- Commit after each phase with a conventional prefix (`feat(prism-search): add files mode`).
