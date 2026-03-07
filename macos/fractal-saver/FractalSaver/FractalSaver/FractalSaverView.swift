import ScreenSaver
import MetalKit

final class FractalSaverView: ScreenSaverView {
    private var metalView: MTKView?
    private var renderer: Renderer?

    override init?(frame: NSRect, isPreview: Bool) {
        super.init(frame: frame, isPreview: isPreview)

        animationTimeInterval = 1.0 / 30.0

        guard let device = MTLCreateSystemDefaultDevice() else {
            return nil
        }

        let mtkView = MTKView(frame: bounds, device: device)
        mtkView.autoresizingMask = [.width, .height]
        mtkView.clearColor = MTLClearColor(red: 0.02, green: 0.02, blue: 0.03, alpha: 1.0)
        mtkView.framebufferOnly = false
        mtkView.isPaused = true
        mtkView.enableSetNeedsDisplay = true
        mtkView.preferredFramesPerSecond = 30

        addSubview(mtkView)

        guard let renderer = Renderer(
            mtkView: mtkView,
            isPreview: isPreview,
            bundle: Bundle(for: FractalSaverView.self)
        ) else {
            NSLog("FractalSaver: renderer initialization failed; saver will not render")
            return nil
        }
        mtkView.delegate = renderer

        self.metalView = mtkView
        self.renderer = renderer
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
    }

    override var hasConfigureSheet: Bool {
        return false
    }

    override func startAnimation() {
        super.startAnimation()
        metalView?.isPaused = true
    }

    override func stopAnimation() {
        metalView?.isPaused = true
        super.stopAnimation()
    }

    override func animateOneFrame() {
        metalView?.draw()
    }
}
