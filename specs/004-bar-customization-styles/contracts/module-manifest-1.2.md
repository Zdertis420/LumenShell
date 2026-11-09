# Contract: module manifest, interface 1.2

Interface 1.2 is additive to [1.1](../../003-bar/contracts/module-manifest-1.1.md). A module built for 1.0 or 1.1 loads and behaves unchanged. A module that declares `"api": "1.2"` on a shell that provides only 1.1 is refused with the required and available versions (the rule of Feature II). The version note in `docs/module-api.md` says "additive, no migration".

## Change: unset settings (`default: null`)

A setting rule may give `"default": null`:

```json
"fontSize": {"type": "integer", "default": null, "min": 1, "max": 96, "description": "Text size; unset follows the design token"}
```

| Rule | Behavior |
|------|----------|
| Allowed types | `string`, `integer`, `number`, `boolean`. A `stringlist` default stays a required array |
| Validation | `min`, `max`, `enum`, `minLength` and `maxLength` still apply to every value the user gives. A `default: null` with `enum` is valid (the default need not be in the enum) |
| Effective value | The user's value when set and valid, otherwise unset: `api.settings.values.<key>` is `null` (read it with `??`, which treats `null` and `undefined` alike) |
| User value `null` | An error that names the key with the ordinary type message (for example `modules.<id>.<key>: must be an integer between 1 and 96`); the key stays unset. To unset a key, delete it |
| Merged schema (`lumen modules schema`) | The property has its `type` and rules and **no** `default` |
| `lumen modules show` | Prints `(unset)` for the value of such a key |
| Override-only | Unset keys write nothing; reset removes the override, which returns the key to unset |
| Module use | The module falls back to a documented value, normally a token such as `api.style.spacing.sm`. A module MUST document what unset means for each such key |

The `modules` program accepts the rule and the `config` helper treats an absent key with a null default as having no effective value. The agreement test (`module_rules_agree_test.go`) covers both programs with the new rule.

## The settings of the bundled item modules

The three bundled item modules (`bar-clock`, `bar-sound`, `bar-battery`) declare the same nine settings, in addition to their own: `fontFamily`, `fontSize`, `fontWeight`, `fontItalic`, `padding`, `radius`, `backgroundRole`, `backgroundOpacity`, `textRole`, each with `default: null`, as listed in [data-model.md](../data-model.md). A test fails if the declarations differ between the three manifests, or if a role list differs from `Theme.qml`.

Any module may declare the same keys. There is no special permission and no special treatment; a third-party item that reads them is as customizable as a built-in one (FR-023). `examples/hello-bar-item/` shows it.

## Fixtures

New fixtures under `scripts/tests/fixtures/modules/`: `null-default-good` (one setting of each type with a null default), `err-null-default-stringlist` (a null default on a `stringlist` is invalid), `err-null-default-enum-value` (a user value outside the enum for a null default is reported), and `api12-item` (declares 1.2, used with a 1.1 limit by the scan test).
