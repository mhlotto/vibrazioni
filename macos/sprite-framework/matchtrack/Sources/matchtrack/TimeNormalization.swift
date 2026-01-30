import Foundation

final class TimeNormalization {
    private let hasMarkers: Bool
    private var matchStartElapsed: Double?
    private var endedElapsed: Double?

    init(hasMarkers: Bool = false) {
        self.hasMarkers = hasMarkers
    }

    func recordMarker(button: ButtonConfig, elapsedSeconds: Double) {
        let key = "\(button.label) \(button.statKey)".lowercased()
        if matchStartElapsed == nil {
            if key.contains("start") || key.contains("kick") || key.contains("begin") {
                matchStartElapsed = elapsedSeconds
                return
            }
        }
        if key.contains("end") || key.contains("final") {
            if key.contains("half") || key.contains("period") {
                return
            }
            endedElapsed = elapsedSeconds
            if matchStartElapsed == nil {
                matchStartElapsed = elapsedSeconds
            }
            return
        }
        if matchStartElapsed == nil {
            matchStartElapsed = elapsedSeconds
        }
    }

    func matchTimeEstimateSeconds(elapsedSeconds: Double) -> Double {
        guard let start = matchStartElapsed else {
            return hasMarkers ? 0 : elapsedSeconds
        }
        let end = endedElapsed ?? elapsedSeconds
        return max(0, end - start)
    }

    func status(for elapsedSeconds: Double) -> MatchStatus {
        if let end = endedElapsed, elapsedSeconds >= end {
            return .ended
        }
        guard let start = matchStartElapsed else { return .notStarted }
        if elapsedSeconds >= start {
            return .running
        }
        return .notStarted
    }
}

enum MatchStatus: String {
    case notStarted = "Not Started"
    case running = "Running"
    case ended = "Ended"
}
