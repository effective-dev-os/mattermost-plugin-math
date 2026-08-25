package mathexpr

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{name: "space separated x", input: "2 x 2", want: "2 * 2"},
		{name: "no space x", input: "2x2", want: "2*2"},
		{name: "asterisk", input: "2*2", want: "2*2"},
		{name: "unicode times", input: "2×2", want: "2*2"},
		{name: "unicode divide", input: "2÷2", want: "2/2"},
		{name: "digit before paren", input: "2(3+4)", want: "2*(3+4)"},
		{name: "paren before digit", input: "(3+4)2", want: "(3+4)*2"},
		{name: "paren before paren", input: "(2+3)(4+5)", want: "(2+3)*(4+5)"},
		{name: "digit before function", input: "2sqrt(4)", want: "2*sqrt(4)"},
		{name: "unicode minus U+2212", input: "2−2", want: "2-2"},
		{name: "en dash U+2013", input: "2–2", want: "2-2"},
		{name: "em dash U+2014", input: "2—2", want: "2-2"},
		{name: "comma decimals", input: "1,5 + 2,5", want: "1.5 + 2.5"},
		{name: "trailing percent", input: "50%", want: "(50/100)"},
		{name: "mid expression percent", input: "50% + 10", want: "(50/100) + 10"},
		{name: "percent on paren group", input: "(2+3)%", want: "((2+3)/100)"},
		{name: "percent on function call", input: "sqrt(4)%", want: "(sqrt(4)/100)"},
		{name: "percent then modulo elimination", input: "10%3", want: "(10/100)*3"},
		{name: "unary minus percent", input: "-5%", want: "(-5/100)"},
		{name: "binary minus not absorbed by percent", input: "3-5%", want: "3-(5/100)"},
		{name: "hex literal preserved", input: "0x10", want: "0x10"},
		{name: "hex literal not confused with trailing zero", input: "20x5", want: "20*5"},
		{name: "scientific notation preserved", input: "1e2", want: "1e2"},
		{name: "scientific notation with sign", input: "2e-3", want: "2e-3"},

		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "whitespace only", input: "   ", wantErr: ErrEmptyInput},
		{name: "over length", input: strings.Repeat("1", maxInputRunes+1), wantErr: ErrTooLong},
		{name: "leading percent has no operand", input: "%5", wantErr: ErrUnsupportedSyntax},
		{name: "semicolon rejected", input: "2; 1", wantErr: ErrUnsupportedSyntax},
		{name: "array literal rejected", input: "[1,2,3]", wantErr: ErrUnsupportedSyntax},
		{name: "string literal rejected", input: `"abc"`, wantErr: ErrUnsupportedSyntax},
		{name: "uppercase func name rejected", input: "SQRT(4)", wantErr: ErrUnsupportedSyntax},
		{name: "unsupported identifier rejected", input: "2 + max(1,2)", wantErr: ErrUnsupportedSyntax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Normalize(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReplaceUnicodeSymbols(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2×3", "2*3"},
		{"2÷3", "2/3"},
		{"2−3", "2-3"},
		{"0x10", "0x10"},
		{"0X1F", "0X1F"},
		{"2x3", "2*3"},
		{"2 x 3", "2 * 3"},
		{"x", "x"},
	}
	for _, tt := range tests {
		if got := replaceUnicodeSymbols(tt.input); got != tt.want {
			t.Errorf("replaceUnicodeSymbols(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReplaceCommaDecimals(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1,5", "1.5"},
		{"1,5 + 2,5", "1.5 + 2.5"},
		{"(1,2,3)", "(1.2.3)"},
		{",5", ",5"},
		{"5,", "5,"},
	}
	for _, tt := range tests {
		if got := replaceCommaDecimals(tt.input); got != tt.want {
			t.Errorf("replaceCommaDecimals(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInsertImplicitMultiplication(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2(3+4)", "2*(3+4)"},
		{"(3+4)2", "(3+4)*2"},
		{"(2+3)(4+5)", "(2+3)*(4+5)"},
		{"2sqrt(4)", "2*sqrt(4)"},
		{"1e5", "1e5"},
		{"2e-3", "2e-3"},
	}
	for _, tt := range tests {
		if got := insertImplicitMultiplication(tt.input); got != tt.want {
			t.Errorf("insertImplicitMultiplication(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateAllowlist(t *testing.T) {
	valid := []string{"2+2", "0x10", "1e5", "2e-3", "sqrt(4)", "sin(90)", "cos(0)", "log(100)", "abs(-5)", "2*3.5"}
	for _, s := range valid {
		if err := validateAllowlist(s); err != nil {
			t.Errorf("validateAllowlist(%q) unexpected error: %v", s, err)
		}
	}

	invalid := []string{"2;1", "[1,2]", `"a"`, "SQRT(4)", "max(1,2)", "2#3", "2|3", "2=3"}
	for _, s := range invalid {
		if err := validateAllowlist(s); !errors.Is(err, ErrUnsupportedSyntax) {
			t.Errorf("validateAllowlist(%q) error = %v, want ErrUnsupportedSyntax", s, err)
		}
	}
}
