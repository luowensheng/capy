# ios-app

**One Capy source → a complete iOS SwiftUI app skeleton.**

Same DSL shape as `android-app/` (deliberately). Demonstrates that
ONE source can target multiple platforms by swapping libraries.

## What you write

```
app "Habit Tracker"
    bundle_id "com.example.habits"
    version "0.1.0"
    build 1
    deployment_target "16.0"

    screen Home    "Today's habits"
    screen History "Past 30 days"

    feature Home    "Daily checklist"
    feature History "Calendar view"
end
```

## What you get

```
out/
├── README.md
├── Info.plist                ← bundle id, version, launch screen
├── Package.swift             ← SPM manifest
└── Sources/
    ├── App.swift             ← @main App entry
    ├── RootView.swift        ← TabView wiring per declared screen
    └── Screens.swift         ← One SwiftUI View per screen
```

Generated `App.swift`:

```swift
@main
struct HabitTrackerApp: App {
    var body: some Scene {
        WindowGroup { RootView() }
    }
}
```

Note `HabitTracker` (PascalCase) is derived from `"Habit Tracker"`
automatically via the new `pascalCase` template helper.

## Run

```sh
../../capy run --out-dir out lib.capy script.capy
cd out && swift build      # or open the directory in Xcode
```

## Multi-platform with one source

The Android sample (`samples/android-app/`) accepts the same source
shape — `app/screen/feature` — and emits Kotlin instead of Swift.
Run both and you get parallel project skeletons for both stores:

```sh
capy run --out-dir android-out  samples/android-app/lib.capy  script.capy
capy run --out-dir ios-out      samples/ios-app/lib.capy      script.capy
```

That's two platforms, one source of truth, zero drift.
