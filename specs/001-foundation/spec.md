# Feature Specification: Foundation

**Feature Branch**: `001-foundation`

**Created**: 2026-11-06

**Status**: Draft

**Input**: User description: "Feature I. Foundation." Clarified by the user: Foundation is the project structure and the core singletons.

**Constitution principles touched**: I (thin QML UI, heavy logic in Go), II (theming through one theme service), III (modular components), IV (graceful degradation), V (layered, override-only configuration), VI (shared design language).

## Overview

Foundation sets up the repository so that every later Lumen feature (bar, notifications, launcher, and so on) has a clear place to live. It also provides the small set of core singletons that all components share: one owner each for configuration, theme, design tokens, the shell's command interface and helper scripts, and logging. Foundation ships no end-user components. Its users are Lumen's developers and module authors, and its deliverable is a skeleton that starts, shows that the singletons work, and can be built on.

## Clarifications

### Session 2026-11-06

- Q: Which core singletons should Foundation define? → A: Exactly five: Config, Theme, Style, Backend, Log. A Compositor singleton and a Modules singleton are deferred to later features.
- Q: How is non-trivial logic split from the visual layer? → A: There is no standalone backend service. The only long-running process is the shell. A one-shot `lumen` command-line controller is the single interface to it (`lumen start` starts the shell, `lumen theme <theme>` sets a theme), and it reaches the running shell through Quickshell's own IPC. Small helper scripts for heavy tasks (parsing, filtering large data) are run from QML as processes. The "Backend" singleton is the shell-side owner of the IPC command handlers and helper runs.
- Q: Who starts and supervises the controller? → A: The user's session starts the shell with `lumen start` (for example from niri's startup entries or a systemd user unit) and supervises it. The controller is one-shot, so nothing supervises or reconnects to it. Lumen itself does not restart a crashed shell.
- Q: Does `lumen theme <theme>` persist the choice? → A: Yes by default: it writes the theme choice to the user's override file, and `--temp` applies it for the running shell only. Resetting the theme removes the override.
- Q: What happens when `lumen theme <theme>` runs with no shell running? → A: It exits with an error and saves nothing. Persisting is done by the running shell, so the controller never touches the user file.
- Q: What does the "selected theme or preset" configuration layer contribute? → A: The preset's palette and its mode. It sets no other keys in Foundation.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Clear project structure that builds and runs (Priority: P1)

A developer clones the repository and finds a documented layout that separates the visual layer, the controller and helper scripts, the shared singletons, modules, bundled data (presets, schemas), and docs. Following the README, they build the scripts that need building and start an empty shell that loads the singletons and shows a minimal themed test surface.

**Why this priority**: Every later feature needs a place to go and a working build and run loop.

**Independent Test**: On a fresh clone, follow only the README to build and launch, and confirm the test surface appears and the lint, format, and test commands run clean.

**Acceptance Scenarios**:

1. **Given** a fresh clone and the documented dependencies, **When** the developer follows the README, **Then** the scripts build and the shell starts with no errors.
2. **Given** the repository, **When** the developer looks for where a new component, helper script, preset, or doc belongs, **Then** the layout documentation names the location unambiguously.
3. **Given** the project, **When** the developer runs the documented format, lint, and test commands, **Then** they pass on the untouched skeleton.

---

### User Story 2 - Single-owner Config singleton (Priority: P1)

A component reads its settings from the Config singleton, which gives the result of layering defaults, the selected theme or preset, and the user's JSON overrides. When the user file changes, consumers update live. Invalid values produce a clear error and fall back to defaults.

**Why this priority**: Configuration underpins every other singleton and every later feature.

**Independent Test**: Read a value through the singleton, edit the user file, and confirm the live update. Introduce an invalid value, then remove the override, and confirm the fallback and the reset.

**Acceptance Scenarios**:

1. **Given** no user file, **When** the shell starts, **Then** defaults apply and no file filled with defaults is created.
2. **Given** a user file with one override, **When** it is saved, **Then** the consuming component updates live and the file still holds only that override.
3. **Given** an invalid value, **When** the file is saved, **Then** a clear error names the key and problem, and only that key falls back to its default.
4. **Given** a reset of a setting, **When** it is applied, **Then** the override is removed from the file.
5. **Given** unknown keys in the file, **When** Lumen rewrites it, **Then** the unknown keys are preserved.

---

### User Story 3 - Theme singleton with wallpaper and preset sources (Priority: P1)

A component reads semantic color roles from the Theme singleton and never hardcodes a color. The source is either a palette generated from the wallpaper or a bundled preset (Tokyo Night, Catppuccin flavors). Changing the source updates all consumers live. A fallback palette covers failure.

**Why this priority**: Theming is Lumen's defining feature, and every component depends on this singleton.

**Independent Test**: With the test surface running, switch between a light and a dark wallpaper-derived palette, two presets, and the fallback, and confirm live updates and contrast compliance.

**Acceptance Scenarios**:

