import QtQuick
import QtQuick.Shapes
import Quickshell
import Quickshell.Wayland
import "geometry.js" as Geometry

// The frame style's extra surfaces on one screen (specs/004-bar-customization-styles,
// contracts/bar-surfaces.md, research.md R4 and R4a): a border panel on each side that has no
// bar (each reserves its own width), and four corner blocks. The compositor lays a panel out
// inside the space that earlier panels reserved, so the panels can come up short at the screen
// corners; each corner block starts at its screen corner, ignores the other surfaces' zones,
// and covers the whole corner square in the frame color except for the quarter disc of the
// rounded inner corner. The strip panel (BarWindow.qml) is the last part; all share the frame
// role color, so there is no seam. The panels take no keyboard focus; the blocks take no input
// and reserve nothing.
// BarWindow creates this through a LazyLoader that is active only in the frame style, and
// each part below exists only while it has a size.
Scope {
    id: root

    required property var api
    required property var look
    // The sizes of this screen (geometry.js): edge, vertical, frameBorder, frameRadius, reserved.
    required property var sizes
    required property var screen
    readonly property string edge: root.sizes.edge
    readonly property bool vertical: root.sizes.vertical
    readonly property int border: root.sizes.frameBorder
    readonly property var reserved: root.sizes.reserved
    readonly property color frameColor: root.look.colors.frame
    // The sides that carry a border: every side but the bar's.
    readonly property var borderSides: root.border > 0 ? ["left", "right", "top", "bottom"].filter((s) => {
        return s !== root.edge;
    }) : []
    // The corner blocks that have a size, also for a radius of 0 (geometry.js cornerBlocks).
    readonly property var blocks: Geometry.cornerBlocks(root.sizes)

    // Border panels. The side parallel to the bar spans the whole screen; the two across it
    // run between the bar's strip and that opposite side, so no two panels overlap.
    Variants {
        model: root.borderSides

        PanelWindow {
            // No margins (research R4b): the compositor already lays this panel out inside the
            // space that the strip and the other panels reserved, so a margin for that space
            // would count twice and leave a gap next to the corner block.

            id: panel

            required property string modelData
            readonly property bool leftRight: modelData === "left" || modelData === "right"
            readonly property bool acrossBar: root.vertical ? (modelData === "top" || modelData === "bottom") : (modelData === "left" || modelData === "right")
            // The space the strip takes on the bar's edge, and the border on the opposite side.
            readonly property int stripSpace: root.reserved[root.edge]

            screen: root.screen
            color: root.frameColor
            exclusionMode: ExclusionMode.Auto
            WlrLayershell.namespace: "lumen-bar-border-" + modelData
            // Both sizes are given, so the toolkit never draws a first frame at a default size:
            // across the panel its border width, along it the room between the strip and the
            // opposite border (or the whole screen for the side parallel to the bar).
            implicitWidth: leftRight ? root.border : (root.vertical ? root.screen.width - stripSpace - root.border : root.screen.width)
            implicitHeight: leftRight ? (root.vertical ? root.screen.height : root.screen.height - stripSpace - root.border) : root.border

            anchors {
                left: modelData === "left" || (modelData !== "right" && (root.vertical ? acrossBar : true))
                right: modelData === "right" || (modelData !== "left" && (root.vertical ? acrossBar : true))
                top: modelData === "top" || (modelData !== "bottom" && (root.vertical ? true : acrossBar))
                bottom: modelData === "bottom" || (modelData !== "top" && (root.vertical ? true : acrossBar))
            }

            Connections {
                function onPaletteIdChanged() {
                    root.api.log.debug("bar border " + panel.modelData + " follows theme " + root.api.theme.paletteId);
                }

                target: root.api.theme
            }

        }

    }

    // Corner blocks: each covers the corner square of the frame, so the frame is unbroken at the
    // screen corner, and leaves a quarter disc transparent so the area inside looks rounded.
    Variants {
        model: root.blocks

        PanelWindow {
            id: block

            required property var modelData
            readonly property int r: modelData.radius
            // The band's points as the toolkit's point type, closed by repeating the first.
            readonly property var bandPoints: modelData.band.concat([modelData.band[0]]).map((p) => {
                return Qt.point(p.x, p.y);
            })

            screen: root.screen
            color: "transparent"
            exclusionMode: ExclusionMode.Ignore
            WlrLayershell.namespace: "lumen-bar-corner-" + modelData.id
            implicitWidth: modelData.width
            implicitHeight: modelData.height

            anchors {
                top: modelData.atTop
                bottom: !modelData.atTop
                left: modelData.atLeft
                right: !modelData.atLeft
            }

            Shape {
                anchors.fill: parent
                preferredRendererType: Shape.CurveRenderer

                // The band: the block without its inner square (an L), up to the screen corner.
                ShapePath {
                    fillColor: root.frameColor
                    strokeColor: "transparent"
                    // The six corners of the L, as one closed polyline (the first point is the start).
                    startX: block.bandPoints[0].x
                    startY: block.bandPoints[0].y

                    PathPolyline {
                        path: block.bandPoints
                    }

                }

                // The spandrel: the inner square without the quarter disc, which rounds the corner
                // of the area inside the frame. Nothing is drawn for a radius of 0.
                ShapePath {
                    fillColor: block.r > 0 ? root.frameColor : "transparent"
                    strokeColor: "transparent"
                    startX: block.modelData.p0.x
                    startY: block.modelData.p0.y

                    PathLine {
                        x: block.modelData.p1.x
                        y: block.modelData.p1.y
                    }

                    PathArc {
                        x: block.modelData.p2.x
                        y: block.modelData.p2.y
                        radiusX: block.r
                        radiusY: block.r
                        direction: block.modelData.clockwise ? PathArc.Clockwise : PathArc.Counterclockwise
                    }

                    PathLine {
                        x: block.modelData.p0.x
                        y: block.modelData.p0.y
                    }

                }

            }

            Connections {
                function onPaletteIdChanged() {
                    root.api.log.debug("bar corner " + block.modelData.id + " follows theme " + root.api.theme.paletteId);
                }

                target: root.api.theme
            }

            // No input at all: an empty region.
            mask: Region {
            }

        }

    }

}
