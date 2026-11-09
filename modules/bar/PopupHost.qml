import QtQuick
import Quickshell
import Quickshell.Wayland
import "popup.js" as Popup

// The popup surface of one bar (specs/003-bar/contracts/bar-slot.md). A transparent layer
// surface that covers the screen while a popup is open: a click anywhere outside the popup
// closes it, Escape closes it, and the popup content sits in a frame placed by popup.js
// (research R9: this does not depend on how the compositor treats popup grabs).
PanelWindow {
    id: host

    required property var api
    required property string edge
    // The bar's resolved look and sizes (Main.qml and geometry.js): the popup's colors, corner
    // radius, padding and border come from the bar's popup settings.
    required property var look
    required property var sizes
    readonly property int padding: host.sizes.popupPadding
    readonly property int twice: 2 // documented constant: padding is on both sides of the content
    // Where the opening item is, in screen coordinates: {x, y, width, height}.
    property var anchorRect: ({
        "x": 0,
        "y": 0,
        "width": 0,
        "height": 0
    })
    // The content object created from the item's component, or null.
    property var content: null
    property bool open: false
    readonly property real maxFraction: 0.5 // documented constant: a popup takes at most half of the screen

    signal closeRequested()

    // Shows `component` in the frame. size is {width, height} (0 means the content's own size).
    function show(component, size) {
        host.close();
        const made = component.createObject(frame, {
        });
        if (made === null) {
            host.api.log.warn("popup content could not be created: " + component.errorString());
            return false;
        }
        host.content = made;
        // The wished size is the content's, plus the padding on both sides.
        const inset = host.padding * host.twice;
        const wish = {
            "width": (size.width > 0 ? size.width : made.implicitWidth) + inset,
            "height": (size.height > 0 ? size.height : made.implicitHeight) + inset
        };
        const r = Popup.place(host.anchorRect, wish, {
            "x": 0,
            "y": 0,
            "width": host.width,
            "height": host.height
        }, host.edge, host.maxFraction);
        frame.x = r.x;
        frame.y = r.y;
        frame.width = r.width;
        frame.height = r.height;
        made.x = host.padding;
        made.y = host.padding;
        made.width = Math.max(0, r.width - inset);
        made.height = Math.max(0, r.height - inset);
        host.open = true;
        keys.forceActiveFocus();
        return true;
    }

    function close() {
        if (host.content) {
            host.content.destroy();
            host.content = null;
        }
        host.open = false;
    }

    visible: open
    color: "transparent"
    exclusionMode: ExclusionMode.Ignore
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "lumen-bar-popup"
    WlrLayershell.keyboardFocus: open ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None
    Component.onDestruction: close()

    anchors {
        left: true
        right: true
        top: true
        bottom: true
    }

    // A click outside the popup closes it.
    MouseArea {
        anchors.fill: parent
        onClicked: host.closeRequested()
    }

    Item {
        id: keys

        focus: true
        Keys.onEscapePressed: host.closeRequested()
    }

    // The popup: surface and outline from theme roles, content inside.
    Rectangle {
        id: frame

        color: host.look.colors.popupBackground
        radius: host.sizes.popupRadius
        border.width: host.sizes.popupBorderWidth
        border.color: host.look.colors.popupBorder
        visible: host.open
        clip: true

        // A click inside must not reach the closing area behind it.
        MouseArea {
            anchors.fill: parent
            z: -1
        }

    }

}
