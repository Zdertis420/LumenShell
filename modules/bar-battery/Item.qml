import QtQuick
import Quickshell.Services.UPower

// Battery level and charging state for the bar (an item contributed to bar/items). Display
// only. Without a laptop battery the item has no width and shows nothing, and that is not an
// error (FR-007).
Item {
    id: root

    required property var api
    property var slot
    readonly property int percentScale: 100 // documented constant: UPower reports a fraction
    readonly property var device: UPower.displayDevice
    readonly property bool present: device !== null && device !== undefined && device.isPresent && device.isLaptopBattery
    // UPower's percentage is a fraction from 0 to 1 (specs/003-bar/research.md, R11a).
    readonly property int percent: present ? Math.round(Math.min(Math.max(device.percentage, 0), 1) * percentScale) : 0
    readonly property bool charging: present && (device.state === UPowerDeviceState.Charging || device.state === UPowerDeviceState.PendingCharge)
    readonly property bool critical: present && !charging && percent <= root.api.settings.values.criticalLevel
    readonly property bool low: present && !charging && percent <= root.api.settings.values.lowLevel
    readonly property real maxWidth: slot ? slot.maxWidth : label.implicitWidth

    visible: present
    implicitWidth: present ? Math.min(label.implicitWidth + api.style.spacing.md * 2, maxWidth) : 0
    implicitHeight: slot ? slot.thickness : label.implicitHeight

    Text {
        id: label

        anchors.centerIn: parent
        width: Math.min(implicitWidth, root.width)
        elide: Text.ElideRight
        text: (root.charging ? "Bat + " : "Bat ") + root.percent + "%"
        color: root.critical ? root.api.theme.error : (root.low ? root.api.theme.tertiary : root.api.theme.onSurface)
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
