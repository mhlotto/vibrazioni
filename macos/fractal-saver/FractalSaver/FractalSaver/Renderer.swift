import Foundation
import Metal
import MetalKit
import QuartzCore
import simd

struct Vertex {
    var position: SIMD2<Float>
}

struct Uniforms {
    var center: SIMD2<Float>
    var zoom: Float
    var time: Float
    var fractalConstant: SIMD2<Float>
    var resolution: SIMD2<Float>
    var maxIterations: Int32
    // Explicit padding so Swift side is 48 bytes and matches MSL constant-buffer alignment.
    var pad0: Int32 = 0
    var pad1: Int32 = 0
    var pad2: Int32 = 0
}

final class Renderer: NSObject, MTKViewDelegate {
    private struct AnchorRegion {
        var center: SIMD2<Float>
        var preferredZoom: Float
        var radius: Float
    }

    private let device: MTLDevice
    private let commandQueue: MTLCommandQueue
    private let pipelineState: MTLRenderPipelineState
    private let vertexBuffer: MTLBuffer
    private let isPreview: Bool

    private var centerX: Float = -0.7436439
    private var centerY: Float = 0.1318259
    private var zoom: Float = 2.8
    private var elapsedTime: Float = 0.0

    private let startTime: CFTimeInterval
    private var zoomCycleStartTime: CFTimeInterval
    private var rng = DeterministicRNG(seed: 0x4D795F4A756C6961)

    private let zoomStartDefault: Float = 2.8
    private let zoomEnd: Float = 0.0005
    private let zoomCycleDuration: Double = 30.0
    private var cycleZoomStart: Float = 2.8
    private var cycleZoomDecayRate: Double = log(2.8 / 0.0005) / 30.0
    private let fallbackCenter = SIMD2<Float>(-0.7436439, 0.1318259)

    private var cycleAnchorCenter = SIMD2<Float>(-0.7436439, 0.1318259)
    private var currentAnchorIndex: Int = 0
    private var lastAnchorIndex: Int?

    private let anchors: [AnchorRegion] = [
        // Main cardioid / seahorse valley area.
        AnchorRegion(center: SIMD2<Float>(-0.7436439, 0.1318259), preferredZoom: 2.4, radius: 0.045),
        // Elephant valley.
        AnchorRegion(center: SIMD2<Float>(-0.7453, 0.1127), preferredZoom: 2.3, radius: 0.040),
        // Minibrot region.
        AnchorRegion(center: SIMD2<Float>(-1.25066, 0.02012), preferredZoom: 2.8, radius: 0.050),
        // Spiral region.
        AnchorRegion(center: SIMD2<Float>(-0.1011, 0.9563), preferredZoom: 2.6, radius: 0.035),
        // Valley filament region.
        AnchorRegion(center: SIMD2<Float>(-0.39054, -0.58679), preferredZoom: 2.5, radius: 0.040),
        // Antenna detail.
        AnchorRegion(center: SIMD2<Float>(-0.15652, 1.03225), preferredZoom: 2.6, radius: 0.032),
        // Left-side detailed tendrils.
        AnchorRegion(center: SIMD2<Float>(-1.7687788, -0.001739), preferredZoom: 2.8, radius: 0.052),
        // Bulb transition region.
        AnchorRegion(center: SIMD2<Float>(0.2869, 0.0143), preferredZoom: 2.4, radius: 0.030),
    ]

    private var debugFramesRemaining: Int = 6
    private var loggedMissingDrawable = false
    private var loggedMissingPassDescriptor = false

