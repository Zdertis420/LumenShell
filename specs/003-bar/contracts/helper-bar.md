# Contract: helper changes for the bar

Same common format as Foundation's helper contract (`specs/001-foundation/contracts/helper-protocol.md`): one JSON object on stdin, one `{ok, protocol, data | error}` document on stdout, ndjson diagnostics on stderr with `source: "helper:modules"`, `LUMEN_LOG_LEVEL` honored. Nothing here writes a file or runs module code.

## `modules scan` (additions)

Input is unchanged, with `"apiVersion": "1.1"`. Each module entry gains, when the manifest is valid:

```json
{
  "extensionPoints": [{"id": "items", "description": "Items shown in the bar"}],
  "contributions": [
    {"id": "clock", "name": "Clock", "target": {"module": "bar", "point": "items"},
     "file": "/…/bar-clock/Item.qml", "key": "bar-clock/clock", "short": "bar-clock"}
  ],
  "commands": ["focus"]
}
```

- `file` is made absolute; `key` is `<module>/<id>`; `short` is the module id for the contributor's first-declared contribution to a given target point, otherwise empty. (The short form is per target: a module that contributes to two hosts has a short form for the first item of each.)
- The validation rules are in [module-manifest-1.1.md](module-manifest-1.1.md). The warning for `extensionPoints` and `contributions` is removed; `requires` still warns.
- `commands` entries are returned as a plain list.

## `modules items`

Resolves the user's lists against the available contributions.

Input:

```json
{
  "point": {"module": "bar", "point": "items"},
  "lists": {"left": [], "center": ["bar-clock"], "right": ["bar-sound", "nope", "bar-sound/sound"]},
  "contributions": [
    {"key": "bar-clock/clock", "short": "bar-clock", "module": "bar-clock", "item": "clock",
     "name": "Clock", "moduleState": "loaded", "enabled": true, "reason": ""},
    {"key": "bar-sound/sound", "short": "bar-sound", "module": "bar-sound", "item": "sound",
     "name": "Sound", "moduleState": "loaded", "enabled": true, "reason": ""}
  ]
}
```

Data:

```json
{
  "left": [],
  "center": [{"name": "bar-clock", "key": "bar-clock/clock", "module": "bar-clock", "item": "clock"}],
  "right": [{"name": "bar-sound", "key": "bar-sound/sound", "module": "bar-sound", "item": "sound"}],
  "warnings": [
    {"place": "right", "index": 1, "entry": "nope", "problem": "no such item"},
    {"place": "right", "index": 2, "entry": "bar-sound/sound", "problem": "duplicate of right entry 0"}
  ],
  "available": [
    {"key": "bar-clock/clock", "short": "bar-clock", "name": "Clock", "state": "placed"},
    {"key": "bar-sound/sound", "short": "bar-sound", "name": "Sound", "state": "placed"}
  ]
}
```

Rules (the same order as in [bar-config.md](bar-config.md)):

1. An entry that is not a string of the allowed form (`^[a-z][a-z0-9-]{1,31}(/[a-z][a-z0-9-]{0,31})?$`) gives `not a valid item name`. (Entries that are not strings at all never reach this command: the `config` helper drops them first, see below. The command still tolerates them and reports `not a string`.)
2. An entry whose long form is already placed gives `duplicate of <place> entry <index>`.
3. No contribution matches the long or the short form: `no such item`.
4. A match whose `moduleState` is `disabled` or whose `enabled` is false: `module <id> is not enabled; run: lumen modules enable <id>`.
5. A match whose `moduleState` is `failed`: `module <id> failed: <reason>`.
6. Otherwise the entry is placed in its list, preserving order.

`available[].state` is `placed` (named and resolved) or `available` (not named, and the module is loaded). The command never fails on bad entries; `ok` is false only for malformed input (not JSON, lists not arrays, more than 64 entries in a list, more than 512 contributions), with code `invalid_input`.

Entries are not evaluated against screens or widths; that is the bar's work.

## `modules schema` (addition)

Setting rules of type `stringlist` appear as `{"type": "array", "items": {"type": "string", "minLength": 1, "maxLength": 64}, "maxItems": <maxItems>, "default": [...]}`.

## `config` helper changes

- New rule kind `stringlist` for module settings: the value is an array of at most `maxItems` entries (default 64), each a string of 1 to 80 characters.
  - A value that is not an array, or has more than `maxItems` entries, is `{key: "modules.<id>.<key>", problem, fallback}` with the module's default, and only that key falls back.
  - An entry that is not a string, is empty, or is longer than 80 characters is **dropped on its own**. The rest of the list is returned in order, and each dropped entry is reported as `{key: "modules.<id>.<key>[<index>]", problem, value, dropped: true}` (`index` counts in the stored list; `value` is the entry as written, cut to 80 characters). `set` accepts such a list too, and validates the rest as usual (a write never fails because of an entry that would be dropped on read, so hand-edited files and `set` behave the same).
- `string` module settings honor the optional `minLength` and `maxLength` (counted in characters): a value outside them is `{key, problem, fallback}` with the module's default, and only that key falls back. `modules schema` shows them as `minLength` and `maxLength`.
- `set` accepts a JSON array for a `stringlist` key and validates it as above, with one atomic write. `unset` removes the override as for any key. `resolve` returns the array under `values.modules.<id>.<key>`, with `sources` as usual.
- Core keys are unchanged. Without `modules` in the input, behavior is exactly as before.

## Shared rule semantics

The `modules` and `config` programs keep separate implementations of the rule check. The fixture test (`module_rules_agree_test.go`) is extended with `stringlist` cases (valid, too many entries, empty list; and for a default in a manifest: a non-string or too long entry makes the manifest invalid, while the same entry in a stored value is dropped on its own by `config`) and must give the verdicts described above through each program.

## The notify helper (module `bar`)

`modules/bar/bin/notify` follows the module helper contract of Modules.

- Input: `{"summary": "<text>", "body": "<text>"}` (both up to 200 characters).
- Runs `notify-send` with the two texts, detached, and answers `{"ok": true, "protocol": 1, "data": {}}` at once.
- Without `notify-send`: `{"ok": false, "protocol": 1, "error": {"code": "no_notifier", "message": "notify-send is not installed; install libnotify to get notifications"}}`.
- The bar calls it once per change that drops a wrong-type list entry; a failure is logged once and nothing else changes.

## The mixer helper (module `bar-sound`)

`modules/bar-sound/bin/open-mixer` follows the module helper contract of Modules.

- Input: `{"command": "<program and arguments, or empty>"}`.
- With a command: starts it detached. Without one: starts the first of `easyeffects`, `pavucontrol` that is found in `PATH`.
- Output: `{"ok": true, "protocol": 1, "data": {"started": "<program>"}}` at once, or `{"ok": false, "protocol": 1, "error": {"code": "no_mixer", "message": "no audio mixer found (install EasyEffects or pavucontrol, or set modules.bar-sound.mixer)"}}`, or `{"code": "start_failed", ...}`.
- The started program is a separate process, detached from the helper, so the helper exits immediately and the time limit is never reached.
