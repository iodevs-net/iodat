//go:build linux

package collector

import "testing"

func TestParseBlocksToGB(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1000215216", 512},   // ~500GB
		{"0", 0},
		{"", 0},
		{"abc", 0},
		{"2000430432", 1024}, // ~1TB
	}
	for _, tc := range tests {
		got := parseBlocksToGB(tc.input)
		if got != tc.expected {
			t.Errorf("parseBlocksToGB(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"1000", 1000},
		{"", 0},
		{"abc", 0},
		{"  -1  ", -1},
	}
	for _, tc := range tests {
		got := parseInt64(tc.input)
		if got != tc.expected {
			t.Errorf("parseInt64(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}
