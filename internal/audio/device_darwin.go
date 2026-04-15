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
