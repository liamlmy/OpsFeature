package feature

import (
	"errors"
	"math"
	"strconv"

	"github.com/liamlmy/OpsFeature/utils"
)

//  1. value —— 特征值的无装箱表示
//  2. dedupSet —— 多值特征的去重容器
//  3. Base.emit —— fid 产出的唯一收口

type kind uint8

const (
	kindIface kind = iota // 兜底：调用方本来就持有 interface{}，或动态类型是 float32/uint32
	kindFloat64
	kindString
	kindInt64
	kindUint64
	kindBool
)

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

type dedupSet struct {
	hint int // 调用方在循环前设好的条数，首次建 map 时作容量提示

	f   map[float64]struct{}
	s   map[string]struct{}
	i64 map[int64]struct{}
	u64 map[uint64]struct{}
	ifc map[interface{}]struct{}
}

func (d *dedupSet) reset(hint int) {
	d.hint = hint
	if d.f != nil {
		if len(d.f) > maxKeptBuf {
			d.f = nil
		} else {
			for k := range d.f {
				delete(d.f, k)
			}
		}
	}
	if d.s != nil {
		if len(d.s) > maxKeptBuf {
			d.s = nil
		} else {
			for k := range d.s {
				delete(d.s, k)
			}
		}
	}
	if d.i64 != nil {
		if len(d.i64) > maxKeptBuf {
			d.i64 = nil
		} else {
			for k := range d.i64 {
				delete(d.i64, k)
			}
		}
	}
	if d.u64 != nil {
		if len(d.u64) > maxKeptBuf {
			d.u64 = nil
		} else {
			for k := range d.u64 {
				delete(d.u64, k)
			}
		}
	}
	if d.ifc != nil {
		if len(d.ifc) > maxKeptBuf {
			d.ifc = nil
		} else {
			for k := range d.ifc {
				delete(d.ifc, k)
			}
		}
	}
}

func (d *dedupSet) seen(v value) bool {
	switch v.k {
	case kindFloat64:
		if d.f == nil {
			d.f = make(map[float64]struct{}, d.hint)
		}
		_, ok := d.f[v.f]
		d.f[v.f] = struct{}{}
		return ok
	case kindString:
		if d.s == nil {
			d.s = make(map[string]struct{}, d.hint)
		}
		_, ok := d.s[v.s]
		d.s[v.s] = struct{}{}
		return ok
	case kindInt64:
		if d.i64 == nil {
			d.i64 = make(map[int64]struct{}, d.hint)
		}
		k := int64(v.u)
		_, ok := d.i64[k]
		d.i64[k] = struct{}{}
		return ok
	case kindUint64, kindBool:
		if d.u64 == nil {
			d.u64 = make(map[uint64]struct{}, d.hint)
		}
		_, ok := d.u64[v.u]
		d.u64[v.u] = struct{}{}
		return ok
	default:
		if d.ifc == nil {
			d.ifc = make(map[interface{}]struct{}, d.hint)
		}
		_, ok := d.ifc[v.iface]
		d.ifc[v.iface] = struct{}{}
		return ok
	}
}

type emitCtx struct {
	st  *State
	out *[]Fid

	dedup *dedupSet

	addcol bool
	acc    []string
}

func (c *emitCtx) enableDedup(n int) {
	if n == 1 {
		return
	}
	c.dedup = &c.st.dedup
	c.dedup.reset(n)
}

func (c *emitCtx) reserve(n int) {
	if c.addcol && n > 0 && cap(c.acc) < n {
		c.acc = make([]string, 0, n)
	}
}

func (b *Base) emit(v value, outSlot int, ctx *emitCtx) {
	if !b.isSeq && ctx.dedup != nil {
		if ctx.dedup.seen(v) {
			return
		}
	}

	if b.addcol {
		if s, err := v.toString(); err == nil {
			ctx.acc = append(ctx.acc, s)
		} else {
			logf("addcol convert failed, feature=%s class=%s slot_id=%d err=%v",
				b.name, b.class, b.slotID, err)
		}
	}

	if b.slotID == 0 {
		return
	}

	var fid uint64
	need := true
	if b.listCache && !b.strOut {
		if c := ctx.st.fids[b.name]; c != nil {
			if cached, ok := c.get(v); ok {
				fid, need = cached, false
			}
		}
	}

	if need {
		var err error
		fid, err = b.computeFid(v)
		if err != nil {
			logf("compute fid failed, feature=%s class=%s hash_version=%d slot_id=%d err=%v",
				b.name, b.class, b.hashVersion, b.slotID, err)
			return
		}
		if b.listCache {
			c := ctx.st.fids[b.name]
			if c == nil {
				c = &fidCache{}
				ctx.st.fids[b.name] = c
			}
			c.put(v, fid)
		}
	}

	if b.strOut {
		return
	}
	*ctx.out = append(*ctx.out, Fid{Slot: outSlot, Val: fid})
}

func (b *Base) computeFid(v value) (uint64, error) {
	switch b.hashVersion {
	case hashVersionRawValue, hashVersionSlotValue:
		coeff := 1.0
		if b.coeff > coeffEpsilon {
			coeff = b.coeff
		}
		val, err := v.toUint64(coeff)
		if err != nil {
			return 0, err
		}
		if b.hashVersion == hashVersionSlotValue {
			return utils.CombineSlotAndVal(b.slotID, val), nil
		}
		return val, nil

	case hashVersionFloatBits:
		f, err := v.toFloat32()
		if err != nil {
			return 0, err
		}
		return uint64(math.Float32bits(f)), nil

	case hashVersionShareSlot:
		s, err := v.toString()
		if err != nil {
			return 0, err
		}
		return utils.CombineSlotWithHash(b.shareSlot, utils.CityHash64String(s)), nil

	default:
		return utils.CombineSlotWithHash(b.slotID, v.hash()), nil
	}
}

func (b *Base) emitFloat64(v float64, outSlot int, ctx *emitCtx) {
	b.emit(f64Value(v), outSlot, ctx)
}

func (b *Base) emitString(v string, outSlot int, ctx *emitCtx) {
	b.emit(strValue(v), outSlot, ctx)
}

func (b *Base) emitInt64(v int64, outSlot int, ctx *emitCtx) {
	b.emit(i64Value(v), outSlot, ctx)
}

func (b *Base) emitUint64(v uint64, outSlot int, ctx *emitCtx) {
	b.emit(u64Value(v), outSlot, ctx)
}

func (b *Base) emitIface(v interface{}, outSlot int, ctx *emitCtx) {
	b.emit(ifaceValue(v), outSlot, ctx)
}

func (b *Base) flush(ctx *emitCtx) {
	if b.addcol && len(ctx.acc) > 0 {
		ctx.st.inner[b.name] = ctx.acc
	}
}
