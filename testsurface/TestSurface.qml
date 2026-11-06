import "../singletons"
import QtQuick
import QtQuick.Layouts
import Quickshell

// Minimal surface that shows the five core singletons working. For verification only.
// Colors come only from Theme; spacing, radii, fonts and durations only from Style.
FloatingWindow {
    id: win

    readonly property var roleNames: ["primary", "onPrimary", "secondary", "onSecondary", "tertiary", "onTertiary", "error", "onError", "background", "onBackground", "surface", "onSurface", "surfaceVariant", "onSurfaceVariant", "surfaceContainer", "outline"]
    readonly property int unit: Style.spacing.xl

    title: "Lumen test surface"
    color: Theme.background
    implicitWidth: Style.spacing.xl * 32
    implicitHeight: Style.spacing.xl * 26

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Style.spacing.lg
        spacing: Style.spacing.md

        RowLayout {
            Layout.fillWidth: true

            Text {
                text: "Lumen"
                color: Theme.onBackground
                font.pixelSize: Style.font.headline
                font.weight: Style.font.weightBold
            }

            Item {
                Layout.fillWidth: true
            }

            Text {
                text: Theme.paletteId + " · " + Theme.mode + " · " + Theme.status
                color: Theme.onBackground
                font.pixelSize: Style.font.body
            }

        }

        Section {
            heading: "Theme"

            Flow {
                Layout.fillWidth: true
                spacing: Style.spacing.sm

                Repeater {
                    model: win.roleNames

                    Rectangle {
                        required property string modelData

                        width: Style.spacing.xl * 3
                        height: win.unit
                        radius: Style.radius.sm
                        color: Theme[modelData]
                        border.color: Theme.outline

                        Text {
                            anchors.centerIn: parent
                            text: parent.modelData
                            color: Theme.onSurface
                            font.pixelSize: Style.font.caption
                        }

                        Behavior on color {
                            ColorAnimation {
                                duration: Style.motion.fast
                            }

                        }

                    }

                }

            }

            Line {
                visible: Theme.reason !== ""
                text: "status: " + Theme.status + " — " + Theme.reason
            }

        }

        Section {
            heading: "Config"

            Line {
                text: JSON.stringify(Config.values)
            }

            Repeater {
                model: Config.errors

                Line {
                    required property var modelData

                    color: Theme.error
                    text: modelData.key + ": " + modelData.problem + " (fallback: " + modelData.fallback + ")"
                }

            }

        }

        Section {
            heading: "Backend"

            Line {
                text: "helper runs: " + Backend.runCount + " · helper dir: " + Backend.helperDir
            }

            Repeater {
                model: Object.keys(Backend.lastErrors)

                Line {
                    required property string modelData

                    color: Theme.error
                    text: modelData + ": " + Backend.lastErrors[modelData].state + " — " + Backend.lastErrors[modelData].message
                }

            }

        }

        Section {
            heading: "Style"

            RowLayout {
                spacing: Style.spacing.sm

                Repeater {
                    model: Object.keys(Style.spacing)

                    Rectangle {
                        required property string modelData

                        implicitWidth: Style.spacing[modelData] * 2
                        implicitHeight: Style.spacing.xl
                        radius: Style.radius.sm
                        color: Theme.secondary
                    }

                }

            }

        }

        Section {
            heading: "Log"
            Layout.fillHeight: true

            Repeater {
                model: Log.recent()

                Line {
                    required property var modelData

                    font.pixelSize: Style.font.caption
                    color: modelData.level === "error" ? Theme.error : Theme.onSurface
                    text: Log.brief(modelData)
                    elide: Text.ElideRight
                    maximumLineCount: 1
                }

            }

        }

    }

    component Section: Rectangle {
        id: section

        property string heading: ""
        default property alias content: body.data

        Layout.fillWidth: true
        implicitHeight: body.implicitHeight + Style.spacing.lg * 2
        color: Theme.surfaceContainer
        radius: Style.radius.md

        ColumnLayout {
            id: body

            anchors.fill: parent
            anchors.margins: Style.spacing.lg
            spacing: Style.spacing.sm

            Text {
                text: section.heading
                color: Theme.primary
                font.pixelSize: Style.font.title
                font.weight: Style.font.weightBold
            }

        }

        Behavior on color {
            ColorAnimation {
                duration: Style.motion.fast
            }

        }

    }

    component Line: Text {
        Layout.fillWidth: true
        color: Theme.onSurface
        font.pixelSize: Style.font.body
        font.family: Style.font.monoFamily
        wrapMode: Text.WrapAnywhere
    }

}
