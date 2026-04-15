// Package engine implements the main audio recording loop.
package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/tiroq/memofy/internal/audio"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/metadata"
	"github.com/tiroq/memofy/internal/monitor"
	"github.com/tiroq/memofy/internal/statemachine"
	"github.com/tiroq/memofy/internal/wav"
)

// Engine is the main recording controller.
type Engine struct {
	cfg         config.Config
	sm          *statemachine.StateMachine
	mon         *monitor.Monitor
	stream      *audio.Stream
	writer      *wav.Writer
	logger      *log.Logger
	mu          sync.Mutex
	running     bool
	stopCh      chan struct{}
	recordStart time.Time
	outputDir   string
	currentFile string
	monSnapshot monitor.Snapshot
}

// New creates a new Engine with the given configuration.
func New(cfg config.Config, logger *log.Logger) *Engine {
	if logger == nil {
		logger = log.New(os.Stderr, "[memofy] ", log.LstdFlags)
	}
	dur := time.Duration(cfg.Audio.SilenceSeconds) * time.Second
	return &Engine{
		cfg:    cfg,
		sm:     statemachine.New(dur),
		mon:    monitor.New(),
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start initializes audio capture and begins the recording loop.
func (e *Engine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.mu.Unlock()
	e.outputDir = e.cfg.Output.Dir
	if err := os.MkdirAll(e.outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := audio.Init(); err != nil {
		return fmt.Errorf("audio init: %w", err)
	}
	dev, err := e.findDevice()
	if err != nil {
		audio.Terminate()
		return err
	}
	e.logger.Printf("Using device: %s (idx=%d ch=%d rate=%.0f)",
		dev.Name, dev.Index, dev.MaxInputCh, dev.SampleRate)
	channels := e.cfg.Audio.Channels
	if channels > dev.MaxInputCh {
		channels = dev.MaxInputCh
	}
	sampleRate := e.cfg.Audio.SampleRate
	if sampleRate == 0 {
		sampleRate = int(dev.SampleRate)
	}
	stream, err := audio.OpenStream(audio.CaptureConfig{
		DeviceIndex:     dev.Index,
		SampleRate:      sampleRate,
		Channels:        channels,
		FramesPerBuffer: 4096,
	})
	if err != nil {
		audio.Terminate()
		return fmt.Errorf("open stream: %w", err)
	}
	e.stream = stream
	if err := stream.Start(); err != nil {
		stream.Close()
		audio.Terminate()
		return fmt.Errorf("start stream: %w", err)
	}
	e.mu.Lock()
	e.running = true
	e.stopCh = make(chan struct{})
	e.mu.Unlock()
	e.sm.SetOnStateChange(func(from, to statemachine.State) {
		e.logger.Printf("State: %s -> %s", from, to)
	})
	go e.loop()
	go e.pollMonitor()
	e.logger.Printf("Started (threshold=%.4f silence=%ds)",
		e.cfg.Audio.Threshold, e.cfg.Audio.SilenceSeconds)
	return nil
}

// Stop gracefully shuts down the engine.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	close(e.stopCh)
	e.mu.Unlock()
	e.finalizeRecording()
	if e.stream != nil {
		e.stream.Stop()
		e.stream.Close()
	}
	audio.Terminate()
	e.logger.Println("Engine stopped")
}

// Status returns a human-readable status string.
func (e *Engine) Status() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	state := e.sm.CurrentState()
	snap := e.mon.Current()
	s := fmt.Sprintf("State: %s", state)
	if state == statemachine.StateRecording || state == statemachine.StateSilenceWait {
		s += fmt.Sprintf(" | File: %s", filepath.Base(e.currentFile))
		s += fmt.Sprintf(" | Duration: %s", time.Since(e.recordStart).Truncate(time.Second))
	}
	if state == statemachine.StateSilenceWait {
		s += fmt.Sprintf(" | Silence: %s", e.sm.SilenceElapsed().Truncate(time.Second))
	}
	if snap.ZoomRunning {
		s += " | Zoom"
	}
	if snap.TeamsRunning {
		s += " | Teams"
	}
	return s
}

func (e *Engine) loop() {
	bufSize := e.stream.FramesPerBuffer() * e.stream.Channels()
	buf := make([]float32, bufSize)
	for {
		select {
		case <-e.stopCh:
			return
		default:
		}
		if err := e.stream.Read(buf); err != nil {
			e.logger.Printf("Read error: %v", err)
			continue
		}
		rms := audio.RMS(buf)
		action := e.sm.ProcessAudio(rms, e.cfg.Audio.Threshold)
		switch action {
		case statemachine.ActionStartRecording:
			e.startRecording()
		case statemachine.ActionContinue:
			e.writeAudio(buf)
		case statemachine.ActionStopRecording:
			e.finalizeRecording()
			e.sm.Reset()
		}
	}
}

func (e *Engine) pollMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			snap := e.mon.Poll()
			e.mu.Lock()
			e.monSnapshot = snap
			e.mu.Unlock()
		}
	}
}

