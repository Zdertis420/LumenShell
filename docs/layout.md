# Layout: where things go

| You are adding... | Put it in |
|-------------------|-----------|
| A shared service all components use | `singletons/` (and register it in `singletons/qmldir`). There are six: Config, Theme, Style, Backend, Log, and Modules (the module registry) |
| A shell component (bar, launcher, notifications, ...) | `modules/<name>/` with a `module.json` and a `Main.qml`. It must use only the module interface (`api`), see [module-api.md](module-api.md) |
| An item for the bar | A module with a contribution to `bar/items` (a `module.json` with `contributions` and an `Item.qml`); bundled ones are `modules/bar-<name>/`. See [bar.md](bar.md) and `examples/hello-bar-item/` |
| How the bar is drawn or placed | `modules/bar/`: `BarWindow.qml` (the panel, the strip and the islands), `FrameBorders.qml` (the frame's border panels and rounded corners), `geometry.js` (sizes and limits), `layout.js` (the fitting rule along the bar), `popup.js` (popup placement). The pure `.js` files have QML unit tests in `scripts/tests/bar_*_test.qml` |
| How the Prism search window is drawn or placed | `modules/prism/`: `PrismWindow.qml` (the surface, panel, input, results and footer), `ResultRow.qml` (one row), `geometry.js` (sizes, the nine placements, fonts, contrast), `prefix.js` (the `@x` prefix) and `sample.js` (preview rows). The pure `.js` files have QML unit tests in `scripts/tests/prism_*_test.qml`. See [prism.md](prism.md) |
| How Prism searches (the six modes) | `modules/prism-search/`: `Source.qml` (the query state, the helper runs, the rows and the actions), `stages.js` (the stage plans and the ordered release of rows; QML unit tests in `scripts/tests/search_stages_test.qml`). The matching, walking, parsing and starting of programs is the Go program `scripts/src/search/`, built by `make build` into `modules/prism-search/bin/search` (`MODULE_PROGRAMS` in the `Makefile`); its tests are `scripts/tests/search_*_test.go`. See [prism.md](prism.md#search-modes) |
| Something that supplies rows to Prism | A module with a contribution to `prism/sources` (see `examples/hello-prism-source/` and [prism.md](prism.md#sources)) |
| A module you write for yourself or to share | `~/.local/share/lumen/modules/<name>/` (or `LUMEN_USER_MODULES_DIR`); see [writing-a-module.md](writing-a-module.md) |
| An example module for documentation | `examples/<name>/` |
| The manifest schema | `data/schema/module.schema.json`, kept in step with `scripts/src/modules/manifest.go` by a test |
| The module interface (what `api` offers) | `singletons/ModuleApi.qml`, `specs/002-modules/contracts/module-api.md`, and `docs/module-api.md` (bump the interface version for changes) |
| A helper that does heavy work (parsing, filtering, validation) | `scripts/src/<name>/` as a standalone Go `package main`; add `<name>` to `PROGRAMS` in the `Makefile` |
| A test for a helper | `scripts/tests/<name>_test.go` (black-box: run the built program) |
| A fixture used by tests | `scripts/tests/fixtures/` |
| A bundled preset | `data/presets/<id>.json` and add the id to `data/schema/config.schema.json` and `specs/001-foundation/contracts/config-keys.md` |
| A config key | `data/schema/config.schema.json`, the defaults and rules in `scripts/src/config/schema.go`, and `specs/001-foundation/contracts/config-keys.md` (a test keeps them in sync) |
| An IPC function | `singletons/Backend.qml` and `specs/001-foundation/contracts/ipc-protocol.md` (bump `protocol` for breaking changes) |
| A document | `docs/` |
| A specification or plan | `specs/<feature>/` |

## Rules

- QML holds presentation and light glue only. Parsing, validation, merging, and color work belong in a Go program.
- No shared Go code between programs. Each program under `scripts/src/` is built on its own, with only the standard library. If a dependency ever becomes unavoidable, add a single `go.mod` in `scripts/` and record why in the plan.
- The controller (`scripts/src/lumen`) only calls `qs`. It never reads or writes configuration.
- Components never hardcode colors, spacing, radii, font sizes, or durations. Use `Theme` and `Style`.
- Terminology: *helper programs* are the Go programs run by Backend (`config`, `palette`, `modules`); `scripts/` is only the directory name. A module can ship its own helper programs, declared in its manifest and run through `api.run`.
- Bundled modules and examples may not touch the singletons directly; a test checks this. Test fixtures with deliberately broken QML live in `scripts/tests/fixtures/`, which `make fmt` and `make lint` skip.

## Compositor-specific code

niri is the only supported compositor. Compositor-specific code is isolated in three places, so adding another compositor means changing only these:

- `checkSession` in `scripts/src/lumen/main.go` (is a supported session running?)
- `detectWallpaper` in `scripts/src/palette/wallpaper.go` (which wallpaper is set?)
- `checkSession` in `shell.qml` (the shell's own session check)

Modules do not contain compositor-specific code; a later Compositor singleton will offer it through the module interface.

This is isolation, not a full abstraction layer. A Compositor singleton (workspaces, windows, outputs) is left to a later feature.
