package collector

import "testing"

func TestCleanCPUName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz", "Intel Core i7-10700"},
		{"AMD Ryzen 5 5600X", "AMD Ryzen 5 5600X"},
		{"Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz", "Intel Xeon E5-2680 v4"},
		{"", ""},
		{"  CPU  ", ""},
		{"CPU", ""},
		{"Apple M1 Pro", "Apple M1 Pro"},
		{"12th Gen Intel(R) Core(TM) i9-12900K", "12th Gen Intel Core i9-12900K"},
	}
	for _, tc := range tests {
		got := cleanCPUName(tc.input)
		if got != tc.expected {
			t.Errorf("cleanCPUName(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

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
		got := ParseInt(tc.input)
		if got != tc.expected {
			t.Errorf("ParseInt(%q) = %d, want %d", tc.input, got, tc.expected)
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
		{"9223372036854775807", 9223372036854775807},
	}
	for _, tc := range tests {
		got := ParseInt64(tc.input)
		if got != tc.expected {
			t.Errorf("ParseInt64(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestParseFloat64(t *testing.T) {
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
		got := ParseFloat64(tc.input)
		if got != tc.expected {
			t.Errorf("ParseFloat64(%q) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}

func TestFromBlocks(t *testing.T) {
	tests := []struct {
		input    int64
		expected ByteSize
	}{
		{1000215216, ByteSize(1000215216) * 512},
		{0, 0},
		{2000430432, ByteSize(2000430432) * 512},
	}
	for _, tc := range tests {
		got := FromBlocks(tc.input)
		if got != tc.expected {
			t.Errorf("FromBlocks(%d) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestFromBytes(t *testing.T) {
	if got := FromBytes(17179869184); got != ByteSize(17179869184) {
		t.Errorf("FromBytes(17179869184) = %d, want %d", got, ByteSize(17179869184))
	}
	if got := FromBytes(0); got != 0 {
		t.Errorf("FromBytes(0) = %d, want 0", got)
	}
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input    string
		expected ByteSize
		wantErr  bool
	}{
		{"500.24 GB", ByteSize(500240000000), false},  // 500.24 × 1e9
		{"1 TB", 1000000000000, false},
		{"256 MB", 256000000, false},
		{"512 KB", 512000, false},
		{"", 0, true},
		{"abc", 0, true},
		{"2.5 TB", 2500000000000, false},
		{"  16  GB  ", 16000000000, false}, // with spaces
	}
	for _, tc := range tests {
		got, err := ParseByteSize(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseByteSize(%q) expected error, got %d", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseByteSize(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.expected {
			t.Errorf("ParseByteSize(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestByteSizeGB(t *testing.T) {
	tests := []struct {
		input    ByteSize
		expected int
	}{
		{500 * GB, 500},
		{TB, 1000},
		{0, 0},
		{1500 * MB, 1},  // 1.5 GB → trunca a 1
	}
	for _, tc := range tests {
		got := tc.input.GB()
		if got != tc.expected {
			t.Errorf("%d.GB() = %d, want %d", tc.input, got, tc.expected)
		}
	}
}
