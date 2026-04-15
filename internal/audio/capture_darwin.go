//go:build darwin

// Package audio — macOS CoreAudio capture via AudioQueue Services.
// Replaces PortAudio on macOS for native system audio capture.
package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
#include <AudioToolbox/AudioToolbox.h>
#include <CoreAudio/CoreAudio.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>

#define CA_MAX_DEVICES 64
#define CA_NUM_BUFFERS 3

// ---- Device enumeration ----

typedef struct {
	UInt32 deviceID;
	char   name[256];
	int    maxInputChannels;
	double sampleRate;
} CADeviceInfo;

static int ca_listInputDevices(CADeviceInfo *out, int maxCount) {
	AudioObjectPropertyAddress prop = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		0 // kAudioObjectPropertyElementMain
	};
	UInt32 size = 0;
	if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &prop, 0, NULL, &size) != noErr)
		return 0;

	int count = size / sizeof(AudioDeviceID);
	AudioDeviceID *ids = (AudioDeviceID *)malloc(size);
	if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &prop, 0, NULL, &size, ids) != noErr) {
		free(ids);
		return 0;
	}

	int inputCount = 0;
	for (int i = 0; i < count && inputCount < maxCount; i++) {
		// Check input channel count
		AudioObjectPropertyAddress inputProp = {
			kAudioDevicePropertyStreamConfiguration,
			kAudioObjectPropertyScopeInput,
			0
		};
		UInt32 bufSize = 0;
		if (AudioObjectGetPropertyDataSize(ids[i], &inputProp, 0, NULL, &bufSize) != noErr)
			continue;

		AudioBufferList *bufList = (AudioBufferList *)malloc(bufSize);
		if (AudioObjectGetPropertyData(ids[i], &inputProp, 0, NULL, &bufSize, bufList) != noErr) {
			free(bufList);
			continue;
		}

		int ch = 0;
		for (UInt32 b = 0; b < bufList->mNumberBuffers; b++)
			ch += bufList->mBuffers[b].mNumberChannels;
		free(bufList);

		if (ch == 0)
			continue;

		out[inputCount].deviceID = ids[i];
		out[inputCount].maxInputChannels = ch;

		// Get device name
		AudioObjectPropertyAddress nameProp = {
			kAudioObjectPropertyName,
			kAudioObjectPropertyScopeGlobal,
			0
		};
		CFStringRef name = NULL;
		UInt32 nameSize = sizeof(CFStringRef);
		if (AudioObjectGetPropertyData(ids[i], &nameProp, 0, NULL, &nameSize, &name) == noErr && name) {
			CFStringGetCString(name, out[inputCount].name, 256, kCFStringEncodingUTF8);
			CFRelease(name);
		}

		// Get nominal sample rate
		AudioObjectPropertyAddress rateProp = {
			kAudioDevicePropertyNominalSampleRate,
			kAudioObjectPropertyScopeGlobal,
			0
		};
		Float64 rate = 44100.0;
		UInt32 rateSize = sizeof(Float64);
		AudioObjectGetPropertyData(ids[i], &rateProp, 0, NULL, &rateSize, &rate);
		out[inputCount].sampleRate = rate;

		inputCount++;
	}

	free(ids);
	return inputCount;
}

