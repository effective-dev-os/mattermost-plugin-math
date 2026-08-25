package mathexpr

import (
	"fmt"
	"math"

	"github.com/expr-lang/expr"
)

var evalOptions = []expr.Option{
	expr.Env(map[string]any{}),
	expr.AsFloat64(),
	expr.Function("sqrt", func(params ...any) (any, error) {
		return math.Sqrt(params[0].(float64)), nil
	}, new(func(float64) float64)),
	expr.Function("sin", func(params ...any) (any, error) {
		return math.Sin(params[0].(float64) * math.Pi / 180), nil
	}, new(func(float64) float64)),
	expr.Function("cos", func(params ...any) (any, error) {
		return math.Cos(params[0].(float64) * math.Pi / 180), nil
	}, new(func(float64) float64)),
	// log is base-10 (calculator convention), not natural log, mirroring the
	// degrees convention chosen for sin/cos: chat users expect log(100) = 2.
	expr.Function("log", func(params ...any) (any, error) {
		return math.Log10(params[0].(float64)), nil
	}, new(func(float64) float64)),
}

// Evaluate normalizes and evaluates raw as a math expression, returning a
// finite float64 result or one of the sentinel errors in errors.go.
func Evaluate(raw string) (float64, error) {
	normalized, err := Normalize(raw)
	if err != nil {
		return 0, err
	}

	program, err := expr.Compile(normalized, evalOptions...)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrCompile, err)
	}

	out, err := expr.Run(program, map[string]any{})
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrRuntime, err)
	}

	v, ok := out.(float64)
	if !ok {
		return 0, fmt.Errorf("%w: unexpected result type %T", ErrRuntime, out)
	}

	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, ErrNonFiniteResult
	}

	return v, nil
}
