# Contract: the `modules` program and the `config` helper changes

Same common format as Foundation's helper contract (`specs/001-foundation/contracts/helper-protocol.md`): one JSON object on stdin, one `{ok, protocol, data | error}` document on stdout, ndjson diagnostics on stderr with `source: "helper:modules"`, `LUMEN_LOG_LEVEL` honored.

## `modules scan`

Input:

```json
{"bundledDir": "/…/modules", "userDir": "/home/u/.local/share/lumen/modules", "apiVersion": "1.0"}
```

`userDir` may be empty or point to a directory that does not exist; that is not an error and nothing is created.

Data:

```json
{
  "apiVersion": "1.0",
  "modules": [
    {
      "id": "hello", "dir": "/…/hello", "dirName": "hello", "origin": "user",
      "valid": true, "defaultEnabled": false,
      "main": "/…/hello/Main.qml",
      "manifest": { "…": "the parsed manifest, reserved fields included" },
      "errors": [], "warnings": [{"field": "contributions", "problem": "contributions is not supported in module interface 1.0"}]
    }
  ]
}
```

Rules:

- Order: bundled modules first, then user modules, each sorted by directory name.
- A directory with no `module.json`, with unreadable or invalid JSON, or with a manifest over 64 KiB gives an entry with `valid: false`, an id of `dir:` plus the directory name (every character outside `[A-Za-z0-9._-]` replaced by `_`, cut to 64 characters) when no usable id exists, and an error on field `""`. When the name had to be changed, the error also says so, for example `directory name "bad name!" was shown as dir:bad_name_`.
- Duplicate ids follow the rule in the manifest contract; both entries are returned, the loser invalid.
- `main` is the absolute path of the entry file when the manifest is valid.
- At most 256 directories are scanned; further ones are reported as a single error entry `dir:…` with `problem: "too many modules"`.
- Never writes anything and never runs module code.

## `modules schema`

Input: `{"bundledDir": …, "userDir": …, "apiVersion": "1.0", "dataDir": "/…/data"}`.

Data: `{"schema": <JSON Schema>}` — the core schema from `dataDir/schema/config.schema.json` extended with `properties.modules.properties.<id>` for every valid module (the `enabled` boolean plus the module's settings with types, enums, limits, defaults, and descriptions). Intended for editors; run it directly from a terminal.

## `config` helper changes

All three subcommands (`resolve`, `set`, `unset`) accept an extra input field:

```json
"modules": [
  {"id": "hello", "defaultEnabled": false,
   "settings": {"greeting": {"type": "string", "default": "Hello"},
                "size": {"type": "integer", "default": 3, "min": 1, "max": 5}}}
]
```

- **`resolve`** also returns, under `values.modules.<id>`, `enabled` (default from `defaultEnabled`) and every declared setting with defaults filled in, with matching entries in `sources`. Invalid module values are dropped and reported as `{key: "modules.<id>.<key>", problem, fallback}` with the module's default as the fallback. Keys under `modules.<id>` that are not declared, and whole `modules.<id>` objects for ids not in the list, are not validated and not reported.
- **`set`** and **`unset`** accept keys `modules.<id>.<key>` and `modules.<id>.enabled` for ids in the list (validated like core keys, one atomic write), and refuse unknown module ids or undeclared keys with `invalid_key`. Empty `modules.<id>` parents are removed on unset.
- Rule types in `config` now are `string`, `integer`, `number`, and `boolean`. Core keys are unchanged.
- Without `modules` in the input, behavior is exactly Foundation's.

## Shared rule semantics

The `modules` program (validating defaults and rule fields) and the `config` program (validating values) each carry their own implementation of the rule check. Fixture-driven tests assert they agree: every fixture rule's default is accepted by `config`, and a table of good and bad values gives the same verdict through both.
