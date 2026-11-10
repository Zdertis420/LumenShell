import QtQuick
import Quickshell
import "geometry.js" as Geometry
import "prefix.js" as Prefix
import "rows.js" as Rows
import "sample.js" as Sample

// The Prism module: a Spotlight-like search window (specs/005-prism-window). This is the
// window only: it is opened and closed by commands the user binds in the compositor, it
// takes the keyboard while it is open (a search box, Principle VIII), and other modules
// supply the rows through the extension point "sources" (contracts/prism-source-slot.md).
// This file uses only the public module interface.
Scope {
    id: root

    required property var api
    readonly property var settings: api.settings.values
    // The design tokens that the unset settings fall back to.
    readonly property var tokens: ({
        "spacing": api.style.spacing,
        "radius": api.style.radius,
        "font": {
            "family": api.style.font.family,
            "caption": api.style.font.caption,
            "body": api.style.font.body,
            "title": api.style.font.title
        },
        "border": api.style.border,
        "motion": api.style.motion
    })
    // The families the toolkit lists, read once (it takes about a millisecond).
    readonly property var installedFamilies: Qt.fontFamilies()
    readonly property var fontResult: Geometry.fonts(settings, tokens, installedFamilies)
    readonly property var letterResult: Prefix.checkLetters(settings)
    readonly property int rowLimit: Rows.ROW_LIMIT
    readonly property var colors: ({
        "window": paint("windowRole", "windowOpacity"),
        "border": paint("borderRole", "borderOpacity"),
        "input": paint("inputRole", "inputOpacity"),
        "inputText": paint("inputTextRole", ""),
        "placeholder": paint("placeholderRole", ""),
        "prefix": paint("prefixRole", "prefixOpacity"),
        "title": paint("titleRole", ""),
        "subtitle": paint("subtitleRole", ""),
        "hint": paint("hintRole", ""),
        "selected": paint("selectedRole", "selectedOpacity"),
        "selectedText": paint("selectedTextRole", ""),
        "hover": paint("hoverRole", "hoverOpacity"),
        "separator": paint("separatorRole", ""),
        "footer": paint("footerRole", ""),
        "empty": paint("emptyRole", ""),
        "match": paint("matchRole", ""),
        "message": paint("messageRole", ""),
        "error": paint("errorRole", ""),
        "shadow": paint("shadowRole", "shadowOpacity"),
        "backdrop": paint("backdropRole", "backdropOpacity")
    })
    readonly property var motion: {
        const duration = settings.animationDuration ?? tokens.motion.normal;
        return {
            "kind": settings.animation ?? "fade",
            "duration": duration,
            "scale": settings.animationScale ?? 1,
            "easing": tokens.motion.easing,
            "animated": (settings.animation ?? "fade") !== "none" && duration > 0
        };
    }
    // Everything the window needs to draw.
    readonly property var look: ({
        "settings": settings,
        "tokens": tokens,
        "colors": colors,
        "fonts": fontResult.fonts,
        "motion": motion
    })
    // ---- warnings (shown by `lumen modules show prism`) ----
    // Contrast pairs that do not reach WCAG AA (data-model.md "Contrast pairs").
    readonly property var contrastResult: Geometry.contrastWarnings({
        "windowRole": baseColor("windowRole"),
        "inputRole": baseColor("inputRole"),
        "selectedRole": baseColor("selectedRole"),
        "inputTextRole": baseColor("inputTextRole"),
        "placeholderRole": baseColor("placeholderRole"),
        "titleRole": baseColor("titleRole"),
        "subtitleRole": baseColor("subtitleRole"),
        "hintRole": baseColor("hintRole"),
        "selectedTextRole": baseColor("selectedTextRole"),
        "footerRole": baseColor("footerRole"),
        "emptyRole": baseColor("emptyRole"),
        "matchRole": baseColor("matchRole"),
        "messageRole": baseColor("messageRole"),
        "errorRole": baseColor("errorRole"),
        "inputOpacity": settings.inputOpacity ?? 1,
        "selectedOpacity": settings.selectedOpacity ?? 1
    })
    readonly property var letterWarnings: letterResult.errors.map((e) => {
        return {
            "key": e.key,
            "problem": e.problem + "; " + (e.fallback === null ? "this mode has no prefix" : "using \"" + e.fallback + "\"")
        };
    })
    // Limits that the screen puts on the sizes, from the window while it exists.
    property var windowWarnings: []
    property var sourceWarnings: ({
    })
    readonly property var lookWarnings: fontResult.warnings.concat(letterWarnings, contrastResult, screenWarnings, windowWarnings)
    readonly property string lookWarningsKey: JSON.stringify(lookWarnings)
    property string reportedKey: ""
    // Row problems per source, as item warnings.
    readonly property var itemWarnings: Object.keys(sourceWarnings).sort().reduce((acc, key) => {
        return acc.concat(sourceWarnings[key]);
    }, [])
    readonly property string itemWarningsKey: JSON.stringify(itemWarnings)
    // ---- the window's state (data-model.md) ----
    property bool shown: false
    // The window exists while it is shown or its closing animation runs.
    property bool windowActive: false
    property int commandGeneration: 0
    property string text: ""
    readonly property var parsed: Prefix.parse(text, letterResult.letters)
    readonly property string mode: parsed.mode
    readonly property string query: parsed.query
    readonly property int prefixLength: parsed.prefixLength
    property int selectedIndex: -1
    // Whether the user moved the selection since the text last changed (reported by the window).
    property bool navigated: false
    // Where a source wants the text cursor after setText (Prism 1.2.0); see the window.
    property var cursorRequest: ({
        "pos": -1,
        "n": 0
    })
    // ---- sources and rows ----
    property var sources: ({
    })
    // contribution key -> {slot, item}
    property var sourceRows: ({
    })
    // contribution key -> validated rows
    readonly property var contribs: api.extensions.contributions("sources")
    readonly property var supplied: combine(sourceRows)
    // Which row to select when the rows change next (see setSourceRows).
    property int pendingIndex: 0
    // contribution key -> {text, kind, seq}: the message each source set (Prism 1.1.0)
    property var messages: ({
    })
    property int messageSeq: 0
    readonly property var message: {
        let best = null;
        for (const key of Object.keys(messages)) {
            const m = messages[key];
            if (m.text !== "" && (best === null || m.seq > best.seq))
                best = m;

        }
        return best === null ? {
            "text": "",
            "kind": "info"
        } : best;
    }
    readonly property var rows: settings.preview === true ? Sample.filter(Sample.ROWS, query) : supplied
    readonly property var selectedRow: selectedIndex >= 0 && selectedIndex < rows.length ? rows[selectedIndex] : null
    readonly property var screenWarnings: {
        const name = settings.screen ?? "";
        if (name === "")
            return [];

        for (const s of Quickshell.screens) {
            if (s.name === name)
                return [];

        }
        return [{
            "key": "screen",
            "problem": "no output named \"" + name + "\"; opening on the focused screen"
        }];
    }

    // ---- colors: roles of the theme chosen by name, with an opacity; they follow the theme live ----
    function baseColor(roleKey: string) : color {
        const roles = root.api.theme.roles;
        return roles[root.settings[roleKey]] ?? "transparent";
    }

    function paint(roleKey: string, opacityKey: string) : color {
        return Qt.alpha(root.baseColor(roleKey), opacityKey === "" ? 1 : (root.settings[opacityKey] ?? 1));
    }

    // The rows of all sources, in the order of the contributions.
    function combine(map) : variant {
        const out = [];
        for (const c of root.contribs) {
            const rows = map[c.key];
            if (rows)
                for (const r of rows) out.push(r);

        }
        return out;
    }

    // Sets the rows of one source. With keep (Prism 1.1.0) the selected row stays selected if
    // it is still there, but only after the user moved the selection; until then, and without
    // keep, the first row is selected, as in Feature V.
    function setSourceRows(key: string, list, keep) {
        if (!root.sources[key])
            return ;

        const checked = Rows.validateRows(key, list);
        const rows = Object.assign({
        }, root.sourceRows);
        rows[key] = checked.rows;
        // Decide the selection before the rows change: the window reacts to the change at once.
        root.pendingIndex = Rows.selectionAfter(root.rows, root.selectedIndex, root.settings.preview === true ? root.rows : root.combine(rows), keep === true && root.navigated);
        root.sourceRows = rows;
        root.pendingIndex = 0;
        const warnings = Object.assign({
        }, root.sourceWarnings);
        if (checked.warnings.length > 0)
            warnings[key] = checked.warnings;
        else
            delete warnings[key];
        root.sourceWarnings = warnings;
        for (const w of checked.warnings) api.log.warn(key + ": " + w.problem)
    }

    // Creates the item of each available contribution once, and drops the ones that went away
    // or failed (their rows with them, FR-031).
    function syncSources() {
        const live = {
        };
        for (const c of root.contribs) live[c.key] = c.state
        const next = Object.assign({
        }, root.sources);
        let changed = false;
        for (const key of Object.keys(next)) {
            if (live[key] === "shown" || live[key] === "available")
                continue;

            root.dropSource(key, next);
            changed = true;
        }
        for (const c of root.contribs) {
            if (next[c.key] || (c.state !== "available" && c.state !== "shown"))
                continue;

            const slot = slotComponent.createObject(root, {
                "host": root,
                "key": c.key
            });
            const item = root.api.extensions.create("sources", c.key, holder, slot);
            if (item === null) {
                slot.destroy();
                continue;
            }
            next[c.key] = {
                "slot": slot,
                "item": item
            };
            changed = true;
        }
        if (changed)
            root.sources = next;

    }

    function dropSource(key: string, map) {
        const entry = map[key];
        if (!entry)
            return ;

        root.api.extensions.destroyItem(entry.item);
        entry.slot.destroy();
        delete map[key];
        const rows = Object.assign({
        }, root.sourceRows);
        delete rows[key];
        root.sourceRows = rows;
        const warnings = Object.assign({
        }, root.sourceWarnings);
        delete warnings[key];
        root.sourceWarnings = warnings;
        const messages = Object.assign({
        }, root.messages);
        delete messages[key];
        root.messages = messages;
    }

    // Sets the message line of one source (Prism 1.1.0). An empty text clears it.
    function setSourceMessage(key: string, text, kind) {
        if (!root.sources[key])
            return ;

        const checked = Rows.checkMessage(text, kind);
        const messages = Object.assign({
        }, root.messages);
        root.messageSeq += 1;
        messages[key] = {
            "text": checked.text,
            "kind": checked.kind,
            "seq": root.messageSeq
        };
        root.messages = messages;
    }

    // Messages belong to one query: they are cleared when the text changes.
    function resetMessages() {
        if (Object.keys(root.messages).length > 0)
            root.messages = ({
        });

    }

    // Tells the source that supplied the chosen row (and nobody else). Sample rows have no
    // source and do nothing.
    function chooseRow(index: int) {
        const row = index >= 0 && index < root.rows.length ? root.rows[index] : null;
        if (row === null || row.source === "")
            return ;

        const entry = root.sources[row.source];
        if (entry)
            entry.slot.chosen(row.id);

    }

    // Tab (Prism 1.2.0): every source may complete the text; Prism does nothing else.
    function requestComplete() {
        for (const key of Object.keys(root.sources)) {
            const entry = root.sources[key];
            if (entry && entry.slot)
                entry.slot.completeRequested();

        }
    }

    // A source set the text (answer to Tab): like typing, plus where the cursor goes.
    function setTextFromSource(value, cursor) {
        if (typeof value !== "string") {
            api.log.warn("a source gave setText something that is not a string");
            return ;
        }
        root.setText(value);
        const pos = typeof cursor === "number" && isFinite(cursor) ? Math.max(0, Math.min(value.length, Math.floor(cursor))) : value.length;
        root.cursorRequest = {
            "pos": pos,
            "n": root.cursorRequest.n + 1
        };
    }

    // ---- commands (contracts/prism-surface.md) ----
    // Opens the window. While it is open this re-creates it, which moves it to the screen that
    // has the focus now (a surface without an assigned screen opens where the compositor
    // focus is) and keeps the text. Two commands in a row end in the state of the last one.
    function open() : string {
        const gen = ++root.commandGeneration;
        if (root.windowActive) {
            root.windowActive = false;
            Qt.callLater(() => {
                if (gen !== root.commandGeneration)
                    return ;

                root.shown = true;
                root.windowActive = true;
            });
            return "opened";
        }
        if (root.settings.keepText !== true) {
            root.resetMessages();
            root.text = "";
        }
        root.shown = true;
        root.windowActive = true;
        return "opened";
    }

    function close() : string {
        root.commandGeneration++;
        if (!root.shown && !root.windowActive)
            return "already closed";

        root.shown = false;
        return "closed";
    }

    function toggle() : string {
        return root.shown ? root.close() : root.open();
    }

    function clear() : string {
        root.resetMessages();
        root.text = "";
        return "cleared";
    }

    function setText(value: string) {
        root.resetMessages();
        root.navigated = false;
        root.text = value;
    }

    function setWindowWarnings(list) {
        if (JSON.stringify(list) !== JSON.stringify(root.windowWarnings))
            root.windowWarnings = list;

    }

    // Logged once per change, and shown by `lumen modules show prism`.
    function report() {
        api.extensions.report("sources", {
            "warnings": root.itemWarnings,
            "dropped": [],
            "lookWarnings": root.lookWarnings
        });
    }

    // The screen named in the settings, or null to let the compositor choose (the focused
    // output). A named screen that is absent is a warning.
    function wantedScreen() : variant {
        const name = root.settings.screen ?? "";
        if (name === "")
            return null;

        for (const s of Quickshell.screens) {
            if (s.name === name)
                return s;

        }
        return null;
    }

    onContribsChanged: Qt.callLater(root.syncSources)
    onLookWarningsKeyChanged: {
        if (root.lookWarningsKey === root.reportedKey)
            return ;

        root.reportedKey = root.lookWarningsKey;
        for (const w of root.lookWarnings) api.log.warn(w.key + ": " + w.problem)
        root.report();
    }
    onItemWarningsKeyChanged: root.report()
    Component.onCompleted: {
        api.commands.handler = (name) => {
            switch (name) {
            case "toggle":
                return root.toggle();
            case "open":
                return root.open();
            case "close":
                return root.close();
            case "clear":
                return root.clear();
            }
            return "";
        };
        api.log.info("prism loaded");
        root.syncSources();
        root.reportedKey = root.lookWarningsKey;
        for (const w of root.lookWarnings) api.log.warn(w.key + ": " + w.problem)
        root.report();
    }
    Component.onDestruction: {
        root.shown = false;
        root.windowActive = false;
        const map = Object.assign({
        }, root.sources);
        for (const key of Object.keys(map)) root.dropSource(key, map)
    }

    Component {
        id: slotComponent

        // What a source gets (contracts/prism-source-slot.md): facts that follow the window,
        // and the two things it may do, set its rows and ask Prism to close.
        QtObject {
            id: slot

            required property var host
            required property string key
            readonly property string mode: host.mode
            readonly property string query: host.query
            readonly property string text: host.text
            readonly property bool open: host.shown
            readonly property string selectedId: host.selectedRow && host.selectedRow.source === key ? host.selectedRow.id : ""
            readonly property int rowLimit: host.rowLimit

            signal chosen(string id)
            // Tab was pressed (Prism 1.2.0).
            signal completeRequested()

            function setText(text, cursor) {
                host.setTextFromSource(text, cursor);
            }

            function setRows(rows, keepSelection) {
                host.setSourceRows(key, rows, keepSelection === true);
            }

            function setMessage(text, kind) {
                host.setSourceMessage(key, text, kind);
            }

            function close() {
                host.close();
            }

        }

    }

    Item {
        id: holder

        visible: false
    }

    // The window exists only while it is shown or closing, so a hidden Prism has no surface,
    // no input region and no keyboard focus (FR-002).
    LazyLoader {
        active: root.windowActive

        PrismWindow {
            look: root.look
            rows: root.rows
            text: root.text
            shown: root.shown
            prefixLength: root.prefixLength
            message: root.message.text
            messageKind: root.message.kind
            pendingIndex: root.pendingIndex
            wantedScreen: root.wantedScreen()
            onCloseRequested: root.close()
            onTextEdited: (value) => {
                return root.setText(value);
            }
            onSelected: (index) => {
                root.selectedIndex = index;
            }
            onNavigatedChanged: root.navigated = navigated
            onCompleteRequested: root.requestComplete()
            cursorRequest: root.cursorRequest
            onRowChosen: (index) => {
                return root.chooseRow(index);
            }
            onWarningsChanged: root.setWindowWarnings(warnings)
            onHidden: {
                root.windowActive = false;
                root.selectedIndex = -1;
                root.navigated = false;
                root.setWindowWarnings([]);
            }
        }

    }

}
