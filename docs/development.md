# Development

## Build without a Go module

There is no `go.mod`. Each program is built from its own directory:

```text
go build -o scripts/bin/config scripts/src/config/*.go
```

`make build` does this for every program. `make lint` runs `gofmt -l`, `go vet` per program, and `qmllint`.

## Run a helper alone

Helpers read one JSON object on stdin and write one JSON object on stdout. Diagnostics go to stderr as ndjson.

```text
echo '{"userFile":"/tmp/c.json","dataDir":"data"}' | scripts/bin/config resolve
echo '{"dataDir":"data","source":"preset","preset":"tokyo-night","hasPrevious":false}' | scripts/bin/palette
```

Contracts: `specs/001-foundation/contracts/`.

## Tests

```text
make test      # go test scripts/tests/*_test.go (builds the programs itself)
make smoke     # scripts/tests/smoke.sh (needs a niri session)
```

Tests are black-box: they run the built programs and check stdout, stderr, and exit codes.

## QML rules

- Presentation and light glue only. Heavy logic goes into a Go program.
- All helper calls are asynchronous (`Backend.run`) and have a time limit.
- Singletons are created lazily, so `shell.qml` touches all of them at startup.
- `Qt.quit()` has no receiver in Quickshell 0.3; `shell.qml` stops itself with `kill -TERM` (exit status 143).

## Modules

- The module registry is `singletons/Modules.qml`; each loaded module gets its own `singletons/ModuleApi.qml` object as `api`. Scanning and manifest validation are done by the `modules` program.
- Run the scanner by hand: `echo '{"bundledDir":"modules","userDir":"","apiVersion":"1.0"}' | scripts/bin/modules scan`, and print the merged configuration schema with `... | scripts/bin/modules schema` (add `"dataDir":"data"`).
- Fixtures for the tests are in `scripts/tests/fixtures/modules/`. Some contain deliberately broken QML, so `make fmt` and `make lint` skip that directory.
- `scripts/tests/smoke.sh --modules-cycles 100` loads and unloads a module 100 times and checks memory.
- Reloading a module re-reads its `module.json` and `Main.qml` (the component is loaded from a URL with a changing `?r=<n>`). Other QML files of a module are cached by the engine and need a shell restart.

## Versioning

The module interface has its own `major.minor` version (`Modules.apiVersion`): adding a capability raises the minor number; a removal or a change of meaning raises the major number and needs a migration note in `docs/module-api.md`.

The IPC protocol (`singletons/Backend.qml`, `contracts/ipc-protocol.md`) and the helper JSON formats carry an integer `protocol`. Adding a function or field is compatible. Removing or changing the meaning of one bumps `protocol` and needs a migration note here:

```text
### Migration: protocol N -> N+1 (date)
What changed:
What to do:
```

## matugen status (task T035a)

**Verified with matugen 4.2.0** (2026-11-06). The `palette` helper runs:

```text
matugen image <path> --json hex --type <scheme> --mode <light|dark> --prefer saturation --dry-run
```

- `--prefer` is required. Without it matugen asks on a terminal when an image has several candidate source colors, and fails when no terminal is attached.
- `--dry-run` keeps matugen from writing templates or running hooks from the user's own matugen setup.
- Output shape: `{"colors": {"<name>": {"dark": {"color": "#rrggbb"}, "default": {...}, "light": {...}}, ...}}`. All 16 color names in the table in `specs/001-foundation/data-model.md` exist in it.
- Scheme types: matugen 4.2.0 offers `scheme-content`, `scheme-expressive`, `scheme-fidelity`, `scheme-fruit-salad`, `scheme-monochrome`, `scheme-neutral`, `scheme-rainbow`, `scheme-tonal-spot`, `scheme-vibrant` and `scheme-smart`. Lumen accepts all of them in `theme.scheme`.

The tests use `scripts/tests/fixtures/matugen-fake.sh`, which imitates this output (and fails without `--prefer`, like the real one). `scripts/tests/real_matugen_test.go` runs the real matugen when it is installed, for every scheme in both modes on three test images, and checks contrast. If matugen changes its flags or output, those tests fail and the helper degrades safely to the fallback palette with the reason shown on the test surface.

## Measurements

Recorded when measured (see tasks T040, T057 of feature 001 and T062, T063 of feature 002):

- matugen run time (matugen 4.2.0, 96x96 test images, whole `palette` helper run): slowest of 60 scheme/mode/image combinations 8 ms. In a live shell, a wallpaper palette was applied about 7 ms after the config change; a changed wallpaper file was picked up within the 2 s poll
- Idle CPU over 5 minutes (2026-11-06, Quickshell 0.3.2, test surface open, no config file yet): 87 ticks / 300 s = 0.29% of one core (target < 1%). That run included the 500 ms retry that waits for a missing config file to appear.
- Idle CPU over 5 minutes with 11 modules loaded (the test surface and 10 copies of the example module, config file present, shell plus its log follower): 0 ticks / 300 s, below 0.01% of one core (SC-009, target < 1%). Startup itself cost about 23 ticks.
- Module load time (SC-002): `lumen modules enable hello` returns in 141–159 ms with the module loaded; an `enabled` edit in the config file shows the module loaded 105–157 ms later; the bundled test surface is loaded about 18 ms after the shell logs "starting".
- Module load/unload memory (SC-007, `scripts/tests/smoke.sh --modules-cycles 100`): 402,148 kB after the first cycle, 405,088 kB after the 100th (+0.7%, limit +10%).
- Author guide (SC-001): `docs/writing-a-module.md` was followed from an empty directory (manifest, `Main.qml`, `rescan`, `show`, `enable`, then the helper step). A first run exposed a gap (the manifest declared a helper that a later step creates); the guide was fixed and the second run loaded the module in 159 ms of command time. Typing the files takes a few minutes; nothing is left that needs outside help.
- Memory over 100 reloads (`scripts/tests/smoke.sh --reload 100`): 393,288 kB after the first reload, 388,848 kB after the 100th (limit +10%)
