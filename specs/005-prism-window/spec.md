# Feature Specification: Prism Window (Skeleton)

**Feature Branch**: `005-prism-window`

**Created**: 2026-11-09

**Status**: Draft

**Input**: User description: "Feature V. "Prism" multi-menu. This feature covers only the skeleton window: positioning, theming and other customizations". Clarified by the user: Prism is a Spotlight-like multi-menu that can search apps, files and commands (and run them), do a web search or grep a file; this feature covers only the window itself, not its functionality.

**Constitution principles touched**: VII (the window is fully customizable: nine placements, colors by role, fonts, shape and spacing), II (every color is a semantic role of the active theme), VI (one design language through the shared tokens), VIII (Prism is a keyboard-input module: it takes the keyboard only when the user's bound command opens it, and releases it when it closes), III (Prism is a module built on the public module interface), V (every setting is layered, override-only, with per-key fallback), IV (a bad value, a missing font or a screen that does not fit degrades only what it touches).

## Overview

**Prism** will be Lumen's Spotlight-like multi-menu: one box the user opens with a key, types into, and gets results from many sources (applications, files, commands that can be run, a web search, a search inside a file). All of that searching, and the running of results, comes in later features.

This feature builds **only the window**: the empty, fully styled shell that those features will fill. It decides how Prism is opened and closed, where it sits on the screen, how big it is, how it looks and how it moves, and what its parts are. The parts are:

1. **The input field**: where the user types. It shows a placeholder text when empty.
2. **The results area**: below (or above) the input, a scrollable list region with a row look (icon, title, subtitle, a right-side hint), a selected-row highlight, and an empty state. In this feature it holds no real results.
3. **The prefix**: Prism has no mode indicator and no mode switcher. The user chooses a mode by typing a short prefix that starts with `@` at the beginning of the input: `@a` apps, `@f` files, `@w` web search, `@s` commands, `@g` grep a file. With no prefix, Prism is in the `default` mode. The window recognizes the prefix, separates it from the rest of the text, and gives both to modules; what each mode does is a later feature. The recognized prefix is highlighted (a chip in its own color role, with an opacity) so the user sees that it was understood.
4. **The footer**: an optional strip with key hints.

Because there is no real content yet, Prism also offers a **preview mode** that fills the results area with a fixed set of sample rows, so the user and the tests can see the whole look (selected row, icons, long titles, empty state) while setting colors, fonts and shapes. The sample rows are not search results and do nothing when chosen.

What this feature delivers:

- **Open and close** by explicit user command, with the keyboard given to the input and returned on close.
- **Placement** at any of **nine anchor positions** (four corners, four edge centers and the center), with offsets, on the focused screen or a named one, and a size that is limited to what fits.
- **Theming** with semantic roles of the active theme, with opacity, live.
- **Customization** of fonts, radii, borders, paddings, gaps, row size, shadow, backdrop dimming and animation, all as named settings.
- **A stable shape for later features**: the mode (from the prefix), the query text and the selected row are visible to modules through the public module interface, so that the search features can be added as modules without changing the window.

## Clarifications

### Session 2026-11-09

- Q: What is Prism? → A: A Spotlight-like multi-menu (search through apps, files and commands, run commands, web search, grep a file). This feature covers only the window. Searching, results, running anything, history, ranking and sources are separate features.
- Q: Does the window take the keyboard? → A: Yes, by its purpose. Under Principle VIII Prism is a module that obviously needs the keyboard (a search box), so the window takes keyboard focus when it is opened. It is opened only by the explicit command that the user binds in the compositor (`lumen modules call prism toggle`); commands are the only way to open it. It never takes the keyboard at load, on hover or by itself, ships no default binding, registers no global key, declares in its manifest that it takes the keyboard, and gives the focus back to the previously focused window when it closes. Escape closes it. While Prism is open it holds the keyboard exclusively, so the user cannot type into or use other windows until it closes; this is how keyboard focus works and is expected, not a defect.
- Q: How does the user choose a mode? → A: By a prefix typed at the start of the input, not by any indicator or switcher: `@a` apps, `@f` files, `@w` web, `@s` commands, `@g` grep. No prefix means the `default` mode. Prism only recognizes the prefix and passes the mode and the rest of the text to modules; the behavior of each mode is a later feature. The prefix is followed by a space or the end of the text; an unknown `@x` is plain text in the default mode.
- Q: Can the user change the mode prefixes? → A: Yes. The letter of each prefix is a setting, with `a`, `f`, `w`, `s` and `g` as defaults; the `@` sign is fixed.
- Q: Does Escape clear non-empty text first? → A: No. Escape always closes the window at once, with or without text; the `clear` command empties the input.
- Q: How many positions? → A: Nine (the three-by-three grid), per Principle VII for the launcher kind of component. The default is a Spotlight-like place: horizontally centered, in the upper part of the screen.
- Q: One window or one per screen? → A: One window, on one screen at a time: the screen with the focused window or the pointer, or a screen the user names.
- Q: Real content? → A: None. No search, no results, no actions. The input accepts and edits text, and the text is available to modules; nothing else happens on Enter in this feature except that Prism reports the chosen row to modules through the interface (a later feature acts on it) and, for sample rows, does nothing.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Open Prism with a key and close it (Priority: P1)

A user binds `lumen modules call prism toggle` to a key in their compositor. They press it and a rounded search window appears in the upper middle of the screen, with a blinking cursor in the input and a placeholder. They type, and the text appears. They press Escape and the window disappears, and the window they were working in gets the keyboard back.

**Why this priority**: Everything else is about this window. If it does not open, take the keyboard and give it back reliably, nothing else matters.

**Independent Test**: With only the default configuration, run the toggle command, type text, press Escape, and check that the window appears, accepts text, closes, and the previous window is focused again.

**Acceptance Scenarios**:

1. **Given** the default configuration, **When** Lumen starts, **Then** no Prism window is shown and Prism holds no keyboard focus.
2. **Given** the window is hidden, **When** the user runs the toggle command, **Then** within 200 ms the window is shown on the focused screen and the input has the keyboard and is ready for typing.
3. **Given** the window is open, **When** the user presses Escape, runs toggle or close, or clicks outside the window (if that setting is on), **Then** the window closes within 200 ms and the previously focused window gets the keyboard back.
4. **Given** the window is open, **When** the user types, **Then** the input shows the text, supports editing (cursor movement, delete, select all, paste), and the placeholder shows only when the input is empty.
5. **Given** the window is open on one screen, **When** the user runs the open command from another screen, **Then** the window moves to that screen and never exists on two screens at once.
6. **Given** the window is open, **When** the user closes and opens it again, **Then** the input is empty by default (or keeps its text if the user's setting says so).
7. **Given** two quick toggle commands, **When** they both arrive, **Then** the window ends in the state of the last one and never shows twice.

---

### User Story 2 - Put the window where I want it (Priority: P1)

A user prefers Prism in the center of the screen, or at the bottom like a console. They set the anchor and an offset. The window opens there on the next press, and moves without closing if they change it while it is open.

**Why this priority**: Placement is named in the request, and Principle VII requires nine positions for this kind of component.

**Independent Test**: Set each of the nine anchors in turn and open the window; check on screen that it sits at that corner, edge or center of the usable area, with the offset applied.

**Acceptance Scenarios**:

1. **Given** each of the nine anchors (`top-left`, `top`, `top-right`, `left`, `center`, `right`, `bottom-left`, `bottom`, `bottom-right`), **When** the window opens, **Then** it sits at that anchor of the usable area of the screen (the area not taken by the bar), moved by the user's horizontal and vertical offsets.
2. **Given** the default configuration, **When** the window opens, **Then** it is horizontally centered and its top is at about one fifth of the usable height, like a Spotlight window.
3. **Given** a size larger than the usable area, **When** the window opens, **Then** it is limited to the usable area and one warning names the key.
4. **Given** an offset that would push the window partly outside the usable area, **When** it opens, **Then** the window is moved back so that it is fully visible and one warning names the key.
5. **Given** the window is open, **When** the user changes the anchor, offset, size or screen in the file, **Then** the window moves or resizes within 1 second without closing.
6. **Given** a named screen, **When** the window opens, **Then** it opens there; **Given** that screen is absent, **Then** it opens on the focused screen with one warning.
7. **Given** an unknown anchor, **When** the configuration is read, **Then** the anchor falls back to the default with an error naming `modules.prism` and the key.
8. **Given** the results area is empty or full, **When** the window is anchored at a bottom edge, **Then** the window keeps its anchor edge fixed and the results area grows away from it (upward at the bottom anchors, downward at the top anchors).

---

### User Story 3 - Theme and style the window (Priority: P1)

A user sets the window background to a theme role with some transparency, picks a `primary` accent for the selected row, enlarges the radius, picks a different font and dims the screen behind it. Everything changes live. When they change the wallpaper, the window follows the new palette without flicker.

**Why this priority**: Theming and customization are the rest of the request, and Principles II and VII are non-negotiable.

**Independent Test**: Open the window in preview mode, change each color role, radius, border width, font, row size, shadow and backdrop setting one at a time in the file, and check the window changes within 1 second each time; then change the theme preset and wallpaper.

**Acceptance Scenarios**:

1. **Given** the default configuration, **When** the window opens, **Then** every color on it is a role of the active theme and the text has WCAG AA contrast against its background.
2. **Given** any color setting, **When** the user changes the role or opacity, **Then** the change applies within 1 second without closing the window.
3. **Given** a theme or wallpaper change while the window is open, **Then** all its parts follow within 1 second without flicker.
4. **Given** a role name the theme does not have, **When** the configuration is read, **Then** that part falls back to its default role with an error naming the key and the role.
5. **Given** a font family that is not installed, **When** the window opens, **Then** the default family is used with one warning naming the key and the family.
6. **Given** a radius, border width, padding, gap, row height or shadow setting, **When** it changes, **Then** the window follows it live, and a radius larger than half of the smaller side of its shape is limited with one warning.
7. **Given** the backdrop setting, **When** the window is open, **Then** the rest of the screen is dimmed as set, without blur (default: off), and a click on it closes the window if the click-outside setting is on.
8. **Given** the two bundled presets, a light and a dark wallpaper palette, and the fallback palette, **When** the window is shown in preview mode, **Then** it is readable and keeps its shape in each.

---

### User Story 4 - See the whole look with preview rows (Priority: P2)

A user setting up their theme wants to see what the selected row, a long title, an icon and the empty state will look like before any real source exists. They turn on the preview mode and the results area shows a fixed set of sample rows; they move the selection with the arrow keys, and every setting they change is visible at once.

**Why this priority**: Without real content, the preview is the only way to judge the customization, and the later search features reuse the same row look.

**Independent Test**: Turn preview mode on, open the window, move the selection with the arrow keys and the mouse, and check the selected-row look, scrolling, long titles and the empty state when the input matches nothing.

**Acceptance Scenarios**:

1. **Given** preview mode on, **When** the window opens, **Then** the results area shows the sample rows (with icon, title, subtitle and a right-side hint, one with a very long title, enough rows to need scrolling) and the first row is selected.
2. **Given** preview rows, **When** the user presses Down, Up, Page Down, Page Up, Home or End while the window holds the keyboard (Home and End move the selection only while the input is empty, and move the text cursor otherwise), **Then** the selection moves, the selected row is kept in view and the highlight follows the settings.
3. **Given** preview rows, **When** the pointer moves over a row, **Then** the row is hovered only (the selection does not change until a click), and a click selects it.
4. **Given** preview rows, **When** the user types, **Then** the sample rows whose title contains the text stay (a plain contains match, for the preview only) and when none match, the empty state is shown.
5. **Given** preview mode off (the default), **When** the window opens, **Then** the results area shows the empty state, which says that no source is installed.
6. **Given** a selected sample row, **When** the user presses Enter or double-clicks, **Then** nothing is run and the window stays open (sample rows have no action).

---

### User Story 5 - Later features can fill the window (Priority: P2)

A later feature, written as a module, wants to put real results into Prism. Through the public module interface it can read the mode and the query the user typed and the selection, supply rows, and ask Prism to close, without touching how the window is placed or styled.

**Why this priority**: The window is a skeleton, so the way in for the content must exist and be stable, even though no real source is built here.

**Independent Test**: Enable an example module that supplies three rows from the typed text; check the rows appear in the results area with the window's row look, selection works, and choosing a row is reported to the module.

**Acceptance Scenarios**:

1. **Given** a module that uses the Prism interface, **When** the user types, **Then** the module receives the current text within 50 ms of each change.
2. **Given** a module that supplies rows, **When** it sets them, **Then** the results area shows them with the configured look; **When** it sets none, **Then** the empty state shows.
3. **Given** a chosen row (Enter or double-click), **When** the choice is made, **Then** the module that supplied the row is told which row, and Prism does nothing else with it.
4. **Given** the input `@f report`, **When** the module is told, **Then** it receives the mode `files` and the query `report` (with the default letters); **Given** `report` with no prefix, **Then** it receives the mode `default` and the query `report`; **Given** `@x report` (not a known prefix), **Then** it receives `default` and the whole text.
4a. **Given** a recognized prefix, **When** it is typed, **Then** it is highlighted by a chip in the prefix color role while the text keeps the input text color, and an unknown `@x` is drawn as plain text.
5. **Given** a module that fails or is unloaded while it supplies rows, **When** the failure happens, **Then** its rows disappear, the window stays usable, and the failure is logged with the module's source.
6. **Given** a module, **When** it uses the interface, **Then** it cannot move, resize or restyle the window, and receives nothing private.

---

### User Story 6 - A bad choice degrades only itself (Priority: P2)

A user mistypes a value, uses a font they do not have, or sets a window larger than the screen. The window still opens, only the wrong key falls back to its default, and `lumen modules show prism` names the key and the problem.

**Why this priority**: Required by Principles IV, V and VII, and a shell must not break over a typo.

**Independent Test**: Put several wrong values in the file and check that the window opens, each wrong key falls back alone, and each is named in the module's report.

**Acceptance Scenarios**:

1. **Given** a value of the wrong type or out of range, **When** the configuration is read, **Then** that key falls back to its default with an error naming `modules.prism` and the key, and every other key keeps its value.
2. **Given** the Prism module is disabled or unloaded, **When** the user runs its commands, **Then** they report that the module is not running, and no window, backdrop or reserved space is left behind.
3. **Given** the window is open and the module is reloaded, **When** the reload finishes, **Then** the window is closed and the previously focused window has the keyboard.

---

### Edge Cases

- No focused window and no pointer on any screen: the window opens on the first screen.
- The screen layout changes while the window is open (a screen is unplugged): the window moves to a remaining screen and stays fully visible.
- The input holds a very long text: it scrolls inside the field and never changes the window size.
- A very long row title or subtitle: shortened with an ellipsis; the row height does not change.
- A row icon that does not exist: a neutral placeholder icon is shown; nothing fails.
- Text in right-to-left scripts and combined characters: shown and edited correctly.
- Input method (IME) composition: the composed text is shown in the input as the user types it.
- The bar changes its edge or style while Prism is open: Prism stays inside the usable area.
- Background opacity of 0: text and selection keep their own colors and the contrast rule is checked against what is behind the window, or a warning says it cannot be.
- A fractional display scale: sizes in logical pixels keep their proportions.
- Animation turned off or with a duration of 0: the window shows and hides at once, in the same final state.
- Opening while the pointer rests over a row: the row is not selected until the pointer moves.
- Maximum height reached: the results area scrolls and the window does not grow past the limit.

## Requirements *(mandatory)*

### Functional Requirements

**The window and its life**

- **FR-001**: Prism MUST be a module built only on the public module interface, with a manifest, a settings schema and commands. Its manifest or documentation MUST declare that it takes the keyboard when opened.
- **FR-002**: Prism MUST show at most one window, on one screen at a time, and MUST show nothing, reserve no space and hold no keyboard while the window is hidden.
- **FR-003**: Prism MUST declare the commands `toggle`, `open`, `close` and `clear` (empties the input) reachable through the module interface (for example `lumen modules call prism toggle`), MUST ship no default key binding, MUST register no global key, and MUST NOT open itself on any event the user did not ask for.
- **FR-004**: The window MUST be shown within 200 ms of an open command and hidden within 200 ms of a close command. While hidden it MUST use no measurable idle CPU.
- **FR-005**: When opened by command the window MUST take the keyboard, place the cursor in the input, and on close MUST return the keyboard to the previously focused window. The window MUST close on Escape at once, whether or not the input has text, on the toggle and close commands and, if the setting is on (default on), on a click outside it.
- **FR-006**: The window MUST NOT take the keyboard at load, on hover, or on any event the user did not ask for.
- **FR-007**: On an open command from another screen the window MUST move there, never exist on two screens, and a screen that disappears MUST move the window to a remaining screen without a crash.
- **FR-008**: The window MUST float over the desktop and MUST NOT cover or move other windows by reserving space.
- **FR-009**: On open the input MUST be empty unless the user's setting keeps the last text. Two commands in quick succession MUST end in the state of the last one.

**Parts of the window**

- **FR-010**: The window MUST consist of an input field with a placeholder text (set by the user), a results area with a scroll, a row look (icon, title, subtitle, right-side hint, selected and hovered states), an empty state, and an optional footer with key hints. Each part except the input MUST be switchable on or off by a setting where it makes sense (footer, subtitle, icon, hint). The window MUST have no mode indicator and no mode switcher.
- **FR-010 (prefix)**: The window MUST recognize a mode prefix at the very start of the input: `@a` (apps), `@f` (files), `@w` (web search), `@s` (commands) and `@g` (grep a file), followed by a space or the end of the text. The `@` sign MUST be fixed, and the letter of each of the five prefixes MUST be a named setting (default as listed) holding one lowercase letter or digit; a value that is invalid or duplicates another prefix's letter MUST fall back to that prefix's default with an error naming the key. Matching MUST be case-sensitive. The mode name given to modules MUST stay the fixed name of the mode (`apps`, `files`, `web`, `commands`, `grep`), not the letter, so that modules do not depend on the user's letters. With no recognized prefix the mode MUST be `default`, and an unknown `@x` MUST be treated as plain text in the default mode. The window MUST separate the mode from the query (the text after the prefix and one space) and give both to modules, and MUST highlight a recognized prefix with a chip in its own color role and opacity. The window MUST NOT act on the mode itself.
- **FR-011**: The input MUST support normal text editing (typing, cursor movement, selection, delete, cut, copy and paste, select all) and input-method composition, and MUST NOT limit the length of text it keeps.
- **FR-012**: The results area MUST have a maximum number of visible rows, set by the user, and MUST scroll beyond it. The window MUST NOT grow beyond its size limit, and the anchor edge MUST stay fixed when the results area grows or shrinks.
- **FR-013**: With the keyboard held, the selection MUST move with Up, Down, Page Up, Page Down, Home and End (Home and End only while the input is empty; in a non-empty input they move the text cursor), MUST stay in view, and MUST wrap or stop at the ends as the user sets (default: stop). The pointer MUST hover a row without changing the selection until a click.
- **FR-014**: The empty state MUST say that no source is installed when nothing supplies rows, and that nothing matched when the input has text and no row remains. Both texts are settings.
- **FR-015**: The **preview mode** (default off) MUST fill the results area with a fixed set of sample rows that exercise the look (icon, long title, subtitle, hint, enough rows to scroll). Sample rows are matched by a plain contains test for the preview only, and choosing one MUST do nothing.

**Placement**

- **FR-016**: The window MUST be placeable at each of nine anchors of the usable area of the screen: `top-left`, `top`, `top-right`, `left`, `center`, `right`, `bottom-left`, `bottom` and `bottom-right`. The default MUST be a Spotlight-like place: horizontally centered with the top at about one fifth of the usable height.
- **FR-017**: The user MUST be able to set a horizontal and a vertical offset from the anchor, in logical pixels, and a margin to the edges of the usable area.
- **FR-018**: The user MUST be able to name the screen the window opens on, or leave it to follow the focus (default). A named screen that is absent MUST fall back to the focused screen with one warning.
- **FR-019**: The window size MUST be a width and a maximum height in logical pixels, or a share of the usable area, and MUST be limited to the usable area with one warning when it does not fit. A window that would be partly outside the usable area MUST be moved back with one warning.
- **FR-020**: Changing the anchor, offset, margin, size or screen while the window is open MUST apply within 1 second without closing it.

**Theming and customization**

- **FR-021**: Every color of the window MUST be a semantic role of the active theme chosen by name, never a raw color: window background, border, input background, input text, placeholder text, mode prefix, row text, row subtitle, row hint, selected row background and its text, hovered row background, separator, footer, empty-state text, shadow and backdrop. Backgrounds, shadow and backdrop have an opacity. A role the theme does not have MUST fall back to that part's default with an error naming the key and the role. The window MUST follow theme changes live without flicker, and its text MUST meet WCAG AA contrast against its background (a choice that breaks it is corrected or reported, per Principle II).
- **FR-022**: Fonts MUST be settable for the window as a whole and for its text roles (input, placeholder, row title, row subtitle, row hint, footer, empty state): family, size, weight and italic. A family that is not installed MUST fall back to the default with one warning naming the key and the family.
- **FR-023**: The following MUST be named settings with documented defaults, validation, live effect and per-key fallback: the corner radius of the window, input and rows; the border width; the padding of the window, the input and the rows; the gap between the input and the results, and between rows; the row height; the icon size; the separator line (on or off, width); the shadow (on or off, blur, offset, opacity); the backdrop (off, dim, with strength); the input position (`top` or `bottom` of the window, default `top`); the placeholder text; the empty-state texts; and the footer hints.
- **FR-024**: Animation MUST be configurable: the kind (`fade`, `scale`, `slide`, `none`), the duration, and a way to turn it off. The default MUST come from the shared motion tokens. With `none` or a zero duration the window MUST show and hide at once.
- **FR-025**: Sizes MUST be in logical pixels so proportions are the same at any display scale. A value out of range MUST fall back with an error naming the module and the key. A valid value that cannot be used on a small screen MUST be limited with one warning. A radius larger than half of the smaller side of its shape MUST be limited with one warning.
- **FR-026**: The defaults of every setting MUST come from the shared design tokens (Style and Theme) where a token exists, so that a later global override of a token reaches Prism without change.
- **FR-027**: No visual value in the Prism module MAY be fixed in code where the user cannot override it. A test MUST scan the module and fail on a size, radius, margin, padding, font value, duration or color that is neither a token nor a declared setting.
- **FR-028**: Every setting MUST follow the layered, override-only rules of the configuration: defaults are never written to the user's file, unknown keys are kept, a bad value falls back for that key only, a reset removes the override, and a change applies within 1 second. Every setting MUST appear in Prism's schema and in the merged schema.

**The interface for later features**

- **FR-029**: The module interface MUST let a module read the current mode and query (and the whole input text) and the selected row, supply rows (icon, title, subtitle, hint, an identifier), and ask Prism to close. The mode and query MUST reach the module within 50 ms of each change. A module MUST NOT be able to move, resize or restyle the window, and MUST receive nothing private.
- **FR-030**: When a row is chosen (Enter or double-click) Prism MUST tell the module that supplied it which row was chosen, and do nothing else with it. Prism MUST NOT search, rank, run or open anything in this feature.
- **FR-031**: A module that fails or is unloaded while it supplies rows MUST lose its rows only; the window and other parts MUST keep working, and the failure MUST be logged with the module's source.
- **FR-032**: The interface additions MUST be versioned and documented, with a migration note if anything changes for existing modules, and an example module that supplies a few rows from the typed text MUST be included.

**Failure, documentation and checks**

- **FR-033**: A bad setting, a missing font, a missing icon or a missing screen MUST NOT crash or freeze the shell or the window (Principle IV).
- **FR-034**: Prism MUST appear in `lumen modules show prism` with its state, settings errors and warnings.
- **FR-035**: The documentation MUST list Prism's keyboard commands and which need a binding, with an example binding for niri; say that Prism takes the keyboard when opened because it is a search box, and gives it back on close; list the popup it opens (the window itself, closed by Escape); describe the interface for later features with the working example; and list the whole customization surface.
- **FR-036**: Automated or documented checks MUST show: that the window holds no keyboard focus until its command runs; that Escape closes it and returns the focus; that every color comes from a role; and that each customization setting applies live.

### Customization surface *(required by principle VII's quality gate)*

| Aspect | What the user can set |
|--------|-----------------------|
| Colors (roles + opacity) | window, border, input, input text, placeholder, mode prefix, row text, subtitle, hint, selected row and its text, hovered row, separator, footer, empty state, shadow, backdrop |
| Fonts | family, size, weight, italic: global to the window, and per role (input, placeholder, row title, subtitle, hint, footer, empty state) |
| Shape | radius of window, input and rows; border width; separator; shadow blur and offset; backdrop strength |
| Spacing | padding of window, input and rows; gap between input and results; gap between rows; row height; icon size; margin to the edges |
| Placement | nine anchors, horizontal and vertical offset, width and maximum height (pixels or share), screen choice, input position |
| Mode prefixes | the letter of each of the five prefixes (apps, files, web, commands, grep) |
| Content shown | placeholder text, empty-state texts, footer hints, which row parts are shown, preview mode |
| Behavior | close on click outside, keep last text, selection wrap, visible row count |
| Motion | animation kind, duration, off switch |

### Keyboard and popups *(required by principle VIII's quality gate)*

- **Commands**: `toggle`, `open`, `close`, `clear`. All need a binding in the compositor; none has a default binding.
- **Implicit keyboard**: yes, only after the opening command, because Prism is a search box (a module whose purpose is keyboard input). It takes no focus at load, on hover or by itself, and releases the focus when it closes.
- **Keys while open**: text editing, Up, Down, Page Up, Page Down, Home, End, Enter and Escape.
- **Popups**: the Prism window itself; it closes on Escape.

### Key Entities

- **Prism window**: the single floating window; has a state (hidden or shown), a screen, an anchor, an offset and a size.
- **Input**: the text the user types and the placeholder; it holds an optional mode prefix and the query that follows.
- **Mode**: `default`, or one chosen by a prefix (`apps`, `files`, `web`, `commands`, `grep`); given to modules, never acted on by the window.
- **Row**: an icon, a title, a subtitle, a right-side hint and an identifier; supplied by a module, or a sample row in preview mode; can be hovered and selected.
- **Source**: a module that supplies rows to Prism and is told which row was chosen; it has no access to the window itself.
- **Results area**: the scrollable list of rows, with the selected row and the empty state.
- **Prism settings**: the module's named settings (placement, size, colors by role, fonts, shape, spacing, content shown, behavior, motion), layered over the design tokens.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user who binds one key can open Prism and start typing within 200 ms, and close it with Escape in under 200 ms, on the default configuration with no other setup.
- **SC-002**: For each of the nine anchors, the window sits within 1 logical pixel of the expected position of the usable area, offsets included, on 100% of checked screens (one, two and a scaled screen).
- **SC-003**: Every setting of the customization surface changes the open window within 1 second, with no restart and no flicker, in 100% of the checked settings.
- **SC-004**: Prism never holds keyboard focus before its command runs, and after any close the previously focused window gets it back, in 100% of checked runs.
- **SC-005**: In the two presets, a light wallpaper palette, a dark one and the fallback palette, all text meets WCAG AA contrast and a theme change is visible within 1 second.
- **SC-006**: Typing in the input never drops or reorders characters, and the text reaches a listening module within 50 ms, in 100% of checked runs of 200 characters typed at a fast rate.
- **SC-007**: A module author can supply rows to Prism by following the documentation in under 10 minutes, with no change to the window.
- **SC-008**: With 10 deliberately wrong settings in the file, the window still opens, each wrong key falls back alone, and 100% of them are named in the module's report.
- **SC-009**: The window and its parts keep their frame rate at the display refresh rate while opening, scrolling a list of 1000 rows and typing; and while hidden Prism adds no measurable idle CPU or reserved space.

## Assumptions

- Prism is a new module in the `modules` directory, enabled by default, but the window stays hidden until the user's command.
- The nine anchors follow Principle VII for the launcher kind of component; a module MAY offer more positions later.
- Searching apps, files and commands, running commands, web search, grep in a file, ranking, history, icons from the system, sources and the list of modes are separate features. This feature supplies none of them.
- The preview mode and its sample rows exist only to judge and test the look; they are not a product feature and are off by default.
- A global override of design tokens, a user-written theme and the settings app are separate features. Prism's defaults come from the tokens, so they reach Prism without change.
- The window is not blurred behind it in this feature (the backdrop is only a dim), because a compositor-level blur depends on the compositor and is left to a later feature.
- One Prism window at a time is enough; one window per screen at once is out of scope.
- Rows, the mode and the query reach later features through Prism's own source slot, so the module interface gains no member. The only interface change is a higher limit of settings per module (the customization surface is larger than the old limit); it is additive and needs no migration (see the plan).
- The response-time and frame-rate targets (SC-001, SC-006, SC-009) are checked by hand on a real session, with the method written in the quickstart; only the open and close cycles and memory are scripted.
- Every spacing, radius and size that the window draws takes its value from a named setting or a shared token; values that have no key of their own follow a documented neighbor key (see the plan's contract), so none is fixed in code.
