import QtQuick
import Quickshell

// The bar module: a frame with three places (left, center, right) on every selected screen.
// Everything shown in it is an item that another module contributes to the extension point
// "items"; the user's lists decide which items appear and in which order
// (specs/003-bar/contracts/bar-config.md). This file uses only the public module interface.
Scope {
    id: root

    required property var api
    readonly property var settings: api.settings.values
    readonly property string edge: settings.edge ?? "top"
    readonly property var screenFilter: settings.screens ?? ["all"]
    // The bar's thickness: the body font plus twice the medium spacing.
    readonly property int thickness: api.style.font.body + api.style.spacing.md * 2
    readonly property int itemLimit: 32 // documented constant: the bar shows at most this many items
    readonly property int textLimit: 200 // documented constant: notification text length
    readonly property int notifyTimeoutMs: 3000 // documented constant: time limit of the notification helper
    // The resolved lists, as returned by api.extensions.resolve (after the item limit).
    property var resolved: ({
        "left": [],
        "center": [],
        "right": []
    })
    property int generation: 0
    property bool resolvePending: false
    // Re-resolve when the lists change ...
    readonly property string listsKey: JSON.stringify([settings.left ?? [], settings.center ?? [], settings.right ?? []])
    // ... and when a module changes state. Only the states that decide whether an item can be
    // placed are in the key, so creating an item (which changes contribution states) does not
    // start another resolution.
    readonly property string moduleStates: JSON.stringify(api.modules.list.map((m) => {
        return [m.id, m.state, m.enabled];
    }))
    // ---- what `lumen modules show bar` prints ----
    property var warnings: []
    property var droppedByScreen: ({
    })
    // ---- list entries of the wrong type that the config helper dropped ----
    readonly property var droppedEntries: api.settings.errors.filter((e) => {
        return e.dropped === true;
    })
    readonly property string droppedEntriesKey: JSON.stringify(droppedEntries)
    property string announcedKey: ""
    // The bar windows, for the focus command.
    property var windows: []

    // `lumen modules call bar focus`: puts the keyboard focus into the bar on focusScreen (the
    // first bar when that is empty or not shown). Answers with what happened.
    function focusBar() : string {
        const live = root.windows.filter((w) => {
            return w;
        });
        if (live.length === 0)
            return "no bar is shown";

        const wanted = root.settings.focusScreen ?? "";
        let target = live[0];
        for (const w of live) {
            if (wanted !== "" && w.screen && w.screen.name === wanted)
                target = w;

        }
        for (const w of live) {
            if (w !== target)
                w.endFocus();

        }
        const name = target.startFocus();
        return name === "" ? "no interactive items" : "focus on " + name;
    }

    // One popup at a time across all bars.
    function closeOtherPopups(except) {
        for (const w of root.windows) {
            if (w && w !== except)
                w.closePopup();

        }
    }

    function wantScreen(screen) : bool {
        return root.screenFilter.indexOf("all") >= 0 || root.screenFilter.indexOf(screen.name) >= 0;
    }

    // "right[2]: 42 is not a string; entry dropped"
    function describeDropped(e) : string {
        return e.key + ": " + JSON.stringify(e.value) + " " + e.problem;
    }

    // One log line per dropped entry and one notification per change (FR-012).
    function announceDroppedEntries() {
        if (root.droppedEntriesKey === root.announcedKey)
            return ;

        root.announcedKey = root.droppedEntriesKey;
        if (root.droppedEntries.length === 0)
            return ;

        const lines = root.droppedEntries.map((e) => {
            return root.describeDropped(e);
        });
        for (const line of lines) api.log.error(line)
        const body = lines.join("; ").slice(0, root.textLimit);
        api.run("notify", {
            "summary": "Lumen bar: list entries dropped",
            "body": body
        }, root.notifyTimeoutMs, (req) => {
            if (req.state !== "ok")
                root.api.log.warn("no notification sent: " + (req.error ? req.error.message : req.state));

        });
    }

    // Warnings of the resolution: logged once per change, and shown by `lumen modules show`.
    function setWarnings(list) {
        if (JSON.stringify(list) === JSON.stringify(root.warnings))
            return ;

        root.warnings = list;
        for (const w of list) api.log.warn(w.place + "[" + w.index + "]: " + JSON.stringify(w.entry) + ": " + w.problem)
        root.report();
    }

    // Items the fitting rule dropped on one screen: logged once per change.
    function setDropped(screenName: string, names) {
        const before = root.droppedByScreen[screenName] ?? [];
        if (JSON.stringify(before) === JSON.stringify(names))
            return ;

        const next = Object.assign({
        }, root.droppedByScreen);
        if (names.length === 0)
            delete next[screenName];
        else
            next[screenName] = names;
        root.droppedByScreen = next;
        if (names.length > 0)
            api.log.warn("bar on " + screenName + ": no room for " + names.join(", "));
        else
            api.log.info("bar on " + screenName + ": all items fit again");
        root.report();
    }

    function report() {
        const dropped = [];
        for (const screen of Object.keys(root.droppedByScreen)) dropped.push({
            "screen": screen,
            "items": root.droppedByScreen[screen]
        })
        api.extensions.report("items", {
            "warnings": root.warnings,
            "dropped": dropped
        });
    }

    // Keeps at most itemLimit items (left, then center, then right) and refuses the rest.
    function applyLimit(lists) : variant {
        let count = 0;
        const out = {
        };
        for (const place of ["left", "center", "right"]) {
            out[place] = [];
            for (const ref of (lists[place] ?? [])) {
                if (count < root.itemLimit) {
                    out[place].push(ref);
                    count += 1;
                } else {
                    api.extensions.reject("items", ref.key, "bar item limit (" + root.itemLimit + ") reached");
                }
            }
        }
        return out;
    }

    function resolve() {
        const gen = ++root.generation;
        api.extensions.resolve("items", {
            "left": root.settings.left ?? [],
            "center": root.settings.center ?? [],
            "right": root.settings.right ?? []
        }, (result) => {
            if (gen !== root.generation)
                return ;

            if (result.error) {
                root.api.log.warn("cannot resolve the item lists: " + result.error);
                return ;
            }
            root.setWarnings(result.warnings);
            root.resolved = root.applyLimit(result.lists);
        });
    }

    // Bursts of changes in one turn of the event loop give one resolution.
    function scheduleResolve() {
        if (root.resolvePending)
            return ;

        root.resolvePending = true;
        Qt.callLater(() => {
            root.resolvePending = false;
            root.resolve();
        });
    }

    onListsKeyChanged: scheduleResolve()
    onModuleStatesChanged: scheduleResolve()
    onDroppedEntriesKeyChanged: announceDroppedEntries()
    Component.onCompleted: {
        api.commands.handler = (name) => {
            if (name === "focus")
                return root.focusBar();

            return "";
        };
        api.log.info("bar loaded");
        scheduleResolve();
        announceDroppedEntries();
    }

    Variants {
        model: Quickshell.screens.filter((s) => {
            return root.wantScreen(s);
        })

        BarWindow {
            required property var modelData

            screen: modelData
            api: root.api
            resolved: root.resolved
            edge: root.edge
            thickness: root.thickness
            onDroppedNamesChanged: root.setDropped(modelData.name, droppedNames)
            onPopupOpened: root.closeOtherPopups(this)
            Component.onCompleted: root.windows = root.windows.concat([this])
            Component.onDestruction: {
                root.setDropped(modelData.name, []);
                root.windows = root.windows.filter((w) => {
                    return w && w !== this;
                });
            }
        }

    }

}
