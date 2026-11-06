# Writing a Lumen module

A module is a directory with a manifest (`module.json`) and a QML entry file. Lumen finds it, checks it, and loads it when you enable it. You never edit Lumen's own files. This guide builds a small module step by step; the finished result is in [`examples/hello-module/`](../examples/hello-module/). The complete interface is in [module-api.md](module-api.md).

## 1. Make the directory

User modules live in `$XDG_DATA_HOME/lumen/modules/` (usually `~/.local/share/lumen/modules/`). Lumen never creates or writes this directory, so create it yourself:

```text
mkdir -p ~/.local/share/lumen/modules/hello
cd ~/.local/share/lumen/modules/hello
```

For development you can point Lumen at another directory with `LUMEN_USER_MODULES_DIR`.

## 2. Write the manifest

`module.json`:

```json
{
  "id": "hello",
  "name": "Hello",
  "version": "1.0.0",
  "api": "1.0",
  "description": "Shows a greeting window that follows the theme.",
  "settings": {
    "greeting": {"type": "string", "default": "Hello", "description": "Text shown in the window"},
    "size": {"type": "integer", "default": 3, "min": 1, "max": 5}
  }
}
```

- `id` is the module's name everywhere: lower case letters, digits and `-`, 2 to 32 characters, unique.
- `api` is the module interface version you built for (`major.minor`). Lumen loads the module if the major number matches and the minor number is not higher than Lumen's. Today that means `"1.0"`.
- `settings` are the options users can change. Each has a type, a default, and optional limits.
- Later you can add `permissions` (what the module will ask Lumen to do for it; see [module-api.md](module-api.md); users see this list before they enable the module) and `helpers` (programs the module ships, see step 5).

The full list of fields and limits is in `specs/002-modules/contracts/module-manifest.md`, and the schema is `data/schema/module.schema.json`.

## 3. Write the entry file

`Main.qml` (the root must be a `Scope`, with one `required property var api`):

```qml
import QtQuick
import Quickshell

Scope {
    required property var api

    Component.onCompleted: api.log.info("hello loaded")

    FloatingWindow {
        title: "Hello"
        color: api.theme.background
        implicitWidth: api.style.spacing.xl * 8
        implicitHeight: api.style.spacing.xl * 4

        Text {
            anchors.centerIn: parent
            text: api.settings.values.greeting ?? "Hello"
            color: api.theme.onBackground
            font.pixelSize: api.style.font.headline
        }
    }
}
```

Rules the example follows, and yours should too:

- Every color comes from `api.theme`; every spacing, radius, font size, and duration from `api.style`. Never write a color or a size as a literal, or the module will not follow the theme.
- Heavy work goes into a helper program, not into QML.
- Everything talks to Lumen through `api` and nothing else.

## 4. Enable it

New user modules are listed but stay off until you enable them, because a module runs with the shell's own privileges. Look at what it asks for first:

```text
lumen modules rescan
lumen modules list
lumen modules show hello
lumen modules enable hello
```

`list` shows `hello` as `disabled` (a module with permissions shows them in the last column). `show` prints everything: the permissions, the settings and their values, and any errors or warnings. `enable` loads it at once (usually well under a second) and keeps it enabled after restarts. `disable` unloads it.

## 5. Add a helper program

Create the program first, then declare it. If the manifest names a helper file that does not exist, the module is listed as failed (`helpers[0].path: helper file bin/hello not found`).

`bin/hello` (make it executable with `chmod +x bin/hello`):

```sh
#!/bin/sh
cat >/dev/null
echo '{"ok":true,"protocol":1,"data":{"message":"hello from the helper"}}'
```

A helper reads one JSON object on stdin and writes one response document on stdout: `{"ok": true, "protocol": 1, "data": ...}` or `{"ok": false, "protocol": 1, "error": {"code": "...", "message": "..."}}`. You can test it alone: `echo '{}' | bin/hello`.

Declare it in `module.json`, together with the permission that allows running helpers (add these two fields next to `settings`):

```json
  "permissions": ["helpers"],
  "helpers": [{"name": "hello", "path": "bin/hello"}]
```

Then run `lumen modules reload hello`. Call it from QML. The module needs the `helpers` permission and the helper must be declared in the manifest:

```qml
Component.onCompleted: api.run("hello", {}, 0, (request) => {
    // request.state is "ok", "failed", "timeout", or "denied"
    if (request.state === "ok")
        label.text = request.data.message;
})
```

The call is asynchronous and has a time limit (`0` uses the default). If the helper fails or hangs, only your module's feature is affected, and the log names your module and the helper.

## 6. Settings

Users change settings in `~/.config/lumen/config.json`, under your module's id:

```json
{"modules": {"hello": {"greeting": "Hi", "size": 4}}}
```

`api.settings.values` updates within a second. A bad value (`"size": 99`) produces an error that names the module and key, and only that key falls back to its default. `api.settings.errors` lists such problems so your module can show them. Your module can also change its own settings with `api.settings.set(key, value)` and `api.settings.reset(key)`.

## 7. Reading errors, and reloading

- A module that cannot be loaded (bad manifest, missing file, a QML syntax error, a newer interface than Lumen has) is listed as `failed` with the reason, and nothing else is affected. `lumen modules show <id>` has the details.
- Runtime errors in your QML (a failing handler or binding) are written to the shell log and counted for your module. If a module logs three within 5 seconds of loading, Lumen unloads it and marks it `failed`; it is never loaded again by itself.
- After fixing a module, run `lumen modules reload <id>`. This re-reads the manifest and the entry file.
- If you only changed a secondary QML file of your module (not `Main.qml`), restart the shell: the engine caches those files.

## 8. The trust model

A module is code that runs inside your shell. Permissions control what the `api` object does for it, and `lumen modules` shows them before you enable anything, but they are not a sandbox. Do not enable modules you do not trust.

## Checklist before you share a module

- [ ] `lumen modules show <id>` has no errors or warnings.
- [ ] Switching themes (`lumen theme tokyo-night`, `lumen theme catppuccin-latte`) changes every color in your module.
- [ ] No color, spacing, font size, or duration is a literal.
- [ ] Interactive elements work from the keyboard.
- [ ] Declared permissions are the ones you really use, and no more.
