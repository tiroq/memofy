package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/detector"
	"github.com/tiroq/memofy/internal/fileutil"
	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/internal/obsws"
	"github.com/tiroq/memofy/internal/statemachine"
)

const (
	obsWebSocketURL = "ws://localhost:4455"
	obsPassword     = "" // Default: no password
	logPrefix       = "[memofy-core]"
)

var (
	// Version is set at build time via -ldflags "-X main.Version=..."
	Version = "dev"

	outLog *log.Logger
	errLog *log.Logger

	// Meeting context for file renaming
	currentMeetingTitle string
	currentMeetingStart time.Time
	currentMeetingApp   detector.DetectedApp

	// Logging counters
	noMeetingLogCounter int
)

func main() {
	// Initialize logging
	if err := initLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}

	outLog.Println("Starting Memofy Core v" + Version + "...")

	// Check macOS permissions
	if err := checkPermissions(); err != nil {
		errLog.Printf("Permission check failed: %v", err)
		errLog.Println("Please grant Screen Recording and Accessibility permissions in System Preferences > Security & Privacy")
		os.Exit(1)
	}

	// Load detection configuration
	cfg, err := config.LoadDetectionRules()
	if err != nil {
		errLog.Printf("Failed to load detection config: %v", err)
		os.Exit(1)
	}
	outLog.Printf("Loaded detection config: %d rules, poll_interval=%ds, thresholds=%d/%d",
		len(cfg.Rules), cfg.PollInterval, cfg.StartThreshold, cfg.StopThreshold)

	// Initialize OBS - auto-start if needed
	outLog.Println("Checking OBS status...")
	if err := obsws.StartOBSIfNeeded(); err != nil {
		errLog.Printf("Failed to start OBS: %v (continuing anyway)", err)
	}

	// Initialize OBS WebSocket client
	obsClient := obsws.NewClient(obsWebSocketURL, obsPassword)
	if err := obsClient.Connect(); err != nil {
		errLog.Printf("Failed to connect to OBS: %v", err)
		errLog.Println("Please ensure OBS is running and WebSocket server is enabled")
		errLog.Println("  1. Open OBS Studio")
		errLog.Println("  2. Go to Tools > obs-websocket Settings")
		errLog.Println("  3. Enable 'Enable WebSocket server'")
		errLog.Println("  4. Set port to 4455 (default)")
		os.Exit(1)
	}
	defer obsClient.Disconnect()

	obsVersion, wsVersion, _ := obsClient.GetVersion()
	outLog.Printf("Connected to OBS %s (WebSocket %s)", obsVersion, wsVersion)

	// Validate and create required sources (audio + display capture)
	outLog.Println("Checking OBS recording sources...")
	if err := obsClient.EnsureRequiredSources(); err != nil {
		errLog.Printf("Warning: Could not ensure sources: %v", err)
		errLog.Println("  This may cause black/silent recordings")
		errLog.Println("  Please manually add Display Capture and Audio Input sources to your scene")
	} else {
		outLog.Println("OBS recording sources validated (audio + display capture ready)")
	}

	// Set up event handlers
	obsClient.OnRecordStateChanged(func(recording bool) {
		if recording {
			outLog.Println("OBS recording state changed: STARTED")
		} else {
			outLog.Println("OBS recording state changed: STOPPED")
		}
	})

	obsClient.OnDisconnected(func() {
		errLog.Println("OBS disconnected - will attempt reconnection")
	})

	// Initialize state machine
	stateMachine := statemachine.NewStateMachine(cfg)
	outLog.Printf("State machine initialized in %s mode", stateMachine.CurrentMode())

	// Initialize status directory
	statusDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		errLog.Printf("Failed to create status directory: %v", err)
		os.Exit(1)
	}

	// Write initial status
	if err := writeStatus(stateMachine, &detector.DetectionState{}, obsClient); err != nil {
		errLog.Printf("Failed to write initial status: %v", err)
	}

	// Start command file watcher
	go watchCommands(stateMachine, obsClient)

	// Main detection loop
	ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer ticker.Stop()

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	outLog.Printf("Starting detection loop (polling every %ds)...", cfg.PollInterval)

	for {
		select {
		case <-ticker.C:
			// Run detection
			detectionState, err := detector.DetectMeeting(cfg)
			if err != nil {
				errLog.Printf("Detection error: %v", err)
				continue
			}

			// Log detection result with details
			logDetectionResult(detectionState)

			// Process detection through state machine
			shouldStart, shouldStop, app := stateMachine.ProcessDetection(*detectionState)

			// Handle recording actions
			if shouldStart {
				// Capture meeting context for filename
				currentMeetingTitle = detectionState.WindowTitle
				currentMeetingStart = time.Now()
				currentMeetingApp = app
				handleStartRecording(stateMachine, obsClient, app, detectionState.WindowTitle)
			} else if shouldStop {
				handleStopRecording(stateMachine, obsClient)
			}

			// Write status update
			if err := writeStatus(stateMachine, detectionState, obsClient); err != nil {
				errLog.Printf("Failed to write status: %v", err)
			}

		case <-sigChan:
			outLog.Println("Received shutdown signal")

			// Stop recording if active
			if stateMachine.IsRecording() {
				outLog.Println("Stopping active recording before shutdown...")
				handleStopRecording(stateMachine, obsClient)
			}

			outLog.Println("Shutting down gracefully")
			return
		}
	}
}

