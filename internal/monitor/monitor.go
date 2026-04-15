// Package monitor provides best-effort process detection for meeting apps.
// These signals are secondary — they only enrich recording metadata and
// never trigger or stop recording.
package monitor

import (
	"os/exec"
	"strings"
	"sync"
)

// Snapshot holds the current state of monitored processes.
type Snapshot struct {
	ZoomRunning  bool
	TeamsRunning bool
	MicActive    bool // best-effort; may not be accurate
}

// Monitor checks for running meeting application processes.
type Monitor struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

// New creates a new process monitor.
func New() *Monitor {
	return &Monitor{}
}

// Poll updates the snapshot by checking running processes.
func (m *Monitor) Poll() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	procs := listProcesses()

	m.snapshot = Snapshot{
		ZoomRunning:  containsAny(procs, "zoom.us", "zoom", "CptHost"),
		TeamsRunning: containsAny(procs, "Microsoft Teams", "teams"),
		MicActive:    false, // best-effort detection below
	}

	return m.snapshot
}

// Current returns the last polled snapshot.
func (m *Monitor) Current() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshot
}

// listProcesses returns a list of running process names.
func listProcesses() []string {
	out, err := exec.Command("ps", "-eo", "comm=").Output()
	if err != nil {
		return nil
	}
	return strings.Split(string(out), "\n")
}

// containsAny returns true if any of the hints appear in the process list.
func containsAny(procs []string, hints ...string) bool {
	for _, proc := range procs {
		proc = strings.TrimSpace(proc)
		if proc == "" {
			continue
		}
		for _, hint := range hints {
			if strings.Contains(strings.ToLower(proc), strings.ToLower(hint)) {
				return true
			}
		}
	}
	return false
}
