package feature

import (
	"fmt"
	"math"

	"github.com/liamlmy/OpsFeature/utils"
)

// Fid is a feature ID and its output slot.
type Fid struct {
	Slot int    // 输出槽位
	Val  uint64 // 特征 id
}

// emitCtx collects intermediate values and FIDs for one operator.
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
	if ctx.st.err != nil {
		return
	}
	if !v.comparable() {
		ctx.st.fail(fmt.Errorf("feature %q: unsupported non-comparable value %T", b.name, v.iface))
		return
	}
	if !b.isSeq && ctx.dedup != nil {
		if ctx.dedup.seen(v) {
			return
		}
	}

	if b.addcol {
		if s, err := v.toString(); err == nil {
			ctx.acc = append(ctx.acc, s)
		} else {
			ctx.st.fail(fmt.Errorf("feature %q addcol: %w", b.name, err))
			return
		}
	}

	if b.slotID == 0 || b.strOut {
		return
	}

	var fid uint64
	need := true
	if b.listCache {
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
			ctx.st.fail(fmt.Errorf("feature %q FID: %w", b.name, err))
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

	*ctx.out = append(*ctx.out, Fid{Slot: outSlot, Val: fid})
}

func (b *Base) computeFid(v value) (uint64, error) {
	switch b.hashVersion {
	case hashVersionRawValue, hashVersionSlotValue:
		val, err := v.toUint64(b.scale)
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
