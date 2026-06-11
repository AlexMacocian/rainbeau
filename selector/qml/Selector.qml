import QtQuick
import Quickshell
import Quickshell.Hyprland
import Quickshell.Wayland

// Full-width, centered overlay shown on the focused screen only. Hosts the
// horizontal carousel of parallelogram theme tiles and handles keyboard
// navigation.
PanelWindow {
    id: win

    required property var shell
    property var modelData

    screen: modelData

    // Only show on the focused monitor; without this, Variants would draw an
    // identical (unfocused, non-interactive) copy on every other screen.
    visible: Hyprland.focusedMonitor && modelData && Hyprland.focusedMonitor.name === modelData.name

    color: "transparent"
    exclusiveZone: 0

    anchors {
        left: true
        right: true
        top: true
        bottom: true
    }

    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.keyboardFocus: WlrKeyboardFocus.Exclusive

    readonly property int tileWidth: Math.round(Math.min(360, width / 4))

    Rectangle {
        id: backdrop
        anchors.fill: parent
        color: "#cc0b0b0f"

        MouseArea {
            anchors.fill: parent
            onClicked: win.shell.cancel()
        }

        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.top: parent.top
            anchors.topMargin: parent.height * 0.12
            text: "Select a theme"
            color: "#f0f0f0"
            font.pixelSize: 28
            font.bold: true
        }

        Text {
            visible: win.shell.loaded && (!win.shell.themes || win.shell.themes.length === 0)
            anchors.centerIn: parent
            text: "No themes found"
            color: "#f0f0f0"
            font.pixelSize: 22
        }

        ListView {
            id: list

            anchors.left: parent.left
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            height: parent.height * 0.55

            orientation: ListView.Horizontal
            model: win.shell.themes
            spacing: 6
            focus: true
            clip: false

            highlightRangeMode: ListView.StrictlyEnforceRange
            preferredHighlightBegin: (width - win.tileWidth) / 2
            preferredHighlightEnd: (width + win.tileWidth) / 2
            highlightMoveDuration: 220
            highlightMoveVelocity: -1

            delegate: ThemeTile {
                width: win.tileWidth
                height: list.height
                themeName: modelData.name
                thumbnailPath: modelData.thumbnail
                imagePath: modelData.image || ""
                fontFamily: modelData.font || ""
                bgColor: (modelData.colors && modelData.colors.bg0) ? modelData.colors.bg0 : "#1a1610"
                activeBorderColor: win.shell.activeBorder
                inactiveBorderColor: win.shell.inactiveBorder
                isCurrent: ListView.isCurrentItem

                onChosen: {
                    list.currentIndex = index;
                    win.shell.choose(modelData.path);
                }
            }

            Keys.onReturnPressed: if (currentIndex >= 0 && win.shell.themes.length > 0)
                win.shell.choose(win.shell.themes[currentIndex].path)
            Keys.onEnterPressed: if (currentIndex >= 0 && win.shell.themes.length > 0)
                win.shell.choose(win.shell.themes[currentIndex].path)
            Keys.onEscapePressed: win.shell.cancel()
            Keys.onPressed: event => {
                if (event.key === Qt.Key_Q) {
                    win.shell.cancel();
                    event.accepted = true;
                }
            }
        }
    }
}
