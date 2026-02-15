package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidationResult contains the result of an OBS compatibility check
type ValidationResult struct {
	OK       bool
	Message  string
	Issues   []string
	Warnings []string
	Fixes    []string
}

// ValidateOBSVersion checks if OBS version meets minimum requirements
func ValidateOBSVersion(versionString string) *ValidationResult {
	result := &ValidationResult{OK: true}

	// Parse version string like "29.1.3" or "30.0.0-beta1"
	// Extract major.minor.patch
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionString)

	if len(matches) < 4 {
		result.OK = false
		result.Message = fmt.Sprintf("Could not parse OBS version: %s", versionString)
		result.Issues = append(result.Issues, "Invalid version format")
		result.Fixes = append(result.Fixes, "Update OBS to latest version from https://obsproject.com")
		return result
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])

	// Minimum required: OBS 28.0 (has WebSocket v5 support)
	if major < 28 {
		result.OK = false
		result.Issues = append(result.Issues, fmt.Sprintf("OBS version %d.%d is too old (requires 28.0+)", major, minor))
		result.Fixes = append(result.Fixes, "Update OBS to version 28.0 or later from https://obsproject.com")
		result.Message = fmt.Sprintf("OBS %d.%d requires update to 28.0+", major, minor)
		return result
	}

	result.Message = fmt.Sprintf("OBS %d.%d is compatible (requires 28.0+)", major, minor)
	return result
}

// ValidateWebSocketPlugin checks if OBS WebSocket plugin is available
func ValidateWebSocketPlugin() *ValidationResult {
	result := &ValidationResult{OK: true}

	// Note: In actual usage, this should check the OBS plugin registry
	// For now, we'll provide guidance
	result.Message = "WebSocket plugin check requires running OBS connection"
	result.Warnings = append(result.Warnings, "WebSocket plugin validation is best done during actual connection")
	result.Fixes = append(result.Fixes, "Ensure OBS > Tools > obs-websocket Settings shows 'WebSocket Server' is enabled")

	return result
}

// ValidateSceneExists checks if a scene is accessible
func ValidateSceneExists(sceneName string) *ValidationResult {
	result := &ValidationResult{OK: true}

	if sceneName == "" {
		result.OK = false
		result.Issues = append(result.Issues, "No active scene found in OBS")
		result.Fixes = append(result.Fixes, "Create a scene in OBS or ensure a scene is selected as 'Current'")
		result.Message = "No active scene"
		return result
	}

	result.Message = fmt.Sprintf("Active scene '%s' is accessible", sceneName)
	return result
}

// SuggestedFixes returns user-friendly troubleshooting for common errors
func SuggestedFixes(errorCode int, errorMsg string) []string {
	var fixes []string

	switch errorCode {
	case 204:
		// InvalidRequest - likely OBS version or request type mismatch
		fixes = append(fixes, "OBS rejected the request (code 204: InvalidRequest)")
		fixes = append(fixes, "This usually means:")
		fixes = append(fixes, "  1. OBS version is incompatible (need 28.0+)")
		fixes = append(fixes, "  2. obs-websocket plugin is not installed or disabled")
		fixes = append(fixes, "  3. WebSocket server is not enabled in OBS")
		fixes = append(fixes, "")
		fixes = append(fixes, "Steps to fix:")
		fixes = append(fixes, "  1. Check OBS version: About OBS > Version")
		fixes = append(fixes, "  2. Verify WebSocket: OBS > Tools > obs-websocket Settings")
		fixes = append(fixes, "  3. Enable WebSocket server and set port to 4455")
		fixes = append(fixes, "  4. Restart OBS and try again")

	case 203:
		// Timeout
		fixes = append(fixes, "OBS request timed out (code 203)")
		fixes = append(fixes, "")
		fixes = append(fixes, "Possible causes:")
		fixes = append(fixes, "  - OBS is busy or frozen")
		fixes = append(fixes, "  - Network connectivity issue")
		fixes = append(fixes, "  - WebSocket server is not responding")
		fixes = append(fixes, "")
		fixes = append(fixes, "Steps to fix:")
		fixes = append(fixes, "  1. Restart OBS completely")
		fixes = append(fixes, "  2. Check System Preferences > Network > connection")
		fixes = append(fixes, "  3. Verify memofy-core can reach localhost:4455")

	case 500, 600:
		// Generic error or resource not found
		fixes = append(fixes, fmt.Sprintf("OBS error code %d: %s", errorCode, errorMsg))
		fixes = append(fixes, "")
		fixes = append(fixes, "Steps to try:")
		fixes = append(fixes, "  1. Restart OBS")
		fixes = append(fixes, "  2. Check for OBS updates")
		fixes = append(fixes, "  3. Review OBS logs: Help > Logs > View Current Log")

	default:
		if strings.Contains(errorMsg, "not connected") {
			fixes = append(fixes, "Cannot connect to OBS WebSocket")
			fixes = append(fixes, "")
			fixes = append(fixes, "Verify:")
			fixes = append(fixes, "  1. OBS is running")
			fixes = append(fixes, "  2. WebSocket server is enabled in OBS")
			fixes = append(fixes, "  3. Port 4455 is not blocked by firewall")
			fixes = append(fixes, "  4. No other process is using port 4455")
		} else {
			fixes = append(fixes, fmt.Sprintf("Error: %s", errorMsg))
			fixes = append(fixes, "Contact support or check logs for more details")
		}
	}

	return fixes
}

// CheckOBSHealth performs a comprehensive OBS health check
func CheckOBSHealth(obsVersion, wsVersion string) *ValidationResult {
	result := &ValidationResult{OK: true}
	var messages []string

	// Check OBS version
	versionCheck := ValidateOBSVersion(obsVersion)
	if !versionCheck.OK {
		result.OK = false
		result.Issues = append(result.Issues, versionCheck.Issues...)
		result.Fixes = append(result.Fixes, versionCheck.Fixes...)
		messages = append(messages, versionCheck.Message)
	} else {
		messages = append(messages, versionCheck.Message)
	}

	// Check WebSocket version
	wsCheck := validateWebSocketVersion(wsVersion)
	if !wsCheck.OK {
		result.OK = false
		result.Issues = append(result.Issues, wsCheck.Issues...)
		result.Fixes = append(result.Fixes, wsCheck.Fixes...)
		messages = append(messages, wsCheck.Message)
	} else {
		messages = append(messages, wsCheck.Message)
	}

	result.Message = strings.Join(messages, " | ")

	if result.OK {
		result.Message = "OBS health check passed: " + result.Message
	} else {
		result.Message = "OBS health check FAILED: " + result.Message
	}

	return result
}

// validateWebSocketVersion checks if WebSocket version is compatible
func validateWebSocketVersion(wsVersion string) *ValidationResult {
	result := &ValidationResult{OK: true}

	// WebSocket v5 required for OBS 28+
	if !strings.HasPrefix(wsVersion, "5.") {
		result.OK = false
		result.Issues = append(result.Issues, fmt.Sprintf("WebSocket v%s detected (requires 5.x)", wsVersion))
		result.Fixes = append(result.Fixes, "Update obs-websocket plugin to v5.0 or later")
		result.Message = fmt.Sprintf("WebSocket v%s is incompatible", wsVersion)
		return result
	}

	result.Message = fmt.Sprintf("WebSocket v%s is compatible", wsVersion)
	return result
}
