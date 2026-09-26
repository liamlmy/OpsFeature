package operator

import (
	"fmt"
	"math"
	"strings"

	"github.com/liamlmy/OpsFeature/feature"
)

// Combine 同时实现 class=Combine 与 class=Combine_multi，阶数由构造时传入的 arity 决定。
//
// args[0] 必须是字面量 "addcol"
// args[1] hash_version
// args[2] coeff
type Combine struct {
	feature.Base
	arity int // 2 或 3
}

func newCombine(arity int) func() feature.Operator {
	return func() feature.Operator { return &Combine{arity: arity} }
}

func (c *Combine) Init() error {
	if err := c.ParseHashArgs(1, 2); err != nil {
		return err
	}
	if err := c.RequireDependsExactly(c.arity); err != nil {
		return err
	}
	if got := c.ArgStr(0, ""); got != "addcol" {
		return fmt.Errorf("operator: feature %q args[0] must be \"addcol\", got %q", c.Name(), got)
	}
	return nil
}

func (c *Combine) Emit(in *feature.Input, st *feature.State, out *[]feature.Fid) error {
	ctx := feature.NewEmitCtx(st, out, c.AddCol())
	defer c.Flush(ctx)

	var buf [3][]string
	lists := buf[:0]
	total := 1
	for i := 0; i < c.arity; i++ {
		l := c.DependStringList(c.Depends()[i], in, st)
		if len(l) == 0 {
			return nil
		}
		lists = append(lists, l)
		total *= len(l)
	}

	ctx.EnableDedup(total)
	ctx.Reserve(total)

	switch c.arity {
	case 2:
		for _, a := range lists[0] {
			for _, b := range lists[1] {
				c.EmitString(join2(a, b), c.SlotID(), ctx)
			}
		}
	case 3:
		for _, a := range lists[0] {
			for _, b := range lists[1] {
				for _, d := range lists[2] {
					c.EmitString(join3(a, b, d), c.SlotID(), ctx)
				}
			}
		}
	}
	return nil
}

func join2(a, b string) string {
	var sb strings.Builder
	sb.Grow(len(a) + 1 + len(b))
	sb.WriteString(a)
	sb.WriteByte('_')
	sb.WriteString(b)
	return sb.String()
}

func join3(a, b, c string) string {
	var sb strings.Builder
	sb.Grow(len(a) + len(b) + len(c) + 2)
	sb.WriteString(a)
	sb.WriteByte('_')
	sb.WriteString(b)
	sb.WriteByte('_')
	sb.WriteString(c)
	return sb.String()
}

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
	if err := s.ParseHashArgs(0, 1); err != nil {
		return err
	}
	if err := s.RequireDepends(1); err != nil {
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
