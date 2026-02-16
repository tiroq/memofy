# Menu-Click Update Solution

## Problem
macOS AppKit requires all GUI operations to execute on the main thread. The file watcher runs in a background goroutine, so it cannot directly update the menu bar icon or menu without causing crashes (SIGILL/SIGTRAP/SIGABRT).

## Solution
Implement a **queue-based approach** where background threads queue updates and the main thread applies them during user interaction.

### Architecture

```
[Background Thread]              [Main Thread]
watchStatusFile()                
    ↓                           
fsnotify detects change         
    ↓                           
UpdateStatus()                  
    ↓                           
performUpdateOnMainThread()     
    ↓                           
Set pendingUpdate = true        
                                
                                User clicks menu bar icon
                                    ↓
                                rebuildMenu() called
                                    ↓
                                Check HasPendingUpdate()
                                    ↓
                                ApplyPendingUpdate()
                                    ↓
                                updateMenuBarIcon()
                                    ↓
                                rebuildMenu() (again)
```

### Key Components

#### 1. StatusBarApp Fields
```go
type StatusBarApp struct {
    // ... other fields ...
    pendingUpdate bool
    pendingStatus *ipc.StatusSnapshot
}
```

#### 2. Queue Update (Background Thread Safe)
```go
func (app *StatusBarApp) performUpdateOnMainThread(status *ipc.StatusSnapshot) {
    app.pendingUpdate = true
    app.pendingStatus = status
}
```

#### 3. Check for Pending Updates
```go
func (app *StatusBarApp) HasPendingUpdate() bool {
    return app.pendingUpdate
}
```

#### 4. Apply Updates (Main Thread Only)
```go
func (app *StatusBarApp) ApplyPendingUpdate() {
    if !app.pendingUpdate || app.pendingStatus == nil {
        return
    }
    
    status := app.pendingStatus
    app.pendingUpdate = false
    app.pendingStatus = nil
    
    // Safe to call GUI methods here - we're on main thread
    app.updateMenuBarIcon(status)
    app.rebuildMenu()
}
```

#### 5. Automatic Application on Menu Click
```go
func (app *StatusBarApp) rebuildMenu() {
    // Apply pending updates before rebuilding menu
    if app.HasPendingUpdate() {
        log.Printf("[DEBUG] ✓ Menu click detected - applying pending update on main thread")
        app.ApplyPendingUpdate()
        return // ApplyPendingUpdate calls rebuildMenu again
    }
    
    // Normal menu rebuild logic...
}
```

## Benefits

✅ **Thread-Safe**: All GUI operations happen on main thread  
✅ **Simple**: No timers, no complex threading  
✅ **Efficient**: Updates only applied when needed (on user interaction)  
✅ **Crash-Free**: No SIGILL/SIGTRAP/SIGABRT errors  
✅ **macOS Pattern**: Similar to how Dropbox, etc. update menu bar icons  

## Trade-offs

⚠️ **Delayed Updates**: Menu bar icon doesn't update in real-time; only when user clicks it  
- Acceptable because core functionality (OBS control) still works immediately in background  
- User must interact with menu to see status anyway  
- Status changes are rare (mode switches, recording state)  

## Testing

### Manual Test
1. Start the application: `./bin/memofy-ui`
2. Modify status file: `echo 'pause' > ~/.cache/memofy/cmd.txt`
3. Click the menu bar icon
4. Verify icon and menu reflect new status

### Automated Test
```bash
# Watch logs with status changes
./bin/memofy-ui > /tmp/memofy-ui-test.log 2>&1 &
tail -f /tmp/memofy-ui-test.log

# Trigger status changes
echo 'pause' > ~/.cache/memofy/cmd.txt
echo 'auto' > ~/.cache/memofy/cmd.txt

# Look for: "[DEBUG] ✓ Menu click detected - applying pending update on main thread"
```

## Alternative Approaches Tried

### ❌ NSTimer Before Run Loop
**Problem**: Crashed with SIGABRT because run loop wasn't started yet  
**Code**: `timer := foundation.Timer_ScheduledTimerWithTimeInterval(...)`  
**Error**: "SIGABRT: abort" during app.Run()  

### ❌ Direct GUI Calls from Background Thread
**Problem**: Crashed with SIGILL/SIGTRAP  
**Code**: Called `Button.SetTitle()` from watchStatusFile goroutine  
**Error**: "unexpected fault address" or "trace/breakpoint trap"  

### ✅ Menu-Click Triggered Updates (Current Solution)
**Why it works**: macOS calls rebuildMenu() on main thread when user clicks menu bar icon  
**Benefit**: Guarantees all GUI updates happen on correct thread  

## References

- Original crash analysis: [PID_FILE_ISSUES.md](PID_FILE_ISSUES.md)
- macOS threading requirements: [PROCESS_LIFECYCLE.md](../architecture/PROCESS_LIFECYCLE.md)
- Implementation PR: TBD

## Future Improvements

If real-time updates are required, consider:

1. **NSTimer After Run Loop Starts**
   - Schedule timer from a callback that runs after `app.Run()` starts
   - Use `DispatchQueue.main.async` if available in darwinkit

2. **Custom Run Loop Observer**
   - Hook into CFRunLoop to periodically check for updates
   - More complex but provides tighter control

3. **Notification-Based Approach**
   - Post NSNotification from background thread
   - Observe notification on main thread to trigger updates
