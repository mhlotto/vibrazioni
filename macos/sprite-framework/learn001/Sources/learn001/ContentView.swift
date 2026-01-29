import Foundation
import SpriteKit
import SwiftUI

struct ContentView: View {
    @StateObject private var hub: SystemSignalHub
    private let scene: DemoScene

    private static let timeFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm:ss"
        return formatter
    }()

    init() {
        let hub = SystemSignalHub()
        _hub = StateObject(wrappedValue: hub)
        let scene = DemoScene(eventStream: hub.events)
        scene.scaleMode = .resizeFill
        scene.backgroundColor = .black
        self.scene = scene
    }

    var body: some View {
        SpriteView(scene: scene, options: [.ignoresSiblingOrder])
            .ignoresSafeArea()
            .overlay(alignment: .topLeading) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("System signals -> bouncing balls")
                        .font(.system(size: 13, weight: .semibold, design: .rounded))
                    ForEach(hub.recentEvents) { event in
                        HStack(spacing: 6) {
                            Circle()
                                .fill(Self.eventColor(event))
                                .frame(width: 8, height: 8)
                                .overlay(Circle().stroke(.white.opacity(0.3), lineWidth: 0.5))
                            Text(Self.eventLine(event))
                                .font(.system(size: 11, weight: .regular, design: .monospaced))
                                .foregroundStyle(event.kind == .error ? .red.opacity(0.9) : .white.opacity(0.9))
                        }
                    }
                    Divider().overlay(.white.opacity(0.2))
                    VStack(alignment: .leading, spacing: 4) {
                        Toggle("Network", isOn: $hub.networkEnabled)
                        Toggle("Latency", isOn: $hub.latencyEnabled)
                        Toggle("CPU", isOn: $hub.cpuEnabled)
                        Toggle("Memory", isOn: $hub.memoryEnabled)
                        Toggle("Mic", isOn: $hub.audioEnabled)
                    }
                    .font(.system(size: 11, weight: .regular, design: .rounded))
                    .toggleStyle(.switch)
                }
                .padding(10)
                .background(.black.opacity(0.45))
                .cornerRadius(10)
                .padding()
                .foregroundStyle(.white)
            }
            .onAppear {
                hub.start()
            }
            .onDisappear {
                hub.stop()
            }
    }

    private static func eventLine(_ event: BallEvent) -> String {
        let time = timeFormatter.string(from: event.timestamp)
        let mag = String(format: "%.1f", event.magnitude)
        return "\(time) \(event.kind.rawValue.uppercased()) \(event.key) \(mag)"
    }

    private static func eventColor(_ event: BallEvent) -> Color {
        if event.kind == .error {
            return Color(red: 1.0, green: 0.35, blue: 0.35, opacity: 0.95)
        }
        let hue = stableHue(for: event.key)
        return Color(hue: hue, saturation: 0.75, brightness: 0.95, opacity: 0.95)
    }

    private static func stableHue(for key: String) -> Double {
        var hash: UInt64 = 0xcbf29ce484222325
        for byte in key.utf8 {
            hash ^= UInt64(byte)
            hash &*= 0x100000001b3
        }
        return Double(hash % 360) / 360.0
    }
}

final class DemoScene: SKScene {
    private var lastSize: CGSize = .zero
    private let eventStream: AsyncStream<BallEvent>
    private var eventTask: Task<Void, Never>?

    init(eventStream: AsyncStream<BallEvent>) {
        self.eventStream = eventStream
        super.init(size: .zero)
    }

