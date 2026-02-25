package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tiroq/memofy/internal/asr"
	"github.com/tiroq/memofy/internal/asr/googlestt"
	"github.com/tiroq/memofy/internal/asr/localwhisper"
	"github.com/tiroq/memofy/internal/asr/remotewhisper"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/detector"
	"github.com/tiroq/memofy/internal/diaglog"
	"github.com/tiroq/memofy/internal/fileutil"
	"github.com/tiroq/memofy/internal/ipc"
	"github.com/tiroq/memofy/internal/obsws"
	"github.com/tiroq/memofy/internal/pidfile"
	"github.com/tiroq/memofy/internal/recorder"
	"github.com/tiroq/memofy/internal/statemachine"
	"github.com/tiroq/memofy/internal/transcript"
	"github.com/tiroq/memofy/internal/validation"
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

var (
	// T033: ASR registry, set during startup if ASR is configured. FR-013
	asrRegistry *asr.Registry

	// T033: ASR mutex guards asrTranscribing flag. FR-013
	asrMu           sync.Mutex
	asrTranscribing bool
	asrLastFile     string
	asrLastErr      string

	// T033: package-level config ref for goroutine access. FR-013
	detCfg *config.DetectionConfig
)

func main() {
	// T020: --export-diag subcommand: read log, write bundle, exit (FR-006).
	if len(os.Args) > 1 && os.Args[1] == "--export-diag" {
		logPath := os.Getenv("MEMOFY_LOG_PATH")
		if logPath == "" {
			logPath = "/tmp/memofy-debug.log"
		}
		diaglog.Version = Version
		path, n, err := diaglog.Export(logPath, ".")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "hint: run with MEMOFY_DEBUG_RECORDING=true to enable logging")
				os.Exit(1)
			}
			os.Exit(2)
		}
		fmt.Printf("Wrote: %s (%d lines)\n", path, n)
		os.Exit(0)
	}

	// Recover from any panics and log them
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC in memofy-core: %v\n", r)
			if outLog != nil {
				outLog.Printf("PANIC: %v", r)
			}
			if errLog != nil {
				errLog.Printf("PANIC: %v", r)
			}
			os.Exit(1)
		}
	}()

	// Initialize logging
	if err := initLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}

	outLog.Println("===========================================")
	outLog.Println("Starting Memofy Core v" + Version + "...")
	outLog.Printf("PID: %d", os.Getpid())
	outLog.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))
	outLog.Println("===========================================")

	// Check for duplicate instances
	pidFilePath := pidfile.GetPIDFilePath("memofy-core")
	outLog.Printf("Checking PID file: %s", pidFilePath)
	pf, err := pidfile.New(pidFilePath)
	if err != nil {
		errLog.Printf("Failed to create PID file: %v", err)
		errLog.Println("Another instance of memofy-core may already be running.")
		errLog.Printf("If you're sure no other instance is running, remove: %s", pidFilePath)
		os.Exit(1)
	}
	defer func() {
		outLog.Println("Cleaning up before exit...")
		if err := pf.Remove(); err != nil {
			errLog.Printf("Warning: failed to remove PID file: %v", err)
		}
	}()
	outLog.Printf("PID file created: %s (PID %d)", pidFilePath, os.Getpid())

	// Check macOS permissions
	outLog.Println("[STARTUP] Checking macOS permissions...")
	if err := checkPermissions(); err != nil {
		errLog.Printf("Permission check failed: %v", err)
		errLog.Println("Please grant Screen Recording and Accessibility permissions in System Preferences > Security & Privacy")
		os.Exit(1)
	}
	outLog.Println("[STARTUP] Permissions check passed")

	// Load detection configuration
	outLog.Println("[STARTUP] Loading detection configuration...")
	cfg, err := config.LoadDetectionRules()
	if err != nil {
		errLog.Printf("Failed to load detection config: %v", err)
		os.Exit(1)
	}
	outLog.Printf("[STARTUP] Loaded detection config: %d rules, poll_interval=%ds, thresholds=%d/%d",
		len(cfg.Rules), cfg.PollInterval, cfg.StartThreshold, cfg.StopThreshold)
	detCfg = cfg // T033: store for goroutine access (FR-013)

	// T033: Initialize ASR registry if ASR is configured. FR-013
	if cfg.ASR != nil && cfg.ASR.Enabled {
		asrRegistry = asr.NewRegistry()
		switch cfg.ASR.Backend {
		case "remote_whisper_api":
			c := remotewhisper.NewClient(remotewhisper.Config{
				BaseURL:        cfg.ASR.Remote.BaseURL,
				Token:          cfg.ASR.Remote.Token,
				TimeoutSeconds: cfg.ASR.Remote.TimeoutSeconds,
				Retries:        cfg.ASR.Remote.Retries,
				Model:          cfg.ASR.Remote.Model,
			})
			asrRegistry.Register("remote_whisper_api", c)
		case "local_whisper":
			b := localwhisper.NewBackend(localwhisper.Config{
				BinaryPath: cfg.ASR.Local.BinaryPath,
				ModelPath:  cfg.ASR.Local.ModelPath,
				Model:      cfg.ASR.Local.Model,
				Threads:    cfg.ASR.Local.Threads,
			})
			asrRegistry.Register("local_whisper", b)
		case "google_stt":
			b := googlestt.NewBackend(googlestt.Config{
				CredentialsFile: cfg.ASR.Google.CredentialsFile,
				LanguageCode:    cfg.ASR.Google.LanguageCode,
			})
			asrRegistry.Register("google_stt", b)
		}
		if cfg.ASR.FallbackBackend != "" {
			asrRegistry.SetFallback(cfg.ASR.FallbackBackend)
		}
		outLog.Printf("[STARTUP] ASR enabled (backend=%s, mode=%s)", cfg.ASR.Backend, cfg.ASR.Mode)
	} else {
		outLog.Println("[STARTUP] ASR disabled (not configured)")
	}

	// Initialize OBS - auto-start if needed
	outLog.Println("[STARTUP] Checking OBS status...")
	if err := obsws.StartOBSIfNeeded(); err != nil {
		errLog.Printf("[STARTUP] Failed to start OBS: %v (continuing anyway)", err)
	}

	// Initialize OBS WebSocket client
	outLog.Println("[STARTUP] Connecting to OBS WebSocket at " + obsWebSocketURL + "...")
	obsClient := obsws.NewClient(obsWebSocketURL, obsPassword)
	if err := obsClient.Connect(); err != nil {
		errLog.Printf("[STARTUP] Failed to connect to OBS: %v", err)
		errLog.Println("Please ensure OBS is running and WebSocket server is enabled")
		errLog.Println("  1. Open OBS Studio")
		errLog.Println("  2. Go to Tools > obs-websocket Settings")
		errLog.Println("  3. Enable 'Enable WebSocket server'")
		errLog.Println("  4. Set port to 4455 (default)")
		os.Exit(1)
	}
	outLog.Println("[STARTUP] Successfully connected to OBS, setting up deferred cleanup...")
	defer func() {
		outLog.Println("[SHUTDOWN] Disconnecting from OBS...")
		obsClient.Disconnect()
	}()

	obsVersion, wsVersion, _ := obsClient.GetVersion()
	outLog.Printf("[STARTUP] Connected to OBS %s (WebSocket %s)", obsVersion, wsVersion)

	// Validate OBS compatibility
	outLog.Println("[STARTUP] Validating OBS compatibility...")
	healthCheck := validation.CheckOBSHealth(obsVersion, wsVersion)
	outLog.Printf("[STARTUP] OBS Health: %s", healthCheck.Message)
	if !healthCheck.OK {
		errLog.Println("[STARTUP] WARNING: OBS compatibility check found issues:")
		for _, issue := range healthCheck.Issues {
			errLog.Printf("  - %s", issue)
		}
		errLog.Println("")
		errLog.Println("Suggested fixes:")
		for _, fix := range healthCheck.Fixes {
			errLog.Printf("  - %s", fix)
		}
		errLog.Println("")
		errLog.Println("Continuing anyway, but recording may not work properly.")
		errLog.Println("Run 'memofy-ctl diagnose' for more information.")
	}

	// Validate and create required sources (audio + display capture)
	outLog.Println("[STARTUP] Checking OBS recording sources...")
	if err := obsClient.EnsureRequiredSources(); err != nil {
		errLog.Printf("Warning: Could not ensure sources: %v", err)
		errLog.Println("  This may cause black/silent recordings")
		errLog.Println("  Please manually add Display Capture and Audio Input sources to your scene")
	} else {
		outLog.Println("[STARTUP] OBS recording sources validated (audio + display capture ready)")
	}

	// Set up event handlers
	outLog.Println("[STARTUP] Setting up OBS event handlers...")
	obsClient.OnRecordStateChanged(func(recording bool) {
		if recording {
			outLog.Println("[EVENT] OBS recording state changed: STARTED")
		} else {
			outLog.Println("[EVENT] OBS recording state changed: STOPPED")
		}
	})

	obsClient.OnDisconnected(func() {
		errLog.Println("[EVENT] OBS disconnected - will attempt reconnection")
	})
	outLog.Println("[STARTUP] Event handlers registered")

	// Initialize state machine
	outLog.Println("[STARTUP] Initializing state machine...")
	stateMachine := statemachine.NewStateMachine(cfg)
	outLog.Printf("[STARTUP] State machine initialized in %s mode", stateMachine.CurrentMode())

	// T022: init diaglog (FR-001/FR-004/FR-005)
	logPath := os.Getenv("MEMOFY_LOG_PATH")
	if logPath == "" {
		logPath = "/tmp/memofy-debug.log"
	}
	diagLogger, diagErr := diaglog.New(logPath)
	if diagErr != nil {
		errLog.Printf("[STARTUP] WARNING: could not open diagnostic log at %s: %v (continuing)", logPath, diagErr)
		diagLogger = diaglog.NewNoOp()
	}
	defer func() { _ = diagLogger.Close() }()
	diaglog.Version = Version

	// Wire debounce from env var (FR-008)
	debounceDur := 5 * time.Second
	if ms := os.Getenv("MEMOFY_MANUAL_DEBOUNCE_MS"); ms != "" {
		if msInt, err := strconv.Atoi(ms); err == nil && msInt > 0 {
			debounceDur = time.Duration(msInt) * time.Millisecond
		}
	}
	stateMachine.SetLogger(diagLogger)
	stateMachine.SetDebounceDuration(debounceDur)
	obsClient.SetLogger(diagLogger) // T022: wire logger into OBS client (FR-001)

	// T035: ASR backend health check at startup. FR-013
	if asrRegistry != nil {
		for _, name := range asrRegistry.Backends() {
			b, _ := asrRegistry.Get(name)
			if b == nil {
				continue
			}
			hs, err := b.HealthCheck()
			if err != nil {
				errLog.Printf("[STARTUP] ASR health check error (backend=%s): %v", name, err)
				diagLogger.Log(diaglog.LogEntry{
					Component: diaglog.ComponentASR,
					Event:     diaglog.EventASRHealthCheck,
					Payload: map[string]interface{}{
						"backend": name,
						"ok":      false,
						"error":   err.Error(),
					},
				})
			} else if !hs.OK {
				errLog.Printf("[STARTUP] WARNING: ASR backend %s unhealthy: %s", name, hs.Message)
				diagLogger.Log(diaglog.LogEntry{
					Component: diaglog.ComponentASR,
					Event:     diaglog.EventASRHealthCheck,
					Payload: map[string]interface{}{
						"backend": name,
						"ok":      false,
						"message": hs.Message,
					},
				})
			} else {
				outLog.Printf("[STARTUP] ASR backend %s healthy (latency=%s)", name, hs.Latency)
				diagLogger.Log(diaglog.LogEntry{
					Component: diaglog.ComponentASR,
					Event:     diaglog.EventASRHealthCheck,
					Payload: map[string]interface{}{
						"backend": name,
						"ok":      true,
						"latency": hs.Latency.String(),
					},
				})
			}
		}
	}

	// T031: Wrap OBS client in recorder interface (FR-013)
	rec := recorder.NewOBSAdapter(obsClient)

	// Initialize OBS WebSocket client
	outLog.Println("[STARTUP] Creating status directory...")
	statusDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		errLog.Printf("Failed to create status directory: %v", err)
		os.Exit(1)
	}

	// Write initial status
	outLog.Println("[STARTUP] Writing initial status...")
	if err := writeStatus(stateMachine, &detector.DetectionState{}, rec); err != nil {
		errLog.Printf("Failed to write initial status: %v", err)
	}

	// Start command file watcher
	outLog.Println("[STARTUP] Starting command file watcher...")
	go watchCommands(stateMachine, rec)

	// Main detection loop
	ticker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer ticker.Stop()

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	outLog.Println("[STARTUP] Signal handlers registered (SIGINT, SIGTERM)")

	outLog.Printf("[STARTUP] Starting detection loop (polling every %ds)...", cfg.PollInterval)
	outLog.Println("===========================================")
	outLog.Println("[RUNNING] Memofy Core is running and monitoring")

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
				handleStartRecording(stateMachine, rec, app, detectionState.WindowTitle)
			} else if shouldStop {
				handleStopRecording(stateMachine, rec, statemachine.StopRequest{
					RequestOrigin: statemachine.OriginAuto,
					Reason:        "auto_detection_stop",
					Component:     "auto-detector",
				})
			}

			// Write status update
			if err := writeStatus(stateMachine, detectionState, rec); err != nil {
				errLog.Printf("Failed to write status: %v", err)
			}

		case <-sigChan:
			outLog.Println("===========================================")
			outLog.Printf("[SHUTDOWN] Received shutdown signal at %s", time.Now().Format(time.RFC3339))

			// Stop recording if active
			if stateMachine.IsRecording() {
				outLog.Println("[SHUTDOWN] Recording is active - stopping before shutdown...")
				handleStopRecording(stateMachine, rec, statemachine.StopRequest{
					RequestOrigin: statemachine.OriginManual,
					Reason:        "daemon_shutdown",
					Component:     "memofy-core",
				})
			}

			outLog.Println("[SHUTDOWN] Shutting down gracefully")
			outLog.Println("===========================================")
			return
		}
	}
}

