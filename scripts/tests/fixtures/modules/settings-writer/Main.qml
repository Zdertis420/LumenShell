import QtQuick
import Quickshell

// Writes and resets its own settings through the api, and tries an undeclared one.
Scope {
    required property var api

    Component.onCompleted: {
        api.log.info("writer set undeclared " + api.settings.set("nosuch", 1));
        api.settings.set("size", 99, r => api.log.info("writer set size 99 state " + r.state));
        api.settings.set("size", 4, r => {
            api.log.info("writer set size 4 state " + r.state + " value " + api.settings.values.size);
            api.settings.reset("size", r2 => api.log.info("writer reset size state " + r2.state + " value " + api.settings.values.size));
        });
    }
}
