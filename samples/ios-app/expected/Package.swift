// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "HabitTracker",
    platforms: [ .iOS("16.0") ],
    targets: [
        .executableTarget(name: "HabitTracker", path: "Sources"),
    ]
)
