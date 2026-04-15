package macui

import (
	"fmt"
	"log"
	"os/exec"
)

// SendNotification sends a native macOS notification using osascript.
func SendNotification(title, subtitle, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s"`,
		escapeAppleScript(message),
		escapeAppleScript(title),
		escapeAppleScript(subtitle))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Notification error: %v, output: %s", err, output)
		return err
	}
	return nil
}

// SendErrorNotification sends an error notification.
func SendErrorNotification(title, errorMsg string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		escapeAppleScript(errorMsg),
		escapeAppleScript(title))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error notification failed: %v, output: %s", err, output)
		return err
	}
	return nil
}

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
