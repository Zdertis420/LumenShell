import QtQuick
import Quickshell

// The example module. Colors come from api.theme, spacing and sizes from api.style, its
// text and size from its own settings, and a message from its helper program. Nothing
// here talks to Lumen except through the `api` object.
Scope {
    id: root

    required property var api
    // Result of the helper run: its message, or the reason it did not work.
    property string helperMessage: "asking the helper..."
    // The two settings, with a fallback in case they are not available yet.
    readonly property string greeting: api.settings.values.greeting ?? "Hello"
    readonly property int size: api.settings.values.size ?? 1

    Component.onCompleted: {
        api.log.info("hello loaded");
        api.run("hello", {
        }, 0, (request) => {
            root.helperMessage = request.state === "ok" ? request.data.message : "helper " + request.state;
            api.log.info("helper answered", {
                "state": request.state
            });
        });
    }

    FloatingWindow {
        title: "Hello"
        color: root.api.theme.background
        implicitWidth: root.api.style.spacing.xl * 4 * root.size
        implicitHeight: root.api.style.spacing.xl * 2 * root.size

        Column {
            anchors.centerIn: parent
            spacing: root.api.style.spacing.sm

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: root.greeting
                color: root.api.theme.onBackground
                font.pixelSize: root.api.style.font.headline
                font.weight: root.api.style.font.weightBold
            }

            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                text: root.helperMessage
                color: root.api.theme.onSurfaceVariant
                font.pixelSize: root.api.style.font.body
            }

        }

    }

}