// handleStartRecording starts OBS recording with appropriate filename
func handleStartRecording(sm *statemachine.StateMachine, rec recorder.Recorder, app detector.DetectedApp, windowTitle string) {
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

	if err := rec.StartRecording(tempFilename); err != nil {
		errLog.Printf("Failed to start recording: %v", err)
		return
	}

	sm.StartRecording(app)
	outLog.Printf("Recording started successfully (app=%s, streak=%d, title=%q)", app, sm.GetDetectionStreak(), windowTitle)
}

// handleStopRecording stops OBS recording when the authority check passes,
// then renames the output file to the proper format.
func handleStopRecording(sm *statemachine.StateMachine, rec recorder.Recorder, req statemachine.StopRequest) {
	if !sm.IsRecording() {
		return
	}
	duration := sm.RecordingDuration()
	outLog.Printf("Stopping recording after %s (absence_streak=%d, origin=%s, reason=%s)",
		duration, sm.GetAbsenceStreak(), req.RequestOrigin, req.Reason)

	// Authority check without mutating state (FR-007/FR-008); state is only
	// committed after OBS confirms the stop succeeded.
	if !sm.CanStop(req) {
		errLog.Printf("Stop request rejected (origin=%s): manual session protection active", req.RequestOrigin)
		return
	}

	result, err := rec.StopRecording(req.Reason)
	if err != nil {
		errLog.Printf("Failed to stop recording: %v", err)
		return
	}

	// Commit state change only after OBS confirmed the stop (FR-007/FR-008).
	sm.StopRecording(req)

	outLog.Printf("Recording stopped successfully: %s", result.OutputPath)

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
	finalPath, err := fileutil.RenameRecording(result.OutputPath, newBasename)
	if err != nil {
		errLog.Printf("Failed to rename recording: %v (original path: %s)", err, result.OutputPath)
		return
	}

	outLog.Printf("Recording renamed to: %s", finalPath)

	// T034: Write sidecar metadata JSON alongside recording. FR-013
	stopTime := time.Now()
	recDuration := stopTime.Sub(currentMeetingStart)
	meta := &fileutil.RecordingMetadata{
		Version:         Version,
		SessionID:       sm.SessionID(),
		StartedAt:       currentMeetingStart,
		StoppedAt:       stopTime,
		Duration:        recDuration.String(),
		DurationMs:      recDuration.Milliseconds(),
		App:             string(currentMeetingApp),
		WindowTitle:     currentMeetingTitle,
		Origin:          string(req.RequestOrigin),
		RecorderBackend: "obs",
		OutputFile:      finalPath,
	}
	if err := fileutil.WriteMetadata(finalPath, meta); err != nil {
		errLog.Printf("Failed to write metadata for %s: %v", finalPath, err)
	}

	// T033: Launch batch ASR transcription in a goroutine if configured. FR-013
	if asrRegistry != nil {
		formats := []string{"txt"}
		if detCfg != nil && detCfg.ASR != nil && len(detCfg.ASR.OutputFormats) > 0 {
			formats = detCfg.ASR.OutputFormats
		}
		go func(filePath string, fmts []string) {
			asrMu.Lock()
			asrTranscribing = true
			asrLastFile = filePath
			asrLastErr = ""
			asrMu.Unlock()

			defer func() {
				asrMu.Lock()
				asrTranscribing = false
				asrMu.Unlock()
			}()

			t, err := asrRegistry.TranscribeWithFallback(filePath, asr.TranscribeOptions{
				Timestamps: true,
			})
			if err != nil {
				asrMu.Lock()
				asrLastErr = err.Error()
				asrMu.Unlock()
				errLog.Printf("ASR transcription failed for %s: %v", filePath, err)
				return
			}
			basePath := strings.TrimSuffix(filePath, filepath.Ext(filePath))
			if err := transcript.WriteAll(basePath, t, fmts); err != nil {
				asrMu.Lock()
				asrLastErr = err.Error()
				asrMu.Unlock()
				errLog.Printf("Failed to write transcript for %s: %v", filePath, err)
				return
			}
			outLog.Printf("ASR transcript written: %s (.%s)", basePath, strings.Join(fmts, ", ."))
		}(finalPath, formats)
	}

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
func writeStatus(sm *statemachine.StateMachine, detection *detector.DetectionState, rec recorder.Recorder) error {
	// T031: map recorder.RecorderState → obsws.RecordingState for backward-compatible JSON (FR-013)
	rs := rec.GetState()
	recordingState := obsws.RecordingState{
		Recording:   rs.Recording,
		StartTime:   rs.StartTime,
		Duration:    rs.Duration,
		OutputPath:  rs.OutputPath,
		LastUpdated: time.Now(),
	}
	if rs.Connected {
		recordingState.OBSStatus = "connected"
	} else {
		recordingState.OBSStatus = "disconnected"
	}

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
		OBSConnected:     rec.IsConnected(),
		// T023: populate session fields (FR-012)
		RecordingOrigin: string(sm.SessionOrigin()),
		SessionID:       sm.SessionID(),
		// T033: populate ASR transcription state (FR-013)
		ASRState: func() *ipc.ASRState {
			asrMu.Lock()
			defer asrMu.Unlock()
			if !asrTranscribing && asrLastFile == "" && asrLastErr == "" {
				return nil
			}
			return &ipc.ASRState{
				Transcribing:        asrTranscribing,
				LastTranscribedFile: asrLastFile,
				LastError:           asrLastErr,
			}
		}(),
	}

	return ipc.WriteStatus(&status)
}

