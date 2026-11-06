import QtQuick
import Quickshell

Scope {
    required property var api

    Component.onCompleted: api.log.info("fixture runtime-once loaded")

    Timer {
        interval: 300
        running: true
        onTriggered: {
            throw new Error("fixture runtime error once");
        }
    }
}
