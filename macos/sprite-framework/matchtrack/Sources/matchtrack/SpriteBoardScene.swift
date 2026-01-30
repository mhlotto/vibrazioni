import AppKit
import SpriteKit

final class MatchBoardScene: SKScene {
    private let config: AppConfig
    private let sessionManager: SessionManager
    private let clickHandler: ((ButtonConfig) -> Void)?
    private var buttonNodes: [String: ButtonNode] = [:]
    private var groupHeaders: [SKLabelNode] = []
    private var gridOverlay: SKShapeNode?

    private var activeNode: ButtonNode?
    private var dragOffset: CGPoint = .zero
    private var pressLocation: CGPoint = .zero
    private var pressTime: TimeInterval = 0
    private var isDragging: Bool = false

    private let headerNode = SKLabelNode(fontNamed: "Menlo-Bold")
    private let timeNode = SKLabelNode(fontNamed: "Menlo")
    private let exitButton = ControlButtonNode(label: "Exit", style: .danger)

    private let layoutDebouncer = Debouncer()

    init(config: AppConfig, sessionManager: SessionManager, onClick: ((ButtonConfig) -> Void)? = nil) {
        self.config = config
        self.sessionManager = sessionManager
        self.clickHandler = onClick
        super.init(size: .zero)
        scaleMode = .resizeFill
        backgroundColor = .black
    }

