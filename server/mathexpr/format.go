package mathexpr

import (
	"math"
	"strconv"
)

// snapEpsilon controls how close a result must be to an integer before it is
// snapped to that integer. This is what makes degree-based trig print clean
// (sin(180) -> "0" instead of "1.2246467991473515e-16").
const snapEpsilon = 1e-9

// FormatResult renders a float64 result as a compact, user-facing string.
func FormatResult(v float64) string {
	if rounded := math.Round(v); math.Abs(v-rounded) < snapEpsilon {
		v = rounded
	}
	if v == 0 {
		v = 0 // normalize -0 to 0
	}
	return strconv.FormatFloat(v, 'g', 12, 64)
}
