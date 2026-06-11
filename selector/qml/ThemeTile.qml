import QtQuick

// A single theme preview rendered as a skewed parallelogram: horizontal edges stay
// flat while vertical edges tilt. The focused tile animates more upright and larger.
Item {
    id: tile

    property string themeName: ""
    property string thumbnailPath: ""
    property string bgColor: "#1a1610"
    property string activeBorderColor: "#ffffff"
    property string inactiveBorderColor: "#555555"
    property bool generated: false
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

            Image {
                anchors.fill: parent
                anchors.margins: 4
                source: tile.thumbnailPath ? "file://" + tile.thumbnailPath : ""
                // Generated palette previews stretch to fill the whole card;
                // real images zoom-to-fill (preserve ratio, crop the overflow).
                fillMode: tile.generated ? Image.Stretch : Image.PreserveAspectCrop
                asynchronous: true
                cache: false
            }

            Rectangle {
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.bottom: parent.bottom
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
                anchors.margins: 10
                text: tile.themeName
                color: "#ffffff"
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
