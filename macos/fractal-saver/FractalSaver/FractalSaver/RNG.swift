import Foundation

// Small deterministic RNG for repeatable saver behavior.
struct DeterministicRNG {
    private var state: UInt64

    init(seed: UInt64) {
        state = (seed == 0) ? 0x9E3779B97F4A7C15 : seed
    }

    mutating func nextUInt64() -> UInt64 {
        var x = state
        x ^= x >> 12
        x ^= x << 25
        x ^= x >> 27
        state = x
        return x &* 2685821657736338717
    }

    mutating func nextUnitFloat() -> Float {
        let value = nextUInt64() >> 40 // top 24 bits
        return Float(value) / Float(1 << 24)
    }

    mutating func nextFloat(in range: ClosedRange<Float>) -> Float {
        let t = nextUnitFloat()
        return range.lowerBound + (range.upperBound - range.lowerBound) * t
    }
}
