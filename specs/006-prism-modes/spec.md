# Feature Specification: Prism Modes

**Feature Branch**: `006-prism-modes`

**Created**: 2026-11-10

**Status**: Draft

**Input**: User description: "Feature VI. Prism modes. Prism operates in 6 modes: default, apps, files, commands, grep and web. Every mode works by its rules: Apps: searches desktop entries only (system-wide + user's locals). Files: if the user specifies only a file name, search in the user's home (priority ~/{Desktop,Downloads,Documents} -> ~ (just home) -> other directories in ~ -> ~/.{cache,local,config}) + system-wide (shown collapsed because there may be hundreds of system files and caches); the user may also specify a searching directory, in which case search only in the given directory. Commands: search only in all default bin directories (system-wide + user's locals); in this feature only searching for them. Grep: awaits a regex string and a file (or files). Web: opens the user's default browser with the user's query. Default: searches for apps first, then files (without directory specified), and in the very end (after collapsed system files) suggests web search."

**Constitution principles touched**: IV (a failed launch or a missing `EDITOR`/`BROWSER` shows an error and degrades only that action), I (all searching, matching and ordering is done by the backend; the window only shows rows), III (the modes are modules/sources built on the public Prism interface from Feature V, not changes to the window), IV (an unreadable directory, a huge tree or a bad pattern degrades only that source), V and VII (the search scopes, ordering groups, limits and the search engine are named, override-only settings), VIII (no new keyboard commands; modes are chosen by the prefix typed in the already open window), and the technical constraint "no network access by default" (the only outward action is opening the user's own browser on the user's explicit Enter).

## Overview

Feature V built the Prism window and the prefix that selects a mode (`@a` apps, `@f` files, `@s` commands, `@g` grep, `@w` web, no prefix = default), but the window searches nothing. This feature gives each of the six modes its **searching rules**, so that Prism starts to be useful: type a few letters and see the application, the file, the command or the match you want.

The six modes:

1. **Apps**: finds installed applications by their desktop entries, system-wide and the user's own.
2. **Files**: finds files by name, in a fixed priority order inside the user's home plus a collapsed group of system files; or only inside a directory the user names.
3. **Commands**: finds executable commands in the standard command directories, system-wide and the user's own.
4. **Grep**: takes a regular expression and one or more files and lists the matching lines.
5. **Web**: sends the typed query to the user's default browser as a web search.
6. **Default** (no prefix): shows applications first, then files, then a collapsed group of system files, and last a suggestion to search the web.

Choosing a result (Enter or double click) does something useful in four of the modes: an application is **launched**, a file is **opened in the user's editor** (the `EDITOR` environment variable), a grep match is opened **in the same editor at its line** when the editor allows it, and the web row opens the user's **browser** (the `BROWSER` environment variable). If the needed variable is not set, Prism shows an error that says which one is missing and how to set it, and does nothing else.

What this feature does **not** do: it does **not run commands** from the commands mode (their effects can be unpredictable; running them is a later feature), and it does not rank by usage history.

## Clarifications

### Session 2026-11-10

- Q: When the user chooses a result, should this feature launch or open it? → A: Yes for four kinds. Apps are launched; files are opened with the editor named by the `EDITOR` environment variable; a grep match is opened in that editor on the matching line when possible; the web row opens the browser named by the `BROWSER` environment variable. If `EDITOR` or `BROWSER` is not set, Prism shows an error naming the variable (it may be set in a shell startup file such as `.bashrc`, which Lumen may not read) and does nothing else. Commands are NOT run in this feature (their results may be unpredictable); running them is a later feature.
- Q: If `EDITOR` is a terminal editor such as `vim`, where is the file opened? → A: In a terminal emulator. The terminal is a named setting whose default is the `TERMINAL` environment variable. If neither the setting nor `TERMINAL` is set, Prism shows an error naming them and does nothing else.
- Q: With several words in the query, must every word match, or the whole text as one phrase? → A: Every word must appear in the name, in any order, ignoring case and accents.
- Q: How does the user name a directory in files mode? → A: By writing a directory path in the query, for example `@f report ~/Documents/work`. A word that is a path (starts with `/`, `~/` or `./`, or is `~`) and names an existing directory is the search directory; the other words are the file name. With a directory, only that directory (and below) is searched, and there is no system group.
- Q: How does the user give grep its pattern and files? → A: In the query as `<regex> <file> [<file> ...]`: the first word is the regular expression (in quotes if it holds spaces), the rest are files. Missing the pattern or the files shows a hint, not an error.
- Q: Which search engine does web use? → A: The search engine is a named setting with a good default; the query is sent to the user's default browser as a search page for that engine.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find and see applications with `@a` (Priority: P1)

A user opens Prism, types `@a fire` and sees the applications whose names match, taken from the system's installed applications and from the user's own, each with its icon and name, best match first.

**Why this priority**: Applications are the most common thing people search for, and the default mode depends on it.

