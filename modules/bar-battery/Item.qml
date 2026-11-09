import QtQuick
import Quickshell.Services.UPower

// Battery level and charging state for the bar (an item contributed to bar/items). Display
// only. Without a laptop battery the item has no size and shows nothing, and that is not an
// error (FR-007). On a vertical bar it shows the label above the level. Font, padding, corner
// radius and colors follow the bar unless this module's own settings say otherwise
// (ItemLook.qml); the low and critical colors are theme roles as before.
Item {
    id: root

    required property var api
    property var slot
    readonly property int percentScale: 100 // documented constant: UPower reports a fraction
    readonly property int twice: 2 // documented constant: padding is on both sides of the text
    readonly property var device: UPower.displayDevice
    readonly property bool present: device !== null && device !== undefined && device.isPresent && device.isLaptopBattery
    // UPower's percentage is a fraction from 0 to 1 (specs/003-bar/research.md, R11a).
    readonly property int percent: present ? Math.round(Math.min(Math.max(device.percentage, 0), 1) * percentScale) : 0
    readonly property bool charging: present && (device.state === UPowerDeviceState.Charging || device.state === UPowerDeviceState.PendingCharge)
    readonly property bool critical: present && !charging && percent <= root.api.settings.values.criticalLevel
    readonly property bool low: present && !charging && percent <= root.api.settings.values.lowLevel
    readonly property real maxWidth: slot ? slot.maxWidth : label.implicitWidth
    readonly property real maxLength: slot ? slot.maxLength : label.implicitHeight

    visible: present
    implicitWidth: !present ? 0 : (look.vertical ? look.thickness : Math.min(label.implicitWidth + look.padding * twice, maxWidth))
    implicitHeight: !present ? 0 : (look.vertical ? Math.min(label.implicitHeight + look.padding * twice, maxLength) : (slot ? slot.thickness : label.implicitHeight))

    ItemLook {
        id: look

        api: root.api
        slot: root.slot
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
        // Vertical: the label above the level ("Bat" over "85%", "Chg" over "85%" while charging).
        text: look.vertical ? (root.charging ? "Chg\n" : "Bat\n") + root.percent + "%" : (root.charging ? "Bat + " : "Bat ") + root.percent + "%"
        color: root.critical ? root.api.theme.error : (root.low ? root.api.theme.tertiary : look.textColor)
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
