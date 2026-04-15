//go:build linux

package audio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertToM4A converts a WAV file to M4A/AAC using ffmpeg.
// Returns the path to the converted file.
func ConvertToM4A(wavPath string, spec FormatSpec) (string, error) {
	m4aPath := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + ".m4a"

	args := []string{
		"-i", wavPath,
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%dk", spec.BitrateKbps),
		"-ac", fmt.Sprintf("%d", spec.Channels),
		"-ar", fmt.Sprintf("%d", spec.SampleRate),
		"-y", // overwrite without asking
		m4aPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w (output: %s)", err, string(output))
	}

	// Remove intermediate WAV
	os.Remove(wavPath)

	return m4aPath, nil
}

// CanConvertToM4A returns true if ffmpeg is available.
func CanConvertToM4A() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}
