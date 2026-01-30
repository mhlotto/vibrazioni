import SwiftUI
import Yams

struct NewSessionDialog: View {
    let onCreate: (URL) -> Void
    let onUseSample: () -> Void
    let onCancel: () -> Void

    @State private var matchName = "Friendly_vs_Rivals"
    @State private var teamName = "MyTeam"
    @State private var startTime = Date()
    @State private var includeStartTime = true
    @State private var playersText = ""

    @State private var includeMarkers = true
    @State private var includeTeamGoal = true
    @State private var includeTeamShot = true
    @State private var includeTeamFoul = true
    @State private var includeYellow = false
    @State private var includeRed = false
    @State private var includePlayerPass = true

    @State private var errorMessage: String?
    @State private var playersFocused: Bool = true

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("New Session")
                .font(.system(size: 18, weight: .semibold, design: .rounded))

            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text("Match Name")
                        .frame(width: 110, alignment: .leading)
                    TextField("Match name", text: $matchName)
                        .textFieldStyle(.roundedBorder)
                }
                HStack {
                    Text("Team")
                        .frame(width: 110, alignment: .leading)
                    TextField("Team name", text: $teamName)
                        .textFieldStyle(.roundedBorder)
                }
                Toggle("Include start time", isOn: $includeStartTime)
                DatePicker("Start time", selection: $startTime)
                    .datePickerStyle(.compact)
                    .disabled(!includeStartTime)
            }

            VStack(alignment: .leading, spacing: 8) {
                Text("Players (one per line)")
                MultilineTextView(text: $playersText, isFocused: $playersFocused)
                    .frame(height: 140)
            }

            VStack(alignment: .leading, spacing: 6) {
                Text("Default buttons")
                    .font(.system(size: 13, weight: .semibold, design: .rounded))
                Toggle("Markers (start/end halves)", isOn: $includeMarkers)
                Toggle("Team: Goal", isOn: $includeTeamGoal)
                Toggle("Team: Shot On", isOn: $includeTeamShot)
                Toggle("Team: Foul", isOn: $includeTeamFoul)
                Toggle("Team: Yellow Card", isOn: $includeYellow)
                Toggle("Team: Red Card", isOn: $includeRed)
                Toggle("Per-player Pass + / -", isOn: $includePlayerPass)
            }
            .toggleStyle(.switch)

            if let errorMessage {
                Text(errorMessage)
                    .foregroundStyle(.red)
                    .font(.system(size: 12, weight: .regular, design: .rounded))
            }

            HStack {
                Button("Use Sample Config") {
                    onUseSample()
                }
                Spacer()
                Button("Cancel") {
                    onCancel()
                }
                Button("Create Session") {
                    createSession()
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding(20)
        .frame(width: 540)
        .onAppear {
            DispatchQueue.main.async {
                playersFocused = true
            }
        }
    }

    private func createSession() {
        do {
            let config = buildConfig()
            let yaml = try YAMLEncoder().encode(config)
            let url = try writeConfigFile(yaml: yaml, matchName: config.match.matchName)
            onCreate(url)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func buildConfig() -> AppConfig {
        let match = MatchConfig(
            matchName: matchName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Match" : matchName,
            teamName: teamName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Team" : teamName,
            startTimeUTC: includeStartTime ? iso8601(startTime) : nil
        )

        var buttons: [ButtonConfig] = []
        if includeMarkers {
            buttons.append(contentsOf: [
                ButtonConfig(id: "start_first_half", label: "Start 1st Half", group: "Markers", size: .large, statKey: "startFirstHalf", playerId: nil, kind: .marker),
                ButtonConfig(id: "end_first_half", label: "End 1st Half", group: "Markers", size: .large, statKey: "endFirstHalf", playerId: nil, kind: .marker),
                ButtonConfig(id: "start_second_half", label: "Start 2nd Half", group: "Markers", size: .large, statKey: "startSecondHalf", playerId: nil, kind: .marker),
                ButtonConfig(id: "end_second_half", label: "End 2nd Half", group: "Markers", size: .large, statKey: "endSecondHalf", playerId: nil, kind: .marker)
            ])
        }

        if includeTeamGoal {
            buttons.append(ButtonConfig(id: "goal_for", label: "Goal", group: "Team", size: .large, statKey: "goals", playerId: nil, kind: .stat))
        }
        if includeTeamShot {
            buttons.append(ButtonConfig(id: "shot_on", label: "Shot On", group: "Team", size: .medium, statKey: "shotsOnTarget", playerId: nil, kind: .stat))
        }
        if includeTeamFoul {
            buttons.append(ButtonConfig(id: "foul", label: "Foul", group: "Team", size: .medium, statKey: "fouls", playerId: nil, kind: .stat))
        }
        if includeYellow {
            buttons.append(ButtonConfig(id: "yellow_card", label: "Yellow", group: "Team", size: .medium, statKey: "yellowCard", playerId: nil, kind: .stat))
        }
        if includeRed {
            buttons.append(ButtonConfig(id: "red_card", label: "Red", group: "Team", size: .medium, statKey: "redCard", playerId: nil, kind: .stat))
        }

        if includePlayerPass {
            let players = parsePlayers(playersText)
            for player in players {
                let passPlusId = "\(player.id)_pass_plus"
                let passMinusId = "\(player.id)_pass_minus"
                buttons.append(ButtonConfig(id: passPlusId, label: "\(player.name) Pass+", group: "Players", size: .small, statKey: "passesComplete", playerId: player.id, kind: .stat))
                buttons.append(ButtonConfig(id: passMinusId, label: "\(player.name) Pass-", group: "Players", size: .small, statKey: "passesMiss", playerId: player.id, kind: .stat))
            }
        }

        let ui = UIConfig(
            grid: GridConfig(enabled: true, cellSize: 40, origin: .bottomLeft, margin: 16),
            drag: DragConfig(enabled: true, activation: .immediate, longPressSeconds: 0.25, modifierKey: .option, showGridOverlay: true, snapOnDrop: true, hapticOnSnap: false, preventDragIfClickedWithinSeconds: 0.12)
        )
        return AppConfig(match: match, buttons: buttons, ui: ui)
    }

    private func parsePlayers(_ text: String) -> [PlayerEntry] {
        let lines = text
            .split(whereSeparator: \.isNewline)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        var entries: [PlayerEntry] = []
        var usedIds = Set<String>()
        var counter = 1

        for name in lines {
            var baseId = slugify(name)
            if baseId.isEmpty {
                baseId = "p\(counter)"
                counter += 1
            }
            var candidate = baseId
            var suffix = 2
            while usedIds.contains(candidate) {
                candidate = "\(baseId)_\(suffix)"
                suffix += 1
            }
            usedIds.insert(candidate)
            entries.append(PlayerEntry(id: candidate, name: name))
        }

        return entries
    }

    private func slugify(_ value: String) -> String {
        let allowed = CharacterSet.alphanumerics
        let filtered = value.lowercased().unicodeScalars.map { allowed.contains($0) ? Character($0) : "_" }
        let raw = String(filtered)
        let collapsed = raw.replacingOccurrences(of: "__", with: "_")
        return collapsed.trimmingCharacters(in: CharacterSet(charactersIn: "_"))
    }

    private func iso8601(_ date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.string(from: date)
    }

    private func writeConfigFile(yaml: String, matchName: String) throws -> URL {
        let fm = FileManager.default
        let docs = fm.urls(for: .documentDirectory, in: .userDomainMask).first ?? fm.homeDirectoryForCurrentUser
        let configDir = docs.appendingPathComponent("MatchTracker/Configs")
        try fm.createDirectory(at: configDir, withIntermediateDirectories: true)
        let safeName = matchName.components(separatedBy: CharacterSet.alphanumerics.inverted)
            .filter { !$0.isEmpty }
            .joined(separator: "_")
        let stamp = timeStamp()
        let url = configDir.appendingPathComponent("\(safeName)_\(stamp).yaml")
        try yaml.write(to: url, atomically: true, encoding: .utf8)
        return url
    }

    private func timeStamp() -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyyMMdd_HHmmss'Z'"
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter.string(from: Date())
    }
}

private struct PlayerEntry {
    let id: String
    let name: String
}
