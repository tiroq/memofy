package obsws

import (
	"encoding/json"
	"fmt"
	"time"
)

// GetRecordStatus queries OBS for current recording status
func (c *Client) GetRecordStatus() (*RecordingState, error) {
	resp, err := c.sendRequest("GetRecordStatus", nil)
	if err != nil {
		return nil, err
	}

	var data struct {
		OutputActive   bool   `json:"outputActive"`
		OutputPaused   bool   `json:"outputPaused"`
		OutputTimecode string `json:"outputTimecode"`
		OutputDuration int    `json:"outputDuration"` // milliseconds
		OutputBytes    int64  `json:"outputBytes"`
	}

	if err := json.Unmarshal(resp.ResponseData, &data); err != nil {
		return nil, err
	}

	// Update cached state
	c.stateMu.Lock()
	c.recordingState.Recording = data.OutputActive
	c.recordingState.Duration = data.OutputDuration / 1000 // Convert to seconds
	c.recordingState.LastUpdated = time.Now()
	state := c.recordingState
	c.stateMu.Unlock()

	return &state, nil
}

// StartRecord initiates recording with specified filename
func (c *Client) StartRecord(filename string) error {
	// Get current record directory from OBS
	resp, err := c.sendRequest("GetRecordDirectory", nil)
	if err != nil {
		return fmt.Errorf("failed to get record directory: %w", err)
	}

	var dirData struct {
		RecordDirectory string `json:"recordDirectory"`
	}
	if err := json.Unmarshal(resp.ResponseData, &dirData); err != nil {
		return fmt.Errorf("failed to parse record directory: %w", err)
	}

	// Start recording
	_, err = c.sendRequest("StartRecord", nil)
	if err != nil {
		return err
	}

	// Update cached state
	c.stateMu.Lock()
	c.recordingState.Recording = true
	c.recordingState.StartTime = time.Now()
	c.recordingState.OutputPath = fmt.Sprintf("%s/%s", dirData.RecordDirectory, filename)
	c.recordingState.LastUpdated = time.Now()
	c.stateMu.Unlock()

	return nil
}

// StopRecord stops the current recording. reason is a machine-readable reason
// code (e.g. "user_stop", "auto_detection_stop") logged via FR-003 in the
// ws_send entry for the StopRecord request.
func (c *Client) StopRecord(reason string) (string, error) {
	// Pass reason as request data so it is merged into the ws_send log entry (FR-003).
	resp, err := c.sendRequest("StopRecord", map[string]interface{}{
		"reason": reason,
	})
	if err != nil {
		return "", err
	}

	var data struct {
		OutputPath string `json:"outputPath"`
	}

	if err := json.Unmarshal(resp.ResponseData, &data); err != nil {
		return "", err
	}

	// Update cached state
	c.stateMu.Lock()
	c.recordingState.Recording = false
	c.recordingState.OutputPath = data.OutputPath
	c.recordingState.Duration = 0
	c.recordingState.LastUpdated = time.Now()
	c.stateMu.Unlock()

	return data.OutputPath, nil
}

// GetRecordingState returns the cached recording state
func (c *Client) GetRecordingState() RecordingState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.recordingState
}

// GetVersion retrieves OBS and WebSocket plugin versions
func (c *Client) GetVersion() (string, string, error) {
	resp, err := c.sendRequest("GetVersion", nil)
	if err != nil {
		return "", "", err
	}

	var data struct {
		OBSVersion          string `json:"obsVersion"`
		OBSWebSocketVersion string `json:"obsWebSocketVersion"`
	}

	if err := json.Unmarshal(resp.ResponseData, &data); err != nil {
		return "", "", err
	}

	return data.OBSVersion, data.OBSWebSocketVersion, nil
}

// SetFilenameFormatting configures OBS recording filename format
func (c *Client) SetFilenameFormatting(format string) error {
	_, err := c.sendRequest("SetProfileParameter", map[string]interface{}{
		"parameterCategory": "Output",
		"parameterName":     "FilenameFormatting",
		"parameterValue":    format,
	})

	return err
}
