import QtQuick
import QtQuick.Layouts
import Quickshell

// Lumen's test surface, as a bundled module. It uses only the public module interface
// (the `api` object): no singleton is touched directly.
Scope {
    id: root

    required property var api
    readonly property var roleNames: ["primary", "onPrimary", "secondary", "onSecondary", "tertiary", "onTertiary", "error", "onError", "background", "onBackground", "surface", "onSurface", "surfaceVariant", "onSurfaceVariant", "surfaceContainer", "outline"]

    // "HH:MM:SS level source message"
    function brief(entry) : string {
        return String(entry.time).split("T")[1].split(".")[0] + " " + entry.level + " " + entry.source + " " + entry.msg;
    }

    Component.onCompleted: api.log.info("test surface loaded")

    FloatingWindow {
        title: "Lumen test surface"
        color: root.api.theme.background
        implicitWidth: root.api.style.spacing.xl * 32
        implicitHeight: root.api.style.spacing.xl * 26

        ColumnLayout {
            anchors.fill: parent
            anchors.margins: root.api.style.spacing.lg
            spacing: root.api.style.spacing.md

            RowLayout {
                Layout.fillWidth: true

                Text {
                    text: "Lumen"
                    color: root.api.theme.onBackground
                    font.pixelSize: root.api.style.font.headline
                    font.weight: root.api.style.font.weightBold
                }

                Item {
                    Layout.fillWidth: true
                }

                Text {
                    text: root.api.theme.paletteId + " · " + root.api.theme.mode + " · " + root.api.theme.status
                    color: root.api.theme.onBackground
                    font.pixelSize: root.api.style.font.body
                }

            }

            Section {
                api: root.api
                heading: "Theme"

                Flow {
                    Layout.fillWidth: true
                    spacing: root.api.style.spacing.sm

                    Repeater {
                        model: root.roleNames

                        Rectangle {
                            required property string modelData

                            width: root.api.style.spacing.xl * 3
                            height: root.api.style.spacing.xl
                            radius: root.api.style.radius.sm
                            color: root.api.theme[modelData]
                            border.color: root.api.theme.outline

                            Text {
                                anchors.centerIn: parent
                                text: parent.modelData
                                color: root.api.theme.onSurface
                                font.pixelSize: root.api.style.font.caption
                            }

                            Behavior on color {
                                ColorAnimation {
                                    duration: root.api.style.motion.fast
                                }

                            }

                        }

                    }

                }

                Line {
                    api: root.api
                    visible: root.api.theme.reason !== ""
                    text: "status: " + root.api.theme.status + " — " + root.api.theme.reason
                }

            }

            Section {
                api: root.api
                heading: "Config"

                Line {
                    api: root.api
                    text: JSON.stringify(root.api.config.values)
                }

                Repeater {
                    model: root.api.config.errors

                    Line {
                        required property var modelData

                        api: root.api
                        color: root.api.theme.error
                        text: modelData.key + ": " + modelData.problem + " (fallback: " + modelData.fallback + ")"
                    }

                }

            }

            Section {
                api: root.api
                heading: "Modules"

                Repeater {
                    model: root.api.modules.list

                    Line {
                        required property var modelData

                        api: root.api
                        color: modelData.state === "failed" ? root.api.theme.error : root.api.theme.onSurface
                        text: modelData.id + " " + modelData.version + " " + modelData.state + (modelData.lastError ? " — " + modelData.lastError : "") + (modelData.settingErrors.length > 0 ? " — settings: " + modelData.settingErrors.map((e) => {
                            return e.key + " " + e.problem;
                        }).join("; ") : "")
                    }

                }

            }

            Section {
                api: root.api
                heading: "Style"

                RowLayout {
                    spacing: root.api.style.spacing.sm

                    Repeater {
                        model: Object.keys(root.api.style.spacing)

                        Rectangle {
                            required property string modelData

                            implicitWidth: root.api.style.spacing[modelData] * 2
                            implicitHeight: root.api.style.spacing.xl
                            radius: root.api.style.radius.sm
                            color: root.api.theme.secondary
                        }

                    }

                }

            }

            Section {
                api: root.api
                heading: "Log"
                Layout.fillHeight: true

                Repeater {
                    model: root.api.log.recent()

                    Line {
                        required property var modelData

                        api: root.api
                        font.pixelSize: root.api.style.font.caption
                        color: modelData.level === "error" ? root.api.theme.error : root.api.theme.onSurface
                        text: root.brief(modelData)
                        elide: Text.ElideRight
                        maximumLineCount: 1
                    }

                }

            }

        }

    }

}
