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
	// -d aac@<rate> : AAC codec at target sample rate (rate embedded in format string)
	// -f m4af       : M4A container
	// -b            : bitrate in bits/sec
	// -c            : channel count
	dataFormat := "aac"
	if spec.SampleRate > 0 {
		dataFormat = fmt.Sprintf("aac@%d", spec.SampleRate)
	}

	args := []string{
		wavPath,
		m4aPath,
		"-d", dataFormat,
		"-f", "m4af",
		"-b", fmt.Sprintf("%d", spec.BitrateKbps*1000),
		"-c", fmt.Sprintf("%d", spec.Channels),
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
