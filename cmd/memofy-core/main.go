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
	outLog *log.Logger
	errLog *log.Logger
)

func main() {
	// Initialize logging
	if err := initLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}

	outLog.Println("Starting Memofy Core v0.1...")

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

	// Initialize OBS WebSocket client
	obsClient := obsws.NewClient(obsWebSocketURL, obsPassword)
	if err := obsClient.Connect(); err != nil {
		errLog.Printf("Failed to connect to OBS: %v", err)
		errLog.Println("Make sure OBS is running with WebSocket server enabled")
		os.Exit(1)
	}
	defer obsClient.Disconnect()

	obsVersion, wsVersion, _ := obsClient.GetVersion()
	outLog.Printf("Connected to OBS %s (WebSocket %s)", obsVersion, wsVersion)

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
				handleStartRecording(stateMachine, obsClient, app)
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
func handleStartRecording(sm *statemachine.StateMachine, obs *obsws.Client, app detector.DetectedApp) {
	// Generate filename: YYYY-MM-DD_HHMM_Application_Title.mp4
	now := time.Now()
	appName := "Meeting"
	if app == detector.AppZoom {
		appName = "Zoom"
	} else if app == detector.AppTeams {
		appName = "Teams"
	}

	filename := fmt.Sprintf("%s_%s_%s.mp4",
		now.Format("2006-01-02"),
		now.Format("1504"),
		appName)

	outLog.Printf("Starting recording: %s", filename)

	if err := obs.StartRecord(filename); err != nil {
		errLog.Printf("Failed to start recording: %v", err)
		return
	}

	sm.StartRecording(app)
	outLog.Printf("Recording started successfully (app=%s, streak=%d)", app, sm.GetDetectionStreak())
}

// handleStopRecording stops OBS recording
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
}

// logDetectionResult logs detection details for debugging
func logDetectionResult(state *detector.DetectionState) {
	if state.MeetingDetected {
		outLog.Printf("Detection: MEETING DETECTED (app=%s, confidence=%s) [zoom_proc=%v, zoom_host=%v, zoom_window=%v, teams_proc=%v, teams_window=%v]",
			state.DetectedApp,
			state.ConfidenceLevel,
			state.RawDetections.ZoomProcessRunning,
			state.RawDetections.ZoomHostRunning,
			state.RawDetections.ZoomWindowMatch,
			state.RawDetections.TeamsProcessRunning,
			state.RawDetections.TeamsWindowMatch)
	} else {
		// Only log "no meeting" every 10 polls to reduce log spam
		// (Could add a counter here if needed)
	}
}

// writeStatus updates the status.json file
func writeStatus(sm *statemachine.StateMachine, detection *detector.DetectionState, obs *obsws.Client) error {
	recordingState := obs.GetRecordingState()

	status := ipc.StatusSnapshot{
		Mode:            sm.CurrentMode(),
		DetectionState:  *detection,
		RecordingState:  recordingState,
		DetectionStreak: sm.GetDetectionStreak(),
		AbsenceStreak:   sm.GetAbsenceStreak(),
		LastUpdated:     time.Now(),
	}

	return ipc.WriteStatus(&status)
}

// watchCommands monitors cmd.txt for manual control commands
func watchCommands(sm *statemachine.StateMachine, obs *obsws.Client) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errLog.Printf("Failed to create file watcher: %v", err)
		return
	}
	defer watcher.Close()

	cmdPath := filepath.Join(os.Getenv("HOME"), ".cache", "memofy", "cmd.txt")
	cmdDir := filepath.Dir(cmdPath)

	if err := watcher.Add(cmdDir); err != nil {
		errLog.Printf("Failed to watch command directory: %v", err)
		return
	}

	outLog.Println("Command watcher started")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
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
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			errLog.Printf("File watcher error: %v", err)
		}
	}
}

// handleCommand processes manual control commands
func handleCommand(cmd string, sm *statemachine.StateMachine, obs *obsws.Client) {
	outLog.Printf("Received command: %s", cmd)

	switch cmd {
	case ipc.CmdStart:
		if err := sm.ForceStart(detector.AppNone); err != nil {
			errLog.Printf("ForceStart failed: %v", err)
			return
		}
		handleStartRecording(sm, obs, detector.AppNone)

	case ipc.CmdStop:
		if err := sm.ForceStop(); err != nil {
			errLog.Printf("ForceStop failed: %v", err)
			return
		}
		handleStopRecording(sm, obs)

	case ipc.CmdAuto:
		sm.SetMode(ipc.ModeAuto)
		outLog.Println("Mode changed to AUTO")

	case ipc.CmdPaused:
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
			handleStartRecording(sm, obs, detector.AppNone)
		}

	case ipc.CmdQuit:
		outLog.Println("Quit command received - shutting down")
		os.Exit(0)

	default:
		errLog.Printf("Unknown command: %s", cmd)
	}
}

// initLogging sets up log files
func initLogging() error {
	outFile, err := os.OpenFile("/tmp/memofy-core.out.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	errFile, err := os.OpenFile("/tmp/memofy-core.err.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	outLog = log.New(outFile, logPrefix+" ", log.LstdFlags)
	errLog = log.New(errFile, logPrefix+" ERROR: ", log.LstdFlags)

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
