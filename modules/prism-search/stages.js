.pragma library

// The stages of a search and the rows they become (specs/006-prism-modes, data-model.md and
// research R3). Pure functions: no QML types, so they are unit tested without a shell
// (scripts/tests/search_stages_test.qml).
//
// A search is a plan: ordered stages, each one request to the `search` helper. The helper
// answers each stage once; the stages run in parallel, and build() shows the longest prefix of
// stages that have answered, so rows only ever appear below the rows already shown.

var ROW_LIMIT = 1000; // documented constant: Prism takes at most this many rows from a source
var RUN_MARGIN_MS = 500; // documented constant: the run's time limit is the stage budget plus this
var SYSTEM = "system";
var FILE_STAGES = ["priority", "home", "homeDirs", "dotDirs", "system"];

var HINTS = {
    "apps": "Type part of an application name.",
    "files": "Type a file name.",
    "commands": "Type part of a command name.",
    "grep": "Type a pattern and at least one file.",
    "web": "Type what to search for.",
    "default": ""
};

function trim(s) {
    return String(s === undefined || s === null ? "" : s).trim();
}

// True when the mode must ask the helper for the directory of the query before the stages.
function needsScope(mode, query) {
    return mode === "files" && trim(query) !== "";
}

function pick(cfg, key, fallback) {
    var v = cfg[key];
    return v === undefined || v === null ? fallback : v;
}

// The settings that every file stage passes to the helper.
function fileRequest(stage, name, cfg, dir) {
    var budget = pick(cfg, "searchTimeMs", 1500);
    var req = {
        "op": "files",
        "stage": stage,
        "name": name,
        "limit": pick(cfg, "rowsPerGroup", 20),
        "maxFiles": pick(cfg, "maxFiles", 200000),
        "budgetMs": budget,
        "hidden": pick(cfg, "hiddenFiles", false),
        "hiddenHomeDirs": pick(cfg, "hiddenHomeDirs", false),
        "systemRoots": pick(cfg, "systemRoots", [])
    };
    if (dir)
        req.dir = dir;
    return req;
}

function stage(id, req) {
    return {
        "id": id,
        "req": req,
        "timeoutMs": (req.budgetMs || 0) + RUN_MARGIN_MS
    };
}

function homeStages(name, cfg) {
    var out = [];
    for (var i = 0; i < FILE_STAGES.length; i++) {
        if (FILE_STAGES[i] === SYSTEM && pick(cfg, "systemGroup", true) === false)
            continue;
        out.push(stage(FILE_STAGES[i], fileRequest(FILE_STAGES[i], name, cfg, "")));
    }
    return out;
}

// Whether Tab completes anything in this mode (FR-038): the directory word of the files mode
// and the file words of the grep mode. The helper decides whether the last word qualifies.
function completable(mode) {
    return mode === "files" || mode === "grep";
}

// The plan of a search. `scope` is the helper's answer for a files query ({dir, name, hint}),
// or null for the other modes. Returns {stages, web, hint, query}.
function plan(mode, query, cfg, scope) {
    var q = trim(query);
    var out = {
        "mode": mode,
        "query": q,
        "stages": [],
        "web": false,
        "hint": ""
    };
    if (q === "") {
        out.hint = HINTS[mode] || "";
        return out;
    }
    // A lone "@" is the start of a mode prefix: the user is about to type a letter, so it is
    // not searched as a name (Feature VI refinement). Longer text such as "@swc" is a query.
    if (mode === "default" && q === "@")
        return out;
    var budget = pick(cfg, "searchTimeMs", 1500);
    var limit = pick(cfg, "rowsPerGroup", 20);
    switch (mode) {
    case "apps":
        out.stages.push(stage("apps", {
            "op": "apps",
            "query": q,
            "limit": limit,
            "budgetMs": budget
        }));
        break;
    case "commands":
        out.stages.push(stage("commands", {
            "op": "commands",
            "query": q,
            "limit": limit,
            "extraDirs": pick(cfg, "commandDirs", []),
            "budgetMs": budget
        }));
        break;
    case "grep":
        out.stages.push(stage("grep", {
            "op": "grep",
            "query": trim(query),
            "limit": Math.min(pick(cfg, "grepMaxMatches", 500), ROW_LIMIT),
            "caseRule": pick(cfg, "grepCase", "smart"),
            "budgetMs": budget
        }));
        break;
    case "web":
        out.web = true;
        break;
    case "files":
        var name = scope ? trim(scope.name) : q;
        out.hint = scope && scope.hint ? scope.hint : "";
        if (name === "") {
            out.hint = HINTS.files;
        } else if (scope && scope.dir) {
            out.stages.push(stage("dir", fileRequest("dir", name, cfg, scope.dir)));
        } else {
            out.stages = homeStages(name, cfg);
        }
        break;
    default:
        out.stages.push(stage("apps", {
            "op": "apps",
            "query": q,
            "limit": limit,
            "budgetMs": budget
        }));
        out.stages = out.stages.concat(homeStages(q, cfg));
        out.web = true;
        break;
    }
    return out;
}

