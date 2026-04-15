//go:build darwin || linux

package audio

/*
#include <portaudio.h>
#include <string.h>
*/
import "C"

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

var initOnce sync.Once

// Init initializes PortAudio. Must be called before any other audio functions.
func Init() error {
	var initErr error
	initOnce.Do(func() {
		if err := C.Pa_Initialize(); err != C.paNoError {
			initErr = fmt.Errorf("portaudio init: %s", C.GoString(C.Pa_GetErrorText(err)))
		}
	})
	return initErr
}

// Terminate releases PortAudio resources. Call once at program exit.
func Terminate() {
	C.Pa_Terminate()
}

// ListInputDevices returns all available audio input devices.
func ListInputDevices() []DeviceInfo {
	n := int(C.Pa_GetDeviceCount())
	var devices []DeviceInfo
	for i := 0; i < n; i++ {
		info := C.Pa_GetDeviceInfo(C.PaDeviceIndex(i))
		if info == nil || int(info.maxInputChannels) == 0 {
			continue
		}
		devices = append(devices, DeviceInfo{
			Index:      i,
			Name:       C.GoString(info.name),
			MaxInputCh: int(info.maxInputChannels),
			SampleRate: float64(info.defaultSampleRate),
		})
	}
	return devices
}

// FindDevice searches for an input device whose name contains the given substring.
// Returns nil if not found.
func FindDevice(namePart string) *DeviceInfo {
	n := int(C.Pa_GetDeviceCount())
	for i := 0; i < n; i++ {
		info := C.Pa_GetDeviceInfo(C.PaDeviceIndex(i))
		if info == nil || int(info.maxInputChannels) == 0 {
			continue
		}
		name := C.GoString(info.name)
		if strings.Contains(strings.ToLower(name), strings.ToLower(namePart)) {
			return &DeviceInfo{
				Index:      i,
				Name:       name,
				MaxInputCh: int(info.maxInputChannels),
				SampleRate: float64(info.defaultSampleRate),
			}
		}
	}
	return nil
}

// DefaultInputDevice returns the system default input device.
func DefaultInputDevice() (*DeviceInfo, error) {
	idx := C.Pa_GetDefaultInputDevice()
	if idx == C.paNoDevice {
		return nil, fmt.Errorf("no default input device")
	}
	info := C.Pa_GetDeviceInfo(idx)
	if info == nil {
		return nil, fmt.Errorf("could not get default device info")
	}
	return &DeviceInfo{
		Index:      int(idx),
		Name:       C.GoString(info.name),
		MaxInputCh: int(info.maxInputChannels),
		SampleRate: float64(info.defaultSampleRate),
	}, nil
}

// Stream captures audio from a PortAudio input device.
type Stream struct {
	stream     unsafe.Pointer // *C.PaStream (opaque)
	sampleRate int
	channels   int
	bufSize    int
	mu         sync.Mutex
	running    bool
}

// OpenStream opens a PortAudio input stream for the given device.
func OpenStream(cfg CaptureConfig) (*Stream, error) {
	devInfo := C.Pa_GetDeviceInfo(C.PaDeviceIndex(cfg.DeviceIndex))
	if devInfo == nil {
		return nil, fmt.Errorf("invalid device index %d", cfg.DeviceIndex)
	}

	inputParams := C.PaStreamParameters{
		device:                    C.PaDeviceIndex(cfg.DeviceIndex),
		channelCount:              C.int(cfg.Channels),
		sampleFormat:              C.paFloat32,
		suggestedLatency:          devInfo.defaultLowInputLatency,
		hostApiSpecificStreamInfo: nil,
	}

	var stream unsafe.Pointer
	err := C.Pa_OpenStream(
		(*unsafe.Pointer)(unsafe.Pointer(&stream)),
		&inputParams,
		nil, // no output
		C.double(cfg.SampleRate),
		C.ulong(cfg.FramesPerBuffer),
		C.paClipOff,
		nil, nil, // blocking mode
	)
	if err != C.paNoError {
		return nil, fmt.Errorf("open stream: %s", C.GoString(C.Pa_GetErrorText(err)))
	}

	return &Stream{
		stream:     stream,
		sampleRate: cfg.SampleRate,
		channels:   cfg.Channels,
		bufSize:    cfg.FramesPerBuffer,
	}, nil
}

// Start begins audio capture.
func (s *Stream) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	err := C.Pa_StartStream(s.stream)
	if err != C.paNoError {
		return fmt.Errorf("start stream: %s", C.GoString(C.Pa_GetErrorText(err)))
	}
	s.running = true
	return nil
}

// Read fills buf with interleaved float32 audio samples.
// buf must have length >= framesPerBuffer * channels.
func (s *Stream) Read(buf []float32) error {
	frames := len(buf) / s.channels
	err := C.Pa_ReadStream(s.stream, unsafe.Pointer(&buf[0]), C.ulong(frames))
	if err != C.paNoError {
		return fmt.Errorf("read stream: %s", C.GoString(C.Pa_GetErrorText(err)))
	}
	return nil
}

// Stop halts audio capture.
func (s *Stream) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	err := C.Pa_StopStream(s.stream)
	s.running = false
	if err != C.paNoError {
		return fmt.Errorf("stop stream: %s", C.GoString(C.Pa_GetErrorText(err)))
	}
	return nil
}

// Close closes the stream and releases resources.
func (s *Stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stream == nil {
		return nil
	}
	if s.running {
		C.Pa_StopStream(s.stream)
		s.running = false
	}
	err := C.Pa_CloseStream(s.stream)
	s.stream = nil
	if err != C.paNoError {
		return fmt.Errorf("close stream: %s", C.GoString(C.Pa_GetErrorText(err)))
	}
	return nil
}

// FramesPerBuffer returns the buffer size in frames.
func (s *Stream) FramesPerBuffer() int { return s.bufSize }

// Channels returns the number of channels.
func (s *Stream) Channels() int { return s.channels }

// SampleRate returns the stream's sample rate.
func (s *Stream) SampleRate() int { return s.sampleRate }

// IsRunning returns true if the stream is active.
func (s *Stream) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
