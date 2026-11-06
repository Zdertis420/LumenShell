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

## Versioning

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

Recorded when measured (see tasks T040, T057):

- matugen run time (matugen 4.2.0, 96x96 test images, whole `palette` helper run): slowest of 60 scheme/mode/image combinations 8 ms. In a live shell, a wallpaper palette was applied about 7 ms after the config change; a changed wallpaper file was picked up within the 2 s poll
- Idle CPU over 5 minutes (2026-11-06, Quickshell 0.3.2, test surface open): 87 ticks / 300 s = 0.29% of one core (target < 1%)
- Memory over 100 reloads (`scripts/tests/smoke.sh --reload 100`): 393,288 kB after the first reload, 388,848 kB after the 100th (limit +10%)
