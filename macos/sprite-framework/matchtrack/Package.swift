// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "matchtrack",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "matchtrack", targets: ["matchtrack"])
    ],
    dependencies: [
        .package(url: "https://github.com/jpsim/Yams.git", from: "5.1.3")
    ],
    targets: [
        .executableTarget(
            name: "matchtrack",
            dependencies: [
                .product(name: "Yams", package: "Yams")
            ],
            path: "Sources/matchtrack",
            resources: [
                .process("Resources")
            ]
        )
    ]
)
