# Research: Prism Window (Skeleton)

All items are decisions for the plan; spikes are facts that need a short experiment in a real Quickshell on niri, each with a fallback so none blocks the work.

## R1. Placing the window in the usable area (spike)

- **Decision**: One transparent overlay-layer `PanelWindow` anchored to all four edges, with `exclusionMode: Normal` (zero zone of its own). A layer surface anchored on all sides with no exclusive zone of its own is sized by the compositor to the area that other surfaces' exclusive zones leave (the bar's reserved space). The panel is placed inside that surface by `geometry.js`, so the "usable area" of the spec is simply the surface's size.
- **Rationale**: Modules cannot read what the bar reserves, and should not (Principle III). Letting the compositor shrink the surface needs no new interface.
- **Spike**: Confirm on niri that an overlay surface with all anchors and zone 0 is shrunk by the bar's zone (and by the frame style's border panels of Feature IV). Expected from the layer-shell rules (zones apply to surfaces that anchor to the same edges), but niri's treatment across layers must be seen.
- **Fallback**: Put the window on the `Top` layer (the bar's layer, where zones certainly apply), accepting that it then sits under full-screen apps; or, if neither works, cover the whole screen and document that an anchored window may overlap the bar by its margin (FR-016 then reads "of the screen").
- **Alternatives**: Reading the bar's thickness through `api.modules.settingsOf("bar")` (needs a permission, duplicates the bar's geometry rules, breaks for another bar module): rejected.

## R2. Which screen it opens on (spike)

- **Decision**: With the `screen` setting empty, create the window without assigning a screen, so the compositor picks the output (niri uses the focused output for a layer surface without one). With a name set, assign that screen if it exists, otherwise create without one and warn. The window lives in a `Loader` and is created on open and destroyed on close, so every open picks the output again; an open command while open destroys and re-creates it (FR-007: never on two screens).
- **Rationale**: "The screen with the focused window" is compositor knowledge. Asking the compositor through the layer-shell protocol keeps compositor-specific code out of the module, as `docs/layout.md` demands.
- **Spike**: Confirm that Quickshell 0.3.2 passes a null output to the compositor when `screen` is not set, and that niri then chooses the focused output.
- **Fallback**: One helper `modules/prism/bin/focused-output` that runs `niri msg --json focused-output` and returns the name; documented in `docs/layout.md` as a fourth compositor-specific place, with a Go replacement later in a Compositor singleton.
- **Alternatives**: A hidden persistent surface per screen plus a pointer tracker: more surfaces and idle cost, rejected by SC-009.

## R3. Keyboard focus and giving it back

- **Decision**: `WlrLayershell.keyboardFocus: Exclusive` only while the window exists and is shown; `None` otherwise. The input gets `forceActiveFocus()` on open. Escape is handled on the root item (`Keys.onEscapePressed`) and closes at once (clarification: no clearing first). The compositor returns focus to the previously focused window when the surface is destroyed or loses exclusivity; the bar's popup already depends on this (Feature III).
- **Rationale**: Principle VIII allows a keyboard-input module to take focus when the user's command opens it. Destroying the surface on close makes the release certain.
- **Alternatives**: `OnDemand` focus: the user would have to click the window first, which defeats a search box. Rejected.

## R4. Marking the recognized prefix (spike)

- **Decision**: A rounded "chip" rectangle behind the prefix characters, in the `prefixRole` color with `prefixOpacity` (default 0.25), placed from `TextInput.positionToRectangle` of the prefix start and end. The glyphs keep the input text color, so contrast is that of the input text on a light tint of the prefix role. The chip follows the text scroll of a long input.
- **Rationale**: `TextInput` cannot color part of its text, and rich-text editors reset the cursor when reformatted. A chip needs no text duplication and keeps editing, selection, IME and cursor behavior native.
- **Spike**: Check that `positionToRectangle` follows horizontal scrolling and that the chip stays aligned while typing and with right-to-left text.
- **Fallback**: Drop the chip and color the whole input text by mode with the same role setting (one color for "prefix recognized").
- **Spec note**: The spec's "drawn in its own color role" is met by this chip (a highlight in the prefix role). The spec text is adjusted to say "highlighted".

## R5. Input methods under an exclusive layer-shell focus (spike)

- **Decision**: Use the stock `TextInput` and rely on Qt's text-input support for composition (spec edge case). Do nothing special.
- **Spike**: Type with an input method (for example a Compose sequence and a CJK engine) into the window on niri.
- **Fallback**: If composition does not reach a layer-shell surface, document it as a known limitation of the compositor path in `docs/prism.md`; the window stays usable for direct text.

## R6. Shadow and backdrop

- **Decision**: The shadow is a `MultiEffect` (Qt 6.12, `QtQuick.Effects`) on the panel, created by a `Loader` only while `shadow` is on. The backdrop is a plain full-size rectangle in the backdrop role with opacity; no blur (spec assumption). Both are inside the one surface, so no extra surface exists.
- **Alternatives**: Layered translucent rectangles for a fake shadow: no blur control, rejected. Compositor blur: compositor-specific, left to a later feature.

## R7. Opening animation

- **Decision**: One `ParallelAnimation` driven by a single `progress` value (0 to 1) over `animationDuration` (unset means `Style.motion.normal`): `fade` changes opacity, `scale` changes opacity and scale from `animationScale` (default 0.96) to 1 around the panel's anchor point, `slide` moves the panel by a distance from the anchor's nearest edge (its own token, `Style.spacing.xl`). With `none` or duration 0 `progress` jumps. The surface is destroyed when the closing animation ends.
- **Rationale**: One value keeps the four kinds simple and testable. Height changes from the row count are not animated (they would jitter while typing).
- **Alternatives**: A spring per kind: more settings, rejected (YAGNI).

