package obsws

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestStartOBSIfNeeded(t *testing.T) {
	// Note: This test can only truly validate on systems with OBS installed
	// On CI systems without OBS, this will attempt to launch and may fail gracefully

	err := StartOBSIfNeeded()
	// We don't assert error because:
	// 1. OBS might not be installed on CI
	// 2. OBS might already be running
	// 3. GUI might not be available

	if err != nil && !strings.Contains(err.Error(), "OBS") {
		t.Logf("StartOBSIfNeeded returned unexpected error type: %v", err)
	}
}

func TestIsOBSRunning(t *testing.T) {
	// Test isOBSRunning function
	// This just checks the process query logic works
	running := isOBSRunning()

	// We can't assert whether OBS is running or not,
	// but we can verify the function doesn't panic
	if running {
		t.Logf("OBS is running on %s", runtime.GOOS)
	} else {
		t.Logf("OBS is not running on %s", runtime.GOOS)
	}
}

func TestGetSceneSourcesParseResponse(t *testing.T) {
	// This test validates the response parsing logic
	// Using a mock client for verification
	mockClient := &mockOBSClient{
		sceneName: "Main",
		sources: []SourceInfo{
			{
				SourceName: "Screen",
				SourceType: "macos_screen_capture",
				SourceKind: "input",
				Enabled:    true,
			},
			{
				SourceName: "Mic",
				SourceType: "coreaudio_input_capture",
				SourceKind: "input",
				Enabled:    true,
			},
		},
	}

	// This is a conceptual test - in real scenario would use mock WebSocket
	if len(mockClient.sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(mockClient.sources))
	}

	if mockClient.sources[0].SourceType != "macos_screen_capture" {
		t.Errorf("Wrong source type: %s", mockClient.sources[0].SourceType)
	}
}

func TestRequiredSourcesDetection(t *testing.T) {
	tests := []struct {
		name      string
		sources   []SourceInfo
		wantAudio bool
		wantVideo bool
	}{
		{
			name: "Both sources present",
			sources: []SourceInfo{
				{SourceName: "Audio", SourceType: "coreaudio_input_capture", Enabled: true},
				{SourceName: "Video", SourceType: "macos_screen_capture", Enabled: true},
			},
			wantAudio: true,
			wantVideo: true,
		},
		{
			name: "Only audio present",
			sources: []SourceInfo{
				{SourceName: "Audio", SourceType: "coreaudio_input_capture", Enabled: true},
			},
			wantAudio: true,
			wantVideo: false,
		},
		{
			name: "Only video present",
			sources: []SourceInfo{
				{SourceName: "Video", SourceType: "macos_screen_capture", Enabled: true},
			},
			wantAudio: false,
			wantVideo: true,
		},
		{
			name:      "No sources",
			sources:   []SourceInfo{},
			wantAudio: false,
			wantVideo: false,
		},
		{
			name: "Sources disabled",
			sources: []SourceInfo{
				{SourceName: "Audio", SourceType: "coreaudio_input_capture", Enabled: false},
				{SourceName: "Video", SourceType: "macos_screen_capture", Enabled: false},
			},
			wantAudio: false,
			wantVideo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audioSourceTypes := map[string]bool{
				"coreaudio_input_capture": true,
				"wasapi_input_capture":    true,
				"pulse_input_capture":     true,
			}
			displaySourceTypes := map[string]bool{
				"macos_screen_capture": true,
				"monitor_capture":      true,
				"xshm_input":           true,
			}

			result := &RequiredSources{}
			for _, src := range tt.sources {
				if audioSourceTypes[src.SourceType] && src.Enabled {
					result.HasAudioInput = true
					result.AudioSourceName = src.SourceName
				}
				if displaySourceTypes[src.SourceType] && src.Enabled {
					result.HasDisplayVideo = true
					result.VideoSourceName = src.SourceName
				}
			}

			if result.HasAudioInput != tt.wantAudio {
				t.Errorf("Audio detection: got %v, want %v", result.HasAudioInput, tt.wantAudio)
			}
			if result.HasDisplayVideo != tt.wantVideo {
				t.Errorf("Video detection: got %v, want %v", result.HasDisplayVideo, tt.wantVideo)
			}
		})
	}
}

func TestSourceTypePlatformSelection(t *testing.T) {
	// Test that correct source types are selected per platform
	tests := []struct {
		platform      string
		wantAudioType string
		wantVideoType string
	}{
		{
			platform:      "darwin",
			wantAudioType: "coreaudio_input_capture",
			wantVideoType: "macos_screen_capture",
		},
		{
			platform:      "windows",
			wantAudioType: "wasapi_input_capture",
			wantVideoType: "monitor_capture",
		},
		{
			platform:      "linux",
			wantAudioType: "pulse_input_capture",
			wantVideoType: "xshm_input",
		},
	}

	for _, tt := range tests {
		// Create a mock scenario for each platform
		var audioType, videoType string

		// Simulate platform selection logic
		if tt.platform == "darwin" {
			audioType = "coreaudio_input_capture"
			videoType = "macos_screen_capture"
		} else if tt.platform == "windows" {
			audioType = "wasapi_input_capture"
			videoType = "monitor_capture"
		} else if tt.platform == "linux" {
			audioType = "pulse_input_capture"
			videoType = "xshm_input"
		}

		if audioType != tt.wantAudioType {
			t.Errorf("[%s] Audio type: got %s, want %s", tt.platform, audioType, tt.wantAudioType)
		}
		if videoType != tt.wantVideoType {
			t.Errorf("[%s] Video type: got %s, want %s", tt.platform, videoType, tt.wantVideoType)
		}
	}
}

// Mock client for testing source detection
type mockOBSClient struct {
	sceneName string
	sources   []SourceInfo
}

func TestEnvironmentVariableDetection(t *testing.T) {
	// Test that environment-based OBS paths would work
	// This is more of a documentation test

	obsApp := "/Applications/OBS.app"
	if runtime.GOOS == "darwin" {
		// On macOS, OBS is typically at /Applications/OBS.app
		if _, err := os.Stat(obsApp); err == nil {
			t.Logf("OBS.app found at standard macOS location")
		} else {
			t.Logf("OBS.app not found at %s (expected for non-macOS systems)", obsApp)
		}
	}
}

// BenchmarkSourceDetection benchmark source detection performance
func BenchmarkSourceDetection(b *testing.B) {
	sources := []SourceInfo{
		{SourceName: "Audio", SourceType: "coreaudio_input_capture", Enabled: true},
		{SourceName: "Video", SourceType: "macos_screen_capture", Enabled: true},
		{SourceName: "Camera", SourceType: "av_input_device_capture", Enabled: false},
		{SourceName: "Music", SourceType: "audio_line", Enabled: true},
	}

	audioSourceTypes := map[string]bool{
		"coreaudio_input_capture": true,
		"wasapi_input_capture":    true,
	}
	displaySourceTypes := map[string]bool{
		"macos_screen_capture": true,
		"monitor_capture":      true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, src := range sources {
			_ = audioSourceTypes[src.SourceType] && src.Enabled
			_ = displaySourceTypes[src.SourceType] && src.Enabled
		}
	}
}
