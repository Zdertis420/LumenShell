# Quickstart: Validating Modules

Run these once the feature is implemented. Contracts: [module-manifest.md](contracts/module-manifest.md), [module-api.md](contracts/module-api.md), [ipc-modules.md](contracts/ipc-modules.md), [cli-modules.md](contracts/cli-modules.md), [helper-modules.md](contracts/helper-modules.md).

## Prerequisites

- Foundation validated (`specs/001-foundation/quickstart.md`), niri session, Go, Quickshell, Qt 6 tools, `make`
- A scratch user modules directory: `export LUMEN_USER_MODULES_DIR=$(mktemp -d)` and a scratch config: `export LUMEN_CONFIG_FILE=$(mktemp -d)/config.json`

## 1. Build and gates

```text
make build
make fmt lint test
```

Expected: four programs in `scripts/bin/` (`lumen`, `config`, `palette`, `modules`); everything passes with zero warnings.

## 2. The built-in module (US2, SC-010)

```text
scripts/bin/lumen start
scripts/bin/lumen modules list
```

Expected: the test surface window appears; the list shows `test-surface` as `bundled`, `loaded`. The test that checks which `api.` members the surface uses passes (`make test`).

## 3. Write and load a module (US1, SC-001, SC-002)

```text
cp -r examples/hello-module "$LUMEN_USER_MODULES_DIR/"
scripts/bin/lumen modules rescan
scripts/bin/lumen modules list        # hello: user, disabled, permissions: helpers
scripts/bin/lumen modules enable hello
```

Expected: after `rescan` the module is listed as `disabled` with its permissions and does not run. After `enable` (under 1 s), the hello window appears and the state is `loaded`. Restart the shell: it comes back up enabled. `lumen modules disable hello` unloads it, and the override disappears from the config file because the default is off.

## 4. Services through the interface (US2, SC-004)

With hello loaded: `lumen theme tokyo-night` and `lumen theme catppuccin-latte` change the hello window's colors live. Its log lines appear in the shell stream with source `module:hello`. The helper result shows in the window; break the helper (`chmod -x`) and `lumen modules reload hello` reports the module failed with the reason.

## 5. Settings (US3, SC-005, SC-006)

1. Add `{"modules":{"hello":{"greeting":"Hi"}}}` to the config file: the window updates in under 1 s and the file holds only that override.
2. Set `"size": 99` (limit is 5): an error names `modules.hello.size`, only that key falls back, other modules are unaffected.
3. Remove the key through the module's own reset, or by editing the file: the override is gone. A `modules.gone.x` entry for an uninstalled module survives all of this.

## 6. Broken modules (US4, SC-003)

Copy each fixture from `scripts/tests/fixtures/modules/` into `$LUMEN_USER_MODULES_DIR` (bad manifest, missing file, newer interface, duplicate id, visual error, runtime error storm, failing helper, hanging helper), run `lumen modules rescan`, then enable all of them.

Expected: the good modules keep running, the shell answers `lumen version` throughout, and each broken module is listed with its id and a clear reason. The error-storm module is unloaded after three errors within 5 s and is not loaded again until `lumen modules reload <id>`.

## 7. Commands (US5, SC-008)

```text
scripts/bin/lumen modules show hello
scripts/bin/lumen modules enable nope        # exit 7
scripts/bin/lumen modules enable "bad id"    # exit 2
```

Stop the shell and run `scripts/bin/lumen modules list`: exit 3. Each command takes under 1 s.

## 8. Resource checks (SC-007, SC-009)

- `scripts/tests/smoke.sh --modules-cycles 100` loads and unloads a module 100 times: memory within 10% of the use after the first cycle.
- Copy the example module under 10 different ids, enable them all, idle 5 minutes: average CPU under 1%.

## 9. Scripted run

```text
make smoke
```

Runs the Foundation checks plus the module checks (steps 2, 3, 5, 6, and 7) against a real shell with a temporary config and module directory.