1. **Given** the wallpaper source, **When** the wallpaper changes, **Then** consumers update live with no restart or flicker.
2. **Given** a preset source, **When** the user picks another preset, **Then** consumers update live.
3. **Given** the palette generator is missing or fails, **When** a palette is needed, **Then** the fallback palette is used and the reason is reported.
4. **Given** a palette below the accessible text contrast minimum, **When** it is applied, **Then** it is adjusted to comply or rejected.

---

### User Story 4 - Backend singleton for controller commands and helper runs (Priority: P2)

A user or script runs `lumen <command>` in a terminal, and the running shell reacts through the Backend singleton. Components also run helper scripts through the same singleton. All of this is asynchronous. If a command is invalid, the shell is not running, or a helper fails or hangs, the shell stays responsive, the caller gets a clear error, and only the affected feature degrades.

**Why this priority**: It is the boundary for Constitution Principle I, but the first components can start with Config and Theme.

**Independent Test**: Run `lumen start`, issue valid and invalid `lumen` commands from a terminal, and make a helper fail and hang. Confirm the shell stays responsive and the errors are clear.

**Acceptance Scenarios**:

1. **Given** no running shell, **When** the user runs `lumen start`, **Then** the shell starts, and running it again reports that a shell is already running without starting a second one.
2. **Given** a running shell, **When** the user runs a valid `lumen` command, **Then** the shell applies it live and the command exits with success.
3. **Given** no running shell or an invalid command, **When** the user runs a `lumen` command, **Then** it exits with a non-zero status and a clear message, and the shell (if running) is unaffected.
4. **Given** a helper that runs slowly or hangs, **When** it exceeds its time limit, **Then** it is stopped, the call fails with a clear error, and the display does not block.
5. **Given** a helper that exits with an error or invalid output, **When** the result is read, **Then** the error is reported with the helper's name and only the calling feature degrades.

---

### User Story 5 - Shared design tokens and logging (Priority: P3)

Components take spacing, radii, typography, and motion values from the Style singleton, and write diagnostics through the Log singleton. Changing a token changes every consumer. Log output is structured, leveled, and shared between the visual layer, the controller, and the helper scripts.

**Why this priority**: These keep later components coherent and debuggable, but they are not blocking.

**Independent Test**: Change a token and confirm all test-surface elements follow. Emit logs at each level from both layers and confirm they appear in one place, filtered by level.

**Acceptance Scenarios**:

1. **Given** a token change, **When** it is applied, **Then** all consumers update consistently.
2. **Given** log calls from the visual layer, the controller, and helper scripts, **When** they run, **Then** entries appear in one stream with level, source, and time.

---

### Edge Cases

- The user file is empty, unreadable, or not valid JSON: defaults apply, a clear error shows, and the file is never overwritten or discarded.
- The user file is saved repeatedly in a short time: the last saved state wins.
- The wallpaper file is missing or corrupt: the last good palette stays.
- A palette cannot be made compliant: it is rejected and the previous or fallback palette stays.
- A singleton is read before its first data arrives (for example, before the first helper result): it returns defaults or a loading state without blocking.
- The shell crashes: Lumen does not restart it; restarting is the session's job.
- Lumen is launched outside niri or with a missing required dependency: it reports the problem and exits cleanly.
- Config directories do not exist yet: they are created only in Lumen's own locations.

## Requirements *(mandatory)*

### Functional Requirements

**Project structure**

- **FR-001**: The repository MUST have a documented layout separating the visual layer, the controller and helper scripts, the shared singletons, modules, bundled data (presets, schemas), and documentation.
- **FR-002**: The project MUST provide documented commands to build, run, format, lint, and test, and they MUST pass on the skeleton.
- **FR-003**: The README MUST list all dependencies, mark optional ones, and describe how each degrades when absent.
- **FR-004**: The skeleton MUST start a shell that loads all core singletons and shows a minimal test surface for verification only.

**Singletons in general**

- **FR-005**: The project MUST provide these core singletons: Config, Theme, Style (design tokens), Backend, and Log. Each MUST have exactly one owner and one documented interface.
- **FR-006**: Components MUST consume shared state only through these singletons.
- **FR-007**: Singletons MUST keep non-trivial logic in the controller or helper programs, with the visual layer limited to presentation and light glue.

**Config**

