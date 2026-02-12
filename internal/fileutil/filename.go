package fileutil

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizeForFilename sanitizes a string for safe use in filenames
func SanitizeForFilename(input string) string {
	if input == "" {
		return "Meeting"
	}

	// Replace illegal filename characters with underscores
	// Illegal chars: / \ : * ? " < > |
	illegalChars := regexp.MustCompile(`[\/\\:*?"<>|]`)
	sanitized := illegalChars.ReplaceAllString(input, "_")

	// Replace multiple spaces/underscores with single hyphen
	whitespace := regexp.MustCompile(`[\s_]+`)
	sanitized = whitespace.ReplaceAllString(sanitized, "-")

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	// Limit length to 50 characters for reasonable filenames
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
		// Remove trailing hyphen if truncation created one
		sanitized = strings.TrimRight(sanitized, "-")
	}

	// Fallback if sanitization resulted in empty string
	if sanitized == "" {
		return "Meeting"
	}

	return sanitized
}

// RenameRecording renames an OBS recording file to the Memofy format
// Format: YYYY-MM-DD_HHMM_Application_Title.mp4
func RenameRecording(obsPath, newBasename string) (string, error) {
	if obsPath == "" {
		return "", nil
	}

	// Check if original file exists
	if _, err := os.Stat(obsPath); os.IsNotExist(err) {
		return obsPath, nil // File doesn't exist, return original path
	}

	// Get directory and extension from OBS path
	dir := filepath.Dir(obsPath)
	ext := filepath.Ext(obsPath)

	// Build new path
	newPath := filepath.Join(dir, newBasename+ext)

	// Avoid renaming if paths are the same
	if obsPath == newPath {
		return obsPath, nil
	}

	// Check if destination already exists
	if _, err := os.Stat(newPath); err == nil {
		// File exists, append a number
		base := strings.TrimSuffix(newBasename, ext)
		for i := 2; i < 100; i++ {
			tryPath := filepath.Join(dir, base+"_"+string(rune('0'+i))+ext)
			if _, err := os.Stat(tryPath); os.IsNotExist(err) {
				newPath = tryPath
				break
			}
		}
	}

	// Rename the file
	if err := os.Rename(obsPath, newPath); err != nil {
		return obsPath, err
	}

	return newPath, nil
}
