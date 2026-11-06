// The only source of color in Lumen. Components read semantic roles (primary, surface,
// onSurface, ...) from here and never hardcode a color.

import QtQuick
import Quickshell
import Quickshell.Io
pragma Singleton

// The palette comes from a bundled preset or from the wallpaper (through matugen). The
// `palette` helper program does the work and guarantees WCAG AA text contrast; this
// singleton runs it asynchronously and applies the result in one assignment, so every
// consumer updates together with no flicker. The previous palette stays on screen until
// the new one has arrived.
Singleton {
    id: root

    // Which palette is shown: a preset id, "wallpaper", or "fallback".
    property string paletteId: "fallback"
    property string mode: "dark"
    // "ok": the requested palette is shown.
    // "fallback": the built-in palette is shown; `reason` says why.
    // "stale": the previous palette was kept; `reason` says why.
    property string status: "fallback"
    property string reason: "loading"
    // The 16 semantic roles as one object; the color properties below bind to it.
    property var roles: ({
    })
    property color primary: roles.primary ?? "transparent"
    property color onPrimary: roles.onPrimary ?? "transparent"
    property color secondary: roles.secondary ?? "transparent"
    property color onSecondary: roles.onSecondary ?? "transparent"
    property color tertiary: roles.tertiary ?? "transparent"
    property color onTertiary: roles.onTertiary ?? "transparent"
    property color error: roles.error ?? "transparent"
    property color onError: roles.onError ?? "transparent"
    property color background: roles.background ?? "transparent"
    property color onBackground: roles.onBackground ?? "transparent"
    property color surface: roles.surface ?? "transparent"
    property color onSurface: roles.onSurface ?? "transparent"
    property color surfaceVariant: roles.surfaceVariant ?? "transparent"
    property color onSurfaceVariant: roles.onSurfaceVariant ?? "transparent"
    property color surfaceContainer: roles.surfaceContainer ?? "transparent"
    property color outline: roles.outline ?? "transparent"
    // True once a requested (non-fallback) palette has been applied.
    property bool hasGoodPalette: false
    // Identifies the inputs of the palette on screen, to skip needless reruns.
    property string appliedKey: ""
    // path:mtime of the wallpaper behind the palette on screen (wallpaper source only).
    property string wallpaperKey: ""
    property int generation: 0

    function inputs() : variant {
        return {
            "source": Config.value("theme.source"),
            "preset": Config.value("theme.preset"),
            "wallpaper": Config.value("theme.wallpaper"),
            "scheme": Config.value("theme.scheme"),
            "mode": Config.value("theme.mode")
        };
    }

    // Runs the palette helper for the current configuration. Without `force`, it does
    // nothing when the inputs have not changed since the palette on screen.
    function refresh(force: bool) {
        if (!Config.ready)
            return ;

        const input = root.inputs();
        const key = JSON.stringify(input);
        if (!force && key === root.appliedKey && root.hasGoodPalette)
            return ;

        const gen = ++root.generation;
        input.dataDir = Backend.dataDir;
        input.hasPrevious = root.hasGoodPalette;
        Backend.run("palette", [], input, 0, (req) => {
            if (gen !== root.generation)
                return ;

            root.handle(req, key, input.source);
        });
    }

    function handle(req, key: string, source: string) {
        if (req.state !== "ok") {
            // The helper itself failed or hung. Keep what is shown.
            const why = "palette helper " + req.state + ": " + (req.error ? req.error.message : "");
            root.status = root.hasGoodPalette ? "stale" : "fallback";
            root.reason = why;
            return ;
        }
        const data = req.data;
        if (data.status === "stale") {
            root.status = "stale";
            root.reason = data.reason;
            Log.warn("theme: keeping previous palette: " + data.reason);
            if (source === "wallpaper")
                root.wallpaperKey = data.wallpaper + ":" + data.wallpaperMtime;

            return ;
        }
        root.apply(data.palette);
        root.appliedKey = key;
        if (data.status === "fallback") {
            root.hasGoodPalette = false;
            root.status = "fallback";
            root.reason = data.reason;
            Log.warn("theme: using fallback palette: " + data.reason);
        } else {
            root.hasGoodPalette = true;
            root.status = "ok";
            root.reason = "";
            Log.info("theme applied", {
                "palette": data.palette.id,
                "mode": data.palette.mode,
                "adjusted": data.adjusted.length
            });
        }
        if (source === "wallpaper")
            root.wallpaperKey = data.wallpaper + ":" + data.wallpaperMtime;

    }

    // Replaces all roles at once.
    function apply(palette) {
        root.roles = palette.roles;
        root.paletteId = palette.id;
        root.mode = palette.mode;
    }

    // Detects a wallpaper change (a different file, or the same file rewritten).
    function probeWallpaper() {
        if (Config.value("theme.source") !== "wallpaper")
            return ;

        Backend.run("palette", [], {
            "dataDir": Backend.dataDir,
            "source": "wallpaper",
            "wallpaper": Config.value("theme.wallpaper"),
            "detectOnly": true
        }, 0, (req) => {
            if (req.state !== "ok")
                return ;

            const now = req.data.wallpaper + ":" + req.data.wallpaperMtime;
            if (root.wallpaperKey !== "" && now !== root.wallpaperKey)
                root.refresh(true);

        });
    }

    Connections {
        function onReloaded() {
            root.refresh(false);
        }

        target: Config
    }

    // Wallpaper tools (awww, swww) give no change event, so a cheap check runs while the
    // wallpaper source is selected.
    Timer {
        interval: 2000
        repeat: true
        running: Config.ready && Config.value("theme.source") === "wallpaper"
        onTriggered: root.probeWallpaper()
    }

    // Initial colors come from the fallback palette file, so QML holds no color literals.
    FileView {
        path: Quickshell.shellDir + "/data/fallback-palette.json"
        blockLoading: true
        onLoaded: {
            if (root.paletteId === "fallback" && !root.hasGoodPalette) {
                try {
                    root.apply(JSON.parse(text()));
                } catch (e) {
                    Log.error("fallback palette unreadable: " + e);
                }
            }
        }
    }

}
