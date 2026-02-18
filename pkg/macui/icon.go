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
func loadMenubarIcon() appkit.Image {
	img := appkit.ImageClass.Alloc().InitWithData(menubarIconPNG)
	img.SetSize(foundation.Size{Width: menubarIconSize, Height: menubarIconSize})
	return img
}

// tintedMenubarIcon returns a copy of the base icon tinted with the given NSColor.
//
// Uses LockFocus/UnlockFocus + NSGraphicsContext compositing — compatible with
// macOS 10.14+. Avoids -[NSImage imageWithTintColor:] which requires macOS 12+.
//
// Algorithm:
//  1. Create a new blank image of the same size.
//  2. lockFocus — redirects subsequent drawing into the new image.
//  3. Draw the original image (CompositeCopy) to copy its pixels + alpha mask.
//  4. Set compositing op to DestinationIn, then fill the entire rect with the
//     tint color — this keeps only existing pixels, recolored with the tint.
//  5. unlockFocus and return.
func tintedMenubarIcon(color appkit.Color) appkit.Image {
	size := foundation.Size{Width: menubarIconSize, Height: menubarIconSize}
	base := loadMenubarIcon()

	tinted := appkit.ImageClass.Alloc().InitWithSize(size)
	objc.Call[objc.Void](tinted, objc.Sel("lockFocus"))

	rect := foundation.Rect{
		Origin: foundation.Point{X: 0, Y: 0},
		Size:   size,
	}

	// Draw source image (copies pixels including alpha).
	base.DrawInRectFromRectOperationFraction(rect, rect, appkit.CompositeCopy, 1.0)

	// Switch the current context to DestinationIn compositing, then fill with
	// the tint color — this recolors all opaque pixels to the chosen color.
	ctx := appkit.GraphicsContext_CurrentContext()
	ctx.SetCompositingOperation(appkit.CompositeDestinationIn)
	color.SetFill()
	appkit.BezierPath_FillRect(rect)

	objc.Call[objc.Void](tinted, objc.Sel("unlockFocus"))
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
