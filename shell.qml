import QtQuick
import Quickshell
import Quickshell.Io
import "singletons"
import "testsurface"

ShellRoot {
    id: root

    // True once the startup checks pass; the test surface only appears then.
    property bool checksPassed: false

    // Compositor-specific check (niri only for now). LUMEN_SKIP_SESSION_CHECK=1 is for tests.
    function checkSession() : bool {
        if (Quickshell.env("LUMEN_SKIP_SESSION_CHECK") === "1")
            return true;

        const socket = Quickshell.env("NIRI_SOCKET");
        return socket !== null && socket !== undefined && socket !== "";
    }

    // Reports the problem and stops the shell. Qt.quit() has no receiver in Quickshell
    // 0.3, so the shell asks the system to end its own process.
    function fail(msg: string) {
        Log.error(msg);
        Quickshell.execDetached(["kill", "-TERM", String(Quickshell.processId)]);
    }

    // QML singletons are created lazily. Touch all of them at startup so each one
    // exists before anything (for example an IPC call) needs it.
    Component.onCompleted: {
        Log.info("lumen shell starting", {
            "backend": Backend.ready,
            "config": Config.ready,
            "theme": Theme.paletteId,
            "spacing": Style.spacing.md
        });
        if (!checkSession()) {
            fail("not running under niri (NIRI_SOCKET is not set); exiting");
            return ;
        }
        binariesCheck.running = true;
    }

    // The helper programs must be built. Checked without blocking the UI thread.
    Process {
        id: binariesCheck

        command: ["sh", "-c", "test -x \"$1/config\" && test -x \"$1/palette\"", "sh", Backend.helperDir]
        onExited: (exitCode) => {
            if (exitCode === 0)
                root.checksPassed = true;
            else
                root.fail("helper programs missing in " + Backend.helperDir + "; run `make build`");
        }
    }

    LazyLoader {
        active: root.checksPassed

        TestSurface {
        }

    }

}
