import SwiftUI

struct StatsView: View {
    @ObservedObject var statsStore: StatsStore

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Live Stats")
                .font(.system(size: 18, weight: .semibold, design: .rounded))

            Text("Total clicks: \(totalClicks)")
                .font(.system(size: 12, weight: .regular, design: .monospaced))

            if statsStore.snapshot.countsByStatKey.isEmpty {
                Text("No events yet")
                    .foregroundStyle(.secondary)
            }

            statSection(title: "By Stat Key", items: statsStore.snapshot.countsByStatKey)
            statSection(title: "By Button", items: statsStore.snapshot.countsByButtonId)

            Spacer()
        }
        .padding(16)
        .frame(minWidth: 320, minHeight: 420)
    }

    private var totalClicks: Int {
        statsStore.snapshot.countsByButtonId.values.reduce(0, +)
    }

    @ViewBuilder
    private func statSection(title: String, items: [String: Int]) -> some View {
        if !items.isEmpty {
            Text(title)
                .font(.system(size: 13, weight: .semibold, design: .rounded))

            ScrollView {
                VStack(alignment: .leading, spacing: 6) {
                    ForEach(items.sorted(by: { $0.key < $1.key }), id: \.key) { key, value in
                        HStack {
                            Text(key)
                                .font(.system(size: 12, weight: .regular, design: .monospaced))
                            Spacer()
                            Text("\(value)")
                                .font(.system(size: 12, weight: .semibold, design: .monospaced))
                        }
                    }
                }
            }
            .frame(maxHeight: 200)
        }
    }
}
