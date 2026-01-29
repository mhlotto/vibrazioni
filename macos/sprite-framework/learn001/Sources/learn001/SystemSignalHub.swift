import AVFoundation
import Foundation
import Network

final class SystemSignalHub: ObservableObject, @unchecked Sendable {
    @Published private(set) var recentEvents: [BallEvent] = []
    @Published var networkEnabled: Bool = true { didSet { updateNetworkMonitor() } }
    @Published var latencyEnabled: Bool = true { didSet { updateLatencyProbe() } }
    @Published var cpuEnabled: Bool = true { didSet { updateSampler() } }
    @Published var memoryEnabled: Bool = true { didSet { updateSampler() } }
    @Published var audioEnabled: Bool = true { didSet { updateAudioMonitor() } }

    private let stream: AsyncStream<BallEvent>
    private var continuation: AsyncStream<BallEvent>.Continuation?
    private let monitorQueue = DispatchQueue(label: "learn001.system-signal-hub")

    private var pathMonitor: NWPathMonitor?
    private var latencyTimer: DispatchSourceTimer?
    private var samplerTimer: DispatchSourceTimer?
    private var audioEngine: AVAudioEngine?
    private var lastAudioEmit: UInt64 = 0
    private var lastPathKey: String = ""
    private var started = false

    private let urlSession: URLSession = {
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 5
        config.requestCachePolicy = .reloadIgnoringLocalCacheData
        return URLSession(configuration: config)
    }()

    var events: AsyncStream<BallEvent> { stream }

    init() {
        var localContinuation: AsyncStream<BallEvent>.Continuation?
        stream = AsyncStream { continuation in
            localContinuation = continuation
        }
        continuation = localContinuation
    }

    deinit {
        stop()
    }

    func start() {
        guard !started else { return }
        started = true
        updateNetworkMonitor()
        updateLatencyProbe()
        updateSampler()
        updateAudioMonitor()
    }

    func stop() {
        stopPathMonitor()
        stopLatencyProbe()
        stopSampler()
        stopAudioMonitor()
        started = false
    }

