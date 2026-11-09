import QtQuick

// An item written for Feature III: it assumes a horizontal bar. It sizes itself to the
// thickness in height, and to a long text in width, and knows nothing of the direction.
Item {
    id: root

    required property var api
    property var slot

    implicitWidth: label.implicitWidth
    implicitHeight: slot ? slot.thickness : label.implicitHeight
    Component.onCompleted: root.api.log.info("wide item created")

    Text {
        id: label

        anchors.verticalCenter: parent.verticalCenter
        text: "A rather long text for a bar"
        color: root.api.theme.onSurface
        font.pixelSize: root.api.style.font.body
    }

}
