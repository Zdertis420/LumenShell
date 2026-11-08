import QtQuick
import Quickshell
import Quickshell.Services.Pipewire

// Keeps the default sound output's properties up to date while the module is loaded. It lives
// here and not in Item.qml, which the bar creates once for every bar: a tracker per item
// instance leaked memory on every bar reload (measured, specs/003-bar/research.md).
Scope {
    required property var api

    PwObjectTracker {
        objects: Pipewire.defaultAudioSink ? [Pipewire.defaultAudioSink] : []
    }

}
