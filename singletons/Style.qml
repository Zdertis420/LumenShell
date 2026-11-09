import QtQuick
import Quickshell
pragma Singleton

// Design tokens shared by every Lumen component, including third-party ones.
Singleton {
    readonly property var spacing: ({
        "xs": 4,
        "sm": 8,
        "md": 12,
        "lg": 16,
        "xl": 24
    })
    readonly property var radius: ({
        "sm": 4,
        "md": 8,
        "lg": 16,
        "full": 9999
    })
    readonly property var font: ({
        "family": "sans-serif",
        "monoFamily": "monospace",
        "caption": 11,
        "body": 13,
        "title": 16,
        "headline": 22,
        "weightRegular": Font.Normal,
        "weightMedium": Font.Medium,
        "weightBold": Font.Bold
    })
    readonly property var motion: ({
        "fast": 100,
        "normal": 200,
        "slow": 400,
        "easing": Easing.OutCubic
    })
    // Line widths: `thin` is the hairline of outlines and focus rings.
    readonly property var border: ({
        "thin": 1
    })
}
