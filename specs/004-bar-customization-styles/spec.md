# Feature Specification: Bar Customization and Styles

**Feature Branch**: `004-bar-customization-styles`

**Created**: 2026-11-08

**Status**: Draft

**Input**: User description: "Feature IV. Bring bar to Constitution VII. Also add 3 styles for bar: islands (or pills) where places are separated, solid (as it is now) and frame like this [Image #1] (position still must be configurable)"

**Constitution principles touched**: VII (the bar and its items become fully customizable: colors, fonts, shape, spacing and placement, and the bar is the first component brought to the principle), VI (one design language), VIII (explicit keyboard interactivity: this feature adds no key handling, and none of its new surfaces takes keyboard focus), II (every bar color is still a semantic role from Theme, now chosen by the user per part of the bar), V (every new setting lives in the layered, override-only configuration with per-key fallback), III (the new facts reach third-party items through the public module interface only), IV (a bad value, a missing font or a screen that does not fit degrades only what it touches).

## Overview

Feature III gave Lumen a bar, but its look and place are mostly fixed: one flat strip, on the top or bottom edge, with spacing, shape and fonts decided in code. Constitution principle VII now says that a user must be able to change any visible aspect of Lumen without touching code. This feature brings the bar, and the three built-in items that live in it, up to that rule, and adds the three looks that people expect from a bar:

1. **Solid**: the strip as it is today. One continuous bar along the screen edge, across the whole length of the screen.
2. **Islands** (pills): the three places are separate rounded shapes with gaps between them. Left, center and right each look like their own pill. The space between the pills shows the desktop and does not catch the pointer.
3. **Frame**: the bar is part of a border that goes around the whole screen. The strip on the chosen edge holds the items, and thin border strips on the other three sides join it, so the desktop looks like a rounded window seen through the frame, with rounded inner corners (as in the reference picture, where the strip is on the left edge and the screen content has rounded corners).

Whatever the style, the user decides where the bar is: **any of the four edges** (top, bottom, left, right). A vertical bar is new: items stack from top to bottom, and the built-in items adapt (for example the clock shows hours above minutes, as in the reference picture).

Everything about how the bar looks comes from the configuration file, for every screen, live and without restart: colors (chosen as semantic roles of the active theme, with opacity), fonts (family, size, weight and style), the corner radius of every part, the margins, paddings and gaps, the border width of the frame, and the bar's thickness. The built-in items follow the bar's choices by default and can be customized on their own. Third-party items follow the bar's choices without extra work, because the bar gives them the resolved facts, and they can take their own settings as any module can.

## Clarifications

### Session 2026-11-08

- Q: Which part of principle VII does this feature cover? → A: The bar and its three built-in items, in full: colors by role, fonts, radius, margins and paddings, and placement on four edges. Global changes that affect every component (a theme the user writes, global font, radius and spacing overrides, the settings app) are separate features. The bar's values default to the shared design tokens, so a later global override reaches the bar without change.
- Q: Does the frame style draw on all four sides? → A: Yes. The strip on the chosen edge carries the items; the other three sides get a border strip of a configurable width (it may be set to zero), all in the bar's color, and the frame's inner corners are rounded with a configurable radius. Windows never go under the frame.
- Q: What does the user choose for colors? → A: A semantic role of the active theme for each part of the bar (bar background, island background, frame, item text, and so on) and an opacity for backgrounds. The bar never takes a raw color from its own settings; a user who wants a color the theme does not have writes a theme, which is a separate feature (principle II).
- Q: What happens to a third-party item that was written for a horizontal bar when the bar is vertical? → A: It is still shown. It is told the bar's direction and its available size; if it ignores them, it is clipped to the bar's thickness, as an item wider than its limit is clipped today, and nothing else is disturbed.
- Q: What is the default? → A: Solid, on the top edge, looking as the bar does after Feature III. A user who sets nothing sees no change.
- Q: What happens to an island whose place holds no shown items? → A: It is not shown. If every place is empty in the islands style, nothing is drawn and no space is reserved on that screen; the island and its space appear when an item is shown.
- Q: Does this feature change how the bar is used from the keyboard? → A: No. Keyboard use stays explicit: a module reacts to the keyboard only through its own command that the user binds in the compositor (for the bar, `lumen modules call bar focus`, as in Feature III), except modules that obviously need the keyboard, such as a launcher, which may take it on their own. Every popup closes on Escape (FR-040 of Feature III). This feature adds no key handling: the focus command and its keys work as in Feature III on every edge, and the sound item's keys do not change.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose how the bar looks: solid, islands or frame (Priority: P1)

