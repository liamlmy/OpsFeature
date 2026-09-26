package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	delta             = 1e-8
	PI                = 3.141592653589793
	TimeFormat        = "2006-01-02 15:04:05"
	Factor       int  = 100000
	RAD               = PI / 180.0
	EARTH_RADIUS      = 6378137
	m            uint = 0x5bd1e995
	r            int  = 24
	seed         int  = 97
)

type Grid struct {
	XMin     float64
	YMin     float64
	XMax     float64
	YMax     float64
	XLen     int64
	YLen     int64
	Gap      float64
	BeginPos int64
}

func (g Grid) GetAxisIndex(v, begin, step float64, mx int) int64 {
	if Less(v, begin) {
		return 0
	}
	ind := int64((v - begin) / step)
	if ind > int64(mx) {
		ind = int64(mx)
	}
	return ind + 1
}

func (g Grid) Size() int64 {
	if g.XLen == 0 && g.YLen == 0 {
		return 0
	}
	return (g.XLen + 2) * (g.YLen + 2)
}

func (g *Grid) SetBeginPos(p int64) {
	g.BeginPos = p
}

func (g Grid) GetPos(x, y float64) int64 {
	xInd := g.GetAxisIndex(x, g.XMin, g.Gap, int(g.XLen))
	yInd := g.GetAxisIndex(y, g.YMin, g.Gap, int(g.YLen))
	pos := yInd*(g.XLen+2) + xInd
	return pos + g.BeginPos
}

func Equal(x, y float64) bool {
	diff := x - y
	if diff < delta && diff > -delta {
		return true
	} else {
		return false
	}
}

func LessEqual(x, y float64) bool {
	diff := x - y
	if diff < delta {
		return true
	} else {
		return false
	}
}

func Less(x, y float64) bool {
	diff := x - y
	if diff < -delta {
		return true
	}
	return false
}

func BinarySearch(array []float64, value float64) int64 {
	if array == nil || len(array) == 0 {
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

func ConvertStr2Time(timestamp, format string) (t time.Time, err error) {
	if !strings.Contains(format, "%Y") || !strings.Contains(format, "%m") || !strings.Contains(format, "%d") {
		return
	}
	if !strings.Contains(format, "%H") {
		timestamp += "00"
		format += "%H"
	}

	if !strings.Contains(format, "%M") {
		timestamp += "00"
		format += "%M"
	}

	if !strings.Contains(format, "%S") {
		timestamp += "00"
		format += "%S"
	}
	t, err = time.ParseInLocation(TimeFormat, timestamp, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q with format %q: %w", timestamp, format, err)
	}
	return t, nil
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

func GetAngle(xBeginLng, xBeginLat, xEndLng, xEndLat, yBeginLng, yBeginLat, yEndLng, yEndLat float64) float64 {
	directionX1 := xEndLng - xBeginLng
	directionY1 := xEndLat - xBeginLat
	directionX2 := yEndLng - yBeginLng
	directionY2 := yEndLat - yBeginLat

	module1 := math.Sqrt(directionX1*directionX1 + directionY1*directionY1)
	module2 := math.Sqrt(directionX2*directionX2 + directionY2*directionY2)
	inner := directionX1*directionX2 + directionY1*directionY2
	mo := module1 * module2

	if mo < 1e-7 {
		return -1
	}

	cosineAngle := inner / mo
	radianAngle := math.Acos(cosineAngle)
	angle := (radianAngle / PI) * 180
	return angle
}

func GetDirectDistance(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := lat1 - lat2
	dLng := lng1 - lng2
	d := math.Sqrt(dLat*dLat + dLng*dLng)
	return d * float64(Factor)
}

func GetSphereDistance(lat1, lng1, lat2, lng2 float64) float64 {
	radLat1 := lat1 * RAD
	radLat2 := lat2 * RAD
	a := radLat1 - radLat2
	b := (lng1 - lng2) * RAD
	s := 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2)))
	s = s * EARTH_RADIUS
	return s
}

func MurMurHash(key []rune, len int) uint {
	var h uint = uint(seed ^ len)
	var data uint
	data = uint(key[0])
	for len > 4 {
		k := data
		k *= m
		k ^= k >> r
		k *= m
		h *= m
		h ^= k
		data += 4
		len -= 4
	}
	switch len {
	case 3:
		h ^= uint(key[2]) << 16
	case 2:
		h ^= uint(key[1]) << 8
	case 1:
		h ^= uint(key[0])
	}
	h *= m
	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}

func BKDRHash(str string) uint {
	var bSeed uint = 31
	var hash uint = 0
	for _, ele := range str {
		hash = hash*bSeed + uint(ele)
	}
	return hash & 0x7FFFFFFF
}

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

	//fmt.Println("unit: ", unit)
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
		// bug log.Errorf("_SetFloatPrecisionNewNew||||err=%v", err)
		return SetFloatPrecision(value, precision)
	}
	return finalValue
}
