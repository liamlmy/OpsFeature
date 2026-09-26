package operator

import (
	"fmt"

	"github.com/liamlmy/OpsFeature/feature"
)

// Direct 对应 class=Direct：不做变换，直接把依赖列的值转成 fid。
//
// args[0] hash_version
// args[1] coeff
// args[2] valueType —— 决定从依赖列里怎么取值，见下方 8 个常量
// args[3] col / userValue —— 仅部分 valueType 使用，含义随分支变化
type Direct struct {
	feature.Base

	valueType directValueType
	col       int
}

type directValueType uint8

const (
	dvDouble     directValueType = iota // 连续数值
	dvStringPos                         // 单值类别（按依赖列下标取）
	dvKVIntFloat                        // int key、float64 value
	dvKVIntStr                          // int key、string value
	dvStringDict                        // 多值词典（逗号分隔）
	dvKVStrFloat                        // string key、float64 value
	dvTags                              // 带权多值，key 是 int64
	dvAddCol                            // 格式依赖上述算子的产出
)

var directValueTypes = map[string]directValueType{
	"double":       dvDouble,
	"str_pos":      dvStringPos,
	"kv_int_float": dvKVIntFloat,
	"kv_int_str":   dvKVIntStr,
	"str_dict":     dvStringDict,
	"kv_str_float": dvKVStrFloat,
	"tags":         dvTags,
	"add_col":      dvAddCol,
}

func (d *Direct) Init() error {
	if err := d.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := d.RequireDepends(1); err != nil {
		return err
	}

	name := d.ArgStr(2, "double")
	vt, ok := directValueTypes[name]
	if !ok {
		return fmt.Errorf("operator: feature %q unknown args[2] valueType %q", d.Name(), name)
	}
	d.valueType = vt

	col, err := d.ArgInt(3)
	if err != nil {
		return err
	}
	d.col = col
	return nil
}

func (d *Direct) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	dependName := d.Depends()[0]
	slot := d.SlotID()
	ctx := feature.NewEmitCtx(st, out, d.AddCol())
	defer d.Flush(ctx)

	switch d.valueType {

	case dvDouble:
		val := d.DependFloat64(dependName, in, st)
		d.EmitFloat64(normalizeMissing(dependName, val), slot, ctx)

	case dvStringPos:
		val := d.DependStringAt(0, in, st)
		if val == "" {
			return nil
		}
		d.EmitString(val, slot, ctx)

	case dvKVIntFloat:
		src, _ := d.Source(feature.SrcKVIntFloat, dependName, in, st).(*feature.KVIntFloatSource)
		if src == nil {
			return nil
		}
		if val, ok := src.Get(int64(d.col)); ok {
			d.EmitFloat64(normalizeMissing(dependName, val), slot, ctx)
		}

	case dvKVIntStr:
		src, _ := d.Source(feature.SrcKVIntStr, dependName, in, st).(*feature.KVIntStrSource)
		if src == nil {
			return nil
		}
		if val, ok := src.Get(int64(d.col)); ok {
			d.EmitString(val, slot, ctx)
		}

	case dvStringDict:
		raw := d.DependStringAt(0, in, st)
		if raw == "" {
			return nil
		}
		n := countByte(raw, ',') + 1
		ctx.EnableDedup(n)
		ctx.Reserve(n)
		rest := raw
		for i := 0; i < n; i++ {
			var field string
			field, rest = feature.NextField(rest, ',')
			d.EmitString(field, slot, ctx)
		}

	case dvKVStrFloat:
		src, _ := d.Source(feature.SrcKVStrFloat, dependName, in, st).(*feature.KVStrFloatSource)
		if src == nil {
			return nil
		}
		ctx.EnableDedup(src.Len())
		ctx.Reserve(src.Len())
		keys := src.Keys(st)
		if d.col == 0 {
			for _, k := range keys {
				d.EmitString(k, slot, ctx)
			}
		} else {
			for _, k := range keys {
				v, _ := src.Get(k)
				d.EmitFloat64(v, slot, ctx)
			}
		}

	case dvTags:
		src, _ := d.Source(feature.SrcKVIntFloat, dependName, in, st).(*feature.KVIntFloatSource)
		if src == nil {
			return nil
		}
		keys := src.Keys(st)
		ctx.EnableDedup(len(keys))
		ctx.Reserve(len(keys))
		if d.col == 0 {
			for _, k := range keys {
				d.EmitInt64(k, slot, ctx)
			}
		} else {
			for _, k := range keys {
				v, _ := src.Get(k)
				d.EmitFloat64(v, slot, ctx)
			}
		}

	case dvAddCol:
		raw, ok := d.DependRaw(dependName, in, st)
		if !ok || raw == nil {
			return nil
		}
		switch v := raw.(type) {
		case []string:
			ctx.EnableDedup(len(v))
			ctx.Reserve(len(v))
			for _, x := range v {
				d.EmitString(x, slot, ctx)
			}
		case []float64:
			ctx.EnableDedup(len(v))
			ctx.Reserve(len(v))
			for _, x := range v {
				d.EmitFloat64(x, slot, ctx)
			}
		case []int:
			ctx.EnableDedup(len(v))
			ctx.Reserve(len(v))
			for _, x := range v {
				d.EmitInt64(int64(x), slot, ctx)
			}
		default:
			d.EmitIface(raw, slot, ctx) // 单值，无需去重
		}
	}
	return nil
}

func normalizeMissing(dependName string, val float64) float64 {
	if val != -999.0 {
		return val
	}
	if dependName == "UBDiscountRatio" || dependName == "UBDiscountAvg" {
		return val
	}
	return 0
}

func zeroIfMissing(val float64) float64 {
	if val == -999.0 {
		return 0
	}
	return val
}

func countByte(s string, sep byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			n++
		}
	}
	return n
}
