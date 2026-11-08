import QtQuick

Item {
    id: root

    required property var api
    property var slot

    implicitWidth: label.implicitWidth + api.style.spacing.md
    implicitHeight: slot ? slot.thickness : label.implicitHeight

    Text {
        id: label

        anchors.centerIn: parent
        text: "two"
        color: root.api.theme.onSurface
        font.pixelSize: root.api.style.font.body
    }
}
