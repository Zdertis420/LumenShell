import "../../modules/prism-search/stages.js" as Stages
import QtQuick
import QtTest

// Unit tests of the stage plans and the ordered release of rows
// (modules/prism-search/stages.js; specs/006-prism-modes research R3, data-model.md).
TestCase {
    readonly property var cfg: ({
        "rowsPerGroup": 20,
        "maxFiles": 1000,
        "searchTimeMs": 1500,
        "systemGroup": true,
        "systemRoots": ["/usr"],
        "commandDirs": [],
        "grepMaxMatches": 500,
        "grepCase": "smart"
    })

    function ids(p) {
        return p.stages.map((s) => {
            return s.id;
        });
    }

    function ready(data) {
        return {
            "status": "ready",
            "data": data
        };
    }

    function fileRows(prefix, n) {
        const rows = [];
        for (let i = 0; i < n; i++) rows.push({
            "key": "/h/" + prefix + i,
            "title": prefix + i,
            "subtitle": "~",
            "icon": "text-x-generic"
        })
        return rows;
    }

    function rowIds(built) {
        return built.rows.map((r) => {
            return r.id;
        });
    }

    // ---- plans ----
    function test_emptyQueryHasHintAndNoStages() {
        for (const mode of ["apps", "files", "commands", "grep", "web"]) {
            const p = Stages.plan(mode, "   ", cfg, null);
            compare(p.stages.length, 0, mode);
            verify(p.hint !== "", mode);
        }
        const d = Stages.plan("default", "", cfg, null);
        compare(d.stages.length, 0);
        compare(d.hint, "");
        compare(d.web, false);
    }

    function test_onlyFilesAndGrepCompleteWithTab() {
        verify(Stages.completable("files"));
        verify(Stages.completable("grep"));
        for (const mode of ["default", "apps", "commands", "web"])
            verify(!Stages.completable(mode), mode);
    }

    function test_loneAtSignIsLikeAnEmptyQuery() {
        for (const text of ["@", " @ "]) {
            const p = Stages.plan("default", text, cfg, null);
            compare(p.stages.length, 0, text);
            compare(p.web, false, text);
            compare(p.hint, "", text);
        }
        // Longer text is still an ordinary query, and other modes are unchanged.
        const longer = Stages.plan("default", "@swc", cfg, null);
        verify(longer.stages.length > 0);
        compare(longer.web, true);
        verify(Stages.plan("files", "@", cfg, null).stages.length > 0);
    }

    function test_defaultPlanIsAppsThenFileStagesThenWeb() {
        const p = Stages.plan("default", "report", cfg, null);
        compare(ids(p).join(","), "apps,priority,home,homeDirs,dotDirs,system");
        compare(p.web, true);
        compare(p.stages[1].req.name, "report");
        compare(p.stages[1].req.dir, undefined, "the default mode never takes a directory");
        compare(p.stages[0].timeoutMs, 2000);
    }

    function test_systemStageIsSkippedWhenTheGroupIsOff() {
        const off = Object.assign({
        }, cfg, {
            "systemGroup": false
        });
        compare(ids(Stages.plan("default", "x", off, null)).join(","), "apps,priority,home,homeDirs,dotDirs");
    }

    function test_filesNeedsScopeFirstAndUsesIt() {
        verify(Stages.needsScope("files", "report"));
        verify(!Stages.needsScope("files", " "));
        verify(!Stages.needsScope("default", "report ~/x"));
        const home = Stages.plan("files", "report", cfg, {
            "dir": "",
            "name": "report",
            "hint": ""
        });
        compare(ids(home).join(","), "priority,home,homeDirs,dotDirs,system");
        const dir = Stages.plan("files", "report ~/d", cfg, {
            "dir": "/h/d",
            "name": "report",
            "hint": ""
        });
        compare(ids(dir).join(","), "dir");
        compare(dir.stages[0].req.dir, "/h/d");
        compare(dir.web, false);
    }

    function test_filesWithOnlyADirectoryAsksForAName() {
        const p = Stages.plan("files", "~/d", cfg, {
            "dir": "/h/d",
            "name": "",
            "hint": ""
        });
        compare(p.stages.length, 0);
        compare(p.hint, "Type a file name.");
    }

    function test_otherModesHaveOneStage() {
        compare(ids(Stages.plan("apps", "fire", cfg, null)).join(","), "apps");
        compare(ids(Stages.plan("commands", "ff", cfg, null)).join(","), "commands");
        const g = Stages.plan("grep", "TODO a.md", cfg, null);
        compare(ids(g).join(","), "grep");
        compare(g.stages[0].req.caseRule, "smart");
        const w = Stages.plan("web", "a b", cfg, null);
        compare(w.stages.length, 0);
        compare(w.web, true);
    }

    function test_grepLimitIsAtMostWhatPrismTakes() {
        const big = Object.assign({
        }, cfg, {
            "grepMaxMatches": 10000
        });
        compare(Stages.plan("grep", "x a", big, null).stages[0].req.limit, 1000);
    }

    // ---- ordered release ----
    function test_laterStagesAreHeldBackUntilEarlierOnesAnswer() {
        const p = Stages.plan("default", "report", cfg, null);
        const states = Stages.initialStates(p);
        // The files answer first; the apps are still waiting: nothing but the web row shows.
        states.priority = ready({
            "rows": fileRows("p", 2),
            "total": 2,
            "cut": false
        });
        let built = Stages.build(p, states, false);
        compare(rowIds(built).join(","), "web");
        states.apps = ready({
            "rows": [{
                "key": "a.desktop",
                "title": "App",
                "subtitle": "d",
                "icon": "i"
            }],
            "total": 1,
            "cut": false
        });
        built = Stages.build(p, states, false);
        compare(rowIds(built).join(","), "apps:0,priority:0,priority:1,web");
        compare(built.done, false);
    }

    function test_webRowIsAlwaysLastAndAlwaysThereForAQuery() {
        const p = Stages.plan("default", "report", cfg, null);
        const states = Stages.initialStates(p);
        for (const id of ids(p)) states[id] = ready({
            "rows": [],
            "total": 0,
            "cut": false
        })
        const built = Stages.build(p, states, false);
        compare(rowIds(built).join(","), "web");
        compare(built.done, true);
        compare(built.rows[0].title, "Search the web for «report»");
        compare(built.actions.web.kind, "web");
        compare(built.actions.web.query, "report");
    }

    function test_rowIdsAreStableWhenLaterStagesArrive() {
        const p = Stages.plan("default", "r", cfg, null);
        const states = Stages.initialStates(p);
        states.apps = ready({
            "rows": [{
                "key": "a",
                "title": "A"
            }],
            "total": 1
        });
        const before = rowIds(Stages.build(p, states, false));
        states.priority = ready({
            "rows": fileRows("p", 1),
            "total": 1
        });
        const after = rowIds(Stages.build(p, states, false));
        compare(after[0], before[0]);
        verify(after.indexOf("priority:0") > 0);
    }

    function test_systemGroupIsCollapsedUntilExpanded() {
        const p = Stages.plan("files", "r", cfg, {
            "dir": "",
            "name": "r",
            "hint": ""
        });
        const states = Stages.initialStates(p);
        for (const id of ids(p)) states[id] = ready({
            "rows": [],
            "total": 0
        })
        states.system = ready({
            "rows": fileRows("s", 3),
            "total": 57,
            "cut": true
        });
        let built = Stages.build(p, states, false);
        compare(rowIds(built).join(","), "g:system");
        compare(built.rows[0].title, "System files (57+)");
        compare(built.rows[0].hint, "expand");
        compare(built.actions["g:system"].kind, "group");
        built = Stages.build(p, states, true);
        compare(rowIds(built).join(","), "g:system,system:0,system:1,system:2,m:system");
        compare(built.rows[0].hint, "collapse");
        compare(built.rows[4].title, "54+ more");
    }

    function test_noSystemMatchesMeansNoGroup() {
        const p = Stages.plan("files", "r", cfg, {
            "dir": "",
            "name": "r",
            "hint": ""
        });
        const states = Stages.initialStates(p);
        for (const id of ids(p)) states[id] = ready({
            "rows": [],
            "total": 0
        })
        compare(Stages.build(p, states, false).rows.length, 0);
    }

    function test_aFullGroupGetsAMoreRow() {
        const p = Stages.plan("apps", "a", cfg, null);
        const states = {
            "apps": ready({
                "rows": [{
                    "key": "a",
                    "title": "A"
                }],
                "total": 5,
                "cut": false
            })
        };
        const built = Stages.build(p, states, false);
        compare(built.rows[1].title, "4 more");
        compare(built.actions["m:apps"].kind, "more");
    }

    function test_failedStageLosesOnlyItsRows() {
        const p = Stages.plan("default", "r", cfg, null);
        const states = Stages.initialStates(p);
        states.apps = {
            "status": "failed",
            "error": "Search failed."
        };
        for (const id of ids(p).slice(1)) states[id] = ready({
            "rows": fileRows(id, 1),
            "total": 1
        })
        const built = Stages.build(p, states, false);
        verify(rowIds(built).indexOf("priority:0") >= 0);
        compare(built.message, null, "rows exist, so the failure is not the message");
    }

    function test_failureIsTheMessageWhenNothingElseShows() {
        const p = Stages.plan("apps", "r", cfg, null);
        const built = Stages.build(p, {
            "apps": {
                "status": "failed",
                "error": "Search failed."
            }
        }, false);
        compare(built.rows.length, 0);
        compare(built.message.kind, "error");
    }

    // ---- actions ----
    function test_actionsPerKind() {
        const app = Stages.build(Stages.plan("apps", "a", cfg, null), {
            "apps": ready({
                "rows": [{
                    "key": "x.desktop",
                    "title": "X"
                }],
                "total": 1
            })
        }, false);
        compare(app.actions["apps:0"].kind, "app");
        compare(app.actions["apps:0"].id, "x.desktop");
        const cmd = Stages.build(Stages.plan("commands", "a", cfg, null), {
            "commands": ready({
                "rows": [{
                    "key": "/usr/bin/x",
                    "title": "x"
                }],
                "total": 1
            })
        }, false);
        compare(cmd.actions["commands:0"].kind, "command", "a command is never run");
        const grep = Stages.build(Stages.plan("grep", "a f", cfg, null), {
            "grep": ready({
                "rows": [{
                    "key": "/h/f.md\u000012",
                    "title": "a line",
                    "subtitle": "f.md:12",
                    "marks": [[0, 1]]
                }],
                "total": 1,
                "messages": []
            })
        }, false);
        compare(grep.actions["grep:0"].path, "/h/f.md");
        compare(grep.actions["grep:0"].line, 12);
        compare(JSON.stringify(grep.rows[0].marks), "[[0,1]]");
        const file = Stages.build(Stages.plan("files", "a", cfg, {
            "dir": "/d",
            "name": "a",
            "hint": ""
        }), {
            "dir": ready({
                "rows": fileRows("f", 1),
                "total": 1
            })
        }, false);
        compare(file.actions["dir:0"].kind, "file");
        compare(file.actions["dir:0"].path, "/h/f0");
    }

    // ---- messages ----
    function test_grepMessagesErrorBeatsInfo() {
        const p = Stages.plan("grep", "a f", cfg, null);
        const built = Stages.build(p, {
            "grep": ready({
                "rows": [],
                "total": 0,
                "messages": [{
                    "kind": "info",
                    "text": "Binary file skipped: b"
                }, {
                    "kind": "error",
                    "text": "File not found: c"
                }]
            })
        }, false);
        compare(built.message.kind, "error");
        compare(built.message.text, "File not found: c");
    }

    function test_hintsForEmptyQueries() {
        const p = Stages.plan("grep", "", cfg, null);
        const built = Stages.build(p, {
        }, false);
        compare(built.message.kind, "info");
        compare(built.message.text, "Type a pattern and at least one file.");
        compare(Stages.build(Stages.plan("default", "", cfg, null), {
        }, false).message, null);
    }

    function test_directoryNotFoundHintShowsAlongsideRows() {
        const p = Stages.plan("files", "r ~/nope", cfg, {
            "dir": "",
            "name": "r ~/nope",
            "hint": "directory not found: ~/nope"
        });
        const states = Stages.initialStates(p);
        for (const id of ids(p)) states[id] = ready({
            "rows": fileRows(id, 1),
            "total": 1
        })
        const built = Stages.build(p, states, false);
        verify(built.rows.length > 0);
        compare(built.message.text, "directory not found: ~/nope");
    }

    function test_failureText() {
        compare(Stages.failureText({
            "state": "failed",
            "error": {
                "code": "no_editor",
                "message": "EDITOR is not set."
            }
        }), "EDITOR is not set.");
        compare(Stages.failureText({
            "state": "timeout"
        }), "The search took too long.");
        compare(Stages.failureText({
            "state": "cancelled"
        }), "");
        verify(Stages.failureText({
            "state": "failed",
            "error": {
                "code": "failed",
                "message": "x"
            }
        }).indexOf("make build") > 0);
    }

}
