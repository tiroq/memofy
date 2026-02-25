# pkg/macui — macOS Status Bar UI

## OVERVIEW

AppKit-based macOS menu bar app via darwinkit. Owns all GUI: status icon, dropdown menu, settings window, notifications. **All AppKit calls must happen on the main OS thread.**

## STRUCTURE

```
macui/
├── statusbar.go           # StatusBarApp: menu, icon, tray interactions
├── settings.go            # SettingsWindow: preferences UI + OBS config
├── settings_logic_test.go # Settings logic tests (non-GUI)
├── notifications.go       # macOS notification delivery
├── about.go               # About panel
└── icon.go                # Status bar icon rendering
```

## KEY TYPES

| Type | Purpose |
|------|---------|
| `StatusBarApp` | Top-level: owns NSStatusItem, menu, update timer |
| `SettingsWindow` | Preferences panel: OBS URL/password, detection rules |
| `SettingsFields` | Form field values for settings |

## CONSTRUCTOR + UPDATE CYCLE

```go
app := macui.NewStatusBarApp(version)
app.UpdateStatus(snapshot)     // Call from main thread only
app.StartUpdateTimer()         // NSTimer on main run loop — flushes pending updates
```

`UpdateStatus()` is safe to call from background goroutines — it enqueues; timer flushes on main thread.

## GUI THREADING RULES (CRITICAL)

- `runtime.LockOSThread()` in `main()` — **must not be removed**
- All `appkit.*` / `darwinkit.*` calls must be on the thread that called `app.Run()`
- Use `app.StartUpdateTimer()` + enqueue pattern — never call AppKit from goroutines directly

## ANTI-PATTERNS

- **No AppKit imports outside this package and `cmd/memofy-ui/`**
- **No blocking calls on main thread** — status updates are enqueued, not applied inline
- **No direct IPC reads here** — UI reads `ipc.ReadStatus()`; macui only consumes `*ipc.StatusSnapshot`
