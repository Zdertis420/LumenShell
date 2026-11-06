# Feature Specification: Modules

**Feature Branch**: `002-modules`

**Created**: 2026-11-06

**Status**: Draft

**Input**: User description: "Feature II. Modules."

**Constitution principles touched**: III (modular components and open extensibility), I (heavy logic stays in helper programs), IV (a failing module degrades only itself), V (module settings use the same layered, override-only configuration), VI (modules share one design language through Style and Theme), II (modules take every color from Theme).

## Overview

Foundation gave Lumen its project layout and five core singletons. Modules turns that into something other parts can be built on. A *module* is a self-contained piece of the shell (a bar, a launcher, a widget, a panel) that Lumen discovers, validates, loads, and unloads on its own. Every built-in part of Lumen will be a module that uses exactly the same public interface that a third-party author gets, so anyone can add or replace a part without changing Lumen's core.

Modules ships the module interface, the registry that owns it, a way to list and control modules from the terminal, and one example module. It ships no end-user component of its own: Foundation's test surface becomes the first built-in module, which proves that built-in code needs nothing private. Its users are Lumen's developers and module authors.

## Clarifications

### Session 2026-11-06

- Q: Does a module run in its own process? → A: No. A module is shell-side presentation code plus, optionally, helper programs that it runs through Backend with time limits. There is no per-module resident process, which keeps Foundation's rule that the shell is the only long-running process. Heavy logic goes into the module's helper programs (Principle I).
- Q: Are permissions enforced? → A: Permissions are declared in the manifest, shown to the user, and enforced wherever Lumen mediates access (running helper programs, reading and writing configuration, reading other modules' data). Modules run inside the shell and have the shell's privileges, so a permission cannot stop a module from using system facilities directly; Lumen documents this trust model plainly instead of claiming a sandbox.
- Q: Where do module settings live? → A: In the same user file as all other configuration, under a namespace owned by the module id, with the same layered and override-only rules.
- Q: Does a module in the user modules directory start automatically? → A: No. Modules from the user directory are discovered, validated, and listed with their declared permissions, but stay unloaded until the user enables them. Bundled modules load automatically. The same on/off choice can be switched from the terminal for any module.
- Q: Which older modules keep working when the module interface changes? → A: The interface version has a major and a minor number. Adding features raises the minor number, and older modules keep loading. A removal or change of meaning raises the major number, and modules built for another major version are refused with a clear message. A module that needs a newer minor number than Lumen provides is also refused.
- Q: Should one module be able to add items to another module (extension points, such as a widget inside the bar or a search source in the launcher) in this feature? → A: No. Modules are independent in this feature. Preparation only: the manifest reserves names for extension points, contributions, and required modules (accepted but ignored, with a visible warning), and the read-only module list includes each module's manifest, so the feature that builds the bar can add this as a compatible change.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Write a module and see it load (Priority: P1)

A module author creates a directory with a manifest and a small visual part, puts it in the user modules directory, and Lumen discovers it and lists it. After the author enables it, Lumen loads it. The author follows the documented example to get there, without touching Lumen's own files.

**Why this priority**: Nothing else in Lumen can be built without a working way to define and load a module. A shell that can only load its own parts is not extensible.

**Independent Test**: Copy the example module into the user modules directory, start the shell, and confirm that the module appears in the module list as not enabled, with its permissions shown. Enable it and confirm that it is listed as loaded and that its visual part is on screen.

**Acceptance Scenarios**:

1. **Given** a valid module in the user modules directory, **When** the shell starts, **Then** the module is listed as not enabled with its declared permissions and does not run. **When** the user enables it, **Then** it is listed as loaded and its visual part appears, without any change to Lumen's own files.
2. **Given** a bundled (built-in) module and a user module with different ids, **When** the shell starts, **Then** the bundled one loads, the user one is listed as not enabled, and the list shows where each one comes from.
3. **Given** two modules with the same id, **When** the shell starts, **Then** the built-in one loads, the other is not loaded, and the list shows a clear error naming the id and both locations.
4. **Given** a manifest with a missing or malformed field, **When** the shell starts, **Then** that module is not loaded, and the error names the module, the field, and the problem.
5. **Given** a module that requires a newer module interface version than Lumen provides, **When** the shell starts, **Then** it is not loaded, and the error states the required and the available version.

---

### User Story 2 - Use the shared services through the public interface (Priority: P1)

A module takes its colors from Theme, its spacing, radii, typography and motion from Style, writes diagnostics through Log, reads its own settings, and runs its own helper programs through Backend. It reaches all of these only through the documented module interface. Lumen's own test surface is rewritten as a built-in module that does the same.

**Why this priority**: The constitution requires that built-in parts use the same interface as third parties. Without this, modules cannot look or behave like part of Lumen.

**Independent Test**: Load the example module (after enabling it), switch themes, and confirm that its colors follow live; check that the built-in test-surface module uses no access that the example module lacks.

**Acceptance Scenarios**:

1. **Given** a loaded module, **When** the theme changes, **Then** the module's colors update live with the rest of the shell and without restart.
2. **Given** a loaded module, **When** it writes a log message, **Then** the entry appears in the shared log stream with the module's id as its source.
3. **Given** a module that declares a helper program, **When** it runs the helper, **Then** the run is asynchronous, has a time limit, and follows the same failure rules as Foundation's helpers.
4. **Given** the built-in test-surface module, **When** its access is compared with the documented interface, **Then** it uses nothing beyond what a third-party module can use.
5. **Given** a module that tries to use a service it did not declare a permission for (for example running a helper it did not declare), **When** it makes the request, **Then** Lumen refuses it, logs the refusal with the module id, and the module keeps running.

---

### User Story 3 - Module settings in the user's config file (Priority: P2)

A module declares the settings it supports, with types, limits, and defaults. The user changes them by editing the same JSON file as every other setting, and the module reacts live. Invalid values produce a clear error and fall back to the module's defaults.

**Why this priority**: Modules without configuration are of limited use, and the constitution requires that module-declared schemas are validated like the rest of the configuration.

**Independent Test**: With the example module enabled, edit its setting in the user file, confirm the live update, enter an invalid value, then reset it and confirm the override is removed.

**Acceptance Scenarios**:

1. **Given** a module that declares a setting, **When** the user sets a valid value in the user file, **Then** the module updates live and the file holds only that override.
2. **Given** an invalid value for a module setting, **When** the file is saved, **Then** a clear error names the module and the key, only that key falls back to the module's default, and other modules are unaffected.
3. **Given** a reset of a module setting, **When** it is applied, **Then** the override is removed from the file.
4. **Given** settings for a module that is not installed, **When** Lumen rewrites the file, **Then** those settings are preserved as unknown keys.
5. **Given** a module that declares a setting whose name would collide with a Lumen core setting or another module's setting, **When** it is loaded, **Then** the collision is impossible because settings are namespaced by module id.

---

### User Story 4 - A broken module only breaks itself (Priority: P2)

A module fails: its manifest is bad, its visual part has an error, it throws while running, or its helper program fails or hangs. Lumen reports the failure clearly and carries on with every other module.

**Why this priority**: A shell is always running, and a third-party module must never be able to take the desktop down.

**Independent Test**: Load a set of deliberately broken modules next to a good one, and confirm the good module and the shell stay responsive while each failure is reported with the module's id.

**Acceptance Scenarios**:

1. **Given** a module whose visual part has an error, **When** the shell loads it, **Then** the module is marked failed with the error text, and all other modules load normally.
2. **Given** a running module that raises an error while handling an event, **When** the error happens, **Then** it is logged with the module id, and the shell and other modules keep running.
3. **Given** a module whose helper program fails or hangs, **When** the limit is reached, **Then** the helper is stopped, the failure names the module and the helper, and only that module's feature degrades.
4. **Given** a failed module, **When** the user fixes it and asks Lumen to reload it, **Then** it loads again without a shell restart.
5. **Given** a module that logs three runtime errors within 5 seconds of loading, **When** the third error is seen, **Then** Lumen unloads it, marks it failed, does not load it again by itself, and says how to try again. A module that fails to load is likewise never retried automatically.

---

### User Story 5 - See and control modules from the terminal (Priority: P3)

A user or script runs `lumen` commands to list modules and their state, show one module's details, enable or disable a module, and reload one module or rescan the directories.

**Why this priority**: It makes modules observable and controllable, but the module system works without it.

**Independent Test**: Run each module command against a running shell with good and broken modules and check the output and the exit codes.

**Acceptance Scenarios**:

1. **Given** a running shell, **When** the user lists modules, **Then** each module's id, version, origin, state (loaded, disabled, failed), and last error are shown.
2. **Given** a loaded module, **When** the user disables it, **Then** it unloads live, the choice is kept across restarts, and it stays disabled until enabled again.
3. **Given** a module that is not enabled (a new user module, or one the user disabled), **When** the user enables it, **Then** it loads live and stays enabled after a restart. Its declared permissions are shown in the list and in the details before and after enabling.
4. **Given** a module id that does not exist, **When** the user runs a module command with it, **Then** the command exits with a non-zero status and a clear message, and the shell is unaffected.
5. **Given** a new module directory added while the shell runs, **When** the user asks for a rescan, **Then** the new module is discovered and loaded without a restart.
6. **Given** no running shell, **When** the user runs a module command, **Then** it exits with the same "not running" status as Foundation's commands.

---

### Edge Cases

- The user modules directory does not exist: it is not created automatically, and the module list simply shows only bundled modules.
- A module directory has no manifest, or the manifest is not valid JSON: the module is listed as failed with a clear message, and nothing else is affected.
- A module's visual part exists but the manifest names a missing file: the module fails with the file name in the error.
- A module is removed from disk while loaded: it keeps running until it is reloaded or the shell restarts, and a reload then reports it as gone.
- A module is unloaded: everything it created (visual parts, running helper runs, subscriptions) is released, and repeated load and unload does not grow memory.
- A module's settings change while it is unloaded: the new values apply when it loads again.
- A module declares a very long list of settings or a huge manifest: validation rejects manifests above a documented size.
- Two modules are loaded at the same time and both write logs or run helpers: their entries and runs stay attributed to the right module.
- The interface version changes: a module built for the same major number and a lower or equal minor number keeps loading, and a module for another major number, or for a higher minor number than Lumen has, fails with a clear message.
- A module blocks the display thread with endless work: Lumen cannot interrupt it, and the documentation says so (see Assumptions).
- A module id uses characters that are unsafe in paths or setting names: the manifest is rejected.
- A manifest uses one of the reserved fields (`extensionPoints`, `contributions`, `requires`): the module loads normally, the field does nothing, and a warning says so.
- The shell is launched outside niri or with a missing dependency: Foundation's behavior still applies.

## Requirements *(mandatory)*

### Functional Requirements

**Discovery and manifest**

- **FR-001**: Lumen MUST discover modules in two places: a bundled directory shipped with Lumen and a user modules directory in the user's own data location, and MUST show the origin of each module.
- **FR-002**: Each module MUST have a manifest with at least: a unique id, a version, the module interface version it requires, its declared permissions, and its settings schema (which may be empty).
- **FR-001a**: Bundled modules MUST be enabled by default. Modules from the user modules directory MUST be discovered, validated, and listed, but MUST NOT be loaded until the user enables them.
- **FR-003**: Lumen MUST validate each manifest against a published manifest schema, whether or not the module is enabled, and, for every problem, report the module id, the field, and the problem.
- **FR-004**: Module ids MUST be unique and limited to a documented safe character set. On a duplicate id, the bundled module MUST win and the other MUST be reported with both locations.
- **FR-005**: The module interface version MUST have a major and a minor number. A module MUST be loaded only if its required major number equals Lumen's and its required minor number is not higher than Lumen's. Otherwise it MUST be reported with the required and available versions.
- **FR-005a**: The manifest schema MUST reserve the optional fields `extensionPoints`, `contributions`, and `requires` for later use. In this feature a module that includes any of them MUST still load, the fields MUST have no effect, and the module list MUST show a warning that names the field and says it is not supported in this interface version.

**Public interface**

- **FR-006**: The module interface MUST give a module access to: Theme roles, Style tokens, Log, its own settings, the list of modules with each one's manifest and state (read-only), and running its own declared helper programs through Backend. A module MUST NOT need any other private access.
- **FR-007**: Every built-in part of Lumen MUST be implemented as a module through this same interface. Foundation's test surface MUST be converted into the first built-in module.
- **FR-008**: Log entries written by a module MUST carry the module id as their source, in the shared stream.
- **FR-009**: Helper programs run by a module MUST follow Foundation's rules: separate process, asynchronous, time-limited, failures reported by name, and only the calling module degraded.
- **FR-010**: The module interface MUST be versioned and documented. Adding a capability MUST raise the minor number. A breaking change (a removal or a change of meaning) MUST raise the major number and come with a migration note.

**Permissions**

- **FR-011**: Manifests MUST declare permissions from a documented list. Lumen MUST show them to the user in the module list and details, including for modules that are not yet enabled, so the user can see them before the module runs.
- **FR-012**: Lumen MUST enforce permissions wherever it mediates access (running helper programs, reading or writing configuration, reading other modules' data), MUST refuse and log a request that was not declared, and MUST keep the module running.
- **FR-013**: The documentation MUST state that modules run with the shell's own privileges and that permissions are not a sandbox.

**Settings**

- **FR-014**: A module's settings MUST be declared in its manifest schema, with types, limits, and defaults, and MUST be merged into the published configuration schema.
- **FR-015**: A module's settings MUST live in the user's single configuration file under a namespace owned by the module id, and MUST follow Foundation's rules: layered resolution, override-only writes, unknown keys preserved, validation with clear errors, live reload, and reset removing the override.
- **FR-016**: A bad module setting MUST fall back to the module's own default for that key only, with an error that names the module and the key.

**Isolation**

- **FR-017**: A failure in one module (bad manifest, load error, runtime error, helper failure or hang) MUST NOT stop the shell or any other module.
- **FR-018**: Every module MUST have a visible state (loaded, disabled, or failed) and, when failed, its last error. Errors MUST be logged with the module id.
- **FR-019**: A module that fails to load, or that logs three runtime errors within 5 seconds of loading, MUST be unloaded if loaded, MUST be reported as failed, and MUST NOT be loaded again automatically. The user MUST be able to reload it.

**Lifecycle and control**

- **FR-020**: Modules MUST be loadable, unloadable, and reloadable one at a time without restarting the shell and without affecting other modules.
- **FR-021**: Unloading a module MUST release everything it created, so that repeated load and unload does not grow memory without bound.
- **FR-022**: The `lumen` controller MUST provide module commands: list, show one module's details, enable, disable, reload one module, and rescan the directories. Each MUST use the existing documented, versioned command channel, with distinct exit statuses for "not running", "unknown module", and invalid arguments.
- **FR-023**: Enabling or disabling a module MUST persist across restarts. The choice MUST be saved as an override in the module's settings namespace only when it differs from the default (on for bundled modules, off for user modules), and MUST be removed from the file when it matches the default again. Reloading and rescanning MUST NOT write to the file.
- **FR-024**: A single owner (the Modules singleton) MUST hold the registry: the list of modules, their states, and their errors.

**Documentation, example, and reliability**

- **FR-025**: The project MUST include documentation of the module interface and at least one example module that a new author can copy.
- **FR-026**: The system MUST NOT use the network or send telemetry by default. A module that needs the network MUST declare it as a permission, and Lumen MUST show that permission.
- **FR-027**: Loading and unloading modules MUST NOT cause unbounded memory growth, and idle modules MUST NOT use noticeable CPU.

### Key Entities

- **Module**: A self-contained piece of the shell with an id, a version, a manifest, a visual part, optional helper programs, and a state.
- **Manifest**: The module's description: id, version, required interface version, permissions, settings schema, and the files it uses.
- **Manifest Schema**: The published definition of a valid manifest.
- **Module Interface**: The versioned, documented set of things a module can use (Theme, Style, Log, settings, helper runs, the read-only module list).
- **Module Interface Version**: A major and a minor number that say which interface a module was built for and which one Lumen provides.
- **Permission**: A declared capability of a module, shown to the user and enforced where Lumen mediates access.
- **Module Settings**: The values a module declares, stored under its namespace in the user file.
- **Module State**: Loaded, disabled (including "not yet enabled"), or failed, with the last error.
- **Module Origin**: Bundled or user directory.
- **Registry**: The single owner of the module list, states, and errors (the Modules singleton).
- **Example Module**: A small, documented module to copy.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new author, using only the documentation and the example module, gets their own module loaded and visible in under 10 minutes.
- **SC-002**: A valid enabled module appears (loaded and on screen) within 1 second of shell start, or within 1 second of enabling, reloading or rescanning it.
- **SC-003**: In a test set of at least 8 kinds of broken modules (bad manifest, missing file, version mismatch, duplicate id, visual error, runtime error, failing helper, hanging helper), 100% leave the shell and every good module running, and each failure is reported with the module id and a clear reason.
- **SC-004**: A theme change reaches 100% of a loaded module's colors within 1 second, without restart or flicker.
- **SC-005**: A saved valid module setting reaches the module in under 1 second. An invalid one always shows an error naming the module and key, and falls back for that key only.
- **SC-006**: After any sequence of module setting edits, enables, disables, and resets, the user file holds no value the user did not explicitly set, and all unknown keys are still there.
- **SC-007**: Over 100 load and unload cycles of a module, memory use does not keep rising (final use within 10% of use after the first cycle).
- **SC-008**: Every module command takes effect in under 1 second, and invalid ones or ones sent with no shell running exit with a clear message in under 1 second.
- **SC-009**: With 10 idle modules loaded, average CPU use over 5 minutes stays under 1%.
- **SC-010**: The built-in test-surface module uses no access that the documented interface does not offer to a third-party module (checked by a test that lists the access it uses).

## Assumptions

- The users are Lumen developers, module authors, and people who install modules, not casual end users. There is still no graphical module manager; that is left to the settings app.
- Modules are shell-side presentation code, plus optional helper programs run through Backend. Native plugins and separate resident processes per module are out of scope.
- Modules run inside the shell with the shell's privileges. Permissions are declared and enforced only where Lumen mediates access; Lumen does not claim a sandbox. A module that blocks the display thread or crashes the whole process through its own code cannot be contained; only errors that the shell can catch are isolated.
- Modules from the user directory stay off until the user enables them, because they run with the shell's privileges and a file can land in that directory without the user noticing. Lumen does not ask again if a module's declared permissions change after it was enabled; the details view shows the current ones.
- The user modules directory follows the standard per-user data location, and Lumen does not create it or write into it.
- The bundled module wins over a user module with the same id, so a user cannot silently replace a built-in part by accident. Replacing a built-in module deliberately (by disabling it and adding another) is supported through its own id.
- Directories are scanned at start and when the user asks for a rescan. Watching the directories for changes is not part of this feature.
- A module can be hot-reloaded from disk with the reload command, but live editing of a module's files is not watched.
- The first built-in module is Foundation's test surface. The bar, notifications, launcher, and other parts are separate later features that will each be a module.
- Compositor state (a Compositor singleton for workspaces, windows and outputs) is out of scope and is a separate later feature.
- Modules adding items to other modules (extension points and contributions), dependencies between modules, and any other module-to-module communication beyond reading the module list are out of scope. The reserved manifest fields (FR-005a) and the manifest-bearing module list (FR-006) are the only preparation for them.
- niri remains the only supported compositor, and X11 is out of scope.
