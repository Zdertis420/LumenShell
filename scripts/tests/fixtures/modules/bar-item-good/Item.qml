import QtQuick

Item {
    id: root

    required property var api
    property var slot

    implicitWidth: label.implicitWidth + api.style.spacing.md
    implicitHeight: slot ? slot.thickness : label.implicitHeight
    Component.onCompleted: api.log.info("fixture bar-item-good created, label " + api.settings.values.label)

    Text {
        id: label

        anchors.centerIn: parent
        text: root.api.settings.values.label
        color: root.api.theme.onSurface
        font.pixelSize: root.api.style.font.body
    }
}
