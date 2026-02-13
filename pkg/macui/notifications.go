package macui

import (
	"fmt"
	"log"
	"os/exec"
)

// SendNotification sends a native macOS notification using osascript
func SendNotification(title, subtitle, message string) error {
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
	script := fmt.Sprintf(`
tell app "System Events"
	activate
	display dialog "%s" buttons {"Open Settings", "Dismiss"} default button "Dismiss" with title "%s" with icon caution
end tell
`, escapeAppleScript(errorMsg), appName)

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
