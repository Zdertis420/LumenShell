# Quickstart: validating Prism Modes

How to prove the feature works. Contracts: [search-helper.md](contracts/search-helper.md), [search-settings.md](contracts/search-settings.md), [prism-source-slot-1.1.md](contracts/prism-source-slot-1.1.md). Data: [data-model.md](data-model.md).

## Prerequisites

- niri session, Quickshell, Go 1.27, `qmltestrunner` (Qt 6).
- `make build` (also builds the helper into `modules/prism-search/bin/search`).
- A key bound in niri to `lumen modules call prism toggle` (Feature V).
- For the actions: `EDITOR`, `TERMINAL` and `BROWSER` in niri's `environment` block (`environment { EDITOR "nvim"; TERMINAL "foot"; BROWSER "firefox" }`), then restart the session.

## 1. Automated checks

```text
make lint
make test
```

Expected: gofmt, vet and qmllint clean; the new tests pass:

| Test | Proves |
|------|--------|
| `scripts/tests/search_scope_test.go` | directory form, missing directory hint, quoted paths (US2) |
| `search_apps_test.go` | visibility, precedence, localized name, cache invalidation (US1) |
| `search_files_test.go` | group order and contents, symlinks, hardlinks, limits, hidden rules, unreadable folders (US2) |
| `search_commands_test.go` | executables only, dedupe by name, merged-usr (US3) |
| `search_grep_test.go` | split with quotes, smart case, invalid pattern, binary, missing file, cut at the limit (US4) |
| `search_match_test.go` | all-words match, accents, ranking (FR-006) |
| `search_open_test.go` | editor and terminal commands per editor and terminal, line arguments, missing `EDITOR`/`TERMINAL`/`BROWSER` errors, URL encoding, `BROWSER` forms (US2, US4, US5) with a fake program that records its arguments |
| `prism_search_settings_test.go` | the 13 settings, defaults, ranges, bad values fall back (US7) |
| `prism_slot_1_1_test.qml` | marks validation, keep-selection (R4, R5) |
| `search_stages_test.qml` | ordered release of stages: web always last, late answers ignored (US6) |
| existing scans | no hardcoded colors, no magic values, API-only use, no network in `modules/prism-search/` |

## 2. Fixture home for manual checks

```text
export FIX=$(mktemp -d)   # then create: $FIX/Desktop/report-a.txt, Downloads/report-b.txt,
# Documents/report-c.txt, report-d.txt (in $FIX), notes/report-e.txt, .config/app/report-f.txt
```

Run the shell with `HOME=$FIX` (see `docs/development.md`), plus a user applications directory with two desktop entries (one hidden, one with the same id as a system one).

## 3. Manual scenarios on a real session

1. **Apps** (US1): `@a fire` lists the matching apps once, user entry first; Enter launches and closes; a failing launch keeps Prism open with an error.
2. **Files order** (US2): `@f report` shows a to f in the order of FR-012 and a collapsed "System files (N)"; Enter on it expands; a new query collapses it.
3. **Directory form**: `@f report ~/Documents` shows only that directory and no system group.
4. **Open**: Enter on a file opens `EDITOR` in the terminal; unset `EDITOR` and repeat: the error names `EDITOR` and the niri fix; Prism stays open.
5. **Commands** (US3): `@s ffm` lists executables with paths; Enter does nothing.
6. **Grep** (US4): `@g "TODO|FIXME" a.md b.md` lists lines with highlights; Enter opens the editor on that line; an invalid pattern, a missing file and a binary file each give a message.
7. **Web** (US5): `@w a b&c é`, Enter: `BROWSER` opens the engine page with the exact text; nothing opens before Enter; unset `BROWSER`: error.
8. **Default** (US6): `report` shows apps, home files, collapsed system files, then "Search the web for «report»" as the last row.
8a. **Lone `@`**: type only `@`: no rows and no message; then `@s`: the commands mode; `@swc`-like text with no prefix match is an ordinary query.
8b. **Tab completion** (US8): in the fixture home type `@f report ~/Doc` and Tab: `~/Documents/`; `@g needle notes/re` and Tab: `notes/report-note.md`; a prefix with several candidates extends to the common start and shows "N matches: …"; no candidate: unchanged with a message; a folder with a space is quoted and the cursor stays before the closing quote; Tab in `@a`, `@s`, `@w`, the default mode, or after a trailing space does nothing.
8c. **Prefix highlight**: type `@s`: the highlight behind `@s` shows its whole rounded outline with a small margin on all four sides (by eye; try `radius`, `inputPadding` and `rowRadius` changes).
9. **Selection while results arrive**: type a query and do not touch the keys: when all results are shown the first row is selected (not the last, not the web row). Then repeat, pressing Down once while the system group is still searching: that row stays selected as more rows arrive.
10. **Settings** (US7): change `rowsPerGroup`, `searchUrl`, `systemGroup`; the next search follows within a second; put a bad `searchUrl`: the key falls back and `lumen modules show prism-search` names it.
11. **Failure**: `chmod 000` a folder in the fixture, rename the helper: Prism keeps working, with a message.

## 4. Performance (by hand, method fixed)

- Apps: first match within 100 ms with 500 entries (SC-001).
- Files: first rows within 200 ms on a home with 200,000 files, window smooth while the rest is searched (SC-002). Time with `lumen modules show` timings and a screen recording at 60 fps.
- A hung mount (a stalled network share) does not freeze the window (SC-011).
- Findability (SC-009): ask three people (or yourself on three different days) to find a known app, file, command and line of text with one prefix and a few letters; at least 90% of the attempts must succeed in under 10 seconds on the first try.

## 5. Done when

All automated tests pass, scenarios 1 to 11 behave as written, the performance checks hold, and the docs (`docs/prism.md`, `docs/layout.md`, `README.md`) describe the modes, the settings, `EDITOR`/`TERMINAL`/`BROWSER` and the fact that commands are not run.