## R8. Where the panel's vertical position comes from

- **Decision**: The panel's top edge is fixed for the top row of anchors and for the middle row; its bottom edge is fixed for the bottom row. The middle row (`left`, `center`, `right`) places the top so that the panel at its **maximum** height is centered, so the window never jumps as results come and go. The default (`anchor: top`, `offsetY` unset) puts the top at one fifth of the usable height, like Spotlight; `offsetY` set means that exact distance from the anchor edge.
- **Rationale**: FR-012 requires the anchor edge to stay fixed while the results grow away from it. A top-fixed middle anchor satisfies it and avoids motion.
- **Alternatives**: Keep the center of the panel fixed (grows both ways): the input moves while typing, which is the worst place for it. Rejected.

## R9. The source slot and the row model

- **Decision**: Prism declares the extension point `sources`. For each available contribution, Prism creates the item through `api.extensions.create("sources", name, holder, slot)` with a per-source slot object (see `contracts/prism-source-slot.md`). Rows are plain objects `{id, title, subtitle?, hint?, icon?}`; the source calls `slot.setRows(rows)`. Prism concatenates the rows in the order of the contributions, keeps `(sourceName, id)` as the row key, and routes `chosen` to the supplying slot. Sources are created when the module loads and live until it unloads, so a source can keep warm state; they get `slot.open` to know when to work.
- **Rationale**: This reuses the bar's proven pattern (declared point, created items, `slot`) with no new `api` member, so the interface stays at 1.2 apart from the limit.
- **Alternatives**: A new `api.prism` member: a larger interface change for no gain. Per-mode extension points: the mode is already a field of the slot.

## R10. The prefix rule

- **Decision**: In `prefix.js`, `parse(text, letters)` returns `{mode, query, prefixLength}`. The text must start with `@` followed by exactly one configured letter and then a space or the end of the text; otherwise the mode is `default` and the query is the whole text. The query is the text after the prefix and a single space. Letters are case-sensitive. The five letters are validated by the Go rules as one lowercase letter or digit (`pattern`), and `geometry.js`'s settings check turns a duplicate letter into an error that names the later key and falls that key back to its default (the default letters are distinct, so the result is always valid).
- **Rationale**: A pure function with a table test covers every spec case: `@f`, `@f x`, `@fx` (not a prefix), `@x y`, `@@f`, an empty text, and swapped letters.

## R11. Settings count and limit

- **Decision**: 105 keys: placement 9, behavior 8, content 8, prefix letters 5, shape and spacing 14, backdrop 1, motion 3, prefix chip 1, colors 24 (17 roles and 7 opacities), fonts 32 (4 global and 7 roles by 4). The per-module limit rises from 64 to 128 (the schema's `maxProperties` and `maxSettings` in `manifest.go`), declared as interface 1.3. Key names stay within the existing 32 character pattern.
- **Rationale**: The spec asks for fonts per role with family, size, weight and italic, and for 17 colored parts. Fewer keys would drop surface that Principle VII requires.
- **Alternatives**: Nested settings objects: a larger change to the setting types and the settings app. Rejected for this feature.

## R12. Contrast

- **Decision**: `geometry.js` offers `contrast(a, b)` (WCAG relative luminance ratio) and the module checks the input text, row text, subtitle, hint and selected text against their backgrounds after blending opacity over the window background. A pair below 4.5 logs one warning that names both keys (the user's choice is kept; the default roles are chosen to pass in every preset, and the quickstart verifies it). Principle II's "adjust or reject" applies to palettes (the theme service does it); a user's own role pairing is warned about, not rewritten, which matches how the bar treats user choices.
- **Alternatives**: Silently correcting the user's choice: surprising, rejected.

## Spike results (2026-11-09, niri, Quickshell 0.3.2, three outputs)

- **R1 (usable area): confirmed, no fallback.** A fake bar (top, 40 px, `exclusionMode: Auto`) and an overlay-layer `PanelWindow` anchored to all four edges with `exclusionMode: Normal` gave the overlay a size of 1920 x 1040 on a 1920 x 1080 output: the compositor shrinks it by the bar's reserved space. The frame style of Feature IV (border panels with their own zones) works the same way by the layer-shell rule, and is checked by hand in quickstart section 4.
- **R2 (output choice): confirmed, no fallback.** A `PanelWindow` with no `screen` assigned opened on the focused output, and again on the other output after `niri msg action focus-monitor-next` (DP-2, then DP-1). No compositor-specific code or helper is needed.
- **R4 (prefix chip): confirmed.** `TextInput.positionToRectangle(position)` returns coordinates in the item's own space including the horizontal scroll (x was -124 for position 0 once the long text had scrolled to the end), so the chip follows the scroll; it must be clipped to the input's visible area. The chip branch is chosen.
- **R5 (input method): not checkable without a person.** Left as a by-hand check in quickstart section 3 (compose sequence and a CJK engine). If composition fails, `docs/prism.md` records it as a known limitation.

## R13. What is deliberately not done

No search, ranking, history or running of anything; no blur; no icons theme logic beyond `Quickshell.iconPath` with a placeholder; no per-screen windows; no key bindings. The footer hints are plain text the user sets.
