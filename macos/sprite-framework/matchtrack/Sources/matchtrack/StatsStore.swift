import Foundation

@MainActor
final class StatsStore: ObservableObject {
    @Published private(set) var snapshot = StatsSnapshot()

    func reset() {
        snapshot = StatsSnapshot()
    }

    func update(snapshot: StatsSnapshot) {
        self.snapshot = snapshot
    }
}
