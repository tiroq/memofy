//go:build linux

package audio

import "strings"

// FindSystemAudioDevice finds the best system audio capture device on Linux.
// It searches for PulseAudio/PipeWire monitor sources (e.g. "Monitor of ...").
// Falls back to the default input device if no monitor is found.
func FindSystemAudioDevice(hint string) *DeviceInfo {
	if hint != "" && hint != "default" {
		if dev := FindDevice(hint); dev != nil {
			return dev
		}
	}

	// Look for monitor sources (system audio loopback)
	devices := ListInputDevices()
	for i := range devices {
		name := strings.ToLower(devices[i].Name)
		if strings.Contains(name, "monitor") {
			return &devices[i]
		}
	}

	// Fall back to default input
	dev, _ := DefaultInputDevice()
	return dev
}
