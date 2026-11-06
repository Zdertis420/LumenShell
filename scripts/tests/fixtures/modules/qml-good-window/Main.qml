import QtQuick
import Quickshell

Scope {
    required property var api

    Component.onCompleted: api.log.info("fixture good-window loaded")

    FloatingWindow {
        color: api.theme.background
        implicitWidth: api.style.spacing.xl * 8
        implicitHeight: api.style.spacing.xl * 4

        Text {
            anchors.centerIn: parent
            text: "good window"
            color: api.theme.onBackground
            font.pixelSize: api.style.font.body
        }
    }
}
