// Memofy is a lightweight automatic audio recorder that captures system sound
// when activity is detected. It uses silence-based splitting to create separate
// recording files per session.
//
// Usage:
//
//	memofy run          Start recording daemon
//	memofy status       Show current status
//	memofy doctor       Check system setup
//	memofy test-audio   Test audio capture
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/tiroq/memofy/internal/audio"
	"github.com/tiroq/memofy/internal/autoupdate"
	"github.com/tiroq/memofy/internal/config"
	"github.com/tiroq/memofy/internal/engine"
	"github.com/tiroq/memofy/internal/pidfile"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		cmdRun()
	case "status":
		cmdStatus()
	case "doctor":
		cmdDoctor()
	case "test-audio":
		cmdTestAudio()
	case "check-updates":
		cmdCheckUpdates()
	case "version", "--version", "-v":
		fmt.Printf("memofy %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`memofy - Lightweight automatic audio recorder

Usage:
  memofy <command>

Commands:
  run              Start the recording daemon
  status           Show current recording status
  doctor           Check system setup and dependencies
  test-audio       Test audio capture for 5 seconds
  check-updates    Check for new versions on GitHub
  version          Show version information

Options:
  -c, --config PATH   Path to config file (default: ~/.config/memofy/config.yaml)

Environment:
  MEMOFY_DEBUG_RECORDING=true   Enable debug logging`)
}

func loadConfig() config.Config {
	// Check for --config flag
	configPath := ""
	for i, arg := range os.Args {
		if (arg == "-c" || arg == "--config") && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	if configPath != "" {
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", configPath, err)
			os.Exit(1)
		}
		return cfg
	}

	return config.LoadOrDefault()
}

func cmdRun() {
	cfg := loadConfig()
	logger := log.New(os.Stderr, "[memofy] ", log.LstdFlags)

	logger.Printf("Memofy %s starting (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)

	// Single instance enforcement
	pf, err := pidfile.New(pidfile.GetPIDFilePath("memofy"))
	if err != nil {
		logger.Fatalf("Cannot start: %v", err)
	}
	defer pf.Remove()

	eng := engine.New(cfg, logger)
	eng.SetVersion(Version)

	if err := eng.Start(); err != nil {
		logger.Fatalf("Start failed: %v", err)
	}

	// Platform-specific run loop: macOS starts menu bar UI, Linux waits for signal.
	platformRunLoop(eng, cfg, Version, logger)
}

func cmdStatus() {
	cfg := loadConfig()
	logger := log.New(os.Stderr, "", 0)

	fmt.Printf("Platform:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Format:       %s\n", cfg.Audio.FormatProfile)
	fmt.Printf("Output Dir:   %s\n", cfg.Output.Dir)
	fmt.Printf("Threshold:    %.4f\n", cfg.Audio.Threshold)
	fmt.Printf("Silence:      %ds\n", cfg.Audio.SilenceSeconds)

	eng := engine.New(cfg, logger)
	fmt.Println(eng.Status())
}

func cmdCheckUpdates() {
	checker := autoupdate.NewUpdateChecker("tiroq", "memofy", Version, "")
	checker.SetChannel(autoupdate.ChannelStable)

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	available, release, err := checker.IsUpdateAvailable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Update check failed: %v\n", err)
		os.Exit(1)
	}

	if !available {
		fmt.Println("You are up to date.")
		return
	}

	fmt.Printf("\nNew version available: %s\n", release.TagName)
	fmt.Printf("Release page: https://github.com/tiroq/memofy/releases/tag/%s\n", release.TagName)
	if release.Body != "" {
		fmt.Printf("\nRelease notes:\n%s\n", release.Body)
	}
}

func cmdDoctor() {
	fmt.Printf("Memofy Doctor (%s/%s)\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Version: %s\n\n", Version)

	ok := true

	// Check audio backend
	fmt.Print("Audio: ")
	if err := audio.Init(); err != nil {
		fmt.Printf("FAIL - %v\n", err)
		ok = false
	} else {
		fmt.Println("OK")
		defer audio.Terminate()
	}

	// List audio devices
	devices := audio.ListInputDevices()
	fmt.Printf("Input devices: %d found\n", len(devices))
	for _, d := range devices {
		fmt.Printf("  [%d] %s (%d ch, %.0f Hz)\n", d.Index, d.Name, d.MaxInputCh, d.SampleRate)
	}

	// Check for system audio device
	cfg := loadConfig()
	fmt.Print("\nSystem audio device: ")
	switch runtime.GOOS {
	case "darwin":
		hint := cfg.Platform.MacOSDevice
		dev := audio.FindSystemAudioDevice(hint)
		if dev != nil {
			fmt.Printf("OK - %s\n", dev.Name)
		} else {
			fmt.Printf("NOT FOUND - Install BlackHole (https://existential.audio/blackhole/)\n")
			ok = false
		}
	case "linux":
		hint := cfg.Platform.LinuxDevice
		dev := audio.FindSystemAudioDevice(hint)
		if dev != nil {
			fmt.Printf("OK - %s\n", dev.Name)
		} else {
			fmt.Println("NOT FOUND - Check PulseAudio/PipeWire configuration")
			ok = false
		}
	default:
		fmt.Printf("UNSUPPORTED PLATFORM - %s\n", runtime.GOOS)
		ok = false
	}

	// Check output directory
	fmt.Printf("\nOutput directory: %s\n", cfg.Output.Dir)
	if err := os.MkdirAll(cfg.Output.Dir, 0755); err != nil {
		fmt.Printf("  FAIL - cannot create: %v\n", err)
		ok = false
	} else {
		fmt.Println("  OK")
	}

	// Check config
	fmt.Printf("\nConfig file: %s\n", config.DefaultConfigPath())
	if _, err := os.Stat(config.DefaultConfigPath()); err == nil {
		fmt.Println("  Found")
	} else {
		fmt.Println("  Not found (using defaults)")
	}

	// Check recorder backend (conversion tool)
	fmt.Print("\nRecorder backend: ")
	if audio.CanConvertToM4A() {
		fmt.Println("OK - M4A conversion available")
	} else {
		switch runtime.GOOS {
		case "darwin":
			fmt.Println("WARNING - afconvert not found (M4A output may fail)")
		case "linux":
			fmt.Println("WARNING - ffmpeg not found (M4A output may fail, install ffmpeg)")
		}
	}

	// Format profile
	fmt.Printf("Format profile: %s\n", cfg.Audio.FormatProfile)

	fmt.Println()
	if ok {
		fmt.Println("All checks passed. Ready to record.")
	} else {
		fmt.Println("Some checks failed. Fix the issues above before running.")
		os.Exit(1)
	}
}

func cmdTestAudio() {
	cfg := loadConfig()

	fmt.Println("Testing audio capture for 5 seconds...")
	fmt.Println()

	if err := audio.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Audio init failed: %v\n", err)
		os.Exit(1)
	}
	defer audio.Terminate()

	// Find device
	var dev *audio.DeviceInfo
	switch runtime.GOOS {
	case "darwin":
		dev = audio.FindSystemAudioDevice(cfg.Platform.MacOSDevice)
	case "linux":
		dev = audio.FindSystemAudioDevice(cfg.Platform.LinuxDevice)
	}
	if dev == nil {
		var err error
		dev, err = audio.DefaultInputDevice()
		if err != nil {
			fmt.Fprintf(os.Stderr, "No audio device found: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Warning: Using default device (may not capture system audio)\n")
	}

	fmt.Printf("Device: %s\n", dev.Name)

	channels := cfg.Audio.Channels
	if channels > dev.MaxInputCh {
		channels = dev.MaxInputCh
	}

	stream, err := audio.OpenStream(audio.CaptureConfig{
		DeviceIndex:     dev.Index,
		SampleRate:      cfg.Audio.SampleRate,
		Channels:        channels,
		FramesPerBuffer: 4096,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Open stream failed: %v\n", err)
		os.Exit(1)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Start stream failed: %v\n", err)
		os.Exit(1)
	}
	defer stream.Stop()

	buf := make([]float32, 4096*channels)
	threshold := cfg.Audio.Threshold
	end := time.After(5 * time.Second)

	fmt.Printf("Threshold: %.4f\n\n", threshold)
	fmt.Println("Level    | Status")
	fmt.Println("---------+---------")

	for {
		select {
		case <-end:
			fmt.Println("\nTest complete.")
			return
		default:
		}

		if err := stream.Read(buf); err != nil {
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
			continue
		}

		rms := audio.RMS(buf)
		bar := renderBar(rms, 40)
		status := "silence"
		if rms >= threshold {
			status = "SOUND"
		}
		fmt.Printf("\r%.6f %s %s   ", rms, bar, status)
	}
}

func renderBar(level float64, width int) string {
	filled := int(level * float64(width) * 10) // scale up for visibility
	if filled > width {
		filled = width
	}
	bar := make([]byte, width)
	for i := range bar {
		if i < filled {
			bar[i] = '#'
		} else {
			bar[i] = '-'
		}
	}
	return string(bar)
}
