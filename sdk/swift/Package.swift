// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "FProtoSDK",
    platforms: [.iOS(.v15), .macOS(.v12)],
    products: [
        .library(name: "FProtoSDK", targets: ["FProtoSDK"]),
    ],
    dependencies: [
        .package(url: "https://github.com/daltoniam/Starscream.git", from: "4.0.0"),
    ],
    targets: [
        .target(
            name: "FProtoSDK",
            dependencies: ["Starscream"],
            path: "Sources/FProtoSDK"
        ),
        .testTarget(
            name: "FProtoSDKTests",
            dependencies: ["FProtoSDK"],
            path: "Tests/FProtoSDKTests"
        ),
    ]
)
