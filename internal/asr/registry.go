package asr

import (
	"fmt"
	"sync"
)

// T026: Backend registry with primary/fallback support. FR-013

// Registry manages ASR backends and supports fallback transcription.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]Backend
	primary  string
	fallback string
}

// NewRegistry creates an empty backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]Backend),
	}
}

// Register adds a backend to the registry. The first registered backend
// becomes the primary by default.
func (r *Registry) Register(name string, b Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[name] = b
	if r.primary == "" {
		r.primary = name
	}
}

// SetPrimary sets the primary backend by name.
func (r *Registry) SetPrimary(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.primary = name
}

// SetFallback sets the fallback backend by name.
func (r *Registry) SetFallback(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = name
}

// Get returns a backend by name, or false if not found.
func (r *Registry) Get(name string) (Backend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.backends[name]
	return b, ok
}

// Primary returns the primary backend, or nil if none configured.
func (r *Registry) Primary() Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.backends[r.primary]
}

// Fallback returns the fallback backend, or nil if none configured.
func (r *Registry) Fallback() Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.fallback == "" {
		return nil
	}
	return r.backends[r.fallback]
}

// Backends returns the names of all registered backends.
func (r *Registry) Backends() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.backends))
	for name := range r.backends {
		names = append(names, name)
	}
	return names
}

// TranscribeWithFallback tries the primary backend first, falling back on error.
func (r *Registry) TranscribeWithFallback(filePath string, opts TranscribeOptions) (*Transcript, error) {
	primary := r.Primary()
	if primary == nil {
		return nil, fmt.Errorf("asr: no primary backend configured")
	}

	transcript, err := primary.TranscribeFile(filePath, opts)
	if err == nil {
		return transcript, nil
	}

	fallback := r.Fallback()
	if fallback == nil {
		return nil, fmt.Errorf("asr: primary backend %q failed: %w", r.primary, err)
	}

	transcript, fbErr := fallback.TranscribeFile(filePath, opts)
	if fbErr != nil {
		return nil, fmt.Errorf("asr: primary %q failed (%v), fallback %q also failed: %w", r.primary, err, r.fallback, fbErr)
	}

	return transcript, nil
}