    init?(mtkView: MTKView, isPreview: Bool, bundle: Bundle) {
        let now = CACurrentMediaTime()
        self.startTime = now
        self.zoomCycleStartTime = now

        guard let device = mtkView.device,
              let commandQueue = device.makeCommandQueue(),
              let library = Renderer.loadLibrary(device: device, bundle: bundle),
              let vertexFunction = library.makeFunction(name: "vertex_main"),
              let fragmentFunction = library.makeFunction(name: "fragment_main") else {
            NSLog("FractalSaver: failed to initialize Metal pipeline")
            return nil
        }

        self.device = device
        self.commandQueue = commandQueue
        self.isPreview = isPreview

        let descriptor = MTLRenderPipelineDescriptor()
        descriptor.vertexFunction = vertexFunction
        descriptor.fragmentFunction = fragmentFunction
        descriptor.colorAttachments[0].pixelFormat = mtkView.colorPixelFormat

        do {
            pipelineState = try device.makeRenderPipelineState(descriptor: descriptor)
        } catch {
            NSLog("FractalSaver: failed to create pipeline state: %@", error.localizedDescription)
            return nil
        }

        let vertices: [Vertex] = [
            Vertex(position: SIMD2<Float>(-1.0, -1.0)),
            Vertex(position: SIMD2<Float>( 1.0, -1.0)),
            Vertex(position: SIMD2<Float>(-1.0,  1.0)),
            Vertex(position: SIMD2<Float>( 1.0, -1.0)),
            Vertex(position: SIMD2<Float>( 1.0,  1.0)),
            Vertex(position: SIMD2<Float>(-1.0,  1.0)),
        ]

        guard let vertexBuffer = device.makeBuffer(bytes: vertices,
                                                   length: vertices.count * MemoryLayout<Vertex>.stride,
                                                   options: .storageModeShared) else {
            NSLog("FractalSaver: failed to create vertex buffer")
            return nil
        }

        self.vertexBuffer = vertexBuffer

        super.init()
        beginNewCycle(at: now)

        NSLog(
            "FractalSaver: Uniforms Swift layout size=%d stride=%d align=%d",
            MemoryLayout<Uniforms>.size,
            MemoryLayout<Uniforms>.stride,
            MemoryLayout<Uniforms>.alignment
        )
    }

    func mtkView(_ view: MTKView, drawableSizeWillChange size: CGSize) {
        _ = size
    }

    func draw(in view: MTKView) {
        updateDrawableSizeIfNeeded(in: view)

        guard view.drawableSize.width > 0, view.drawableSize.height > 0 else {
            return
        }

        guard let drawable = view.currentDrawable else {
            if !loggedMissingDrawable {
                loggedMissingDrawable = true
                NSLog("FractalSaver: draw skipped because currentDrawable is nil")
            }
            return
        }
        guard let passDescriptor = view.currentRenderPassDescriptor else {
            if !loggedMissingPassDescriptor {
                loggedMissingPassDescriptor = true
                NSLog("FractalSaver: draw skipped because currentRenderPassDescriptor is nil")
            }
            return
        }
        guard let commandBuffer = commandQueue.makeCommandBuffer(),
              let encoder = commandBuffer.makeRenderCommandEncoder(descriptor: passDescriptor) else {
            return
        }

        let now = CACurrentMediaTime()
        elapsedTime = Float(max(0.0, now - startTime))

        var cycleElapsed = max(0.0, now - zoomCycleStartTime)
        zoom = cycleZoomStart * Float(exp(-cycleZoomDecayRate * cycleElapsed))

        if zoom <= zoomEnd || !zoom.isFinite {
            beginNewCycle(at: now)
            cycleElapsed = 0.0
            zoom = cycleZoomStart
        }

        let safeZoom = max(zoom, 0.00001)
        var maxIterations = Int32(100 + log(1.0 / safeZoom) * 20.0)
        if isPreview {
            maxIterations = max(40, maxIterations / 2)
        }
        maxIterations = min(max(maxIterations, 32), 3000)

        let cyclePhase = Float(cycleElapsed / zoomCycleDuration)
        let theta = 2.0 * Float.pi * cyclePhase
        let orbitOffset = SIMD2<Float>(
            0.006 * cos(theta),
            0.0045 * sin(theta)
        )
        // Keep motion cinematic and calm: weak, clamped zoom influence.
        let zoomNorm = max(0.0, min(1.0, zoom / max(cycleZoomStart, zoomEnd)))
        let orbitScale = max(0.32, min(0.52, 0.32 + 0.20 * sqrt(zoomNorm)))
        let effectiveCenter = clampPoint(cycleAnchorCenter + orbitOffset * orbitScale)

        centerX = effectiveCenter.x
        centerY = effectiveCenter.y

        var uniforms = Uniforms(
            center: effectiveCenter,
            zoom: zoom,
            time: elapsedTime,
            // Unused by Mandelbrot shader path; kept for compatibility with existing uniforms.
            fractalConstant: SIMD2<Float>(0.0, 0.0),
            resolution: SIMD2<Float>(Float(drawable.texture.width), Float(drawable.texture.height)),
            maxIterations: maxIterations
        )

        if debugFramesRemaining > 0 {
            NSLog(
                "FractalSaver: draw zoom=%.6f anchor=(%.5f, %.5f) center=(%.5f, %.5f) anchorIdx=%d maxIter=%d",
                uniforms.zoom,
                cycleAnchorCenter.x,
                cycleAnchorCenter.y,
                uniforms.center.x,
                uniforms.center.y,
                currentAnchorIndex,
                uniforms.maxIterations
            )
            debugFramesRemaining -= 1
        }

        encoder.setRenderPipelineState(pipelineState)
        encoder.setVertexBuffer(vertexBuffer, offset: 0, index: 0)
        encoder.setFragmentBytes(&uniforms, length: MemoryLayout<Uniforms>.stride, index: 1)
        encoder.drawPrimitives(type: .triangle, vertexStart: 0, vertexCount: 6)
        encoder.endEncoding()

        commandBuffer.present(drawable)
        commandBuffer.commit()
    }

