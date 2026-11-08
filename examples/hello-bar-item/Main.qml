import QtQuick
import Quickshell

// The entry file is loaded when the module is enabled. This module has no state of its own,
// so it is empty: the bar creates Item.qml once for every bar.
Scope {
    required property var api
}
