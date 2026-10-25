package utils

import (
	"fmt"
	"math"
	"strconv"
)

func SetFloatPrecision(value float64, precision int) float64 {
	length := 0
	data := int(value)
	for ; data != 0; data /= 10 {
		length += 1
	}
	pre := precision - length
	if pre <= 0 {
		pre = 0
	}
	str := "%." + fmt.Sprintf("%d", pre) + "f"
	floatNum, _ := strconv.ParseFloat(fmt.Sprintf(str, value), 64)
	return floatNum
}

func SetFloatPrecisionNew(value float64, precision int) float64 {
	length := 0
	data := int(value)
	for ; data != 0; data /= 10 {
		length += 1
	}
	pre := precision - length
	if pre <= 0 {
		pre = 0
	}

	unit := math.Pow10(pre)

	return math.Trunc(value*unit) / unit
}

func SetFloatPrecisionNewNew(value float64, precision int) float64 {
	length := 0
	data := int(value)
	for ; data != 0; data /= 10 {
		length += 1
	}
	pre := precision - length
	if pre <= 0 {
		pre = 0
	}

	valueStr := strconv.FormatFloat(value, 'f', pre, 64)
	finalValue, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return SetFloatPrecision(value, precision)
	}
	return finalValue
}