// handleStartRecording starts OBS recording with appropriate filename
func handleStartRecording(sm *statemachine.StateMachine, obs *obsws.Client, app detector.DetectedApp, windowTitle string) {
	// Generate temporary filename for OBS
	now := time.Now()
	appName := "Meeting"
	switch app {
	case detector.AppZoom:
		appName = "Zoom"
	case detector.AppTeams:
		appName = "Teams"
	}

	tempFilename := fmt.Sprintf("%s_%s_%s_temp.mp4",
		now.Format("2006-01-02"),
		now.Format("1504"),
		appName)

	outLog.Printf("Starting recording: %s (will rename after stop)", tempFilename)

	if err := obs.StartRecord(tempFilename); err != nil {
		errLog.Printf("Failed to start recording: %v", err)
		return
	}

	sm.StartRecording(app)
	outLog.Printf("Recording started successfully (app=%s, streak=%d, title=%q)", app, sm.GetDetectionStreak(), windowTitle)
}

// handleStopRecording stops OBS recording and renames to proper format
func handleStopRecording(sm *statemachine.StateMachine, obs *obsws.Client) {
	duration := sm.RecordingDuration()
	outLog.Printf("Stopping recording after %s (absence_streak=%d)", duration, sm.GetAbsenceStreak())

	outputPath, err := obs.StopRecord()
	if err != nil {
		errLog.Printf("Failed to stop recording: %v", err)
		return
	}

	sm.StopRecording()
	outLog.Printf("Recording stopped successfully: %s", outputPath)

	// Rename file to proper format: YYYY-MM-DD_HHMM_Application_Title.mp4
	appName := "Meeting"
	switch currentMeetingApp {
	case detector.AppZoom:
		appName = "Zoom"
	case detector.AppTeams:
		appName = "Teams"
	}

	// Sanitize window title for filename
	titlePart := fileutil.SanitizeForFilename(currentMeetingTitle)

	// Build new filename
	newBasename := fmt.Sprintf("%s_%s_%s_%s",
		currentMeetingStart.Format("2006-01-02"),
		currentMeetingStart.Format("1504"),
		appName,
		titlePart)

	// Rename the file
	finalPath, err := fileutil.RenameRecording(outputPath, newBasename)
	if err != nil {
		errLog.Printf("Failed to rename recording: %v (original path: %s)", err, outputPath)
		return
	}

	outLog.Printf("Recording renamed to: %s", finalPath)

	// Clear meeting context
	currentMeetingTitle = ""
	currentMeetingApp = detector.AppNone
}

