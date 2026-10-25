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

// BinarySearch returns the first threshold >= value (using LessEqual's
// historical tolerance), or len(array) above all thresholds. Equal thresholds
// retain their configured positions. An empty table returns -1.
func BinarySearch(array []float64, value float64) int64 {
	if len(array) == 0 {
		return -1
	}
	left, right := 0, len(array)
	for left < right {
		mid := left + (right-left)/2
		if LessEqual(value, array[mid]) {
			right = mid
		} else {
			left = mid + 1
		}
	}
	return int64(left)
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
