import QtQuick
import Quickshell

// Time and date for the bar (an item contributed to bar/items). Display only.
Item {
    id: root

    required property var api
    property var slot
    readonly property real maxWidth: slot ? slot.maxWidth : label.implicitWidth
    // The current time, refreshed by a timer. (A SystemClock per item instance leaked memory
    // on every bar reload; a timer does not, and the time zone is read from the system each time.)
    property date now: new Date()
    readonly property int refreshMs: 1000 // documented constant: how often the time is read

    implicitWidth: Math.min(label.implicitWidth + api.style.spacing.md * 2, maxWidth)
    implicitHeight: slot ? slot.thickness : label.implicitHeight

    Timer {
        interval: root.refreshMs
        repeat: true
        running: true
        onTriggered: root.now = new Date()
    }

    Text {
        id: label

        anchors.centerIn: parent
        width: Math.min(implicitWidth, root.width)
        elide: Text.ElideRight
        text: Qt.formatDateTime(root.now, root.api.settings.values.format)
        color: root.api.theme.onSurface
        font.family: root.api.style.font.family
        font.pixelSize: root.api.style.font.body
    }

    Connections {
        function onPaletteIdChanged() {
            root.api.log.debug("theme " + root.api.theme.paletteId);
        }

        target: root.api.theme
    }

}
