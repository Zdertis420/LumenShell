import QtQuick

// The item the bar shows. `api` is this module's own interface object (theme, style, log,
// settings); `slot` is what the bar tells the item about its place, including slot.popup.
Item {
    id: root

    required property var api
    property var slot
    // Items that can be used from the keyboard set this. While the bar has the keyboard focus
    // the arrow keys move between such items, and every other key (Enter here) reaches the
    // focused one.
    property bool interactive: true
    readonly property bool focused: slot ? slot.focused : false
    readonly property bool popupOpen: slot && slot.popup ? slot.popup.isOpen : false
    readonly property int debugCycles: 100 // documented constant: F12 opens and closes the popup this many times

    // Opens the popup next to the item, or closes it when it is open.
    function togglePopup() {
        if (!slot || !slot.popup)
            return ;

        if (slot.popup.isOpen)
            slot.popup.close();
        else
            slot.popup.open(popupContent, {
        });
    }

    implicitWidth: label.implicitWidth + api.style.spacing.md * 2
    implicitHeight: slot ? slot.thickness : label.implicitHeight
    Component.onCompleted: api.log.info("hello item created")
    onPopupOpenChanged: api.log.info(popupOpen ? "popup opened" : "popup closed")
    Keys.onPressed: (event) => {
        if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
            root.api.log.info("hello item activated");
            root.togglePopup();
            event.accepted = true;
        } else if (event.key === Qt.Key_F12) {
            // A hidden key for the memory check: open and close the popup many times.
            for (let i = 0; i < root.debugCycles; i++) {
                root.togglePopup();
                root.togglePopup();
            }
            event.accepted = true;
        }
    }

    // The visible focus indicator, from theme roles.
    Rectangle {
        anchors.fill: parent
        anchors.margins: 1
        visible: root.focused
        color: "transparent"
        radius: root.api.style.radius.sm
        border.width: 1
        border.color: root.api.theme.primary
    }

    Text {
        id: label

        anchors.centerIn: parent
        text: root.api.settings.values.text
        color: root.api.theme.onSurface
        font.family: root.api.style.font.family
        font.pixelSize: root.api.style.font.body
    }

    MouseArea {
        anchors.fill: parent
        onClicked: {
            root.api.log.info("hello item activated");
            root.togglePopup();
        }
    }

    // What the popup shows. It is created inside the popup, in this item's context, so it can
    // use `root.api` for colors and sizes and follows theme changes like everything else.
    Component {
        id: popupContent

        Item {
            implicitWidth: hint.implicitWidth + root.api.style.spacing.lg * 2
            implicitHeight: hint.implicitHeight + root.api.style.spacing.lg * 2

            Text {
                id: hint

                anchors.centerIn: parent
                text: root.api.settings.values.text + " (popup)"
                color: root.api.theme.onSurface
                font.family: root.api.style.font.family
                font.pixelSize: root.api.style.font.body
            }

        }

    }

    Connections {
        function onPaletteIdChanged() {
            root.api.log.debug("theme " + root.api.theme.paletteId);
        }

        target: root.api.theme
    }

}
