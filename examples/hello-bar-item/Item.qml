import QtQuick

// The item the bar shows. `api` is this module's own interface object (theme, style, log,
// settings); `slot` is what the bar tells the item about its place, including slot.popup.
// The item follows the bar (interface 1.2): it reads the bar's direction, font, padding and
// colors from the slot, and adapts to a horizontal and to a vertical bar. Its own optional
// setting `fontSize` has no default (null), so unset it follows the bar.
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
    readonly property int twice: 2 // documented constant: padding is on both sides of the text
    // What the bar says about its place, with the design tokens as fallbacks for a bar without them.
    readonly property bool vertical: slot ? slot.vertical : false
    readonly property int padding: slot ? slot.itemPadding : api.style.spacing.md
    readonly property int radius: slot ? slot.itemRadius : api.style.radius.sm
    readonly property string family: slot ? slot.font.family : api.style.font.family
    // An unset own setting (null) falls back to the bar's size, then to the token.
    readonly property int fontSize: api.settings.values.fontSize ?? (slot ? slot.font.pixelSize : api.style.font.body)
    readonly property color textColor: api.theme.roles[slot ? slot.textRole : "onSurface"] ?? api.theme.onSurface
    readonly property color focusColor: api.theme.roles[slot ? slot.focusRole : "primary"] ?? api.theme.primary

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

    // Along the bar the item is as long as its text and padding; across it, as thick as the bar.
    implicitWidth: vertical ? slot.thickness : label.implicitWidth + padding * twice
    implicitHeight: vertical ? label.implicitHeight + padding * twice : (slot ? slot.thickness : label.implicitHeight)
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
        anchors.margins: root.api.style.border.thin
        visible: root.focused
        color: "transparent"
        radius: root.radius
        border.width: root.api.style.border.thin
        border.color: root.focusColor
    }

    Text {
        id: label

        anchors.centerIn: parent
        // Vertical: the text is clipped to the bar's thickness, which is all an item can use.
        width: Math.min(implicitWidth, root.vertical ? root.width : implicitWidth)
        elide: root.vertical ? Text.ElideRight : Text.ElideNone
        text: root.api.settings.values.text
        color: root.textColor
        font.family: root.family
        font.pixelSize: root.fontSize
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
                color: root.textColor
                font.family: root.family
                font.pixelSize: root.fontSize
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
