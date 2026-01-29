import Foundation

struct BallEvent: Identifiable {
    let id = UUID()
    let key: String
    let magnitude: Double
    let kind: Kind
    let timestamp: Date
}

enum Kind: String {
    case network
    case latency
    case cpu
    case memory
    case audio
    case error
}
