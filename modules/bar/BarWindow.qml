import QtQuick
import Quickshell
import Quickshell.Wayland
import "layout.js" as Layout

// The bar on one screen: a layer-shell panel with three places. Items are created through
// the module interface (api.extensions.create); this window never touches an item's insides.
// Items are laid out here by hand from their measured widths, so the fitting rule
// (layout.js) can drop the ones that do not fit, and so a list edit only creates, destroys
// and moves the items that changed (FR-015, FR-015a).
// The panel reserves its space through exclusionMode Auto. Never assign exclusiveZone here:
// doing so switches the mode to Normal (specs/003-bar/research.md, R8a).
PanelWindow {
    id: win

    required property var api
    // The resolved lists: {left: [{name, key, module, item}], center: [...], right: [...]}.
    required property var resolved
    required property string edge
    required property int thickness
    readonly property real maxWidthFraction: 0.5 // documented constant: an item gets at most half of the bar
    readonly property int halves: 2 // documented constant: an item is centered across the bar's thickness
    readonly property int gap: win.api.style.spacing.sm
    readonly property int margin: win.api.style.spacing.sm
    // The items this window created, in list order within each place:
    // [{key, name, place, holder, item, slot}].
    property var entries: []
    // Names of the items the fitting rule dropped, in drop order.
    property var droppedNames: []
    property bool layoutPending: false
    property bool focusActive: false
    property int focusIndex: 0
    // The key of the item whose popup is open on this bar, or "".
    property string popupKey: ""

    // Emitted just before a popup opens here, so the other bars close theirs (one popup at a time).
    signal popupOpened()

    // slot.popup.open for the item with this key. Refused for an item that is not shown.
    function openPopup(key: string, component, options) : bool {
        let entry = null;
        for (const e of win.entries) {
            if (e.key === key)
                entry = e;

        }
        if (entry === null || !entry.item || !entry.item.visible || entry.holder.dropped) {
            win.api.log.warn("popup refused: item " + key + " is not shown");
            return false;
        }
        if (!component)
            return false;

        win.popupOpened();
        win.closePopup();
        const top = win.edge === "top" ? 0 : (win.screen ? win.screen.height : 0) - win.thickness;
        // The popup window exists only while a popup is open (a window per bar that is never
        // used would cost memory on every bar reload).
        popupLoader.active = true;
        const host = popupLoader.item;
        if (host === null) {
            win.api.log.warn("popup window could not be created");
            return false;
        }
        host.anchorRect = {
            "x": entry.holder.x,
            "y": top + entry.holder.y,
            "width": entry.holder.width,
            "height": entry.holder.height
        };
        if (!host.show(component, {
            "width": options.width ?? 0,
            "height": options.height ?? 0
        })) {
            popupLoader.active = false;
            return false;
        }
        win.popupKey = key;
        win.api.log.debug("popup opened for " + key);
        return true;
    }

    function closePopup() {
        if (win.popupKey === "")
            return ;

        const key = win.popupKey;
        if (popupLoader.item)
            popupLoader.item.close();

        popupLoader.active = false;
        win.popupKey = "";
        win.api.log.debug("popup closed for " + key);
        // The keyboard focus goes back to the item (and the bar, if it had the focus).
        if (win.focusActive)
            keyCatcher.forceActiveFocus();

    }

    function scheduleLayout() {
        if (win.layoutPending)
            return ;

        win.layoutPending = true;
        Qt.callLater(() => {
            win.layoutPending = false;
            win.layout();
        });
    }

    function destroyEntry(e) {
        if (win.popupKey === e.key)
            win.closePopup();

        if (e.item)
            win.api.extensions.destroyItem(e.item);

        e.slot.destroy();
        e.holder.destroy();
        win.api.log.debug("removed item " + e.key + " on " + (win.screen ? win.screen.name : ""));
    }

    // Drops entries whose item no longer exists (its module was unloaded).
    function prune() {
        const keep = [];
        for (const e of win.entries) {
            if (e.item !== null && e.item !== undefined)
                keep.push(e);
            else
                win.destroyEntry(e);
        }
        if (keep.length !== win.entries.length)
            win.entries = keep;

    }

    function createEntry(place, ref) : variant {
        const slot = slotComponent.createObject(win, {
            "place": place,
            "edge": Qt.binding(() => {
                return win.edge;
            }),
            "thickness": Qt.binding(() => {
                return win.thickness;
            }),
            "maxWidth": Qt.binding(() => {
                return Math.floor(win.width * win.maxWidthFraction);
            }),
            "screenName": win.screen ? win.screen.name : ""
        });
        slot.popup = popupApiComponent.createObject(slot, {
            "key": ref.key
        });
        const holder = holderComponent.createObject(content, {
        });
        const item = win.api.extensions.create("items", ref.key, holder, slot);
        if (item === null) {
            slot.destroy();
            holder.destroy();
            return null;
        }
        item.implicitWidthChanged.connect(win.scheduleLayout);
        item.visibleChanged.connect(win.scheduleLayout);
        win.api.log.debug("created item " + ref.key + " on " + (win.screen ? win.screen.name : ""));
        return {
            "key": ref.key,
            "name": ref.name,
            "place": place,
            "holder": holder,
            "item": item,
            "slot": slot
        };
    }

    // Brings the items in line with the resolved lists: unchanged items stay, removed or
    // moved ones are destroyed, new ones are created, and the order follows the lists.
    function sync() {
        win.prune();
        const wanted = {
        };
        for (const place of ["left", "center", "right"]) {
            for (const ref of (win.resolved[place] ?? [])) wanted[ref.key] = place
        }
        const existing = {
        };
        for (const e of win.entries) {
            if (wanted[e.key] === e.place)
                existing[e.key] = e;
            else
                win.destroyEntry(e);
        }
        const next = [];
        for (const place of ["left", "center", "right"]) {
            for (const ref of (win.resolved[place] ?? [])) {
                const e = existing[ref.key] ?? win.createEntry(place, ref);
                if (e !== null)
                    next.push(e);

            }
        }
        win.entries = next;
        win.scheduleLayout();
        if (win.focusActive)
            win.applyFocus();

    }

    // ---- keyboard focus (specs/003-bar/contracts/bar-slot.md) ----
    // The interactive items that can be reached now, in list order.
    function interactiveEntries() : variant {
        return win.entries.filter((e) => {
            return e.item && e.item.interactive === true && e.item.visible && !e.holder.dropped;
        });
    }

    // Takes the keyboard focus. Returns the screen name, or "" when no item is interactive.
    function startFocus() : string {
        if (win.interactiveEntries().length === 0)
            return "";

        win.focusIndex = 0;
        win.focusActive = true;
        win.applyFocus();
        return win.screen ? win.screen.name : "";
    }

    function endFocus() {
        win.focusActive = false;
        keyCatcher.Keys.forwardTo = [];
        for (const e of win.entries) e.slot.focused = false
    }

    // Marks the focused item, and forwards every key but Left, Right and Escape to it.
    function applyFocus() {
        const list = win.interactiveEntries();
        if (list.length === 0) {
            win.endFocus();
            return ;
        }
        win.focusIndex = Math.min(Math.max(win.focusIndex, 0), list.length - 1);
        for (const e of win.entries) e.slot.focused = (e === list[win.focusIndex])
        keyCatcher.Keys.forwardTo = [list[win.focusIndex].item];
    }

    function moveFocus(delta: int) {
        const n = win.interactiveEntries().length;
        if (n === 0)
            return ;

        win.focusIndex = (win.focusIndex + delta + n) % n;
        win.applyFocus();
    }

    // Measures the items, applies the fitting rule, and positions what stays.
    function layout() {
        win.prune();
        const byPlace = {
            "left": [],
            "center": [],
            "right": []
        };
        const widths = {
            "left": [],
            "center": [],
            "right": []
        };
        for (const e of win.entries) {
            byPlace[e.place].push(e);
            const w = e.item.visible ? Math.min(e.item.implicitWidth, e.slot.maxWidth) : 0;
            widths[e.place].push(Math.max(0, Math.ceil(w)));
        }
        const available = Math.max(0, win.width - win.margin - win.margin);
        const fit = Layout.fit(widths, available, win.gap);
        const counts = {
            "left": fit.left,
            "center": fit.center,
            "right": fit.right
        };
        const rightWidth = Layout.widthOf(widths, counts, "right", win.gap);
        const starts = {
            "left": win.margin,
            "center": win.margin + fit.centerX,
            "right": win.width - win.margin - rightWidth
        };
        for (const place of ["left", "center", "right"]) {
            let x = starts[place];
            const list = byPlace[place];
            for (let i = 0; i < list.length; i++) {
                const e = list[i];
                const w = widths[place][i];
                const kept = i < counts[place];
                e.holder.dropped = !kept;
                e.holder.height = win.thickness;
                e.holder.width = kept ? w : 0;
                e.item.width = w;
                e.item.y = Math.round((win.thickness - e.item.height) / win.halves);
                if (kept && w > 0) {
                    e.holder.x = x;
                    x += w + win.gap;
                }
            }
        }
        const names = fit.dropped.map((d) => {
            return byPlace[d.place][d.index].name;
        });
        if (JSON.stringify(names) !== JSON.stringify(win.droppedNames))
            win.droppedNames = names;

    }

    onResolvedChanged: sync()
    onWidthChanged: scheduleLayout()
    onThicknessChanged: scheduleLayout()
    Component.onCompleted: sync()
    Component.onDestruction: {
        for (const e of win.entries) win.destroyEntry(e)
    }
    exclusionMode: ExclusionMode.Auto
    // Exclusive keyboard focus only while the focus command has put the focus in the bar.
    WlrLayershell.keyboardFocus: win.focusActive ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None
    implicitHeight: win.thickness
    color: win.api.theme.surface

    anchors {
        left: true
        right: true
        top: win.edge === "top"
        bottom: win.edge === "bottom"
    }

    Component {
        id: slotComponent

        // What an item receives from the bar (specs/003-bar/contracts/bar-slot.md).
        QtObject {
            property string place: ""
            property string edge: "top"
            property int thickness: 0
            property int maxWidth: 0
            property string screenName: ""
            property bool focused: false
            // open(component, options), close(), isOpen, content: see popupApiComponent.
            property var popup: null
        }

    }

    // slot.popup: one per item. open() creates the component's content in a popup next to the
    // item (options: {width, height}), close() closes it, isOpen and content report its state.
    Component {
        id: popupApiComponent

        QtObject {
            id: p

            property string key: ""
            readonly property bool isOpen: win.popupKey === p.key && p.key !== ""
            readonly property var content: isOpen && popupLoader.item ? popupLoader.item.content : null

            function open(component, options) : bool {
                return win.openPopup(p.key, component, options ?? {
                });
            }

            function close() {
                if (p.isOpen)
                    win.closePopup();

            }

        }

    }

    LazyLoader {
        id: popupLoader

        active: false

        PopupHost {
            api: win.api
            edge: win.edge
            screen: win.screen
            onCloseRequested: win.closePopup()
        }

    }

    // One per item: positions it, clips it to the width the bar gives it, and hides it when
    // the fitting rule dropped it. The item's own `visible` is left alone.
    Component {
        id: holderComponent

        Item {
            property bool dropped: false

            clip: true
            visible: !dropped
        }

    }

    Item {
        id: keyCatcher

        anchors.fill: parent
        focus: true
        Keys.onPressed: (event) => {
            if (!win.focusActive)
                return ;

            if (event.key === Qt.Key_Escape) {
                win.endFocus();
                event.accepted = true;
            } else if (event.key === Qt.Key_Left) {
                win.moveFocus(-1);
                event.accepted = true;
            } else if (event.key === Qt.Key_Right) {
                win.moveFocus(1);
                event.accepted = true;
            }
        }
    }

    Item {
        id: content

        anchors.fill: parent
    }

}
