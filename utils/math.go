package utils

// delta is the tolerance used by the comparison helpers.
const delta = 1e-8

func Equal(x, y float64) bool {
	diff := x - y
	return diff < delta && diff > -delta
}

func LessEqual(x, y float64) bool {
	diff := x - y
	return diff < delta
}

func Less(x, y float64) bool {
	diff := x - y
	return diff < -delta
}

func BinarySearch(array []float64, value float64) int64 {
	if len(array) == 0 {
		return -1
	}

	if len(array) == 1 {
		return 0
	}

	length := len(array)
	var mid int
	left := 0
	right := length - 1

	if LessEqual(value, array[left]) {
		return int64(left)
	}

	if !LessEqual(value, array[right]) {
		return int64(right + 1)
	}

	for left+1 < right {
		mid = left + (right-left)/2
		if LessEqual(value, array[mid]) {
			right = mid
		} else {
			left = mid
		}
	}
	return int64(right)
}

func Float2Int(val float64) int {
	i := int(val)
	if val > 0.0 {
		if float64(i+1)-val < 1e-6 {
			i++
		}
	} else {
		if val+float64(1)-float64(i) < 1e-6 {
			i--
		}
	}
	return i
}
