# Threading Fix for macOS UI Crashes

## Problem
The menu bar UI was crashing with `SIGILL: illegal instruction` and `SIGTRAP: trace trap` errors when the status file changed. 

### Root Cause
The `watchStatusFile()` function runs in a background goroutine (via fsnotify). When status changes were detected, it called `UpdateStatus()` which modified AppKit GUI elements (`Menu.AddItem`, `Button.SetTitle`, etc.). 

**macOS AppKit requires all GUI operations to run on the main thread only.**  Calling GUI methods from background threads causes immediate crashes.

### Crash Location
```
pkg/macui/statusbar.go:220  - rebuildMenu() called from watchStatusFile goroutine
  → Menu.AddItem() - AppKit GUI operation from background thread
  → SIGILL/SIGTRAP crash
```

## Temporary Fix (Current State)
**Disabled all GUI updates from the background thread** in `performUpdateOnMainThread()`:
- Commented out `updateMenuBarIcon()` 
- Commented out `rebuildMenu()`  
- Status changes are logged but UI is not updated dynamically

The app now:
- ✅ Does NOT crash
- ✅ Shows correct initial state at startup
- ❌ Does NOT update menu bar icon when status changes
- ❌ Does NOT rebuild menu when recording state changes

## Proper Solution (TODO)
Implement main-thread dispatch for GUI updates:

### Option 1: Timer-based polling (Simple)
Add a timer in main.go that runs on the main thread and periodically updates the UI:
```go
ticker := time.NewTicker(500 * time.Millisecond)
go func() {
    for range ticker.C {
        if statusBarApp.HasPendingUpdate() {
            statusBarApp.ApplyPendingUpdate() // runs on main thread
        }
    }
}()
```

### Option 2: Channel-based (Better)
Use a channel to queue UI updates and process them on the main thread

### Option 3: GCD Main Queue Dispatch (macOS native, best)
Use `dispatch_async` to the main queue (requires CGO or darwinkit support):
```objc
dispatch_async(dispatch_get_main_queue(), ^{
    // GUI update code here
});
```

## Files Modified
- `pkg/macui/statusbar.go` - Disabled GUI updates from background thread
- Added warnings in comments about threading requirements

## Testing
Before fix:
- Crash within seconds of startup
- SIGILL/SIGTRAP in Menu.AddItem

After fix:
- Runs stably (tested 5+ seconds with multiple status changes)
- No crashes
- Status logging works correctly
