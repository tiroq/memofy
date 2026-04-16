//go:build darwin

package micdetect

/*
#cgo LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

// Compatibility with older SDKs.
#ifndef kAudioObjectPropertyElementMain
#define kAudioObjectPropertyElementMain 0
#endif

// Audio process object properties (macOS 14+).
// Defined using raw FourCC values to avoid SDK version dependency.
#define kMD_ProcessObjectList      'prs#'
#define kMD_ProcessPID             'ppid'
#define kMD_ProcessBundleID        'pbid'
#define kMD_ProcessIsRunningInput  'piri'
#define kMD_ProcessIsRunningOutput 'piro'

// md_process_count returns the number of audio process objects, or -1 on error.
static int md_process_count(void) {
	AudioObjectPropertyAddress prop = {
		kMD_ProcessObjectList,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = 0;
	if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject,
			&prop, 0, NULL, &sz) != noErr)
		return -1;
	return (int)(sz / sizeof(AudioObjectID));
}

// md_get_process_list fills out with up to maxn process object IDs.
// Returns the number of objects written, or -1 on error.
static int md_get_process_list(AudioObjectID *out, int maxn) {
	AudioObjectPropertyAddress prop = {
		kMD_ProcessObjectList,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = (UInt32)(maxn * sizeof(AudioObjectID));
	if (AudioObjectGetPropertyData(kAudioObjectSystemObject,
			&prop, 0, NULL, &sz, out) != noErr)
		return -1;
	return (int)(sz / sizeof(AudioObjectID));
}

// md_get_pid reads the PID for a process object. Returns 0 on success, -1 on error.
static int md_get_pid(AudioObjectID obj, pid_t *pid) {
	AudioObjectPropertyAddress prop = {
		kMD_ProcessPID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = sizeof(pid_t);
	if (AudioObjectGetPropertyData(obj, &prop, 0, NULL, &sz, pid) != noErr)
		return -1;
	return 0;
}

// md_get_bundle_id reads the bundle identifier for a process object.
// Returns 0 on success, -1 on error.
static int md_get_bundle_id(AudioObjectID obj, char *buf, int bufLen) {
	AudioObjectPropertyAddress prop = {
		kMD_ProcessBundleID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	CFStringRef ref = NULL;
	UInt32 sz = sizeof(ref);
	if (AudioObjectGetPropertyData(obj, &prop, 0, NULL, &sz, &ref) != noErr)
		return -1;
	if (ref == NULL)
		return -1;
	Boolean ok = CFStringGetCString(ref, buf, (CFIndex)bufLen, kCFStringEncodingUTF8);
	CFRelease(ref);
	return ok ? 0 : -1;
}

// md_get_is_running_input reads whether the process has active audio input.
// Returns 0 on success, -1 on error.
static int md_get_is_running_input(AudioObjectID obj, UInt32 *running) {
	AudioObjectPropertyAddress prop = {
		kMD_ProcessIsRunningInput,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = sizeof(UInt32);
	if (AudioObjectGetPropertyData(obj, &prop, 0, NULL, &sz, running) != noErr)
		return -1;
	return 0;
}

// md_get_is_running_output reads whether the process has active audio output.
// Returns 0 on success, -1 on error.
static int md_get_is_running_output(AudioObjectID obj, UInt32 *running) {
	AudioObjectPropertyAddress prop = {
		kMD_ProcessIsRunningOutput,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = sizeof(UInt32);
	if (AudioObjectGetPropertyData(obj, &prop, 0, NULL, &sz, running) != noErr)
		return -1;
	return 0;
}
*/
import "C"

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"syscall"
)

// IsSupported returns true if the current system supports mic detection.
func IsSupported() bool {
	major, _, err := macOSVersion()
	if err != nil {
		return false
	}
	return major >= 14
}

// MacOSVersionString returns the macOS version as a string (e.g. "14.6").
func MacOSVersionString() string {
	ver, err := syscall.Sysctl("kern.osproductversion")
	if err != nil {
		return "unknown"
	}
	return ver
}

// ActiveMicUsers returns processes currently using microphone input.
// Returns ErrUnsupportedVersion on macOS < 14.
// An empty slice with nil error means no processes are using the microphone.
func ActiveMicUsers() ([]ActiveProcess, error) {
	major, _, err := macOSVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to determine macOS version: %w", err)
	}
	if major < 14 {
		return nil, ErrUnsupportedVersion
	}

	all, err := enumerateProcesses()
	if err != nil {
		return nil, err
	}
	return filterActiveInput(all), nil
}

// ActiveMicUserBundleIDs returns bundle identifiers of processes using microphone input.
func ActiveMicUserBundleIDs() ([]string, error) {
	procs, err := ActiveMicUsers()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(procs))
	for _, p := range procs {
		if p.BundleID != "" {
			ids = append(ids, p.BundleID)
		}
	}
	return ids, nil
}

// enumerateProcesses reads all audio process objects and their properties.
func enumerateProcesses() ([]ActiveProcess, error) {
	count := int(C.md_process_count())
	if count < 0 {
		return nil, fmt.Errorf("%w: could not get process list size", ErrEnumerationFailed)
	}
	if count == 0 {
		return nil, nil
	}

	ids := make([]C.AudioObjectID, count)
	n := int(C.md_get_process_list(&ids[0], C.int(count)))
	if n < 0 {
		return nil, fmt.Errorf("%w: could not read process list", ErrEnumerationFailed)
	}

	debugLog("mic detect: found %d audio process objects", n)

	var procs []ActiveProcess
	for i := 0; i < n; i++ {
		objID := ids[i]

		var pid C.pid_t
		if C.md_get_pid(objID, &pid) != 0 {
			debugLog("mic detect: skipping process object %d: PID read failed", objID)
			continue
		}

		bundleBuf := make([]C.char, 512)
		bundleID := ""
		if C.md_get_bundle_id(objID, &bundleBuf[0], 512) == 0 {
			bundleID = C.GoString(&bundleBuf[0])
		} else {
			debugLog("mic detect: pid=%d: bundle ID read failed", int(pid))
		}

		var runInput C.UInt32
		inputOK := C.md_get_is_running_input(objID, &runInput) == 0

		var runOutput C.UInt32
		outputOK := C.md_get_is_running_output(objID, &runOutput) == 0

		p := ActiveProcess{
			PID:      int(pid),
			BundleID: bundleID,
		}
		if inputOK {
			p.RunningInput = runInput != 0
			debugLog("mic detect: pid=%d bundle=%s input=%v", p.PID, p.BundleID, p.RunningInput)
		}
		if outputOK {
			p.RunningOutput = runOutput != 0
		}

		procs = append(procs, p)
	}

	return procs, nil
}

// macOSVersion parses the macOS version from sysctl.
func macOSVersion() (major, minor int, err error) {
	ver, err := syscall.Sysctl("kern.osproductversion")
	if err != nil {
		return 0, 0, fmt.Errorf("sysctl kern.osproductversion: %w", err)
	}
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("unexpected version format: %s", ver)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}
	return major, minor, nil
}

var debugLogger *log.Logger

// SetDebugLogger sets an optional logger for debug output.
func SetDebugLogger(l *log.Logger) {
	debugLogger = l
}

func debugLog(format string, args ...any) {
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}
