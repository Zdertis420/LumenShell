import QtQuick
import Quickshell

Scope {
    required property var api

    Component.onCompleted: api.log.info("fixture error-storm loaded")

    Timer {
        interval: 100
        repeat: true
        running: true
        onTriggered: {
            throw new Error("fixture error storm");
        }
    }
}
