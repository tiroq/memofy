//go:build darwin

// Native CoreAudio backend for macOS. Uses AudioUnit (AUHAL) for capture
// and the Audio HAL API for device enumeration. No external dependencies
// — only Apple system frameworks.
package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <AudioToolbox/AudioToolbox.h>
#include <CoreFoundation/CoreFoundation.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>

// Compatibility with older SDKs.
#ifndef kAudioObjectPropertyElementMain
#define kAudioObjectPropertyElementMain 0
#endif

// ---- ring-buffered capture stream ----

typedef struct {
	AudioComponentInstance unit;
	float    *ringBuf;
	int       ringCapacity;   // total float slots
	int64_t   writePos;
	int64_t   readPos;
	int       channels;
	int       framesPerBuffer;
	int       sampleRate;
	float    *renderBuf;      // pre-allocated for the callback
	int       renderBufBytes;
	pthread_mutex_t mutex;
	pthread_cond_t  cond;
	int       running;
} CAStream;

// ---- device enumeration helpers ----

static int ca_device_count(void) {
	AudioObjectPropertyAddress prop = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = 0;
	if (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject,
			&prop, 0, NULL, &sz) != noErr)
		return 0;
	return (int)(sz / sizeof(AudioDeviceID));
}

static int ca_all_devices(AudioDeviceID *out, int maxn) {
	AudioObjectPropertyAddress prop = {
		kAudioHardwarePropertyDevices,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = (UInt32)(maxn * sizeof(AudioDeviceID));
	if (AudioObjectGetPropertyData(kAudioObjectSystemObject,
			&prop, 0, NULL, &sz, out) != noErr)
		return 0;
	return (int)(sz / sizeof(AudioDeviceID));
}

static int ca_device_name(AudioDeviceID d, char *buf, int len) {
	AudioObjectPropertyAddress prop = {
		kAudioObjectPropertyName,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	CFStringRef ref = NULL;
	UInt32 sz = sizeof(ref);
	if (AudioObjectGetPropertyData(d, &prop, 0, NULL, &sz, &ref) != noErr)
		return -1;
	Boolean ok = CFStringGetCString(ref, buf, (CFIndex)len, kCFStringEncodingUTF8);
	CFRelease(ref);
	return ok ? 0 : -1;
}

static int ca_input_channels(AudioDeviceID d) {
	AudioObjectPropertyAddress prop = {
		kAudioDevicePropertyStreamConfiguration,
		kAudioObjectPropertyScopeInput,
		kAudioObjectPropertyElementMain,
	};
	UInt32 sz = 0;
	if (AudioObjectGetPropertyDataSize(d, &prop, 0, NULL, &sz) != noErr || sz == 0)
		return 0;
	AudioBufferList *bl = (AudioBufferList *)malloc(sz);
	if (AudioObjectGetPropertyData(d, &prop, 0, NULL, &sz, bl) != noErr) {
		free(bl);
		return 0;
	}
	int ch = 0;
	for (UInt32 i = 0; i < bl->mNumberBuffers; i++)
		ch += (int)bl->mBuffers[i].mNumberChannels;
	free(bl);
	return ch;
}

static double ca_nominal_rate(AudioDeviceID d) {
	AudioObjectPropertyAddress prop = {
		kAudioDevicePropertyNominalSampleRate,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	Float64 r = 0;
	UInt32 sz = sizeof(r);
	AudioObjectGetPropertyData(d, &prop, 0, NULL, &sz, &r);
	return (double)r;
}

static AudioDeviceID ca_default_input(void) {
	AudioObjectPropertyAddress prop = {
		kAudioHardwarePropertyDefaultInputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	AudioDeviceID d = kAudioObjectUnknown;
	UInt32 sz = sizeof(d);
	AudioObjectGetPropertyData(kAudioObjectSystemObject,
		&prop, 0, NULL, &sz, &d);
	return d;
}

// ---- AUHAL capture callback ----

static OSStatus ca_render_cb(
	void                        *ref,
	AudioUnitRenderActionFlags  *flags,
	const AudioTimeStamp        *ts,
	UInt32                       bus,
	UInt32                       nframes,
	AudioBufferList             *ioData)
{
	CAStream *s = (CAStream *)ref;
	int bytes = (int)(nframes * (UInt32)s->channels * sizeof(float));
	if (bytes > s->renderBufBytes)
		return noErr; // unexpected oversize callback — skip safely

	AudioBufferList abl;
	abl.mNumberBuffers = 1;
	abl.mBuffers[0].mNumberChannels = (UInt32)s->channels;
	abl.mBuffers[0].mDataByteSize   = (UInt32)bytes;
	abl.mBuffers[0].mData           = s->renderBuf;

	OSStatus err = AudioUnitRender(s->unit, flags, ts, 1, nframes, &abl);
	if (err != noErr)
		return err;

	int n = (int)nframes * s->channels;
	pthread_mutex_lock(&s->mutex);
	for (int i = 0; i < n; i++) {
		s->ringBuf[(int)(s->writePos % (int64_t)s->ringCapacity)] = s->renderBuf[i];
		s->writePos++;
	}
	if (s->writePos - s->readPos > (int64_t)s->ringCapacity)
		s->readPos = s->writePos - (int64_t)s->ringCapacity;
	pthread_cond_signal(&s->cond);
	pthread_mutex_unlock(&s->mutex);
	return noErr;
}

// ---- stream lifecycle ----

static CAStream *ca_stream_open(AudioDeviceID dev, int rate,
								int ch, int fpb)
{
	CAStream *s = (CAStream *)calloc(1, sizeof(CAStream));
	if (!s) return NULL;
	s->channels        = ch;
	s->framesPerBuffer = fpb;
	s->sampleRate      = rate;
	s->ringCapacity    = rate * ch * 4; // 4 s buffer
	s->ringBuf         = (float *)calloc((size_t)s->ringCapacity, sizeof(float));
	s->renderBufBytes  = fpb * ch * 2 * (int)sizeof(float);
	s->renderBuf       = (float *)malloc((size_t)s->renderBufBytes);
	pthread_mutex_init(&s->mutex, NULL);
	pthread_cond_init(&s->cond, NULL);

	// Locate HAL Output component
	AudioComponentDescription desc = {
		.componentType         = kAudioUnitType_Output,
		.componentSubType      = kAudioUnitSubType_HALOutput,
		.componentManufacturer = kAudioUnitManufacturer_Apple,
	};
	AudioComponent comp = AudioComponentFindNext(NULL, &desc);
	if (!comp) goto fail;
	if (AudioComponentInstanceNew(comp, &s->unit) != noErr) goto fail;

	// Enable input (bus 1), disable output (bus 0)
	UInt32 flag = 1;
	AudioUnitSetProperty(s->unit, kAudioOutputUnitProperty_EnableIO,
		kAudioUnitScope_Input, 1, &flag, sizeof(flag));
	flag = 0;
	AudioUnitSetProperty(s->unit, kAudioOutputUnitProperty_EnableIO,
		kAudioUnitScope_Output, 0, &flag, sizeof(flag));

	// Assign device
	AudioUnitSetProperty(s->unit, kAudioOutputUnitProperty_CurrentDevice,
		kAudioUnitScope_Global, 0, &dev, sizeof(dev));

	// Desired format: interleaved float32
	AudioStreamBasicDescription fmt;
	memset(&fmt, 0, sizeof(fmt));
	fmt.mSampleRate       = (Float64)rate;
	fmt.mFormatID         = kAudioFormatLinearPCM;
	fmt.mFormatFlags      = kAudioFormatFlagIsFloat | kAudioFormatFlagIsPacked;
	fmt.mBytesPerPacket   = (UInt32)(ch * sizeof(float));
	fmt.mFramesPerPacket  = 1;
	fmt.mBytesPerFrame    = (UInt32)(ch * sizeof(float));
	fmt.mChannelsPerFrame = (UInt32)ch;
	fmt.mBitsPerChannel   = 32;
	AudioUnitSetProperty(s->unit, kAudioUnitProperty_StreamFormat,
		kAudioUnitScope_Output, 1, &fmt, sizeof(fmt));

	// Buffer size
	UInt32 buf = (UInt32)fpb;
	AudioUnitSetProperty(s->unit, kAudioDevicePropertyBufferFrameSize,
		kAudioUnitScope_Global, 0, &buf, sizeof(buf));

	// Render callback
	AURenderCallbackStruct cb = { .inputProc = ca_render_cb, .inputProcRefCon = s };
	AudioUnitSetProperty(s->unit, kAudioOutputUnitProperty_SetInputCallback,
		kAudioUnitScope_Global, 0, &cb, sizeof(cb));

	if (AudioUnitInitialize(s->unit) != noErr) {
		AudioComponentInstanceDispose(s->unit);
		goto fail;
	}
	return s;

fail:
	free(s->ringBuf);
	free(s->renderBuf);
	pthread_mutex_destroy(&s->mutex);
	pthread_cond_destroy(&s->cond);
	free(s);
	return NULL;
}

static int ca_stream_start(CAStream *s) {
	pthread_mutex_lock(&s->mutex);
	s->running = 1;
	pthread_mutex_unlock(&s->mutex);
	return (AudioOutputUnitStart(s->unit) == noErr) ? 0 : -1;
}

static int ca_stream_read(CAStream *s, float *out, int frames) {
	int need = frames * s->channels;
	pthread_mutex_lock(&s->mutex);
	while (s->running) {
		if (s->writePos - s->readPos >= (int64_t)need) break;
		pthread_cond_wait(&s->cond, &s->mutex);
	}
	if (!s->running) {
		pthread_mutex_unlock(&s->mutex);
		return -1;
	}
	for (int i = 0; i < need; i++) {
		out[i] = s->ringBuf[(int)(s->readPos % (int64_t)s->ringCapacity)];
		s->readPos++;
	}
	pthread_mutex_unlock(&s->mutex);
	return 0;
}

static void ca_stream_stop(CAStream *s) {
	AudioOutputUnitStop(s->unit);
	pthread_mutex_lock(&s->mutex);
	s->running = 0;
	pthread_cond_signal(&s->cond);
	pthread_mutex_unlock(&s->mutex);
}

static void ca_stream_close(CAStream *s) {
	if (!s) return;
	if (s->running) ca_stream_stop(s);
	AudioUnitUninitialize(s->unit);
	AudioComponentInstanceDispose(s->unit);
	pthread_mutex_destroy(&s->mutex);
	pthread_cond_destroy(&s->cond);
	free(s->ringBuf);
	free(s->renderBuf);
	free(s);
}
*/
import "C"

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

// Init initializes the audio backend. No global state needed for CoreAudio.
func Init() error { return nil }

// Terminate releases audio resources. No-op for CoreAudio.
func Terminate() {}

// ListInputDevices returns all available audio input devices.
func ListInputDevices() []DeviceInfo {
	count := int(C.ca_device_count())
	if count == 0 {
		return nil
	}
	ids := make([]C.AudioDeviceID, count)
	n := int(C.ca_all_devices(&ids[0], C.int(count)))

	var out []DeviceInfo
	var nameBuf [256]C.char
	for i := 0; i < n; i++ {
		ch := int(C.ca_input_channels(ids[i]))
		if ch == 0 {
			continue
		}
		if C.ca_device_name(ids[i], &nameBuf[0], 256) != 0 {
			continue
		}
		out = append(out, DeviceInfo{
			Index:      int(ids[i]),
			Name:       C.GoString(&nameBuf[0]),
			MaxInputCh: ch,
			SampleRate: float64(C.ca_nominal_rate(ids[i])),
		})
	}
	return out
}

// FindDevice searches for an input device whose name contains the given substring.
func FindDevice(namePart string) *DeviceInfo {
	lower := strings.ToLower(namePart)
	for _, d := range ListInputDevices() {
		if strings.Contains(strings.ToLower(d.Name), lower) {
			return &d
		}
	}
	return nil
}

// DefaultInputDevice returns the system default input device.
func DefaultInputDevice() (*DeviceInfo, error) {
	id := C.ca_default_input()
	if id == C.kAudioObjectUnknown {
		return nil, fmt.Errorf("no default input device")
	}
	var nameBuf [256]C.char
	if C.ca_device_name(id, &nameBuf[0], 256) != 0 {
		return nil, fmt.Errorf("could not get device name")
	}
	return &DeviceInfo{
		Index:      int(id),
		Name:       C.GoString(&nameBuf[0]),
		MaxInputCh: int(C.ca_input_channels(id)),
		SampleRate: float64(C.ca_nominal_rate(id)),
	}, nil
}

// Stream captures audio from a CoreAudio input device via AUHAL AudioUnit.
type Stream struct {
	mu         sync.Mutex // protects s for concurrent Stop()/Close() calls
	s          *C.CAStream
	sampleRate int
	channels   int
	bufSize    int
}

// OpenStream opens a CoreAudio input stream for the given device.
func OpenStream(cfg CaptureConfig) (*Stream, error) {
	cs := C.ca_stream_open(
		C.AudioDeviceID(cfg.DeviceIndex),
		C.int(cfg.SampleRate),
		C.int(cfg.Channels),
		C.int(cfg.FramesPerBuffer),
	)
	if cs == nil {
		return nil, fmt.Errorf("failed to open CoreAudio stream for device %d", cfg.DeviceIndex)
	}
	return &Stream{
		s:          cs,
		sampleRate: cfg.SampleRate,
		channels:   cfg.Channels,
		bufSize:    cfg.FramesPerBuffer,
	}, nil
}

// Start begins audio capture.
func (st *Stream) Start() error {
	if C.ca_stream_start(st.s) != 0 {
		return fmt.Errorf("failed to start CoreAudio stream")
	}
	return nil
}

// Read fills buf with interleaved float32 audio samples.
func (st *Stream) Read(buf []float32) error {
	frames := len(buf) / st.channels
	ret := C.ca_stream_read(st.s, (*C.float)(unsafe.Pointer(&buf[0])), C.int(frames))
	if ret != 0 {
		return fmt.Errorf("CoreAudio stream stopped")
	}
	return nil
}

// Stop halts audio capture. Safe to call concurrently with Close().
func (st *Stream) Stop() error {
	st.mu.Lock()
	s := st.s
	st.mu.Unlock()
	if s != nil {
		C.ca_stream_stop(s)
	}
	return nil
}

// Close releases all resources. Safe to call concurrently with Stop().
func (st *Stream) Close() error {
	st.mu.Lock()
	s := st.s
	st.s = nil
	st.mu.Unlock()
	if s != nil {
		C.ca_stream_close(s)
	}
	return nil
}

// FramesPerBuffer returns the buffer size in frames.
func (st *Stream) FramesPerBuffer() int { return st.bufSize }

// Channels returns the number of channels.
func (st *Stream) Channels() int { return st.channels }

// SampleRate returns the stream's sample rate.
func (st *Stream) SampleRate() int { return st.sampleRate }