// updateStatusMode updates only the mode in status.json, preserving detection state
func updateStatusMode(sm *statemachine.StateMachine, rec recorder.Recorder) error {
	// Read current status
	status, err := ipc.ReadStatus()
	if err != nil {
		// If status doesn't exist yet, create a new one with empty detection
		return writeStatus(sm, &detector.DetectionState{}, rec)
	}

	// Update only the mode and timestamp
	status.Mode = sm.CurrentMode()
	status.Timestamp = time.Now()
	status.OBSConnected = rec.IsConnected()

	return ipc.WriteStatus(status)
}

// watchCommands monitors cmd.txt for manual control commands
func watchCommands(sm *statemachine.StateMachine, rec recorder.Recorder) {
	cmdPath := filepath.Join(os.Getenv("HOME"), ".cache", "memofy", "cmd.txt")
	cmdDir := filepath.Dir(cmdPath)

	// Try to use fsnotify for efficient file watching
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errLog.Printf("fsnotify not available, falling back to polling: %v", err)
		watchCommandsWithPolling(cmdPath, sm, rec)
		return
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			errLog.Printf("Failed to close watcher: %v", err)
		}
	}()

	if err := watcher.Add(cmdDir); err != nil {
		errLog.Printf("Failed to watch command directory, falling back to polling: %v", err)
		watchCommandsWithPolling(cmdPath, sm, rec)
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
				watchCommandsWithPolling(cmdPath, sm, rec)
				return
			}

			if event.Name == cmdPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				// Small delay to ensure write is complete
				time.Sleep(50 * time.Millisecond)

				cmd, err := ipc.ReadCommand()
				if err != nil || cmd == "" {
					continue
				}

				handleCommand(cmd, sm, rec)
				lastCheckTime = time.Now()
			}

		case <-pollTicker.C:
			// Fallback polling: check for commands if file was modified since last check
			if fileInfo, err := os.Stat(cmdPath); err == nil {
				if fileInfo.ModTime().After(lastCheckTime) {
					time.Sleep(50 * time.Millisecond) // Ensure write is complete

					cmd, err := ipc.ReadCommand()
					if err == nil && cmd != "" {
						handleCommand(cmd, sm, rec)
						lastCheckTime = time.Now()
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				outLog.Println("fsnotify error channel closed, switching to polling")
				watchCommandsWithPolling(cmdPath, sm, rec)
				return
			}
			errLog.Printf("File watcher error: %v", err)
		}
	}
}

