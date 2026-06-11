import QtQuick

// A single theme preview rendered as a skewed parallelogram: horizontal edges stay
// flat while vertical edges tilt. The focused tile animates more upright and larger.
Item {
    id: tile

    property string themeName: ""
    property string thumbnailPath: ""
    property string imagePath: ""
    property string fontFamily: ""
    property string bgColor: "#1a1610"
    property string activeBorderColor: "#ffffff"
    property string inactiveBorderColor: "#555555"
    property bool isCurrent: false

    signal chosen()

    // All cards keep the same shear; selection is shown via the border color and a
    // subtle scale-up that brings the focused card to the front.
    readonly property real shearAmount: -0.22
    z: isCurrent ? 1 : 0

    Item {
        id: card
        anchors.centerIn: parent
        width: tile.width
        height: tile.height
        scale: tile.isCurrent ? 1.06 : 1.0

        Behavior on scale { NumberAnimation { duration: 160; easing.type: Easing.OutCubic } }

        transform: Matrix4x4 {
            matrix: Qt.matrix4x4(
                1, tile.shearAmount, 0, 0,
                0, 1, 0, 0,
                0, 0, 1, 0,
                0, 0, 0, 1)
        }

        Rectangle {
            anchors.fill: parent
            color: tile.bgColor
            clip: true
            border.color: tile.isCurrent ? tile.activeBorderColor : tile.inactiveBorderColor
            border.width: 4

            // Palette placeholder: rendered up front, always available, fills the
            // whole card.
            Image {
                id: paletteImage
                anchors.fill: parent
                anchors.margins: 4
                source: tile.thumbnailPath ? "file://" + tile.thumbnailPath : ""
                fillMode: Image.Stretch
                asynchronous: true
                cache: false
            }

            // Real preview: rendered in the background, may appear on disk after
            // the picker is shown. It fades in over the placeholder once ready.
            // Its bottom edge stops short so the palette placeholder stays visible
            // as a thin strip along the bottom of the card.
            Image {
                id: realImage
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.bottom: parent.bottom
                anchors.leftMargin: 4
                anchors.rightMargin: 4
                anchors.topMargin: 4
                anchors.bottomMargin: 30
                fillMode: Image.PreserveAspectCrop
                clip: true
                asynchronous: true
                cache: false
                opacity: status === Image.Ready ? 1 : 0
                source: tile.imagePath ? "file://" + tile.imagePath : ""

                Behavior on opacity { NumberAnimation { duration: 220 } }

                // The background-generated file may not exist yet when first
                // loaded; retry by toggling the source until the load succeeds.
                Timer {
                    interval: 400
                    repeat: true
                    running: tile.imagePath !== "" && realImage.status !== Image.Ready
                    onTriggered: {
                        let target = "file://" + tile.imagePath;
                        realImage.source = "";
                        realImage.source = target;
                    }
                }
            }

            Rectangle {
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 30
                height: parent.height * 0.28
                gradient: Gradient {
                    GradientStop { position: 0.0; color: "#00000000" }
                    GradientStop { position: 1.0; color: "#cc000000" }
                }
            }

            Text {
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                anchors.bottomMargin: 40
                anchors.leftMargin: 10
                anchors.rightMargin: 10
                text: tile.themeName
                color: "#ffffff"
                font.family: tile.fontFamily || undefined
                font.pixelSize: 18
                font.bold: true
                elide: Text.ElideRight
            }
        }

        MouseArea {
            anchors.fill: parent
            onClicked: tile.chosen()
        }
    }
}
