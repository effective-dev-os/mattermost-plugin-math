package mathexpr

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxInputRunes = 1024

// percentFuncNames and allowedFuncNames are intentionally distinct: the plan
// scopes the percent rewrite's function-call lookback to sqrt/sin/cos/log,
// while the final allowlist (and implicit multiplication) also accepts abs.
var percentFuncNames = map[string]bool{
	"sqrt": true,
	"sin":  true,
	"cos":  true,
	"log":  true,
}

var allowedFuncNames = map[string]bool{
	"sqrt": true,
	"sin":  true,
	"cos":  true,
	"log":  true,
	"abs":  true,
}

// Normalize runs the fixed normalization pipeline over raw user input and
// returns an expr-lang-compatible expression string.
//
// Step order is load-bearing, not incidental: unicode/comma rewrites must
// run before the percent rewrite (so "%"'s backward scan sees plain ASCII
// digits), the percent rewrite must run before implicit multiplication (so
// the parens it emits, e.g. "50%" -> "(50/100)", participate in the
// paren-adjacency rules), and the allowlist validation must run last, after
// every rewrite has had a chance to turn a disallowed character (",", "%",
// "×", unicode dashes) into an allowed one.
func Normalize(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrEmptyInput
	}
	if utf8.RuneCountInString(trimmed) > maxInputRunes {
		return "", ErrTooLong
	}

	s := replaceUnicodeSymbols(trimmed)
	s = replaceCommaDecimals(s)

	s, err := rewritePercent(s)
	if err != nil {
		return "", err
	}

	s = insertImplicitMultiplication(s)

	if err := validateAllowlist(s); err != nil {
		return "", err
	}

	return s, nil
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isAlpha(r rune) bool { return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') }

func isHexDigit(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }

// replaceUnicodeSymbols rewrites ×/÷/unicode dashes to their ASCII
// equivalents, and rewrites any "x"/"X" adjacent to a digit (ignoring
// intervening whitespace, so "2 x 2" and "2x2" both work) to "*". A "0x"/"0X"
// hex-literal prefix is excluded from the digit-adjacent-x rule, since it
// must stay a hex literal (e.g. "0x10" must not become "0*10").
func replaceUnicodeSymbols(s string) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	for i, r := range runes {
		switch r {
		case '×':
			out = append(out, '*')
		case '÷':
			out = append(out, '/')
		case '−', '–', '—':
			out = append(out, '-')
		case 'x', 'X':
			switch {
			case isHexLiteralPrefixX(runes, i):
				out = append(out, r)
			case isDigitAdjacent(runes, i):
				out = append(out, '*')
			default:
				out = append(out, r)
			}
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

// isHexLiteralPrefixX reports whether the x/X at index i immediately follows
// a "0" that starts a fresh numeric token (i.e. not the trailing "0" of a
// larger number like "20" or a decimal like "1.0").
func isHexLiteralPrefixX(runes []rune, i int) bool {
	if i == 0 || runes[i-1] != '0' {
		return false
	}
	if i-2 >= 0 && (isDigit(runes[i-2]) || runes[i-2] == '.') {
		return false
	}
	return true
}

// isDigitAdjacent reports whether the rune at index i has a digit as its
// nearest non-whitespace neighbor on at least one side.
func isDigitAdjacent(runes []rune, i int) bool {
	li := i - 1
	for li >= 0 && isSpace(runes[li]) {
		li--
	}
	if li >= 0 && isDigit(runes[li]) {
		return true
	}
	ri := i + 1
	for ri < len(runes) && isSpace(runes[ri]) {
		ri++
	}
	return ri < len(runes) && isDigit(runes[ri])
}

// replaceCommaDecimals rewrites a "," between two digits to a decimal point.
func replaceCommaDecimals(s string) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	for i, r := range runes {
		if r == ',' && i > 0 && i+1 < len(runes) && isDigit(runes[i-1]) && isDigit(runes[i+1]) {
			out = append(out, '.')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

// rewritePercent rewrites every "%" into a "(<operand>/100)" division,
// regardless of its position in the expression. This deliberately removes
// expr-lang's modulo operator from anything reachable through this command:
// "10%3" first becomes "(10/100)3" here, and the implicit-multiplication
// pass that follows turns it into "(10/100)*3" = 0.3, not "10 mod 3 = 1".
// Percent always wins over modulo.
func rewritePercent(s string) (string, error) {
	for {
		runes := []rune(s)
		idx := indexRune(runes, '%')
		if idx == -1 {
			return s, nil
		}
		start, err := findPercentOperandStart(runes, idx)
		if err != nil {
			return "", err
		}
		operand := string(runes[start:idx])
		replacement := "(" + operand + "/100)"
		s = string(runes[:start]) + replacement + string(runes[idx+1:])
	}
}

func indexRune(runes []rune, target rune) int {
	for i, r := range runes {
		if r == target {
			return i
		}
	}
	return -1
}

// findPercentOperandStart scans backward from a "%" at percentIdx to find
// the start of the operand it applies to: a numeric literal, or a balanced
// "(...)" group (tracking paren depth by hand, since RE2 can't match
// balanced groups), optionally preceded by a sqrt/sin/cos/log call name, and
// optionally preceded by a unary minus.
func findPercentOperandStart(s []rune, percentIdx int) (int, error) {
	if percentIdx == 0 {
		return 0, fmt.Errorf("%w: %% has no preceding operand", ErrUnsupportedSyntax)
	}

	j := percentIdx - 1
	var start int
	var err error

	switch {
	case s[j] == ')':
		start, err = findPercentParenOperandStart(s, j)
	case isDigit(s[j]) || s[j] == '.':
		start, err = findPercentNumericOperandStart(s, j, percentIdx)
	default:
		err = fmt.Errorf("%w: %% has no valid preceding operand", ErrUnsupportedSyntax)
	}
	if err != nil {
		return 0, err
	}

	if start > 0 && s[start-1] == '-' && isUnaryMinusContext(s, start-1) {
		start--
	}

	return start, nil
}

// findPercentParenOperandStart handles the "%" operand case where the char
// immediately before "%" is a ")", i.e. a parenthesized group or a
// sqrt/sin/cos/log(...) call.
func findPercentParenOperandStart(s []rune, closeParenIdx int) (int, error) {
	depth := 1
	j := closeParenIdx - 1
	for j >= 0 && depth > 0 {
		switch s[j] {
		case ')':
			depth++
		case '(':
			depth--
		}
		if depth == 0 {
			break
		}
		j--
	}
	if depth != 0 {
		return 0, fmt.Errorf("%w: unbalanced parentheses before %%", ErrUnsupportedSyntax)
	}
	parenStart := j
	start := parenStart

	k := parenStart - 1
	for k >= 0 && isAlpha(s[k]) {
		k--
	}
	if k+1 < parenStart {
		name := string(s[k+1 : parenStart])
		if percentFuncNames[name] {
			start = k + 1
		}
	}
	return start, nil
}

// findPercentNumericOperandStart handles the "%" operand case where the char
// immediately before "%" is a digit or ".", i.e. a plain numeric literal.
func findPercentNumericOperandStart(s []rune, digitIdx, percentIdx int) (int, error) {
	k := digitIdx
	for k >= 0 && (isDigit(s[k]) || s[k] == '.') {
		k--
	}
	start := k + 1
	if start == percentIdx {
		return 0, fmt.Errorf("%w: %% has no numeric operand", ErrUnsupportedSyntax)
	}
	return start, nil
}

// isUnaryMinusContext reports whether the "-" at idx is acting as a unary
// sign (start of the expression, or immediately preceded by another
// operator/open-paren) rather than binary subtraction.
func isUnaryMinusContext(s []rune, idx int) bool {
	k := idx - 1
	for k >= 0 && isSpace(s[k]) {
		k--
	}
	if k < 0 {
		return true
	}
	switch s[k] {
	case '+', '-', '*', '/', '^', '(':
		return true
	default:
		return false
	}
}

// insertImplicitMultiplication inserts "*" between: a digit and an
// immediately following "(", a ")" and an immediately following digit, a
// ")" and an immediately following "(", and a digit and an immediately
// following sqrt/sin/cos/log/abs call. Scientific notation ("1e5", "2e-3")
// is naturally excluded: "e"/"E" is never one of the recognized function
// names, so no special-case guard is needed here.
func insertImplicitMultiplication(s string) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes)+8)
	for i, r := range runes {
		out = append(out, r)
		switch {
		case r == ')':
			if i+1 < len(runes) {
				next := runes[i+1]
				if isDigit(next) || next == '(' {
					out = append(out, '*')
				}
			}
		case isDigit(r):
			if i+1 < len(runes) {
				next := runes[i+1]
				if next == '(' {
					out = append(out, '*')
				} else if funcNameStartsAt(runes, i+1) {
					out = append(out, '*')
				}
			}
		}
	}
	return string(out)
}

func funcNameStartsAt(runes []rune, i int) bool {
	for name := range allowedFuncNames {
		nr := []rune(name)
		if i+len(nr) > len(runes) {
			continue
		}
		match := true
		for k, c := range nr {
			if runes[i+k] != c {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// validateAllowlist is the final, primary defense: the normalized string may
// only contain digits, ".", whitespace, "+ - * / ^ ( )", 0x/0X hex literals,
// scientific-notation exponents, and identifier runs exactly matching
// sqrt/sin/cos/log/abs (case-sensitive, lowercase only). Anything else is
// rejected. This single allowlist step (not a second DisableAllBuiltins
// mechanism) is what keeps expr-lang's other builtins and string/array/pipe
// syntax unreachable.
func validateAllowlist(s string) error {
	runes := []rune(s)
	n := len(runes)
	i := 0

	for i < n {
		r := runes[i]
		switch {
		case isSpace(r):
			i++
		case r == '.' || r == '+' || r == '-' || r == '*' || r == '/' || r == '^' || r == '(' || r == ')':
			i++
		case isDigit(r):
			start := i
			for i < n && isDigit(runes[i]) {
				i++
			}
			if i-start == 1 && runes[start] == '0' && i < n && (runes[i] == 'x' || runes[i] == 'X') {
				i++
				hexStart := i
				for i < n && isHexDigit(runes[i]) {
					i++
				}
				if i == hexStart {
					return fmt.Errorf("%w: invalid hex literal", ErrUnsupportedSyntax)
				}
				continue
			}
			if i < n && (runes[i] == 'e' || runes[i] == 'E') {
				j := i + 1
				if j < n && (runes[j] == '+' || runes[j] == '-') {
					j++
				}
				if j < n && isDigit(runes[j]) {
					for j < n && isDigit(runes[j]) {
						j++
					}
					i = j
				}
			}
		case isAlpha(r):
			start := i
			for i < n && isAlpha(runes[i]) {
				i++
			}
			name := string(runes[start:i])
			if !allowedFuncNames[name] {
				return fmt.Errorf("%w: unsupported identifier %q", ErrUnsupportedSyntax, name)
			}
		default:
			return fmt.Errorf("%w: unsupported character %q", ErrUnsupportedSyntax, string(r))
		}
	}
	return nil
}
