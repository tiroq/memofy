package macui

import (
	_ "embed"

	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
	"github.com/tiroq/memofy/internal/ipc"
)

//go:embed assets/menubar-icon.png
var menubarIconPNG []byte

// menubarIconSize is the logical size for the menu bar icon (18pt = 36px @2x)
const menubarIconSize = 18.0

// loadMenubarIcon loads the embedded PNG and returns an NSImage sized for the menu bar.
// The image is NOT marked as a template so we can apply explicit tint colors.
func loadMenubarIcon() appkit.Image {
	img := appkit.ImageClass.Alloc().InitWithData(menubarIconPNG)
	img.SetSize(foundation.Size{Width: menubarIconSize, Height: menubarIconSize})
	return img
}

// tintedMenubarIcon returns a copy of the base icon tinted with the given NSColor.
// Uses -[NSImage imageWithTintColor:] (macOS 10.14+).
func tintedMenubarIcon(color appkit.Color) appkit.Image {
	base := loadMenubarIcon()
	// Mark as template so imageWithTintColor works correctly
	objc.Call[objc.Void](base, objc.Sel("setTemplate:"), true)
	tinted := objc.Call[appkit.Image](base, objc.Sel("imageWithTintColor:"), color)
	return tinted
}

// iconForStatus returns the correct tinted logo image for the given status.
//
// Color scheme:
//   - Error       → systemRed
//   - Recording   → systemRed (bright, active)
//   - Paused      → systemOrange
//   - Meeting det.→ systemYellow
//   - OBS conn.   → systemGreen
//   - Idle        → systemGray
func iconForStatus(status *ipc.StatusSnapshot) appkit.Image {
	if status == nil {
		return tintedMenubarIcon(appkit.Color_SystemGrayColor())
	}

	if status.LastError != "" {
		return tintedMenubarIcon(appkit.Color_SystemRedColor())
	}

	// Check if actively recording
	if isActivelyRecording(status) {
		return tintedMenubarIcon(appkit.Color_SystemRedColor())
	}

	if status.OBSConnected {
		switch status.Mode {
		case ipc.ModePaused:
			return tintedMenubarIcon(appkit.Color_SystemOrangeColor())
		case ipc.ModeManual:
			return tintedMenubarIcon(appkit.Color_SystemBlueColor())
		}
		return tintedMenubarIcon(appkit.Color_SystemGreenColor())
	}

	if status.TeamsDetected || status.ZoomDetected || status.GoogleMeetActive {
		return tintedMenubarIcon(appkit.Color_SystemYellowColor())
	}

	return tintedMenubarIcon(appkit.Color_SystemGrayColor())
}

// isActivelyRecording extracts the recording boolean from the status snapshot.
func isActivelyRecording(status *ipc.StatusSnapshot) bool {
	if status == nil {
		return false
	}
	if recordingState, ok := status.RecordingState.(map[string]interface{}); ok {
		if recording, exists := recordingState["recording"]; exists {
			if b, ok := recording.(bool); ok {
				return b
			}
		}
	}
	return false
}
