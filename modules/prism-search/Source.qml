import QtQuick
import "stages.js" as Stages

// The search source of Prism (specs/006-prism-modes). It receives the mode and the query from
// Prism's slot, splits a search into stages, runs each stage in the `search` helper, shows the
// rows in a fixed order, and acts on the row the user chooses. It draws nothing, takes no
// keyboard, and does no heavy work: matching, walking, parsing and starting programs are in
// the helper (Principle I).
Item {
    id: root

    required property var api
    property var slot
    readonly property var cfg: api.settings.values
    readonly property int scopeTimeoutMs: 2000 // documented constant: time limit of the quick folder check
    readonly property int actionTimeoutMs: 5000 // documented constant: time limit of launching or opening
    // The number of the current search. A late answer of an older search is ignored.
    property int seq: 0
    property var runs: []
    property var searchPlan: null
    property var states: ({
    })
    property var actions: ({
    })
    property bool expanded: false
    property bool firstPublish: true
    // The message of the last Tab completion {text, kind, forText}: shown while the window text
    // is still forText, because every search publish resets the message line.
    property var note: null

    function cancelRuns() {
        for (const r of root.runs) {
            try {
                r.cancel();
            } catch (e) {
            }
        }
        root.runs = [];
    }

    // Starts a new search for the current mode and query.
    function restart() {
        root.cancelRuns();
        const n = ++root.seq;
        root.expanded = false;
        root.actions = ({
        });
        root.states = ({
        });
        root.searchPlan = null;
        root.firstPublish = true;
        if (!root.slot || !root.slot.open)
            return ;

        const mode = root.slot.mode;
        const query = root.slot.query;
        if (Stages.needsScope(mode, query)) {
            const req = root.api.run("search", {
                "op": "scope",
                "query": query
            }, root.scopeTimeoutMs, (res) => {
                if (n !== root.seq)
                    return ;

                root.begin(n, mode, query, res.state === "ok" ? res.data : {
                    "dir": "",
                    "name": query,
                    "hint": ""
                });
            });
            root.runs = root.runs.concat([req]);
            return ;
        }
        root.begin(n, mode, query, null);
    }

    function begin(n: int, mode: string, query: string, scope) {
        const p = Stages.plan(mode, query, root.cfg, scope);
        root.searchPlan = p;
        root.states = Stages.initialStates(p);
        root.publish();
        const started = [];
        for (const st of p.stages) {
            const req = root.api.run("search", st.req, st.timeoutMs, (res) => {
                if (n !== root.seq)
                    return ;

                const next = Object.assign({
                }, root.states);
                next[st.id] = res.state === "ok" ? {
                    "status": "ready",
                    "data": res.data
                } : {
                    "status": "failed",
                    "error": Stages.failureText(res)
                };
                root.states = next;
                root.publish();
            });
            started.push(req);
        }
        root.runs = root.runs.concat(started);
    }

    // Gives Prism the rows that are ready. The first publish of a search selects the first
    // row; later ones keep the selected row (Prism 1.1.0).
    function publish() {
        if (!root.slot || !root.searchPlan)
            return ;

        const built = Stages.build(root.searchPlan, root.states, root.expanded);
        root.actions = built.actions;
        root.slot.setRows(built.rows, !root.firstPublish);
        root.firstPublish = false;
        // A stage message wins; otherwise the note of a Tab completion, while the text is the one it belongs to.
        const shown = built.message ? built.message : (root.note && root.note.forText === root.slot.text ? root.note : null);
        root.slot.setMessage(shown ? shown.text : "", shown ? shown.kind : "info");
        // Counts only, never the query or a title (Feature VI FR-029); the smoke test reads it.
        root.api.log.debug("search published rows=" + built.rows.length + " done=" + built.done + " message=" + (built.message ? built.message.kind : "none"));
    }

    // Tab (Prism 1.2.0, FR-038): asks the helper to complete the last directory or file word,
    // then writes the answer into the input. Only the files and grep modes take part.
    function complete() {
        if (!root.slot || !root.slot.open || !Stages.completable(root.slot.mode))
            return ;

        const mode = root.slot.mode;
        const query = root.slot.query;
        const text = root.slot.text;
        const prefixLength = text.length - query.length;
        root.api.run("search", {
            "op": "complete",
            "mode": mode,
            "query": query
        }, root.scopeTimeoutMs, (res) => {
            // The text moved on while the helper ran: the answer is for text that is gone.
            if (!root.slot || !root.slot.open || root.slot.text !== text)
                return ;

            if (res.state !== "ok") {
                root.slot.setMessage(Stages.failureText(res), "error");
                return ;
            }
            const d = res.data;
            // Kinds and counts only, never the text (FR-029).
            root.api.log.debug("search completed changed=" + d.changed + " message=" + (d.message ? d.message.kind : "none"));
            const next = d.changed ? text.substring(0, prefixLength) + d.query : text;
            root.note = d.message ? {
                "text": d.message.text,
                "kind": d.message.kind,
                "forText": next
            } : null;
            if (d.changed)
                root.slot.setText(next, prefixLength + d.cursor);
            if (root.note)
                root.slot.setMessage(root.note.text, root.note.kind);
        });
    }

    // Runs an action of a chosen row; on success Prism closes, on failure it stays open with
    // the helper's message (which says what is missing and how to set it).
    function act(input) {
        root.api.run("search", input, root.actionTimeoutMs, (res) => {
            // The kind of action and the outcome only: never a path or a query.
            root.api.log.debug("search action " + input.op + " " + res.state + (res.error ? " " + res.error.code : ""));
            if (res.state === "ok") {
                root.slot.close();
                return ;
            }
            root.slot.setMessage(Stages.failureText(res), "error");
        });
    }

    function choose(id: string) {
        const a = root.actions[id];
        root.api.log.debug("search chosen " + (a ? a.kind : "unknown row"));
        if (!a)
            return ;

        const terminal = {
            "terminal": root.cfg.terminal ?? "",
            "terminalExecFlag": root.cfg.terminalExecFlag ?? ""
        };
        switch (a.kind) {
        case "group":
            root.expanded = !root.expanded;
            root.publish();
            break;
        case "more":
            root.slot.setMessage("Refine your search to see more.", "info");
            break;
        case "app":
            root.act(Object.assign({
                "op": "launch",
                "id": a.id
            }, terminal));
            break;
        case "file":
        case "grep":
            root.act(Object.assign({
                "op": "open",
                "path": a.path,
                "line": a.line ?? 0
            }, terminal));
            break;
        case "web":
            root.act({
                "op": "web",
                "query": a.query,
                "searchUrl": root.cfg.searchUrl
            });
            break;
        }
    }

    Component.onDestruction: root.cancelRuns()

    Connections {
        function onQueryChanged() {
            Qt.callLater(root.restart);
        }

        function onModeChanged() {
            Qt.callLater(root.restart);
        }

        function onOpenChanged() {
            if (root.slot.open) {
                Qt.callLater(root.restart);
                return ;
            }
            // Closed: stop every run and leave no rows behind.
            root.cancelRuns();
            root.seq++;
            root.searchPlan = null;
            root.slot.setRows([], false);
            root.slot.setMessage("", "info");
        }

        function onChosen(id) {
            root.choose(id);
        }

        function onCompleteRequested() {
            root.complete();
        }

        target: root.slot
    }

}
