package mathexpr

import (
	"math"
	"testing"
)

func TestFormatResult(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{120, "120"},
		{0.5, "0.5"},
		{0, "0"},
		{-0.05, "-0.05"},
		{2, "2"},
		{0.5 + 3.552713678800501e-17, "0.5"},
		{math.Sin(math.Pi), "0"},
		{math.Cos(math.Pi / 2), "0"},
	}
	for _, tt := range tests {
		if got := FormatResult(tt.input); got != tt.want {
			t.Errorf("FormatResult(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
