# Contract: module manifest, interface 1.3

Interface 1.3 is additive to [1.2](../../004-bar-customization-styles/contracts/module-manifest-1.2.md). A module built for 1.0, 1.1 or 1.2 loads and behaves unchanged. A module that declares `"api": "1.3"` needs a shell that provides 1.3 and is refused by an older one with the usual message.

## Change: the settings limit

| Rule | 1.2 | 1.3 |
|------|-----|-----|
| Settings per module (`maxProperties` in `data/schema/module.schema.json`, `maxSettings` in `scripts/src/modules/manifest.go`) | 64 | 128 |

Nothing else changes: the key pattern (`^[a-z][A-Za-z0-9]{0,31}$`), the types, `default: null`, validation and the merged schema are as in 1.2.

## Where it is applied

- `data/schema/module.schema.json`: `maxProperties` 128, and the `api` value `1.3` is accepted.
- `scripts/src/modules/manifest.go`: `maxSettings = 128`.
- `singletons/Modules.qml`: `apiVersion` is `"1.3"`.
- The agreement test (`module_rules_agree_test.go`, `schema_sync_test.go`) keeps the two numbers equal.

## Fixtures and tests

`scripts/tests/fixtures/modules/settings-128-good` (128 keys, loads) and `settings-129-bad` (129 keys, refused with an error that names the limit). A module with `"api": "1.3"` is refused on a 1.2 rules fixture. `docs/module-api.md` documents 1.3 with the line "No migrations: version 1.3 only raised a limit."

## Migration

None. No existing module needs to change.
