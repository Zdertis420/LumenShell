import QtQuick
import Quickshell.Services.Pipewire

// Output volume and mute state for the bar (an item contributed to bar/items). Hidden when
// there is no default sound output (FR-007).
// Pointer: left click opens an audio mixer, the wheel changes the volume, right click mutes.
// Keyboard (while the bar has the focus): Enter opens the mixer, Up and Down change the
// volume, M mutes (FR-016a).
Item {
    id: root

    required property var api
    property var slot
    // The bar's key convention (specs/003-bar/contracts/bar-slot.md): interactive items can be
    // reached with the arrow keys.
    property bool interactive: true
    readonly property int percentScale: 100 // documented constant: Pipewire reports a fraction
    readonly property int mixerTimeoutMs: 3000 // documented constant: time limit of the mixer helper
    readonly property var sink: Pipewire.defaultAudioSink
    readonly property var audio: sink ? sink.audio : null
    readonly property bool available: audio !== null && audio !== undefined
    readonly property bool muted: available && audio.muted
    readonly property int percent: available ? Math.round(Math.min(Math.max(audio.volume, 0), 1) * percentScale) : 0
    readonly property real maxWidth: slot ? slot.maxWidth : label.implicitWidth
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
    implicitWidth: available ? Math.min(label.implicitWidth + api.style.spacing.md * 2, maxWidth) : 0
    implicitHeight: slot ? slot.thickness : label.implicitHeight
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

    // The default sink's properties are kept up to date by the tracker in Main.qml (one per
    // module, not one per item instance).
    // The visible focus indicator.
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
        width: Math.min(implicitWidth, root.width)
        elide: Text.ElideRight
        text: root.muted ? "Vol off" : "Vol " + root.percent + "%"
        color: root.muted ? root.api.theme.onSurfaceVariant : root.api.theme.onSurface
        font.family: root.api.style.font.family
        font.pixelSize: root.api.style.font.body
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
