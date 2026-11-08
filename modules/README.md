# modules/

Bundled modules. They load automatically (see [docs/module-api.md](../docs/module-api.md)).

| Module | What it is |
|--------|------------|
| `bar` | The bar: three places (left, center, right) on each screen, filled from the lists in the configuration file ([docs/bar.md](../docs/bar.md)) |
| `bar-clock` | Time and date item for the bar |
| `bar-sound` | Output volume and mute item for the bar (with a helper that opens a mixer) |
| `bar-battery` | Battery level item for the bar (hidden without a battery) |
| `test-surface` | A window that shows the theme, configuration, module list and log, for verification |

Every module here uses only the public module interface (`api`); a test checks it.
