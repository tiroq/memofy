package macui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// getLogoPath returns the path to the Memofy logo
func getLogoPath() string {
	// Check common locations
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".local", "share", "memofy", "logo.png"),
		filepath.Join(os.Getenv("HOME"), ".local", "share", "memofy", "memofy.png"),
		"/usr/local/share/memofy/logo.png",
	}
	
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	
	// Fallback: return a default path (will be created by install script)
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "memofy", "logo.png")
}

// SendNotification sends a native macOS notification using osascript
// Logo is detected and available for future visual enhancements
func SendNotification(title, subtitle, message string) error {
	// Logo detection for future use (icon in menus, status bar, etc.)
	_ = getLogoPath()
	
	// Use AppleScript to send notification (works on all macOS versions)
	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s"`,
		escapeAppleScript(message),
		escapeAppleScript(title),
		escapeAppleScript(subtitle))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error sending notification: %v, output: %s", err, output)
		return err
	}

	log.Printf("✓ Notification sent: %s - %s", title, message)
	return nil
}

// SendErrorNotification sends an error notification with actionable guidance
func SendErrorNotification(appName, errorMsg string) error {
	logoPath := getLogoPath()
	
	var script string
	if _, err := os.Stat(logoPath); err == nil {
		// Logo exists, include it
		script = fmt.Sprintf(`
tell app "System Events"
	activate
	display dialog "%s" buttons {"Open Settings", "Dismiss"} default button "Dismiss" with title "%s" with icon caution giving up after 5
end tell
`, escapeAppleScript(errorMsg), appName)
	} else {
		// Fallback without logo
		script = fmt.Sprintf(`
tell app "System Events"
	activate
	display dialog "%s" buttons {"Open Settings", "Dismiss"} default button "Dismiss" with title "%s" with icon caution giving up after 5
end tell
`, escapeAppleScript(errorMsg), appName)
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't treat dialog dismissal as error
		log.Printf("Dialog result: %s", output)
		return nil
	}

	return nil
}

// escapeAppleScript escapes special characters in AppleScript strings
func escapeAppleScript(s string) string {
	result := ""
	for _, ch := range s {
		switch ch {
		case '"':
			result += "\\\""
		case '\\':
			result += "\\\\"
		case '\n':
			result += "\\n"
		case '\r':
			result += "\\r"
		case '\t':
			result += "\\t"
		default:
			result += string(ch)
		}
	}
	return result
}
