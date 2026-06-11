import QtQuick
import Quickshell
import Quickshell.Io

// Root of the Rainbeau theme picker. Loads the theme manifest produced by the Go
// side, exposes it to per-screen Selector windows, and writes the chosen theme
// path back to the result file before quitting.
ShellRoot {
    id: root

    property var themes: []
    property string activeBorder: "#ffffff"
    property string inactiveBorder: "#555555"
    property bool loaded: false

    readonly property string manifestPath: Quickshell.env("RAINBEAU_MANIFEST")
    readonly property string resultPath: Quickshell.env("RAINBEAU_RESULT")

    function choose(path) {
        writeProcess.command = [
            "bash", "-c",
            "printf '%s' '" + String(path).replace(/'/g, "'\"'\"'") + "' > '" + root.resultPath + "'"
        ];
        writeProcess.running = true;
    }

    function cancel() {
        Qt.quit();
    }

    Process {
        id: writeProcess
        running: false
        onExited: Qt.quit()
    }

    Process {
        id: loadProcess
        command: ["cat", root.manifestPath]
        running: true

        stdout: StdioCollector {
            id: collector
            waitForEnd: true
        }

        onExited: exitCode => {
            if (exitCode === 0) {
                try {
                    let manifest = JSON.parse(collector.text);
                    root.themes = manifest.themes || [];
                    if (manifest.activeBorder) root.activeBorder = manifest.activeBorder;
                    if (manifest.inactiveBorder) root.inactiveBorder = manifest.inactiveBorder;
                } catch (e) {
                    console.warn("Rainbeau: failed to parse manifest:", e);
                }
            }
            root.loaded = true;
        }
    }

    Variants {
        model: Quickshell.screens

        Selector {
            shell: root
        }
    }
}
