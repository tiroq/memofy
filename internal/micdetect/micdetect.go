// Package micdetect detects which applications are currently using
// microphone input on macOS 14+ via Core Audio process objects.
// On unsupported platforms or macOS versions, all functions return
// explicit errors.
package micdetect

import "errors"

// ActiveProcess represents a process with audio activity.
type ActiveProcess struct {
	PID           int
	BundleID      string
	RunningInput  bool
	RunningOutput bool
}

var (
	// ErrUnsupportedPlatform is returned on non-macOS systems.
	ErrUnsupportedPlatform = errors.New("mic detection is only supported on macOS")
	// ErrUnsupportedVersion is returned when macOS version is below 14.
	ErrUnsupportedVersion = errors.New("mic detection requires macOS 14+")
	// ErrEnumerationFailed is returned when Core Audio process enumeration fails.
	ErrEnumerationFailed = errors.New("failed to enumerate audio processes")
	// ErrPropertyReadFailed is returned when a Core Audio property read fails.
	ErrPropertyReadFailed = errors.New("failed to read audio process property")
)

// filterActiveInput returns only processes with active microphone input.
func filterActiveInput(procs []ActiveProcess) []ActiveProcess {
	var result []ActiveProcess
	for _, p := range procs {
		if p.RunningInput {
			result = append(result, p)
		}
	}
	return result
}
