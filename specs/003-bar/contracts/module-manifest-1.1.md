# Contract: module manifest additions (interface 1.1)

Additions to [specs/002-modules/contracts/module-manifest.md](../../002-modules/contracts/module-manifest.md). The published schema `data/schema/module.schema.json` is updated to match; the `modules` program validates with its own rules, and a test keeps both in step.

## New and changed fields

```json
{
  "id": "bar",
  "name": "Bar",
  "version": "1.0.0",
  "api": "1.1",
  "extensionPoints": [
    {"id": "items", "description": "Items shown in the bar"}
  ],
  "commands": ["focus"],
  "settings": {
    "left":   {"type": "stringlist", "default": [], "maxItems": 64},
    "center": {"type": "stringlist", "default": ["bar-clock"]}
  }
}
```

```json
{
  "id": "hello-bar-item",
  "name": "Hello bar item",
  "version": "1.0.0",
  "api": "1.1",
  "permissions": ["contribute:bar"],
  "contributions": [
    {"id": "greeting", "name": "Greeting", "target": {"module": "bar", "point": "items"}, "file": "Item.qml"}
  ]
}
```

| Field | Rules |
|-------|-------|
| `extensionPoints` | Optional. Up to 8 entries `{id, description}`. `id` is `^[a-z][a-z0-9-]{0,31}$` and unique in the module. `description` is up to 160 characters |
| `contributions` | Optional. Up to 16 entries `{id, name, target, file}`. `id` follows the same pattern and is unique in the module. `name` is 1 to 64 characters. `target` is `{module, point}`; `module` is a module id and `point` a point id (neither may contain `/`). `file` is a relative path inside the module directory, ending in `.qml`, and the file must exist |
| `commands` | Optional. Up to 16 names, `^[a-z][a-z0-9-]{0,31}$`, no duplicates |
| `permissions` | Gains the form `contribute:<module-id>` (the id follows the module id pattern). Counts toward the limit of 16 |
| setting `type` | Gains `stringlist`: `default` is an array of strings (each 1 to 80 characters); optional `maxItems` (1 to 64, default 64); `enum`, `min`, `max` are not allowed |
| setting rule fields | `string` settings gain optional `minLength` and `maxLength` (integers from 0 to 4096, `minLength` not above `maxLength`; the default must satisfy them). They are not allowed on other types. Used by the clock's `format` (1 to 64) and the sound item's `mixer` (0 to 128) |
| `requires` | Still reserved: accepted with any value, ignored, and reported as a warning `requires is not supported in module interface 1.1` |

## Validation (at scan, whether or not the module is enabled)

- Every contribution's `target.module` MUST appear as `contribute:<module>` in `permissions`; otherwise the module is invalid with `{field: "contributions[i].target.module", problem: "permission contribute:<module> is not declared"}`.
- A contribution to a module or point that does not exist is **not** a manifest error: the target may be installed later. It is reported at run time as `waiting` or `rejected`.
- Duplicate extension point ids, duplicate contribution ids, a missing contribution file, or a file path that leaves the module directory make the manifest invalid, with the offending field and the other entry named.
- A module may contribute to its own extension point. This is allowed and needs no permission entry.
- A module that is invalid is never loaded; its extension points and contributions are not registered.

## Compatibility

- `api` is checked as before: the major numbers must be equal and the module's minor must not be higher than Lumen's. A 1.0 module loads on 1.1 unchanged. A module with `"api": "1.1"` on a 1.0 shell is refused with `requires interface 1.1, Lumen provides 1.0` (the existing rule).
- A module with `"api": "1.0"` that uses `extensionPoints`, `contributions` or `commands` is accepted: the fields take effect. The version only says what the module needs at minimum; authors who use them SHOULD declare 1.1, and the documentation says so.
- Adding these optional fields and the permission form is a minor change. No migration note is needed.

## Limits and errors

Unchanged from Modules (64 KiB manifest, 16 permissions, 64 settings, 8 helpers). The `stringlist` default and every stored value is limited to `maxItems` (at most 64) entries of at most 80 characters. In the manifest a bad default makes the module invalid; in a stored value a bad entry is dropped on its own (see [helper-bar.md](helper-bar.md)).
