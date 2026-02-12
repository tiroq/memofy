package macui

import (
	"fmt"
	"log"

	"github.com/tiroq/memofy/internal/ipc"
)

// StatusBarApp represents the menu bar application
// NOTE: Full macOS menu bar implementation requires platform-specific darwinkit code
// This is a stub that compiles and can monitor status, but doesn't show UI
type StatusBarApp struct {
	currentStatus *ipc.StatusSnapshot
}

// NewStatusBarApp creates and initializes the menu bar application
func NewStatusBarApp() *StatusBarApp {
	log.Println("⚠️  StatusBarApp: Stub implementation")
	log.Println("   Full macOS menu bar UI requires platform-specific darwinkit integration")
	log.Println("   For now, use command-line tools to control the daemon:")
	log.Println("   - echo 'start' > ~/.cache/memofy/cmd.txt")
	log.Println("   - echo 'stop' > ~/.cache/memofy/cmd.txt")
	log.Println("   - echo 'pause' > ~/.cache/memofy/cmd.txt")
	log.Println("   - cat ~/.cache/memofy/status.json")

	return &StatusBarApp{}
}

// UpdateStatus refreshes the UI based on current status
func (app *StatusBarApp) UpdateStatus(status *ipc.StatusSnapshot) {
	app.currentStatus = status

	// Log status changes for visibility
	log.Printf("📊 Status: Mode=%s, Teams=%v, Zoom=%v, OBS=%v, StartStreak=%d, StopStreak=%d",
		status.Mode,
		status.TeamsDetected,
		status.ZoomDetected,
		status.OBSConnected,
		status.StartStreak,
		status.StopStreak)
}

// sendCommand writes a command to the command file
func (app *StatusBarApp) sendCommand(cmd ipc.Command) {
	if err := ipc.WriteCommand(cmd); err != nil {
		fmt.Printf("Error sending command %s: %v\n", cmd, err)
	} else {
		log.Printf("✓ Command sent: %s", cmd)
	}
}
