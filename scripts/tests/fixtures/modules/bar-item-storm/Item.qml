import QtQuick

Item {
    required property var api
    property var slot

    implicitWidth: 40
    implicitHeight: slot ? slot.thickness : 20
    Component.onCompleted: api.log.info("fixture bar-item-storm created")

    Timer {
        interval: 100
        repeat: true
        running: true
        onTriggered: {
            throw new Error("fixture bar item error storm");
        }
    }
}
