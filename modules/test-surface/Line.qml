import QtQuick
import QtQuick.Layouts

// One line of monospace text.
Text {
    required property var api

    Layout.fillWidth: true
    color: api.theme.onSurface
    font.pixelSize: api.style.font.body
    font.family: api.style.font.monoFamily
    wrapMode: Text.WrapAnywhere
}
