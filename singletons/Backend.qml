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
    // and emits finished(request). request.state: "ok" | "failed" | "timeout".
    function run(program: string, args, input, timeoutMs: int, callback) {
        const limit = timeoutMs > 0 ? timeoutMs : defaultTimeoutMs;
        const req = requestComponent.createObject(root, {
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

        const copy = Object.assign({
        }, root.lastErrors);
        copy[req.helper] = {
            "message": req.error ? req.error.message : "unknown error",
            "code": req.error ? req.error.code : "",
            "state": req.state,
            "time": new Date().toISOString()
        };
        root.lastErrors = copy;
        Log.error("helper " + req.helper + " " + req.state + ": " + (req.error ? req.error.message : ""), {
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

        target: "lumen"
    }

    Component {
        id: requestComponent

        Scope {
            id: req

            property string helper: ""
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
