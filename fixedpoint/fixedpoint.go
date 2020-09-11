// Package fixedpoint provides types and functions for performing
// fixed-point arithmetic.
//
// See https://en.wikipedia.org/wiki/Fixed-point_arithmetic for a
// longer explanation.
package fixedpoint

import (
	"fmt"
	"math"
	"strings"
)

// Value is a fixed-point decimal value. The zero value represents zero.
type Value struct {
	value    int32
	decimals uint16
}

// FromInt converts an integer to a fixed-point value.
func FromInt(i int32) Value {
	return Value{value: i}
}

// Int converts fp to an integer, truncating any fractional component.
func (fp Value) Int() int {
	v := int(fp.value)
	for i := uint16(0); i < fp.decimals; i++ {
		v /= 10
	}
	return v
}

// Float64 converts fp into a floating point value.
func (fp Value) Float64() float64 {
	e := math.Pow10(int(fp.decimals))
	return float64(fp.value) / e
}

// Add adds two fixed-point values together.
func (fp Value) Add(fp2 Value) Value {
	fp, fp2 = normalize(fp, fp2)
	return Value{
		value:    fp.value + fp2.value,
		decimals: fp.decimals,
	}
}

// Sub subtracts fp2 from fp and returns the result.
func (fp Value) Sub(fp2 Value) Value {
	fp, fp2 = normalize(fp, fp2)
	return Value{
		value:    fp.value - fp2.value,
		decimals: fp.decimals,
	}
}

// Equal reports whether fp is equivalent to fp2.
func (fp Value) Equal(fp2 Value) bool {
	fp, fp2 = normalize(fp, fp2)
	return fp.value == fp2.value
}

// Less reports whether fp is less than fp2.
func (fp Value) Less(fp2 Value) bool {
	fp, fp2 = normalize(fp, fp2)
	return fp.value < fp2.value
}

// Neg returns the negation of fp.
func (fp Value) Neg() Value {
	return Value{value: -fp.value, decimals: fp.decimals}
}

// Shift10 multiplies the value by 10^n.
func (fp Value) Shift10(n int16) Value {
	if n < 0 {
		return Value{value: fp.value, decimals: fp.decimals - uint16(n)}
	}
	return Value{value: fp.value, decimals: fp.decimals + uint16(n)}
}

// String formats the value as a decimal string like "3.14".
func (fp Value) String() string {
	// TODO(someday): Make efficient by printing digits.
	return fmt.Sprint(fp.Float64())
}

// Format formats the value for display in the fmt package.
func (fp Value) Format(f fmt.State, c rune) {
	if c == 'v' && f.Flag('+') {
		fmt.Fprintf(f, "fixedpoint.Value{value: %d, decimals: %d}", fp.value, fp.decimals)
		return
	}
	// TODO(someday): Make efficient in more common cases.
	formatStr := new(strings.Builder)
	formatStr.WriteByte('%')
	if f.Flag('+') {
		formatStr.WriteByte('+')
	}
	if f.Flag('-') {
		formatStr.WriteByte('-')
	}
	if f.Flag(' ') {
		formatStr.WriteByte(' ')
	}
	if f.Flag('0') {
		formatStr.WriteByte('0')
	}
	if w, ok := f.Width(); ok {
		fmt.Fprint(formatStr, w)
	}
	if prec, ok := f.Precision(); ok {
		formatStr.WriteByte('.')
		fmt.Fprint(formatStr, prec)
	}
	formatStr.WriteRune(c)
	fmt.Fprintf(f, formatStr.String(), fp.Float64())
}

func normalize(fp1, fp2 Value) (Value, Value) {
	for fp1.decimals > fp2.decimals {
		fp2.value *= 10
		fp2.decimals++
	}
	for fp1.decimals < fp2.decimals {
		fp1.value *= 10
		fp1.decimals++
	}
	return fp1, fp2
}
