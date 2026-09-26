package operator

import (
	"fmt"

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
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDepends(1); err != nil {
		return err
	}
	var err error
	s.slots, err = s.InitFakeSlots()
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
	slots feature.FakeSlotRange
}

func (s *SeqField) Init() error {
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDepends(1); err != nil {
		return err
	}
	var err error
	s.slots, err = s.InitFakeSlots()
	return err
}

func (s *SeqField) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	src, _ := s.Source(feature.SrcSessionField, s.Depends()[0], in, st).(*feature.SessionFieldSource)
	if src == nil {
		return nil
	}

	ctx := feature.NewEmitCtx(st, out, s.AddCol())
	defer s.Flush(ctx)

	hasBase := len(s.Depends()) >= 2
	var baseTs float64
	if hasBase {
		baseTs = s.DependFloat64(s.Depends()[1], in, st)
	}

	data := src.Data()
	n := clampSeqLen(&s.Base, len(data), s.slots.Num())
	ctx.Reserve(n)

	fakeSlot := s.slots.Start()
	for i := 0; i < n; i++ {
		raw := data[i]
		if !hasBase {
			s.EmitIface(raw, fakeSlot, ctx)
			fakeSlot++
			continue
		}
		f, ok := feature.ToFloat64(raw)
		if !ok {
			continue
		}
		s.EmitFloat64((baseTs-f)/86400, fakeSlot, ctx)
		fakeSlot++
	}
	return nil
}

// SeqEmb 对应 class=Seq_emb。
type SeqEmb struct {
	feature.Base
	slots feature.FakeSlotRange
}

func (s *SeqEmb) Init() error {
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDepends(1); err != nil {
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
	return nil
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
