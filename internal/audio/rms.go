package audio

import "math"

// RMS calculates the root mean square of interleaved float32 audio samples.
// Returns 0 for empty input.
func RMS(samples []float32) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}
