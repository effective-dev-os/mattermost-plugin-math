package mathexpr

import "errors"

var (
	ErrEmptyInput        = errors.New("expression is empty")
	ErrTooLong           = errors.New("expression is too long")
	ErrUnsupportedSyntax = errors.New("unsupported syntax in expression")
	ErrCompile           = errors.New("could not compile expression")
	ErrRuntime           = errors.New("error evaluating expression")
	ErrNonFiniteResult   = errors.New("result is not a finite number")
)
