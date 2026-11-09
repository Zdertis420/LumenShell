import QtQuick

// An item for interface 1.2: it reads every new slot fact, follows the bar's font, and logs
// what it saw, so a test can check the facts (the `facts` command answers with the same line).
Item {
    id: root

    required property var api
    property var slot
    readonly property string line: slot ? "vertical=" + slot.vertical + " edge=" + slot.edge + " thickness=" + slot.thickness + " maxLength=" + slot.maxLength + " maxWidth=" + slot.maxWidth + " fontSize=" + slot.font.pixelSize + " itemPadding=" + slot.itemPadding + " itemRadius=" + slot.itemRadius + " textRole=" + slot.textRole + " focusRole=" + slot.focusRole : "no slot"

    implicitWidth: slot && slot.vertical ? slot.thickness : label.implicitWidth
    implicitHeight: slot && slot.vertical ? label.implicitHeight : (slot ? slot.thickness : label.implicitHeight)
    onLineChanged: root.api.log.info("slot facts: " + root.line)
    Component.onCompleted: {
        root.api.log.info("slot facts: " + root.line);
        root.api.commands.handler = (name) => {
            return root.line;
        };
    }

    Text {
        id: label

        anchors.centerIn: parent
        text: "x"
        color: root.api.theme.roles[root.slot ? root.slot.textRole : "onSurface"]
        font.family: root.slot ? root.slot.font.family : root.api.style.font.family
        font.pixelSize: root.slot ? root.slot.font.pixelSize : root.api.style.font.body
    }

}
