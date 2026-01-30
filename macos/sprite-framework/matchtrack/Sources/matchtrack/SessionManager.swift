import CoreGraphics
import Foundation

final class SessionManager {
    private(set) var sessionDirectory: URL?
    private(set) var statsURL: URL?
    private(set) var logURL: URL?
    private(set) var layoutURL: URL?
    private(set) var lastLayoutURL: URL?

    private var logHandle: FileHandle?
    private let logQueue = DispatchQueue(label: "matchtrack.log-writes")
    private var stats = StatsSnapshot()
    private var config: AppConfig?
    private var timeNormalizer = TimeNormalization()

    private let isoFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    private var sessionStartDateUTC: Date = Date()
    private var sessionStartUptime: TimeInterval = ProcessInfo.processInfo.systemUptime

    func startSession(config: AppConfig) throws {
        shutdown()
        self.config = config
        sessionStartDateUTC = Date()
        sessionStartUptime = ProcessInfo.processInfo.systemUptime
        let hasMarkers = config.buttons.contains { $0.kind == .marker }
        timeNormalizer = TimeNormalization(hasMarkers: hasMarkers)

        let fm = FileManager.default
        let docs = fm.urls(for: .documentDirectory, in: .userDomainMask).first ?? fm.homeDirectoryForCurrentUser
        let dateFolder = Self.dayStamp(from: sessionStartDateUTC)
        let safeName = Self.safeName(config.match.matchName)
        let timeStamp = Self.timeStamp(from: sessionStartDateUTC)
        let sessionDir = docs
            .appendingPathComponent("MatchTracker")
            .appendingPathComponent(dateFolder)
            .appendingPathComponent("\(safeName)_\(timeStamp)")
        try fm.createDirectory(at: sessionDir, withIntermediateDirectories: true)

        sessionDirectory = sessionDir
        statsURL = sessionDir.appendingPathComponent("stats.json")
        logURL = sessionDir.appendingPathComponent("log.jsonl")
        layoutURL = sessionDir.appendingPathComponent("layout.json")
        lastLayoutURL = docs.appendingPathComponent("MatchTracker/last_layout.json")

        let configCopy = sessionDir.appendingPathComponent("config.yaml")
        if let configURL = ConfigSourceTracker.shared.get() {
            _ = try? fm.copyItem(at: configURL, to: configCopy)
        }

        if !fm.fileExists(atPath: logURL!.path) {
            fm.createFile(atPath: logURL!.path, contents: nil)
        }
        logHandle = try FileHandle(forWritingTo: logURL!)
        try logHandle?.seekToEnd()

        saveStats()
    }

    func recordClick(button: ButtonConfig) {
        guard let config else { return }
        let elapsed = elapsedSeconds()
        let matchTime = timeNormalizer.matchTimeEstimateSeconds(elapsedSeconds: elapsed)
        stats.increment(button: button)
        let event = LogEvent(
            utcTimestamp: isoFormatter.string(from: Date()),
            monotonicElapsedSecondsSinceSessionStart: elapsed,
            matchTimeEstimateSeconds: matchTime,
            buttonId: button.id,
            label: button.label,
            group: button.group,
            statKey: button.statKey,
            size: button.size.rawValue,
            playerId: button.playerId,
            kind: button.kind.rawValue
        )
        appendLog(event: event)
        if button.kind == .marker {
            timeNormalizer.recordMarker(button: button, elapsedSeconds: elapsed)
        }
        if config.buttons.count > 0 {
            saveStats()
        }
    }

    func recordMarker(button: ButtonConfig) {
        recordClick(button: button)
    }

    func saveStats() {
        guard let statsURL else { return }
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        do {
            let data = try encoder.encode(stats)
            try Self.atomicWrite(data: data, to: statsURL)
        } catch {
            print("Failed to save stats: \(error)")
        }
    }

    func saveLayout(layout: [String: CGPoint]) {
        guard let layoutURL else { return }
        let payload = LayoutSnapshot(entries: layout.map { LayoutEntry(id: $0.key, x: Double($0.value.x), y: Double($0.value.y)) })
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        do {
            let data = try encoder.encode(payload)
            try Self.atomicWrite(data: data, to: layoutURL)
            if let lastLayoutURL {
                try Self.atomicWrite(data: data, to: lastLayoutURL)
            }
        } catch {
            print("Failed to save layout: \(error)")
        }
    }

    func loadLayout() -> [String: CGPoint] {
        let decoder = JSONDecoder()
        if let layoutURL, let data = try? Data(contentsOf: layoutURL), let payload = try? decoder.decode(LayoutSnapshot.self, from: data) {
            return payload.toPositions()
        }
        if let lastLayoutURL, let data = try? Data(contentsOf: lastLayoutURL), let payload = try? decoder.decode(LayoutSnapshot.self, from: data) {
            return payload.toPositions()
        }
        return [:]
    }

    func shutdown() {
        saveStats()
        logQueue.sync {
            do {
                try logHandle?.synchronize()
            } catch {
                print("Failed to sync log: \(error)")
            }
        }
        try? logHandle?.close()
        logHandle = nil
        sessionDirectory = nil
    }

    func currentElapsedSeconds() -> Double {
        elapsedSeconds()
    }

    func currentMatchTimeSeconds() -> Double {
        timeNormalizer.matchTimeEstimateSeconds(elapsedSeconds: elapsedSeconds())
    }

    func currentStatus() -> MatchStatus {
        timeNormalizer.status(for: elapsedSeconds())
    }

    private func appendLog(event: LogEvent) {
        guard let logHandle else { return }
        logQueue.async {
            do {
                let data = try JSONEncoder().encode(event)
                logHandle.write(data)
                logHandle.write("\n".data(using: .utf8) ?? Data())
            } catch {
                print("Failed to append log: \(error)")
            }
        }
    }

    private func elapsedSeconds() -> Double {
        ProcessInfo.processInfo.systemUptime - sessionStartUptime
    }

    private static func atomicWrite(data: Data, to url: URL) throws {
        let tempURL = url.appendingPathExtension("tmp")
        try data.write(to: tempURL, options: .atomic)
        _ = try FileManager.default.replaceItemAt(url, withItemAt: tempURL)
    }

    private static func dayStamp(from date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter.string(from: date)
    }

    private static func timeStamp(from date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HHmmss'Z'"
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter.string(from: date)
    }

    private static func safeName(_ name: String) -> String {
        let invalid = CharacterSet.alphanumerics.inverted
        return name.components(separatedBy: invalid).filter { !$0.isEmpty }.joined(separator: "_")
    }
}

struct StatsSnapshot: Codable {
    var countsByButtonId: [String: Int] = [:]
    var countsByStatKey: [String: Int] = [:]

    mutating func increment(button: ButtonConfig) {
        countsByButtonId[button.id, default: 0] += 1
        countsByStatKey[button.statKey, default: 0] += 1
    }
}

struct LogEvent: Codable {
    let utcTimestamp: String
    let monotonicElapsedSecondsSinceSessionStart: Double
    let matchTimeEstimateSeconds: Double
    let buttonId: String
    let label: String
    let group: String
    let statKey: String
    let size: String
    let playerId: String?
    let kind: String
}

struct LayoutSnapshot: Codable {
    var entries: [LayoutEntry]

    func toPositions() -> [String: CGPoint] {
        var dict: [String: CGPoint] = [:]
        for entry in entries {
            dict[entry.id] = CGPoint(x: entry.x, y: entry.y)
        }
        return dict
    }
}

struct LayoutEntry: Codable {
    var id: String
    var x: Double
    var y: Double
}
