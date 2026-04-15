# pkg/macui — macOS Menu Bar UI

## OVERVIEW

AppKit-based macOS menu bar app via darwinkit. Provides status icon, dropdown menu, settings window, about dialog, and notifications for the audio recorder. **All AppKit calls must happen on the main OS thread.**

## STRUCTURE

```
macui/
├── statusbar.go           # StatusBarApp: menu, icon, engine status polling
├── settings.go            # SettingsWindow: audio & monitoring config
├── settings_logic.go      # Pure logic: field parsing, config building (no AppKit)
├── settings_logic_test.go # Settings logic unit tests
├── notifications.go       # macOS notification delivery via osascript
├── about.go               # About dialog + update check
└── icon.go                # Status bar icon rendering + tinting
```

## KEY TYPES

| Type | Purpose |
|------|---------|
| `StatusBarApp` | Menu bar icon, menu, polls engine for status |
| `SettingsWindow` | Preferences panel: audio, monitoring, general |
| `SettingsFields` | Form field values (pure Go, testable) |
| `AboutWindow` | About dialog + update checker |

## CONSTRUCTOR

```go
app := macui.NewStatusBarApp(version, engine, cfg)
app.StartUpdateTimer()  // NSTimer on main run loop — polls engine status
```

## GUI THREADING RULES

- `runtime.LockOSThread()` in `main()` — must not be removed
- All AppKit calls must be on the thread that called `app.Run()`
- Use `StartUpdateTimer()` for periodic status polling on main thread
- Engine runs in a background goroutine

## ANTI-PATTERNS

- **No AppKit imports outside this package**
- **No blocking calls on main thread**
- **No IPC — engine is in-process**
