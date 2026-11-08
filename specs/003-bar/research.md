# Research: Bar

Facts about Quickshell 0.3.2 below come from reading the installed type descriptions (`/usr/lib/qt6/qml/Quickshell/**/*.qmltypes`) and the Modules code. Items marked **Spike** are not settled by reading and become the first tasks of the implementation; each has a fallback so the plan does not depend on the outcome.

## R1. How a module contributes to another

- **Decision**: A host module declares `extensionPoints` (id, description); a contributor declares `contributions` (`id`, `name`, `target: {module, point}`, `file`). The Modules singleton keeps a **contribution registry** next to the module registry. A host reads it through `api.extensions` and creates an item with `api.extensions.create(point, itemRef, parent, slot)`. Modules loads the contributor's `file` with `Qt.createComponent("file://<contributor dir>/<file>?r=<n>")`, creates it with `{api: <contributor's api object>, slot: <the host's slot object>}`, records the object under the contributor and the host, and returns it. The contributed file's root type is any `Item`; it declares `required property var api` and `property var slot`.
- **Rationale**: This reuses the load path that Modules already proved (R1 and R2 of Modules), keeps the registry in the single owner (FR-024 of Modules), and lets runtime error attribution keep working: the contributed file lives in the contributor's directory, so the existing longest-prefix rule attributes its errors to the contributor.
- **Refusing a contribution (FR-028)**: A host calls `api.extensions.reject(pointId, itemRef, reason)` to refuse a contribution that goes beyond the limits it documents. The registry marks that contribution `rejected` with the reason, `create` for it then returns null, and the module list and `lumen modules show` print the reason. The mark is cleared at the host's next `resolve`, so a refused item is considered again after every change. The bar's limit is 32 items in total across its three lists; the items beyond it, in list order, are refused with `bar item limit (32) reached`.
- **Cleanup**: Unloading or reloading a contributor destroys every object it contributed. Unloading or reloading the host destroys every object created for it. Both go through one function, so nothing is left behind (FR-025).
- **Alternatives**: Contributors push a component to the host (the host would then run code from another module's directory and the contributor could not be attributed or cleaned up). A global item registry without a target (breaks the idea of extension points).

## R2. Interface version 1.1 and the reserved fields

- **Decision**: `apiVersion` becomes `"1.1"`. The scan accepts manifests with `api` up to 1.1. `extensionPoints` and `contributions` take effect and no longer warn. `requires` stays reserved (accepted, ignored, warned). New optional top-level field: `commands`. New setting type: `stringlist`. New permission form: `contribute:<module-id>`.
- **Consequence for tests**: The `warn-reserved` fixture of Modules uses `contributions`, so it changes verdict on 1.1. Its test is updated to use `requires`, and a new fixture covers a 1.0 module with a now-valid `contributions` block.
- **Rationale**: Everything is optional and additive, so a 1.0 module loads unchanged (FR-020, SC-007) and no migration note is needed.

## R3. Contribution permission

- **Decision**: A contributor must declare `contribute:<host-id>` (for example `contribute:bar`). The scan checks that every contribution's target module appears in the declared permissions. At run time `api.extensions.create` re-checks it, because the host could be reloaded with a different identity. It is shown by `lumen modules list` and `show` with the other permissions, including for modules that are not enabled.
- **Rationale**: FR-027. The existing permission list is an enum; the form `contribute:<id>` is added as a pattern, the same way as a fixed name, and is enforced where Lumen mediates the access (the create call). Like all permissions, it is not a sandbox.

## R4. List settings: a `stringlist` type

- **Decision**: The setting rule `{type: "stringlist", default: [...], maxItems?: n}` holds an array of strings (each 1 to 80 characters, at most `maxItems`, default 64). The limit of 80 leaves room for the longest item name (`module-id/item-id`, 65 characters) and for output names. Both the `modules` program (checks the default and the rule; a default that breaks them makes the manifest invalid) and the `config` helper (checks values) learn it.
- **Per-entry rule (clarified 2026-11-07, FR-012)**: In a user value, an entry that is not a string, is empty, or is longer than 80 characters is **dropped on its own**: the other entries stay, in order, and the helper reports `{key: "modules.<id>.<key>[<index>]", problem, value, dropped: true}`. The whole key falls back to the module's default, with an error naming the key, only when the value is not an array or holds more than `maxItems` entries.
- **Reporting**: The bar reads the dropped-entry errors from `api.settings.errors`, writes one error log line per entry (`right[2]: 42 is not a string; entry dropped`, source `module:bar`) and sends one desktop notification per change (R19). Entries that are strings but do not match an item are skipped by `modules items` with warnings instead (R5), and do not cause a notification.
- **String limits**: Interface 1.0 has no length limit for `string` settings, and the clock's `format` and the mixer command need one (FR-017, R12, R13). `string` rules gain optional `minLength` and `maxLength`, validated by both programs. Like `stringlist`, this is optional and additive.
- **Alternatives**: Three separate settings per item (does not give ordering as a list), objects with `{name, place}` (waybar uses lists, and the user asked for the lists).

## R5. Resolving the lists in Go

- **Decision**: A new subcommand `modules items` takes the three lists and a table of the available contributions with their module states, and returns the resolved lists plus warnings. Rules: both name forms (R6), duplicates (including short and long form of the same item), unknown names, modules not enabled (FR-012a), and, as a safety check only, entries that are not strings (the `config` helper has already dropped those, R4). The QML side calls it through `Backend.run` whenever the lists, the module list or a module state change, and builds the items from the answer.
- **Rationale**: Principle I and testability. It is the whole of FR-009 to FR-012a in one pure function with a black-box test.
- **Alternatives**: Resolving in QML (not unit testable without a running shell, and it is the part most likely to grow rules).

## R6. Item names

- **Decision**: Long form `module-id/item-id`. Short form `module-id`, meaning that module's first-declared contribution targeting `bar/items`. Module ids already match `^[a-z][a-z0-9-]{1,31}$`; item ids use `^[a-z][a-z0-9-]{0,31}$`; neither can contain a slash. Both forms resolve to the same internal key `module-id/item-id`, so the duplicate rule works on that key.
- **Rationale**: Clarification 2. The built-in item modules are named so that the short form reads naturally: `bar-clock`, `bar-sound`, `bar-battery`.

## R7. Which modules a list can name (enabled state)

- **Decision**: The items subcommand receives each module's `enabled` and `state`. A named item whose module is `disabled` yields a warning that says how to enable it, and is skipped. When the module is enabled later, the registry changes, the QML side asks for the resolution again, and the item appears without any change to the lists (FR-012a). A named item whose module is `failed` is skipped with the module's error. Naming never writes to the config.
- **Rationale**: Clarification 1 (trust model of Modules).

## R8. One panel per screen, and screen names

- **Decision**: The bar creates its windows with a `Variants` whose model is `Quickshell.screens` filtered by the `screens` setting. Each window is a panel (layer-shell) anchored to the left, right and chosen edge, with exclusion mode `Auto` so the compositor reserves the space. The compositor's output name is `ShellScreen.name`. The `screens` setting is a `stringlist` with default `["all"]`; the entry `all` matches every screen, any other entry matches an output name exactly, and an empty list shows no bar. Names that match no connected screen are ignored without error, and a screen connected later gets its bar because the model follows `Quickshell.screens`.
- **Deviation noted**: The spec says "a list, or the single word `all`". The plan implements the single word as a one-entry list (`["all"]`), so one setting type covers both. The documentation says so.
- **Spike (R8a)**: Check on niri that `exclusionMode: Auto` with three anchors reserves exactly the bar's height and that windows are not covered; check a bottom-edge bar the same way. **Fallback**: set `exclusiveZone` explicitly to the bar's height.

## R9. Popups and closing on outside activity

- **Decision**: The bar owns a single popup host. `slot.popup.open(component, options)` creates the content inside a popup window anchored to the item's rectangle on the same screen; the host limits it to a maximum size, moves it to fit the screen, and never lets it cover the item. `close()` and `isOpen` complete the interface. The host closes the popup on Escape, on opening another popup, and when the item, its module or the bar goes away.
- **Spike (R9a)**: Verify how outside clicks reach the compositor on niri for Quickshell's popup window type (an xdg popup with a grab, or not). **Fallback**: a transparent full-screen layer-shell surface that sits below the popup and above windows, which closes the popup when clicked; it exists only while a popup is open.
- **Rationale**: Clarification (popup interface in, no built-in popups). One popup at a time is FR-041.

## R10. Overflow: the fitting function

- **Decision**: `modules/bar/layout.js` exports `fit(widths, available, gap)`: `widths` are the measured widths of the items in the left, center and right lists, in order. It returns, per list, the number of items that stay, dropping from the end of the right list first, then the center, then the left, until the total plus gaps fits. The center list is placed in the middle of the screen when it fits there, otherwise between the left and right lists. The bar logs which items were dropped (once per change) and sets a property the module list can show, and items come back when width allows (FR-015a).
- **Test**: `qmltestrunner` (installed with Qt 6) runs `scripts/tests/bar_layout_test.qml` against `layout.js`: all fit, one dropped from the right, right exhausted then center, all three, widening brings items back, zero available, empty lists.
- **Rationale**: A pure function is testable without a shell. The width measurements must come from the live items, so the function is the only logic that has to stay in QML (see the Constitution Check).

## R11. Battery

- **Decision**: `bar-battery` uses the UPower service's display device. It is shown only when `isPresent` and `isLaptopBattery` are true. It shows the percentage and the charging state, with colors from the theme roles for low and critical levels (below a documented threshold setting).
- **Spike (R11a)**: The type description gives `percentage` as a double without saying whether it is 0 to 1 or 0 to 100. Read it on the real device (and on a machine without a battery) before writing the item. **Fallback**: normalize in the item (`> 1` means a percentage).
- **Degradation**: No UPower service or no battery means the item has nothing to show, so it reports itself as not shown without any error (FR-007).

## R12. Sound

- **Decision**: `bar-sound` binds the Pipewire service's default audio sink (kept alive with the service's object tracker) and shows the volume and the muted state. Scroll or Up/Down changes the volume by the `step` setting (default 5, 1 to 25, clamped to 0 to 100); right click or `M` toggles mute; left click or Enter runs the helper `open-mixer`. With no default sink the item is hidden (FR-007).
- **Mixer helper**: `modules/bar-sound/bin/open-mixer` is a POSIX `sh` script. Input: `{"command": "<setting or empty>"}`. With a setting it runs that program; otherwise it takes the first of `easyeffects`, `pavucontrol` that is installed. It starts the program detached (`setsid`, output discarded), answers `{"ok": true, ...}` at once, or `{"ok": false, "error": {"code": "no_mixer", ...}}` when nothing is installed. The module declares the `helpers` permission, which satisfies "starting the program is a declared permission" (FR-016b).
- **Setting**: `mixer` (string, default empty meaning automatic), `step` (integer).
- **Rationale**: Keeps the item pure presentation plus a service binding (Principle I); the shell helper is the same mechanism the example module already uses.

