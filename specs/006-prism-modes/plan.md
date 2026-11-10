# Implementation Plan: Prism Modes

**Branch**: `006-prism-modes` | **Date**: 2026-11-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/006-prism-modes/spec.md`

## Summary

Prism (Feature V) is an empty window with a prefix parser and a source slot. This feature fills it: the six modes (default, apps, files, commands, grep, web) with their searching rules, and the actions that follow a choice (launch an app, open a file or grep match in `EDITOR` inside a terminal, open the web search in `BROWSER`; commands are never run). The pieces:

1. **One bundled source module, `modules/prism-search/`**, with one `Source.qml` contributing to `prism/sources`. It holds the query state, starts helper runs, orders their results and maps a chosen row to an action. It draws nothing and uses only `api`.
2. **One Go helper, `scripts/src/search/`** (standard library only), with the operations `scope`, `apps`, `files`, `commands`, `grep`, `launch`, `open` and `web`. All matching, ranking, directory walking, desktop-entry parsing, regex work and process starting is here (Principle I). It is built by `make build` into `modules/prism-search/bin/search`.
3. **Staged, parallel, cancellable runs.** A query is split into ordered stages (apps; four home file stages; system files), run as separate helper processes; a new query cancels the old runs; results are released in order so rows only ever appear below the rows already shown (research R3). The default mode's web row is built locally and is always last.
4. **Prism 1.1.0 (additive).** Three small additions to the window so the modes can be shown properly: `setRows(rows, keepSelection)`, `setMessage(text, kind)`, and row `marks`; plus three color-role settings (`matchRole`, `messageRole`, `errorRole`). A 1.0 source works unchanged (research R4, R5). `keepSelection` takes effect only after the user moved the selection since the text changed (a `navigated` flag in the window; FR-031a).
4a. **Tab completion (refinement 2026-11-10, US8)**: Prism 1.2.0 adds the Tab key (`completeRequested` signal on every source slot, forwarded from the window) and `slot.setText(text, cursor?)`; the helper gets a `complete` operation (`scripts/src/search/complete.go`, one directory listing, no walk) and `splitWords` is refactored into `splitSpans` to find the raw last word. `Source.qml` asks the helper for files and grep only (`stages.js completable(mode)`), writes the result with `setText` and keeps a note shown by `publish()` while the text equals the completed text. No new settings (108 stay); Tab is a key of the open window, not a command (Principle VIII, documented in `docs/prism.md`). Details: research "Refinement: Tab completion", `contracts/prism-source-slot-1.2.md`, `contracts/search-helper.md`.
5. **Settings**: 13 settings of `prism-search` (limits, scopes, hidden files, system roots, extra command folders, grep rules, search engine, terminal) and 3 of Prism; no change to the module interface (`api` 1.3 stays; the Prism count goes 105 to 108, under the limit 128).
6. **Environment, not guessing.** `EDITOR`, `TERMINAL` and `BROWSER` are read from the environment Lumen runs in; a missing one gives an error that names it and shows how to set it for niri. A shell startup file is not read.
7. **Checks and docs**: Go black-box tests per operation with fixture homes and fake programs, QML unit tests for the ordered release and the slot additions, extended static scans, `docs/prism.md` and layout updates, and a manual quickstart.

No new dependency, no resident process, no network access by Prism itself, IPC protocol stays at 1, no new keyboard command.

## Technical Context

**Language/Version**: Go 1.27 (stdlib only), QML/Qt 6.12 (Quickshell 0.3.2).

**Primary Dependencies**: None new. `os`, `io/fs`, `path/filepath`, `regexp`, `bufio`, `os/exec`, `syscall` (for detached start and inode identity), `encoding/json`, `unicode`.

**Storage**: Files only. Settings in `$XDG_CONFIG_HOME/lumen/config.json` (`modules.prism-search.*`, `modules.prism.*`). One derived cache, `$XDG_CACHE_HOME/lumen/prism-apps.json`, safe to delete. Nothing is indexed on disk for files.

**Testing**: `go test` black-box tests that run the built helper against temp-dir fixtures (own `HOME`, `XDG_DATA_DIRS`, `PATH`, and fake `EDITOR`/`TERMINAL`/`BROWSER` programs that record their arguments); `qmltestrunner` for the pure `.js` files (`stages.js` in the source module, `rows.js` in Prism); the existing static scans extended to `modules/prism-search/`; the quickstart on a real session.

**Target Platform**: Linux, Wayland, niri (as Features I to V).

**Project Type**: Desktop shell (QML UI + small Go programs).

**Performance Goals**: Apps: first match in 100 ms (SC-001). Files: first rows in 200 ms on a home with 200,000 files (SC-002), window smooth meanwhile. Other targets from the spec.

**Constraints**: The UI thread never blocks: every search is a helper process. Each stage has a time budget (default 1.5 s) and a file cap; a hung mount cannot outlive its budget. Late answers are dropped. No network access by Prism; the only outward action is starting the user's browser after Enter. Logs never contain queries, patterns or typed paths.

**Scale/Scope**: 8 helper operations, 1 module, 1 source, 16 new settings (13 + 3), 6 modes.

No NEEDS CLARIFICATION remain; see [research.md](research.md). Four spikes (S1 to S4: detached start survives reload, home scan speed, terminal exec flags, niri `environment` reaching the shell) are facts about the system; each has a fallback, so none blocks the plan.

## Constitution Check

*GATE: passed before Phase 0 and re-checked after Phase 1.* Constitution version 2.0.0.

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Thin QML, heavy logic in Go | PASS | Matching, ranking, walking, parsing, regex, URL building and process starting are in the Go helper. QML keeps the query state, the stage order and the row-to-action map (light glue, unit tested). Backend calls are asynchronous (`api.run`). |
| II. Matugen theming | PASS | No new color in code; the three new roles are theme role names with the existing role rules and the role-list test. |
| III. Modular components | PASS | The modes are a module on the public interface using Prism's documented extension point. Prism's change is additive and versioned (1.1.0), documented with an example. `api` is untouched. |
| IV. Reliability and degradation | PASS | Per-stage budgets, cancel, skip unreadable paths, a failing stage loses only its rows, a missing helper gives one message, errors keep Prism open, a bad setting falls back by key. |
| V. Layered, override-only config | PASS | New settings are ordinary module settings: defaults not written, bad value falls back for its key, applied to the next search. |
| VI. Consistent UX | PASS | Rows use Feature V's row look; messages use its empty-state font. |
| VII. Total customizability | PASS | No new visible component. The three new visible values are role settings; all limits, scopes, the engine and the terminal are settings. The magic-values scan covers the new module. |
| VIII. Explicit keyboard interactivity | PASS | No new command, no key handling, no focus change. Enter and click act on rows inside the already open window. Documented in `docs/prism.md`. |
| Technical constraints | PASS | No new dependency; XDG paths; the cache is under `$XDG_CACHE_HOME/lumen`; no network by Prism; niri-specific code only as an optional spike fallback. The `XDG_CURRENT_DESKTOP` check for `OnlyShowIn` is environment based, not compositor code. |
| Quality gates | PASS | gofmt, vet, tests; qmllint, qmlformat; customization surface and keyboard note in the spec; static scans extended; tests listed in the quickstart. |

**Post-design re-check (after Phase 1)**: unchanged. One module, one Go program and a minor Prism version; no singleton, resident process or dependency.

### Complexity Tracking

No violations. Choices that add complexity on purpose:

| Choice | Why | Cheaper alternative and when it applies |
|--------|-----|-----------------------------------------|
| Parallel staged runs with ordered release | `api.run` answers once; the spec wants the first rows fast and a strict group order | One run per query returning everything. Used if spike S2 shows the process spawn cost outweighs the gain. |
| Three additions to Prism (keep selection, message line, marks) | The window cannot show highlights, hints or errors, and it jumps the selection on every update | Show hints as rows and drop highlights; acceptable only if the window change is refused. |
| Derived apps cache | 100 ms target for apps on every keystroke | Parse every time (a few tens of ms for 500 entries); used if the cache proves fragile. |
| Compiled helper built into a module folder | `api.run` only runs a module's own helpers | None without leaving the module interface. |

## Project Structure

### Documentation (this feature)

```text
specs/006-prism-modes/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── search-helper.md             # the eight operations, inputs, outputs, errors
│   ├── search-settings.md           # 13 module settings and 3 Prism settings
│   └── prism-source-slot-1.1.md     # setRows(keepSelection), setMessage, marks
├── checklists/requirements.md
└── tasks.md                         # created later by /speckit-tasks
```

### Source Code (repository root)

```text
modules/
├── prism/                           # CHANGED (1.0.0 -> 1.1.0)
│   ├── module.json                  # + matchRole, messageRole, errorRole; version 1.1.0
│   ├── Main.qml                     # setRows keepSelection, setMessage, message state
│   ├── geometry.js                  # contrast check extended to the three new roles
│   ├── PrismWindow.qml              # message line
│   ├── ResultRow.qml                # marked (highlighted) title ranges
│   └── rows.js                      # NEW: pure: row validation incl. marks, keep-selection rule
└── prism-search/                    # NEW
    ├── module.json                  # 13 settings, permissions contribute:prism + helpers, helper `search`
    ├── Main.qml                     # empty Scope (module entry)
    ├── Source.qml                   # query state, runs, cancel, ordered release, actions
    ├── stages.js                    # NEW: pure: stages per mode, ordered release, row building
    └── bin/search                   # built by make (gitignored)
