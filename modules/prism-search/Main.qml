import QtQuick
import Quickshell

// The entry file is loaded when the module is enabled. The module has no state of its own:
// Prism creates Source.qml (the search) once, through its extension point "sources".
Scope {
    required property var api
}