    private func beginNewCycle(at now: CFTimeInterval) {
        zoomCycleStartTime = now

        let idx = nextAnchorIndex()
        currentAnchorIndex = idx
        let anchor = anchors[idx]

        cycleZoomStart = max(anchor.preferredZoom, zoomEnd * 1.1)
        cycleZoomDecayRate = log(Double(cycleZoomStart / zoomEnd)) / zoomCycleDuration

        let localTarget = localTargetNearAnchor(anchor)
        var chosenCenter = refineBoundaryNear(localTarget)
        var chosenQuality = quickBoundaryQuality(at: chosenCenter, iterations: 64)
        if chosenQuality < 0.22 {
            let recovered = findBestBoundaryFallback()
            chosenCenter = recovered.point
            chosenQuality = recovered.quality
        }
        if chosenQuality < 0.18 {
            chosenCenter = fallbackCenter
        }

        cycleAnchorCenter = clampPoint(chosenCenter)
        centerX = cycleAnchorCenter.x
        centerY = cycleAnchorCenter.y
    }

    private func nextAnchorIndex() -> Int {
        var idx = randomIndex(count: anchors.count)
        if let last = lastAnchorIndex, anchors.count > 1, idx == last {
            idx = (idx + 1 + randomIndex(count: anchors.count - 1)) % anchors.count
        }
        lastAnchorIndex = idx
        return idx
    }

    private func randomIndex(count: Int) -> Int {
        if count <= 1 { return 0 }
        let t = rng.nextFloat(in: 0.0 ... 0.999999)
        return min(count - 1, Int(t * Float(count)))
    }

    private func localTargetNearAnchor(_ anchor: AnchorRegion) -> SIMD2<Float> {
        let offset = randomDiskOffset(radius: anchor.radius)
        return clampPoint(anchor.center + offset)
    }

    private func refineBoundaryNear(_ seed: SIMD2<Float>) -> SIMD2<Float> {
        let quickIterations = 40
        var bestPoint = clampPoint(seed)
        var bestQuality = boundaryBandQuality(
            iterations: mandelbrotEscapeIterations(c: bestPoint, maxIterations: quickIterations),
            maxIterations: quickIterations
        )

        // Gentle local refinement around the chosen seed.
        for _ in 0..<18 {
            let candidate = clampPoint(seed + randomDiskOffset(radius: 0.06))
            let iters = mandelbrotEscapeIterations(c: candidate, maxIterations: quickIterations)
            let quality = boundaryBandQuality(iterations: iters, maxIterations: quickIterations)
            if quality > bestQuality {
                bestQuality = quality
                bestPoint = candidate
            }
        }

        // Fallback nudge if refinement quality is poor.
        if bestQuality < 0.20 {
            bestPoint = clampPoint(seed + randomDiskOffset(radius: 0.10))
        }

        return bestPoint
    }