A user opens the configuration file, sets the bar's style to `islands`, saves, and the bar changes into three separate rounded pills (left, center, right) with the desktop visible between them. They change it to `frame` and the bar becomes a border around the screen with rounded inner corners. They change it back to `solid`.

**Why this priority**: The three looks are the visible half of this feature and the reason most users would customize the bar in the first place.

**Independent Test**: Start with the default bar, change only the style setting between its three values, and check each result on screen: one continuous strip, three separate pills, a frame around the screen. Nothing else needs to be configured.

**Acceptance Scenarios**:

1. **Given** the default configuration, **When** the user starts Lumen, **Then** the bar looks as it did after Feature III (solid, top edge).
2. **Given** a running bar, **When** the user sets the style to islands, **Then** within 1 second each place that holds at least one item is drawn as its own rounded shape, the gaps between the shapes show the desktop, and a click in a gap reaches the window below.
3. **Given** the islands style and an empty place, **When** the bar is drawn, **Then** nothing is drawn for that place.
4. **Given** a running bar, **When** the user sets the style to frame, **Then** within 1 second a border of the bar's color surrounds the whole screen, the items sit in the strip on the chosen edge, and the area inside the frame has rounded corners.
4a. **Given** the frame style on any edge, **When** the user looks at the four corners of the screen, **Then** the frame is one unbroken band of one color: the strip and the border sides meet at every corner with no gap, whatever order the compositor lays its surfaces out in, and the same holds when the frame radius or a border width is 0.
4b. **Given** the frame style on any edge (bottom and left were reported), **When** the user looks along each side of the screen, **Then** there is no gap anywhere between a corner and the next part of the frame: each border panel meets the strip and the corner blocks edge to edge, not only at the very corner of the screen.
5. **Given** any style, **When** the user maximizes a window, **Then** the window stops at the inner edge of the bar or frame and is never covered by it.
6. **Given** any style, **When** the user changes the theme, **Then** every part of the bar follows within 1 second with no flicker.
7. **Given** an unknown style value, **When** the configuration is read, **Then** the style falls back to solid and the error names `modules.bar` and the key.

---

### User Story 2 - Put the bar on any of the four edges (Priority: P1)

A user wants the bar on the left, as in the reference picture. They set the edge to `left`. The bar moves to the left edge, its items stack from top to bottom (left items at the top, right items at the bottom), and the clock shows hours above minutes.

**Why this priority**: Position was explicitly required to stay configurable in every style, and the constitution now asks for four edges for the bar. Without vertical layout the frame picture cannot be built.

**Independent Test**: Set the edge to each of top, bottom, left and right, with each of the three styles, and check on screen that the bar sits on that edge, keeps its items in list order along the bar, and reserves its space.

**Acceptance Scenarios**:

1. **Given** the edge set to left or right, **When** the bar is shown, **Then** the bar is a vertical strip on that edge, the items of the left place are at the top, the center items in the middle and the right items at the bottom, each in list order.
2. **Given** a vertical bar, **When** the built-in items are shown, **Then** each fits the bar's thickness without clipping (the clock shows hours above minutes by default; the sound and battery items stay readable).
3. **Given** a vertical bar, **When** the user runs the focus command, **Then** it works as in Feature III (this feature changes no keys).
4. **Given** a vertical bar and more items than fit the height of the screen, **When** the bar is drawn, **Then** the same dropping rule as before is applied along the height (end of the right list first, then the center, then the left), nothing overlaps, and the dropped items are named in `lumen modules show bar`.
5. **Given** a running bar, **When** the user changes the edge, **Then** the bar moves within 1 second, the reserved space moves with it, and the items that did not change are not recreated more than once.
6. **Given** an edge that is not one of the four values, **When** the configuration is read, **Then** the edge falls back to top with an error naming the module and the key.

---

### User Story 3 - Change every visual detail of the bar from the file (Priority: P1)

