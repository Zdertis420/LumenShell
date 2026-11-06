// The object a module receives as `api` (module interface 1.0, specs/002-modules/contracts/
// module-api.md). One instance per loaded module. The Modules singleton creates it and
// sets the properties below; a module only ever sees the members documented there.

import QtQuick
import Quickshell

// Calls that need a permission are checked here. A refused call logs a warning with the
// module id, returns a harmless result, and leaves the module running.
QtObject {
    // ---- permissions ----
    // ---- logging ----
    // ---- this module's own settings (no permission needed) ----
    // ---- the module list (read-only) ----
    // ---- core configuration (permissions config.read and config.write) ----
    // ---- helper programs (permission helpers, and the helper must be declared) ----

    id: api

    // Set by Modules when the object is created.
    property string moduleId: ""
    property string moduleDir: ""
    property var manifest: ({
    })
    property string apiVersion: ""
    property var host: null
    readonly property string id: moduleId
    readonly property string name: manifest.name ?? moduleId
    readonly property string version: manifest.version ?? ""
    // Theme roles (live) and design tokens: always allowed.
    readonly property var theme: Theme
    readonly property var style: Style
    // Private state.
    property var pending: []
    property bool alive: true
    readonly property QtObject
    log: QtObject {
        function debug(msg, fields) {
            Log.write("debug", api.source(), String(msg), fields);
        }

        function info(msg, fields) {
            Log.write("info", api.source(), String(msg), fields);
        }

        function warn(msg, fields) {
            Log.write("warn", api.source(), String(msg), fields);
        }

        function error(msg, fields) {
            Log.write("error", api.source(), String(msg), fields);
        }

        // The recent shared log entries (permission log.read).
        function recent() : variant {
            if (!api.has("log.read")) {
                api.deny("api.log.recent", "log.read");
                return [];
            }
            return Log.recent();
        }

    }

    readonly property QtObject
    settings: QtObject {
        // Resolved values of the declared settings, defaults filled in. Reactive.
        readonly property var values: {
            const all = Config.value("modules." + api.moduleId) ?? {
            };
            const out = {
            };
            for (const k in all) {
                if (k !== "enabled")
                    out[k] = all[k];

            }
            return out;
        }
        // Problems with this module's settings: [{key, problem, fallback}]. Reactive.
        readonly property var errors: {
            const prefix = "modules." + api.moduleId + ".";
            const out = [];
            for (const e of Config.errors) {
                if (e.key.indexOf(prefix) === 0)
                    out.push({
                    "key": e.key.substring(prefix.length),
                    "problem": e.problem,
                    "fallback": e.fallback
                });

            }
            return out;
        }

        function declared(key: string) : bool {
            const rules = api.manifest.settings ?? {
            };
            return rules[key] !== undefined;
        }

        // Saves an override for one of this module's own settings.
        function set(key: string, value, callback) {
            if (!declared(key)) {
                api.log.warn("setting '" + key + "' is not declared in the manifest");
                if (callback)
                    callback({
                    "state": "failed",
                    "error": {
                        "code": "invalid_key",
                        "message": "setting '" + key + "' is not declared"
                    }
                });

                return false;
            }
            Config.set("modules." + api.moduleId + "." + key, value, callback);
            return true;
        }

        // Removes the override of one of this module's own settings.
        function reset(key: string, callback) {
            if (!declared(key)) {
                api.log.warn("setting '" + key + "' is not declared in the manifest");
                if (callback)
                    callback({
                    "state": "failed",
                    "error": {
                        "code": "invalid_key",
                        "message": "setting '" + key + "' is not declared"
                    }
                });

                return false;
            }
            Config.reset("modules." + api.moduleId + "." + key, callback);
            return true;
        }

    }

    readonly property QtObject
    modules: QtObject {
        readonly property var list: api.host ? api.host.publicList() : []

        // Another module's resolved settings (permission modules.read-settings).
        function settingsOf(id: string) : variant {
            if (!api.has("modules.read-settings")) {
                api.deny("api.modules.settingsOf", "modules.read-settings");
                return undefined;
            }
            return Config.value("modules." + id);
        }

    }

    readonly property QtObject
    config: QtObject {
        // Core configuration without the modules subtree. Reactive.
        readonly property var values: {
            // Without config.read this is empty. It is not logged: a binding is evaluated
            // when the object is created, whether or not the module ever looks at it.
            if (!api.has("config.read"))
                return {
            };

            const out = {
            };
            for (const k in Config.values) {
                if (k !== "modules")
                    out[k] = Config.values[k];

            }
            return out;
        }
        readonly property var errors: {
            if (!api.has("config.read"))
                return [];

            return Config.errors.filter((e) => {
                return e.key.indexOf("modules.") !== 0;
            });
        }

        function value(key: string) : variant {
            if (!api.has("config.read")) {
                api.deny("api.config.value", "config.read");
                return undefined;
            }
            if (key === "modules" || key.indexOf("modules.") === 0)
                return undefined;

            return Config.value(key);
        }

        function set(key: string, value, callback) {
            if (!api.has("config.write")) {
                api.deny("api.config.set", "config.write");
                return false;
            }
            if (key === "modules" || key.indexOf("modules.") === 0) {
                api.log.warn("api.config.set cannot change module settings; use api.settings");
                return false;
            }
            Config.set(key, value, callback);
            return true;
        }

        function reset(key: string, callback) {
            if (!api.has("config.write")) {
                api.deny("api.config.reset", "config.write");
                return false;
            }
            if (key === "modules" || key.indexOf("modules.") === 0) {
                api.log.warn("api.config.reset cannot change module settings; use api.settings");
                return false;
            }
            Config.reset(key, callback);
            return true;
        }

    }

    function has(permission: string) : bool {
        const list = api.manifest.permissions ?? [];
        return list.indexOf(permission) >= 0;
    }

    function source() : string {
        return "module:" + api.moduleId;
    }

    // Logs a refused function call (every time).
    function deny(call: string, permission: string) {
        Log.write("warn", api.source(), "module " + api.moduleId + " was refused " + call + ": permission " + permission + " not declared", {
            "call": call,
            "permission": permission
        });
    }

    // Runs one of the module's own declared helper programs. callback(request) is called
    // once; request.state is "ok", "failed", "timeout", or "denied".
    function run(helper: string, input, timeoutMs: int, callback) {
        if (!api.has("helpers")) {
            api.deny("api.run", "helpers");
            return api.denied(helper, "permission helpers not declared", callback);
        }
        let decl = null;
        for (const h of (api.manifest.helpers ?? [])) {
            if (h.name === helper)
                decl = h;

        }
        if (decl === null) {
            Log.write("warn", api.source(), "module " + api.moduleId + " was refused api.run: helper '" + helper + "' is not declared in the manifest", {
                "call": "api.run",
                "helper": helper
            });
            return api.denied(helper, "helper '" + helper + "' is not declared in the manifest", callback);
        }
        const req = Backend.run(api.moduleDir + "/" + decl.path, [], input, timeoutMs, (done) => {
            // The request is destroyed after this callback; keep a plain copy.
            const result = {
                "state": done.state,
                "helper": helper,
                "data": done.data,
                "error": done.error,
                "durationMs": done.durationMs
            };
            api.pending = api.pending.filter((p) => {
                return p !== done;
            });
            try {
                if (callback)
                    callback(result);

            } catch (e) {
                api.log.error("callback for helper '" + helper + "' threw: " + e);
            }
        }, api.source());
        req.helper = helper;
        api.pending = api.pending.concat([req]);
        return req;
    }

    function denied(helper: string, message: string, callback) : variant {
        const result = {
            "state": "denied",
            "helper": helper,
            "data": null,
            "error": {
                "code": "denied",
                "message": message
            }
        };
        if (callback)
            Qt.callLater(() => {
            if (api.alive)
                callback(result);

        });

        return result;
    }

    // Called by Modules before the module is destroyed: kills pending helper runs without
    // calling their callbacks.
    function cancelAll() {
        api.alive = false;
        for (const req of api.pending) {
            try {
                req.cancel();
            } catch (e) {
            }
        }
        api.pending = [];
    }

}
