import QtQuick
import QtQuick.Layouts

// A titled panel. Colors, spacing, radii and durations all come from the api object.
Rectangle {
    id: section

    required property var api
    property string heading: ""
    default property alias content: body.data

    Layout.fillWidth: true
    implicitHeight: body.implicitHeight + api.style.spacing.lg * 2
    color: api.theme.surfaceContainer
    radius: api.style.radius.md

    ColumnLayout {
        id: body

        anchors.fill: parent
        anchors.margins: section.api.style.spacing.lg
        spacing: section.api.style.spacing.sm

        Text {
            text: section.heading
            color: section.api.theme.primary
            font.pixelSize: section.api.style.font.title
            font.weight: section.api.style.font.weightBold
        }

    }

    Behavior on color {
        ColorAnimation {
            duration: section.api.style.motion.fast
        }

    }

}
