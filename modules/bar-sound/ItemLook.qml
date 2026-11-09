import QtQuick

// What a bundled bar item looks like (specs/004-bar-customization-styles, contracts/bar-config-1.2.md
// "Items"): the item's own setting if set, else the bar's value from the slot, else the design
// token (when the item runs without a bar). This file is identical in bar-clock, bar-sound and
// bar-battery, because a module cannot import code from another module; a test keeps the three
// copies equal.
QtObject {
    id: look

    required property var api
    property var slot: null
    readonly property var own: api.settings.values
    readonly property var barFont: slot ? slot.font : null
    readonly property var weightByName: ({
        "thin": Font.Thin,
        "light": Font.Light,
        "regular": Font.Normal,
        "medium": Font.Medium,
        "semibold": Font.DemiBold,
        "bold": Font.Bold,
        "black": Font.Black
    })
    // Font.
    readonly property string family: own.fontFamily ?? (barFont ? barFont.family : api.style.font.family)
    readonly property int pixelSize: own.fontSize ?? (barFont ? barFont.pixelSize : api.style.font.body)
    readonly property int weight: own.fontWeight ? (weightByName[own.fontWeight] ?? Font.Normal) : (barFont ? barFont.weight : Font.Normal)
    readonly property bool italic: own.fontItalic ?? (barFont ? barFont.italic : false)
    // Shape and spacing.
    readonly property int padding: own.padding ?? (slot ? slot.itemPadding : api.style.spacing.md)
    readonly property int radius: own.radius ?? (slot ? slot.itemRadius : api.style.radius.sm)
    // Direction and size across the bar.
    readonly property bool vertical: slot ? slot.vertical : false
    readonly property int thickness: slot ? slot.thickness : 0
    // Colors: theme roles chosen by name, following the theme live (the roles object is replaced as a whole).
    readonly property var roles: api.theme.roles
    readonly property string textRole: own.textRole ?? (slot ? slot.textRole : "onSurface")
    readonly property color textColor: roles[textRole] ?? api.theme.onSurface
    readonly property bool hasBackground: own.backgroundRole !== null && own.backgroundRole !== undefined
    readonly property color backgroundColor: hasBackground ? Qt.alpha(roles[own.backgroundRole] ?? "transparent", own.backgroundOpacity ?? 1) : "transparent"
    readonly property color focusColor: roles[slot ? slot.focusRole : "primary"] ?? api.theme.primary
    // One debug line when what the item looks like changes (the smoke test looks for it).
    readonly property string summary: "family=" + family + " size=" + pixelSize + " weight=" + weight + " italic=" + italic + " padding=" + padding + " radius=" + radius + " role=" + textRole + " vertical=" + vertical

    onSummaryChanged: api.log.debug("look " + summary)
    Component.onCompleted: api.log.debug("look " + summary)
}
