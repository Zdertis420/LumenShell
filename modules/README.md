# modules/

Bundled modules. They load automatically (see [docs/module-api.md](../docs/module-api.md)).

| Module | What it is |
|--------|------------|
| `bar` | The bar: three places (left, center, right) on each screen, on any of four edges, in one of three styles (solid, islands, frame), filled from the lists in the configuration file ([docs/bar.md](../docs/bar.md)) |
| `bar-clock` | Time and date item for the bar (hours above minutes on a vertical bar) |
| `bar-sound` | Output volume and mute item for the bar (with a helper that opens a mixer) |
| `bar-battery` | Battery level item for the bar (hidden without a battery) |
| `prism` | The Prism search window (the window only): opened by a command you bind, placed at one of nine positions, themed by roles, with a mode prefix (`@a`, `@f`, `@w`, `@s`, `@g`) and rows that other modules supply ([docs/prism.md](../docs/prism.md)) |

The three item modules each carry the same small `ItemLook.qml` (their own font, padding, radius and color settings, else the bar's values); a test keeps the copies equal.
| `test-surface` | A window that shows the theme, configuration, module list and log, for verification |

Every module here uses only the public module interface (`api`); a test checks it.
