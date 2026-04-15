// Package siglevel provides audio signal level analysis with hysteresis.
package siglevel

import (
	"math"
	"sync"
)

// Analyzer tracks audio signal levels and determines whether the signal
// is "active" (above threshold with hysteresis to avoid flickering).
type Analyzer struct {
	threshold   float64
	hysteresis  float64
	ratio       float64
	wasActive   bool
	history     []float64
	historyIdx  int
	historyFull bool
	historySize int
	peakLevel   float64
	mu          sync.Mutex
}

// NewAnalyzer creates a signal level analyzer.
//   - threshold: RMS level above which signal is considered active
//   - hysteresis: absolute hysteresis band (e.g. 0.005)
//   - ratio: hysteresis ratio relative to threshold (e.g. 0.6)
//   - windowSize: number of samples to keep in rolling history
func NewAnalyzer(threshold, hysteresis, ratio float64, windowSize int) *Analyzer {
	if windowSize < 1 {
		windowSize = 100
	}
	return &Analyzer{
		threshold:   threshold,
		hysteresis:  hysteresis,
		ratio:       ratio,
		history:     make([]float64, windowSize),
		historySize: windowSize,
	}
}

// RMS computes the root-mean-square of a float32 audio buffer.
func RMS(buf []float32) float64 {
	if len(buf) == 0 {
		return 0
	}
	var sum float64
	for _, s := range buf {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(buf)))
}

// Analyze computes the RMS level of buf and records it in history.
// Returns the RMS level.
func (a *Analyzer) Analyze(buf []float32) float64 {
	level := RMS(buf)

	a.mu.Lock()
	a.history[a.historyIdx] = level
	a.historyIdx++
	if a.historyIdx >= a.historySize {
		a.historyIdx = 0
		a.historyFull = true
	}
	if level > a.peakLevel {
		a.peakLevel = level
	}
	a.mu.Unlock()

	return level
}

// IsActive determines whether the given level is "active" using hysteresis.
// Once active, the level must drop below (threshold * ratio) to go inactive.
// Once inactive, the level must rise above threshold to go active.
func (a *Analyzer) IsActive(level float64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.wasActive {
		// Must drop below the lower hysteresis bound to deactivate
		lowerBound := a.threshold * a.ratio
		if a.hysteresis > 0 {
			lowerBound = a.threshold - a.hysteresis
		}
		if level < lowerBound {
			a.wasActive = false
		}
	} else {
		if level >= a.threshold {
			a.wasActive = true
		}
	}

	return a.wasActive
}

// AverageLevel returns the average level from the rolling history.
func (a *Analyzer) AverageLevel() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	count := a.historyIdx
	if a.historyFull {
		count = a.historySize
	}
	if count == 0 {
		return 0
	}

	var sum float64
	for i := 0; i < count; i++ {
		sum += a.history[i]
	}
	return sum / float64(count)
}

// PeakLevel returns the highest level seen since the last reset.
func (a *Analyzer) PeakLevel() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.peakLevel
}

// ResetStats clears the rolling history and peak level.
func (a *Analyzer) ResetStats() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.peakLevel = 0
	a.historyIdx = 0
	a.historyFull = false
	for i := range a.history {
		a.history[i] = 0
	}
}