**Independent Test**: With a known set of desktop entries (system and user), type `@a` plus part of a name and check that exactly the matching, visible entries are listed, once each, with the user's entry winning when the same application exists in both places.

**Acceptance Scenarios**:

1. **Given** installed applications, **When** the user types `@a fire`, **Then** the results list the applications whose name (or generic name or keywords) matches `fire`, best match first, each with icon, name and a short description.
2. **Given** a desktop entry marked as hidden or not to be shown in menus, **When** the user searches, **Then** it is not listed.
3. **Given** the same application with an entry in the system and one in the user's local applications, **When** the user searches, **Then** it is listed once, using the user's entry.
4. **Given** `@a` with no text, **When** the mode is active, **Then** the applications are listed alphabetically.
5. **Given** a file that is not a desktop entry in an applications directory, **When** the user searches, **Then** it is never listed.
6. **Given** an unreadable or malformed desktop entry, **When** the user searches, **Then** that entry is skipped, the others are listed, and one warning is logged.
7. **Given** an application is installed or removed while Prism is running, **When** the user searches again, **Then** the list reflects the change within 5 seconds.
8. **Given** an app result, **When** the user presses Enter or double-clicks it, **Then** the application is launched as its desktop entry says and Prism closes; **Given** the launch fails, **Then** Prism stays open and shows an error naming the application.

---

### User Story 2 - Find files by name with `@f` (Priority: P1)

A user types `@f report` and sees files named like `report`, first those from `Desktop`, `Downloads` and `Documents`, then the rest of the home, and below them a collapsed line "System files (N)" that they can expand. If they type `@f report ~/Documents/work`, only that directory is searched.

**Why this priority**: Files are the second main source, and the ordering is the heart of the request.

**Independent Test**: Create files named alike in each priority location, in a regular home directory, in a hidden config directory and in a system directory; type `@f <name>` and check the order of the groups, the collapsed system group and its count; then repeat with a directory and check that only it is searched.

**Acceptance Scenarios**:

1. **Given** matching files in several places, **When** the user types `@f name` (no directory), **Then** results appear in this order: `~/Desktop`, `~/Downloads`, `~/Documents`; then files directly in `~`; then other directories in `~`; then `~/.cache`, `~/.local`, `~/.config`; then the system-wide group.
2. **Given** matching system files, **When** results are shown, **Then** they are in one collapsed group that shows only its title and the number of matches; the user can expand and collapse it, and it is collapsed every time a new search starts.
3. **Given** `@f name <directory>` where the directory exists, **When** the search runs, **Then** only that directory and what is below it is searched, the results are not grouped by the priority order, and there is no system group.
4. **Given** a named directory that does not exist or cannot be read, **When** the user types it, **Then** it is treated as part of the file name and no error is shown; if it looks like a path to a missing directory, an informational hint line (not an error) says that the directory was not found.
5. **Given** a very large number of matches, **When** results are shown, **Then** each group shows at most a set number of rows (a setting) with a "N more" line, and the window stays responsive.
6. **Given** a search in progress, **When** the user types another character, **Then** the old search stops, only the new one's results are shown, and rows already shown from the first groups are not removed while still matching.
7. **Given** an empty file name, **When** `@f` is active, **Then** no files are listed and a hint asks the user to type a name.
8. **Given** a matching file in a directory the user cannot read, **When** the search runs, **Then** that directory is skipped without an error and the rest is listed.
9. **Given** symbolic links, **When** the search walks the directories, **Then** it never loops and never leaves the searched tree by following a link to a directory.
10. **Given** a file result and `EDITOR` is set, **When** the user presses Enter or double-clicks it, **Then** the file is opened with the editor named by `EDITOR`, and Prism closes.
11. **Given** `EDITOR` is not set, **When** the user chooses a file result, **Then** Prism stays open and shows an error that `EDITOR` is not set and can be set in the user's environment (for example in `.bashrc`).
12. **Given** a result that is a directory, **When** it is chosen, **Then** nothing is opened and a message says that only files are opened.

---

### User Story 3 - Find commands with `@s` (Priority: P2)

