import QtQuick
import Quickshell
import Quickshell.Io
pragma Singleton

// Single owner of the module registry: which modules exist, whether each is enabled,
// loaded, disabled, or failed, and why. Scanning and manifest validation happen in the
// `modules` helper program; this singleton runs it, keeps the result, and loads and
// unloads the enabled modules one at a time. See specs/002-modules/.
Singleton {
    // A failed module is never loaded again by itself; reload or enable clears this.
    // ---- runtime errors: counted by following the shell's own log ----
    // A module's runtime errors (a failing handler or binding) only reach the log:
    // "WARN scene: file:///<module dir>/Main.qml[line:col]: TypeError: ...". The log is
    // followed with `qs log --follow` and each such line is attributed to the module whose
    // directory the file is in. Lines the shell writes itself (they start with "{") are
    // ignored, so the follower cannot feed on its own output.
    // ---- reload ----
    // ---- enabling and disabling (saved in the config file, FR-023) ----
    // ---- views for the controller ----

    id: root

    // The module interface version Lumen provides (major.minor). 1.1 added extension points,
    // contributions, module commands, and the stringlist setting type (specs/003-bar). 1.2 added
    // unset settings (a null default) and the bar's placement facts (specs/004-bar-customization-styles).
    readonly property string apiVersion: "1.2"
    // Module records (plain objects, see specs/002-modules/data-model.md). Replaced as a
    // whole on every change, so bindings update.
    property var list: []
    // True after the first scan has been applied.
    property bool ready: false
    // "active" while runtime errors of modules are counted; "degraded" when they cannot be.
    property string attribution: "active"
    readonly property string bundledDir: Quickshell.shellDir + "/modules"
    // LUMEN_USER_MODULES_DIR overrides the location. Lumen never creates or writes this directory.
    readonly property string userDir: {
        const override = Quickshell.env("LUMEN_USER_MODULES_DIR");
        if (override)
            return override;

        const xdg = Quickshell.env("XDG_DATA_HOME");
        return (xdg ? xdg : Quickshell.env("HOME") + "/.local/share") + "/lumen/modules";
    }
    // True once Config knows the module definitions (until then `enabled` is unknown).
    property bool defsApplied: false
    property int loadCounter: 0
    // id -> {component, instance, api} for loaded modules.
    property var live: ({
    })
    property bool followerStarted: false
    property int followerFailures: 0
    property double followerStart: 0
    // True once the follower has printed a line. A follower that exits before that (the shell is
    // not registered with `qs log` yet, early in start-up) is restarted from the start of the
    // log, so runtime errors logged in the meantime are not missed.
    property bool followerHadOutput: false
    // ---- interface 1.1: extension points and contributions ----
    // Bumped whenever an item is created, destroyed, or refused, so bindings that read
    // contribution states update.
    readonly property int resultLimit: 4096
    // documented constant: bytes of a command's answer
    property int itemRevision: 0
    // Items created through createItem: [{key, hostId, pointId, contributorId, object, component}].
    // A host may destroy an item's parent, so entries whose object is gone are skipped on read.
    property var items: []
    // Refusals by hosts: "<host>|<point>|<key>" -> reason (cleared by the host's next resolve).
    property var rejects: ({
    })
    // Reports published by hosts: "<host>|<point>" -> {warnings, dropped}.
    property var reports: ({
    })

    function find(id: string) : variant {
        for (const rec of root.list) {
            if (rec.id === id)
                return rec;

        }
        return null;
    }

    // Tells bindings that a record changed.
    function touch() {
        root.list = root.list.slice();
    }

    // Problems with a module's settings, from the config helper: [{key, problem, fallback}]
    // with the key relative to the module (for example "size").
    function settingErrorsOf(id: string) : variant {
        const prefix = "modules." + id + ".";
        const out = [];
        for (const e of Config.errors) {
            if (e.key.indexOf(prefix) === 0)
                out.push({
                "key": e.key.substring(prefix.length),
                "problem": e.problem,
                "fallback": e.fallback,
                "dropped": e.dropped === true,
                "value": e.value
            });

        }
        return out;
    }

    // The read-only view modules get through api.modules.list: copies, no internals.
    function publicList() : variant {
        const out = [];
        for (const rec of root.list) {
            out.push({
                "id": rec.id,
                "name": rec.name,
                "version": rec.version,
                "origin": rec.origin,
                "state": rec.state,
                "enabled": rec.enabled,
                "manifest": rec.manifest,
                "permissions": rec.permissions,
                "errors": rec.errors,
                "warnings": rec.warnings,
                "settingErrors": root.settingErrorsOf(rec.id),
                "runtimeErrors": rec.runtimeErrors,
                "lastError": rec.lastError,
                "extensionPoints": rec.extensionPoints,
                "contributions": rec.contributions,
                "contributionStates": root.contributionStatesOf(rec)
            });
        }
        return out;
    }

    function firstError(errors) : string {
        if (!errors || errors.length === 0)
            return "";

        const e = errors[0];
        return (e.field ? e.field + ": " : "") + e.problem;
    }

    function makeRecord(m, old) : variant {
        const mf = m.manifest ?? {
        };
        return {
            "id": m.id,
            "dir": m.dir,
            "dirName": m.dirName,
            "origin": m.origin,
            "name": mf.name ?? m.id,
            "version": mf.version ?? "",
            "description": mf.description ?? "",
            "manifest": mf,
            "main": m.main ?? "",
            "valid": m.valid,
            "errors": m.errors ?? [],
            "warnings": m.warnings ?? [],
            "permissions": mf.permissions ?? [],
            "defaultEnabled": m.defaultEnabled,
            "enabled": old ? old.enabled : m.defaultEnabled,
            "state": old ? old.state : (m.valid ? "disabled" : "failed"),
            "lastError": old ? old.lastError : (m.valid ? "" : root.firstError(m.errors)),
            "runtimeErrors": old ? old.runtimeErrors : 0,
            "loadedAt": old ? old.loadedAt : 0,
            "blocked": old ? old.blocked : !m.valid,
            "extensionPoints": m.extensionPoints ?? [],
            "contributions": m.contributions ?? [],
            "commands": m.commands ?? []
        };
    }

    // ---- contribution states (specs/003-bar/data-model.md) ----
    // One of: shown, available, waiting, rejected, failed. Reads itemRevision, so a binding
    // that calls this follows item creation and refusals as well as module state changes.
    function contributionState(owner, c) : variant {
        root.itemRevision;
        if (owner.state === "failed")
            return {
            "state": "failed",
            "reason": owner.lastError
        };

        if (owner.state !== "loaded")
            return {
            "state": "waiting",
            "reason": "module " + owner.id + " is not enabled"
        };

        const host = root.find(c.target.module);
        if (host === null)
            return {
            "state": "waiting",
            "reason": "module " + c.target.module + " is not installed"
        };

        if (host.state !== "loaded")
            return {
            "state": "waiting",
            "reason": "module " + host.id + " is not loaded"
        };

        const declares = (host.extensionPoints ?? []).some((p) => {
            return p.id === c.target.point;
        });
        if (!declares)
            return {
            "state": "rejected",
            "reason": "module " + host.id + " declares no extension point '" + c.target.point + "'"
        };

        if (owner.id !== host.id && owner.permissions.indexOf("contribute:" + host.id) < 0)
            return {
            "state": "rejected",
            "reason": "permission contribute:" + host.id + " is not declared"
        };

        const refusal = root.rejects[host.id + "|" + c.target.point + "|" + c.key];
        if (refusal !== undefined)
            return {
            "state": "rejected",
            "reason": refusal
        };

        const shown = root.items.some((it) => {
            return it.key === c.key && it.object !== null && it.object !== undefined;
        });
        return {
            "state": shown ? "shown" : "available",
            "reason": ""
        };
    }

    function contributionStatesOf(rec) : variant {
        return (rec.contributions ?? []).map((c) => {
            const s = root.contributionState(rec, c);
            return {
                "id": c.id,
                "state": s.state,
                "reason": s.reason
            };
        });
    }

    // The contributions that target one extension point, with their states.
    function contributionsTo(moduleId: string, pointId: string) : variant {
        const out = [];
        for (const rec of root.list) {
            for (const c of (rec.contributions ?? [])) {
                if (c.target.module !== moduleId || c.target.point !== pointId)
                    continue;

                const s = root.contributionState(rec, c);
                out.push({
                    "key": c.key,
                    "short": c.short ?? "",
                    "module": rec.id,
                    "item": c.id,
                    "name": c.name,
                    "state": s.state,
                    "reason": s.reason
                });
            }
        }
        return out;
    }

    function findContribution(key: string) : variant {
        for (const rec of root.list) {
            for (const c of (rec.contributions ?? [])) {
                if (c.key === key)
                    return {
                    "owner": rec,
                    "contribution": c
                };

            }
        }
        return null;
    }

    // The definitions of the modules, as the config helper wants them. An invalid module
    // is included without settings, so the user can still enable or disable it. A module
    // without a usable id ("dir:...") cannot be named in the config file and is left out.
    function defs() : variant {
        const out = [];
        for (const rec of root.list) {
            if (rec.id.indexOf("dir:") === 0)
                continue;

            out.push({
                "id": rec.id,
                "defaultEnabled": rec.defaultEnabled,
                "settings": rec.valid ? (rec.manifest.settings ?? {
                }) : {
                }
            });
        }
        return out;
    }

    // Enables or disables a module. The choice is written as an override only when it
    // differs from the module's default, and removed again when it matches. done(result)
    // is called after the module's state has settled; result is {ok, unknown, message}.
    function setEnabled(id: string, enabled: bool, done) {
        const rec = root.find(id);
        if (rec === null) {
            done({
                "ok": false,
                "unknown": true,
                "message": "unknown module '" + id + "'"
            });
            return ;
        }
        if (rec.id.indexOf("dir:") === 0) {
            done({
                "ok": false,
                "message": "the module has no usable id; fix its manifest first"
            });
            return ;
        }
        if (enabled) {
            // The user asks for it: try again even if it failed before.
            rec.blocked = false;
            if (rec.valid && rec.state === "failed")
                rec.state = "disabled";

        }
        const key = "modules." + id + ".enabled";
        const finished = (result) => {
            if (result.state === "ok")
                done({
                "ok": true
            });
            else
                done({
                "ok": false,
                "message": result.error ? result.error.message : "could not save the choice"
            });
        };
        if (enabled === rec.defaultEnabled)
            Config.reset(key, finished);
        else
            Config.set(key, enabled, finished);
    }

    function summaryOf(rec) : variant {
        return {
            "id": rec.id,
            "name": rec.name,
            "version": rec.version,
            "origin": rec.origin,
            "state": rec.state,
            "enabled": rec.enabled,
            "defaultEnabled": rec.defaultEnabled,
            "permissions": rec.permissions,
            "runtimeErrors": rec.runtimeErrors,
            "lastError": rec.lastError,
            "warnings": rec.warnings.map((w) => {
                return (w.field ? w.field + ": " : "") + w.problem;
            }),
            "notes": root.notesOf(rec)
        };
    }

    // Short labels for `lumen modules list`: what the module hosts and contributes to.
    function notesOf(rec) : variant {
        const notes = [];
        const targets = [];
        for (const c of (rec.contributions ?? [])) {
            if (targets.indexOf(c.target.module) < 0)
                targets.push(c.target.module);

        }
        if (targets.length > 0)
            notes.push("contributes to " + targets.join(", "));

        const points = (rec.extensionPoints ?? []).map((p) => {
            return p.id;
        });
        if (points.length > 0)
            notes.push("hosts: " + points.join(", "));

        return notes;
    }

    function detailOf(rec) : variant {
        const detail = root.summaryOf(rec);
        const values = Config.value("modules." + rec.id) ?? {
        };
        const settingValues = {
        };
        for (const k in values) {
            if (k !== "enabled")
                settingValues[k] = values[k];

        }
        detail.description = rec.description;
        detail.dir = rec.dir;
        detail.apiRequired = rec.manifest.api ?? "";
        detail.apiProvided = root.apiVersion;
        detail.errors = rec.errors;
        detail.warnings = rec.warnings;
        detail.settings = rec.manifest.settings ?? {
        };
        detail.settingValues = settingValues;
        detail.settingErrors = root.settingErrorsOf(rec.id);
        detail.helpers = (rec.manifest.helpers ?? []).map((h) => {
            return h.name;
        });
        // Interface 1.1: extension points with their contributions, own contributions,
        // commands, and what a host reported about its points (specs/003-bar/contracts/cli-ipc-bar.md).
        detail.commands = rec.commands ?? [];
        detail.contributions = (rec.contributions ?? []).map((c) => {
            const s = root.contributionState(rec, c);
            return {
                "key": c.key,
                "short": c.short ?? "",
                "name": c.name,
                "target": c.target,
                "state": s.state,
                "reason": s.reason
            };
        });
        detail.extensionPoints = (rec.extensionPoints ?? []).map((p) => {
            return {
                "id": p.id,
                "description": p.description ?? "",
                "contributions": root.contributionsTo(rec.id, p.id)
            };
        });
        let dropped = [];
        let itemWarnings = [];
        let lookWarnings = [];
        for (const p of (rec.extensionPoints ?? [])) {
            const report = root.reports[rec.id + "|" + p.id];
            if (report) {
                dropped = dropped.concat(report.dropped ?? []);
                itemWarnings = itemWarnings.concat(report.warnings ?? []);
                lookWarnings = lookWarnings.concat(report.lookWarnings ?? []);
            }
        }
        detail.dropped = dropped;
        detail.itemWarnings = itemWarnings;
        // Interface 1.2: warnings about the host's own look settings ([{key, problem}]).
        detail.lookWarnings = lookWarnings;
        return detail;
    }

    // Scans both directories (in the helper), updates the records, gives Config the
    // definitions, and applies the enabled states. done(ok) is called afterwards.
    function scan(done) {
        Backend.run("modules", ["scan"], {
            "bundledDir": root.bundledDir,
            "userDir": root.userDir,
            "apiVersion": root.apiVersion
        }, 0, (req) => {
            if (req.state !== "ok") {
                Log.error("module scan " + req.state + ": " + (req.error ? req.error.message : ""));
                root.ready = true;
                if (done)
                    done(false);

                return ;
            }
            // Follow the log before the first module loads, so no error is missed.
            root.startFollower();
            root.applyScan(req.data.modules);
            root.logScanProblems();
            root.defsApplied = true;
            Config.setModules(root.defs(), () => {
                root.applyEnabled();
                root.ready = true;
                Log.info("modules scanned", root.summary());
                if (done)
                    done(true);

            });
        });
    }

    function summary() : variant {
        const counts = {
            "loaded": 0,
            "disabled": 0,
            "failed": 0
        };
        for (const rec of root.list) counts[rec.state] = (counts[rec.state] ?? 0) + 1
        return {
            "modules": root.list.length,
            "loaded": counts.loaded,
            "disabled": counts.disabled,
            "failed": counts.failed
        };
    }

    // Replaces the records with the scan result. A loaded module keeps its record (it
    // stays loaded until reloaded); a loaded module whose directory vanished stays listed.
    function applyScan(mods) {
        const next = [];
        const seen = {
        };
        for (const m of mods) {
            const old = root.find(m.id);
            seen[m.id] = true;
            if (old && old.state === "loaded")
                next.push(old);
            else
                next.push(root.makeRecord(m, old));
        }
        for (const rec of root.list) {
            if (!seen[rec.id] && rec.state === "loaded") {
                const has = rec.warnings.some((w) => {
                    return w.problem === "directory missing";
                });
                if (!has)
                    rec.warnings = rec.warnings.concat([{
                    "field": "",
                    "problem": "directory missing"
                }]);

                next.push(rec);
            }
        }
        root.list = next;
    }

    // Brings every module in line with modules.<id>.enabled. A problem with one module
    // never stops the others.
    function applyEnabled() {
        for (const rec of root.list) {
            try {
                root.applyOne(rec);
            } catch (e) {
                root.fail(rec, "unexpected error: " + String(e.message ?? e));
            }
        }
        root.touch();
    }

    function applyOne(rec) {
        const want = Config.value("modules." + rec.id + ".enabled");
        rec.enabled = want === undefined ? rec.defaultEnabled : want;
        if (!rec.valid)
            return ;

        if (rec.enabled) {
            if (rec.state !== "loaded" && !rec.blocked)
                root.load(rec);

        } else if (rec.state === "loaded") {
            root.unload(rec);
            rec.state = "disabled";
        } else if (rec.state === "failed") {
            // Disabled by the user: no longer "failed", ready to be enabled later.
            rec.state = "disabled";
            rec.blocked = false;
        }
    }

    // Writes one log line per problem the scan found, with the module id as the source.
    function logScanProblems() {
        for (const rec of root.list) {
            if (rec.state === "loaded")
                continue;

            for (const e of rec.errors) root.writeLog("error", rec, "invalid: " + (e.field ? e.field + ": " : "") + e.problem, null)
            for (const w of rec.warnings) root.writeLog("warn", rec, "warning: " + (w.field ? w.field + ": " : "") + w.problem, null)
        }
    }

    function startFollower() {
        if (root.followerStarted)
            return ;

        root.followerStarted = true;
        root.runFollower(false);
    }

    function runFollower(resume: bool) {
        // After a restart only new lines matter (the log is read from its start otherwise).
        follower.command = ["qs", "-p", Quickshell.shellDir, "log", "--follow"].concat(resume ? ["--tail", "1"] : []);
        root.followerStart = Date.now();
        follower.running = true;
    }

    function followerExited() {
        // An exit soon after the start counts as a failure; three in a row stop the attempts.
        root.followerFailures = Date.now() - root.followerStart < 10000 ? root.followerFailures + 1 : 0;
        if (root.followerFailures >= 3) {
            root.attribution = "degraded";
            Log.warn("modules: cannot follow the shell log; runtime errors of modules will not be counted");
            return ;
        }
        restartFollower.start();
    }

    function onLogLine(line: string) {
        const clean = line.replace(/\x1b\[[0-9;]*m/g, "");
        const m = /^\s*(WARN|ERROR)\s+[A-Za-z0-9_.]+:\s+(.*)$/.exec(clean);
        if (!m)
            return ;

        const body = m[2];
        if (body.charAt(0) === "{")
            return ;

        const f = /file:\/\/(\/[^\s\[\]]+?\.qml)/.exec(body);
        if (!f)
            return ;

        let owner = null;
        for (const rec of root.list) {
            if (f[1].indexOf(rec.dir + "/") === 0 && (owner === null || rec.dir.length > owner.dir.length))
                owner = rec;

        }
        if (owner !== null && owner.state === "loaded")
            root.noteRuntimeError(owner, body);

    }

    function noteRuntimeError(rec, text: string) {
        const brief = text.split("file://" + rec.dir + "/").join("").slice(0, 200);
        rec.runtimeErrors += 1;
        rec.lastError = brief;
        root.writeLog("warn", rec, "runtime error: " + brief, {
            "count": rec.runtimeErrors
        });
        if (Date.now() - rec.loadedAt <= 5000)
            rec.earlyErrors = (rec.earlyErrors ?? 0) + 1;

        if ((rec.earlyErrors ?? 0) >= 3) {
            root.unload(rec);
            rec.state = "failed";
            rec.blocked = true;
            rec.lastError = "3 runtime errors within 5 s of loading; unloaded (last: " + brief + ")";
            root.writeLog("error", rec, "failed: " + rec.lastError, null);
        }
        root.touch();
    }

    // Re-reads one module's manifest and loads it again if it is enabled and valid.
    // done(result) gets {ok: bool, unknown: bool} after the module's state has settled.
    function reload(id: string, done) {
        const rec = root.find(id);
        if (rec === null) {
            if (done)
                done({
                "ok": false,
                "unknown": true
            });

            return ;
        }
        if (rec.state === "loaded") {
            root.unload(rec);
            rec.state = "disabled";
        }
        const dir = rec.dir;
        Backend.run("modules", ["scan"], {
            "bundledDir": root.bundledDir,
            "userDir": root.userDir,
            "apiVersion": root.apiVersion
        }, 0, (req) => {
            const current = root.find(id);
            if (req.state !== "ok" || current === null) {
                if (done)
                    done({
                    "ok": false,
                    "unknown": current === null
                });

                return ;
            }
            const fresh = req.data.modules.find((x) => {
                return x.dir === dir;
            });
            const next = root.list.slice();
            const at = next.indexOf(current);
            if (fresh === undefined) {
                current.state = "failed";
                current.valid = false;
                current.blocked = true;
                current.lastError = "module directory is missing";
                root.writeLog("error", current, "failed: " + current.lastError, null);
            } else {
                next[at] = root.makeRecord(fresh, null);
                root.list = next;
                for (const e of next[at].errors) root.writeLog("error", next[at], "invalid: " + (e.field ? e.field + ": " : "") + e.problem, null)
            }
            root.touch();
            Config.setModules(root.defs(), () => {
                root.applyEnabled();
                if (done)
                    done({
                    "ok": true,
                    "unknown": false
                });

            });
        });
    }

    function writeLog(level: string, rec, msg: string, fields) {
        Log.write(level, "module:" + rec.id, msg, fields);
    }

    function fail(rec, msg: string) {
        rec.state = "failed";
        rec.lastError = msg;
        rec.blocked = true;
        root.writeLog("error", rec, "failed: " + msg, null);
    }

    // Loads one module. Never throws: any problem marks only this module as failed.
    function load(rec) {
        let api = null;
        let comp = null;
        try {
            api = apiComponent.createObject(root, {
                "moduleId": rec.id,
                "moduleDir": rec.dir,
                "manifest": rec.manifest,
                "apiVersion": root.apiVersion,
                "host": root
            });
            const url = "file://" + rec.main + "?r=" + (++root.loadCounter);
            comp = Qt.createComponent(url);
            if (comp.status !== Component.Ready) {
                const text = comp.status === Component.Error ? comp.errorString() : "the component did not finish loading";
                throw new Error(text.split("\n")[0].split("file://" + rec.dir + "/").join("").trim());
            }
            const instance = comp.createObject(root, {
                "api": api
            });
            if (instance === null) {
                const text = comp.errorString();
                throw new Error(text ? text.split("\n")[0].split("file://" + rec.dir + "/").join("").trim() : "the module could not be created");
            }
            root.live[rec.id] = {
                "component": comp,
                "instance": instance,
                "api": api
            };
            rec.state = "loaded";
            rec.lastError = "";
            rec.runtimeErrors = 0;
            rec.earlyErrors = 0;
            rec.loadedAt = Date.now();
            root.writeLog("info", rec, "loaded", {
                "version": rec.version
            });
        } catch (e) {
            if (comp)
                comp.destroy();

            if (api)
                api.destroy();

            root.fail(rec, String(e.message ?? e));
        }
    }

    // ---- module commands (interface 1.1) ----
    // Runs a declared command of a loaded module through the handler it assigned to
    // api.commands.handler. Always answers at once: {ok, result} or {error: "<kind>: <message>"}.
    function callCommand(id: string, name: string) : variant {
        const rec = root.find(id);
        if (rec === null)
            return {
            "error": "unknown: unknown module '" + id + "'"
        };

        if (rec.state !== "loaded" || !root.live[id])
            return {
            "error": "failed: module " + id + " is not loaded"
        };

        if ((rec.commands ?? []).indexOf(name) < 0)
            return {
            "error": "failed: unknown command '" + name + "'"
        };

        const handler = root.live[id].api.commands.handler;
        if (typeof handler !== "function")
            return {
            "error": "failed: module " + id + " has no handler for '" + name + "'"
        };

        try {
            const result = handler(name, []);
            return {
                "ok": true,
                "result": String(result ?? "").slice(0, root.resultLimit)
            };
        } catch (e) {
            root.writeLog("error", rec, "command " + name + " threw: " + e, null);
            return {
                "error": "failed: " + String(e.message ?? e)
            };
        }
    }

    // ---- items contributed to extension points (interface 1.1) ----
    // Creates the contributed item named by key for host hostId. Returns the object, or null
    // when the contribution cannot be shown (the reason is the contribution's state or a log
    // line). A component or creation error fails the contributor, never the host.
    function createItem(hostId: string, pointId: string, key: string, parent, slot) : variant {
        const found = root.findContribution(key);
        if (found === null) {
            Log.warn("module " + hostId + " asked for an unknown item '" + key + "'", {
                "source": "module:" + hostId
            });
            return null;
        }
        const owner = found.owner;
        const c = found.contribution;
        if (c.target.module !== hostId || c.target.point !== pointId) {
            Log.warn("module " + hostId + " asked for item '" + key + "', which is not for its point '" + pointId + "'", {
                "source": "module:" + hostId
            });
            return null;
        }
        const state = root.contributionState(owner, c);
        if (state.state !== "available" && state.state !== "shown") {
            if (state.state === "rejected")
                Log.warn("item " + key + " was refused: " + state.reason, {
                "source": "module:" + hostId
            });

            return null;
        }
        const live = root.live[owner.id];
        let comp = null;
        try {
            comp = Qt.createComponent("file://" + c.file + "?r=" + (++root.loadCounter));
            if (comp.status !== Component.Ready) {
                const text = comp.status === Component.Error ? comp.errorString() : "the component did not finish loading";
                throw new Error(text.split("\n")[0].split("file://" + owner.dir + "/").join("").trim());
            }
            const obj = comp.createObject(parent, {
                "api": live.api,
                "slot": slot
            });
            if (obj === null) {
                const text = comp.errorString();
                throw new Error(text ? text.split("\n")[0].split("file://" + owner.dir + "/").join("").trim() : "the item could not be created");
            }
            root.items = root.items.concat([{
                "key": key,
                "hostId": hostId,
                "pointId": pointId,
                "contributorId": owner.id,
                "object": obj,
                "component": comp
            }]);
            root.itemRevision += 1;
            return obj;
        } catch (e) {
            if (comp)
                comp.destroy();

            // The contributor failed to build its visual part: only it is marked failed.
            root.unload(owner);
            root.fail(owner, c.file.split(owner.dir + "/").join("") + ": " + String(e.message ?? e));
            root.touch();
            return null;
        }
    }

    function destroyEntry(entry) {
        try {
            if (entry.object)
                entry.object.destroy();

            entry.component.destroy();
        } catch (e) {
            Log.error("destroying item " + entry.key + " threw: " + e);
        }
    }

    // Destroys one item created by createItem. Returns true when it was one of ours.
    function destroyItem(item) : bool {
        const keep = [];
        let found = false;
        for (const entry of root.items) {
            if (entry.object === item) {
                root.destroyEntry(entry);
                found = true;
            } else {
                keep.push(entry);
            }
        }
        if (found) {
            root.items = keep;
            root.itemRevision += 1;
        }
        return found;
    }

    // Destroys every item the module contributed and every item created for it as a host.
    function destroyItemsOf(moduleId: string) {
        const keep = [];
        let any = false;
        for (const entry of root.items) {
            if (entry.contributorId === moduleId || entry.hostId === moduleId) {
                root.destroyEntry(entry);
                any = true;
            } else {
                keep.push(entry);
            }
        }
        if (any) {
            root.items = keep;
            root.itemRevision += 1;
        }
    }

    // A host refuses a contribution with a reason. The mark lasts until the host's next
    // resolveItems; an item that already exists is destroyed.
    function rejectItem(hostId: string, pointId: string, key: string, reason: string) {
        const found = root.findContribution(key);
        if (found === null || found.contribution.target.module !== hostId || found.contribution.target.point !== pointId) {
            Log.warn("module " + hostId + " refused an unknown item '" + key + "'", {
                "source": "module:" + hostId
            });
            return ;
        }
        const marks = Object.assign({
        }, root.rejects);
        marks[hostId + "|" + pointId + "|" + key] = reason;
        root.rejects = marks;
        const keep = [];
        for (const entry of root.items) {
            if (entry.key === key && entry.hostId === hostId && entry.pointId === pointId)
                root.destroyEntry(entry);
            else
                keep.push(entry);
        }
        root.items = keep;
        root.itemRevision += 1;
    }

    // Stores what a host reports about one of its points (shown by `lumen modules show`).
    function setReport(hostId: string, pointId: string, info) {
        const next = Object.assign({
        }, root.reports);
        next[hostId + "|" + pointId] = info;
        root.reports = next;
    }

    // Resolves a host's lists of item names against the available contributions, in the
    // `modules` helper. callback({lists, warnings, available}) or callback({error}).
    function resolveItems(hostId: string, pointId: string, lists, callback) {
        // A new resolution clears this host's earlier refusals.
        const prefix = hostId + "|" + pointId + "|";
        const next = {
        };
        let cleared = false;
        for (const k in root.rejects) {
            if (k.indexOf(prefix) === 0)
                cleared = true;
            else
                next[k] = root.rejects[k];
        }
        if (cleared) {
            root.rejects = next;
            root.itemRevision += 1;
        }
        const table = [];
        for (const rec of root.list) {
            for (const c of (rec.contributions ?? [])) {
                if (c.target.module !== hostId || c.target.point !== pointId)
                    continue;

                table.push({
                    "key": c.key,
                    "short": c.short ?? "",
                    "module": rec.id,
                    "item": c.id,
                    "name": c.name,
                    "moduleState": rec.state,
                    "enabled": rec.enabled,
                    "reason": rec.lastError ?? "",
                    "target": c.target
                });
            }
        }
        // The host may be reloaded or unloaded while the helper runs; then nobody wants the answer.
        const hostLive = root.live[hostId];
        Backend.run("modules", ["items"], {
            "point": {
                "module": hostId,
                "point": pointId
            },
            "lists": lists,
            "contributions": table
        }, 0, (req) => {
            if (root.live[hostId] !== hostLive || hostLive === undefined)
                return ;

            if (req.state !== "ok") {
                const message = req.error ? req.error.message : req.state;
                Log.error("resolving the items of " + hostId + "/" + pointId + " " + req.state + ": " + message, {
                    "source": "module:" + hostId
                });
                if (callback)
                    callback({
                    "error": message
                });

                return ;
            }
            const d = req.data;
            if (callback)
                callback({
                "lists": {
                    "left": d.left,
                    "center": d.center,
                    "right": d.right
                },
                "warnings": d.warnings,
                "available": d.available
            });

        }, "module:" + hostId);
    }

    // Unloads one module and releases everything it created.
    function unload(rec) {
        // Items first: the ones it contributed and the ones created for it as a host.
        root.destroyItemsOf(rec.id);
        const l = root.live[rec.id];
        if (!l)
            return ;

        try {
            l.api.cancelAll();
            l.instance.destroy();
            l.component.destroy();
            l.api.destroy();
        } catch (e) {
            Log.error("unloading module " + rec.id + " threw: " + e);
        }
        delete root.live[rec.id];
        root.writeLog("info", rec, "unloaded", null);
        // Let the engine free what the module left as garbage now, not at some later collection:
        // repeated reloads of a large module (the bar) otherwise pile it up.
        Qt.callLater(() => {
            return gc();
        });
    }

    Component {
        id: apiComponent

        ModuleApi {
        }

    }

    Process {
        id: follower

        onExited: (exitCode, exitStatus) => {
            return root.followerExited();
        }

        stdout: SplitParser {
            onRead: (line) => {
                root.followerHadOutput = true;
                return root.onLogLine(line);
            }
        }

    }

    Timer {
        id: restartFollower

        interval: 500
        onTriggered: root.runFollower(root.followerHadOutput)
    }

    // The user can switch modules on and off by editing the config file, too.
    Connections {
        function onReloaded() {
            if (root.defsApplied)
                root.applyEnabled();

        }

        target: Config
    }

}