static UInt32 ca_getDefaultInputDevice(void) {
	AudioObjectPropertyAddress prop = {
		kAudioHardwarePropertyDefaultInputDevice,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	AudioDeviceID deviceID = 0;
	UInt32 size = sizeof(AudioDeviceID);
	if (AudioObjectGetPropertyData(kAudioObjectSystemObject, &prop, 0, NULL, &size, &deviceID) != noErr)
		return 0;
	return (UInt32)deviceID;
}

// ---- Audio capture via AudioQueue ----

typedef struct {
	AudioQueueRef       queue;
	AudioQueueBufferRef buffers[CA_NUM_BUFFERS];
	volatile int        running;
	float              *ringBuf;
	int                 ringBufSize; // in float samples
	int                 writePos;
	int                 readPos;
	int                 channels;
	int                 framesPerBuf;
	pthread_mutex_t     mutex;
	pthread_cond_t      cond;
} CACaptureState;

static void ca_inputCallback(
	void *userData,
	AudioQueueRef q,
	AudioQueueBufferRef buf,
	const AudioTimeStamp *startTime,
	UInt32 numPackets,
	const AudioStreamPacketDescription *desc
) {
	CACaptureState *s = (CACaptureState *)userData;
	if (!s->running) return;

	int floatCount = buf->mAudioDataByteSize / sizeof(float);
	float *data = (float *)buf->mAudioData;

	pthread_mutex_lock(&s->mutex);
	for (int i = 0; i < floatCount; i++) {
		s->ringBuf[s->writePos] = data[i];
		s->writePos = (s->writePos + 1) % s->ringBufSize;
	}
	pthread_cond_signal(&s->cond);
	pthread_mutex_unlock(&s->mutex);

	AudioQueueEnqueueBuffer(q, buf, 0, NULL);
}

static CACaptureState* ca_startCapture(UInt32 deviceID, int sampleRate, int channels, int framesPerBuffer) {
	CACaptureState *s = (CACaptureState *)calloc(1, sizeof(CACaptureState));
	s->channels     = channels;
	s->framesPerBuf = framesPerBuffer;
	s->ringBufSize  = framesPerBuffer * channels * 8; // 8x buffer for safety
	s->ringBuf      = (float *)calloc(s->ringBufSize, sizeof(float));
	pthread_mutex_init(&s->mutex, NULL);
	pthread_cond_init(&s->cond, NULL);

	AudioStreamBasicDescription fmt;
	memset(&fmt, 0, sizeof(fmt));
	fmt.mSampleRate       = (Float64)sampleRate;
	fmt.mFormatID         = kAudioFormatLinearPCM;
	fmt.mFormatFlags      = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
	fmt.mBitsPerChannel   = 32;
	fmt.mChannelsPerFrame = channels;
	fmt.mFramesPerPacket  = 1;
	fmt.mBytesPerFrame    = channels * sizeof(float);
	fmt.mBytesPerPacket   = fmt.mBytesPerFrame;

	OSStatus status = AudioQueueNewInput(
		&fmt, ca_inputCallback, s,
		NULL, kCFRunLoopCommonModes, 0, &s->queue);
	if (status != noErr) {
		free(s->ringBuf);
		free(s);
		return NULL;
	}

	// Bind to specific device
	UInt32 devSize = sizeof(AudioDeviceID);
	AudioQueueSetProperty(s->queue, kAudioQueueProperty_CurrentDevice, &deviceID, devSize);

	int bufBytes = framesPerBuffer * channels * sizeof(float);
	for (int i = 0; i < CA_NUM_BUFFERS; i++) {
		AudioQueueAllocateBuffer(s->queue, bufBytes, &s->buffers[i]);
		AudioQueueEnqueueBuffer(s->queue, s->buffers[i], 0, NULL);
	}

	s->running = 1;
	status = AudioQueueStart(s->queue, NULL);
	if (status != noErr) {
		AudioQueueDispose(s->queue, true);
		free(s->ringBuf);
		free(s);
		return NULL;
	}

	return s;
}

// ca_readCapture blocks until framesPerBuffer frames are available, then copies to out.
static int ca_readCapture(CACaptureState *s, float *out, int numFloats) {
	if (!s || !s->running) return -1;

	pthread_mutex_lock(&s->mutex);
	while (s->running) {
		int avail;
		if (s->writePos >= s->readPos)
			avail = s->writePos - s->readPos;
		else
			avail = s->ringBufSize - s->readPos + s->writePos;

		if (avail >= numFloats)
			break;

		// Wait with 1-second timeout to allow checking running flag
		struct timespec ts;
		clock_gettime(CLOCK_REALTIME, &ts);
		ts.tv_sec += 1;
		pthread_cond_timedwait(&s->cond, &s->mutex, &ts);
	}

	if (!s->running) {
		pthread_mutex_unlock(&s->mutex);
		return -1;
	}

	for (int i = 0; i < numFloats; i++) {
		out[i] = s->ringBuf[s->readPos];
		s->readPos = (s->readPos + 1) % s->ringBufSize;
	}
	pthread_mutex_unlock(&s->mutex);
	return numFloats;
}

static void ca_stopCapture(CACaptureState *s) {
	if (!s) return;
	s->running = 0;
	pthread_mutex_lock(&s->mutex);
	pthread_cond_signal(&s->cond);
	pthread_mutex_unlock(&s->mutex);
	AudioQueueStop(s->queue, true);
	AudioQueueDispose(s->queue, true);
	free(s->ringBuf);
	pthread_mutex_destroy(&s->mutex);
	pthread_cond_destroy(&s->cond);
	free(s);
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

// Init is a no-op on macOS (CoreAudio needs no global init).
func Init() error {
	return nil
}

// Terminate is a no-op on macOS.
func Terminate() {}

// ListInputDevices returns all available audio input devices via CoreAudio.
func ListInputDevices() []DeviceInfo {
	var cDevices [64]C.CADeviceInfo
	n := int(C.ca_listInputDevices(&cDevices[0], 64))

	devices := make([]DeviceInfo, 0, n)
	for i := 0; i < n; i++ {
		devices = append(devices, DeviceInfo{
			Index:      int(cDevices[i].deviceID),
			Name:       C.GoString(&cDevices[i].name[0]),
			MaxInputCh: int(cDevices[i].maxInputChannels),
			SampleRate: float64(cDevices[i].sampleRate),
		})
	}
	return devices
}

// FindDevice searches for an input device whose name contains the given substring.
func FindDevice(namePart string) *DeviceInfo {
	devices := ListInputDevices()
	for i := range devices {
		if strings.Contains(strings.ToLower(devices[i].Name), strings.ToLower(namePart)) {
			return &devices[i]
		}
	}
	return nil
}

// DefaultInputDevice returns the system default input device.
func DefaultInputDevice() (*DeviceInfo, error) {
	id := C.ca_getDefaultInputDevice()
	if id == 0 {
		return nil, fmt.Errorf("no default input device")
	}
	// Find it in the full list to get name and channel info
	devices := ListInputDevices()
	for i := range devices {
		if devices[i].Index == int(id) {
			return &devices[i], nil
		}
	}
	return nil, fmt.Errorf("default device (id=%d) not found in device list", id)
}

// Stream captures audio from a CoreAudio input device via AudioQueue.
type Stream struct {
	state      *C.CACaptureState
	deviceID   int
	sampleRate int
	channels   int
	bufSize    int
}

// OpenStream prepares a CoreAudio AudioQueue input stream for the given device.
// Call Start() to begin capture.
func OpenStream(cfg CaptureConfig) (*Stream, error) {
	return &Stream{
		deviceID:   cfg.DeviceIndex,
		sampleRate: cfg.SampleRate,
		channels:   cfg.Channels,
		bufSize:    cfg.FramesPerBuffer,
	}, nil
}

// Start begins audio capture.
func (s *Stream) Start() error {
	state := C.ca_startCapture(
		C.UInt32(s.deviceID),
		C.int(s.sampleRate),
		C.int(s.channels),
		C.int(s.bufSize),
	)
	if state == nil {
		return fmt.Errorf("failed to start CoreAudio capture")
	}
	s.state = state
	return nil
}

// Stop halts audio capture.
func (s *Stream) Stop() error {
	if s.state != nil {
		C.ca_stopCapture(s.state)
		s.state = nil
	}
	return nil
}

// Close releases stream resources.
func (s *Stream) Close() error {
	return s.Stop()
}

// Read fills the buffer with captured audio samples (float32 interleaved).
func (s *Stream) Read(buf []float32) error {
	if s.state == nil {
		return fmt.Errorf("stream not started")
	}
	n := C.ca_readCapture(s.state, (*C.float)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	if n < 0 {
		return fmt.Errorf("audio capture read error")
	}
	return nil
}

// FramesPerBuffer returns the configured frames per buffer.
func (s *Stream) FramesPerBuffer() int { return s.bufSize }

// Channels returns the number of channels.
func (s *Stream) Channels() int { return s.channels }

// SampleRate returns the sample rate.
func (s *Stream) SampleRate() int { return s.sampleRate }