// logDetectionResult logs detection details for debugging
func logDetectionResult(state *detector.DetectionState) {
	if state.MeetingDetected {
		outLog.Printf("Detection: MEETING DETECTED (app=%s, confidence=%s, title=%q) [zoom_proc=%v, zoom_host=%v, zoom_window=%v, teams_proc=%v, teams_window=%v]",
			state.DetectedApp,
			state.ConfidenceLevel,
			state.WindowTitle,
			state.RawDetections.ZoomProcessRunning,
			state.RawDetections.ZoomHostRunning,
			state.RawDetections.ZoomWindowMatch,
			state.RawDetections.TeamsProcessRunning,
			state.RawDetections.TeamsWindowMatch)
		noMeetingLogCounter = 0 // Reset counter when meeting detected
	} else {
		// Log "no meeting" every 20 polls (40-60s) to reduce spam
		noMeetingLogCounter++
		if noMeetingLogCounter >= 20 {
			outLog.Printf("Detection: NO MEETING (zoom_proc=%v, teams_proc=%v)",
				state.RawDetections.ZoomProcessRunning,
				state.RawDetections.TeamsProcessRunning)
			noMeetingLogCounter = 0
		}
	}
}

// writeStatus updates the status.json file
func writeStatus(sm *statemachine.StateMachine, detection *detector.DetectionState, obs *obsws.Client) error {
	recordingState := obs.GetRecordingState()

	status := ipc.StatusSnapshot{
		Mode:             sm.CurrentMode(),
		DetectionState:   *detection,
		RecordingState:   recordingState,
		TeamsDetected:    detection.DetectedApp == detector.AppTeams,
		ZoomDetected:     detection.DetectedApp == detector.AppZoom,
		GoogleMeetActive: detection.DetectedApp == detector.AppGoogleMeet,
		StartStreak:      sm.GetDetectionStreak(),
		StopStreak:       sm.GetAbsenceStreak(),
		Timestamp:        time.Now(),
		OBSConnected:     obs.IsConnected(),
	}

	return ipc.WriteStatus(&status)
}

// watchCommands monitors cmd.txt for manual control commands
func watchCommands(sm *statemachine.StateMachine, obs *obsws.Client) {
	cmdPath := filepath.Join(os.Getenv("HOME"), ".cache", "memofy", "cmd.txt")
	cmdDir := filepath.Dir(cmdPath)

	// Try to use fsnotify for efficient file watching
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errLog.Printf("fsnotify not available, falling back to polling: %v", err)
		watchCommandsWithPolling(cmdPath, sm, obs)
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			errLog.Printf("Failed to close watcher: %v", err)
		}
	}()

	if err := watcher.Add(cmdDir); err != nil {
		errLog.Printf("Failed to watch command directory, falling back to polling: %v", err)
		watchCommandsWithPolling(cmdPath, sm, obs)
		return
	}

	outLog.Println("Command watcher started (using fsnotify)")

	// Add fallback polling ticker in case fsnotify fails
	pollTicker := time.NewTicker(1 * time.Second)
	defer pollTicker.Stop()

	lastCheckTime := time.Now()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				outLog.Println("fsnotify watcher closed, switching to polling")
				watchCommandsWithPolling(cmdPath, sm, obs)
				return
			}

			if event.Name == cmdPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Small delay to ensure write is complete
				time.Sleep(50 * time.Millisecond)

				cmd, err := ipc.ReadCommand()
				if err != nil || cmd == "" {
					continue
				}

				handleCommand(cmd, sm, obs)
				lastCheckTime = time.Now()
			}

		case <-pollTicker.C:
			// Fallback polling: check for commands if file was modified since last check
			if fileInfo, err := os.Stat(cmdPath); err == nil {
				if fileInfo.ModTime().After(lastCheckTime) {
					time.Sleep(50 * time.Millisecond) // Ensure write is complete

					cmd, err := ipc.ReadCommand()
					if err == nil && cmd != "" {
						handleCommand(cmd, sm, obs)
						lastCheckTime = time.Now()
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				outLog.Println("fsnotify error channel closed, switching to polling")
				watchCommandsWithPolling(cmdPath, sm, obs)
				return
			}
			errLog.Printf("File watcher error: %v", err)
		}
	}
}

// watchCommandsWithPolling is a pure polling-based fallback for command monitoring
func watchCommandsWithPolling(cmdPath string, sm *statemachine.StateMachine, obs *obsws.Client) {
	outLog.Println("Command watcher started (using polling fallback, 1s interval)")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastCheckTime := time.Now()

	for range ticker.C {
		// Check if file was modified since last check
		fileInfo, err := os.Stat(cmdPath)
		if err != nil {
			continue // File doesn't exist yet, keep polling
		}

		if fileInfo.ModTime().After(lastCheckTime) {
			time.Sleep(50 * time.Millisecond) // Ensure write is complete

			cmd, err := ipc.ReadCommand()
			if err == nil && cmd != "" {
				handleCommand(cmd, sm, obs)
			}
			lastCheckTime = time.Now()
		}
	}
}

