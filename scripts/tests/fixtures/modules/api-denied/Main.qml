import QtQuick
import Quickshell

// Declares no permissions and calls everything that needs one. Every call must be refused.
Scope {
    required property var api

    Component.onCompleted: {
        api.run("anything", {}, 500, req => api.log.info("denied run state " + req.state));
        api.log.info("denied config.value " + api.config.value("theme.source"));
        api.log.info("denied config.set " + api.config.set("log.level", "debug"));
        api.log.info("denied log.recent " + api.log.recent());
        api.log.info("denied settingsOf " + api.modules.settingsOf("api-probe"));
        api.log.info("denied done");
    }
}
