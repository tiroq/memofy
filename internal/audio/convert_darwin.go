//go:build darwin

package audio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertToM4A converts a WAV file to M4A/AAC using macOS built-in afconvert.
// Returns the path to the converted file.
func ConvertToM4A(wavPath string, spec FormatSpec) (string, error) {
	m4aPath := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + ".m4a"

	// afconvert is built into macOS — no extra dependencies needed.
	// -d aac   : AAC codec
	// -f m4af  : M4A container
	// -b       : bitrate in bits/sec
	// -c 1     : mono
	// --src-rate: target sample rate
	args := []string{
		wavPath,
		m4aPath,
		"-d", "aac",
		"-f", "m4af",
		"-b", fmt.Sprintf("%d", spec.BitrateKbps*1000),
		"-c", fmt.Sprintf("%d", spec.Channels),
	}
	if spec.SampleRate > 0 {
		args = append(args, "--src-rate", fmt.Sprintf("%d", spec.SampleRate))
	}

	cmd := exec.Command("afconvert", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("afconvert failed: %w (output: %s)", err, string(output))
	}

	// Remove intermediate WAV
	os.Remove(wavPath)

	return m4aPath, nil
}

// CanConvertToM4A returns true if the conversion tool is available.
func CanConvertToM4A() bool {
	_, err := exec.LookPath("afconvert")
	return err == nil
}
