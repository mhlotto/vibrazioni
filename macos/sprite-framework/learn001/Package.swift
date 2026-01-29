// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "learn001",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "learn001", targets: ["learn001"])
    ],
    targets: [
        .executableTarget(
            name: "learn001",
            path: "Sources/learn001"
        )
    ]
)