// handleCommand processes manual control commands
func handleCommand(cmd ipc.Command, sm *statemachine.StateMachine, obs *obsws.Client) {
	outLog.Printf("Received command: %s", cmd)

	switch cmd {
	case ipc.CmdStart:
		if err := sm.ForceStart(detector.AppNone); err != nil {
			errLog.Printf("ForceStart failed: %v", err)
			return
		}
		// Capture context for manual start
		currentMeetingTitle = "Manual"
		currentMeetingStart = time.Now()
		currentMeetingApp = detector.AppNone
		handleStartRecording(sm, obs, detector.AppNone, "Manual")

	case ipc.CmdStop:
		if err := sm.ForceStop(); err != nil {
			errLog.Printf("ForceStop failed: %v", err)
			return
		}
		handleStopRecording(sm, obs)

	case ipc.CmdAuto:
		sm.SetMode(ipc.ModeAuto)
		outLog.Println("Mode changed to AUTO")

	case ipc.CmdPause:
		sm.SetMode(ipc.ModePaused)
		outLog.Println("Mode changed to PAUSED")

	case ipc.CmdToggle:
		// Toggle recording state: if recording, stop; else start
		if sm.IsRecording() {
			if err := sm.ForceStop(); err != nil {
				errLog.Printf("ForceStop failed: %v", err)
				return
			}
			handleStopRecording(sm, obs)
		} else {
			if err := sm.ForceStart(detector.AppNone); err != nil {
				errLog.Printf("ForceStart failed: %v", err)
				return
			}
			// Capture context for manual toggle
			currentMeetingTitle = "Manual"
			currentMeetingStart = time.Now()
			currentMeetingApp = detector.AppNone
			handleStartRecording(sm, obs, detector.AppNone, "Manual")
		}

	case ipc.CmdQuit:
		outLog.Println("Quit command received - shutting down")
		os.Exit(0)

	default:
		errLog.Printf("Unknown command: %s", cmd)
	}
}

// initLogging sets up log files with rotation support
func initLogging() error {
	// Create log directory if it doesn't exist
	logDir := "/tmp"

	// Rotate logs if they exceed 10MB
	outLogPath := filepath.Join(logDir, "memofy-core.out.log")
	errLogPath := filepath.Join(logDir, "memofy-core.err.log")

	if err := rotateLogIfNeeded(outLogPath, 10*1024*1024); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to rotate out log: %v\n", err)
	}

	if err := rotateLogIfNeeded(errLogPath, 10*1024*1024); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to rotate err log: %v\n", err)
	}

	outFile, err := os.OpenFile(outLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	errFile, err := os.OpenFile(errLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	outLog = log.New(outFile, logPrefix+" ", log.LstdFlags)
	errLog = log.New(errFile, logPrefix+" ERROR: ", log.LstdFlags)

	return nil
}

// rotateLogIfNeeded rotates a log file if it exceeds maxSize bytes
func rotateLogIfNeeded(logPath string, maxSize int64) error {
	info, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		return nil // Log doesn't exist yet
	}
	if err != nil {
		return err
	}

	if info.Size() < maxSize {
		return nil // Log is under size limit
	}

	// Rotate: rename current log to .old, removing previous .old
	oldPath := logPath + ".old"
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old log: %w", err)
	}

	if err := os.Rename(logPath, oldPath); err != nil {
		return err
	}

	return nil
}

// checkPermissions verifies required macOS permissions
func checkPermissions() error {
	// Note: Actual permission checks require CGO and macOS frameworks
	// For now, this is a placeholder that would use:
	// - CGPreflightScreenCaptureAccess() for Screen Recording
	// - AXIsProcessTrusted() for Accessibility

	// In production, these would call into darwinkit/macos APIs
	outLog.Println("Permission check: Screen Recording - OK (assumed)")
	outLog.Println("Permission check: Accessibility - OK (assumed)")

	return nil
}
