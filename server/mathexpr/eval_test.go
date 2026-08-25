package mathexpr

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr error
	}{
		{name: "basic multiplication", input: "2*3*4*5", want: 120},
		{name: "space x", input: "2 x 2", want: 4},
		{name: "no space x", input: "2x2", want: 4},
		{name: "unicode times", input: "2×2", want: 4},
		{name: "unicode divide", input: "2÷2", want: 1},
		{name: "digit before paren", input: "2(3+4)", want: 14},
		{name: "paren before digit", input: "(3+4)2", want: 14},
		{name: "paren before paren", input: "(2+3)(4+5)", want: 45},
		{name: "digit before function", input: "2sqrt(4)", want: 4},
		{name: "unicode minus", input: "2−2", want: 0},
		{name: "en dash", input: "2–2", want: 0},
		{name: "em dash", input: "2—2", want: 0},
		{name: "comma decimals", input: "1,5 + 2,5", want: 4},
		{name: "trailing percent", input: "50%", want: 0.5},
		{name: "mid expression percent", input: "50% + 10", want: 10.5},
		{name: "percent on paren group", input: "(2+3)%", want: 0.05},
		{name: "percent on function call", input: "sqrt(4)%", want: 0.02},
		{name: "sin degrees", input: "sin(90)", want: 1},
		{name: "cos degrees", input: "cos(0)", want: 1},
		{name: "sin 180 degrees", input: "sin(180)", want: 0},
		{name: "cos 90 degrees", input: "cos(90)", want: 0},
		{name: "log base 10", input: "log(100)", want: 2},
		{name: "hex literal", input: "0x10", want: 16},
		{name: "scientific notation", input: "1e2", want: 100},
		{name: "abs builtin", input: "abs(-5)", want: 5},

		{name: "division by zero", input: "1/0", wantErr: ErrNonFiniteResult},
		{name: "negative division by zero", input: "-1/0", wantErr: ErrNonFiniteResult},
		{name: "zero over zero", input: "0/0", wantErr: ErrNonFiniteResult},
		{name: "incomplete expression", input: "2 +", wantErr: ErrCompile},
		{name: "unbalanced parens", input: "(2+3", wantErr: ErrCompile},
		{name: "empty string", input: "", wantErr: ErrEmptyInput},
		{name: "unsupported syntax semicolon", input: "2; 1", wantErr: ErrUnsupportedSyntax},
		{name: "unsupported syntax array", input: "[1,2,3]", wantErr: ErrUnsupportedSyntax},
		{name: "unsupported syntax string literal", input: `"abc"`, wantErr: ErrUnsupportedSyntax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Evaluate(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Evaluate(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Evaluate(%q) unexpected error: %v", tt.input, err)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateOverLength(t *testing.T) {
	long := strings.Repeat("1", maxInputRunes+1)
	_, err := Evaluate(long)
	if !errors.Is(err, ErrTooLong) {
		t.Fatalf("Evaluate(over-length) error = %v, want ErrTooLong", err)
	}
}
