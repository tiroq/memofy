// Package audio — format profile definitions for recording output.
package audio

// FormatProfile identifies a recording quality preset.
type FormatProfile string

const (
	FormatHigh        FormatProfile = "high"
	FormatBalanced    FormatProfile = "balanced"
	FormatLightweight FormatProfile = "lightweight"
	FormatWAV         FormatProfile = "wav"
)

// FormatSpec describes the complete output format for a recording.
type FormatSpec struct {
	Profile     FormatProfile
	Container   string // "m4a" or "wav"
	Codec       string // "aac" or "pcm_s16le"
	Channels    int
	SampleRate  int
	BitrateKbps int
}

// FormatSpecs maps each profile to its specification.
var FormatSpecs = map[FormatProfile]FormatSpec{
	FormatHigh: {
		Profile:     FormatHigh,
		Container:   "m4a",
		Codec:       "aac",
		Channels:    1,
		SampleRate:  32000,
		BitrateKbps: 64,
	},
	FormatBalanced: {
		Profile:     FormatBalanced,
		Container:   "m4a",
		Codec:       "aac",
		Channels:    1,
		SampleRate:  24000,
		BitrateKbps: 48,
	},
	FormatLightweight: {
		Profile:     FormatLightweight,
		Container:   "m4a",
		Codec:       "aac",
		Channels:    1,
		SampleRate:  16000,
		BitrateKbps: 32,
	},
	FormatWAV: {
		Profile:    FormatWAV,
		Container:  "wav",
		Codec:      "pcm_s16le",
		Channels:   1,
		SampleRate: 44100,
	},
}

// GetFormatSpec returns the FormatSpec for the given profile name.
// Falls back to FormatHigh if the profile is not recognized.
func GetFormatSpec(profile string) FormatSpec {
	if spec, ok := FormatSpecs[FormatProfile(profile)]; ok {
		return spec
	}
	return FormatSpecs[FormatHigh]
}

// ValidProfiles returns the list of valid profile names.
func ValidProfiles() []string {
	return []string{
		string(FormatHigh),
		string(FormatBalanced),
		string(FormatLightweight),
		string(FormatWAV),
	}
}

// IsValidProfile returns true if the profile name is recognized.
func IsValidProfile(profile string) bool {
	_, ok := FormatSpecs[FormatProfile(profile)]
	return ok
}

// FileExtension returns the file extension (with leading dot) for the profile.
func (s FormatSpec) FileExtension() string {
	if s.Container == "m4a" {
		return ".m4a"
	}
	return ".wav"
}