- **FR-008**: Config MUST resolve in layers: built-in defaults, then selected theme or preset values (the preset's palette and mode), then user JSON overrides.
- **FR-009**: The user file MUST hold only explicit overrides, MUST NOT receive defaults, and MUST preserve unknown keys.
- **FR-010**: Config MUST validate against a published schema and report invalid values with key, problem, and fallback used.
- **FR-011**: Config changes MUST apply live, without restart.
- **FR-012**: Resetting a setting MUST remove its override.
- **FR-013**: Config, state, and cache MUST live in the standard per-user locations, and Lumen MUST NOT overwrite user files outside its own directories.

**Theme**

- **FR-014**: Theme MUST be the only source of color, exposing named semantic roles, and no component may hardcode a color.
- **FR-015**: Theme MUST support wallpaper-generated palettes (all scheme types, light and dark) and bundled presets (Tokyo Night and every Catppuccin flavor) mapped to the same roles.
- **FR-016**: Theme changes MUST apply live to all consumers without restart or flicker.
- **FR-017**: Theme MUST use a built-in fallback palette when generation is unavailable or fails, and MUST report why.
- **FR-018**: Every applied palette MUST meet the accessible minimum text contrast, or be adjusted or rejected.

**Style**

- **FR-019**: Style MUST provide spacing, corner radius, typography, and motion values that all components, including third-party ones, use.

**Backend**

- **FR-020**: The project MUST provide a one-shot `lumen` controller as the single command-line interface to the shell, with at least `start` and `theme <theme>`, and Backend MUST receive its commands through Quickshell's IPC using a documented, versioned command set. Backend MUST run helper programs as separate processes from QML. All command handling and helper runs MUST be asynchronous with time limits.
- **FR-021**: The controller MUST exit with a distinct non-zero status and a clear message when the shell is not running, the command is unknown, or the arguments are invalid. Backend MUST degrade only the affected feature when a helper fails, and MUST stop helpers that exceed their time limit.
- **FR-021c**: `lumen theme <theme>` MUST save the choice as an override in the user file by default, and with `--temp` MUST apply it live without writing the file. Resetting the theme MUST remove the override (see FR-012).
- **FR-021b**: `lumen start` MUST refuse to start a second shell when one is already running.
- **FR-021a**: Each helper program MUST be short-lived, take its input and return its result through a documented format, and be runnable on its own from a terminal for testing.
- **FR-022**: Compositor-specific behavior MUST sit behind an abstraction, with niri as the only implementation at first.

**Log**

- **FR-023**: Log MUST provide leveled, structured logging shared by the visual layer, the controller, and helper scripts in one stream.

**Reliability and privacy**

- **FR-024**: The system MUST NOT use the network or send telemetry by default.
- **FR-025**: Reloads of config or theme MUST NOT cause unbounded memory growth.

### Key Entities

- **Singleton**: A single-owner shared service that components consume (Config, Theme, Style, Backend, Log).
- **Configuration Layers**: Defaults, selected theme or preset values, and user overrides, resolved in that order.
- **User Override File**: The user's JSON file with only explicitly changed values.
- **Config Schema**: The published definition of valid configuration.
- **Palette**: A set of semantic color roles (light or dark), generated or from a preset.
- **Preset**: A bundled palette defined as data.
- **Fallback Palette**: The built-in palette used on failure.
- **Design Tokens**: Shared spacing, radius, typography, and motion values.
- **Controller**: The one-shot `lumen` command that starts the shell and sends it commands over Quickshell IPC, then exits.
- **Helper**: A small short-lived program (kept under `scripts/`) run from QML as a process for heavy tasks such as parsing or filtering large data.
- **Protocol**: The versioned command set between the controller and the shell, and the input/output format of each helper.
- **Test Surface**: The minimal screen that shows the singletons working.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer with the dependencies installed goes from a fresh clone to the running test surface in under 10 minutes using only the README.
- **SC-002**: The documented format, lint, and test commands pass with zero warnings on the skeleton.
- **SC-003**: A saved valid config edit shows up in the test surface in under 1 second. An invalid edit always shows an error naming the key, and the shell keeps running.
- **SC-004**: After any sequence of edits and resets, the user file contains no value the user did not explicitly set.
- **SC-005**: A preset or fallback theme change reaches 100% of test-surface elements within 1 second, without restart or visible flicker. A wallpaper-derived palette does so within 3 seconds, and the previous palette stays shown until the new one is ready.
- **SC-006**: 100% of palettes applied (light and dark wallpaper-derived, all presets, fallback) meet the accessible minimum text contrast.
- **SC-007**: A valid `lumen` command takes effect in the running shell in under 1 second, and an invalid one or one sent with no shell running exits with a clear error in under 1 second. A hung helper is stopped within its time limit with no loss of responsiveness.
- **SC-008**: With the skeleton idle, average CPU use stays under 1% over 5 minutes.
- **SC-009**: Over 100 config and theme reloads, memory use does not keep rising (final use within 10% of use after the first reload).
- **SC-010**: A developer can locate where a new component, helper script, or preset belongs from the layout documentation in under 2 minutes.

## Assumptions

- The users are Lumen developers and module authors, not end users.
- niri is the only supported compositor at first; other compositors and X11 are out of scope.
- Out of scope, left to later features: a Compositor singleton (niri workspace, window, and output state), a Modules singleton, the module loading system and manifests, the bar, notifications, launcher, control center, OSD, lockscreen, session menu, and settings app.
- Theme source is chosen through a config key, with no graphical picker in this feature.
- Exporting palettes to external apps (GTK, Qt, terminals) is left to a later feature.
- The wallpaper is read from the session's current wallpaper setting, and wallpaper management is out of scope.
- Dependencies (Quickshell, matugen, and others) are installed by the developer and listed in the README.
