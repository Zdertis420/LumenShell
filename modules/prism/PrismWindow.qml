import QtQuick
import QtQuick.Effects
import Quickshell
import Quickshell.Wayland
import "geometry.js" as Geometry

// The Prism window (specs/005-prism-window, contracts/prism-surface.md): one transparent
// overlay surface that the compositor shrinks to the usable area of one screen (research
// R1), holding the optional backdrop, the shadow and the panel with the input, the results
// and the footer. It exists only while Prism is shown or closing (Main.qml creates and
// destroys it), takes the keyboard while shown, and closes on Escape.
PanelWindow {
    id: win

    required property var look
    required property var rows
    required property string text
    required property bool shown
    required property int prefixLength
    // The one-line message a source set (Prism 1.1.0): a hint or an error, shown between the
    // input and the results.
    property string message: ""
    property string messageKind: "info"
    // Which row to select when the rows change. Main.qml decides, so that a source can keep
    // the selection while rows arrive.
    property int pendingIndex: 0
    // The named screen, or null to let the compositor pick the focused output (research R2).
    property var wantedScreen: null
    readonly property var settings: look.settings
    readonly property var colors: look.colors
    readonly property var fonts: look.fonts
    readonly property var motion: look.motion
    readonly property int settleMs: 50 // documented constant: how long the surface size must stay put before it counts
    // A surface starts with a provisional size and then gets its real one from the compositor,
    // so the panel is placed, and limits are reported, only once the size has settled.
    property bool settled: false
    readonly property bool ready: settled && width > 0 && height > 0
    // The sizes for the area that the compositor gave this surface.
    readonly property var resolved: Geometry.resolve(settings, look.tokens, {
        "width": win.width,
        "height": win.height
    })
    readonly property var sizes: resolved.sizes
    readonly property var warnings: ready ? resolved.warnings : []
    // ---- selection ----
    readonly property int rowCount: rows.length
    readonly property int visibleRows: Math.min(rowCount, sizes.maxRows)
    property int selectedIndex: -1
    // True once the user moved the selection since the text last changed (or the window opened).
    // Main honors a source's keepSelection only then, so the first row stays selected while
    // rows arrive in stages (Feature VI, FR-031a).
    property bool navigated: false
    property int hoverIndex: -1
    readonly property int maxTextLength: 2.14748e+09 // documented constant: the input keeps any length
    // ---- layout of the panel's content ----
    readonly property int frame: sizes.borderWidth + sizes.padding
    readonly property int twice: 2 // documented constant: padding and borders are on both sides
    readonly property int innerWidth: Math.max(0, sizes.width - twice * frame)
    readonly property bool showFooter: settings.showFooter === true && (settings.footerHints ?? []).length > 0
    readonly property string emptyString: text === "" ? (settings.emptyNoSource ?? "") : (settings.emptyNoMatch ?? "")
    readonly property bool showMessage: message !== ""
    readonly property bool showEmpty: rowCount === 0 && emptyString !== "" && !showMessage
    readonly property int inputBoxHeight: input.implicitHeight + twice * sizes.inputPadding
    // The parts in order, their heights and their positions inside the content (data-model:
    // the results area never grows past maxHeight, and the anchor edge stays fixed).
    readonly property var plan: {
        const wantRows = rowCount > 0 ? visibleRows * sizes.rowHeight + (visibleRows - 1) * sizes.rowGap : (showEmpty ? emptyLabel.implicitHeight + twice * sizes.padding : 0);
        const h = {
            "input": inputBoxHeight,
            "message": showMessage ? messageLabel.implicitHeight : 0,
            "results": wantRows,
            "footer": showFooter ? footerRow.implicitHeight : 0
        };
        const order = settings.inputPosition === "bottom" ? ["results", "message", "input", "footer"] : ["input", "message", "results", "footer"];
        const parts = order.filter((k) => {
            return h[k] > 0;
        });
        const sep = settings.separator === true;
        const gapOf = (a, b) => {
            const between = (a === "input" && b === "results") || (a === "results" && b === "input");
            return sep && between ? sizes.gap * twice + sizes.separatorWidth : sizes.gap;
        };
        let fixed = 0;
        for (const k of parts) if (k !== "results") {
            fixed += h[k];
        }
        for (let i = 1; i < parts.length; i++) fixed += gapOf(parts[i - 1], parts[i])
        const room = Math.max(0, sizes.maxHeight - twice * frame - fixed);
        h.results = Math.min(h.results, room);
        const y = {
        };
        let cursor = 0;
        let sepY = -1;
        for (let i = 0; i < parts.length; i++) {
            if (i > 0) {
                const g = gapOf(parts[i - 1], parts[i]);
                if (sep && g !== sizes.gap)
                    sepY = cursor + sizes.gap;

                cursor += g;
            }
            y[parts[i]] = cursor;
            cursor += h[parts[i]];
        }
        return {
            "h": h,
            "y": y,
            "sepY": sepY,
            "total": cursor
        };
    }
    readonly property var placement: Geometry.place(sizes, plan.total + twice * frame)
    // 0 while hidden and 1 while shown; the animation moves it, and the surface is destroyed
    // (by Main.qml, on `hidden`) once it is back at 0.
    property real progress: 0

    signal closeRequested()
    signal textEdited(string value)
    signal selected(int index)
    signal rowChosen(int index)
    // Tab (Prism 1.2.0): the sources may complete the text; Prism itself does nothing else.
    signal completeRequested()
    signal hidden()
    // Where Main wants the text cursor after it set the text from a source ({pos, n}; n makes
    // every request a change). Applied after the text binding updated.
    property var cursorRequest: ({
        "pos": -1,
        "n": 0
    })

    function select(index: int) {
        // rows.length and not rowCount: this runs from onRowsChanged, where the binding of
        // rowCount may not have been re-evaluated yet (a change from 0 to 1 row would select none).
        const count = rows.length;
        const i = count === 0 ? -1 : Math.max(0, Math.min(count - 1, index));
        if (i !== selectedIndex) {
            selectedIndex = i;
            selected(i);
        }
        if (i >= 0)
            list.positionViewAtIndex(i, ListView.Contain);

    }

    // A selection made by the user (keys, click): remembered in `navigated`.
    function userSelect(index: int) {
        navigated = true;
        select(index);
    }

    // Moves the selection; Up and Down wrap when the user asked for it, paging never does.
    function step(delta: int, allowWrap: bool) {
        if (rowCount === 0)
            return ;

        const next = selectedIndex + delta;
        if (allowWrap && settings.wrapSelection === true)
            userSelect((next + rowCount) % rowCount);
        else
            userSelect(next);
    }

    function handleKey(event) {
        switch (event.key) {
        case Qt.Key_Escape:
            closeRequested();
            event.accepted = true;
            break;
        case Qt.Key_Down:
            step(1, true);
            event.accepted = true;
            break;
        case Qt.Key_Up:
            step(-1, true);
            event.accepted = true;
            break;
        case Qt.Key_PageDown:
            step(Math.max(1, visibleRows), false);
            event.accepted = true;
            break;
        case Qt.Key_PageUp:
            step(-Math.max(1, visibleRows), false);
            event.accepted = true;
            break;
        case Qt.Key_Home:
            // In a non-empty input Home and End move the text cursor.
            if (text === "") {
                userSelect(0);
                event.accepted = true;
            }
            break;
        case Qt.Key_End:
            if (text === "") {
                userSelect(rowCount - 1);
                event.accepted = true;
            }
            break;
        case Qt.Key_Tab:
            completeRequested();
            event.accepted = true;
            break;
        case Qt.Key_Return:
        case Qt.Key_Enter:
            if (selectedIndex >= 0)
                rowChosen(selectedIndex);

            event.accepted = true;
            break;
        }
    }

    onWidthChanged: {
        settled = false;
        settle.restart();
    }
    onHeightChanged: {
        settled = false;
        settle.restart();
    }
    onRowsChanged: select(pendingIndex)
    onTextChanged: {
        navigated = false;
        select(0);
    }
    onCursorRequestChanged: {
        const wanted = cursorRequest.pos;
        // After the binding of the text has been applied to the input.
        Qt.callLater(() => {
            if (wanted >= 0)
                input.cursorPosition = Math.min(wanted, input.text.length);

        });
    }
    color: "transparent"
    exclusionMode: ExclusionMode.Normal
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.namespace: "lumen-prism"
    WlrLayershell.keyboardFocus: shown ? WlrKeyboardFocus.Exclusive : WlrKeyboardFocus.None
    visible: true
    onReadyChanged: {
        if (ready)
            input.forceActiveFocus();

    }
    Component.onCompleted: {
        settle.restart();
        progress = shown ? 1 : 0;
        input.forceActiveFocus();
    }
    onShownChanged: progress = shown ? 1 : 0
    onProgressChanged: {
        if (!shown && progress <= 0)
            hidden();

    }

    Timer {
        id: settle

        interval: win.settleMs
        onTriggered: win.settled = true
    }

    // ---- the surface ----
    anchors {
        top: true
        bottom: true
        left: true
        right: true
    }

    Binding {
        target: win
        property: "screen"
        value: win.wantedScreen
        when: win.wantedScreen !== null
    }

    FocusScope {
        id: scope

        anchors.fill: parent
        focus: true
        Keys.onEscapePressed: win.closeRequested()

        // The rest of the screen: dimmed when the backdrop is on, and a click on it closes the
        // window when the user wants that.
        Rectangle {
            anchors.fill: parent
            visible: win.settings.backdrop === true
            color: win.colors.backdrop
            opacity: win.progress
        }

        MouseArea {
            anchors.fill: parent
            onClicked: {
                if (win.settings.closeOnOutside !== false)
                    win.closeRequested();

            }
        }

        Loader {
            active: win.settings.shadow === true && win.ready

            sourceComponent: RectangularShadow {
                x: win.placement.x
                y: win.placement.y
                width: win.placement.width
                height: win.placement.height
                radius: win.sizes.radius
                blur: win.sizes.shadowBlur
                offset.y: win.sizes.shadowOffset
                color: win.colors.shadow
                opacity: win.progress
            }

        }

        Rectangle {
            id: panel

            visible: win.ready
            x: win.placement.x
            y: win.placement.y
            width: win.placement.width
            height: win.placement.height
            radius: win.sizes.radius
            color: win.colors.window
            border.width: win.sizes.borderWidth
            border.color: win.colors.border
            opacity: win.progress
            scale: win.motion.kind === "scale" ? win.motion.scale + (1 - win.motion.scale) * win.progress : 1
            transformOrigin: {
                const top = win.sizes.anchor.indexOf("top") === 0;
                const bottom = win.sizes.anchor.indexOf("bottom") === 0;
                return top ? Item.Top : (bottom ? Item.Bottom : Item.Center);
            }

            // A click on the panel must not reach the closing area behind it.
            MouseArea {
                anchors.fill: parent
            }

            Rectangle {
                id: inputBox

                x: win.frame
                y: win.frame + (win.plan.y.input ?? 0)
                width: win.innerWidth
                height: win.inputBoxHeight
                radius: win.sizes.inputRadius
                color: win.colors.input

                // The highlight behind a recognized prefix. TextInput gives the position of a
                // character in its own space, including the scroll of a long text (research R4).
                Item {
                    // The clip grows by a margin on every side (never beyond the input's
                    // padding), so the rounded outline of the highlight is not cut off; only the
                    // scroll of a long text is clipped.
                    readonly property int margin: Math.min(win.sizes.rowGap, win.sizes.inputPadding)

                    x: input.x - margin
                    y: input.y - margin
                    width: input.width + win.twice * margin
                    height: input.height + win.twice * margin
                    clip: true

                    Rectangle {
                        readonly property rect from: {
                            input.text;
                            input.contentWidth;
                            input.width;
                            return input.positionToRectangle(0);
                        }
                        readonly property rect to: {
                            input.text;
                            input.contentWidth;
                            input.width;
                            return input.positionToRectangle(win.prefixLength);
                        }

                        visible: win.prefixLength > 0
                        x: from.x
                        y: 0
                        width: to.x - from.x + win.twice * parent.margin
                        height: parent.height
                        radius: win.sizes.rowRadius
                        color: win.colors.prefix
                    }

                }

                Text {
                    x: input.x
                    y: input.y
                    width: input.width
                    height: input.height
                    verticalAlignment: Text.AlignVCenter
                    elide: Text.ElideRight
                    visible: input.text === "" && input.preeditText === ""
                    text: win.settings.placeholder ?? ""
                    color: win.colors.placeholder
                    font.family: win.fonts.placeholder.family
                    font.pixelSize: win.fonts.placeholder.pixelSize
                    font.weight: win.fonts.placeholder.weight
                    font.italic: win.fonts.placeholder.italic
                }

                TextInput {
                    id: input

                    x: win.sizes.inputPadding
                    y: win.sizes.inputPadding
                    width: inputBox.width - win.twice * win.sizes.inputPadding
                    clip: true
                    focus: true
                    maximumLength: win.maxTextLength
                    selectByMouse: true
                    text: win.text
                    color: win.colors.inputText
                    selectionColor: win.colors.selected
                    selectedTextColor: win.colors.selectedText
                    font.family: win.fonts.input.family
                    font.pixelSize: win.fonts.input.pixelSize
                    font.weight: win.fonts.input.weight
                    font.italic: win.fonts.input.italic
                    onTextEdited: win.textEdited(text)
                    Keys.onPressed: (event) => {
                        return win.handleKey(event);
                    }
                }

            }

            Rectangle {
                visible: win.plan.sepY >= 0
                x: win.frame
                y: win.frame + win.plan.sepY
                width: win.innerWidth
                height: win.sizes.separatorWidth
                color: win.colors.separator
            }

            Text {
                id: messageLabel

                visible: win.showMessage
                x: win.frame
                y: win.frame + (win.plan.y.message ?? 0)
                width: win.innerWidth
                wrapMode: Text.WordWrap
                text: win.message
                color: win.messageKind === "error" ? win.colors.error : win.colors.message
                font.family: win.fonts.empty.family
                font.pixelSize: win.fonts.empty.pixelSize
                font.weight: win.fonts.empty.weight
                font.italic: win.fonts.empty.italic
            }

            Item {
                id: results

                x: win.frame
                y: win.frame + (win.plan.y.results ?? 0)
                width: win.innerWidth
                height: win.plan.h.results
                visible: height > 0
                clip: true

                ListView {
                    id: list

                    anchors.fill: parent
                    visible: win.rowCount > 0
                    model: win.rows
                    spacing: win.sizes.rowGap
                    boundsBehavior: Flickable.StopAtBounds

                    delegate: ResultRow {
                        required property int index
                        required property var modelData

                        width: list.width
                        height: win.sizes.rowHeight
                        look: win.look
                        sizes: win.sizes
                        entry: modelData
                        selected: index === win.selectedIndex
                        hovered: index === win.hoverIndex
                        onEntered: win.hoverIndex = index
                        onExited: {
                            if (win.hoverIndex === index)
                                win.hoverIndex = -1;

                        }
                        onClicked: win.userSelect(index)
                        onDoubleClicked: {
                            win.userSelect(index);
                            win.rowChosen(index);
                        }
                    }

                }

                Text {
                    id: emptyLabel

                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    visible: win.showEmpty
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.WordWrap
                    text: win.emptyString
                    color: win.colors.empty
                    font.family: win.fonts.empty.family
                    font.pixelSize: win.fonts.empty.pixelSize
                    font.weight: win.fonts.empty.weight
                    font.italic: win.fonts.empty.italic
                }

            }

            Row {
                id: footerRow

                visible: win.showFooter
                x: win.frame
                y: win.frame + (win.plan.y.footer ?? 0)
                width: win.innerWidth
                spacing: win.sizes.rowPadding

                Repeater {
                    model: win.settings.footerHints ?? []

                    Text {
                        required property string modelData

                        text: modelData
                        color: win.colors.footer
                        font.family: win.fonts.footer.family
                        font.pixelSize: win.fonts.footer.pixelSize
                        font.weight: win.fonts.footer.weight
                        font.italic: win.fonts.footer.italic
                    }

                }

            }

            transform: Translate {
                y: win.motion.kind === "slide" ? (1 - win.progress) * win.sizes.slideDistance * (win.sizes.anchor.indexOf("bottom") === 0 ? 1 : -1) : 0
            }

        }

    }

    Behavior on progress {
        enabled: win.motion.animated

        NumberAnimation {
            duration: win.motion.duration
            easing.type: win.motion.easing
        }

    }

}
