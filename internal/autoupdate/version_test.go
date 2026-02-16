package autoupdate
package autoupdate

import "testing"

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.3.0", "0.3.0"},
		{"0.3.0-dirty", "0.3.0"},
		{"0.3.0-2-g5ea24ba", "0.3.0"},
		{"0.3.0-2-g5ea24ba-dirty", "0.3.0"},
		{"0.2.0-rc1", "0.2.0-rc1"},
		{"1.0.0-beta.1", "1.0.0-beta.1"},
		{"0.1.0-dev", "0.1.0-dev"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		version1 string
		version2 string
		expected bool
	}{
		{"0.3.0", "0.2.0", true},
		{"0.2.0", "0.3.0", false},
		{"1.0.0", "0.9.9", true},
		{"0.3.0", "0.3.0", false},
		{"0.3.1", "0.3.0", true},
		{"0.3.0-rc1", "0.2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.version1+" vs "+tt.version2, func(t *testing.T) {
			result := isNewer(tt.version1, tt.version2)
			if result != tt.expected {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.version1, tt.version2, result, tt.expected)
			}
		})
	}
}