A user types `@s ffm` and sees executable commands whose names match, found in the standard command directories (system-wide and the user's own, such as `~/.local/bin`), with the full path shown as a subtitle.

**Why this priority**: Useful and well-bounded; it does not depend on the other modes. Running commands is a later feature.

**Independent Test**: Place an executable file, a non-executable file and a duplicate name in different command directories; check that only the executables are listed, each name once, the first one in command-search order winning, with its path.

**Acceptance Scenarios**:

1. **Given** executables in the default command directories, **When** the user types `@s name`, **Then** the matching executable names are listed, best match first, each with its full path.
2. **Given** a name found in several command directories, **When** results are shown, **Then** it appears once, as the shell would resolve it (the first in the command search order).
3. **Given** a file without execute permission, a directory or a broken link, **When** the user searches, **Then** it is not listed.
4. **Given** the user's local command directories (for example `~/.local/bin`, `~/bin`), **When** they exist, **Then** they are searched too; **When** they do not exist, **Then** they are skipped silently.
5. **Given** a command directory that is not among the default ones (for example one only added to the user's shell), **When** the user searches, **Then** it is not searched unless the user adds it in the settings.
6. **Given** this feature, **When** the user chooses a command, **Then** nothing is run.

---

### User Story 4 - Search inside files with `@g` (Priority: P2)

A user types `@g TODO|FIXME notes.md plan.md` and sees every line of those files that matches the regular expression, each as a row with the file name, the line number and the line text, with the matched part highlighted.

**Why this priority**: A distinct, self-contained mode, but used less often than apps and files.

**Independent Test**: Write two small files with known lines; type a pattern and the two files; check the listed lines, line numbers and highlights; then type an invalid pattern, a missing file and a binary file and check each case is handled with a message and no failure.

**Acceptance Scenarios**:

1. **Given** `@g <regex> <file> [<file>...]` with existing text files, **When** the query is complete, **Then** each matching line is listed with file, line number, and the line text with the match highlighted, in file order and then line order.
2. **Given** only a pattern and no file, or no pattern, **When** the mode is active, **Then** a hint says that a pattern and at least one file are needed, and nothing is searched.
3. **Given** an invalid regular expression, **When** the user is typing, **Then** a message says the pattern is not valid, with no crash, and the previous results are cleared.
4. **Given** a file that does not exist, cannot be read or is a directory, **When** it is named, **Then** a message names that file, and the other named files are still searched.
5. **Given** a binary file, **When** it is named, **Then** it is skipped with a message that it looks like binary data.
6. **Given** a very large file or a pattern with very many matches, **When** the search runs, **Then** the search stops at a set limit (a setting), says that results were cut, and the window stays responsive.
7. **Given** a pattern with spaces in quotes, **When** the user types it, **Then** the quoted text is one pattern.
8. **Given** a file path that starts with `~/`, **When** the user types it, **Then** it is taken from the user's home.
9. **Given** the matching is case, **When** the pattern has no uppercase letters, **Then** the match ignores case; **When** it has any uppercase letter, **Then** it is case-sensitive ("smart case"), unless the user's setting says otherwise.
10. **Given** a grep match and `EDITOR` is set, **When** the user chooses it, **Then** the file is opened in that editor on the matching line when the editor supports it (otherwise at the start of the file), and Prism closes; **Given** `EDITOR` is not set, **Then** the same error as in the files mode is shown.

---

### User Story 5 - Web search with `@w` (Priority: P2)

A user types `@w how to rotate a pdf` and one row says "Search the web for «how to rotate a pdf»". On Enter, the user's default browser opens a web search page for that text.

**Why this priority**: Small, independent, and the only mode with an action.

**Independent Test**: Type a query with spaces, accents and symbols in web mode, press Enter, and check that the default browser opens the search page with the exact text; then check that nothing opens before Enter.

**Acceptance Scenarios**:

1. **Given** `@w` and a non-empty query, **When** the results show, **Then** exactly one row offers the web search for that query.
2. **Given** that row and `BROWSER` is set, **When** the user presses Enter or double-clicks it, **Then** the browser named by `BROWSER` opens the search page of the configured engine with the query properly encoded, and Prism closes.
3. **Given** a query with spaces, accents, quotes and symbols, **When** the page opens, **Then** the engine receives the text exactly as typed.
4. **Given** nothing has been chosen, **When** the user only types, **Then** no network access happens and no browser opens.
5. **Given** `BROWSER` is not set, **When** the user confirms, **Then** Prism stays open and shows an error that `BROWSER` is not set and that it can be set in the user's environment (for example in `.bashrc`); **Given** it is set but the browser cannot be started, **Then** Prism stays open and shows a message that the browser could not be opened.
6. **Given** an empty query, **When** `@w` is active, **Then** no row is offered and a hint asks the user to type a query.
7. **Given** the user sets another search engine, **When** the user confirms a query, **Then** that engine is used.

---

### User Story 6 - The default mode combines them (Priority: P1)

A user types `firefox` with no prefix. Prism shows the applications that match first, then matching files from their home, then a collapsed "System files (N)" line, and as the last row "Search the web for «firefox»".

**Why this priority**: The default mode is what most people use most of the time.

**Independent Test**: With matches in all sources, type a word with no prefix and check the order of the four parts and that each is absent when empty, and that the web suggestion is always the last row.

**Acceptance Scenarios**:

1. **Given** a query with no prefix, **When** results are shown, **Then** the order is: applications, home files (in the same group order as the files mode), the collapsed system files group, and the web search suggestion.
2. **Given** no application matches, **When** results are shown, **Then** the applications part is absent and the rest keep their order.
3. **Given** the default mode, **When** files are searched, **Then** no directory is taken from the text (a path in the query is just text), and commands and grep matches are not mixed in.
4. **Given** a non-empty query, **When** results are shown, **Then** the web suggestion is always present and always last, after the collapsed system group, even when everything else is empty.
5. **Given** the application results come first, **When** the files are still being searched, **Then** the applications are shown at once and the files appear as they are found, without moving the row under the selection.
5a. **Given** the user has not moved the selection since typing, **When** rows arrive in stages (even if the first row shown is only the web suggestion), **Then** the selection is on the first row of the complete list once all results are shown, never on the last one. *(Refinement 2026-11-10.)*
5b. **Given** the user has moved the selection to a row, **When** later rows arrive, **Then** that row stays selected.
6. **Given** an empty query in the default mode, **When** the window is open, **Then** no results are shown (the empty-state text applies).
7. **Given** a prefix that is only partly typed (such as `@` or `@f` without a space and text), **When** the user types, **Then** the mode follows Feature V's rules for recognizing the prefix, and results of the chosen mode are shown as soon as the prefix is recognized. A text that is only `@` (the start of a prefix) shows no results at all, as an empty query does: it is not searched as a file name and no web suggestion appears. *(Refinement 2026-11-10.)*

---

### User Story 7 - Control what is searched (Priority: P3)

A user changes the limits and scopes: how many rows each group shows, which extra command directories are searched, which search engine web uses, whether file search skips hidden files, and whether the system files group is shown at all.

**Why this priority**: Required by the constitution (Principles V and VII), but the defaults must already be good.

**Independent Test**: Change each setting in the file while Prism is open and check that the next search follows it without a restart.

**Acceptance Scenarios**:

1. **Given** a changed search setting, **When** the next search is run, **Then** it follows the setting within 1 second of the change and without a restart.
2. **Given** a bad value (wrong type, out of range, a missing directory), **When** the configuration is read, **Then** that key alone falls back to its default with an error naming the module (`modules.prism-search`) and the key.
3. **Given** the default configuration, **When** the user changes nothing, **Then** every mode works as described in the other stories.

---

### User Story 8 - Complete directory and file names with Tab (Priority: P2) *(Refinement 2026-11-10)*

A user typing a directory in the files mode (`@f report ~/Doc`) or a file in the grep mode (`@g TODO notes/re`) presses Tab and the word being typed is completed from what exists on disk, as in a shell.

**Why this priority**: Typing long paths by hand is slow and error-prone, and both forms already depend on exact paths; it does not change what is searched.

**Independent Test**: In a fixture home, type each form with a partial path, press Tab, and check the text and the message.

**Acceptance Scenarios**:

1. **Given** `@f report ~/Doc` and exactly one directory in the home starting with `Doc`, **When** the user presses Tab, **Then** the text becomes `@f report ~/Documents/` (a directory gets a trailing `/`), and the search for that directory runs.
2. **Given** several directories starting with the typed letters, **When** the user presses Tab, **Then** the word is extended to their longest common start, and a line of message shows how many candidates there are and the first names; a second Tab with nothing to extend changes nothing and keeps that message.
3. **Given** no entry starts with the typed letters, **When** the user presses Tab, **Then** the text does not change and a message says there is no match.
4. **Given** `@g TODO notes/re`, **When** the user presses Tab, **Then** the last word is completed with files and directories (`notes/report-note.md`; a directory gets `/`); the pattern word is never completed.
5. **Given** a name with spaces or single quotes, **When** it is completed, **Then** it is written in double quotes so that the query still reads as one word, and the cursor is left before the closing quote for a directory so the user can keep typing. A name that contains a double quote cannot be written in the query's quoting, so it is not offered (the message says how many were left out).
6. **Given** any other mode, a files-mode word that is not a path (not `~`, `/…`, `~/…` or `./…`), or text that ends with a space, **When** the user presses Tab, **Then** nothing changes and nothing else happens.
7. **Given** a directory that does not exist or cannot be read, **When** the user presses Tab, **Then** the text does not change, a message says why, and nothing fails.

---

### Edge Cases

- Query with leading or trailing spaces: trimmed, and the prefix rule of Feature V still applies.
- Names with accents, other scripts, or mixed case: matched without regard to case and accents for apps, files and commands.
- File names with spaces, quotes or newlines: listed and shown correctly; a path with spaces can be given in quotes.
- A huge home directory (hundreds of thousands of files): the first rows appear quickly, and the search keeps the window responsive; results beyond the limit are summarized.
- Network or removable file systems that are slow or hung: they are skipped after a time limit and the rest is shown; they never freeze the window.
- Two matches with the same name in different priority groups: both are shown, each in its group, never merged.
- The same file reached by two paths (links): listed once.
- Searching while the user is still typing the directory path (such as `@f report ~/Doc`): the partial path does not exist, so it is text for the file name until the directory exists; no flicker of results between steps.
- Modes with another prefix letter set by the user (Feature V): the modes follow the mode names, not the letters.
- The highlight behind a recognized prefix (`@s`) is always drawn whole, with its rounded outline and a small margin around the letters on every side; it is never cut off by the edge of the text area. A long text that scrolls may still hide it when the prefix has scrolled out of view. *(Refinement 2026-11-10.)*
- Text that is just `@`: the user is about to type a mode letter, so nothing is searched and nothing is shown (before this refinement it listed every file and folder whose name contains `@`, such as scoped package folders). Any longer text without a recognized prefix, such as `@swc`, is still an ordinary query, so such names remain findable.
- Only the web suggestion is shown yet (the other stages are still running) and then more rows arrive above it: the selection goes to the new first row, unless the user already moved it.
- A system with no desktop entries, no command directories, or no home: the mode shows its empty state; nothing fails.
- Web query longer than a browser address can hold: a message says it is too long; the browser is not opened.
- Grep on a file that changes while it is read: it reads what it can and does not fail.
- The same line matching in several places: one row per line, all matches highlighted.

## Requirements *(mandatory)*

### Functional Requirements

**General**

- **FR-001**: Each mode MUST be provided through the public Prism source interface of Feature V, so that the window itself is not changed by this feature, and a mode that fails MUST lose only its own rows, per Feature V.
- **FR-002**: Searching, matching, ordering and limiting MUST be done by the backend, never by the window, and MUST NOT block the window; results of a search MUST reach the window as they are found, and a newer query MUST cancel the older one.
- **FR-003**: Each mode MUST use the mode and query that Feature V gives (`default`, `apps`, `files`, `commands`, `grep`, `web`); the modes MUST follow the mode names and not the letters of the prefixes.
- **FR-004**: An empty or whitespace query MUST NOT run a search in any mode. In the modes `apps`, `files`, `commands`, `grep` and `web` it MUST show a short hint of what the mode expects; in the default mode it shows no results and no hint, so that the empty-state text of Feature V applies.
- **FR-005**: Choosing a result (Enter or double click) MUST act as follows, and the choice MUST also be reported to the module interface as in Feature V: an application is launched (FR-010a); a file is opened in the editor named by `EDITOR` (FR-018a); a grep match is opened in that editor on its line (FR-026); the web row opens the browser named by `BROWSER` (FR-028); a command result MUST NOT be run, and choosing it does nothing in this feature. In the default mode each row acts by its own kind.
- **FR-005a**: When `EDITOR` or `BROWSER` is needed and not set (or empty), Prism MUST stay open, MUST show an error that names the variable and says it can be set in the user's environment (for example in a shell startup file such as `.bashrc`, and that the shell that starts Lumen must have it), and MUST do nothing else. Prism MUST NOT guess another editor or browser.
- **FR-005b**: The editor MUST be started inside a terminal emulator: the one named by the terminal setting, or, when the setting is not set, by the `TERMINAL` environment variable. When neither is set (or empty), FR-005a applies to them: Prism MUST stay open, name both the setting and `TERMINAL` in the error, and do nothing else. The terminal and the editor MUST outlive Prism.
- **FR-006**: Matching for apps, files and commands MUST split the text into words (on spaces) and MUST require every word to appear in the name, in any order (for example `report final` finds `final_report.pdf`); it MUST ignore case and accents, and MUST rank better matches first (a name that starts with the text before one that contains it, a shorter name before a longer one, then alphabetical); ranking MUST NOT use history.
- **FR-007**: A directory, file or entry that cannot be read MUST be skipped without stopping the search and without showing an error row; one warning MAY be logged per source.

**Apps**

- **FR-008**: The apps mode MUST search desktop entries only, in the system-wide application directories and in the user's local application directory, using the standard locations for the user's environment.
- **FR-009**: It MUST match on the application name, generic name and keywords, MUST NOT list entries marked as hidden or as not shown in menus, and MUST list an application once when it has entries in several places, preferring the user's local entry.
- **FR-010**: Each app row MUST show its icon, name and a short description; the row's identifier MUST identify the desktop entry.
- **FR-010a**: Choosing an app row MUST launch the application as its desktop entry specifies and close Prism; the launched application MUST NOT depend on Prism staying alive. A launch that fails MUST keep Prism open and show an error naming the application.
- **FR-011**: The list of applications MUST reflect installed or removed applications within 5 seconds, without restarting Prism.

**Files**

- **FR-012**: When the query has no directory, the files mode MUST search the user's home and the system-wide locations, matching on the file name, and show results in these groups, in this order: (1) `~/Desktop`, `~/Downloads`, `~/Documents` (in this order); (2) files directly in `~`; (3) other directories in `~`; (4) `~/.cache`, `~/.local`, `~/.config`; (5) system-wide files.
- **FR-013**: The system-wide group MUST be shown collapsed, as one row with its title and the number of matches, and MUST be expandable and collapsible by the user; every new search MUST start collapsed.
- **FR-014**: When the query holds a directory (a word that is `~` or starts with `/`, `~/` or `./`, and names an existing readable directory), the files mode MUST search only that directory and what is below it, MUST show no groups and no system group, and take the other words as the file name. A relative path with `./` is taken from the user's home.
- **FR-015**: Other hidden directories of the home (those starting with a dot, apart from `.cache`, `.local` and `.config`) MUST NOT be searched, unless a setting turns it on; hidden files inside a searched directory MUST follow the same setting (default: not shown), except when the named directory itself is hidden.
- **FR-016**: Each group MUST show at most a set number of rows (a setting, with a default) followed by a "N more" row, and a search MUST stop after a set number of files or a set time (settings), saying that the result was cut.
- **FR-017**: The search MUST NOT follow symbolic links to directories, MUST show a file reached by several paths once, and MUST skip locations that do not answer within a time limit.
- **FR-018**: Each file row MUST show an icon for its kind, the file name, and its folder (shortened) as the subtitle.
- **FR-018a**: Choosing a file row MUST open that file in the editor named by `EDITOR` and close Prism (FR-005a applies when it is not set). Choosing a directory MUST open nothing and say that only files are opened. A failure to start the editor MUST keep Prism open and show an error.

**Commands**

- **FR-019**: The commands mode MUST search only the default command directories: the system-wide ones and the user's local ones (`~/.local/bin` and `~/bin`), plus the directories the user adds in the settings; a directory that does not exist MUST be skipped silently.
- **FR-020**: It MUST list only regular files (or links to them) that the user is allowed to execute, once per name, resolved as the shell would (the first directory in the search order wins), and each row MUST show the name and its full path.
- **FR-021**: The commands mode MUST only search; it MUST NOT run, preview or open any command, and choosing a command result MUST do nothing except report the choice to the module interface.

**Grep**

- **FR-022**: The grep mode MUST take the query as a regular expression followed by one or more files, split as words (a word in single or double quotes is one word; a relative file path is taken from the user's home, since Prism has no working directory), and MUST list each matching line as a row with the file, the line number and the line text with the matched part marked.
- **FR-023**: The grep mode MUST use smart case by default (case-insensitive when the pattern has no uppercase letter, otherwise case-sensitive), changeable by a setting.
- **FR-024**: Without a pattern or without any file, it MUST show a hint and search nothing; with an invalid pattern, it MUST show a message and no results; with a file that is missing, unreadable or a directory, it MUST name that file in a message and still search the others; a file that looks like binary data MUST be skipped with a message.
- **FR-025**: The grep mode MUST stop at a set number of matches (a setting), say that the result was cut, never hold a whole large file in memory, and keep the window responsive. A file path MUST accept `~/`.
- **FR-026**: Choosing a grep match MUST open its file in the editor named by `EDITOR`, on the matching line when the editor supports it (otherwise at the start of the file), and close Prism (FR-005a applies when it is not set). The grep mode itself MUST NOT edit any file.

**Web**

- **FR-027**: The web mode MUST offer exactly one row, "Search the web for «query»", for a non-empty query, and MUST NOT access the network or open anything until the user confirms with Enter or a double click.
- **FR-028**: On confirmation, the web mode MUST open the browser named by `BROWSER` on the search page of the configured engine with the query correctly encoded, and then close Prism (FR-005a applies when `BROWSER` is not set). If the browser cannot be started, Prism MUST stay open and show a message.
- **FR-029**: The search engine MUST be a named setting with a good default; the query MUST be sent only to the user's browser and MUST NOT be stored, logged with the text, or sent anywhere else by Prism.

**Default mode**

- **FR-030**: In the default mode, with a non-empty query, results MUST appear in this order: applications; home files in the groups of FR-012 (without taking a directory from the text); the collapsed system-wide files group; and, last of all, the web search row, which MUST always be present for a non-empty query.
- **FR-031**: The default mode MUST NOT include commands or grep results, and MUST show applications as soon as they are known, without waiting for the file search; rows already shown MUST NOT move under the selection while later groups arrive.
- **FR-031a**: While the user has not moved the selection since the query last changed, the selection MUST be on the first row of the rows shown, whatever arrives later (so with all results shown it is on the first row, not the last). Once the user moves it, the chosen row MUST stay selected while later rows arrive. *(Refinement 2026-11-10; the web suggestion, always last, must never attract the selection.)*
- **FR-032**: The default mode MUST use the same matching, groups, limits and rules as the apps and files modes (no separate rules).

**Settings, failure, documentation**

- **FR-033**: The following MUST be named, override-only settings with defaults, validation and per-key fallback, applied to the next search: the rows shown per group; the file count and time limits; whether hidden files and hidden home directories are searched; extra command directories; the grep match limit and case rule; the search engine; the terminal used to open the editor; whether the system group is shown; and the time limit for slow locations.
- **FR-034**: A bad value MUST fall back for that key only, with an error naming the module (`modules.prism-search`, or `modules.prism` for the window's own settings) and the key; a missing directory in a setting MUST be skipped with one warning; no search failure MAY crash or freeze the shell or the window (Principle IV).
- **FR-035**: Every new setting MUST appear in Prism's schema and in the merged schema, and in `lumen modules show prism` with its errors and warnings.
- **FR-036**: The documentation MUST describe each mode's rules and the file-ordering groups, the query forms of the files and grep modes with examples, the settings, what choosing each kind of result does, the `EDITOR` and `BROWSER` requirement with how to set them so that Lumen sees them, and the fact that commands are not run; and MUST note that no new keyboard command is added (Principle VIII).
- **FR-037**: Automated checks MUST cover, with fixed test data: the ordering and the collapsed system group; the directory form; command resolution; desktop-entry visibility and precedence; grep splitting, smart case and the invalid or binary cases; web encoding; and the default mode's final order.

**Completion with Tab** *(Refinement 2026-11-10)*

- **FR-038**: Pressing Tab in the window MUST complete the last word of the text, and only in two places: the directory word of the files mode (a word that is `~`, or starts with `/`, `~/` or `./`, as in FR-014) and the file words of the grep mode (every word after the pattern). In every other mode and position Tab MUST change nothing and do nothing else.
- **FR-039**: The candidates MUST be the entries of the one directory named by the typed word (up to its last `/`) whose names start with the typed rest; matching is exact in case first and, if nothing matches, ignoring case. The files mode offers directories only; the grep mode offers files and directories. A relative word is taken from the home, as in FR-014. Only that one directory is read (no walk), within a size and time limit.
- **FR-040**: One candidate MUST be completed in full, with a trailing `/` for a directory. Several MUST extend the word to their longest common start and show one message line with their count and the first names (at most 8). No candidate MUST leave the text unchanged and show a message. Hidden entries are candidates only when the typed rest starts with `.`.
- **FR-041**: A completed name with spaces or single quotes MUST be written in double quotes (a name with a double quote is not offered) so that the quoting rule of the files and grep forms (a path with spaces can be given in quotes) reads it back as one word, with the cursor left inside the closing quote after a directory. After a completion the normal search for the new text MUST run.
- **FR-042**: Completion MUST read nothing but directory listings, MUST never fail or freeze the window for a missing, unreadable or huge directory (it reports a message instead), and MUST add no setting (its wording is fixed text, like the other hints).

### Customization surface *(required by principle VII's quality gate)*

This feature adds no visible component: it fills the Feature V window, which already takes every color, font, radius, margin, padding and placement from tokens and settings. The rows it produces use that same row look. The only new visible parts are plain text (hints, messages, the group title with its count, the "N more" row), which use the row and empty-state styles of Feature V. Their wording is fixed text of the module (it is not a visual value; a translation or rewording is a later feature). Three new color settings of the Prism window cover the new visible parts: `matchRole` (marked text of a grep match), `messageRole` (hint line) and `errorRole` (error line), all theme roles with the contrast rule of Principle II; the message line uses the empty-state font. New non-visual settings are listed in FR-033.

### Keyboard and popups *(required by principle VIII's quality gate)*

- **Commands**: none added (Tab is a key of the open window, not a command). Modes are chosen by the prefix typed in the window opened with Feature V's command.
- **Keys while open**: Feature V's keys, plus **Tab** (new, FR-038: completes the last directory or file word in the files and grep modes, does nothing elsewhere; it is only handled while the window has the keyboard, and is listed in the Prism documentation with the other keys), plus Enter on a result (launches an app, opens a file or grep match in the editor, opens the browser for the web row) and a key or click to expand or collapse the system files group (the standard activation key on a focused group row).
- **Implicit keyboard**: unchanged from Feature V (only after the user's opening command).
- **Popups**: none added.

### Key Entities

- **Mode**: one of `default`, `apps`, `files`, `commands`, `grep`, `web`; has the rules for its search and the rows it produces.
- **Source**: the part that serves one mode's rows to Prism; receives the query and returns rows as they are found.
- **Application entry**: a desktop entry with a name, generic name, keywords, icon, description and visibility; comes from the system or from the user.
- **File match**: a file found by name, with its folder, kind and the group it belongs to.
- **Group**: an ordered set of file matches by location (priority folders, home, other home folders, hidden config folders, system); the system group is collapsible.
- **Command**: an executable name with the full path where it was found.
- **Grep match**: a file, a line number, the line text and the marked part.
- **Web search**: the engine and the query it will open.
- **Search settings**: limits, scopes, extra directories, case rule and engine, layered over defaults.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In apps mode the first matching application appears within 100 ms of the last typed character on a system with 500 desktop entries.
- **SC-002**: In files mode the first rows from the priority folders appear within 200 ms on a home folder with 200,000 files, and the window stays smooth (no dropped frames visible) while the rest is searched.
- **SC-003**: In 100% of checked searches, file groups appear in the exact order of FR-012 and the system files are collapsed until the user expands them.
- **SC-004**: In 100% of checked cases, a query with a named existing directory shows results only from inside that directory and no system group.
- **SC-005**: In commands mode, 100% of listed names are executable and each name appears once, with the path the shell would use.
- **SC-006**: In grep mode, the listed lines and line numbers match a reference search on the same files in 100% of checked cases, including smart case, quoted patterns and several files.
- **SC-007**: In web mode, the browser receives the typed query exactly (including accents, spaces and symbols) in 100% of checked queries, and no browser opens or network access happens before the user confirms.
- **SC-008**: In the default mode, the order apps → home files → collapsed system files → web suggestion holds in 100% of checked queries, and the web suggestion is present for every non-empty query.
- **SC-009**: A user can find a known app, file, command or line of text with one typed prefix and a few letters, in under 10 seconds, on the first try, in at least 90% of checked attempts.
- **SC-010**: With 10 deliberately wrong search settings, Prism still searches in every mode, each wrong key falls back alone, and 100% of them are named in the module's report.
- **SC-012**: In 100% of checked cases Tab completes the typed directory or file word exactly as in the acceptance scenarios of Story 8 (single, several, none, spaces, unreadable), and the completion appears within 100 ms on a directory of 5,000 entries.
- **SC-011**: An unreadable directory, a hung mount, an invalid pattern or a binary file never freezes or closes the window or the shell in any checked run.

## Assumptions

- The window, the prefix recognition, the row look, the source interface and the empty states come from Feature V (`specs/005-prism-window`) and are not changed here, except for the small additive changes of the refinements (Prism 1.1.0 and 1.2.0: Tab completion, `setText`); the six mode names are fixed there.
- Completion works on the last word of the text and assumes the cursor is at the end (the usual case); it does not complete names inside the files mode's name words, only directory words (Refinement 2026-11-10).
- Running a command is a later feature (its results may be unpredictable). Launching apps, opening files and grep matches in `EDITOR`, and opening the browser from `BROWSER`, are part of this feature.
- The editor is always started in a terminal (FR-005b), so GUI editors in `EDITOR` are also started that way; they ignore the terminal. A setting that skips the terminal is a possible later addition.
- `EDITOR`, `BROWSER` and `TERMINAL` are read from the environment Lumen runs in. A variable set only in a shell startup file may not reach Lumen; this is why the error explains how to set it. Prism does not read shell startup files itself.
- "System-wide" desktop entries means the standard shared application directories for the user's environment (including the ones set by the environment's data-directory variable), plus the user's local applications directory.
- "System-wide" files means the standard system locations outside the user's home (for example the system program, library, configuration, variable data and optional software trees). Virtual and device file systems are not searched. The exact list is in the plan and may be extended by a setting.
- "Default bin directories" means the standard system command directories and the user's `~/.local/bin` and `~/bin`; the user can add more in a setting. The exact list is in the plan.
- "Other directories in `~`" means the non-hidden directories directly in the home other than `Desktop`, `Downloads` and `Documents`, searched at any depth.
- Hidden directories of the home other than `.cache`, `.local` and `.config` are not searched by default (a setting can include them).
- Matching on names is an all-words "contains" match (FR-006) with its ranking; smarter fuzzy matching and usage-based ranking are a later feature.
- The default search engine is a privacy-respecting one chosen in the plan; it is a setting, and the browser is the one named by `BROWSER`.
- Opening a link in the browser is the user's explicit action (Enter), so it is consistent with the rule that network features are opt-in; Prism itself does no network access.
- Searching is read-only; no file is created, changed or indexed to disk by this feature (an on-disk index is a possible later optimization, not required here). Opening a result in the editor is the editor's job.
- Grep reads plain text files; compressed, archive and binary formats are not searched.
- Searching several screens, remote machines or file contents in the files mode (searching inside files from the files mode) is out of scope.

## Refinements

- 2026-11-10 (fix): a lone `@` was searched as a file name in the default mode and listed many unrelated rows. It now behaves like an empty query (scenario 7 of the default mode and an edge case).
- 2026-11-10 (fix): with all results shown the selection was on the last row. The first row shown can be only the web suggestion, and keeping "the selected row" across later updates then followed it to the bottom. Added FR-031a, scenarios 5a and 5b and an edge case: the selection stays on the first row until the user moves it.
- 2026-11-10 (addition): Tab completes the directory word of the files mode and the file words of the grep mode (user chose: directory argument, Tab key, files and grep). Added User Story 8, FR-038 to FR-042, SC-012, the Tab key in the keyboard section and two assumptions.
- 2026-11-10 (fix): the highlight behind a recognized prefix had its left edge and rounded outline cropped by the text area. It is now drawn whole (edge case added; the fix is in Prism's window, the prefix highlight of Feature V).
