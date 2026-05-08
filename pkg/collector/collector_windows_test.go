//go:build windows

package collector

import "testing"

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"42", 42},
		{"", 0},
		{"abc", 0},
		{"  -5  ", -5},
	}
	for _, tc := range tests {
		got := parseInt(tc.input)
		if got != tc.expected {
			t.Errorf("parseInt(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"17179869184", 17179869184},
		{"", 0},
		{"abc", 0},
		{"  1.5  ", 1.5},
	}
	for _, tc := range tests {
		got := parseFloat(tc.input)
		if got != tc.expected {
			t.Errorf("parseFloat(%q) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}

func TestMemoryType(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{20, "DDR"},
		{21, "DDR2"},
		{24, "DDR3"},
		{26, "DDR4"},
		{34, "DDR5"},
		{0, ""},
		{99, ""},
	}
	for _, tc := range tests {
		got := memoryType(tc.input)
		if got != tc.expected {
			t.Errorf("memoryType(%d) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestDiskType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SSD", "SSD"},
		{"Solid state drive", "SSD"},
		{"HDD", "HDD"},
		{"Fixed hard disk media", "HDD"},
		{"", ""},
		{"NVMe", ""},
	}
	for _, tc := range tests {
		got := diskType(tc.input)
		if got != tc.expected {
			t.Errorf("diskType(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestDecodeUint16(t *testing.T) {
	tests := []struct {
		input    []uint16
		expected string
	}{
		{[]uint16{68, 69, 76, 76, 0}, "DELL"},
		{[]uint16{}, ""},
		{[]uint16{72, 80, 0}, "HP"},
	}
	for _, tc := range tests {
		got := decodeUint16(tc.input)
		if got != tc.expected {
			t.Errorf("decodeUint16(%v) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