## R13. Clock

- **Decision**: `bar-clock` uses Quickshell's system clock service and formats with Qt's date-time formatting. Settings: `format` (string, 1 to 64 characters, default `ddd d MMM  HH:mm`) and `seconds` (boolean, default false; it only changes how often the clock updates, and the format decides what is shown).
- **Validity of the format**: Qt accepts any string, so "invalid" is defined by the rule limits (empty or longer than 64 characters), which the setting machinery already turns into the error and fallback FR-017 asks for.
- **Rationale**: Time zone and manual clock changes follow because the service reads the system clock (edge case in the spec).

## R14. Keyboard focus and module commands

- **Decision**: A manifest may list `commands` (names). `lumen modules call <id> <command>` reaches the shell as the IPC function `moduleCall(id, command)`, which the Modules singleton forwards to the module's `api.commands.handler`. The module sets the handler when it loads. The bar handles `focus` by giving its panel on the active screen (the screen of the focused output is not known without compositor state, so the first bar is used and the setting `focusScreen` can name one) exclusive keyboard focus, and by returning to the previous state on Escape.
- **Item key convention** (contract `bar-slot.md`): while focused, the bar handles Left/Right (move between items in list order), Escape (leave) and forwards every other key to the focused item. An item that is interactive sets `property bool interactive: true` on its root and handles Enter and its own keys; items without it are skipped by the arrows.
- **Rationale**: Modules cannot call IPC (Modules contract), so the controller needs a way to reach a module. A generic command is a small, documented, permission-free hook that later features (launcher, session menu) will need anyway; the bar is its first user.
- **Alternatives**: A bar-specific IPC function (hard-codes one module into the shell IPC and does not scale to the next module).

