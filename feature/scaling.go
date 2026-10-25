package feature

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"strconv"
)

// fidScale preserves the configured decimal coefficient exactly. The common
// integer-coefficient path uses checked machine arithmetic without allocations.
// A nil scale is the omitted coefficient, whose effective value is one.
type fidScale struct {
	factor                 big.Rat
	integer                uint64
	whole                  bool
	numerator, denominator uint64 // Reduced ratio; denominator==0 means arbitrary precision.
}

func newFIDScale(text string) (*fidScale, error) {
	s := &fidScale{}
	if _, ok := s.factor.SetString(text); !ok {
		return nil, fmt.Errorf("invalid numeric coefficient %q", text)
	}
	if s.factor.Sign() < 0 {
		return nil, fmt.Errorf("coefficient must be nonnegative, got %q", text)
	}
	s.whole = s.factor.IsInt() && s.factor.Num().IsUint64()
	if s.whole {
		s.integer = s.factor.Num().Uint64()
	}
	if s.factor.Num().IsUint64() && s.factor.Denom().IsUint64() {
		s.numerator = s.factor.Num().Uint64()
		s.denominator = s.factor.Denom().Uint64()
	}
	return s, nil
}

func scaleUint64(n uint64, scale *fidScale) (uint64, error) {
	if scale == nil {
		return n, nil
	}
	if scale.whole {
		high, low := bits.Mul64(n, scale.integer)
		if high != 0 {
			return 0, errValueOutOfRange
		}
		return low, nil
	}
	if scale.denominator != 0 {
		high, low := bits.Mul64(n, scale.numerator)
		// Div64 requires high < divisor, exactly the condition that the
		// truncated quotient fits in uint64. The 128-bit product never wraps.
		if high >= scale.denominator {
			return 0, errValueOutOfRange
		}
		quotient, _ := bits.Div64(high, low, scale.denominator)
		return quotient, nil
	}
	var input big.Rat
	input.SetUint64(n)
	return scaleRational(&input, scale)
}

func scaleInt64(n int64, scale *fidScale) (uint64, error) {
	if n < 0 {
		return 0, errValueBelowZero
	}
	return scaleUint64(uint64(n), scale)
}

func scaleFloat(n float64, width int, scale *fidScale) (uint64, error) {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, errValueNotFinite
	}
	if n < 0 {
		return 0, errValueBelowZero
	}
	// 2^64 is exactly representable; MaxUint64 is not representable as float64.
	const uint64Limit = 0x1p64
	if scale == nil || (scale.whole && scale.integer == 1) {
		if n >= uint64Limit {
			return 0, errValueOutOfRange
		}
		return uint64(n), nil
	}
	if scale.whole && scale.integer == 0 {
		return 0, nil
	}
	integral := n == math.Trunc(n)
	if integral && n < uint64Limit {
		return scaleUint64(uint64(n), scale)
	}
	var input big.Rat
	if integral {
		// Large floating-point integers retain their actual representable value.
		input.SetFloat64(n)
	} else {
		// Use the shortest round-trip decimal for fractional inputs. Thus 0.3*10
		// and 10*0.3 both yield 3 instead of leaking binary approximation into IDs.
		if _, ok := input.SetString(strconv.FormatFloat(n, 'g', -1, width)); !ok {
			return 0, errUnsupportedDataType
		}
	}
	return scaleRational(&input, scale)
}

func scaleRational(input *big.Rat, scale *fidScale) (uint64, error) {
	var product big.Rat
	product.Mul(input, &scale.factor)
	var integer big.Int
	integer.Quo(product.Num(), product.Denom()) // nonnegative, truncation toward zero
	if !integer.IsUint64() {
		return 0, errValueOutOfRange
	}
	return integer.Uint64(), nil
}
