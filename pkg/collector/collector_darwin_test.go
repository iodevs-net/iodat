//go:build darwin

package collector

import "testing"

func TestParseSizeToGB(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"500.24 GB", 500},
		{"1 TB", 1000},
		{"256 KB", 0}, // < 1GB → 0
		{"", 0},
		{"abc", 0},
		{"2.5 TB", 2500},
	}
	for _, tc := range tests {
		got := parseSizeToGB(tc.input)
		if got != tc.expected {
			t.Errorf("parseSizeToGB(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}
