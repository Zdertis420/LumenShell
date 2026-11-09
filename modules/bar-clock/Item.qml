import QtQuick
import Quickshell

// Time and date for the bar (an item contributed to bar/items). Display only.
// On a vertical bar it shows the vertical format (hours above minutes by default).
// Font, padding, corner radius and colors follow the bar unless this module's own settings say
// otherwise (ItemLook.qml).
Item {
    id: root

    required property var api
    property var slot
    readonly property real maxWidth: slot ? slot.maxWidth : label.implicitWidth
    readonly property real maxLength: slot ? slot.maxLength : label.implicitHeight
    // The current time, refreshed by a timer. (A SystemClock per item instance leaked memory
    // on every bar reload; a timer does not, and the time zone is read from the system each time.)
    property date now: new Date()
    readonly property int refreshMs: 1000 // documented constant: how often the time is read
    readonly property int twice: 2 // documented constant: padding is on both sides of the text

    implicitWidth: look.vertical ? look.thickness : Math.min(label.implicitWidth + look.padding * twice, maxWidth)
    implicitHeight: look.vertical ? Math.min(label.implicitHeight + look.padding * twice, maxLength) : (slot ? slot.thickness : label.implicitHeight)

    ItemLook {
        id: look

        api: root.api
        slot: root.slot
    }

    Timer {
        interval: root.refreshMs
        repeat: true
        running: true
        onTriggered: root.now = new Date()
    }

    // The item's own background, when its settings ask for one.
    Rectangle {
        anchors.fill: parent
        visible: look.hasBackground
        color: look.backgroundColor
        radius: look.radius
    }

    Text {
        id: label

        anchors.centerIn: parent
        width: Math.min(implicitWidth, root.width)
        horizontalAlignment: Text.AlignHCenter
        elide: look.vertical ? Text.ElideNone : Text.ElideRight
        text: Qt.formatDateTime(root.now, look.vertical ? root.api.settings.values.verticalFormat : root.api.settings.values.format)
        color: look.textColor
        font.family: look.family
        font.pixelSize: look.pixelSize
        font.weight: look.weight
        font.italic: look.italic
    }

    Connections {
        function onPaletteIdChanged() {
            root.api.log.debug("theme " + root.api.theme.paletteId);
        }

        target: root.api.theme
    }

}