- **Spike (R14a)**: Verify on niri that exclusive keyboard focus on a layer-shell panel hands focus back to the previously focused window when the panel gives it up (Escape), and that clicks on other windows during the focus mode behave sensibly. **Fallback**: grab focus as on-demand focus and move it with the compositor's own focus actions, and document the behavior in `docs/bar.md`.

## R15. Runtime errors and failed items

- **Decision**: No change to attribution. A contributed item's file path is inside the contributor's directory, so errors are counted against the contributor. Three early errors unload it, and unloading destroys its items, so the bar's place is released automatically (FR-032).
- **Load errors**: If the contributed component fails to load or create, the contributor is marked failed with the reason (same as a module's visual error), and the bar skips that item.

## R16. Defaults

- **Decision**: Defaults live in the bar's manifest: `edge: "top"`, `screens: ["all"]`, `left: []`, `center: ["bar-clock"]`, `right: ["bar-sound", "bar-battery"]`. The built-in item modules are bundled, so they load by default (Modules FR-001a). An item whose hardware is missing reports itself as not shown without error.

## R19. Notifications for dropped list entries

- **Decision**: The bar module ships a helper `modules/bar/bin/notify`, a POSIX `sh` script like the mixer launcher (R12). Input `{"summary": "...", "body": "..."}`; it runs `notify-send` (from libnotify) with those texts, detached, and answers `{"ok": true}` at once, or `{"ok": false, "error": {"code": "no_notifier", ...}}` when `notify-send` is not installed. The bar declares `helpers` and calls it with `api.run`; a `no_notifier` or other failure is logged once as a warning and nothing else changes.
- **Rule**: One notification per change of the lists that drops at least one wrong-type entry (not one per entry), with the module, place and entries in the body. The same entries are not announced again until the setting changes.
- **Rationale**: Lumen's own notification module is a later feature, and the freedesktop notification service that any daemon provides is the standard way to reach the user meanwhile. It needs no new Go code and no dependency on Lumen internals, and it degrades to the log line when absent (Principle IV).
- **Alternatives**: A notification surface inside the bar (needs popups and a notification model, out of scope). Calling D-Bus from QML (Quickshell has no generic client for it in the installed services).

## R17. Documentation and the example

- **Decision**: `docs/bar.md` (configuring the lists, finding item names with `lumen modules show`, the overflow rule, keyboard use) and updates to `docs/module-api.md` (interface 1.1, still no migration note) and `docs/writing-a-module.md`. `examples/hello-bar-item/` is a copyable item with a setting and a popup (FR-044, SC-003).

## Spike results (2026-11-07, niri 26.04, Quickshell 0.3.2, three outputs)

- **R8a (panel space): confirmed, `Auto` works.** A 40 px panel anchored left, right and top (and again bottom) with `exclusionMode: ExclusionMode.Auto` made niri shrink the tile height of a window on that output from 1050 to 1010 (measured through `niri msg --json windows`, `tile_size`), and the height went back to 1050 when the panel closed. **Pitfall**: assigning `exclusiveZone` in any form switches the mode to `Normal` (a first test that bound `exclusiveZone: 0` reserved nothing). The bar must never assign `exclusiveZone`. The explicit-zone fallback also works (`exclusiveZone: 40` reserved 40 px), so it stays available. Method: `niri msg layers` shows the panel in the Top layer, and the tile size measures the reservation.
- **R9a (outside clicks for popups): not verified here.** There is no way to inject pointer clicks in this headless run (no `ydotool`, `wlrctl` or `dotool`). **Decision**: use the fallback by default, the transparent full-screen layer surface below the popup that closes it on any click, because it depends only on ordinary pointer input to a layer-shell surface and not on popup-grab behavior. Verifying a real outside click is part of T066 (by hand).
- **R11a (battery scale): resolved from the documentation, and the no-battery case verified.** The Quickshell docs define `percentage` as "equivalent to `energy` / `energyCapacity`", so it is a **fraction from 0 to 1**; the item multiplies by 100 (clamped). On this machine (a desktop) `UPower.displayDevice` exists with `isPresent` false, `isLaptopBattery` false, `percentage` 0 and `state` 0, with no devices, so the "hide without error" rule is confirmed. A real charge level could not be read here; check it on a laptop during T073.
- **R14a (keyboard focus hand-back): inconclusive.** A panel with `keyboardFocus: Exclusive` was accepted by niri (`niri msg layers` showed `Keyboard interactivity: exclusive`), but a key injected with `wtype` never reached the panel's key handler and the item never reported active focus, in both exclusive and on-demand modes, so Escape and the hand-back to the previous window could not be tested without a human at the keyboard. `niri msg focused-window` kept reporting the terminal throughout. **Decision**: implement exclusive focus as planned, keep on-demand as the documented fallback, and make the keyboard check (focus enters, keys arrive, Escape returns focus to the previous window) an explicit by-hand item in T054.

## Measurements (2026-11-07/08, this machine: niri 26.04, 3 outputs, desktop without a battery)

- **Idle CPU (SC-011): 0.02% of one core** with the bar and the three built-in items on three outputs, over 2 minutes and again over the full 5 minutes (`smoke.sh --bar-cycles 100 --cpu-minutes 5`, 2026-11-08). Met.
- **Full run, 2026-11-08 (`--bar-cycles 100 --cpu-minutes 5`):** contributor 100 cycles 454,724 kB to 473,180 kB (+4.1%, pass); popups 524,528 kB to 524,676 kB (pass); bar reload 473,216 kB to 524,960 kB (+10.9%), down from 12.5%. **Decision (project owner, 2026-11-08):** the limit for the bar reload is 12%, since runs vary by about 1.5 points and 100 reloads in a row is far beyond real use. Met under that limit. The battery item's per-instance UPower cost (about 140 to 280 kB per reload) is known to remain and is accepted.
- **Flaky smoke checks:** in three full runs the checks "a runtime error is attributed to its module" and "items that do not fit the width are dropped and named" each failed once and passed on the next run (timing-dependent waits in `smoke.sh`, not yet investigated).
- **Popups (SC-014, memory part):** over 600 open and close cycles, memory stayed within 0.2% (482,700 kB after the first 100, 483,380 kB after 600).
- **Contributor load and unload (SC-010):** 100 cycles grew memory by about 3% (455,268 kB to 470,512 kB).
- **Bar reload (SC-010): NOT MET in the combined scenario.** 100 reloads of the bar grew memory from 472,532 kB to 531,888 kB (+12.5%, limit 10%). What I found by isolating it (each number is per reload, noisy by roughly 100 kB):
  - An empty bar (three windows, no items): about 42 kB. No windows at all: about 10 kB. So the windows themselves are cheap.
  - The sound item leaked about 570 kB per reload, from one `PwObjectTracker` per item instance. Moving the tracker to the module's `Main.qml` (one per module) cut it to about 20 kB. **Fixed.**
  - The clock leaked about 160 kB per reload, from one `SystemClock` per instance. A plain `Timer` brought it to about 80 kB. **Fixed in part.**
  - The battery item still costs about 140 to 280 kB per reload (UPower bindings per instance). A module-level singleton for the state showed no clear gain, so it was reverted. **Open.**
  - A plain item (the example) costs about 38 kB per reload for three instances.
  - The default bar now measures roughly 340 to 480 kB per reload, which over 100 reloads is just above 10% of the shell's size. This is the open item behind SC-010 for the bar reload, and `smoke.sh --bar-cycles 100` still reports it as a failure.
- **Not measured:** SC-001, SC-004 and SC-012 timings (start to bar visible, list change to screen, screen unplug and replug). These need a person at a real session.

## R18. Tests of the interface rule for bundled modules

- **Decision**: `module_api_usage_test.go` from Modules lists the `api.` members used by the test surface and checks each is documented. It is extended to every bundled module (`bar` and the three items), and the `slot.` members used by items are checked against `bar-slot.md` the same way (SC-013). The "no hardcoded colors" and "no magic values" tests already scan every `*.qml` outside fixtures.
