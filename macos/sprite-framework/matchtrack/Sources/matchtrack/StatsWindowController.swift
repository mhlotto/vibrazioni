import AppKit
import SwiftUI

final class StatsWindowController: NSWindowController {
    init(statsStore: StatsStore) {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 380, height: 520),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Match Stats"
        window.center()
        window.isReleasedWhenClosed = false
        super.init(window: window)

        let rootView = StatsView(statsStore: statsStore)
        let hostingController = NSHostingController(rootView: rootView)
        window.contentViewController = hostingController
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
}
