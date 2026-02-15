package obsws

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// SourceInfo represents an OBS source
type SourceInfo struct {
	SourceName string `json:"sourceName"`
	SourceType string `json:"sourceType"`
	SourceKind string `json:"sourceKind"` // "input" or "scene"
	Enabled    bool   `json:"enabled"`
}

// RequiredSources tracks audio and video sources needed for meeting recording
type RequiredSources struct {
	HasAudioInput   bool
	HasDisplayVideo bool
	AudioSourceName string
	VideoSourceName string
}

// GetSceneSources retrieves all sources for a scene
func (c *Client) GetSceneSources(sceneName string) ([]SourceInfo, error) {
	resp, err := c.sendRequest("GetSceneSourceList", map[string]interface{}{
		"sceneName": sceneName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get scene sources: %w", err)
	}

	var data struct {
		Sources []SourceInfo `json:"sources"`
	}

	if err := json.Unmarshal(resp.ResponseData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse scene sources: %w", err)
	}

	return data.Sources, nil
}

// GetActiveScene returns the current active scene name
func (c *Client) GetActiveScene() (string, error) {
	resp, err := c.sendRequest("GetCurrentScene", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get current scene: %w", err)
	}

	var data struct {
		CurrentSceneName string `json:"currentSceneName"`
	}

	if err := json.Unmarshal(resp.ResponseData, &data); err != nil {
		return "", fmt.Errorf("failed to parse current scene: %w", err)
	}

	return data.CurrentSceneName, nil
}

// CreateSource creates a new source in a scene
func (c *Client) CreateSource(sceneName, sourceName, sourceType string, settings interface{}) error {
	_, err := c.sendRequest("CreateInput", map[string]interface{}{
		"sceneName":     sceneName,
		"inputName":     sourceName,
		"inputKind":     sourceType,
		"inputSettings": settings,
	})

	if err != nil {
		return fmt.Errorf("failed to create source %q: %w", sourceName, err)
	}

	return nil
}

// CreateSourceWithRetry creates a source with automatic retries on failure
func (c *Client) CreateSourceWithRetry(sceneName, sourceName, sourceType string, settings interface{}, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("[CREATE_RETRY] Attempt %d/%d to create %q (type: %s)...\n", attempt, maxRetries, sourceName, sourceType)

		err := c.CreateSource(sceneName, sourceName, sourceType, settings)
		if err == nil {
			fmt.Printf("[CREATE_RETRY] Success on attempt %d\n", attempt)
			return nil
		}

		lastErr = err
		fmt.Printf("[CREATE_RETRY] Attempt %d failed: %v\n", attempt, err)

		// Check if this is a code 204 error (OBS version mismatch)
		errStr := fmt.Sprintf("%v", err)
		if strings.Contains(errStr, "204") {
			fmt.Println("[CREATE_RETRY] Code 204 (InvalidRequest) detected - likely OBS version issue")
			fmt.Printf("[CREATE_RETRY] Suggestion: Ensure OBS version 28.0+ and obs-websocket v5+\n")
			return err // Don't retry on version mismatch
		}

		// Wait before retrying (except on last attempt)
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * time.Second
			fmt.Printf("[CREATE_RETRY] Waiting %d seconds before retry %d...\n", waitTime/time.Second, attempt+1)
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("failed to create source %q after %d retries: %w", sourceName, maxRetries, lastErr)
}

// CheckAndCreateAudioSource checks for audio input and creates if missing
func (c *Client) CheckAndCreateAudioSource(sceneName string) (string, error) {
	sources, err := c.GetSceneSources(sceneName)
	if err != nil {
		return "", err
	}

	// Check for existing audio sources
	audioSourceTypes := map[string]bool{
		"coreaudio_input_capture": true, // macOS system audio
		"wasapi_input_capture":    true, // Windows audio
		"pulse_input_capture":     true, // Linux audio
		"av_audio_input":          true, // Generic audio input
		"image_audio_input":       true, // Mic input
	}

	for _, src := range sources {
		if audioSourceTypes[src.SourceType] {
			if src.Enabled {
				fmt.Printf("[SOURCE_FOUND] Audio source '%s' already exists and is enabled\n", src.SourceName)
				return src.SourceName, nil
			}
			// Source exists but is disabled, attempt to enable it
			fmt.Printf("[SOURCE_DISABLED] Audio source '%s' exists but is disabled, attempting to enable...\n", src.SourceName)
			return src.SourceName, nil
		}
	}

	// No audio source found, create one
	// macOS uses coreaudio_input_capture for system audio
	var audioSourceType string
	switch runtime.GOOS {
	case "windows":
		audioSourceType = "wasapi_input_capture"
	case "linux":
		audioSourceType = "pulse_input_capture"
	default: // darwin/macOS
		audioSourceType = "coreaudio_input_capture"
	}

	audioSourceName := "Desktop Audio"
	audioSettings := map[string]interface{}{
		"device": "", // Use default device
	}

	fmt.Printf("[CREATE] Creating audio source '%s' (type: %s)\n", audioSourceName, audioSourceType)
	if err := c.CreateSource(sceneName, audioSourceName, audioSourceType, audioSettings); err != nil {
		fmt.Printf("[ERROR] Failed to create audio source: %v\n", err)
		return "", fmt.Errorf("failed to create audio source: %w", err)
	}
	fmt.Printf("[SUCCESS] Audio source '%s' created\n", audioSourceName)

	return audioSourceName, nil
}

// CheckAndCreateDisplaySource checks for display capture and creates if missing
func (c *Client) CheckAndCreateDisplaySource(sceneName string) (string, error) {
	sources, err := c.GetSceneSources(sceneName)
	if err != nil {
		return "", err
	}

	// Check for existing display/window sources
	displaySourceTypes := map[string]bool{
		"macos_screen_capture": true, // macOS screen
		"monitor_capture":      true, // Windows monitor
		"xshm_input":           true, // Linux screen
		"window_capture":       true, // Window capture
		"game_capture":         true, // Game capture
	}

	for _, src := range sources {
		if displaySourceTypes[src.SourceType] {
			if src.Enabled {
				fmt.Printf("[SOURCE_FOUND] Display source '%s' already exists and is enabled\n", src.SourceName)
				return src.SourceName, nil
			}
			fmt.Printf("[SOURCE_DISABLED] Display source '%s' exists but is disabled, attempting to enable...\n", src.SourceName)
			return src.SourceName, nil
		}
	}

	// No display source found, create one
	var displaySourceType string
	switch runtime.GOOS {
	case "windows":
		displaySourceType = "monitor_capture"
	case "linux":
		displaySourceType = "xshm_input"
	default: // darwin/macOS
		displaySourceType = "macos_screen_capture"
	}

	displaySourceName := "Display Capture"
	displaySettings := map[string]interface{}{
		"display": 0, // Primary display
	}

	fmt.Printf("[CREATE] Creating display source '%s' (type: %s)\n", displaySourceName, displaySourceType)
	if err := c.CreateSource(sceneName, displaySourceName, displaySourceType, displaySettings); err != nil {
		fmt.Printf("[ERROR] Failed to create display source: %v\n", err)
		return "", fmt.Errorf("failed to create display source: %w", err)
	}
	fmt.Printf("[SUCCESS] Display source '%s' created\n", displaySourceName)

	return displaySourceName, nil
}

// ValidateRequiredSources checks if audio and video sources exist
func (c *Client) ValidateRequiredSources(sceneName string) (*RequiredSources, error) {
	sources, err := c.GetSceneSources(sceneName)
	if err != nil {
		return nil, err
	}

	result := &RequiredSources{}

	audioSourceTypes := map[string]bool{
		"coreaudio_input_capture": true,
		"wasapi_input_capture":    true,
		"pulse_input_capture":     true,
		"av_audio_input":          true,
		"image_audio_input":       true,
	}

	displaySourceTypes := map[string]bool{
		"macos_screen_capture": true,
		"monitor_capture":      true,
		"xshm_input":           true,
		"window_capture":       true,
		"game_capture":         true,
	}

	for _, src := range sources {
		if audioSourceTypes[src.SourceType] && src.Enabled {
			result.HasAudioInput = true
			result.AudioSourceName = src.SourceName
		}
		if displaySourceTypes[src.SourceType] && src.Enabled {
			result.HasDisplayVideo = true
			result.VideoSourceName = src.SourceName
		}
	}

	return result, nil
}

// EnsureRequiredSources validates sources and creates missing ones
func (c *Client) EnsureRequiredSources() error {
	// Get current scene
	sceneName, err := c.GetActiveScene()
	if err != nil {
		return fmt.Errorf("failed to get active scene: %w", err)
	}

	// Get scene sources to log current state
	sources, err := c.GetSceneSources(sceneName)
	if err != nil {
		return fmt.Errorf("failed to list scene sources: %w", err)
	}

	fmt.Printf("[SOURCES] Active scene: '%s', existing sources: %d\n", sceneName, len(sources))
	for _, src := range sources {
		fmt.Printf("[SOURCE_FOUND] %s (type: %s, enabled: %v)\n", src.SourceName, src.SourceType, src.Enabled)
	}

	// Check and create audio source if missing
	audioSource, err := c.CheckAndCreateAudioSource(sceneName)
	if err != nil {
		return fmt.Errorf("failed to ensure audio source: %w", err)
	}
	fmt.Printf("[SOURCE_CHECK] Audio source: %s\n", audioSource)

	// Check and create display source if missing
	displaySource, err := c.CheckAndCreateDisplaySource(sceneName)
	if err != nil {
		return fmt.Errorf("failed to ensure display source: %w", err)
	}
	fmt.Printf("[SOURCE_CHECK] Display source: %s\n", displaySource)

	// Validate both sources are now enabled
	validation, err := c.ValidateRequiredSources(sceneName)
	if err != nil {
		return fmt.Errorf("failed to validate sources: %w", err)
	}

	if !validation.HasAudioInput {
		return fmt.Errorf("audio source created but not enabled: %s", validation.AudioSourceName)
	}
	if !validation.HasDisplayVideo {
		return fmt.Errorf("display source created but not enabled: %s", validation.VideoSourceName)
	}

	fmt.Println("[VERIFY] All required sources are present and enabled")
	return nil
}

// GetSourceVariant gets a specific variant of a source type
// For meeting recording, we want screen capture + system audio
func (c *Client) GetMeetingRecordingSetup() (*RequiredSources, error) {
	sceneName, err := c.GetActiveScene()
	if err != nil {
		return nil, err
	}

	return c.ValidateRequiredSources(sceneName)
}

// StartOBSIfNeeded launches OBS if it's not running
func StartOBSIfNeeded() error {
	// Check if OBS is already running
	if isOBSRunning() {
		return nil // Already running
	}

	// Launch OBS
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS
		cmd = exec.Command("open", "-a", "OBS")
	case "windows":
		// Windows - try common installation paths
		cmd = exec.Command("OBS.exe")
	case "linux":
		// Linux
		cmd = exec.Command("obs")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start OBS: %w", err)
	}

	// Wait for OBS to start and WebSocket to be ready
	// Give OBS 5 seconds to initialize
	time.Sleep(5 * time.Second)

	return nil
}

// isOBSRunning checks if OBS process is currently running
func isOBSRunning() bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: check if OBS process exists
		cmd = exec.Command("pgrep", "-f", "OBS")
	case "windows":
		// Windows: check if OBS.exe exists in process list
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq OBS.exe")
	case "linux":
		// Linux: check if obs process exists
		cmd = exec.Command("pgrep", "-f", "obs")
	default:
		return false
	}

	err := cmd.Run()
	return err == nil // Returns nil if process found
}