    private func boundaryBandQuality(iterations: Int, maxIterations: Int) -> Float {
        if iterations <= 1 {
            return -2.0
        }
        if iterations >= maxIterations {
            // Penalize deep interior points that tend to look flat/dark.
            return -1.5
        }

        let x = Float(iterations) / Float(maxIterations)
        let target: Float = 0.66
        let halfWidth: Float = 0.30
        let band = max(0.0, 1.0 - abs(x - target) / halfWidth)
        return band - 0.08 * abs(x - target)
    }

    private func quickBoundaryQuality(at point: SIMD2<Float>, iterations: Int) -> Float {
        let iters = mandelbrotEscapeIterations(c: point, maxIterations: iterations)
        return boundaryBandQuality(iterations: iters, maxIterations: iterations)
    }

    private func findBestBoundaryFallback() -> (point: SIMD2<Float>, quality: Float) {
        var bestPoint = fallbackCenter
        var bestQuality = quickBoundaryQuality(at: fallbackCenter, iterations: 72)

        for anchor in anchors {
            let centerQuality = quickBoundaryQuality(at: anchor.center, iterations: 72)
            if centerQuality > bestQuality {
                bestQuality = centerQuality
                bestPoint = anchor.center
            }

            for _ in 0..<12 {
                let candidate = clampPoint(anchor.center + randomDiskOffset(radius: anchor.radius * 1.4))
                let quality = quickBoundaryQuality(at: candidate, iterations: 72)
                if quality > bestQuality {
                    bestQuality = quality
                    bestPoint = candidate
                }
            }
        }

        return (bestPoint, bestQuality)
    }

    private func mandelbrotEscapeIterations(c: SIMD2<Float>, maxIterations: Int) -> Int {
        var z = SIMD2<Float>(0.0, 0.0)
        for i in 0..<maxIterations {
            let len2 = z.x * z.x + z.y * z.y
            if len2 > 4.0 || len2 > 256.0 {
                return i
            }
            z = SIMD2<Float>(
                z.x * z.x - z.y * z.y + c.x,
                2.0 * z.x * z.y + c.y
            )
            if !z.x.isFinite || !z.y.isFinite {
                return i
            }
        }
        return maxIterations
    }

    private func randomDiskOffset(radius: Float) -> SIMD2<Float> {
        let angle = rng.nextFloat(in: 0.0 ... (2.0 * .pi))
        let r = sqrt(rng.nextFloat(in: 0.0 ... 1.0)) * radius
        return SIMD2<Float>(cos(angle) * r, sin(angle) * r)
    }

    private func updateDrawableSizeIfNeeded(in view: MTKView) {
        let backingSize = view.convertToBacking(view.bounds).size
        guard backingSize.width > 0, backingSize.height > 0 else {
            return
        }

        let scale: CGFloat = isPreview ? 0.5 : 1.0
        let target = CGSize(
            width: max(1.0, floor(backingSize.width * scale)),
            height: max(1.0, floor(backingSize.height * scale))
        )

        if abs(view.drawableSize.width - target.width) > 0.5 ||
            abs(view.drawableSize.height - target.height) > 0.5 {
            view.drawableSize = target
        }
    }

    private func clampPoint(_ p: SIMD2<Float>) -> SIMD2<Float> {
        SIMD2<Float>(
            clamp(p.x, min: -2.2, max: 1.2),
            clamp(p.y, min: -1.6, max: 1.6)
        )
    }

    private func clamp(_ value: Float, min: Float, max: Float) -> Float {
        Swift.max(min, Swift.min(max, value))
    }

    private static func loadLibrary(device: MTLDevice, bundle: Bundle) -> MTLLibrary? {
        if let library = try? device.makeDefaultLibrary(bundle: bundle) {
            return library
        }
        if let url = bundle.url(forResource: "default", withExtension: "metallib"),
           let library = try? device.makeLibrary(URL: url) {
            return library
        }
        if let library = device.makeDefaultLibrary() {
            return library
        }
        return nil
    }
}
