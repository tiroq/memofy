//go:build darwin

package audio

import "strings"

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

// meetingBundleToDevice maps known meeting app bundle-ID substrings to the
// virtual audio device name they create. Only when an active mic user matches
// one of these entries is the corresponding device preferred over BlackHole.
var meetingBundleToDevice = []struct {
	bundleSubstr string
	deviceHints  []string
}{
	{"com.microsoft.teams", []string{"Microsoft Teams Audio", "Teams Audio"}},
	{"us.zoom.xos", []string{"ZoomAudioDevice", "Zoom Audio Device"}},
	{"zoom", []string{"ZoomAudioDevice", "Zoom Audio Device"}},
}

// FindMeetingAudioDeviceForBundles returns the virtual audio device created by
// the meeting application that is currently using the microphone.
// bundleIDs is the list of bundle IDs actively using mic input (from monitor).
// Returns nil when none of the active bundles correspond to a known virtual device,
// so the caller falls back to the default capture device (e.g. BlackHole).
func FindMeetingAudioDeviceForBundles(bundleIDs []string) *DeviceInfo {
	for _, entry := range meetingBundleToDevice {
		for _, bid := range bundleIDs {
			if strings.Contains(strings.ToLower(bid), entry.bundleSubstr) {
				for _, hint := range entry.deviceHints {
					if dev := FindDevice(hint); dev != nil {
						return dev
					}
				}
			}
		}
	}
	return nil
}