// watchCommandsWithPolling is a pure polling-based fallback for command monitoring
func watchCommandsWithPolling(cmdPath string, sm *statemachine.StateMachine, rec recorder.Recorder) {
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
				handleCommand(cmd, sm, rec)
			}
			lastCheckTime = time.Now()
		}
	}
}

// handleCommand processes manual control commands
func handleCommand(cmd ipc.Command, sm *statemachine.StateMachine, rec recorder.Recorder) {
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
		handleStartRecording(sm, rec, detector.AppNone, "Manual")

	case ipc.CmdStop:
		handleStopRecording(sm, rec, statemachine.StopRequest{
			RequestOrigin: statemachine.OriginManual,
			Reason:        "user_stop",
			Component:     "memofy-core",
		})

	case ipc.CmdAuto:
		sm.SetMode(ipc.ModeAuto)
		outLog.Println("Mode changed to AUTO")
		// Immediately update status so UI reflects the change
		if err := updateStatusMode(sm, rec); err != nil {
			errLog.Printf("Failed to write status after mode change: %v", err)
		}

	case ipc.CmdPause:
		sm.SetMode(ipc.ModePaused)
		outLog.Println("Mode changed to PAUSED")
		// Immediately update status so UI reflects the change
		if err := updateStatusMode(sm, rec); err != nil {
			errLog.Printf("Failed to write status after mode change: %v", err)
		}

	case ipc.CmdManual:
		sm.SetMode(ipc.ModeManual)
		outLog.Println("Mode changed to MANUAL (detection active, OBS control disabled)")
		// Immediately update status so UI reflects the change
		if err := updateStatusMode(sm, rec); err != nil {
			errLog.Printf("Failed to write status after mode change: %v", err)
		}

	case ipc.CmdToggle:
		// Toggle recording state: if recording, stop; else start
		if sm.IsRecording() {
			handleStopRecording(sm, rec, statemachine.StopRequest{
				RequestOrigin: statemachine.OriginManual,
				Reason:        "user_stop",
				Component:     "memofy-core",
			})
		} else {
			if err := sm.ForceStart(detector.AppNone); err != nil {
				errLog.Printf("ForceStart failed: %v", err)
				return
			}
			// Capture context for manual toggle
			currentMeetingTitle = "Manual"
			currentMeetingStart = time.Now()
			currentMeetingApp = detector.AppNone
			handleStartRecording(sm, rec, detector.AppNone, "Manual")
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
	outLog.Println("[PERMS] Screen Recording - OK (assumed)")
	outLog.Println("[PERMS] Accessibility - OK (assumed)")

	// Try to detect if we have actual permissions by checking a runtime probe
	pwd, err := os.Getwd()
	if err != nil {
		outLog.Printf("[PERMS] WARNING: Could not get working directory: %v", err)
	} else {
		outLog.Printf("[PERMS] Working directory: %s", pwd)
	}

	// Check if we can write to temp directory
	testFile := filepath.Join("/tmp", ".memofy-core-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		outLog.Printf("[PERMS] WARNING: Cannot write to /tmp: %v", err)
	} else {
		_ = os.Remove(testFile)
		outLog.Println("[PERMS] Write test to /tmp: PASS")
	}

	return nil
}
