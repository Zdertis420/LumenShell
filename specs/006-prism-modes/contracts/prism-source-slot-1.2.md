# Contract: Prism source slot, version 1.2 (additive over 1.1)

Prism's version becomes 1.2.0. Everything of 1.0 and 1.1 is unchanged; a source that does not know the new members works as before.

| Member | Description |
|--------|-------------|
| `signal completeRequested()` | Emitted on every source's slot when the user presses Tab while the window has the keyboard. Prism does nothing else with the key. A source that completes something answers with `setText`; others ignore it |
| `slot.setText(text, cursor)` | Replaces the text of the input with `text` (a string; anything else is ignored with a warning). `cursor` is optional, a character index (0 to the length of `text`); it is clamped, and the default is the end. The change is a text change like typing: the mode and query follow, messages are cleared, the first row is selected, and every source gets `onQueryChanged`. A source should call it only as an answer to `completeRequested` |

Rules:

- Tab is handled only while the window is open and has the keyboard (Principle VIII); it adds no command and no setting.
- Prism does not know what a source completes; with no source answering, Tab changes nothing.
- The text limit of the window applies.

Documented in `docs/prism.md` (Sources and Keys). Used by `modules/prism-search` for the files and grep modes.
