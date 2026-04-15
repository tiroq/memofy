// Package audio provides cross-platform audio capture using PortAudio.
// On macOS it captures from BlackHole virtual devices; on Linux from
// PulseAudio/PipeWire monitor sources.
package audio

// DeviceInfo describes an audio input device.
type DeviceInfo struct {
	Index      int
	Name       string
	MaxInputCh int
	SampleRate float64
}

// CaptureConfig holds parameters for opening an audio stream.
type CaptureConfig struct {
	DeviceIndex     int
	SampleRate      int
	Channels        int
	FramesPerBuffer int
}

// DefaultCaptureConfig returns sensible defaults for audio capture.
func DefaultCaptureConfig() CaptureConfig {
	return CaptureConfig{
		SampleRate:      44100,
		Channels:        2,
		FramesPerBuffer: 4096,
	}
}