    required init?(coder aDecoder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func didMove(to view: SKView) {
        view.ignoresSiblingOrder = true
        physicsWorld.gravity = .zero
        buildScene()
    }

    override func didChangeSize(_ oldSize: CGSize) {
        if size != oldSize {
            layoutNodes()
        }
    }

    override func update(_ currentTime: TimeInterval) {
        updateHeader()
    }

    private func buildScene() {
        removeAllChildren()
        buttonNodes.removeAll()
        groupHeaders.removeAll()

        headerNode.fontSize = 18
        headerNode.fontColor = .white
        headerNode.horizontalAlignmentMode = .left
        headerNode.verticalAlignmentMode = .center
        addChild(headerNode)

        timeNode.fontSize = 12
        timeNode.fontColor = NSColor.white.withAlphaComponent(0.85)
        timeNode.horizontalAlignmentMode = .left
        timeNode.verticalAlignmentMode = .center
        addChild(timeNode)

        exitButton.name = "control_exit"
        addChild(exitButton)

        createButtons()
        layoutNodes()
    }

    private func createButtons() {
        for button in config.buttons {
            let node = ButtonNode(config: button)
            node.name = "button_\(button.id)"
            addChild(node)
            buttonNodes[button.id] = node
        }
    }

    private func layoutNodes() {
        let headerHeight: CGFloat = 70
        headerNode.position = CGPoint(x: 20, y: size.height - 30)
        timeNode.position = CGPoint(x: 20, y: size.height - 52)
        exitButton.position = CGPoint(x: size.width - 80, y: size.height - 36)

        updateGridOverlay()

        for header in groupHeaders {
            header.removeFromParent()
        }
        groupHeaders.removeAll()

        let layoutPositions = sessionManager.loadLayout()
        let gridConfig = resolvedGrid()
        let cell = CGFloat(gridConfig.cellSize)
        let margin = CGFloat(gridConfig.margin)
        let maxCols = max(1, Int((size.width - margin * 2) / cell))

        let grouped = groupButtons()
        var row = 0
        var col = 0

        for (groupName, buttons) in grouped {
            let header = SKLabelNode(fontNamed: "Menlo-Bold")
            header.fontSize = 12
            header.fontColor = NSColor.white.withAlphaComponent(0.7)
            header.horizontalAlignmentMode = .left
            header.verticalAlignmentMode = .center
            header.text = groupName
            let headerPos = gridPoint(column: 0, row: row, grid: gridConfig, headerHeight: headerHeight)
            header.position = CGPoint(x: headerPos.x, y: headerPos.y + 24)
            addChild(header)
            groupHeaders.append(header)
            row += 1
            col = 0

            for button in buttons {
                guard let node = buttonNodes[button.id] else { continue }
                if let saved = layoutPositions[button.id] {
                    node.position = saved
                } else {
                    let pos = gridPoint(column: col, row: row, grid: gridConfig, headerHeight: headerHeight)
                    node.position = pos
                }
                col += 1
                if col >= maxCols {
                    col = 0
                    row += 1
                }
            }
            row += 1
        }
    }

    private func updateGridOverlay() {
        gridOverlay?.removeFromParent()
        let dragConfig = resolvedDrag()
        let gridConfig = resolvedGrid()
        guard gridConfig.enabled, dragConfig.showGridOverlay else { return }

        let cell = CGFloat(gridConfig.cellSize)
        let margin = CGFloat(gridConfig.margin)
        let path = CGMutablePath()
        let left = margin
        let right = size.width - margin
        let bottom = margin
        let top = size.height - margin

        var x = left
        while x <= right {
            path.move(to: CGPoint(x: x, y: bottom))
            path.addLine(to: CGPoint(x: x, y: top))
            x += cell
        }

        var y = bottom
        while y <= top {
            path.move(to: CGPoint(x: left, y: y))
            path.addLine(to: CGPoint(x: right, y: y))
            y += cell
        }

        let grid = SKShapeNode(path: path)
        grid.strokeColor = NSColor.white.withAlphaComponent(0.08)
        grid.lineWidth = 0.5
        grid.zPosition = -5
        addChild(grid)
        gridOverlay = grid
    }

    private func updateHeader() {
        let elapsed = sessionManager.currentElapsedSeconds()
        let matchTime = sessionManager.currentMatchTimeSeconds()
        let status = sessionManager.currentStatus().rawValue
        headerNode.text = "\(config.match.matchName) - \(status)"
        timeNode.text = String(format: "Elapsed: %.1fs   Match: %.1fs", elapsed, matchTime)
    }

    // MARK: - Input handling

    override func mouseDown(with event: NSEvent) {
        let location = event.location(in: self)
        pressLocation = location
        pressTime = event.timestamp
        isDragging = false

        if exitButton.contains(location) {
            activeNode = nil
            return
        }

        let nodesAtPoint = nodes(at: location)
        activeNode = nodesAtPoint.compactMap { $0 as? ButtonNode }.first
        if let activeNode {
            dragOffset = CGPoint(x: activeNode.position.x - location.x, y: activeNode.position.y - location.y)
        }
    }

    override func mouseDragged(with event: NSEvent) {
        guard let activeNode else { return }
        let dragConfig = resolvedDrag()
        guard dragConfig.enabled else { return }

        let location = event.location(in: self)
        if !isDragging {
            if shouldStartDrag(event: event, location: location) {
                isDragging = true
            } else {
                return
            }
        }

        let newPosition = CGPoint(x: location.x + dragOffset.x, y: location.y + dragOffset.y)
        activeNode.position = newPosition
    }

    override func mouseUp(with event: NSEvent) {
        let location = event.location(in: self)
        if exitButton.contains(location) {
            let outputPath = sessionManager.sessionDirectory?.path
            sessionManager.shutdown()
            if let outputPath {
                print("Session output directory: \(outputPath)")
            }
            NSApplication.shared.terminate(nil)
            return
        }

        guard let activeNode else { return }
        let dragConfig = resolvedDrag()

        if isDragging {
            if dragConfig.snapOnDrop {
                let snapped = snapToGrid(point: activeNode.position)
                activeNode.position = snapped
            }
            saveLayoutDebounced()
        } else {
            let elapsed = event.timestamp - pressTime
            let threshold = dragConfig.preventDragIfClickedWithinSeconds
            if elapsed <= threshold {
                handleClick(node: activeNode)
            } else {
                handleClick(node: activeNode)
            }
        }

        self.activeNode = nil
        isDragging = false
    }

    private func shouldStartDrag(event: NSEvent, location: CGPoint) -> Bool {
        let dragConfig = resolvedDrag()
        let elapsed = event.timestamp - pressTime
        if elapsed < dragConfig.preventDragIfClickedWithinSeconds {
            return false
        }
        switch dragConfig.activation {
        case .immediate:
            return distance(from: pressLocation, to: location) > 2
        case .longPress:
            return elapsed >= dragConfig.longPressSeconds && distance(from: pressLocation, to: location) > 2
        case .modifier:
            let flags = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
            return flags.contains(dragConfig.modifierKey.flag)
        }
    }

    private func handleClick(node: ButtonNode) {
        node.animateTap()
        sessionManager.recordClick(button: node.config)
        clickHandler?(node.config)
    }

    private func saveLayoutDebounced() {
        let layout = buttonNodes.mapValues { $0.position }
        layoutDebouncer.schedule(after: 0.25) { [weak self] in
            self?.sessionManager.saveLayout(layout: layout)
        }
    }

    private func snapToGrid(point: CGPoint) -> CGPoint {
        let grid = resolvedGrid()
        guard grid.enabled else { return point }
        let cell = CGFloat(grid.cellSize)
        let margin = CGFloat(grid.margin)
        let x = (point.x - margin) / cell
        var y: CGFloat
        if grid.origin == .topLeft {
            y = (size.height - margin - point.y) / cell
        } else {
            y = (point.y - margin) / cell
        }
        let snappedX = margin + round(x) * cell
        let snappedY: CGFloat
        if grid.origin == .topLeft {
            snappedY = size.height - margin - round(y) * cell
        } else {
            snappedY = margin + round(y) * cell
        }
        return CGPoint(x: snappedX, y: snappedY)
    }

    private func gridPoint(column: Int, row: Int, grid: GridConfigValue, headerHeight: CGFloat) -> CGPoint {
        let cell = CGFloat(grid.cellSize)
        let margin = CGFloat(grid.margin)
        let x = margin + CGFloat(column) * cell + cell * 0.5
        if grid.origin == .topLeft {
            let y = size.height - headerHeight - margin - CGFloat(row) * cell - cell * 0.5
            return CGPoint(x: x, y: y)
        }
        let y = margin + CGFloat(row) * cell + cell * 0.5
        return CGPoint(x: x, y: y)
    }

    private func groupButtons() -> [(String, [ButtonConfig])] {
        var order: [String] = []
        var grouped: [String: [ButtonConfig]] = [:]
        for button in config.buttons {
            if grouped[button.group] == nil {
                grouped[button.group] = []
                order.append(button.group)
            }
            grouped[button.group]?.append(button)
        }
        return order.map { ($0, grouped[$0] ?? []) }
    }

    private func resolvedGrid() -> GridConfigValue {
        var grid = config.ui.grid ?? GridConfig()
        grid.applyDefaults()
        return GridConfigValue(from: grid)
    }

    private func resolvedDrag() -> DragConfigValue {
        var drag = config.ui.drag ?? DragConfig()
        drag.applyDefaults()
        return DragConfigValue(from: drag)
    }

    private func distance(from a: CGPoint, to b: CGPoint) -> CGFloat {
        let dx = a.x - b.x
        let dy = a.y - b.y
        return sqrt(dx * dx + dy * dy)
    }
}

private struct DragConfigValue {
    let enabled: Bool
    let activation: DragActivation
    let longPressSeconds: TimeInterval
    let modifierKey: ModifierKey
    let showGridOverlay: Bool
    let snapOnDrop: Bool
    let preventDragIfClickedWithinSeconds: TimeInterval

    init(from config: DragConfig) {
        enabled = config.enabled ?? true
        activation = config.activation ?? .longPress
        longPressSeconds = config.longPressSeconds ?? 0.25
        modifierKey = config.modifierKey ?? .option
        showGridOverlay = config.showGridOverlay ?? true
        snapOnDrop = config.snapOnDrop ?? true
        preventDragIfClickedWithinSeconds = config.preventDragIfClickedWithinSeconds ?? 0.12
    }
}

private struct GridConfigValue {
    let enabled: Bool
    let cellSize: Double
    let origin: GridOrigin
    let margin: Double

    init(from config: GridConfig) {
        enabled = config.enabled ?? true
        cellSize = config.cellSize ?? 40
        origin = config.origin ?? .bottomLeft
        margin = config.margin ?? 16
    }
}

private final class Debouncer {
    private var workItem: DispatchWorkItem?
    func schedule(after delay: TimeInterval, _ block: @escaping () -> Void) {
        workItem?.cancel()
        let item = DispatchWorkItem(block: block)
        workItem = item
        DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: item)
    }
}
