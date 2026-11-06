import QtQuick
import Quickshell
pragma Singleton

// Structured, leveled logging. Every entry is one ndjson line:
// {"time":"...","level":"info","source":"qml","msg":"...", ...fields}
// Helper programs write the same format on stderr; Backend passes those lines
// to forward() unchanged, so everything ends up in one stream.
Singleton {
    id: root

    // Minimum level written to the stream: debug, info, warn, error.
    property string level: "info"
    // The last `maxEntries` entries that passed the filter (shown on the test surface).
    property var entries: []
    readonly property int maxEntries: 100
    readonly property var order: ({
        "debug": 0,
        "info": 1,
        "warn": 2,
        "error": 3
    })
    // How many entries a small view (the test surface) shows.
    readonly property int recentCount: 6

    function recent() : variant {
        return root.entries.slice(-root.recentCount);
    }

    // One short line for display: "HH:MM:SS level source message".
    function brief(entry) : string {
        return String(entry.time).substring(11, 19) + " " + entry.level + " " + entry.source + " " + entry.msg;
    }

    function enabled(lvl: string) : bool {
        const want = order[lvl];
        const have = order[level];
        return want !== undefined && want >= (have === undefined ? 1 : have);
    }

    function debug(msg: string, fields) {
        write("debug", "qml", msg, fields);
    }

    function info(msg: string, fields) {
        write("info", "qml", msg, fields);
    }

    function warn(msg: string, fields) {
        write("warn", "qml", msg, fields);
    }

    function error(msg: string, fields) {
        write("error", "qml", msg, fields);
    }

    function write(lvl: string, source: string, msg: string, fields) {
        if (!enabled(lvl))
            return ;

        const entry = {
            "time": new Date().toISOString(),
            "level": lvl,
            "source": source,
            "msg": msg
        };
        if (fields) {
            for (const k in fields) {
                if (!(k in entry))
                    entry[k] = fields[k];

            }
        }
        emit(entry, JSON.stringify(entry));
    }

    // Takes one stderr line from a helper or the controller. Valid ndjson entries
    // are passed through unchanged; anything else is wrapped as a warning.
    function forward(line: string, source: string) {
        const text = line.trim();
        if (text === "")
            return ;

        let entry = null;
        try {
            entry = JSON.parse(text);
        } catch (e) {
            entry = null;
        }
        if (entry === null || typeof entry !== "object" || !entry.level || !entry.msg) {
            write("warn", source, "unstructured output: " + text, null);
            return ;
        }
        if (!enabled(entry.level))
            return ;

        emit(entry, text);
    }

    function emit(entry, line: string) {
        const copy = root.entries.slice(-(maxEntries - 1));
        copy.push(entry);
        root.entries = copy;
        switch (entry.level) {
        case "error":
            console.error(line);
            break;
        case "warn":
            console.warn(line);
            break;
        case "debug":
            console.debug(line);
            break;
        default:
            console.info(line);
        }
    }

}