    private func emit(_ event: BallEvent) {
        continuation?.yield(event)
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            recentEvents.insert(event, at: 0)
            if recentEvents.count > 7 {
                recentEvents.removeLast(recentEvents.count - 7)
            }
        }
    }

    private func updateNetworkMonitor() {
        guard started else { return }
        if networkEnabled {
            startPathMonitor()
        } else {
            stopPathMonitor()
        }
    }

    private func startPathMonitor() {
        guard pathMonitor == nil else { return }
        let monitor = NWPathMonitor()
        monitor.pathUpdateHandler = { [weak self] path in
            guard let self else { return }
            let status = Self.pathStatusString(path.status)
            let types = path.availableInterfaces
                .map { Self.interfaceTypeString($0.type) }
                .sorted()
            let typeString = types.isEmpty ? "none" : types.joined(separator: ",")
            let key = "path:\(status):\(typeString)"
            guard key != self.lastPathKey else { return }
            self.lastPathKey = key
            let magnitude = Double(max(1, types.count))
            self.emit(BallEvent(key: key, magnitude: magnitude, kind: .network, timestamp: Date()))
        }
        monitor.start(queue: monitorQueue)
        pathMonitor = monitor
    }

    private func stopPathMonitor() {
        pathMonitor?.cancel()
        pathMonitor = nil
    }

    private func updateLatencyProbe() {
        guard started else { return }
        if latencyEnabled {
            startLatencyProbe()
        } else {
            stopLatencyProbe()
        }
    }

    private func startLatencyProbe() {
        guard latencyTimer == nil else { return }
        let timer = DispatchSource.makeTimerSource(queue: monitorQueue)
        timer.schedule(deadline: .now() + 1, repeating: .seconds(6), leeway: .milliseconds(300))
        timer.setEventHandler { [weak self] in
            self?.runLatencyProbe()
        }
        timer.resume()
        latencyTimer = timer
    }

    private func stopLatencyProbe() {
        latencyTimer?.cancel()
        latencyTimer = nil
    }

    private func runLatencyProbe() {
        guard latencyEnabled else { return }
        guard let url = URL(string: "https://example.com/") else { return }
        var request = URLRequest(url: url)
        request.httpMethod = "HEAD"
        let start = DispatchTime.now()
        let task = urlSession.dataTask(with: request) { [weak self] _, response, error in
            guard let self else { return }
            if error != nil {
                self.emit(BallEvent(key: "latency:example.com", magnitude: 1, kind: .error, timestamp: Date()))
                return
            }
            guard (response as? HTTPURLResponse) != nil else {
                self.emit(BallEvent(key: "latency:example.com", magnitude: 1, kind: .error, timestamp: Date()))
                return
            }
            let elapsed = Double(DispatchTime.now().uptimeNanoseconds - start.uptimeNanoseconds) / 1_000_000.0
            self.emit(BallEvent(key: "latency:example.com", magnitude: elapsed, kind: .latency, timestamp: Date()))
        }
        task.resume()
    }

    private func updateSampler() {
        guard started else { return }
        if cpuEnabled || memoryEnabled {
            startSampler()
        } else {
            stopSampler()
        }
    }

    private func startSampler() {
        guard samplerTimer == nil else { return }
        let timer = DispatchSource.makeTimerSource(queue: monitorQueue)
        timer.schedule(deadline: .now() + 0.5, repeating: .milliseconds(800), leeway: .milliseconds(100))
        timer.setEventHandler { [weak self] in
            self?.sampleCPUAndMemory()
        }
        timer.resume()
        samplerTimer = timer
    }

    private func stopSampler() {
        samplerTimer?.cancel()
        samplerTimer = nil
    }

    private func sampleCPUAndMemory() {
        if cpuEnabled {
            let cpu = Self.cpuUsagePercent()
            emit(BallEvent(key: "cpu", magnitude: cpu, kind: .cpu, timestamp: Date()))
        }

        if memoryEnabled {
            let mem = Self.memoryFootprintMB()
            emit(BallEvent(key: "mem", magnitude: mem, kind: .memory, timestamp: Date()))
        }
    }

    private func updateAudioMonitor() {
        guard started else { return }
        if audioEnabled {
            startAudioMonitor()
        } else {
            stopAudioMonitor()
        }
    }

    private func startAudioMonitor() {
        guard audioEngine == nil else { return }
        let engine = AVAudioEngine()
        let input = engine.inputNode
        let format = input.outputFormat(forBus: 0)
        input.installTap(onBus: 0, bufferSize: 1024, format: format) { [weak self] buffer, _ in
            guard let self else { return }
            let now = DispatchTime.now().uptimeNanoseconds
            if now - self.lastAudioEmit < 40_000_000 {
                return
            }
            self.lastAudioEmit = now
            let rms = Self.rmsLevel(buffer: buffer)
            let magnitude = Self.audioMagnitude(from: rms)
            self.emit(BallEvent(key: "mic", magnitude: magnitude, kind: .audio, timestamp: Date()))
        }
        do {
            try engine.start()
            audioEngine = engine
        } catch {
            emit(BallEvent(key: "mic", magnitude: 1, kind: .error, timestamp: Date()))
        }
    }

    private func stopAudioMonitor() {
        audioEngine?.inputNode.removeTap(onBus: 0)
        audioEngine?.stop()
        audioEngine = nil
    }

    private static func rmsLevel(buffer: AVAudioPCMBuffer) -> Double {
        guard let channelData = buffer.floatChannelData else { return 0 }
        let channelDataValue = channelData[0]
        let frameLength = Int(buffer.frameLength)
        if frameLength == 0 { return 0 }
        var sum: Double = 0
        for i in 0..<frameLength {
            let sample = Double(channelDataValue[i])
            sum += sample * sample
        }
        return sqrt(sum / Double(frameLength))
    }

    private static func audioMagnitude(from rms: Double) -> Double {
        let clamped = max(0.000_000_1, rms)
        let db = 20.0 * log10(clamped)
        let normalized = min(1.0, max(0.0, (db + 60.0) / 60.0))
        return normalized * 100.0
    }

    private static func pathStatusString(_ status: NWPath.Status) -> String {
        switch status {
        case .satisfied:
            return "satisfied"
        case .unsatisfied:
            return "unsatisfied"
        case .requiresConnection:
            return "requires"
        @unknown default:
            return "unknown"
        }
    }

    private static func interfaceTypeString(_ type: NWInterface.InterfaceType) -> String {
        switch type {
        case .wifi:
            return "wifi"
        case .wiredEthernet:
            return "ethernet"
        case .cellular:
            return "cellular"
        case .loopback:
            return "loopback"
        case .other:
            return "other"
        @unknown default:
            return "unknown"
        }
    }

    private static func cpuUsagePercent() -> Double {
        var threadList: thread_act_array_t?
        var threadCount: mach_msg_type_number_t = 0
        let result = task_threads(mach_task_self_, &threadList, &threadCount)
        guard result == KERN_SUCCESS, let threadList else { return 0 }

        var totalUsage: Double = 0
        for i in 0..<Int(threadCount) {
            var info = thread_basic_info()
            var count = mach_msg_type_number_t(THREAD_INFO_MAX)
            let kerr = withUnsafeMutablePointer(to: &info) {
                $0.withMemoryRebound(to: integer_t.self, capacity: Int(count)) {
                    thread_info(threadList[i], thread_flavor_t(THREAD_BASIC_INFO), $0, &count)
                }
            }
            if kerr == KERN_SUCCESS {
                if (info.flags & TH_FLAGS_IDLE) == 0 {
                    totalUsage += Double(info.cpu_usage) / Double(TH_USAGE_SCALE) * 100.0
                }
            }
        }

        let size = vm_size_t(threadCount) * vm_size_t(MemoryLayout<thread_t>.size)
        vm_deallocate(mach_task_self_, vm_address_t(bitPattern: threadList), size)

        return min(100.0, max(0.0, totalUsage))
    }

    private static func memoryFootprintMB() -> Double {
        var info = task_vm_info_data_t()
        var count = mach_msg_type_number_t(MemoryLayout<task_vm_info_data_t>.size / MemoryLayout<integer_t>.size)
        let kerr = withUnsafeMutablePointer(to: &info) {
            $0.withMemoryRebound(to: integer_t.self, capacity: Int(count)) {
                task_info(mach_task_self_, task_flavor_t(TASK_VM_INFO), $0, &count)
            }
        }
        guard kerr == KERN_SUCCESS else { return 0 }
        let bytes = Double(info.phys_footprint)
        return bytes / 1_048_576.0
    }
}
