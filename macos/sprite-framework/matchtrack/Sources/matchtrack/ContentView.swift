import AppKit
import Foundation
import SpriteKit
import SwiftUI
import UniformTypeIdentifiers

// Outputs land in ~/Documents/MatchTracker/<YYYY-MM-DD>/<matchName>_<HHmmssZ>/ (stats.json, log.jsonl, layout.json).
// Default YAML config is bundled at Sources/matchtrack/Resources/sample_config.yaml.
struct ContentView: View {
    struct ClickEntry: Identifiable {
        let id = UUID()
        let text: String
    }

    @State private var scene: MatchBoardScene?
    @State private var config: AppConfig?
    @State private var errorMessage: String?
    @State private var showImporter = false
    @State private var recentClicks: [ClickEntry] = []
    @State private var newSessionWindow: NewSessionWindowController?
    @State private var statsWindow: StatsWindowController?
    @StateObject private var statsStore: StatsStore

    private let sessionManager: SessionManager

    init() {
        let store = StatsStore()
        _statsStore = StateObject(wrappedValue: store)
        sessionManager = SessionManager(statsStore: store)
    }

    var body: some View {
        ZStack(alignment: .topLeading) {
            if let scene {
                SpriteView(scene: scene, options: [.ignoresSiblingOrder])
                    .ignoresSafeArea()
            } else {
                Color.black.ignoresSafeArea()
            }
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 12) {
                    Button("Load Config") {
                        showImporter = true
                    }
                    .keyboardShortcut("o", modifiers: [.command])

                    Button("New Session") {
                        presentNewSessionWindow()
                    }

                    Button("Exit") {
                        exitApp()
                    }
                }
                .buttonStyle(.borderedProminent)
                .tint(.gray)

                if let config {
                    Text("Config: \(config.match.matchName) / \(config.match.teamName)")
                        .font(.system(size: 12, weight: .regular, design: .rounded))
                        .foregroundStyle(.white.opacity(0.8))
                }
            }
            .padding(12)
            .background(.black.opacity(0.4))
            .cornerRadius(10)
            .padding()
            recentClicksView()
        }
        .onAppear {
            presentNewSessionWindow()
            NSApplication.shared.activate(ignoringOtherApps: true)
            presentStatsWindow()
        }
        .fileImporter(isPresented: $showImporter, allowedContentTypes: [.yaml]) { result in
            switch result {
            case .success(let url):
                loadConfig(from: url)
            case .failure(let error):
                errorMessage = error.localizedDescription
            }
        }
        .alert("MatchTrack Error", isPresented: Binding(get: { errorMessage != nil }, set: { _ in errorMessage = nil })) {
            Button("OK") { errorMessage = nil }
        } message: {
            Text(errorMessage ?? "Unknown error")
        }
    }

    private func loadDefaultConfig() {
        do {
            let config = try ConfigLoader.loadDefault()
            try sessionManager.startSession(config: config)
            self.config = config
            scene = makeScene(config: config)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadConfig(from url: URL) {
        do {
            let config = try ConfigLoader.load(from: url)
            try sessionManager.startSession(config: config)
            self.config = config
            scene = makeScene(config: config)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func reloadSession() {
        guard let config else {
            loadDefaultConfig()
            return
        }
        do {
            try sessionManager.startSession(config: config)
            scene = makeScene(config: config)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func makeScene(config: AppConfig) -> MatchBoardScene {
        let scene = MatchBoardScene(config: config, sessionManager: sessionManager) { button in
            let label = "\(button.label) (\(button.group))"
            let entry = ClickEntry(text: label)
            recentClicks.append(entry)
            if recentClicks.count > 5 {
                recentClicks.removeFirst(recentClicks.count - 5)
            }
        }
        return scene
    }

    @ViewBuilder
    private func recentClicksView() -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Recent clicks")
                .font(.system(size: 12, weight: .semibold, design: .rounded))
                .foregroundStyle(.white.opacity(0.9))
            ScrollViewReader { proxy in
                ScrollView {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(recentClicks) { entry in
                            Text(entry.text)
                                .font(.system(size: 11, weight: .regular, design: .monospaced))
                                .foregroundStyle(.white.opacity(0.85))
                                .id(entry.id)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .frame(width: 220, height: 110)
                .onChange(of: recentClicks.count) { _ in
                    if let last = recentClicks.last {
                        withAnimation(.easeOut(duration: 0.2)) {
                            proxy.scrollTo(last.id, anchor: .bottom)
                        }
                    }
                }
            }
        }
        .padding(10)
        .background(.black.opacity(0.45))
        .cornerRadius(10)
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topTrailing)
        .allowsHitTesting(false)
    }

    private func exitApp() {
        let outputPath = sessionManager.sessionDirectory?.path
        sessionManager.shutdown()
        if let outputPath {
            print("Session output directory: \(outputPath)")
        }
        NSApplication.shared.terminate(nil)
    }

    private func presentNewSessionWindow() {
        if let window = newSessionWindow?.window {
            window.makeKeyAndOrderFront(nil)
            NSApplication.shared.activate(ignoringOtherApps: true)
            return
        }
        let controller = NewSessionWindowController(onCreate: { url in
            loadConfig(from: url)
        }, onUseSample: {
            loadDefaultConfig()
        }, onClose: {
            DispatchQueue.main.async {
                self.newSessionWindow = nil
            }
        })
        newSessionWindow = controller
        controller.showWindow(nil)
        NSApplication.shared.activate(ignoringOtherApps: true)
    }

    private func presentStatsWindow() {
        if let window = statsWindow?.window {
            window.orderFront(nil)
            return
        }
        let controller = StatsWindowController(statsStore: statsStore)
        statsWindow = controller
        controller.showWindow(nil)
    }
}

extension UTType {
    static var yaml: UTType {
        UTType(filenameExtension: "yaml") ?? .plainText
    }
}
