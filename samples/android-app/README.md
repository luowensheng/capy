# android-app

**One Capy source → a complete Android app skeleton.**

The Kotlin source, layout XML, Android manifest, gradle build config,
string resources, and a README — all generated from a single 15-line
declaration.

## What you write

```
app "Habit Tracker"
    package "com.example.habits"
    min_sdk 24
    target_sdk 34
    version_code 1
    version_name "0.1.0"

    screen Home    "Today's habits"
    screen History "Past 30 days"

    feature Home    "Daily checklist"
    feature Home    "Streak counter"
    feature Home    "Reset day"
    feature History "Calendar view"
    feature History "Per-habit stats"
end
```

## What you get

```
out/
├── README.md
├── settings.gradle.kts
├── app/
│   ├── build.gradle.kts                         ← Kotlin/Android plugin + deps
│   └── src/main/
│       ├── AndroidManifest.xml                  ← package, label, theme
│       ├── java/MainActivity.kt                 ← entry + one Fragment per screen
│       └── res/
│           ├── layout/activity_main.xml         ← FrameLayout container
│           └── values/strings.xml               ← localized labels
```

7 files. Drop them into Android Studio (File → Open) or run
`./gradlew assembleDebug` from the command line.

## Run

```sh
../../capy run --out-dir out lib.capy script.capy
cd out && ./gradlew assembleDebug
```

## Adding a screen

Add one `screen` line plus zero or more `feature` lines:

```
screen Settings "Settings"
feature Settings "Theme toggle"
feature Settings "Notification time"
```

Regenerate. You get a new `SettingsFragment` class with the feature
list in its body, a new entry in `strings.xml`, and the Fragment is
ready to wire into navigation.

The library is the project scaffold; you only edit the scaffold when
the *shape* of the project needs to change (new dependency, new
manifest entry). Day-to-day editing happens in the implementation
files Capy stubbed out.
