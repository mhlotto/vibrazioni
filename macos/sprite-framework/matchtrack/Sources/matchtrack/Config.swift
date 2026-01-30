import AppKit
import CoreGraphics
import Foundation
import Yams

struct AppConfig: Codable {
    var match: MatchConfig
    var buttons: [ButtonConfig]
    var ui: UIConfig

    static func load(from url: URL) throws -> AppConfig {
        let yaml = try String(contentsOf: url, encoding: .utf8)
        var config = try YAMLDecoder().decode(AppConfig.self, from: yaml)
        config.applyDefaultsAndValidate()
        return config
    }

    mutating func applyDefaultsAndValidate() {
        ui.applyDefaults()
        var seen = Set<String>()
        var deduped: [ButtonConfig] = []
        for button in buttons {
            if seen.contains(button.id) {
                print("Duplicate button id ignored: \(button.id)")
                continue
            }
            seen.insert(button.id)
            deduped.append(button)
        }
        buttons = deduped
    }
}

@MainActor
enum ConfigLoader {
    static var lastLoadedURL: URL?

    static func loadDefault() throws -> AppConfig {
        guard let url = Bundle.module.url(forResource: "sample_config", withExtension: "yaml") else {
            throw NSError(domain: "ConfigLoader", code: 1, userInfo: [NSLocalizedDescriptionKey: "Missing sample_config.yaml in bundle."])
        }
        lastLoadedURL = url
        ConfigSourceTracker.shared.set(url)
        return try AppConfig.load(from: url)
    }

    static func load(from url: URL) throws -> AppConfig {
        lastLoadedURL = url
        ConfigSourceTracker.shared.set(url)
        return try AppConfig.load(from: url)
    }
}

final class ConfigSourceTracker: @unchecked Sendable {
    static let shared = ConfigSourceTracker()
    private let queue = DispatchQueue(label: "matchtrack.config-source-tracker")
    private var url: URL?

    func set(_ url: URL?) {
        queue.sync { self.url = url }
    }

    func get() -> URL? {
        queue.sync { url }
    }
}

struct MatchConfig: Codable {
    var matchName: String
    var teamName: String
    var startTimeUTC: String?
}

struct ButtonConfig: Codable, Identifiable {
    var id: String
    var label: String
    var group: String
    var size: ButtonSize
    var statKey: String
    var playerId: String?
    var kind: ButtonKind
}

enum ButtonSize: String, Codable {
    case small
    case medium
    case large

    var dimensions: CGSize {
        switch self {
        case .small:
            return CGSize(width: 100, height: 44)
        case .medium:
            return CGSize(width: 140, height: 54)
        case .large:
            return CGSize(width: 200, height: 70)
        }
    }
}

enum ButtonKind: String, Codable {
    case stat
    case marker
}

struct UIConfig: Codable {
    var grid: GridConfig?
    var drag: DragConfig?

    mutating func applyDefaults() {
        if grid == nil {
            grid = GridConfig()
        } else {
            grid?.applyDefaults()
        }
        if drag == nil {
            drag = DragConfig()
        } else {
            drag?.applyDefaults()
        }
    }
}

struct GridConfig: Codable {
    var enabled: Bool?
    var cellSize: Double?
    var origin: GridOrigin?
    var margin: Double?

    init() {
        enabled = true
        cellSize = 40
        origin = .bottomLeft
        margin = 16
    }

    init(enabled: Bool, cellSize: Double, origin: GridOrigin, margin: Double) {
        self.enabled = enabled
        self.cellSize = cellSize
        self.origin = origin
        self.margin = margin
    }

    mutating func applyDefaults() {
        if enabled == nil { enabled = true }
        if cellSize == nil { cellSize = 40 }
        if origin == nil { origin = .bottomLeft }
        if margin == nil { margin = 16 }
    }
}

enum GridOrigin: String, Codable {
    case topLeft
    case bottomLeft
}

struct DragConfig: Codable {
    var enabled: Bool?
    var activation: DragActivation?
    var longPressSeconds: Double?
    var modifierKey: ModifierKey?
    var showGridOverlay: Bool?
    var snapOnDrop: Bool?
    var hapticOnSnap: Bool?
    var preventDragIfClickedWithinSeconds: Double?

    init() {
        enabled = true
        activation = .immediate
        longPressSeconds = 0.25
        modifierKey = .option
        showGridOverlay = true
        snapOnDrop = true
        hapticOnSnap = false
        preventDragIfClickedWithinSeconds = 0.12
    }

    init(enabled: Bool, activation: DragActivation, longPressSeconds: Double, modifierKey: ModifierKey, showGridOverlay: Bool, snapOnDrop: Bool, hapticOnSnap: Bool, preventDragIfClickedWithinSeconds: Double) {
        self.enabled = enabled
        self.activation = activation
        self.longPressSeconds = longPressSeconds
        self.modifierKey = modifierKey
        self.showGridOverlay = showGridOverlay
        self.snapOnDrop = snapOnDrop
        self.hapticOnSnap = hapticOnSnap
        self.preventDragIfClickedWithinSeconds = preventDragIfClickedWithinSeconds
    }

    mutating func applyDefaults() {
        if enabled == nil { enabled = true }
        if activation == nil { activation = .immediate }
        if longPressSeconds == nil { longPressSeconds = 0.25 }
        if modifierKey == nil { modifierKey = .option }
        if showGridOverlay == nil { showGridOverlay = true }
        if snapOnDrop == nil { snapOnDrop = true }
        if hapticOnSnap == nil { hapticOnSnap = false }
        if preventDragIfClickedWithinSeconds == nil { preventDragIfClickedWithinSeconds = 0.12 }
    }
}

enum DragActivation: String, Codable {
    case immediate
    case longPress
    case modifier
}

enum ModifierKey: String, Codable {
    case option
    case shift
    case command
    case control

    var flag: NSEvent.ModifierFlags {
        switch self {
        case .option:
            return .option
        case .shift:
            return .shift
        case .command:
            return .command
        case .control:
            return .control
        }
    }
}
