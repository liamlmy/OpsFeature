package operator

import (
	"fmt"

	"github.com/liamlmy/OpsFeature/feature"
	"github.com/liamlmy/OpsFeature/utils"
)

// Bucket 对应 class=Bucket，两种模式由 args[2] 选择：
//
//	abslisan —— 取绝对值后按阈值表二分搜桶，args[3:] 是升序阈值
//	dummy    —— 整数在枚举表里查下标，args[3:] 是枚举值；未命中落到末桶
//
// args[0] hash_version
// args[1] coeff
// args[2] 模式
// args[3:] 参数表。
type Bucket struct {
	feature.Base

	mode       bucketMode
	thresholds []float64 // abslisan：配置期解析好的阈值表
	enums      []int     // dummy：配置期解析好的枚举表
}

type bucketMode uint8

const (
	bmAbsLisan bucketMode = iota
	bmDummy
)

var bucketModes = map[string]bucketMode{
	"abslisan": bmAbsLisan,
	"dummy":    bmDummy,
}

func (b *Bucket) Init() error {
	if err := b.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := b.RequireDepends(1); err != nil {
		return err
	}
	if err := b.RequireArgs(4); err != nil {
		return err
	}

	name := b.ArgStr(2, "")
	mode, ok := bucketModes[name]
	if !ok {
		return fmt.Errorf("operator: feature %q unknown args[2] bucket mode %q (want abslisan or dummy)",
			b.Name(), name)
	}
	b.mode = mode

	switch mode {
	case bmAbsLisan:
		b.thresholds = make([]float64, 0, len(b.Args())-3)
		for i := 3; i < len(b.Args()); i++ {
			f, err := b.ArgFloat(i)
			if err != nil {
				return err
			}
			b.thresholds = append(b.thresholds, f)
		}
		if len(b.thresholds) == 0 {
			return fmt.Errorf("operator: feature %q abslisan needs at least one threshold in args[3:]", b.Name())
		}
	case bmDummy:
		b.enums = make([]int, 0, len(b.Args())-3)
		for i := 3; i < len(b.Args()); i++ {
			n, err := b.ArgInt(i)
			if err != nil {
				return err
			}
			b.enums = append(b.enums, n)
		}
		if len(b.enums) == 0 {
			return fmt.Errorf("operator: feature %q dummy needs at least one enum value in args[3:]", b.Name())
		}
	}
	return nil
}

func (b *Bucket) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	ctx := feature.NewEmitCtx(st, out, b.AddCol())
	defer b.Flush(ctx)

	dependName := b.Depends()[0]

	switch b.mode {
	case bmAbsLisan:
		v := b.DependFloat64(dependName, in, st)
		v = utils.SetFloatPrecisionNewNew(v, 6)
		v = normalizeMissing(dependName, v)
		if v < 0 {
			v = -v
		}
		bucket := utils.BinarySearch(b.thresholds, v)
		if bucket == -1 {
			return fmt.Errorf("operator: feature %q abslisan got empty threshold table", b.Name())
		}
		b.EmitInt64(bucket, b.SlotID(), ctx)

	case bmDummy:
		v := b.DependInt(0, in, st)
		if v == -999 {
			v = 0
		}
		pos := uint64(len(b.enums))
		for i, e := range b.enums {
			if e == v {
				pos = uint64(i)
				break
			}
		}
		b.EmitUint64(pos, b.SlotID(), ctx)
	}
	return nil
}

// BucketTruncate 对应 class=Bucket_truncate，将整数区间映射为连续位置。
type BucketTruncate struct {
	feature.Base

	ranges []intRange
	total  int64
}

type intRange struct{ low, high int }

func (d *BucketTruncate) Init() error {
	if err := d.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := d.RequireDependsExactly(1); err != nil {
		return err
	}
	if err := d.RequireArgs(4); err != nil {
		return err
	}
	if (len(d.Args())-2)%2 != 0 {
		return fmt.Errorf("operator: feature %q args[2:] must be [low,high] pairs, got %d values",
			d.Name(), len(d.Args())-2)
	}

	d.ranges = make([]intRange, 0, (len(d.Args())-2)/2)
	for i := 2; i < len(d.Args()); i += 2 {
		low, err := d.ArgInt(i)
		if err != nil {
			return err
		}
		high, err := d.ArgInt(i + 1)
		if err != nil {
			return err
		}
		if low > high {
			low, high = high, low
		}
		d.ranges = append(d.ranges, intRange{low: low, high: high})
		d.total += int64(high-low) + 1
	}
	return nil
}

func (d *BucketTruncate) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	ctx := feature.NewEmitCtx(st, out, d.AddCol())
	defer d.Flush(ctx)

	v := d.DependInt(0, in, st)

	var pos int64
	for _, r := range d.ranges {
		if v >= r.low && v <= r.high {
			pos += int64(v - r.low)
			break
		}
		pos += int64(r.high-r.low) + 1
	}
	if pos >= d.total {
		pos = 0
	}

	d.EmitInt64(pos, d.SlotID(), ctx)
	return nil
}
