package utils

import (
	"math"
	"strconv"
	"strings"
)

// ParseArrayInt replaces invalid elements with paddingValue, including without padding.
func ParseArrayInt(str, sep string, paddingSize int, paddingValue uint64) []uint64 {
	var res []uint64
	if str == "" {
		for i := 0; i < paddingSize; i++ {
			res = append(res, paddingValue)
		}
		return res
	}
	strList := strings.Split(str, sep)
	for _, elem := range strList {
		v, err := strconv.ParseUint(elem, 10, 64)
		if err != nil {
			res = append(res, paddingValue)
		} else {
			res = append(res, v)
		}
		if paddingSize > 0 && len(res) >= paddingSize {
			break
		}
	}
	for i := len(res); i < paddingSize; i++ {
		res = append(res, paddingValue)
	}
	return res
}

// ParseArrayIntNew preserves every input position, replacing invalid elements
// with paddingValue. Positive paddingSize pads empty/short input and truncates long input.
func ParseArrayIntNew(str, sep string, paddingSize int, paddingValue uint64) []uint64 {
	return ParseArrayIntNewInto(nil, str, sep, paddingSize, paddingValue)
}

// ParseArrayIntNewInto follows ParseArrayIntNew rules while reusing dst.
func ParseArrayIntNewInto(dst []uint64, str, sep string, paddingSize int, paddingValue uint64) []uint64 {
	dst = dst[:0]
	if paddingSize > cap(dst) {
		dst = make([]uint64, 0, paddingSize)
	}
	if str != "" {
		for elem := range strings.SplitSeq(str, sep) {
			v, err := strconv.ParseUint(elem, 10, 64)
			if err != nil {
				v = paddingValue
			}
			dst = append(dst, v)
			if paddingSize > 0 && len(dst) >= paddingSize {
				break
			}
		}
	}
	for len(dst) < paddingSize {
		dst = append(dst, paddingValue)
	}
	return dst
}

// ParseArray is the floating-point counterpart of ParseArrayInt.
func ParseArray(str, sep string, paddingSize int, paddingValue float64) []float64 {
	var res []float64
	if str == "" {
		for i := 0; i < paddingSize; i++ {
			res = append(res, paddingValue)
		}
		return res
	}
	strList := strings.Split(str, sep)
	for _, elem := range strList {
		v, err := strconv.ParseFloat(elem, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			res = append(res, paddingValue)
		} else {
			res = append(res, v)
		}
		if paddingSize > 0 && len(res) >= paddingSize {
			break
		}
	}
	for i := len(res); i < paddingSize; i++ {
		res = append(res, paddingValue)
	}
	return res
}

// ParseArrayNew preserves positions, replacing invalid or non-finite numbers
// with paddingValue. Positive paddingSize pads empty/short input and truncates long input.
func ParseArrayNew(str, sep string, paddingSize int, paddingValue float64) []float64 {
	return ParseArrayNewInto(nil, str, sep, paddingSize, paddingValue)
}

// ParseArrayNewInto follows ParseArrayNew rules while reusing dst.
func ParseArrayNewInto(dst []float64, str, sep string, paddingSize int, paddingValue float64) []float64 {
	dst = dst[:0]
	if paddingSize > cap(dst) {
		dst = make([]float64, 0, paddingSize)
	}
	if str != "" {
		for elem := range strings.SplitSeq(str, sep) {
			v, err := strconv.ParseFloat(elem, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				v = paddingValue
			}
			dst = append(dst, v)
			if paddingSize > 0 && len(dst) >= paddingSize {
				break
			}
		}
	}
	for len(dst) < paddingSize {
		dst = append(dst, paddingValue)
	}
	return dst
}
