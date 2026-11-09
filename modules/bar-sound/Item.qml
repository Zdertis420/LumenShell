import QtQuick
import Quickshell.Services.Pipewire

// Output volume and mute state for the bar (an item contributed to bar/items). Hidden when
// there is no default sound output (FR-007).
// Pointer: left click opens an audio mixer, the wheel changes the volume, right click mutes.
// Keyboard (while the bar has the focus): Enter opens the mixer, Up and Down change the
// volume, M mutes (FR-016a). The keys are not changed by feature 004.
// On a vertical bar it shows the label above the level. Font, padding, corner radius and colors
// follow the bar unless this module's own settings say otherwise (ItemLook.qml).
Item {
    id: root

    required property var api
    property var slot
    // The bar's key convention (specs/003-bar/contracts/bar-slot.md): interactive items can be
    // reached with the arrow keys.
    property bool interactive: true
    readonly property int percentScale: 100 // documented constant: Pipewire reports a fraction
    readonly property int mixerTimeoutMs: 3000 // documented constant: time limit of the mixer helper
    readonly property int twice: 2 // documented constant: padding is on both sides of the text
    readonly property var sink: Pipewire.defaultAudioSink
    readonly property var audio: sink ? sink.audio : null
    readonly property bool available: audio !== null && audio !== undefined
    readonly property bool muted: available && audio.muted
    readonly property int percent: available ? Math.round(Math.min(Math.max(audio.volume, 0), 1) * percentScale) : 0
    readonly property real maxWidth: slot ? slot.maxWidth : label.implicitWidth
    readonly property real maxLength: slot ? slot.maxLength : label.implicitHeight
    readonly property bool focused: slot ? slot.focused : false

    // Changes the volume by `steps` steps of the step setting, between 0% and 100%.
    function changeVolume(steps: int) {
        if (!available)
            return ;

        const next = Math.min(Math.max(percent + steps * root.api.settings.values.step, 0), percentScale);
        audio.volume = next / percentScale;
    }

    function toggleMute() {
        if (available)
            audio.muted = !audio.muted;

    }

    // Starts the mixer program through the module's helper. A missing mixer is only logged.
    function openMixer() {
        root.api.run("open-mixer", {
            "command": root.api.settings.values.mixer
        }, root.mixerTimeoutMs, (req) => {
            if (req.state !== "ok")
                root.api.log.warn("mixer: " + (req.error ? req.error.message : req.state));

        });
    }

    visible: available
    implicitWidth: !available ? 0 : (look.vertical ? look.thickness : Math.min(label.implicitWidth + look.padding * twice, maxWidth))
    implicitHeight: !available ? 0 : (look.vertical ? Math.min(label.implicitHeight + look.padding * twice, maxLength) : (slot ? slot.thickness : label.implicitHeight))
    Keys.onPressed: (event) => {
        if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
            root.openMixer();
            event.accepted = true;
        } else if (event.key === Qt.Key_Up) {
            root.changeVolume(1);
            event.accepted = true;
        } else if (event.key === Qt.Key_Down) {
            root.changeVolume(-1);
            event.accepted = true;
        } else if (event.key === Qt.Key_M) {
            root.toggleMute();
            event.accepted = true;
        }
    }

    ItemLook {
        id: look

        api: root.api
        slot: root.slot
    }

    // The default sink's properties are kept up to date by the tracker in Main.qml (one per
    // module, not one per item instance).
    // The item's own background, when its settings ask for one.
    Rectangle {
        anchors.fill: parent
        visible: look.hasBackground
        color: look.backgroundColor
        radius: look.radius
    }

    // The visible focus indicator.
    Rectangle {
        anchors.fill: parent
        anchors.margins: root.api.style.border.thin
        visible: root.focused
        color: "transparent"
        radius: look.radius
        border.width: root.api.style.border.thin
        border.color: look.focusColor
    }

    Text {
        id: label

        anchors.centerIn: parent
        width: Math.min(implicitWidth, root.width)
        horizontalAlignment: Text.AlignHCenter
        elide: look.vertical ? Text.ElideNone : Text.ElideRight
        // Vertical: the label above the level ("Vol" over "45%", "Vol" over "off").
        text: look.vertical ? "Vol\n" + (root.muted ? "off" : root.percent + "%") : (root.muted ? "Vol off" : "Vol " + root.percent + "%")
        color: root.muted ? root.api.theme.onSurfaceVariant : look.textColor
        font.family: look.family
        font.pixelSize: look.pixelSize
        font.weight: look.weight
        font.italic: look.italic
    }

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        onClicked: (mouse) => {
            if (mouse.button === Qt.RightButton)
                root.toggleMute();
            else
                root.openMixer();
        }
        onWheel: (wheel) => {
            if (wheel.angleDelta.y > 0)
                root.changeVolume(1);
            else if (wheel.angleDelta.y < 0)
                root.changeVolume(-1);
        }
    }

    Connections {
        function onPaletteIdChanged() {
            root.api.log.debug("theme " + root.api.theme.paletteId);
        }

        target: root.api.theme
    }

}
