package operator

import (
	"fmt"
	"math"

	"github.com/liamlmy/OpsFeature/feature"
	"github.com/liamlmy/OpsFeature/utils"
)

// Arithmetic 对应 class=Arithmetic：从两个依赖列算出一个派生数值。
//
// args[0] operation — add / minus / multiply / match / div / cos
// args[1] hash_version
// args[2] coeff
type Arithmetic struct {
	feature.Base

	op arithmeticOp
}

type arithmeticOp uint8

const (
	aoAdd      arithmeticOp = iota // x + y
	aoMinus                        // x - y
	aoMultiply                     // x * y
	aoMatch                        // 一致性：都空=2，相等=1，不等=0
	aoDiv                          // x / y
	aoCos                          // 两个向量的余弦相似度
)

var arithmeticOps = map[string]arithmeticOp{
	"add":      aoAdd,
	"minus":    aoMinus,
	"multiply": aoMultiply,
	"match":    aoMatch,
	"div":      aoDiv,
	"cos":      aoCos,
}

func (a *Arithmetic) Init() error {
	if err := a.ParseHashArgs(1, 2); err != nil {
		return err
	}
	if err := a.RequireDepends(2); err != nil {
		return err
	}

	name := a.ArgStr(0, "")
	op, ok := arithmeticOps[name]
	if !ok {
		return fmt.Errorf("operator: feature %q unknown args[0] operation %q", a.Name(), name)
	}
	a.op = op
	return nil
}

func (a *Arithmetic) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	ctx := feature.NewEmitCtx(st, out, a.AddCol())
	defer a.Flush(ctx)

	left, right := a.Depends()[0], a.Depends()[1]

	var val float64
	switch a.op {

	case aoAdd:
		x := zeroIfMissing(a.DependFloat64(left, in, st))
		y := zeroIfMissing(a.DependFloat64(right, in, st))
		val = x + y

	case aoMinus:
		x := zeroIfMissing(a.DependFloat64(left, in, st))
		y := zeroIfMissing(a.DependFloat64(right, in, st))
		val = x - y

	case aoMultiply:
		x := zeroIfMissing(a.DependFloat64(left, in, st))
		y := zeroIfMissing(a.DependFloat64(right, in, st))
		val = x * y

	case aoMatch:
		x := a.DependString(left, in, st)
		y := a.DependString(right, in, st)
		switch {
		case x == "" && y == "":
			val = 2
		case x == y:
			val = 1
		default:
			val = 0
		}

	case aoDiv:
		x := a.DependFloat64(left, in, st)
		y := a.DependFloat64(right, in, st)
		if utils.Equal(x, 0.0) || utils.Equal(y, 0.0) {
			val = 0
		} else {
			val = x / y
		}

	case aoCos:
		v1, _ := a.Source(feature.SrcVector, left, in, st).(*feature.VectorSource)
		v2, _ := a.Source(feature.SrcVector, right, in, st).(*feature.VectorSource)
		if v1 == nil || v2 == nil {
			return nil
		}
		cos, ok := cosine(v1.Data(), v2.Data())
		if !ok {
			feature.Logf("cosine skipped, feature=%s class=%s len(%s)=%d len(%s)=%d",
				a.Name(), a.Class(), left, len(v1.Data()), right, len(v2.Data()))
			return nil
		}
		val = cos
	}

	a.EmitFloat64(val, a.SlotID(), ctx)
	return nil
}

func cosine(a, b []float64) (float64, bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	mod := math.Sqrt(na) * math.Sqrt(nb)
	if utils.Equal(mod, 0.0) {
		return 0, true
	}
	return dot / mod, true
}
