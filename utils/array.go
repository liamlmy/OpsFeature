package utils

import (
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

// ParseArrayIntNew skips invalid elements unless padding is enabled.
// For compatibility, empty input returns nil even when paddingSize is positive.
func ParseArrayIntNew(str, sep string, paddingSize int, paddingValue uint64) []uint64 {
	var res []uint64
	if str == "" {
		return res
	}
	strList := strings.Split(str, sep)
	for _, elem := range strList {
		v, err := strconv.ParseUint(elem, 10, 64)
		if err == nil {
			res = append(res, v)
		} else if paddingSize > 0 {
			res = append(res, paddingValue)
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

// ParseArrayNew skips invalid elements unless padding is enabled.
// Unlike ParseArrayIntNew, empty input is padded to paddingSize.
func ParseArrayNew(str, sep string, paddingSize int, paddingValue float64) []float64 {
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
		if err == nil {
			res = append(res, v)
		} else if paddingSize > 0 {
			res = append(res, paddingValue)
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
