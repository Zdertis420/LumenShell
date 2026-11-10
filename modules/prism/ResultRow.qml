import QtQuick
import Quickshell

// One row of Prism's results (specs/005-prism-window, FR-010): icon, title, subtitle and a
// right-side hint, with the selected and hovered looks. Every color is a role (look.colors),
// every size comes from `sizes` (geometry.js), and the gaps without a key of their own follow
// a neighbor key (contracts/prism-config.md "Values without their own key"). Children refer
// to this item by id and not by `parent`: a delegate is created before it has a parent.
Item {
    id: row

    required property var look
    required property var sizes
    // Not named `data`: that is the default property of Item, which holds the children.
    required property var entry
    required property bool selected
    required property bool hovered
    readonly property int twice: 2 // documented constant: padding is on both sides, and a box is centered by half
    readonly property int thirds: 3 // documented constant: a hint takes at most a third of the row
    readonly property var colors: look.colors
    readonly property var settings: look.settings
    readonly property bool showIcon: settings.showIcon !== false
    readonly property bool showSubtitle: settings.showSubtitle !== false && entry.subtitle !== ""
    readonly property bool showHint: settings.showHint !== false && entry.hint !== ""
    readonly property var marks: entry.marks ?? []
    // The title with its marked parts (Prism 1.1.0) in bold and in the match color; on the
    // selected row the marked parts take the selected text color, which always has contrast
    // against the selected background.
    readonly property string titleHtml: {
        if (marks.length === 0)
            return "";

        const chars = Array.from(entry.title);
        const color = selected ? colors.selectedText : colors.match;
        const escape = (c) => {
            return c === "&" ? "&amp;" : (c === "<" ? "&lt;" : (c === ">" ? "&gt;" : c));
        };
        let out = "";
        let at = 0;
        for (const m of marks) {
            out += chars.slice(at, m[0]).map(escape).join("");
            out += "<b><font color=\"" + color + "\">" + chars.slice(m[0], m[1]).map(escape).join("") + "</font></b>";
            at = m[1];
        }
        return out + chars.slice(at).map(escape).join("");
    }
    readonly property string iconUrl: {
        const name = entry.icon ?? "";
        if (name === "")
            return "";

        return name.charAt(0) === "/" ? "file://" + name : Quickshell.iconPath(name, true);
    }

    signal clicked()
    signal doubleClicked()
    signal entered()
    signal exited()

    Rectangle {
        anchors.fill: row
        radius: row.sizes.rowRadius
        color: row.selected ? row.colors.selected : (row.hovered ? row.colors.hover : "transparent")
    }

    // The icon, or a neutral square in the input's color when there is none.
    Item {
        id: iconBox

        visible: row.showIcon
        x: row.sizes.rowPadding
        y: (row.height - height) / row.twice
        width: row.showIcon ? row.sizes.iconSize : 0
        height: row.sizes.iconSize

        Image {
            id: image

            anchors.fill: iconBox
            source: row.iconUrl
            sourceSize.width: row.sizes.iconSize
            sourceSize.height: row.sizes.iconSize
            asynchronous: true
            fillMode: Image.PreserveAspectFit
            visible: status === Image.Ready
        }

        Rectangle {
            anchors.fill: iconBox
            visible: image.status !== Image.Ready
            radius: row.sizes.rowRadius
            color: row.colors.input
        }

    }

    Text {
        id: hintText

        visible: row.showHint
        x: row.width - width - row.sizes.rowPadding
        y: (row.height - height) / row.twice
        text: row.entry.hint
        elide: Text.ElideRight
        maximumLineCount: 1
        width: Math.min(implicitWidth, row.width / row.thirds)
        color: row.selected ? row.colors.selectedText : row.colors.hint
        font.family: row.look.fonts.hint.family
        font.pixelSize: row.look.fonts.hint.pixelSize
        font.weight: row.look.fonts.hint.weight
        font.italic: row.look.fonts.hint.italic
    }

    Column {
        id: texts

        x: iconBox.x + iconBox.width + (row.showIcon ? row.sizes.rowPadding : 0)
        y: (row.height - height) / row.twice
        width: Math.max(0, row.width - x - (row.showHint ? hintText.width + row.sizes.rowPadding * row.twice : row.sizes.rowPadding))
        spacing: row.sizes.rowGap

        Text {
            width: texts.width
            text: row.marks.length > 0 ? row.titleHtml : row.entry.title
            textFormat: row.marks.length > 0 ? Text.StyledText : Text.PlainText
            elide: Text.ElideRight
            maximumLineCount: 1
            color: row.selected ? row.colors.selectedText : row.colors.title
            font.family: row.look.fonts.title.family
            font.pixelSize: row.look.fonts.title.pixelSize
            font.weight: row.look.fonts.title.weight
            font.italic: row.look.fonts.title.italic
        }

        Text {
            visible: row.showSubtitle
            width: texts.width
            text: row.entry.subtitle
            elide: Text.ElideRight
            maximumLineCount: 1
            color: row.selected ? row.colors.selectedText : row.colors.subtitle
            font.family: row.look.fonts.subtitle.family
            font.pixelSize: row.look.fonts.subtitle.pixelSize
            font.weight: row.look.fonts.subtitle.weight
            font.italic: row.look.fonts.subtitle.italic
        }

    }

    MouseArea {
        anchors.fill: row
        hoverEnabled: true
        // Hovering follows pointer motion only, so a row that appears under a resting
        // pointer is not hovered (and never selected) until the pointer moves.
        onPositionChanged: row.entered()
        onExited: row.exited()
        onClicked: row.clicked()
        onDoubleClicked: row.doubleClicked()
    }

}
