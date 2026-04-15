package audio

import (
	"math"
	"testing"
)

func TestRMS_Silence(t *testing.T) {
	samples := make([]float32, 1024)
	if got := RMS(samples); got != 0 {
		t.Errorf("RMS of silence: got %f, want 0", got)
	}
}

func TestRMS_Empty(t *testing.T) {
	if got := RMS(nil); got != 0 {
		t.Errorf("RMS of nil: got %f, want 0", got)
	}
}

func TestRMS_FullScale(t *testing.T) {
	samples := make([]float32, 1000)
	for i := range samples {
		samples[i] = 1.0
	}
	got := RMS(samples)
	if math.Abs(got-1.0) > 0.001 {
		t.Errorf("RMS of full-scale: got %f, want 1.0", got)
	}
}

func TestRMS_KnownSignal(t *testing.T) {
	// Sine wave at half amplitude: RMS = 0.5 / sqrt(2) ≈ 0.3535
	samples := make([]float32, 44100)
	for i := range samples {
		samples[i] = 0.5 * float32(math.Sin(2*math.Pi*440*float64(i)/44100))
	}
	got := RMS(samples)
	want := 0.5 / math.Sqrt(2)
	if math.Abs(got-want) > 0.01 {
		t.Errorf("RMS of sine: got %f, want ~%f", got, want)
	}
}
