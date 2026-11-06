import QtQuick
import Quickshell

Scope {
    required property var api

    Component.onCompleted: api.run("h", {}, 1500, req => api.log.info("fixture helper-fails state " + req.state))
}
