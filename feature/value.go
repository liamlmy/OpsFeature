package feature

import (
	"errors"
	"strconv"

	"github.com/liamlmy/OpsFeature/utils"
)

// kind identifies the scalar representation stored in a value.
type kind uint8

const (
	kindIface kind = iota // 兜底：调用方本来就持有 interface{}，或动态类型是 float32/uint32
	kindFloat64
	kindString
	kindInt64
	kindUint64
	kindBool
)

// value stores scalar feature values and handles their FID conversions.
type value struct {
	k     kind
	f     float64     // kindFloat64
	u     uint64      // kindInt64(uint64 位模式) / kindUint64 / kindBool(0|1)
	s     string      // kindString
	iface interface{} // 仅 kindIface
}

func f64Value(v float64) value { return value{k: kindFloat64, f: v} }
func strValue(v string) value  { return value{k: kindString, s: v} }
func i64Value(v int64) value   { return value{k: kindInt64, u: uint64(v)} }
func u64Value(v uint64) value  { return value{k: kindUint64, u: v} }

func ifaceValue(v interface{}) value {
	switch x := v.(type) {
	case float64:
		return f64Value(x)
	case string:
		return strValue(x)
	case int64:
		return i64Value(x)
	case int:
		return i64Value(int64(x))
	case uint64:
		return u64Value(x)
	case bool:
		vv := value{k: kindBool}
		if x {
			vv.u = 1
		}
		return vv
	default:
		return value{k: kindIface, iface: v}
	}
}

var (
	errValueBelowZero      = errors.New("operator: value below 0")
	errUnsupportedDataType = errors.New("operator: unsupported data type")
)

const maxUint64AsFloat = float64(1<<64 - 1)

func (v value) toString() (string, error) {
	switch v.k {
	case kindFloat64:
		return strconv.FormatFloat(v.f, 'f', -1, 64), nil
	case kindString:
		return v.s, nil
	case kindInt64:
		return strconv.FormatInt(int64(v.u), 10), nil
	case kindUint64:
		return strconv.FormatUint(v.u, 10), nil
	case kindBool:
		if v.u == 1 {
			return "1", nil
		}
		return "0", nil
	default:
		if f, ok := v.iface.(float32); ok {
			return strconv.FormatFloat(float64(f), 'f', -1, 32), nil
		}
		return "", errUnsupportedDataType
	}
}

func (v value) toUint64(coeff float64) (uint64, error) {
	switch v.k {
	case kindInt64:
		val := int64(v.u) * int64(coeff)
		if val >= 0 {
			return uint64(val), nil
		}
		return 0, errValueBelowZero
	case kindUint64:
		return v.u * uint64(coeff), nil
	case kindFloat64:
		val := v.f * coeff
		if val >= 0 && val <= maxUint64AsFloat {
			return uint64(val), nil
		}
		return 0, errValueBelowZero
	case kindBool:
		if v.u == 1 {
			return uint64(coeff), nil
		}
		return 0, nil
	case kindString:
		return 0, errUnsupportedDataType
	default:
		return convertToUint64(v.iface, coeff)
	}
}

func (v value) toFloat32() (float32, error) {
	switch v.k {
	case kindFloat64:
		return float32(v.f), nil
	case kindInt64:
		return float32(int64(v.u)), nil
	case kindUint64:
		return float32(v.u), nil
	case kindBool:
		if v.u == 1 {
			return 1, nil
		}
		return 0, nil
	case kindString:
		f, err := strconv.ParseFloat(v.s, 64)
		if err != nil {
			return 0, err
		}
		return float32(f), nil
	default:
		return convertToFloat32(v.iface)
	}
}

func (v value) hash() uint64 {
	switch v.k {
	case kindFloat64:
		return utils.CityHash64Float64(v.f)
	case kindString:
		return utils.CityHash64String(v.s)
	case kindInt64:
		return utils.CityHash64Int64(int64(v.u))
	case kindUint64:
		return utils.CityHash64Uint64(v.u)
	case kindBool:
		return utils.CityHash64Bool(v.u == 1)
	default:
		return utils.CityHash64Any(v.iface)
	}
}

func convertToUint64(data interface{}, coeff float64) (uint64, error) {
	switch v := data.(type) {
	case uint32:
		// 保持原有的 uint32 乘法语义（在 uint32 上做乘法，会回绕）
		return uint64(v * uint32(coeff)), nil
	case float32:
		val := v * float32(coeff)
		if val >= 0 {
			return uint64(val), nil
		}
		return 0, errValueBelowZero
	default:
		return 0, errUnsupportedDataType
	}
}

func convertToFloat32(data interface{}) (float32, error) {
	switch v := data.(type) {
	case int8:
		return float32(v), nil
	case int16:
		return float32(v), nil
	case int32:
		return float32(v), nil
	case uint:
		return float32(v), nil
	case uint8:
		return float32(v), nil
	case uint16:
		return float32(v), nil
	case uint32:
		return float32(v), nil
	case float32:
		return v, nil
	default:
		return 0, errUnsupportedDataType
	}
}
