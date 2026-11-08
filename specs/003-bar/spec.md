# Feature Specification: Bar

**Feature Branch**: `003-bar`

**Created**: 2026-11-07

**Status**: Draft

**Input**: User description: "Feature III. Bar. The first end-user component of Lumen: a bar built as a regular module on the public module API from feature 002 (no private access). The bar is itself modular: it has three places for items (left, center, right) and is configured through the config JSON file, in the style of waybar. Spec 002 deliberately deferred module-to-module extension points to this feature, so the spec should address how other modules can contribute items to the bar as a compatible API change."

**Constitution principles touched**: III (the bar is a module on the public interface, and the interface gains extension points), VI (one design language and keyboard operation), II (every color comes from Theme), V (the bar's layout lives in the layered, override-only configuration), IV (a failing item degrades only itself, and missing hardware hides only its item), I (anything heavy stays in helper programs).

## Overview

Modules gave Lumen a way to define, load and isolate parts of the shell, but nothing a user can look at. The bar is the first part a user sees every day: a strip along a screen edge that shows the time, the sound level, the battery and similar status items.

The bar does not contain its items. It is a frame with three places, **left**, **center** and **right**, and each place holds an ordered list of items. Which items appear, in which place and in which order is decided by the user in the configuration file, as in waybar: three lists of item names. Every item is supplied by a module.

Two things ship together:

1. **The bar**, a bundled module that uses exactly the public module interface and nothing private, follows the active theme live, and lays out the items the user has listed.
2. **Extension points**, the module-to-module feature that Modules reserved but left out. A module can declare a place where other modules may add things (the bar declares one for its items), and another module can contribute an item to it. This is a compatible addition to the module interface: it raises the minor number, and modules built for the earlier version keep loading unchanged. The same interface version adds a **popup interface**: a way for an item to open a small surface next to itself. The interface is public and documented, but this feature ships no built-in popups.

The built-in items (clock and date, sound, battery) are separate bundled modules that reach the bar through the same contribution mechanism that third-party items use. That proves that the extension point is enough to build real parts, and a third-party item is as good as a built-in one.

## Clarifications

### Session 2026-11-07

- Q: What does "modular" mean for the bar? → A: The bar is a frame with three places (left, center, right). Everything shown in it is an item supplied by a module through an extension point. The bar module contains no items of its own.
- Q: Who decides which items are shown and where? → A: The user, through three ordered lists in the configuration file. A contribution is shown only if the user's lists (or the shipped default lists) name it. Installing or enabling a contributor does not by itself put anything in the bar.
- Q: Can a contributing module take over or restyle the bar? → A: No. The bar owns the layout. An item supplies a visual part; the bar decides where it goes and how much room it gets.
- Q: Are workspace and active-window items part of this feature? → A: No. They depend on compositor state, which is a separate later feature. The left place is empty by default, and the later feature adds those items as ordinary contributions.
- Q: How far do detail views (menus, popups) go? → A: This feature adds the public popup interface that lets an item open a surface next to itself, but ships no built-in popups. The interface is demonstrated by the example module and tested with fixtures. The calendar, volume and battery details are left to later features that use it.
- Q: Is the system tray part of this feature? → A: No. The tray is its own later feature, because it brings its own menus and protocol. It will be one more contributor to the bar.
- Q: Which compatibility rule applies to the interface change? → A: The existing one. Extension points raise the interface to 1.1. A module built for 1.0 loads on 1.1; a module that needs 1.1 is refused by a 1.0 shell with the required and available versions.
- Q: What can the user do directly on the built-in sound item? → A: Left click opens the audio mixer program (EasyEffects, or a PulseAudio-style mixer where EasyEffects is not available), scrolling changes the volume, and right click toggles mute. Each action has a keyboard equivalent.
- Q: When a screen is too narrow for all the listed items, which are dropped first? → A: Items are dropped from the end of the right list first, then from the end of the center list, then from the end of the left list. Nothing overlaps, and the user can see which items were dropped.
- Q: How does the user name a screen in the bar's screens setting? → A: By the output name the compositor reports (for example `DP-1`), in a list, or by the single word `all`, which is the default.
- Q: What does an item name in the lists look like? → A: Two forms, both always valid. The short form is the module id and means that module's only or first-declared contribution to the bar. The long form is `module-id/item-id` and names any contribution exactly.
- Q: What happens to a list entry that is not a string? → A: It is dropped, an error naming the place and the entry is written to the log, and one desktop notification is sent per change. The other entries keep working. A value that is not a list at all falls back as a whole.
- Q: How does a host refuse a contribution that goes beyond its limits? → A: Through the module interface: the host can mark a contribution as refused, with a reason, and the module list shows it as rejected.
- Q: When the user names an item in a list but the module that supplies it is installed and not enabled, should naming it also enable the module? → A: No. Enabling stays an explicit, separate step. The item is not shown, and a warning names it, says its module is not enabled, and says how to enable it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See a working bar on every screen (Priority: P1)

A user starts Lumen and sees a bar along the edge of each screen, filled according to the default lists: the time and date, the sound level, and the battery level (only on a machine that has a battery). Windows are not covered by it. The user adds or removes a screen and the bar follows.

**Why this priority**: This is the first component a user can see and use. Without it, Lumen is still a developer tool.

**Independent Test**: Start Lumen with the default configuration on a machine with two screens. Check that a bar appears on both within a second of start, that a maximized window stops at the bar's edge, that the clock is correct, and that unplugging one screen removes only its bar.

**Acceptance Scenarios**:

1. **Given** the default configuration, **When** Lumen starts, **Then** one bar is shown on every screen along the top edge, with the default items in their default places.
2. **Given** a running bar, **When** a window is maximized, **Then** it fills the screen except for the space the bar reserves.
3. **Given** a machine without a battery, **When** the bar starts, **Then** the battery item is not shown and nothing is reported as an error.
4. **Given** a second screen is connected or disconnected, **When** the change happens, **Then** a bar appears on it or is released from it, without a restart and without affecting the other bar.
5. **Given** the sound server is not running, **When** the bar starts, **Then** the sound item is hidden or marked unavailable, and every other item works.

---

### User Story 2 - Arrange the bar in the config file (Priority: P1)

A user wants the bar at the bottom, wants the clock in the center, the sound item on the left, no battery item, and the bar only on the main screen. They write three ordered lists and a few values in the configuration file (the same single file as everything else), and the bar updates live. Which items exist is up to the modules installed; the user lists the ones they want, in the order they want.

**Why this priority**: Arranging the bar through the config file is the core of how the bar is used, and the three lists are what make the bar modular.

**Independent Test**: Edit the bar's lists in the user file to move an item between places, reorder items within one place, remove an item and restrict the screens; check each takes effect live. Name an item that does not exist, then enter an invalid value, then reset everything and confirm the file holds no override.

**Acceptance Scenarios**:

1. **Given** the bar is at the top, **When** the user sets the edge to the bottom, **Then** the bar moves within one second and the reserved space moves with it.
2. **Given** an item in the left list, **When** the user moves its name to the center list, **Then** it moves within one second, and the order inside each list is the order of the names.
3. **Given** an item named in a list, **When** the user removes the name, **Then** the item disappears within one second, even though its module stays enabled and loaded.
4. **Given** a name in a list that matches no installed item, **When** the file is saved, **Then** the bar skips it, shows no gap, and a warning names the entry and the place.
5. **Given** the same item named twice, **When** the file is saved, **Then** it is shown once, at its first position, and a warning names the duplicate.
6. **Given** an invalid setting, **When** the file is saved, **Then** an error names the module and key, and only that key falls back to its default.
7. **Given** several overrides, **When** the user resets them, **Then** the overrides are removed from the file and the bar returns to the default lists.

---

### User Story 3 - Another module adds an item to the bar (Priority: P1)

A module author wants a small status item (for example the output of a script, a weather summary, or a VPN indicator) to be available for the bar. They copy an example module, declare in its manifest that it contributes an item to the bar, and enable it. The user then adds the item's name to one of the bar's lists. The item appears in the bar, takes the theme, follows its own settings, and disappears when the name is removed or the module is disabled.

**Why this priority**: This is the feature that turns the bar from a fixed strip into something extensible, and it is the compatible interface change that Modules deferred.

**Independent Test**: Copy the example contributing module into the user modules directory, check that it is listed with its permissions and the item it offers but is not loaded, enable it, add its item name to the right list, and confirm that its item appears in the bar within a second. Disable it and confirm that the item is gone and nothing is left behind.

**Acceptance Scenarios**:

1. **Given** the bar is loaded and a contributing module is installed but not enabled, **When** the user lists modules, **Then** the contributor is shown as disabled, with its permissions and the items it offers and the extension point each targets.
2. **Given** the module is enabled but its item is not named in any list, **When** the bar is shown, **Then** the item is not shown, and the module list says that it is available but not placed.
3. **Given** the item is named in a list, **When** the module is enabled, **Then** its item appears in that place within one second, using the theme and tokens.
4. **Given** a shown item, **When** its module is disabled or reloaded, **Then** the item is removed (or replaced by the reloaded one) and everything it created is released.
5. **Given** a contributor targets a module that is not installed or not enabled, **When** the contributor is enabled, **Then** it loads without error, its item is not shown, and the module list says which module it is waiting for.
6. **Given** the bar is disabled by the user, **When** contributors are loaded, **Then** they stay loaded and idle, show "waiting" in the list, and their items come back when the bar is enabled again.
7. **Given** an item named in a list whose module is installed but not enabled, **When** the bar is shown, **Then** the module stays disabled, the item is not shown, and a warning says the module is not enabled and how to enable it; **When** the user then enables the module, **Then** the item appears in its place.
8. **Given** a module built for interface 1.0 (including all modules from Modules), **When** the shell runs on 1.1, **Then** it loads and behaves exactly as before.

---

### User Story 4 - The bar looks and behaves like Lumen (Priority: P1)

The bar and its built-in items take all colors from the active theme and all sizes, radii, fonts and animation timings from the design tokens. Switching the theme changes the bar at once. Everything interactive can be used from the keyboard.

**Why this priority**: The constitution makes live theming and one design language non-negotiable. A bar that ignores them would break the promise of the whole shell.

**Independent Test**: Switch between two presets and a wallpaper-derived palette while the bar is visible and check the colors. Then focus the bar with the keyboard command and reach and activate every interactive item without a pointer.

**Acceptance Scenarios**:

1. **Given** a visible bar, **When** the user switches the theme, **Then** every bar color follows within one second, without restart or flicker.
2. **Given** the fallback palette is active, **When** the bar is shown, **Then** text and icons keep the contrast the palette guarantees.
3. **Given** the bar is visible, **When** the user runs the documented keyboard command, **Then** keyboard focus moves into the bar, the arrow keys move between items in the order of the lists, Enter activates the focused item, and Escape returns focus to where it was.
4. **Given** a theme change at the moment items are being added, **When** the change finishes, **Then** no item keeps an old color.

---

### User Story 5 - A broken item does not break the bar (Priority: P2)

A contributed item has a syntax error, throws runtime errors, runs a failing or hanging helper, or crashes while loading. The bar and every other item keep working, and the user can see which item failed and why.

**Why this priority**: The bar is always on screen. If a third-party item could take it down, nobody could safely install one.

**Independent Test**: Load a set of deliberately broken contributing modules, named in the bar's lists, next to the built-in items, and check that the bar and all good items keep working while each failure is reported by module id.

**Acceptance Scenarios**:

1. **Given** a contributor whose item has a visual error, **When** it loads, **Then** its item is not shown, the module is listed as failed with the reason, and other items are unaffected.
2. **Given** a contributed item that logs three runtime errors within five seconds of loading, **When** the third error is logged, **Then** the contributor is unloaded and marked failed, and its space in the bar is released.
3. **Given** a failed contributor, **When** the user reloads it after fixing it, **Then** its item appears again.
4. **Given** the bar module itself fails to load, **When** the shell runs, **Then** the shell and all other modules stay running, the bar is listed as failed with the reason, and the contributors wait.

---

### User Story 6 - An item can open a popup (Priority: P3)

A module author wants their bar item to show more when it is clicked or activated from the keyboard, for example a calendar under a clock or a list under a status indicator. They use the public popup interface to open a small surface next to the item, styled with the theme, and close it again. The built-in items do not use it yet; later features (tray menus, calendar, volume and battery details) will.

**Why this priority**: The interface has to exist before later features can build detail views on it, but the bar is complete and useful without any popup.

**Independent Test**: Enable the example module that opens a popup from its item, activate the item with the pointer and with the keyboard, and check that the popup appears next to the item, follows the theme, can be closed with Escape and by clicking elsewhere, and is gone when the item or its module is removed.

**Acceptance Scenarios**:

1. **Given** an item that opens a popup, **When** the user activates the item, **Then** the popup appears next to it, on the same screen, without covering the bar's reserved space or leaving the screen.
2. **Given** an open popup, **When** the user presses Escape or clicks outside it, **Then** it closes and keyboard focus returns to the item.
3. **Given** an open popup, **When** its module is disabled, reloaded or fails, **Then** the popup is closed and everything it created is released.
4. **Given** a theme change while a popup is open, **When** the change finishes, **Then** the popup follows the new colors.
5. **Given** a popup that is open, **When** another item opens its popup, **Then** only one popup stays open.

---

### Edge Cases

- The bar has more items than fit in the width: items are never overlapped. Items are dropped from the end of the right list first, then from the end of the center list, then from the end of the left list, one at a time, until the rest fit. The user can see which items were dropped (a warning in the log and in the bar's state names them), and an item listed earlier in its list is dropped later. When the screen widens, dropped items come back.
- The short and the long form of the same item are both named (for example `bar-clock` and `bar-clock/main`): they are the same item, so the second is a duplicate and is skipped with a warning.
- A list entry has a name with no match, is a duplicate, or is a string that is not a valid item name: it is skipped with a warning that names the entry and the place, and the rest of the lists still work.
- A list entry is not a string (for example a number): it is dropped, an error naming the place and the entry is written to the log, one desktop notification is sent for the change, and the rest of the lists still work. If no notification service is available, only the log entry is written.
- A list setting is not a list at all, or holds more entries than allowed: the whole setting is invalid, an error names the module and the key, and the key falls back to its default.
- The lists are empty: the bar is shown empty and no error is reported.
- An item is named whose module is installed but disabled or not yet enabled: the module is not enabled by the bar's lists, the item is skipped with a warning that says how to enable the module, and it appears as soon as the module is enabled.
- A contributor's item is very wide or very tall: the bar limits it to the room it allots and does not let it resize the bar.
- The screens list names an output that is not connected (or an empty list): no bar is shown for the unmatched name, no error is reported, and a bar appears as soon as a named output is connected. An empty list shows no bar at all.
- Items named for a screen that has no bar (the bar is restricted to some screens): the items are not created for that screen.
- A contributor is enabled before the bar loads: it waits, and shows up when the bar loads and the item is named in a list.
- The bar is reloaded while contributors are loaded: contributed items are removed and re-created, and none is duplicated or leaked.
- A contributor targets an extension point that the host does not declare: the contribution is rejected with a clear message and the contributor still loads.
- A module declares two extension points or two contributions with the same id: the manifest is rejected with a message that names both.
- A module built for 1.0 uses the reserved fields `extensionPoints` or `contributions` and is now loaded on 1.1: the fields now take effect and the warning is removed. If the fields are not valid by the new rules, the module fails with a clear message instead of loading.
- The `requires` field is still reserved: it is accepted and ignored with a warning, as in 1.0.
- No audio mixer program is installed, or the configured one is missing or fails: the sound item still shows the level and still changes volume and mute, the left click reports "no mixer found" or the failure in the log, and nothing else is affected.
- The volume is already at 0% or 100% and the user keeps scrolling: it stays at the limit.
- The default output device changes while the bar is shown: the sound item follows within a second.
- The clock format setting is invalid: the default format is used and an error names the key.
- A popup would extend past the screen edge: it is moved to fit, and never covers the bar item that opened it.
- A popup's content is taller or wider than the popup interface allows: it is limited to the allowed size and does not resize itself beyond it.
- An item asks for a popup while it is not shown (for example it was removed from the list): the request is refused and logged.
- The bar module is disabled: reserved screen space is released at once.
- The system clock changes (time zone or manual change): the clock item follows within a second.
- The user modules directory holds an item with the same name as a built-in one: names include the module id, so the two never collide; the bundled module wins for a duplicate module id, as in Modules.
- The shell is launched outside niri or with a missing dependency: Foundation's behavior still applies.

## Requirements *(mandatory)*

### Functional Requirements

**The bar**

- **FR-001**: The bar MUST be a bundled module, enabled by default, that uses only the public module interface. A test MUST list the access it uses and fail if it uses anything the documented interface does not offer to third parties (as in Modules).
- **FR-002**: The bar MUST appear on every connected screen by default, along the top edge, and MUST follow screens being connected and disconnected without a restart.
- **FR-003**: The bar MUST reserve its space so that windows are not covered, and MUST release it when it is hidden, disabled, or unloaded.
- **FR-004**: The bar MUST have three places, left, center and right, each holding an ordered list of items. The bar MUST contain no items of its own: everything shown in it MUST be supplied by a module through the extension point.
- **FR-005**: The bar and every built-in item MUST take every color from Theme and every spacing, radius, font size and duration from Style, and MUST follow theme changes live.
- **FR-006**: Everything interactive in the bar MUST be operable from the keyboard. The user MUST be able to move focus into the bar with a documented, bindable command, move between items in list order, activate one, and return focus.
- **FR-007**: A missing service or absent hardware MUST hide or mark unavailable only the affected item and MUST NOT report an error for normal absence (such as no battery).

**Configuration of the bar**

- **FR-008**: The bar's configuration MUST live in the user's single configuration file under the bar module's namespace, with Foundation's rules (layered resolution, override-only writes, unknown keys preserved, validation, live reload, reset removes the override).
- **FR-009**: The configuration MUST contain three ordered lists of item names, one for each place (left, center, right). The order of the names MUST be the order of the items in the bar. The bar MUST show an item only if a list names it.
- **FR-010**: An item name MUST identify exactly one contribution and take one of two forms, both always valid: the long form `module-id/item-id`, which names any contribution of that module, and the short form `module-id`, which names that module's only or first-declared contribution to the bar. The slash MUST NOT be allowed in module ids or item ids. The `lumen modules show` command MUST print both forms for each item, and the documentation MUST state the forms, how to find the names of the available items, and that the long form is the one that stays stable if a module adds or reorders items.
- **FR-011**: The bar MUST ship default lists that give the default bar: the clock and date in the center, sound and battery on the right, and the left place empty. Workspace and active-window items are not part of this feature (they need compositor state) and will be added later as ordinary contributions.
- **FR-012**: A name in a list that matches no available item, a duplicate name, or a string that is not a valid item name MUST be skipped with a warning that names the entry and the place, and MUST NOT stop the other entries from working. A list entry of the wrong type (not a string) MUST be dropped, written to the log as an error that names the place and the entry, and reported to the user once per change with a desktop notification (when no notification service is available, the log entry alone is enough); the other entries MUST keep working. A value that is not a list at all is an invalid setting (FR-013).
- **FR-012a**: Naming an item in a list MUST NOT enable its module. If the module is installed but not enabled, the item MUST NOT be shown, and the warning MUST name the entry and the place, say that the module is not enabled, and say how to enable it. When the module is enabled later, the item MUST appear without any change to the lists.
- **FR-013**: The configuration MUST also contain at least: the edge (top or bottom), the screens the bar appears on, and the screen that receives keyboard focus from the focus command (by default the first bar). Screens MUST be named by the output name the compositor reports (for example `DP-1`) in a list; the entry `all` stands for every screen, and the default is a list holding only `all`. An output name that matches no connected screen MUST be ignored without error, and the bar MUST appear on that screen when it is connected later. A bad value MUST fall back to the default for that key only, with an error that names the module and the key.
- **FR-014**: Each item's own settings (for example the clock format) MUST live in its own module's namespace under the same rules, and MUST work the same way for built-in and third-party items.
- **FR-015**: Editing the lists or other bar settings MUST take effect live, without a restart, and without disturbing items that did not change.
- **FR-015a**: When the items do not fit the width of a screen, the bar MUST NOT overlap them. It MUST drop items from the end of the right list first, then the end of the center list, then the end of the left list, until the rest fit, MUST report which items were dropped, and MUST bring them back when room is available.

**Built-in items**

- **FR-016**: The following built-in items MUST ship, each as its own bundled module contributing through the public extension point: clock and date, sound output level with mute, and battery level with charging state. The battery item MUST show the low and the critical state from two level settings with documented defaults. The system tray is not part of this feature.
- **FR-016a**: The sound item MUST show the output level and the muted state, and MUST offer these direct actions: left click (or Enter when focused) opens the audio mixer program; scrolling (or the up and down arrow keys when focused) changes the volume by a step; right click (or the documented mute key, M, when focused) toggles mute. The volume step MUST be a setting with a documented default of 5%, and the volume MUST stay between 0% and 100%.
- **FR-016b**: The audio mixer program MUST be a setting. By default Lumen MUST use EasyEffects if it is installed, and otherwise a PulseAudio-style mixer if one is installed. If none is installed or the program fails to start, the item MUST keep working (level, volume, mute), the action MUST report that no mixer was found or that it failed, and nothing else MUST be affected. Starting the program MUST be declared as a permission of the sound module and shown to the user, and the program MUST run as a separate process outside the shell.
- **FR-017**: The clock and date format MUST be a setting with a documented default, and showing seconds MUST be a setting (it changes how often the clock updates); an invalid format falls back to the default with an error that names the key.
- **FR-018**: Built-in items MUST use no access to the bar that a third-party contributor does not have (checked by the test required in FR-001, which covers the bar and every built-in item).
- **FR-019**: Built-in items MUST NOT open popups in this feature. They are limited to what the bar item itself shows and to their direct actions (for the sound item: opening the mixer program, changing the volume and muting, as in FR-016a).

**Extension points and contributions (module interface 1.1)**

- **FR-020**: The module interface version MUST become 1.1. A module built for 1.0 MUST load and behave unchanged, and a module that needs 1.1 on a 1.0 shell MUST be refused with the required and available versions (FR-005 of Modules).
- **FR-021**: A module MUST be able to declare, in its manifest, extension points: an id that is unique within the module, a short description, and what a contribution to it provides. The bar MUST declare one for its items.
- **FR-022**: A module MUST be able to declare, in its manifest, contributions: the target module id, the target extension point id, an id that is unique within the module, a short display name, and the file with the contributed visual part. A contribution does not say where it appears; the user's lists do.
- **FR-023**: The host decides placement, size and visibility. A contribution MUST NOT be able to move, resize, restyle, or remove the host, other items, or other modules' contributions.
- **FR-024**: A contributed visual part MUST receive the same public interface object as any module (Theme, Style, Log, its own settings, the read-only module list), plus the facts of its placement that it needs, such as the bar's edge, thickness and the screen it is on. It MUST receive nothing private.
- **FR-025**: A contribution MUST be created only while the contributor and the host are both loaded and the host has chosen to show it. If either is unloaded, disabled, reloaded, or failed, or the host no longer shows it, the contribution MUST be removed, everything it created MUST be released, and it MUST come back when the conditions hold again.
- **FR-026**: A contribution to a host that is missing, disabled, or does not declare the extension point MUST NOT stop the contributor from loading. The module list MUST show the contribution as waiting or rejected, with the reason.
- **FR-027**: Contributing MUST require a declared permission that names the target module, MUST be shown to the user with the other permissions (including for modules that are not enabled), and MUST be enforced where Lumen mediates access.
- **FR-028**: A host MUST be able to set a limit on what it accepts (for example size and the number of contributions) and MUST refuse a contribution that exceeds it with a clear message, without affecting other contributions.
- **FR-029**: The read-only module list MUST include each module's declared extension points and contributions and the state of each contribution (shown, available but not placed, waiting, rejected, failed). The `lumen modules show` command MUST print them, with the item name to use in the lists.
- **FR-030**: The manifest schema MUST describe the new fields. The field `requires` MUST stay reserved (accepted, ignored, warned). The warning for `extensionPoints` and `contributions` MUST be removed.
- **FR-031**: The interface documentation MUST describe extension points, and the project MUST include an example module that contributes an item to the bar, which a new author can copy.

**Isolation and reliability**

- **FR-032**: A failure in any contributed item (visual error, runtime error, failing or hanging helper) MUST NOT stop the bar, other items, or the shell. The failure MUST be reported with the contributor's module id.
- **FR-033**: A failure of the bar module MUST NOT stop the shell or any other module, and contributors MUST wait without error until it is loaded again.
- **FR-034**: Items MUST follow Foundation's rules for helper programs: separate process, asynchronous, time-limited, failures reported by name, and only the calling module degraded.
- **FR-035**: The bar MUST NOT use the network or send telemetry. An item that needs the network MUST declare it as a permission and Lumen MUST show it.
- **FR-036**: Loading and unloading the bar or its contributors MUST NOT cause unbounded memory growth, and an idle bar MUST NOT use noticeable CPU.

**Popup interface (module interface 1.1)**

- **FR-037**: The module interface MUST let a bar item open and close one popup: a small surface placed next to the item, on the same screen, with the item's own visual part inside it. The popup interface is part of interface 1.1 and MUST be documented.
- **FR-038**: The host MUST decide the popup's placement and maximum size. A popup MUST stay on its screen, MUST NOT cover the item that opened it, and MUST NOT be able to take over the screen.
- **FR-039**: A popup MUST receive the same public interface object as any module and MUST take every color from Theme and every size and duration from Style, following theme changes live.
- **FR-040**: A popup MUST close on Escape and on activity outside it, and keyboard focus MUST return to the item that opened it. A popup MUST be operable from the keyboard.
- **FR-041**: At most one popup MUST be open at a time. Opening another MUST close the first.
- **FR-042**: A popup MUST be closed and everything it created MUST be released when its item is removed, hidden, or fails, or when its module or the bar is unloaded, disabled or reloaded.
- **FR-043**: A failure in a popup MUST be treated as a failure of its module (reported with the module id, with the same unload rule as other runtime errors) and MUST NOT stop the bar, other items, or the shell.
- **FR-044**: The example module MUST include an item that opens a popup, so that a new author can copy it.

### Key Entities

- **Bar**: The bundled module that shows a frame along a screen edge on each screen and lays out the items the user listed.
- **Bar Place**: One of left, center, right. Holds an ordered list of item names.
- **Item List**: The configuration value for one place: an ordered list of item names.
- **Item Name**: The identifier of one contribution used in the lists: the long form `module-id/item-id`, or the short form `module-id` for the module's only or first-declared contribution to the bar.
- **Bar Item**: One visible thing in the bar, supplied as a contribution by a module.
- **Built-in Item**: A bar item supplied by a bundled module (clock and date, sound, battery).
- **Popup**: A small surface that an item opens next to itself, placed and limited by the host, and closed by Escape or outside activity.
- **Extension Point**: A named place that a module declares and that other modules may contribute to.
- **Contribution**: A declaration, in a contributor's manifest, that it supplies a visual part for a named extension point of another module.
- **Contributor**: A module that declares at least one contribution.
- **Host**: The module that declares the extension point and decides which contributions are shown and where.
- **Contribution State**: Shown, available but not placed (not named in a list), waiting (host missing, disabled or not loaded), rejected (not accepted by the host, with a reason), or failed.
- **Bar Settings**: The values the bar declares (the three lists, the edge, the screens), stored under its namespace.
- **Module Interface Version 1.1**: Interface 1.0 plus extension points and contributions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With the default configuration, the bar is visible on every screen within 1 second of the shell starting, and a maximized window never overlaps it.
- **SC-002**: A theme change reaches 100% of the colors in the bar and its items within 1 second, without restart or flicker, across at least two presets, a light and a dark wallpaper-derived palette, and the fallback palette.
- **SC-003**: A new author, using only the documentation and the example module, gets their own item into the bar, named in a list, in under 10 minutes.
- **SC-004**: A change to the lists (adding, removing, moving or reordering an item) shows in the bar within 1 second, and a valid contributor's item appears within 1 second of enabling, reloading or rescanning its module, and disappears within 1 second of disabling it.
- **SC-005**: In a test set of at least 6 kinds of broken contributors (bad manifest, target missing, visual error, runtime error, failing helper, hanging helper), 100% leave the bar, every good item and the shell running, and each failure is reported with the module id and a clear reason.
- **SC-006**: In a test set of at least 4 kinds of bad list entries (unknown name, duplicate, wrong type, name of an item that failed), 100% are skipped or dropped with a log entry that names the place and the entry, a wrong-type entry also causes one notification per change, and all other entries still work.
- **SC-007**: Every module built for interface 1.0 that loaded before still loads and behaves the same (checked with all the module fixtures from Modules).
- **SC-008**: Every interactive element in the bar and the built-in items can be reached and activated with the keyboard alone.
- **SC-009**: A saved valid bar setting takes effect in under 1 second. An invalid one always shows an error naming the module and key, and falls back for that key only. After any sequence of edits and resets, the user file holds no value the user did not explicitly set, and all unknown keys are still there.
- **SC-010**: Over 100 cycles of loading and unloading a contributor, and 100 cycles of reloading the bar, memory use does not keep rising (final use within 10% of use after the first cycle), and no item is duplicated or left behind.
- **SC-011**: With the bar and all built-in items shown on one screen, average CPU use over 5 minutes stays under 1%.
- **SC-012**: Connecting or disconnecting a screen adds or removes the bar for it within 1 second, and the other bars are not affected.
- **SC-013**: The built-in items and the bar use no access that the documented interface does not offer to a third-party module (checked by a test that lists the access they use).
- **SC-014**: A popup opened by the example module appears within 1 second of activating its item, never leaves the screen, closes on Escape and on outside activity 100% of the time, and over 100 open and close cycles memory use does not keep rising (final use within 10% of use after the first cycle).

## Assumptions

- The users are now people who use the desktop, not only developers. The bar is arranged by editing the configuration file, in the style of waybar; the graphical settings app is a later feature.
- Only the top and bottom edges are supported in this feature. A vertical bar changes how items lay out and is left to a later change.
- The bar does not hide itself automatically, does not float over windows, and does not show an overflow menu. Items that do not fit are dropped by the rule in FR-015a.
- Spacers, separators and groups of items are not part of this feature. A later change can add them as ordinary items, since an item is just a contribution.
- Each item can appear once per bar. With several screens every bar shows its own instance of the listed items, all following the same lists. Showing one item more than once in a bar, or different lists on different screens, is out of scope.
- A desktop notification, as used for wrong-type list entries, is sent to whatever notification service the session has (Lumen's own notification module is a later feature). The program that sends it is an optional dependency, listed in the README; without it only the log entry is written.
- A host can refuse a contribution that goes beyond the limits it documents. The bar accepts at most 32 items in total across its three lists and refuses the rest, in list order, with a visible reason.
- The bar reserves screen space through the compositor's standard mechanism for panels. niri remains the only supported compositor, and X11 stays out of scope.
- Moving keyboard focus into the bar is done with a controller command that the user can bind in the compositor, because a bar does not take focus on its own.
- The system tray, notifications, the launcher, the control center, on-screen display, the lockscreen and the settings app are separate later features. They will each be a module and may contribute to the bar through the extension point defined here, and may use the popup interface.
- Compositor state (workspaces, windows, outputs) is a separate later feature. Workspace and active-window items arrive with it, and the left place of the bar is empty by default until then.
- Popups have no built-in users in this feature. The popup interface is verified by the example module and by test fixtures only.
- EasyEffects and PulseAudio-style mixers are optional dependencies, listed in the README with what happens without them (FR-016b). Lumen never installs them.
- Sound and battery information comes from the system services that Quickshell already offers, which the constitution allows. Heavy work for other items stays in helper programs.
- The module interface stays at one major version. Extension points are an additive change, so no migration note is needed, and the interface documentation records the new minor version.
- Enabling a contributor does not place its item: the user must name it in a list. The same trust model as Modules applies: modules run with the shell's privileges and permissions are not a sandbox.
- The bar module has the id `bar`, and the built-in item modules use ids that start with `bar-` (for example `bar-clock`). The user can replace a built-in item by removing its name from a list and adding another.
