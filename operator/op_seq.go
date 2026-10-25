package operator

import (
	"fmt"
	"math"

	"github.com/liamlmy/OpsFeature/feature"
)

// Seq 对应 class=Seq。
//
// args[0] hash_version
// args[1] coeff
// args[2] slot_start —— fake slot 起始，必须 >= minFakeSlot
// args[3] slot_end   —— fake slot 结束
// args[4] num        —— 序列长度，必须等于 slot_end-slot_start+1，同时作 padding 长度
// args[5] padding 默认值（可选）
type Seq struct {
	feature.Base
	slots feature.FakeSlotRange
}

func (s *Seq) Init() error {
	if err := s.RequireMaxArgs(6); err != nil {
		return err
	}
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDependsExactly(1); err != nil {
		return err
	}
	var err error
	s.slots, err = s.InitFakeSlots()
	if err != nil {
		return err
	}
	_, err = s.ArgUint64(5)
	return err
}

func (s *Seq) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	src, _ := s.Source(feature.SrcSession, s.Depends()[0], in, st).(*feature.SessionSource)
	if src == nil {
		return nil
	}

	ctx := feature.NewEmitCtx(st, out, s.AddCol())
	defer s.Flush(ctx)

	data := src.Data()
	n := clampSeqLen(&s.Base, len(data), s.slots.Num())
	ctx.Reserve(n)
	for i := 0; i < n; i++ {
		s.EmitUint64(data[i], s.slots.Start()+i, ctx)
	}
	return nil
}

// SeqField 对应 class=Seq_field。
type SeqField struct {
	feature.Base
	slots   feature.FakeSlotRange
	padding float64 // Output-space default; never interpret missing values as timestamps.
}

func (s *SeqField) Init() error {
	if err := s.RequireMaxArgs(9); err != nil {
		return err
	}
	if len(s.Depends()) > 2 {
		return fmt.Errorf("operator: feature %q needs one or two depend columns", s.Name())
	}
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDepends(1); err != nil {
		return err
	}
	var err error
	s.slots, err = s.InitFakeSlots()
	if err != nil {
		return err
	}
	element := s.ArgStr(6, "int")
	if element != "int" && element != "str" {
		return fmt.Errorf("operator: feature %q invalid field element type %q", s.Name(), element)
	}
	pad, err := s.ArgInt(7)
	if err != nil {
		return err
	}
	if pad != 0 && pad != 1 {
		return fmt.Errorf("operator: feature %q padding flag must be 0 or 1", s.Name())
	}
	if element == "int" {
		_, err = s.ArgUint64(8)
	}
	if err != nil {
		return err
	}
	if len(s.Depends()) == 2 {
		s.padding, err = s.ArgFloat(8)
	}
	return err
}

func (s *SeqField) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	src, _ := s.Source(feature.SrcSessionField, s.Depends()[0], in, st).(*feature.SessionFieldSource)
	if src == nil {
		return nil
	}

	ctx := feature.NewEmitCtx(st, out, s.AddCol())
	defer s.Flush(ctx)

	hasBase := len(s.Depends()) == 2
	var baseTs float64
	baseReady := false
	data := src.Data()
	n := clampSeqLen(&s.Base, len(data), s.slots.Num())
	ctx.Reserve(n)
	for i := 0; i < n; i++ {
		slot := s.slots.Start() + i
		raw := data[i]
		if !hasBase {
			s.EmitIface(raw, slot, ctx)
			continue
		}
		if !src.Valid(i) {
			s.EmitFloat64(s.padding, slot, ctx)
			continue
		}
		f, ok := feature.ToFloat64(raw)
		if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
			s.EmitFloat64(s.padding, slot, ctx)
			continue
		}
		if !baseReady {
			base, exists := s.DependRaw(s.Depends()[1], in, st)
			baseTs, ok = feature.ToFloat64(base)
			if !exists || !ok || math.IsNaN(baseTs) || math.IsInf(baseTs, 0) {
				return fmt.Errorf("operator: feature %q base timestamp must be finite numeric", s.Name())
			}
			baseReady = true
		}
		delta := (baseTs - f) / 86400
		if math.IsNaN(delta) || math.IsInf(delta, 0) {
			return fmt.Errorf("operator: feature %q timestamp difference overflow", s.Name())
		}
		s.EmitFloat64(delta, slot, ctx)
	}
	return nil
}

// SeqEmb 对应 class=Seq_emb。
type SeqEmb struct {
	feature.Base
	slots feature.FakeSlotRange
}

func (s *SeqEmb) Init() error {
	if err := s.RequireMaxArgs(7); err != nil {
		return err
	}
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDependsExactly(1); err != nil {
		return err
	}
	var err error
	if s.slots, err = s.InitFakeSlots(); err != nil {
		return err
	}

	padTo, err := s.ArgInt(5)
	if err != nil {
		return err
	}
	if padTo != 0 && padTo != s.slots.Num() {
		return fmt.Errorf("operator: feature %q args[5] vector length %d must equal args[4] num %d, "+
			"otherwise fake slot would run past slot_end %d",
			s.Name(), padTo, s.slots.Num(), s.slots.End())
	}
	_, err = s.ArgFloat(6)
	return err
}

func (s *SeqEmb) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	src, _ := s.Source(feature.SrcVector, s.Depends()[0], in, st).(*feature.VectorSource)
	if src == nil {
		return nil
	}

	ctx := feature.NewEmitCtx(st, out, s.AddCol())
	defer s.Flush(ctx)

	data := src.Data()
	n := clampSeqLen(&s.Base, len(data), s.slots.Num())
	ctx.Reserve(n)
	for i := 0; i < n; i++ {
		s.EmitFloat64(data[i], s.slots.Start()+i, ctx)
	}
	return nil
}

func clampSeqLen(b *feature.Base, got, num int) int {
	if got <= num {
		return got
	}
	feature.Logf("sequence longer than num, truncated, feature=%s class=%s got=%d num=%d",
		b.Name(), b.Class(), got, num)
	return num
}
