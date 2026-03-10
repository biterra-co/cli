package cmd

import (
	"testing"
)

func TestParseNumber(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1", 1},
		{"2", 2},
		{" 3 ", 3},
		{"0", 0},
		{"", 0},
		{"x", 0},
		{"42", 42},
	}
	for _, tt := range tests {
		got := parseNumber(tt.in)
		if got != tt.want {
			t.Errorf("parseNumber(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
