import QtQuick
import Quickshell
import Quickshell.Io
pragma Singleton

// Single owner of configuration. Layers: built-in defaults, then the selected preset,
// then the user's overrides (config.json). All merging and validation happens in the
// `config` helper program; this singleton runs it asynchronously and holds the result.
Singleton {
    id: root

    // False until the first result arrives. Until then `values` is empty.
    property bool ready: false
    // Effective nested values, e.g. values.theme.preset.
    property var values: ({
    })
    // Problems found in the user file: [{key, problem, fallback}]. key "" is the whole file.
    property var errors: []
    // Which layer each dotted key came from: default, preset, user, or temp.
    property var sources: ({
    })
    // Theme applied for this shell only (lumen theme --temp). "" means none.
    property string tempTheme: ""
    // Counts resolve runs; a result from an older run is dropped, so the last state wins.
    property int generation: 0
    // LUMEN_CONFIG_FILE overrides the location (tests).
    readonly property string userFile: {
        const override = Quickshell.env("LUMEN_CONFIG_FILE");
        if (override)
            return override;

        const xdg = Quickshell.env("XDG_CONFIG_HOME");
        const base = xdg ? xdg : Quickshell.env("HOME") + "/.config";
        return base + "/lumen/config.json";
    }
    // True while the user file does not exist. The watcher cannot see a file that was
    // never there, so a cheap retry runs until it appears.
    property bool fileMissing: false

    // Emitted after every successfully applied result.
    signal reloaded()

    function value(key: string) : variant {
        let cur = root.values;
        for (const part of key.split(".")) {
            if (cur === null || cur === undefined || typeof cur !== "object")
                return undefined;

            cur = cur[part];
        }
        return cur;
    }

    // Asks the helper to merge and validate. Safe to call at any time.
    function resolve() {
        const gen = ++root.generation;
        Backend.run("config", ["resolve"], {
            "userFile": root.userFile,
            "dataDir": Backend.dataDir,
            "themeOverride": root.tempTheme
        }, 0, (req) => {
            if (gen !== root.generation)
                return ;

            if (req.state === "ok")
                root.apply(req.data);
            else
                root.fallBack(req);
        });
    }

    function apply(data) {
        root.values = data.values;
        root.errors = data.errors;
        root.sources = data.sources;
        root.ready = true;
        Log.info("config resolved", {
            "preset": root.value("theme.preset"),
            "errors": data.errors.length
        });
        const level = root.value("log.level");
        if (level)
            Log.level = level;

        const limit = root.value("helper.timeoutMs");
        if (limit)
            Backend.defaultTimeoutMs = limit;

        root.reloaded();
    }

    // The helper itself failed: keep the previous values and show why.
    function fallBack(req) {
        root.errors = [{
            "key": "",
            "problem": "config helper " + req.state + ": " + (req.error ? req.error.message : ""),
            "fallback": "previous values"
        }];
    }

    // Writes explicit overrides: changes is [{key, value}]. callback(request) is optional.
    function setMany(changes, callback) {
        return Backend.run("config", ["set"], {
            "userFile": root.userFile,
            "dataDir": Backend.dataDir,
            "changes": changes
        }, 0, (req) => {
            if (req.state === "ok")
                root.resolve();

            if (callback)
                callback(req);

        });
    }

    function set(key: string, value, callback) {
        return root.setMany([{
            "key": key,
            "value": value
        }], callback);
    }

    // Removes overrides, which makes the keys fall back to the layer below.
    function resetMany(keys, callback) {
        return Backend.run("config", ["unset"], {
            "userFile": root.userFile,
            "dataDir": Backend.dataDir,
            "keys": keys
        }, 0, (req) => {
            if (req.state === "ok")
                root.resolve();

            if (callback)
                callback(req);

        });
    }

    function reset(key: string, callback) {
        return root.resetMany([key], callback);
    }

    onTempThemeChanged: root.resolve()
    Component.onCompleted: root.resolve()

    // Saved edits are picked up after a short pause; repeated saves collapse into one run.
    Timer {
        id: debounce

        interval: 100
        onTriggered: root.resolve()
    }

    Timer {
        interval: 500
        repeat: true
        running: root.fileMissing
        onTriggered: userFileView.reload()
    }

    FileView {
        id: userFileView

        path: root.userFile
        watchChanges: true
        printErrors: false
        onFileChanged: debounce.restart()
        onLoaded: {
            if (root.fileMissing) {
                root.fileMissing = false;
                debounce.restart();
            }
        }
        onLoadFailed: (error) => {
            // A missing file is normal: defaults apply.
            root.fileMissing = true;
        }
    }

}
