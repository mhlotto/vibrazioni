PVTA Tools Android
==================

This is a basic Android UI for the existing Go PVTA tooling.

Current shape
-------------

- `app/` is a small Jetpack Compose app.
- `go-bridge/` is a tiny Go module that wraps the existing Go logic in
  `pkg/mobilebridge/` for gomobile binding.
- `scripts/build_bridge.sh` builds the Android AAR into
  `app/libs/pvta-mobilebridge.aar`.

What the app exposes
--------------------

- routes
- route status
- vehicles
- stops
- stop status
- departures
- route stops

Bridge build
------------

Prerequisites:

- Go
- gomobile
- gobind
- Android SDK
- Android NDK

Typical setup:

```sh
cd pvta-tools
gomobile init
./android/scripts/build_bridge.sh
```

That should create:

```text
android/app/libs/pvta-mobilebridge.aar
```

Android app build
-----------------

Open `pvta-tools/android/` in Android Studio, or run Gradle from that
directory after the AAR exists.

Notes
-----

- The app uses reflection to load the Go bridge so the Android project can
  exist even before the AAR is generated.
- If the AAR is missing, the app will show a bridge error at runtime.
- The Go bridge methods return JSON strings, and the Android app formats
  those into simple text output similar to the CLI.
