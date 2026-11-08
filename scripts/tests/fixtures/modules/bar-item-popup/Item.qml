import QtQuick

// Test fixture: an item whose popup is driven through module commands (lumen modules call).
// The instance created last (one per bar) answers the commands.
Item {
    id: root

    required property var api
    property var slot

    function run(name) {
        if (name === "open")
            return String(slot.popup.open(content, {
                "width": 120,
                "height": 60
            }));
        if (name === "close") {
            slot.popup.close();
            return "closed";
        }
        if (name === "state")
            return String(slot.popup.isOpen);
        if (name === "cycle") {
            for (let i = 0; i < 100; i++) {
                slot.popup.open(content, {
                });
                slot.popup.close();
            }
            return "cycled";
        }
        return "";
    }

    implicitWidth: 40
    implicitHeight: slot ? slot.thickness : 20
    Component.onCompleted: {
        api.commands.handler = (name) => {
            return root.run(name);
        };
        api.log.info("popup fixture created");
    }

    Component {
        id: content

        Item {
            implicitWidth: 100
            implicitHeight: 50

            Text {
                anchors.centerIn: parent
                text: "popup"
                color: root.api.theme.onSurface
            }

        }

    }

}