    required init?(coder aDecoder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func didMove(to view: SKView) {
        view.showsFPS = true
        view.showsNodeCount = true
        physicsWorld.gravity = CGVector(dx: 0, dy: -2.0)

        addStarfield()
        addFloor()
        spawnBouncers(count: 8)
        startEventLoop()
    }

    override func willMove(from view: SKView) {
        eventTask?.cancel()
        eventTask = nil
    }

    override func didChangeSize(_ oldSize: CGSize) {
        if size != lastSize {
            lastSize = size
            childNode(withName: "floor")?.removeFromParent()
            addFloor()
        }
    }

    func handle(_ event: BallEvent) {
        if event.kind == .audio, event.magnitude < 3 {
            return
        }
        spawnBall(for: event)
    }

    private func startEventLoop() {
        eventTask?.cancel()
        eventTask = Task { [weak self] in
            guard let self else { return }
            for await event in eventStream {
                DispatchQueue.main.async { [weak self] in
                    self?.handle(event)
                }
            }
        }
    }

    // MARK: - Scene content

    private func addStarfield() {
        let stars = SKEmitterNode()
        stars.particleTexture = SKTexture(image: NSImage(systemSymbolName: "sparkle", accessibilityDescription: nil) ?? NSImage())
        stars.particleBirthRate = 45
        stars.particleLifetime = 6
        stars.particleLifetimeRange = 2
        stars.particleSpeed = 10
        stars.particleSpeedRange = 20
        stars.particleAlpha = 0.25
        stars.particleAlphaRange = 0.25
        stars.particleScale = 0.12
        stars.particleScaleRange = 0.10
        stars.particlePositionRange = CGVector(dx: size.width, dy: size.height)
        stars.position = CGPoint(x: size.width / 2, y: size.height / 2)
        stars.zPosition = -10
        addChild(stars)
    }

    private func addFloor() {
        let floor = SKNode()
        floor.name = "floor"
        floor.position = .zero

        let body = SKPhysicsBody(edgeFrom: CGPoint(x: 0, y: 40), to: CGPoint(x: size.width, y: 40))
        body.friction = 0.4
        body.restitution = 0.6
        floor.physicsBody = body
        addChild(floor)

        let line = SKShapeNode(rectOf: CGSize(width: size.width, height: 2))
        line.position = CGPoint(x: size.width / 2, y: 40)
        line.alpha = 0.25
        addChild(line)
    }

    private func spawnBouncers(count: Int) {
        for _ in 0..<count {
            spawnBall(for: BallEvent(
                key: "seed",
                magnitude: Double.random(in: 12...55),
                kind: .network,
                timestamp: Date()
            ))
        }
    }

    private func spawnBall(for event: BallEvent) {
        let radius = ballRadius(for: event.magnitude)
        let node = SKShapeNode(circleOfRadius: radius)
        let color = ballColor(for: event)
        node.position = spawnPoint(for: event.kind)
        node.lineWidth = 0
        node.fillColor = color
        node.glowWidth = event.kind == .error ? 4.0 : 2.0
        node.zPosition = 1

        let body = SKPhysicsBody(circleOfRadius: radius)
        body.mass = 0.2
        body.friction = 0.15
        body.restitution = event.kind == .error ? 0.95 : 0.85
        body.linearDamping = 0.05
        body.angularDamping = 0.05
        node.physicsBody = body

        let impulse = impulseVector(for: event)
        body.applyImpulse(impulse)
        body.applyAngularImpulse(CGFloat.random(in: -1.2...1.2))

        addChild(node)

        let pulse = SKAction.sequence([
            .scale(to: 1.08, duration: 0.35),
            .scale(to: 1.00, duration: 0.35)
        ])
        node.run(.repeatForever(pulse))

        let trail = makeTrailEmitter(color: color)
        trail.targetNode = self
        trail.position = .zero
        node.addChild(trail)

        if event.kind == .error {
            burst(at: node.position, color: color, strong: true)
        } else if event.kind == .latency {
            burst(at: node.position, color: color, strong: false)
        }
    }

    private func spawnPoint(for kind: Kind) -> CGPoint {
        let x = CGFloat.random(in: 40...(max(41, size.width - 40)))
        let yBase = size.height - 60
        if kind == .audio {
            return CGPoint(x: x, y: 80)
        }
        return CGPoint(x: x, y: max(120, yBase))
    }

    private func ballRadius(for magnitude: Double) -> CGFloat {
        let raw = sqrt(max(0, magnitude)) * 3.0
        return clamp(CGFloat(raw), min: 8, max: 42)
    }

    private func ballColor(for event: BallEvent) -> SKColor {
        if event.kind == .error {
            return SKColor(red: 1.0, green: 0.35, blue: 0.35, alpha: 0.95)
        }
        let hue = stableHue(for: event.key)
        return SKColor(hue: hue, saturation: 0.75, brightness: 0.95, alpha: 0.95)
    }

    private func stableHue(for key: String) -> CGFloat {
        var hash: UInt64 = 0xcbf29ce484222325
        for byte in key.utf8 {
            hash ^= UInt64(byte)
            hash &*= 0x100000001b3
        }
        return CGFloat(hash % 360) / 360.0
    }

    private func impulseVector(for event: BallEvent) -> CGVector {
        let magnitude = max(0, event.magnitude)
        let energy = CGFloat(min(2.2, max(0.6, log1p(magnitude) / 4.0 + 0.6)))

        switch event.kind {
        case .network:
            return CGVector(dx: CGFloat.random(in: -35...35), dy: CGFloat.random(in: 20...55))
        case .latency:
            return CGVector(dx: CGFloat.random(in: -25...25), dy: CGFloat.random(in: 30...70) * energy)
        case .cpu:
            return CGVector(dx: CGFloat.random(in: -18...18), dy: CGFloat.random(in: 20...50))
        case .memory:
            return CGVector(dx: CGFloat.random(in: -12...12), dy: CGFloat.random(in: 15...40))
        case .audio:
            return CGVector(dx: CGFloat.random(in: -60...60), dy: CGFloat.random(in: 5...25))
        case .error:
            return CGVector(dx: CGFloat.random(in: -90...90), dy: CGFloat.random(in: 90...150))
        }
    }

    private func makeTrailEmitter(color: SKColor) -> SKEmitterNode {
        let e = SKEmitterNode()
        e.particleTexture = SKTexture(image: NSImage(systemSymbolName: "circle.fill", accessibilityDescription: nil) ?? NSImage())
        e.particleBirthRate = 80
        e.particleLifetime = 0.5
        e.particleLifetimeRange = 0.3
        e.particleSpeed = 0
        e.particleAlpha = 0.20
        e.particleAlphaRange = 0.15
        e.particleScale = 0.10
        e.particleScaleRange = 0.08
        e.particlePositionRange = CGVector(dx: 6, dy: 6)
        e.emissionAngleRange = .pi * 2
        e.particleColor = color
        e.particleColorBlendFactor = 1.0
        return e
    }

    private func burst(at point: CGPoint, color: SKColor, strong: Bool) {
        let burst = SKEmitterNode()
        burst.particleTexture = SKTexture(image: NSImage(systemSymbolName: "sparkle", accessibilityDescription: nil) ?? NSImage())
        burst.particleBirthRate = 0
        burst.numParticlesToEmit = strong ? 120 : 50
        burst.particleLifetime = strong ? 1.0 : 0.6
        burst.particleLifetimeRange = 0.3
        burst.particleSpeed = strong ? 200 : 120
        burst.particleSpeedRange = strong ? 120 : 70
        burst.particleAlpha = 0.8
        burst.particleAlphaRange = 0.2
        burst.particleScale = strong ? 0.24 : 0.16
        burst.particleScaleRange = 0.12
        burst.emissionAngleRange = .pi * 2
        burst.particleColor = color
        burst.particleColorBlendFactor = 1.0
        burst.position = point
        addChild(burst)
        burst.run(.sequence([.wait(forDuration: strong ? 1.6 : 1.0), .removeFromParent()]))
    }

    // MARK: - Interaction

    override func mouseDown(with event: NSEvent) {
        let p = event.location(in: self)
        for _ in 0..<6 {
            let manual = BallEvent(key: "click", magnitude: Double.random(in: 12...80), kind: .network, timestamp: Date())
            spawnBall(for: manual)
        }
        burst(at: p, color: SKColor.white, strong: false)
    }

    private func clamp(_ value: CGFloat, min: CGFloat, max: CGFloat) -> CGFloat {
        Swift.max(min, Swift.min(max, value))
    }
}
