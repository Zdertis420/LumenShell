# Contract: Prism source slot, version 1.1.0 (changes to Feature V's contract)

Everything in `specs/005-prism-window/contracts/prism-source-slot.md` stays. Prism's own version goes from 1.0.0 to 1.1.0 (a minor version: additions only). A source written for 1.0 works unchanged.

## Additions

| Member | Description |
|--------|-------------|
| `slot.setRows(rows, keepSelection)` | The second argument is optional (default `false`). When `true`, Prism keeps the selected row by its `id` if the row is still present, else the nearest index, instead of selecting the first row. Used for updates of the same query. It applies only after the user moved the selection since the text last changed; until then the first row stays selected, so a row that merely happened to be first (such as a lone last-placed web row) never holds the selection |
| `slot.setMessage(text, kind)` | Shows one line of text between the input and the results. `kind` is `info` or `error` (default `info`). An empty text clears it. Prism also clears it when the text of the query changes. One message per source; with several, the last set is shown. Cut to 200 characters |
| row `marks` | Optional list of `[start, end]` ranges (rune offsets, end exclusive) in `title`. Prism draws those parts with the `matchRole` color. Invalid ranges are dropped with a warning that names the source |

## Rules

- The message line is part of the window and is styled only by Prism's settings (`messageRole`, `errorRole`, the empty-state font). A source cannot style or place it.
- The message line is not a result: it cannot be selected, scrolled or counted, and it does not change `rows`.
- Nothing else changes: a source still cannot move, resize or restyle the window, and receives nothing private.

## Documentation and migration

`docs/prism.md` (Sources) gains the three additions with an example. No migration: all additions are optional.

## Tests

- QML unit test: marks validation and the keep-selection rule (as pure functions in a Prism `.js` file, `rows.js`, moved out of `Main.qml`, so they can be tested).
- Static scans extended to the new files and settings.