A user wants a thicker bar, a gap of 10 around it, fully round pills, a bigger italic font, a translucent background that uses the theme's `surfaceVariant` role, and no frame on the sides without items. They do all of it in the configuration file, without touching any code, and each change shows on screen within a second.

**Why this priority**: This is what principle VII asks for and what makes the bar "as customizable as themeable". It is the center of the feature.

**Independent Test**: For each setting in the customization surface below, change it in the file and check that the visible effect matches the setting, applies within 1 second, and that an invalid value falls back for that key only.

**Acceptance Scenarios**:

1. **Given** a running bar, **When** the user sets the bar's thickness, **Then** the bar's thickness on its edge changes within 1 second, the reserved space follows, and the items fit the new thickness.
2. **Given** a running bar, **When** the user sets the outer margin (the gap between the bar and the screen edge), the padding inside the bar, the gap between items, or, for islands, the gap between places, **Then** each change shows within 1 second and the layout stays free of overlaps.
3. **Given** any style, **When** the user sets the corner radius of the bar (of each island, or of the solid strip, and of the frame's inner corners), **Then** the shapes change accordingly, and a radius larger than half of the shape's smaller side is shown as fully round and is not an error.
4. **Given** any style, **When** the user picks a theme role for a part of the bar (the bar background, the island background, the frame, the item text, the highlight of the focused item) and an opacity for the backgrounds, **Then** that part uses the role's color from the active theme with that opacity, and follows the theme when it changes.
5. **Given** a running bar, **When** the user sets the font family, size, weight or italic for the bar, **Then** every text in the bar changes within 1 second, and the bar's thickness and the layout adapt without clipping the text.
6. **Given** the frame style, **When** the user sets the width of the border on the other three sides, **Then** the border changes, and a width of zero removes the border on those sides while the bar's strip and the rounded inner corners stay.
7. **Given** a value that is out of range (a negative size, a size larger than the screen allows, a text size of zero), **When** the configuration is read, **Then** that key falls back to its default with an error that names the module and the key, and every other key keeps its value.
8. **Given** the user's file after any sequence of edits and resets, **Then** it holds only the values the user set, the defaults are not written into it, and unknown keys are kept.

---

### User Story 4 - Items follow the bar and can be customized on their own (Priority: P2)

A user who changed the bar's font and padding sees that the clock, the sound level and the battery follow, and that a third-party item (the example item) follows too. They then give the clock its own bigger font and its own pill-shaped background, without changing the other items.

**Why this priority**: A bar where only the frame is customizable but the items ignore it would not meet principle VII. It builds on stories 1 to 3.

**Independent Test**: Change the bar's font and padding and check that the three built-in items and the example item follow. Then set a font on the clock only and check that only the clock changes.

**Acceptance Scenarios**:

1. **Given** the bar's font family, size, weight and style set by the user, **When** an item that sets no font of its own is shown, **Then** it uses the bar's font, and follows the setting live.
2. **Given** a third-party item written for the previous version of the module interface, **When** the bar's look is changed, **Then** the item still loads and works, and if it uses the shared design tokens it follows the changes that are shared (colors, motion).
3. **Given** a third-party item written for this version, **When** it reads the facts of its placement, **Then** it gets the bar's edge and direction, thickness, available size, resolved font, padding and corner radius, and nothing private.
4. **Given** each built-in item, **When** the user sets its own font, padding, corner radius or background role in that item's own settings, **Then** only that item changes, and the same kind of settings is available to any module through its own manifest.
5. **Given** a vertical bar, **When** a third-party item ignores the direction, **Then** it is still shown, clipped to the bar's thickness, and nothing else is disturbed.
6. **Given** a popup opened by an item, **When** the user has set the bar's radius, padding, border and colors, **Then** the popup uses the matching settings for its own corner radius, padding and colors, and follows them live.

---

### User Story 5 - A bad choice degrades only itself (Priority: P2)

A user types a font that is not installed, a role name that the theme does not have, or an opacity of 3. The bar keeps working, uses the default for that one value, and the log and `lumen modules show bar` say which key was wrong and why.

**Why this priority**: Principle V and IV require per-key fallback, and a bar that fails on a typo would be the first thing the user sees broken.

**Independent Test**: Put one bad value at a time into each kind of setting (font, role, opacity, size, style, edge) and check that only that key falls back and the bar stays up.

**Acceptance Scenarios**:

1. **Given** a font family that is not installed, **When** the bar is shown, **Then** the bar uses the default font family, one warning names the key and the font, and the rest of the settings are applied.
2. **Given** a color role that the active theme does not have, **When** the bar is shown, **Then** that part uses the default role, and an error names the key and the role.
3. **Given** a combination that cannot be honored (a frame border wider than half the screen, or a thickness that leaves no room for items), **When** the bar is shown, **Then** the value is limited to the largest usable value, one warning says so, and the bar is not broken.
4. **Given** several bad values at once, **When** the bar is shown, **Then** every bad key is reported, and every good key is applied.

---

### Edge Cases

- A screen too small for the frame, the margins and the thickness together: the values are limited so that a usable area remains, with a warning that names the keys.
- Islands with all three places empty: nothing is drawn, no space is reserved on that screen, and there is no error. When an item is shown, its island and the reserved space appear.
- Several screens with different sizes or scales: every bar is laid out for its own screen from the same settings, and one screen's overflow does not affect another.
- Changing the edge or style while a popup is open: the popup closes and the item may open it again.
- The frame style with the bar on a screen that has the bar turned off by the screens setting: no frame is drawn on that screen.
- Frame style: the border on the three non-bar sides and the strip meet at the same color and without a visible seam; a theme change changes both together.
- Frame style, corners of the screen: the compositor lays a surface out inside the space that earlier surfaces reserved, so a side panel can end short of a screen corner. The frame MUST NOT show a gap there: the corner blocks (where the strip and the borders meet) are drawn by surfaces that do not depend on that layout. The same layout rule applies along the sides: a border panel MUST NOT add its own offset for space the compositor already keeps free for earlier surfaces (the strip), because the two offsets add up and leave a gap of the strip's width between the panel and the corner block.
- A radius, thickness or gap set to a very large value: limited as above, never an error that stops the bar.
- A font weight or style that the family does not offer: the closest available face is used, without an error.
- A third-party item that is wider than the vertical bar's thickness: clipped (see Story 4).
- Fractional display scale: sizes in the settings are in logical pixels, so the bar has the same proportions at any scale.
- A hot reload of the configuration while a style change is in progress: the last saved value wins and no item is created twice.

## Requirements *(mandatory)*

### Functional Requirements

**Styles**

- **FR-001**: The bar MUST offer exactly three styles chosen by one setting: `solid`, `islands` and `frame`. The default MUST be `solid`, and a user who changes nothing MUST see the bar as it was after Feature III.
- **FR-002**: In the `solid` style the bar MUST be one continuous strip along the chosen edge of each chosen screen, with the outer margin set by the user (default as today).
- **FR-003**: In the `islands` style each place that holds at least one shown item MUST be drawn as its own rounded shape. The three places MUST NOT share a background. A place with no shown items MUST draw nothing. The gaps between the shapes MUST show what is behind them and MUST NOT take pointer input.
- **FR-004**: In the `frame` style the bar MUST draw a border of one color around the whole screen. The strip on the chosen edge holds the items. The border on the other three sides has its own width (zero is allowed). The inner edge of the frame MUST have rounded corners with a radius set by the user. The frame MUST be one unbroken band: at every corner of the screen the strip and the border sides meet with no gap, in the same color, for every edge and every combination of border width and radius (including 0). Windows MUST NOT be drawn under the frame.
- **FR-005**: The style MUST be changeable live without a restart, for every screen at once, without recreating the items more than once and without leaving anything behind from the previous style.
- **FR-006**: In every style the bar MUST reserve its space so that windows are not covered (as in FR-003 of Feature III) and MUST release it when the bar is hidden, disabled or unloaded. The reserved space MUST follow the style, the edge, the thickness and the margins.

**Placement**

- **FR-007**: The bar MUST be placeable on each of four edges: top, bottom, left and right. The default MUST stay top. Placement MUST work with every style.
- **FR-008**: On a vertical bar (left or right) the items MUST be laid out from top to bottom: the left place at the top, the center place in the middle, the right place at the bottom, each in list order.
- **FR-009**: The rule that drops items that do not fit (FR-015a of Feature III) MUST apply along the bar's own direction: the width of a horizontal bar and the height of a vertical bar. The order of dropping, the report and the return of dropped items MUST stay as they are.
- **FR-010**: This feature MUST NOT change how the bar is operated from the keyboard, and MUST follow Principle VIII of the constitution (explicit keyboard interactivity). Keyboard use stays explicit: a module reacts only to its own command that the user binds in the compositor. The only keyboard command among the modules of this feature is the bar's `focus` command; the clock, sound and battery items and the example item have none. No surface that this feature adds or changes (the strip, island, border or corner surfaces) MUST hold keyboard focus: the bar holds it only while the `focus` command's session is active, and a popup only while it is open. Every popup MUST keep closing on Escape (FR-040 of Feature III), and the only popup of this feature is the example item's.
- **FR-011**: The focus command, the screen setting and the `focusScreen` setting MUST keep working on every edge, with the keys they have in Feature III.

**Customization of the bar (the customization surface)**

- **FR-012**: Each of the following MUST be a named setting of the bar in the user's configuration file, with a documented default, validation, live effect and per-key fallback: the style; the edge; the thickness; the outer margin (the gap between the bar and the screen edge); the padding inside the bar; the gap between items; the gap between places (islands); the corner radius of the bar's shapes (solid strip, each island); the corner radius of the frame's inner corners; the width of the frame border on the three non-bar sides; the padding and the corner radius that the bar gives its items; the font family, size, weight and italic for the bar's text; for each background and text part of the bar (strip, island, frame, text, focus highlight), the theme role used; the opacity of each background; and for the popup surface the corner radius, padding, border width and the background and border roles.
- **FR-013**: Every color in the bar MUST be a semantic role of the active theme chosen by name, never a raw color. A role name the theme does not have MUST fall back to that part's default role with an error that names the key and the role. The bar MUST follow theme changes live.
- **FR-014**: Fonts MUST be settable for the bar as a whole. A font family that is not installed MUST fall back to the default family with one warning naming the key and the family. A weight or style that the family lacks MUST use the nearest available face without an error.
- **FR-015**: Sizes (thickness, margins, paddings, gaps, radii, border width) MUST be given in logical pixels, so that the proportions are the same at any display scale. A value out of range MUST fall back with an error naming the module and the key. A value that is valid but cannot be used on a small screen MUST be limited to the largest usable value with one warning. A radius larger than half of the shape's smaller side MUST be shown as fully round and MUST NOT be an error.
- **FR-016**: The defaults of every new setting MUST come from the shared design tokens (Style and Theme) where a token exists, so that a later global override of a token reaches the bar without any change to the bar.
- **FR-017**: No visual value in the bar module MAY be fixed in code where the user cannot override it. A test MUST scan the bar and the built-in items and fail on a size, radius, margin, padding, font value or color that is neither a token nor a declared setting (extending the checks already made for colors and magic values).
- **FR-018**: Every new setting MUST follow the layered, override-only rules of the configuration: defaults are never written to the user's file, unknown keys are kept, a bad value falls back for that key only, a reset removes the override, and a change applies within 1 second. Every new setting MUST appear in the bar's schema and in the merged schema.

**Items**

- **FR-019**: The module interface MUST give every bar item, in the facts of its placement, the bar's edge, whether the bar is horizontal or vertical, thickness, the size available to the item, and the bar's resolved font, padding and corner radius. The facts MUST follow the settings live. An item MUST receive nothing private.
- **FR-020**: The change to the module interface MUST be additive and compatible: a module built for the earlier minor version MUST load and behave unchanged, a module that needs the new version on an older shell MUST be refused with the required and available versions (as in FR-020 of Feature III), and the version note in the documentation MUST say "additive, no migration".
- **FR-021**: An item that sets no font of its own MUST use the bar's font. Built-in items MUST do this.
- **FR-022**: Each built-in item (clock and date, sound, battery) MUST offer settings, in its own module's namespace, for its own font (family, size, weight, italic), its own padding, and its own corner radius and background role and opacity (default: no background of its own). Each MUST work on all four edges: on a vertical bar the clock MUST show its hours above its minutes by default, and the sound and battery items MUST stay readable within the bar's thickness.
- **FR-023**: The same kinds of setting MUST be available to any module through its own manifest, with no special permission, so that a third-party item can be as customizable as a built-in one. The example item MUST show it.
- **FR-024**: An item that ignores the bar's direction or size MUST still be shown and MUST be clipped to the bar's thickness. It MUST NOT be able to move, resize or restyle the bar, other items or other modules' contributions (FR-023 of Feature III).
- **FR-025**: The popup surface that the bar hosts MUST take its corner radius, padding, border and colors from settings of the bar (with defaults from the shared tokens), MUST follow them live, and MUST keep its rules from Feature III (placement, size limits, one popup at a time, closing).

**Reliability and documentation**

- **FR-026**: A bad value in any bar setting MUST NOT stop the bar, other items or the shell. Every bad key MUST be reported with the module and the key, and every good key MUST be applied.
- **FR-027**: Changing the style, the edge or any visual setting MUST NOT cause unbounded memory growth, and an idle bar MUST NOT use noticeable CPU, in any style on any edge.
- **FR-028**: The bar documentation MUST list every setting of the customization surface with its default and range, describe the three styles (with the reference look for the frame), the four edges, the vertical layout and the keyboard use, and include a configuration example for each style.
- **FR-029**: The slot reference in the documentation and the contract MUST describe the new facts of FR-019. The example item MUST read them and adapt to both directions, so that a new author can copy it.
- **FR-030**: The bar MUST NOT use the network or send telemetry, as before.

### Customization surface *(required by principle VII's quality gate)*

| Part | Colors | Fonts | Radius | Margins, paddings, gaps | Placement |
|------|--------|-------|--------|-------------------------|-----------|
| Bar (all styles) | Background role and opacity, text role | Family, size, weight, italic | Strip or pill radius, item radius | Thickness, outer margin, inner padding, item gap, item padding | Four edges, screens |
| Islands | Island background role and opacity | (inherits the bar's) | Radius of each pill | Gap between places | (as the bar) |
| Frame | Frame role and opacity | (inherits the bar's) | Radius of the inner corners | Border width on the other three sides | (as the bar) |
| Focus highlight | Highlight role | (inherits the bar's) | (inherits the item's) | (inherits the item's) | (follows the item) |
| Clock, sound, battery | Background role and opacity, text role | Own family, size, weight, italic | Own radius | Own padding | Follow the bar |
| Popup | Background and border role | (inherits the item's) | Own radius | Own padding | Chosen by the bar, never over the item |
| Third-party items | Through theme roles | Through the bar's facts or own settings | Own settings | Own settings | Follow the bar |

### Key Entities

- **Bar Style**: One of `solid`, `islands` or `frame`; decides how the bar's shapes are drawn.
- **Island**: The rounded shape of one place in the islands style; exists only while the place holds a shown item.
- **Frame**: The border around the screen in the frame style: the strip on the bar's edge plus the border on the other three sides, with rounded inner corners.
- **Edge**: One of top, bottom, left, right. Left and right make the bar vertical.
- **Direction**: Horizontal for top and bottom, vertical for left and right; the direction along which items are laid out.
- **Customization Surface**: The full set of the bar's visual and placement settings, listed above.
- **Role Choice**: A semantic theme role chosen by name for one part of the bar, with an opacity for backgrounds.
- **Placement Facts**: What the bar tells an item about its place: edge, direction, thickness, available size, and the bar's resolved font, padding and corner radius.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every one of the three styles on every one of the four edges (12 combinations), the bar appears correctly on each connected screen within 1 second of the setting change, windows never overlap it, and the shapes match the descriptions of the styles.
- **SC-002**: A user who changes nothing sees no visual difference from the bar of Feature III (checked by comparing the geometry and colors of the default bar before and after).
- **SC-003**: Each setting in the customization surface can be changed in the file and shows on screen within 1 second, in 100% of the settings, with no restart and no code change.
- **SC-004**: In a test set of at least 8 kinds of bad value (unknown style, unknown edge, negative size, oversize value, missing font, unknown role, opacity out of range, wrong type), 100% fall back for that key only, are reported with the module and the key, and leave the bar and every other setting working.
- **SC-005**: A theme change reaches 100% of the colors of the bar and its items, in all three styles, within 1 second and without flicker, across at least two presets, a light and a dark wallpaper-derived palette, and the fallback palette.
- **SC-006**: A test finds no size, radius, margin, padding, font value or color in the bar module and the three built-in items that is neither a shared token nor a declared setting.
- **SC-007**: Every module that loaded and ran in the bar before this feature (all the fixtures from Feature III and Modules) still loads and behaves the same.
- **SC-008**: The focus command and Escape for popups work on all four edges exactly as in Feature III, and no key handling of the bar or the built-in items is changed by this feature. In every style and on every edge, no surface of the bar takes keyboard focus until the focus command runs, and none keeps it after a popup closes or the user leaves with Escape (checked from the compositor's list of layer surfaces; leaving with Escape is checked by hand, because key events cannot be injected in the test environment).
- **SC-009**: On a vertical bar, 100% of the built-in items show without clipping at the default thickness, and an item that ignores the direction is clipped to the thickness without disturbing any other item.
- **SC-010**: Over 100 style or edge changes in a row, memory use stays within 12% of the use after the first change (the same noise margin as the bar reload check in Feature III), no item is duplicated or left behind, and the idle CPU of the shell with the bar stays under 1% over 5 minutes in each style.
- **SC-011**: A new author, using only the documentation and the example item, makes an item that looks right on both a horizontal and a vertical bar in under 10 minutes.
- **SC-012**: After any sequence of edits and resets of the new settings, the user file holds no value the user did not explicitly set, and all unknown keys are still there.

## Assumptions

- **Scope of principle VII**: This feature covers the bar, its three built-in items and the popup surface the bar hosts. Anything global is a separate feature: a theme file the user writes, global overrides of the font, radius and spacing tokens, the graphical settings app, and the other components (notifications with their eight positions, the launcher with its nine). The bar's defaults come from the shared tokens, so those later features reach the bar without change.
- **Colors** are chosen as theme roles, never as raw color values, because the constitution requires every color to come from the theme service. Opacity is a separate number for backgrounds.
- **Frame style, all sides**: the frame goes around every chosen screen. The border width on the three non-bar sides defaults to a small visible width and may be zero. It follows the reference picture, where the strip is on the left, and the same look is available on every edge.
- **Islands and input**: the gaps between islands let the pointer through. The strip itself, each island and each item take pointer input as before.
- **Vertical text**: items on a vertical bar stay upright (not rotated), as in the reference picture, where text is stacked or kept short. Rotated text is not part of this feature.
- **Units**: all sizes are in logical pixels.
- **Built-in item changes**: the clock shows hours above minutes on a vertical bar by default, with the same clock format and seconds settings as before for the horizontal case. The exact vertical forms of sound and battery are decided in planning, subject to FR-022.
- **Interface version**: the new placement facts are additive. Planning decides whether this raises the interface version; the compatibility rule of Modules applies either way.
- **Not in this feature**: new items (workspaces, active window, tray, and the dashboard popup shown in the reference picture), auto-hide, floating over windows, an overflow menu, different lists or styles per screen, rotated text, animated style changes, and per-item placement outside the three places.
- **Hardware**: the compositor must support reserving screen space on each of the four edges. This is expected of niri, the only supported compositor.

## Refinements

### 2026-11-08 (constitution 2.0.0)

- **Addition**: The constitution now has Principle VIII (explicit keyboard interactivity, version 2.0.0). The principles line, FR-010 and SC-008 now cite it, list the keyboard commands of this feature's modules (only the bar's `focus`) and the only popup (the example item's), and require that no surface of the feature holds keyboard focus until the focus command runs. No new behavior is added; the checks make the principle's quality gate testable.
- **Fix**: The owner's earlier note that Principle VI was not met is withdrawn: VI no longer states a keyboard rule (it defers to VIII), and the feature meets VIII.

### 2026-11-08 (frame corners)

- **Fix**: In the frame style there were gaps at the corners of the screen (reported with a screenshot of the frame on the bottom edge: the strip, the top panel and the side panels did not meet at the corners). FR-004, acceptance scenario 4a and an edge case now require one unbroken band at every corner, for every edge and for a border width or radius of 0.

### 2026-11-09 (frame gaps along the sides)

- **Fix**: Gaps were still left in the frame, now along the sides, next to the corner blocks (reported with two screenshots, frame on the bottom edge and on the left edge: a dark gap about 20 px long between a side panel and the corner block next to the strip). Cause: the border panels offset themselves by the strip's width with `margins`, and the compositor already keeps that space free, so the offset counted twice. Acceptance scenario 4b and the edge case on corners now require no gap along the sides, and forbid a panel adding its own offset for space the compositor keeps free.
