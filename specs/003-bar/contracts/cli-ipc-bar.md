# Contract: controller and IPC additions for the bar

Additions to [specs/002-modules/contracts/cli-modules.md](../../002-modules/contracts/cli-modules.md) and [ipc-modules.md](../../002-modules/contracts/ipc-modules.md). The shell IPC stays at **protocol 1**: adding a function is a compatible change. The controller stays stdlib-only and only calls `qs`.

## New command: `lumen modules call <id> <command>`

| Behavior | |
|----------|---|
| Effect | Asks the module `<id>` to run its declared command `<command>` and prints the handler's answer |
| Example | `lumen modules call bar focus` |
| Exit 0 | The handler ran; its text is printed |
| Exit 2 | Invalid arguments (unsafe id, command name not matching `^[a-z][a-z0-9-]{0,31}$`) |
| Exit 3 | The shell is not running |
| Exit 6 | The handler threw, or the module is not loaded (`module <id> is not loaded`), or the command is not declared (`unknown command`) |
| Exit 7 | No module with this id |

Binding example for niri (`config.kdl`): `Mod+B { spawn "lumen" "modules" "call" "bar" "focus"; }`.

## IPC addition

| Function | Arguments | Returns | Effect |
|----------|-----------|---------|--------|
| `moduleCall(id, command)` | module id, command name | JSON text `{"ok": true, "result": "<text>"}`, or `error: unknown: unknown module '<id>'`, or `error: failed: <message>`, or `error: invalid: <message>` | Runs the module's handler and answers at once. The handler MUST be fast; it runs on the UI thread |

Handlers must not block. A handler that needs a helper starts it with `api.run` and returns immediately.

## `lumen modules list`

Unchanged in layout. For a module that contributes or hosts, the NOTES column gains `contributes to bar` or `hosts: items` as appropriate, and for a disabled module the permissions line shows the `contribute:<id>` entries like any other permission.

## `lumen modules show <id>`

New sections (printed only when the module has them):

```text
$ lumen modules show bar-clock
...
Contributions
  ITEM NAME          SHORT NAME   TARGET      STATE
  bar-clock/clock    bar-clock    bar/items   shown

$ lumen modules show bar
...
Extension points
  items   Items shown in the bar   contributions: 3 (shown 3)
Commands
  focus
Dropped items (screen DP-1): bar-battery        # only while items do not fit
Warnings
  right[2]: "weather": no such item
```

- `ITEM NAME` is the long form and `SHORT NAME` the short form (blank for an item that has no short form), so the user can copy what to put in a list.
- `STATE` is one of `shown`, `available` (installed and loaded but not named in a list), `waiting` (with the reason), `rejected` (with the reason), `failed`.
- The warnings come from the last list resolution.

## JSON shapes

`moduleShow(id)` detail gains `extensionPoints` (`[{id, description, contributions: [{key, short, name, state, reason}]}]`), `contributions` (`[{key, short, name, target, state, reason}]`), `commands` (`[string]`), and, for the bar, `dropped` (`[{screen, items}]`) and `itemWarnings` (`[{place, entry, problem}]`). `modulesList()` summary gains `notes` (`[string]`).
