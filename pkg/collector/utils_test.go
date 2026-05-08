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
