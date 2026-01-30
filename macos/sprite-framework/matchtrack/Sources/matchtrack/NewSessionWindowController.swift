import AppKit
import SwiftUI

final class NewSessionWindowController: NSWindowController, NSWindowDelegate {
    private var onClose: (() -> Void)?

    init(onCreate: @escaping (URL) -> Void, onUseSample: @escaping () -> Void, onClose: @escaping () -> Void) {
        self.onClose = onClose
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 560, height: 520),
            styleMask: [.titled, .closable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        window.title = "New Session"
        window.center()
        window.isReleasedWhenClosed = false
        super.init(window: window)
        window.delegate = self

        let rootView = NewSessionDialog(
            onCreate: { [weak self] url in
                onCreate(url)
                self?.close()
            },
            onUseSample: { [weak self] in
                onUseSample()
                self?.close()
            },
            onCancel: { [weak self] in
                self?.close()
            }
        )
        let hostingController = NSHostingController(rootView: rootView)
        window.contentViewController = hostingController
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func windowWillClose(_ notification: Notification) {
        onClose?()
    }

    override func showWindow(_ sender: Any?) {
        super.showWindow(sender)
        if let window = window {
            window.makeKeyAndOrderFront(sender)
        }
        NSApplication.shared.activate(ignoringOtherApps: true)
    }
}
