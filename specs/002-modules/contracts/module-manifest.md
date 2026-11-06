# Contract: module manifest (`module.json`)

Published as `data/schema/module.schema.json` (JSON Schema, draft-07). The `modules` program validates with its own compiled rules; a test keeps both in sync.

## Location

A module is a directory that contains `module.json` and the entry file. Lumen looks in:

- `<root>/modules/<dir>/` (bundled, origin `bundled`)
- `$XDG_DATA_HOME/lumen/modules/<dir>/` (default `~/.local/share/lumen/modules/<dir>/`, origin `user`; overridable with `LUMEN_USER_MODULES_DIR`)

Hidden directories (starting with `.`) and plain files are skipped. The directory name does not have to equal the id. Lumen never creates or writes into either directory.

## Fields

```json
{
  "id": "hello",
  "name": "Hello",
  "version": "1.0.0",
  "api": "1.0",
  "description": "Shows a greeting that follows the theme.",
  "main": "Main.qml",
  "permissions": ["helpers"],
  "settings": {
    "greeting": {"type": "string", "default": "Hello", "description": "Text shown in the window"},
    "size": {"type": "integer", "default": 3, "min": 1, "max": 5}
  },
  "helpers": [{"name": "hello", "path": "bin/hello"}]
}
```

| Field | Rules |
|-------|-------|
| `id` | Required. `^[a-z][a-z0-9-]{1,31}$`. Unique across both directories |
| `name` | Required. 1–64 characters |
| `version` | Required. `MAJOR.MINOR.PATCH`, numbers only |
| `api` | Required. `MAJOR.MINOR`, the interface version the module was built for |
| `description` | Optional. Up to 256 characters |
| `main` | Optional, default `Main.qml`. A relative path inside the module directory, ending in `.qml`; the file must exist |
| `permissions` | Optional. Each entry from the list in [data-model.md](../data-model.md); no duplicates; up to 16 |
| `settings` | Optional. Key → setting rule; up to 64; rule fields in [data-model.md](../data-model.md); the default must satisfy the rule; `enabled` is reserved |
| `helpers` | Optional. Up to 8 entries of `{name, path}`. `name` is `^[a-z][a-z0-9-]{0,31}$` and unique; `path` is relative, stays inside the module directory (no `..`, no absolute paths), and the file must be executable |
| `extensionPoints`, `contributions`, `requires` | Reserved. Accepted with any value, ignored, and reported as a warning: `<field> is not supported in module interface 1.0` |

Any other unknown top-level field is a warning (`unknown field`), not an error, so manifests written for a later minor version keep loading.

## Limits and errors

- A manifest larger than 64 KiB is rejected (`manifest is larger than 64 KiB`).
- Every problem is reported as `{field, problem}` together with the module id (or `dir:<name>` when the id is unusable). Examples: `{"field":"api","problem":"requires interface 2.0, Lumen provides 1.0"}`, `{"field":"id","problem":"duplicate of the bundled module in /…/modules/hello"}`.
- Duplicate ids: the bundled module wins. The other module is invalid, and its error names both directories. Two user modules with the same id: the one whose directory name sorts first wins.
- A module that is invalid is never loaded, whether or not it is enabled.

## Interface compatibility

`api` is checked against the interface version that Lumen provides (currently `1.0`): the major numbers must be equal, and the module's minor number must not be higher.

## Stability

Adding an optional field or permission is a minor change. Removing a field, changing its meaning, or making an optional field required is a major change and needs a migration note in `docs/module-api.md`.
