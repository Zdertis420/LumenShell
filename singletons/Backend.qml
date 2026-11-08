// Shell-side owner of helper runs (and, from US4 on, of the IPC command handlers).

import QtQuick
import Quickshell
import Quickshell.Io
pragma Singleton

// run() starts a helper program as a separate process, writes one JSON document to
// its stdin, reads one JSON document from its stdout, forwards its stderr lines to
// Log, and kills it when the time limit passes. Nothing here blocks the UI thread.
Singleton {
    // ---- Controller commands (Quickshell IPC, protocol 1) ----
    // A handler function returns at once. Work that needs a helper is queued, and its
    // outcome is read with status(). See specs/001-foundation/contracts/ipc-protocol.md.
    // Validate with a trial resolve first; only a valid theme becomes the temp theme.

    id: root

    readonly property bool ready: true
    // Directory holding the helper programs. LUMEN_HELPER_DIR overrides it (tests).
    readonly property string helperDir: {
        const override = Quickshell.env("LUMEN_HELPER_DIR");
        return override ? override : Quickshell.shellDir + "/scripts/bin";
    }
    // Directory holding presets and the schema.
    readonly property string dataDir: Quickshell.shellDir + "/data"
    // Default time limit for a helper run. Config updates it from helper.timeoutMs.
    property int defaultTimeoutMs: 3000
    // Last error per helper name: {message, code, state, time}. A feature that
    // fails because of a helper degrades alone; this is what the test surface shows.
    property var lastErrors: ({
    })
    property int runCount: 0
    readonly property string protocolVersion: "1"
    // State of the last queued command: idle, pending N, done N, or failed N: message.
    property string commandStatus: "idle"
    property int commandSeq: 0

    signal helperFinished(var request)

    // Resolves a helper name ("config") or an explicit path ("/x/y") to a command path.
    function helperPath(program: string) : string {
        return program.indexOf("/") >= 0 ? program : helperDir + "/" + program;
    }

    // Runs `program args...` with `input` (a JS value) as JSON on stdin.
    // callback(request) is called once when the run ends; the request is also returned
    // and emits finished(request). request.state: "ok" | "failed" | "timeout" | "cancelled".
    // tag (optional) names the owner of the run, for example "module:hello"; it appears in
    // error messages and keys lastErrors. request.cancel() kills the process silently.
    function run(program: string, args, input, timeoutMs: int, callback, tag) {
        const limit = timeoutMs > 0 ? timeoutMs : defaultTimeoutMs;
        const req = requestComponent.createObject(root, {
            "tag": tag ? tag : "",
            "helper": program.indexOf("/") >= 0 ? program.split("/").pop() : program,
            "command": [helperPath(program)].concat(args ? args : []),
            "inputText": JSON.stringify(input === undefined ? {
            } : input),
            "timeoutMs": limit,
            "callback": callback
        });
        root.runCount += 1;
        req.start();
        return req;
    }

    function noteResult(req) {
        if (req.state === "ok")
            return ;

        const owner = req.tag ? req.tag + ":" : "";
        const copy = Object.assign({
        }, root.lastErrors);
        copy[owner + req.helper] = {
            "message": req.error ? req.error.message : "unknown error",
            "code": req.error ? req.error.code : "",
            "state": req.state,
            "time": new Date().toISOString()
        };
        root.lastErrors = copy;
        Log.error((req.tag ? req.tag + " " : "") + "helper " + req.helper + " " + req.state + ": " + (req.error ? req.error.message : ""), {
            "helper": req.helper,
            "state": req.state
        });
    }

    function beginCommand() : int {
        root.commandSeq += 1;
        root.commandStatus = "pending " + root.commandSeq;
        return root.commandSeq;
    }

    function endCommand(seq: int, req) {
        if (req.state === "ok") {
            if (root.commandSeq === seq)
                root.commandStatus = "done " + seq;

            return ;
        }
        const code = req.error ? req.error.code : "";
        const invalid = code === "invalid_value" || code === "invalid_theme" || code === "invalid_key";
        const message = req.error ? req.error.message : req.state;
        if (root.commandSeq === seq)
            root.commandStatus = "failed " + seq + ": " + (invalid ? "invalid: " : "") + message;

    }

    // Returns an error reply for a bad or unknown module id, or "" when the id is fine.
    function checkModuleId(id: string) : string {
        if (!/^[a-z][a-z0-9-]{1,31}$/.test(id) && !/^dir:[A-Za-z0-9._-]{1,64}$/.test(id))
            return "error: invalid: invalid module id '" + id + "'";

        if (Modules.find(id) === null)
            return "error: unknown: unknown module '" + id + "'";

        return "";
    }

    // Starts a queued module command for a known id: run(seq) does the work.
    function queueModule(id: string, run) : string {
        const bad = root.checkModuleId(id);
        if (bad)
            return bad;

        const seq = root.beginCommand();
        run(seq);
        return "queued " + seq;
    }

    // result: {ok, unknown, message}. The status is set only after the module settled.
    function endModuleCommand(seq: int, result) {
        if (root.commandSeq !== seq)
            return ;

        if (result.ok)
            root.commandStatus = "done " + seq;
        else
            root.commandStatus = "failed " + seq + ": " + (result.unknown ? "unknown: " : "") + (result.message ?? "the command failed");
    }

    function themeKeys(theme: string) : variant {
        if (theme === "wallpaper")
            return [{
            "key": "theme.source",
            "value": "wallpaper"
        }];

        return [{
            "key": "theme.source",
            "value": "preset"
        }, {
            "key": "theme.preset",
            "value": theme
        }];
    }

    // Saves the theme as an override (persist) or applies it to this shell only.
    function queueTheme(theme: string, persist: bool) : int {
        const seq = root.beginCommand();
        if (persist)
            Config.setMany(root.themeKeys(theme), (req) => {
            if (req.state === "ok")
                Config.tempTheme = "";

            root.endCommand(seq, req);
        });
        else
            root.run("config", ["resolve"], {
            "userFile": Config.userFile,
            "dataDir": root.dataDir,
            "themeOverride": theme
        }, 0, (req) => {
            if (req.state === "ok")
                Config.tempTheme = theme;

            root.endCommand(seq, req);
        });
        return seq;
    }

    function queueReset() : int {
        const seq = root.beginCommand();
        Config.tempTheme = "";
        Config.resetMany(["theme.source", "theme.preset"], (req) => {
            return root.endCommand(seq, req);
        });
        return seq;
    }

    IpcHandler {
        // ---- modules (see specs/002-modules/contracts/ipc-modules.md) ----

        function version() : string {
            return root.protocolVersion;
        }

        function ping() : string {
            return "ok";
        }

        function status() : string {
            return root.commandStatus;
        }

        function log(level: string, source: string, msg: string) : string {
            if (Log.order[level] === undefined)
                return "error: invalid: unknown log level '" + level + "'";

            Log.write(level, source, msg, null);
            return "ok";
        }

        function setTheme(theme: string, persist: string) : string {
            if (!/^[a-z0-9][a-z0-9-]*$/.test(theme))
                return "error: invalid: invalid theme name '" + theme + "'";

            if (persist !== "true" && persist !== "false")
                return "error: invalid: persist must be true or false";

            return "queued " + root.queueTheme(theme, persist === "true");
        }

        function resetTheme() : string {
            return "queued " + root.queueReset();
        }

        function modulesList() : string {
            return JSON.stringify({
                "attribution": Modules.attribution,
                "modules": Modules.list.map((rec) => {
                    return Modules.summaryOf(rec);
                })
            });
        }

        function moduleShow(id: string) : string {
            const bad = root.checkModuleId(id);
            if (bad)
                return bad;

            return JSON.stringify(Modules.detailOf(Modules.find(id)));
        }

        // Runs one declared command of a module and answers at once (interface 1.1,
        // specs/003-bar/contracts/cli-ipc-bar.md).
        function moduleCall(id: string, command: string) : string {
            const bad = root.checkModuleId(id);
            if (bad)
                return bad;

            if (!/^[a-z][a-z0-9-]{0,31}$/.test(command))
                return "error: invalid: invalid command name '" + command + "'";

            const r = Modules.callCommand(id, command);
            return r.error ? "error: " + r.error : JSON.stringify(r);
        }

        function moduleEnable(id: string) : string {
            return root.queueModule(id, (seq) => {
                Modules.setEnabled(id, true, (result) => {
                    return root.endModuleCommand(seq, result);
                });
            });
        }

        function moduleDisable(id: string) : string {
            return root.queueModule(id, (seq) => {
                Modules.setEnabled(id, false, (result) => {
                    return root.endModuleCommand(seq, result);
                });
            });
        }

        function moduleReload(id: string) : string {
            return root.queueModule(id, (seq) => {
                Modules.reload(id, (result) => {
                    return root.endModuleCommand(seq, result);
                });
            });
        }

        function modulesRescan() : string {
            const seq = root.beginCommand();
            Modules.scan((ok) => {
                return root.endModuleCommand(seq, {
                    "ok": ok,
                    "message": "the module scan failed"
                });
            });
            return "queued " + seq;
        }

        target: "lumen"
    }

    Component {
        id: requestComponent

        Scope {
            id: req

            property string helper: ""
            // Owner of the run, for example "module:hello" (empty for Lumen's own runs).
            property string tag: ""
            property var command: []
            property string inputText: "{}"
            property int timeoutMs: 3000
            property var callback: null
            property string state: "running"
            property real durationMs: 0
            property var data: null
            property var error: null
            property real startedAt: Date.now()
            property bool finishedFlag: false
            property bool exitSeen: false
            property bool streamSeen: false
            property int exitCode: 0

            signal finished(var request)

            // Kills the process without calling the callback or emitting any signal.
            // Used when the owner (for example a module) goes away.
            function cancel() {
                if (req.finishedFlag)
                    return ;

                req.finishedFlag = true;
                timer.stop();
                req.callback = null;
                req.state = "cancelled";
                proc.signal(9);
                Qt.callLater(() => {
                    return req.destroy();
                });
            }

            function start() {
                req.startedAt = Date.now();
                proc.command = req.command;
                proc.running = true;
                timer.interval = req.timeoutMs;
                timer.start();
            }

            function complete(state: string, data, error) {
                if (req.finishedFlag)
                    return ;

                req.finishedFlag = true;
                timer.stop();
                req.durationMs = Date.now() - req.startedAt;
                req.data = data;
                req.error = error ? Object.assign({
                    "helper": req.helper
                }, error) : null;
                req.state = state;
                root.noteResult(req);
                try {
                    if (req.callback)
                        req.callback(req);

                } catch (e) {
                    Log.error("callback for helper " + req.helper + " threw: " + e);
                }
                req.finished(req);
                root.helperFinished(req);
                Qt.callLater(() => {
                    return req.destroy();
                });
            }

            // Called when both the process exit and the end of stdout were seen.
            function evaluate() {
                if (req.finishedFlag || !req.exitSeen || !req.streamSeen)
                    return ;

                if (req.exitCode !== 0) {
                    req.complete("failed", null, {
                        "code": "exit_" + req.exitCode,
                        "message": "exited with status " + req.exitCode
                    });
                    return ;
                }
                let doc = null;
                try {
                    doc = JSON.parse(out.text);
                } catch (e) {
                    doc = null;
                }
                if (doc === null || typeof doc !== "object" || typeof doc.ok !== "boolean") {
                    req.complete("failed", null, {
                        "code": "bad_output",
                        "message": "output is not a valid response document"
                    });
                    return ;
                }
                if (doc.ok)
                    req.complete("ok", doc.data, null);
                else
                    req.complete("failed", null, doc.error ? doc.error : {
                    "code": "error",
                    "message": "helper reported an error"
                });
            }

            Process {
                id: proc

                stdinEnabled: true
                // Helpers filter their own diagnostics at the source.
                environment: ({
                    "LUMEN_LOG_LEVEL": Log.level
                })
                onStarted: {
                    proc.write(req.inputText);
                    proc.stdinEnabled = false;
                }
                onExited: (exitCode, exitStatus) => {
                    req.exitCode = exitCode;
                    req.exitSeen = true;
                    if (!req.streamSeen && exitCode !== 0)
                        req.streamSeen = true;

                    req.evaluate();
                }

                stdout: StdioCollector {
                    id: out

                    onStreamFinished: {
                        req.streamSeen = true;
                        req.evaluate();
                    }
                }

                stderr: SplitParser {
                    onRead: (line) => {
                        return Log.forward(line, "helper:" + req.helper);
                    }
                }

            }

            Timer {
                id: timer

                repeat: false
                onTriggered: {
                    if (req.finishedFlag)
                        return ;

                    proc.signal(9);
                    req.complete("timeout", null, {
                        "code": "timeout",
                        "message": "did not finish within " + req.timeoutMs + " ms and was stopped"
                    });
                }
            }

        }

    }

}
