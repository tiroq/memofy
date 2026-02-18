package ipc

import (
	"os"
	"path/filepath"
	"strings"
)

// Command represents user commands from UI to daemon
type Command string

const (
	CmdStart  Command = "start"  // Start recording immediately
	CmdStop   Command = "stop"   // Stop recording immediately
	CmdToggle Command = "toggle" // Toggle recording state
	CmdAuto   Command = "auto"   // Switch to auto mode
	CmdPause  Command = "pause"  // Switch to paused mode
	CmdManual Command = "manual" // Switch to manual mode (detect but never auto-control OBS)
	CmdQuit   Command = "quit"   // Shutdown daemon
)

// WriteCommand writes a command to ~/.cache/memofy/cmd.txt
func WriteCommand(cmd Command) error {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "memofy")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cmdPath := filepath.Join(cacheDir, "cmd.txt")
	return os.WriteFile(cmdPath, []byte(string(cmd)), 0644)
}

// ReadCommand reads and clears ~/.cache/memofy/cmd.txt
// Returns empty string if no command or file doesn't exist
func ReadCommand() (Command, error) {
	cmdPath := filepath.Join(os.Getenv("HOME"), ".cache", "memofy", "cmd.txt")

	// Read the command
	data, err := os.ReadFile(cmdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No command pending
		}
		return "", err
	}

	// Clear the file immediately to prevent re-execution
	if err := os.WriteFile(cmdPath, []byte(""), 0644); err != nil {
		return "", err
	}

	// Parse and validate command
	cmd := Command(strings.TrimSpace(string(data)))

	// Validate it's a known command
	switch cmd {
	case CmdStart, CmdStop, CmdToggle, CmdAuto, CmdPause, CmdManual, CmdQuit:
		return cmd, nil
	case "":
		return "", nil // Empty file
	default:
		// Invalid command - ignore it
		return "", nil
	}
}
