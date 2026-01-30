import AppKit
import SpriteKit

final class ButtonNode: SKShapeNode {
    let config: ButtonConfig
    private let labelNode: SKLabelNode

    init(config: ButtonConfig) {
        self.config = config
        labelNode = SKLabelNode(fontNamed: "Menlo-Bold")
        super.init()

        let size = config.size.dimensions
        let rect = CGRect(origin: CGPoint(x: -size.width / 2, y: -size.height / 2), size: size)
        path = CGPath(roundedRect: rect, cornerWidth: 12, cornerHeight: 12, transform: nil)

        fillColor = ButtonNode.groupColor(for: config.group)
        strokeColor = NSColor.white.withAlphaComponent(0.2)
        lineWidth = 1
        zPosition = 2

        labelNode.text = config.label
        labelNode.fontSize = config.size == .small ? 11 : (config.size == .medium ? 13 : 15)
        labelNode.fontColor = NSColor.white
        labelNode.verticalAlignmentMode = .center
        labelNode.horizontalAlignmentMode = .center
        addChild(labelNode)
    }

    required init?(coder aDecoder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func animateTap() {
        let down = SKAction.scale(to: 0.96, duration: 0.06)
        let up = SKAction.scale(to: 1.0, duration: 0.08)
        run(.sequence([down, up]))
        let flash = SKAction.sequence([
            .colorize(with: .white, colorBlendFactor: 0.25, duration: 0.05),
            .colorize(withColorBlendFactor: 0.0, duration: 0.1)
        ])
        run(flash)
    }

    private static func groupColor(for group: String) -> NSColor {
        var hash: UInt64 = 0xcbf29ce484222325
        for byte in group.utf8 {
            hash ^= UInt64(byte)
            hash &*= 0x100000001b3
        }
        let hue = CGFloat(hash % 360) / 360.0
        return NSColor(calibratedHue: hue, saturation: 0.55, brightness: 0.85, alpha: 0.9)
    }
}

enum ControlButtonStyle {
    case danger
}

final class ControlButtonNode: SKShapeNode {
    private let labelNode: SKLabelNode

    init(label: String, style: ControlButtonStyle) {
        labelNode = SKLabelNode(fontNamed: "Menlo-Bold")
        super.init()

        let size = CGSize(width: 90, height: 34)
        let rect = CGRect(origin: CGPoint(x: -size.width / 2, y: -size.height / 2), size: size)
        path = CGPath(roundedRect: rect, cornerWidth: 8, cornerHeight: 8, transform: nil)
        fillColor = style == .danger ? NSColor.systemRed.withAlphaComponent(0.9) : NSColor.darkGray
        strokeColor = NSColor.white.withAlphaComponent(0.2)
        lineWidth = 1
        zPosition = 5

        labelNode.text = label
        labelNode.fontSize = 13
        labelNode.fontColor = NSColor.white
        labelNode.verticalAlignmentMode = .center
        labelNode.horizontalAlignmentMode = .center
        addChild(labelNode)
    }

    required init?(coder aDecoder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
}
