package operator

import (
	"fmt"
	"math"

	"github.com/liamlmy/OpsFeature/feature"
)

// Scale 对应 class=Scale，对数值执行缩放。
// args[0] hash_version，args[1] coeff，args[2] mode（默认 log）。
type Scale struct {
	feature.Base
	mode scaleMode
}

type scaleMode uint8

const (
	smLog scaleMode = iota
)

var scaleModes = map[string]scaleMode{
	"log": smLog,
}

func (s *Scale) Init() error {
	if err := s.RequireMaxArgs(3); err != nil {
		return err
	}
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDependsExactly(1); err != nil {
		return err
	}
	name := s.ArgStr(2, "log") // 缺省 log，与原实现一致
	mode, ok := scaleModes[name]
	if !ok {
		return fmt.Errorf("operator: feature %q unknown args[2] scale mode %q (want log)", s.Name(), name)
	}
	s.mode = mode
	return nil
}

func (s *Scale) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	ctx := feature.NewEmitCtx(st, out, s.AddCol())
	defer s.Flush(ctx)

	v := s.DependFloat64(s.Depends()[0], in, st)
	switch s.mode {
	case smLog:
		if v < 0 {
			v = 0 // 负值先截断，log 才有定义
		}
		v = math.Log(v + 1.0)
	}
	s.EmitFloat64(v, s.SlotID(), ctx)
	return nil
}
