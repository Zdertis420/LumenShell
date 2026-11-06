# Contract: `lumen modules` commands

Added to the one-shot controller (see `specs/001-foundation/contracts/cli.md`). Still stdlib-only, still only calls `qs`, and it never reads or writes configuration or module files.

## Commands

| Command | Behavior |
|---------|----------|
| `lumen modules list` | Prints one line per module: id, version, origin, state, and (for failed ones) the first error. Disabled modules show their permissions so they can be reviewed before enabling. Exits 0 |
| `lumen modules show <id>` | Prints the details: description, directory, origin, state, interface required and provided, permissions, settings with current values, helpers, errors, warnings, runtime error count and last error |
| `lumen modules enable <id>` | Enables and loads the module. Prints the resulting state; exits 6 if the module ended up failed |
| `lumen modules disable <id>` | Disables and unloads the module |
| `lumen modules reload <id>` | Re-reads the module's manifest and loads it again if enabled. Prints the resulting state; exits 6 if it ended up failed |
| `lumen modules rescan` | Scans the module directories again |

Output example:

```text
$ lumen modules list
ID             VERSION  ORIGIN   STATE     NOTES
test-surface   1.0.0    bundled  loaded
hello          1.0.0    user     disabled  permissions: helpers
broken         0.1.0    user     failed    api: requires interface 2.0, Lumen provides 1.0
```

## Exit codes

The Foundation codes still apply (0 success, 2 invalid arguments, 3 shell not running, 5 protocol mismatch, 6 shell failed the command, 8 environment problem, 9 timeout). New:

| Code | Meaning |
|------|---------|
| 7 | Unknown module id |

Every failure prints one clear line on stderr. Each command finishes in under 1 s on a normal system (SC-008). Commands that queue work wait for `status()` like `lumen theme` does.

## Argument rules

- `<id>` must match the manifest id pattern or `dir:` followed by `[A-Za-z0-9._-]{1,64}` (the form shown in the list for directories without a usable manifest); anything else is exit 2 before the shell is contacted.
- `enable`, `disable`, and `reload` wait until the module's state has settled, so the state printed afterwards is final.
- When the shell reports `attribution: degraded` (it could not follow its own log), `list` and `show` print a one-line note that runtime errors are not being counted.
- Unknown subcommand or missing id: exit 2 with the usage line.
- `lumen modules` with no subcommand prints the usage line and exits 2.
