import QtQuick

Item {
    required property var api
    property var slot

    implicitWidth: 40
    implicitHeight: slot ? slot.thickness : 20
    Component.onCompleted: api.run("h", {}, 1500, req => api.log.info("fixture bar-item-helper-fails state " + req.state))
}