scripts/
├── src/search/                      # NEW Go program (package main)
│   ├── main.go                      # protocol, op dispatch, budgets, panic guard
│   ├── scope.go                     # directory word detection, quoting
│   ├── match.go                     # fold, words, rank
│   ├── apps.go                      # desktop entries, visibility, cache, launch
│   ├── files.go                     # stages, walk, dedupe, limits
│   ├── words.go                     # quote-aware word split, ~ expansion
│   ├── terminal.go                  # terminal resolver shared by launch and open
│   ├── commands.go                  # command folders
│   ├── grep.go                      # split, smart case, binary, marks
│   └── open.go                      # EDITOR/TERMINAL/BROWSER, tables, detach, web URL
└── tests/
    ├── search_scope_test.go  search_match_test.go  search_apps_test.go
    ├── search_files_test.go  search_commands_test.go  search_grep_test.go
    ├── search_open_test.go            # fake programs record arguments
    ├── prism_search_settings_test.go
    ├── prism_slot_1_1_test.qml  search_stages_test.qml
    ├── no_magic_values_test.go  module_api_usage_test.go  # extended to prism-search
    └── fixtures/search/               # fixture builders
Makefile                              # MODULE_PROGRAMS, qmltest glob, gitignore for the built helper
.gitignore                            # modules/prism-search/bin/search
docs/
├── prism.md                          # modes, query forms, actions, environment, settings, slot 1.1
├── layout.md                         # where search code goes
└── dependencies.md                   # EDITOR, TERMINAL, BROWSER (optional)
README.md, modules/README.md          # updated
```

**Structure Decision**: Same rules as before. The window stays the window; search is a source module plus a standalone Go program (no shared Go code between programs; the matching and walking code lives only in `search`). The pure `.js` files hold the stage ordering and the row rules and are unit tested without a shell. Later features (usage ranking, running commands, file content search) extend the helper and the source without touching the window.
