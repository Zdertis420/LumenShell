# Quickstart: Validating Foundation

Run these once the feature is implemented. Contracts: [cli.md](contracts/cli.md), [ipc-protocol.md](contracts/ipc-protocol.md), [helper-protocol.md](contracts/helper-protocol.md), [config-keys.md](contracts/config-keys.md).

## Prerequisites

- Linux session running niri (Wayland)
- Go 1.27+, Quickshell 0.3+, Qt 6 tools (`qmllint`, `qmlformat`), `make`
- Optional: matugen (without it, wallpaper themes use the fallback palette)

## 1. Build and quality gates (US1, SC-001, SC-002)

```text
make build
make fmt lint test
```

Expected: binaries in `scripts/bin/`; format, lint, and test pass with zero warnings.

## 2. Start the shell and see the test surface (US1)

```text
scripts/bin/lumen start
```

Expected: the test surface appears, showing Config values, Theme roles, Style tokens, Backend status, and recent Log lines. A second `scripts/bin/lumen start` from another terminal exits with code 4 and the first shell stays up. Right after start, `scripts/bin/lumen version` prints the protocol version, which proves the IPC handler was instantiated.

## 3. Config live update and fallback (US2, SC-003, SC-004)

1. No `~/.config/lumen/config.json`: the surface shows defaults, and no file appears.
2. Write `{"theme":{"preset":"tokyo-night"}}` to the file: the surface updates in under 1 s and the file still holds only that override.
3. Set `log.level` to an invalid value: an error naming `log.level` appears, only that key falls back, and the shell keeps running.
4. Save an invalid JSON file: defaults apply, an error shows, and the file content is unchanged.
5. Remove the override with the reset action: the key leaves the file. Unknown keys added by hand remain.

## 4. Theme sources (US3, SC-005, SC-006)

Check each: a light wallpaper-derived palette, a dark one, two presets, and the fallback (run with matugen hidden from PATH).

Expected: every consumer on the surface changes within 1 s (3 s for wallpaper palettes, with the previous palette shown until the new one is ready) with no flicker, text contrast is at least 4.5:1, and the fallback reason shows in the status line and the log.

## 5. Controller and Backend (US4, SC-007)

```text
scripts/bin/lumen theme catppuccin-latte     # saved, applied live, exit 0
scripts/bin/lumen theme tokyo-night --temp   # applied, file unchanged
scripts/bin/lumen theme --reset              # override removed
scripts/bin/lumen theme nope                 # exit 2, shell unaffected
```

Stop the shell, then run `scripts/bin/lumen theme tokyo-night`: exit 3, a clear message, and no file written. With the fixtures in `scripts/tests/fixtures/`, run a helper that fails and one that hangs: the shell stays responsive, the error names the helper, and the hung helper is killed within its limit.

Run a helper alone: `echo '{"userFile":"/tmp/c.json","dataDir":"data"}' | scripts/bin/config resolve`.

## 6. Style and Log (US5)

Change a token in `singletons/Style.qml`: every surface element follows. Trigger log calls from QML, the controller, and a helper: all appear as ndjson lines in the shell's stderr with level, source, and time, filtered by `log.level`.

## 7. Resource checks (SC-008, SC-009)

- Idle 5 minutes, then check average CPU (`pidstat` or `top`): under 1%.
- `scripts/tests/smoke.sh --reload 100` triggers 100 config and theme reloads. Final memory use stays within 10% of the use after the first reload.

## 8. Scripted run

```text
make smoke
```

Runs steps 2, 3, and 5 where possible and reports pass or fail.
