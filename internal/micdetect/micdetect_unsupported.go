package micdetect
//go:build !darwin

package micdetect

// IsSupported returns false on non-macOS platforms.
func IsSupported() bool {
	return false
}

// MacOSVersionString returns an empty string on non-macOS platforms.
func MacOSVersionString() string {
	return ""
}

// ActiveMicUsers returns ErrUnsupportedPlatform on non-macOS platforms.
func ActiveMicUsers() ([]ActiveProcess, error) {
	return nil, ErrUnsupportedPlatform
}

// ActiveMicUserBundleIDs returns ErrUnsupportedPlatform on non-macOS platforms.
func ActiveMicUserBundleIDs() ([]string, error) {
	return nil, ErrUnsupportedPlatform
}
