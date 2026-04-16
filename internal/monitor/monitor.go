// Package monitor provides best-effort process detection for meeting apps.
// These signals are secondary — they only enrich recording metadata and
// never trigger or stop recording.
package monitor

import (
	"os/exec"
	"strings"
	"sync"

	"github.com/tiroq/memofy/internal/micdetect"
)

// Snapshot holds the current state of monitored processes.
// All fields are best-effort.
type Snapshot struct {
	// ZoomRunning is true when the Zoom app process is running (open, not
	// necessarily in a call).
	ZoomRunning bool
	// ZoomInCall is true when Zoom's call subprocess (CptHost) is active,
	// which only exists during an active Zoom meeting.
	ZoomInCall bool
	// TeamsRunning is true when Microsoft Teams process is running (open,
	// not necessarily in a call).
	TeamsRunning bool
	// MeetRunning is true when a Google Meet tab may be open in a browser.
	MeetRunning bool
	// MicActive is a best-effort indicator that at least one known meeting
	// app is actively accessing the microphone.
	MicActive bool
	// MicBundleIDs contains bundle identifiers of processes actively
	// using microphone input (populated via Core Audio on macOS 14+).
	MicBundleIDs []string
}

// InCall returns true if any meeting app appears to be in an active call.
func (s Snapshot) InCall() bool {
	return s.ZoomInCall || s.MicActive
}

// meetingBundles are bundle IDs of known meeting applications.
var meetingBundles = []string{
	"com.microsoft.teams2",
	"com.microsoft.teams",
	"us.zoom.xos",
	"us.zoom.videomeeting",
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

	procs, pids := listProcessesWithPIDs()

	zoomOpen := containsAny(procs, "zoom.us", "zoom")
	zoomInCall := containsAny(procs, "CptHost") // spawned only during Zoom meetings
	teamsOpen := containsAny(procs, "Microsoft Teams", "teams")
	meetOpen := containsAny(procs, "Google Meet", "meet")

	var micActive bool
	var micBundleIDs []string

	// Prefer Core Audio mic detection (macOS 14+) over lsof.
	if micdetect.IsSupported() {
		if ids, err := micdetect.ActiveMicUserBundleIDs(); err == nil {
			micBundleIDs = ids
			micActive = len(ids) > 0
		}
	} else {
		// Fallback to lsof for older systems.
		var micPIDs []string
		if teamsOpen {
			micPIDs = append(micPIDs, getPIDs(pids, "teams", "microsoft teams")...)
		}
		if zoomInCall {
			micPIDs = append(micPIDs, getPIDs(pids, "cpthost")...)
		}
		if meetOpen {
			micPIDs = append(micPIDs, getPIDs(pids, "chrome", "safari", "firefox")...)
		}
		micActive = micInUseByPIDs(micPIDs)
	}

	m.snapshot = Snapshot{
		ZoomRunning:  zoomOpen || zoomInCall,
		ZoomInCall:   zoomInCall,
		TeamsRunning: teamsOpen,
		MeetRunning:  meetOpen,
		MicActive:    micActive,
		MicBundleIDs: micBundleIDs,
	}

	return m.snapshot
}

// Current returns the last polled snapshot.
func (m *Monitor) Current() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshot
}

// listProcessesWithPIDs returns (name list, pid→name map) for all running processes.
func listProcessesWithPIDs() ([]string, map[string]string) {
	out, err := exec.Command("ps", "-eo", "pid=,comm=").Output()
	if err != nil {
		return nil, nil
	}
	lines := strings.Split(string(out), "\n")
	names := make([]string, 0, len(lines))
	pids := make(map[string]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, " ", 2)
		if len(fields) == 2 {
			pid, name := strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])
			names = append(names, name)
			pids[pid] = name
		}
	}
	return names, pids
}

// getPIDs returns PIDs whose process name contains any of the given hints.
func getPIDs(pids map[string]string, hints ...string) []string {
	var out []string
	for pid, name := range pids {
		lower := strings.ToLower(name)
		for _, h := range hints {
			if strings.Contains(lower, strings.ToLower(h)) {
				out = append(out, pid)
				break
			}
		}
	}
	return out
}

// micInUseByPIDs uses lsof to check if any of the given PIDs have
// audio-related file descriptors open (best-effort on macOS/Linux).
func micInUseByPIDs(pids []string) bool {
	if len(pids) == 0 {
		return false
	}
	out, err := exec.Command("lsof", append([]string{"-b", "-n", "-p", strings.Join(pids, ",")}, []string{}...)...).Output()
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(out))
	return strings.Contains(lower, "audio") ||
		strings.Contains(lower, "coreaudio") ||
		strings.Contains(lower, "microphone") ||
		strings.Contains(lower, "audiotoolbox")
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
