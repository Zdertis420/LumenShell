import QtQuick
import Quickshell
import Quickshell.Wayland
import "geometry.js" as Geometry
import "layout.js" as Layout

// The bar on one screen: a layer-shell panel with three places. Items are created through
// the module interface (api.extensions.create); this window never touches an item's insides.
// Items are laid out here by hand from their measured lengths, so the fitting rule
// (layout.js) can drop the ones that do not fit, and so a list edit only creates, destroys
// and moves the items that changed (FR-015, FR-015a).
// The panel reserves its space through exclusionMode Auto, which reserves its thickness plus
// its margin on the bar's edge (specs/004-bar-customization-styles/research.md, spike R3).
// Never assign exclusiveZone here: doing so switches the mode to Normal.
// One panel per screen draws all three styles and owns the items, so a style or edge change
// re-draws and re-anchors it and never recreates the items (FR-005):
//   solid:   one strip.
//   islands: three rounded shapes (one per place that holds an item); the panel is not mapped
//            while no item is shown, and its input mask is the shapes only.
//   frame:   the strip, colored as the frame; FrameBorders.qml adds the rest.
PanelWindow {
    id: win

    required property var api
    // The resolved lists: {left: [{name, key, module, item}], center: [...], right: [...]}.
    required property var resolved
    // Everything the bar resolved from its settings (Main.qml): style, edge, font, colors, tokens.
    required property var look
    // The screen this bar is on (Main.qml passes the same object as `screen`). Everything here
    // reads it instead of the window's own `screen`, which the toolkit resets when the window
    // is hidden or re-anchored.
    required property var display
    readonly property int unlimited: 1e+06 // documented constant: stands for the screen size until it is known
    // This screen's sizes and the warnings that its limits caused (modules/bar/geometry.js).
    readonly property var geo: Geometry.resolve(win.look.settings, win.look.tokens, {
        "width": win.display && win.display.width > 0 ? win.display.width : win.unlimited,
        "height": win.display && win.display.height > 0 ? win.display.height : win.unlimited
    })
    readonly property var sizes: win.geo.sizes
    readonly property var warnings: win.geo.warnings
    readonly property string edge: win.sizes.edge
    readonly property string style: win.sizes.style
    readonly property bool vertical: win.sizes.vertical
    readonly property int thickness: win.sizes.thickness
    // The length of the panel along the bar (its width, or its height on a vertical bar). A panel
    // that is not mapped (islands with nothing shown) has no size yet, so the screen's length
    // minus the margins stands in; that is also what the mapped panel gets.
    readonly property int screenLength: win.vertical ? (win.display ? win.display.height : 0) : (win.display ? win.display.width : 0)
    // What the panel asks for along the bar: the screen's length without the margins.
    readonly property int alongScreen: Math.max(0, win.screenLength - win.sizes.margin - win.sizes.margin)
    // A panel that was just re-anchored still reports its old size until the compositor answers,
    // and a length of at most twice the thickness is taken as such a stale size.
    readonly property int length: {
        const own = win.vertical ? win.height : win.width;
        if (own > win.thickness * win.halves)
            return own;

        return Math.max(0, win.screenLength - win.sizes.margin - win.sizes.margin);
    }
    readonly property real maxLengthFraction: 0.5 // documented constant: an item gets at most half of the bar
    readonly property int halves: 2 // documented constant: an item is centered across the bar's thickness
    readonly property int maxLength: Math.floor(win.length * win.maxLengthFraction)
    // The items this window created, in list order within each place:
    // [{key, name, place, holder, item, slot}].
    property var entries: []
    // Names of the items the fitting rule dropped, in drop order.
    property var droppedNames: []
    // How many items are shown, and where each place's island is: {left, center, right}, each
    // {start, length} along the bar (length 0 for a place with nothing shown).
    property int shownCount: 0
    property var places: ({
        "left": {
            "start": 0,
            "length": 0
        },
        "center": {
            "start": 0,
            "length": 0
        },
        "right": {
            "start": 0,
            "length": 0
        }
    })
    property bool layoutPending: false
    property bool focusActive: false
    property int focusIndex: 0
    // The key of the item whose popup is open on this bar, or "".
    property string popupKey: ""
    property string loggedLook: ""
    // Where the item holders wait while the panel is not mapped (islands with nothing shown):
    // an item inside an unmapped window reads as not visible, which would make the panel
    // stay unmapped for good. This parentless item keeps them measurable.
    property var staging: null
    property var islandLeft: null
    property var islandCenter: null
    property var islandRight: null

    // Emitted just before a popup opens here, so the other bars close theirs (one popup at a time).
    signal popupOpened()

    // Where the panel is on the screen, in screen coordinates (the bar's edge and margin).
    function originX() : real {
        const m = win.sizes.margin;
        if (win.edge === "right")
            return (win.display ? win.display.width : 0) - m - win.thickness;

        return m;
    }

    function originY() : real {
        const m = win.sizes.margin;
        if (win.edge === "bottom")
            return (win.display ? win.display.height : 0) - m - win.thickness;

        return m;
    }

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
        // The popup window exists only while a popup is open (a window per bar that is never
        // used would cost memory on every bar reload).
        popupLoader.active = true;
        const host = popupLoader.item;
        if (host === null) {
            win.api.log.warn("popup window could not be created");
            return false;
        }
        host.anchorRect = {
            "x": win.originX() + entry.holder.x,
            "y": win.originY() + entry.holder.y,
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
        win.api.log.debug("removed item " + e.key + " on " + (win.display ? win.display.name : ""));
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
            "vertical": Qt.binding(() => {
                return win.vertical;
            }),
            "thickness": Qt.binding(() => {
                return win.thickness;
            }),
            "maxLength": Qt.binding(() => {
                return win.maxLength;
            }),
            "maxWidth": Qt.binding(() => {
                return win.vertical ? win.thickness : win.maxLength;
            }),
            "font": Qt.binding(() => {
                return win.look.font;
            }),
            "itemPadding": Qt.binding(() => {
                return win.sizes.itemPadding;
            }),
            "itemRadius": Qt.binding(() => {
                return win.sizes.itemRadius;
            }),
            "textRole": Qt.binding(() => {
                return win.look.textRole;
            }),
            "focusRole": Qt.binding(() => {
                return win.look.focusRole;
            }),
            "screenName": win.display ? win.display.name : ""
        });
        slot.popup = popupApiComponent.createObject(slot, {
            "key": ref.key
        });
        const holder = holderComponent.createObject(win.visible ? content : win.staging, {
        });
        const item = win.api.extensions.create("items", ref.key, holder, slot);
        if (item === null) {
            slot.destroy();
            holder.destroy();
            return null;
        }
        item.implicitWidthChanged.connect(win.scheduleLayout);
        item.implicitHeightChanged.connect(win.scheduleLayout);
        item.visibleChanged.connect(win.scheduleLayout);
        win.api.log.debug("created item " + ref.key + " on " + (win.display ? win.display.name : ""));
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

    // ---- keyboard focus (specs/003-bar/contracts/bar-slot.md; this feature changes no keys) ----
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
        return win.display ? win.display.name : "";
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

    // Measures the items, applies the fitting rule along the bar, and positions what stays.
    function layout() {
        win.prune();
        const byPlace = {
            "left": [],
            "center": [],
            "right": []
        };
        const lengths = {
            "left": [],
            "center": [],
            "right": []
        };
        for (const e of win.entries) {
            byPlace[e.place].push(e);
            const own = win.vertical ? e.item.implicitHeight : e.item.implicitWidth;
            const len = e.item.visible ? Math.min(own, e.slot.maxLength) : 0;
            lengths[e.place].push(Math.max(0, Math.ceil(len)));
        }
        // Solid and frame: padding at both ends of the strip. Islands: padding inside each island.
        const islands = win.style === "islands";
        const endPad = islands ? 0 : win.sizes.padding;
        const placePad = islands ? win.sizes.padding : 0;
        const opts = {
            "itemGap": win.sizes.itemGap,
            "placeGap": win.sizes.placeGap,
            "placePadding": placePad
        };
        const available = Math.max(0, win.length - endPad - endPad);
        const fit = Layout.fit(lengths, available, opts);
        const counts = {
            "left": fit.left,
            "center": fit.center,
            "right": fit.right
        };
        const placeLength = {
            "left": Layout.widthOf(lengths, counts, "left", opts),
            "center": Layout.widthOf(lengths, counts, "center", opts),
            "right": Layout.widthOf(lengths, counts, "right", opts)
        };
        const starts = {
            "left": endPad,
            "center": endPad + fit.centerX,
            "right": win.length - endPad - placeLength.right
        };
        let shown = 0;
        for (const place of ["left", "center", "right"]) {
            let pos = starts[place] + placePad;
            const list = byPlace[place];
            for (let i = 0; i < list.length; i++) {
                const e = list[i];
                const len = lengths[place][i];
                const kept = i < counts[place];
                e.holder.dropped = !kept;
                if (win.vertical) {
                    e.holder.width = win.thickness;
                    e.holder.height = kept ? len : 0;
                    e.item.width = win.thickness;
                    e.item.height = len;
                    e.item.x = 0;
                    e.item.y = 0;
                } else {
                    e.holder.height = win.thickness;
                    e.holder.width = kept ? len : 0;
                    e.item.width = len;
                    e.item.x = 0;
                    e.item.y = Math.round((win.thickness - e.item.height) / win.halves);
                }
                if (kept && len > 0) {
                    shown += 1;
                    if (win.vertical) {
                        e.holder.x = 0;
                        e.holder.y = pos;
                    } else {
                        e.holder.x = pos;
                        e.holder.y = 0;
                    }
                    pos += len + win.sizes.itemGap;
                }
            }
        }
        const nextPlaces = {
        };
        for (const place of ["left", "center", "right"]) nextPlaces[place] = {
            "start": starts[place],
            "length": placeLength[place]
        }
        if (JSON.stringify(nextPlaces) !== JSON.stringify(win.places))
            win.places = nextPlaces;

        if (shown !== win.shownCount)
            win.shownCount = shown;

        const names = fit.dropped.map((d) => {
            return byPlace[d.place][d.index].name;
        });
        if (JSON.stringify(names) !== JSON.stringify(win.droppedNames))
            win.droppedNames = names;

    }

    // One debug line per change of what the bar resolved for this screen (used by the smoke test).
    function logLook() {
        const c = win.look.colors;
        const s = win.sizes;
        const f = win.look.font;
        // key=value pairs, so that a test can look for one of them.
        const line = ["style=" + win.style, "edge=" + win.edge, "thickness=" + s.thickness, "margin=" + s.margin, "padding=" + s.padding, "itemGap=" + s.itemGap, "placeGap=" + s.placeGap, "itemPadding=" + s.itemPadding, "itemRadius=" + s.itemRadius, "radius=" + s.radius, "frameRadius=" + s.frameRadius, "frameBorder=" + s.frameBorder, "popupRadius=" + s.popupRadius, "popupPadding=" + s.popupPadding, "popupBorderWidth=" + s.popupBorderWidth, "fontFamily=" + f.family, "fontSize=" + f.pixelSize, "fontWeight=" + f.weight, "fontItalic=" + f.italic, "colors=" + [String(c.strip), String(c.island), String(c.frame), String(c.popupBackground), String(c.popupBorder)].join(","), "roles=" + win.look.textRole + "," + win.look.focusRole].join(" ");
        if (line === win.loggedLook)
            return ;

        win.loggedLook = line;
        win.api.log.debug("look on " + (win.display ? win.display.name : "") + ": " + line);
    }

    onResolvedChanged: sync()
    onLengthChanged: scheduleLayout()
    onSizesChanged: {
        scheduleLayout();
        Qt.callLater(win.logLook);
    }
    onStyleChanged: {
        // A popup is placed for one style and edge: it closes when either changes.
        closePopup();
        scheduleLayout();
    }
    onEdgeChanged: closePopup()
    onLookChanged: Qt.callLater(win.logLook)
    // The holders move between the panel and the staging item as the panel is mapped and unmapped.
    onVisibleChanged: {
        const parent = win.visible ? content : win.staging;
        for (const e of win.entries) e.holder.parent = parent
        win.scheduleLayout();
    }
    Component.onCompleted: {
        win.staging = stagingComponent.createObject(null);
        sync();
        logLook();
    }
    Component.onDestruction: {
        for (const e of win.entries) win.destroyEntry(e)
        if (win.staging)
            win.staging.destroy();

    }
    // Islands: the panel is mapped only while an item is shown, so no surface and no reserved
    // space remain on a screen where every place is empty (spec Clarifications).
    visible: win.style !== "islands" || win.shownCount > 0
    exclusionMode: ExclusionMode.Auto
    // Exclusive keyboard focus only while the focus command has put the focus in the bar
    // (Principle VIII: the bar takes no keyboard focus on its own).
    WlrLayershell.keyboardFocus: win.focusActive ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None
    WlrLayershell.namespace: "lumen-bar"
    // Both sizes are given (the toolkit would otherwise draw a first frame at a default size,
    // and memory for it is not given back): across the bar its thickness, along it the length
    // of the screen minus the margins.
    implicitHeight: win.vertical ? win.alongScreen : win.thickness
    implicitWidth: win.vertical ? win.thickness : win.alongScreen
    // Shapes are drawn by the rectangles below; the panel itself is transparent.
    color: "transparent"
    // Islands: only the shapes take pointer input, the gaps between them let it through.
    mask: win.style === "islands" ? islandsMask : null

    // The margin is a gap on the bar's edge and on both ends along the bar (a margin on a side
    // the panel is not anchored to is ignored).
    margins {
        top: win.edge === "bottom" ? 0 : win.sizes.margin
        bottom: win.edge === "top" ? 0 : win.sizes.margin
        left: win.edge === "right" ? 0 : win.sizes.margin
        right: win.edge === "left" ? 0 : win.sizes.margin
    }

    anchors {
        left: win.edge !== "right"
        right: win.edge !== "left"
        top: win.edge !== "bottom"
        bottom: win.edge !== "top"
    }

    Component {
        id: slotComponent

        // What an item receives from the bar (specs/003-bar/contracts/bar-slot.md and
        // specs/004-bar-customization-styles/contracts/bar-slot-1.2.md).
        QtObject {
            property string place: ""
            property string edge: "top"
            property bool vertical: false
            property int thickness: 0
            property int maxWidth: 0
            property int maxLength: 0
            property var font: ({
            })
            property int itemPadding: 0
            property int itemRadius: 0
            property string textRole: "onSurface"
            property string focusRole: "primary"
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
            look: win.look
            sizes: win.sizes
            edge: win.edge
            screen: win.display
            onCloseRequested: win.closePopup()
        }

    }

    // The frame style's border panels and corner pieces; they exist only in that style.
    LazyLoader {
        active: win.style === "frame"

        FrameBorders {
            api: win.api
            look: win.look
            sizes: win.sizes
            screen: win.display
        }

    }

    // The input region of the islands style: the union of the three islands.
    Region {
        id: islandsMask

        Region {
            item: islandLeft
        }

        Region {
            item: islandCenter
        }

        Region {
            item: islandRight
        }

    }

    // Solid and frame: the strip. Its color is the background (solid) or the frame (frame).
    Rectangle {
        id: strip

        anchors.fill: parent
        visible: win.style !== "islands"
        color: win.style === "frame" ? win.look.colors.frame : win.look.colors.strip
        radius: Geometry.roundRadius(win.sizes.radius, width, height)
    }

    // Islands: one rounded shape per place that holds a shown item.
    Repeater {
        model: ["left", "center", "right"]

        Rectangle {
            id: island

            required property string modelData
            readonly property var place: win.places[modelData]

            x: win.vertical ? 0 : place.start
            y: win.vertical ? place.start : 0
            width: win.vertical ? win.thickness : place.length
            height: win.vertical ? place.length : win.thickness
            visible: win.style === "islands" && place.length > 0
            color: win.look.colors.island
            radius: Geometry.roundRadius(win.sizes.radius, width, height)
            Component.onCompleted: {
                if (modelData === "left")
                    islandLeft = this;
                else if (modelData === "center")
                    islandCenter = this;
                else
                    islandRight = this;
            }
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

    // One per item: positions it, clips it to the length the bar gives it, and makes it
    // invisible when the fitting rule dropped it. The item's own `visible` is left alone.
    Component {
        id: stagingComponent

        Item {
        }

    }

    Component {
        id: holderComponent

        Item {
            property bool dropped: false

            clip: true
            // A dropped item is made invisible and inert, not hidden: hiding the holder would make
            // the item's own `visible` read false, its length would count as 0, it would be
            // un-dropped, shown again, dropped again, and so on (the flaky "dropped items" check
            // of Feature III's smoke test).
            opacity: dropped ? 0 : 1
            enabled: !dropped
        }

    }

    Item {
        id: content

        anchors.fill: parent
    }

    // The surface follows the theme through the bindings above; this line lets the smoke test see it.
    Connections {
        function onPaletteIdChanged() {
            win.api.log.debug("bar strip follows theme " + win.api.theme.paletteId);
        }

        target: win.api.theme
    }

}