func (e *Engine) startRecording() {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()
	e.recordStart = now
	snap := e.monSnapshot
	micSuffix := "nomic"
	if snap.MicActive {
		micSuffix = "mic"
	}
	filename := fmt.Sprintf("%s_audio_%s.wav",
		now.Format("2006-01-02_150405"), micSuffix)
	path := filepath.Join(e.outputDir, filename)
	w, err := wav.Create(path, e.stream.SampleRate(), e.stream.Channels())
	if err != nil {
		e.logger.Printf("Failed to create WAV: %v", err)
		e.sm.Reset()
		return
	}
	e.writer = w
	e.currentFile = path
	e.logger.Printf("Recording started: %s", filename)
}

func (e *Engine) writeAudio(samples []float32) {
	e.mu.Lock()
	w := e.writer
	e.mu.Unlock()
	if w == nil {
		return
	}
	if err := w.Write(samples); err != nil {
		e.logger.Printf("Write error: %v", err)
	}
}

func (e *Engine) finalizeRecording() {
	e.mu.Lock()
	w := e.writer
	file := e.currentFile
	start := e.recordStart
	snap := e.monSnapshot
	e.writer = nil
	e.currentFile = ""
	e.mu.Unlock()
	if w == nil {
		return
	}
	if err := w.Close(); err != nil {
		e.logger.Printf("Close WAV error: %v", err)
	}
	meta := metadata.Recording{
		StartedAt:    start,
		EndedAt:      time.Now(),
		MicActive:    snap.MicActive,
		ZoomRunning:  snap.ZoomRunning,
		TeamsRunning: snap.TeamsRunning,
		Platform:     runtime.GOOS,
	}
	if err := metadata.Write(file, meta); err != nil {
		e.logger.Printf("Metadata error: %v", err)
	}
	dur := time.Since(start).Truncate(time.Second)
	e.logger.Printf("Finalized: %s (%s)", filepath.Base(file), dur)
}

func (e *Engine) findDevice() (*audio.DeviceInfo, error) {
	device := e.cfg.Audio.Device
	if device != "auto" && device != "" {
		dev := audio.FindDevice(device)
		if dev != nil {
			return dev, nil
		}
		return nil, fmt.Errorf("device %q not found", device)
	}
	var hint string
	switch runtime.GOOS {
	case "darwin":
		hint = e.cfg.Platform.MacOSDevice
	case "linux":
		hint = e.cfg.Platform.LinuxDevice
	}
	dev := audio.FindSystemAudioDevice(hint)
	if dev != nil {
		return dev, nil
	}
	dev, err := audio.DefaultInputDevice()
	if err != nil {
		return nil, fmt.Errorf("no audio device found: %w", err)
	}
	e.logger.Printf("Warning: using default device %q", dev.Name)
	return dev, nil
}