// All stages waiting.
function initialStates(p) {
    var s = {};
    for (var i = 0; i < p.stages.length; i++)
        s[p.stages[i].id] = {
            "status": "waiting"
        };
    return s;
}

// What the user sees when a run did not give an answer.
function failureText(res) {
    if (res && res.error && res.error.message && res.error.code && res.error.code !== "failed")
        return res.error.message;
    if (res && res.state === "timeout")
        return "The search took too long.";
    if (res && res.state === "cancelled")
        return "";
    return "Search is not available (is the helper built? run make build).";
}

function groupTitle(total, cut) {
    return "System files (" + total + (cut ? "+" : "") + ")";
}

function moreRow(stageId, total, shown, cut) {
    var n = total - shown;
    return {
        "id": "m:" + stageId,
        "title": (cut ? n + "+" : String(n)) + " more",
        "subtitle": "Refine your search to see them",
        "icon": "view-more-symbolic"
    };
}

function grepKey(key) {
    var at = String(key).indexOf("\u0000");
    if (at < 0)
        return {
            "path": String(key),
            "line": 0
        };
    return {
        "path": key.slice(0, at),
        "line": parseInt(key.slice(at + 1), 10) || 0
    };
}

// Turns one stage's answer into rows and actions.
function stageRows(st, data, expanded, out) {
    var rows = data.rows || [];
    var id = st.id;
    var isSystem = id === SYSTEM;
    if (isSystem) {
        if (!data.total)
            return;
        var gid = "g:" + SYSTEM;
        out.rows.push({
            "id": gid,
            "title": groupTitle(data.total, data.cut),
            "hint": expanded ? "collapse" : "expand",
            "icon": "folder"
        });
        out.actions[gid] = {
            "kind": "group"
        };
        if (!expanded)
            return;
    }
    for (var i = 0; i < rows.length; i++) {
        var r = rows[i];
        var rid = id + ":" + i;
        var row = {
            "id": rid,
            "title": r.title
        };
        if (r.subtitle)
            row.subtitle = r.subtitle;
        if (r.icon)
            row.icon = r.icon;
        if (r.marks && r.marks.length)
            row.marks = r.marks;
        out.rows.push(row);
        if (id === "apps")
            out.actions[rid] = {
                "kind": "app",
                "id": r.key
            };
        else if (id === "commands")
            out.actions[rid] = {
                "kind": "command"
            };
        else if (id === "grep") {
            var g = grepKey(r.key);
            out.actions[rid] = {
                "kind": "grep",
                "path": g.path,
                "line": g.line
            };
        } else
            out.actions[rid] = {
                "kind": "file",
                "path": r.key
            };
    }
    if (data.total > rows.length && (id !== "grep")) {
        var more = moreRow(id, data.total, rows.length, data.cut === true);
        out.rows.push(more);
        out.actions[more.id] = {
            "kind": "more"
        };
    }
}

// The first error, else the first info, of a grep answer.
function pickMessage(list) {
    var info = null;
    for (var i = 0; i < (list || []).length; i++) {
        if (list[i].kind === "error")
            return list[i];
        if (info === null)
            info = list[i];
    }
    return info;
}

// The rows, actions and message for the current states. The web row is always last.
// Returns {rows, actions, message: {text, kind} | null, done}.
function build(p, states, expanded) {
    var out = {
        "rows": [],
        "actions": {},
        "message": null,
        "done": true
    };
    var failed = null;
    for (var i = 0; i < p.stages.length; i++) {
        var st = p.stages[i];
        var s = states[st.id];
        if (!s || s.status === "waiting") {
            out.done = false;
            break;
        }
        if (s.status === "failed") {
            if (failed === null && s.error)
                failed = s.error;
            continue;
        }
        stageRows(st, s.data || {}, expanded, out);
        if (st.id === "grep") {
            var m = pickMessage(s.data.messages);
            if (m)
                out.message = {
                    "text": m.text,
                    "kind": m.kind
                };
        }
    }
    if (p.web && p.query !== "") {
        out.rows.push({
            "id": "web",
            "title": "Search the web for «" + p.query + "»",
            "icon": "web-browser"
        });
        out.actions["web"] = {
            "kind": "web",
            "query": p.query
        };
    }
    if (out.message === null && failed !== null && out.rows.length === 0)
        out.message = {
            "text": failed,
            "kind": "error"
        };
    if (out.message === null && p.hint !== "" && out.rows.length === 0)
        out.message = {
            "text": p.hint,
            "kind": "info"
        };
    if (out.message === null && p.mode === "files" && p.hint !== "")
        out.message = {
            "text": p.hint,
            "kind": "info"
        };
    return out;
}
