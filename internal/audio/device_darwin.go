//go:build darwin

package audio

// FindSystemAudioDevice finds the best system audio capture device on macOS.
// It searches for BlackHole virtual audio devices, preferring "BlackHole 2ch".
// Returns nil if no suitable device is found.
func FindSystemAudioDevice(hint string) *DeviceInfo {
	if hint == "" {
		hint = "BlackHole"
	}
	// Try exact match first
	if dev := FindDevice(hint + " 2ch"); dev != nil {
		return dev
	}
	// Try partial match
	return FindDevice(hint)
}

// meetingDeviceHints are substrings found in virtual audio devices created by
// meeting applications. Checked in order; first match wins.
var meetingDeviceHints = []string{
	"Microsoft Teams Audio",
	"Teams Audio",
	"ZoomAudioDevice",
	"Zoom Audio Device",
}

// FindMeetingAudioDevice returns a virtual audio device created by a running
// meeting application, or nil if none is found.
func FindMeetingAudioDevice() *DeviceInfo {
	for _, hint := range meetingDeviceHints {
		if dev := FindDevice(hint); dev != nil {
			return dev
		}
	}
	return nil
}
