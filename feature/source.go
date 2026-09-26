package feature

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/liamlmy/OpsFeature/utils"
)

var errEmptyDepend = errors.New("operator: depend column is empty")

type sourceKind uint8

const (
	srcKVIntFloat   sourceKind = iota // "id:数值,id:数值"     -> map[int64]float64
	srcKVIntStr                       // "id:字符串,id:字符串" -> map[int64]string
	srcKVStrFloat                     // "字符串:数值,..."     -> map[string]float64
	srcSession                        // "1_2_3"               -> []uint64
	srcVector                         // "0.1,0.2,0.3"         -> []float64
	srcSessionField                   // "0:1_2,1:3_4"         -> []interface{}
)

type sourceKey struct {
	depend  string
	kind    sourceKind
	variant string
}

type Source interface {
	parse(b *Base, depend string, in *Input, st *State) error
}

type releasable interface {
	release()
}

const (
	maxPooledEntries = 4096
	sourceMapHint    = 64
)

func (b *Base) source(kind sourceKind, depend string, in *Input, st *State) Source {
	key := sourceKey{depend: depend, kind: kind}
	if kind == srcSessionField {
		key.variant = b.argsVariant
	}
	if s, ok := st.sources[key]; ok {
		return s
	}

	s := newSource(kind)
	if err := s.parse(b, depend, in, st); err != nil {
		if r, ok := s.(releasable); ok {
			r.release()
		}
		return nil
	}
	st.sources[key] = s
	return s
}

func newSource(kind sourceKind) Source {
	switch kind {
	case srcKVIntFloat:
		return getKVIntFloat()
	case srcKVIntStr:
		return getKVIntStr()
	case srcKVStrFloat:
		return getKVStrFloat()
	case srcSessionField:
		return getSessionField()
	case srcSession:
		return getSession()
	case srcVector:
		return getVector()
	default:
		return nil
	}
}

type kvIntFloatSource struct {
	data map[int64]float64
}

func (s *kvIntFloatSource) parse(b *Base, depend string, in *Input, st *State) error {
	raw := b.dependString(depend, in, st)
	if raw == "" {
		return errEmptyDepend
	}
	fillUint64Float(raw, s.data)
	return nil
}

func (s *kvIntFloatSource) Get(col int64) (float64, bool) {
	v, ok := s.data[col]
	return v, ok
}

func (s *kvIntFloatSource) Keys(st *State) []int64 {
	return sortedInt64Keys(s.data, st)
}

type kvIntStrSource struct {
	data map[int64]string
}

func (s *kvIntStrSource) parse(b *Base, depend string, in *Input, st *State) error {
	raw := b.dependString(depend, in, st)
	if raw == "" {
		return errEmptyDepend
	}
	fillUint64String(raw, s.data)
	return nil
}

func (s *kvIntStrSource) Get(col int64) (string, bool) {
	v, ok := s.data[col]
	return v, ok
}

type kvStrFloatSource struct {
	data map[string]float64
}

func (s *kvStrFloatSource) parse(b *Base, depend string, in *Input, st *State) error {
	raw := b.dependString(depend, in, st)
	if raw == "" {
		return errEmptyDepend
	}
	fillStringFloat(raw, s.data)
	return nil
}

func (s *kvStrFloatSource) Keys(st *State) []string {
	return sortedStringKeys(s.data, st)
}

func (s *kvStrFloatSource) Get(k string) (float64, bool) {
	v, ok := s.data[k]
	return v, ok
}

func (s *kvStrFloatSource) Len() int { return len(s.data) }

type sessionSource struct {
	data []uint64
}

func (s *sessionSource) parse(b *Base, depend string, in *Input, st *State) error {
	raw := b.dependString(depend, in, st)
	padTo, err := b.argInt(4)
	if err != nil {
		return err
	}
	padVal, err := b.argUint64(5)
	if err != nil {
		return err
	}
	s.data = utils.ParseArrayIntNew(raw, "_", padTo, padVal)
	return nil
}

type vectorSource struct {
	data []float64
}

func (s *vectorSource) parse(b *Base, depend string, in *Input, st *State) error {
	raw := b.dependString(depend, in, st)
	padTo, err := b.argInt(5)
	if err != nil {
		return err
	}
	padVal, err := b.argFloat(6)
	if err != nil {
		return err
	}
	s.data = utils.ParseArrayNew(raw, ",", padTo, padVal)
	return nil
}

type sessionFieldSource struct {
	data []interface{}
}

func (s *sessionFieldSource) parse(b *Base, depend string, in *Input, st *State) error {
	raw := b.dependString(depend, in, st)

	fieldIndex := b.argStr(5, "0")
	isStr := b.argStr(6, "int") == "str"

	padTo := 0
	if b.argStr(7, "") == "1" {
		n, err := b.argInt(4)
		if err != nil {
			return err
		}
		padTo = n
	}
	defaultStr := b.argStr(8, "")

	var padVal interface{} = defaultStr
	if !isStr {
		var d uint64
		if defaultStr != "" {
			v, err := strconv.ParseUint(defaultStr, 10, 64)
			if err != nil {
				return fmt.Errorf("operator: feature %q args[8] padding default %q is not a uint64: %v",
					b.name, defaultStr, err)
			}
			d = v
		}
		padVal = d
	}

	var seq string
	rest := raw
	for len(rest) > 0 {
		var field string
		field, rest = nextField(rest, ',')
		k, v, ok := splitKVFull(field)
		if !ok {
			continue
		}
		if k == fieldIndex {
			seq = v
			break
		}
	}
	if seq == "" {
		return nil
	}

	for len(seq) > 0 {
		var elem string
		elem, seq = nextField(seq, '_')
		if isStr {
			s.data = append(s.data, elem)
		} else {
			v, err := strconv.ParseUint(elem, 10, 64)
			switch {
			case err == nil:
				s.data = append(s.data, v)
			case padTo > 0:
				s.data = append(s.data, padVal)
			default:
				continue
			}
		}
		if padTo > 0 && len(s.data) == padTo {
			break
		}
	}
	for i := len(s.data); i < padTo; i++ {
		s.data = append(s.data, padVal)
	}
	return nil
}

var (
	kvIntFloatPool = sync.Pool{New: func() interface{} { return &kvIntFloatSource{} }}
	kvIntStrPool   = sync.Pool{New: func() interface{} { return &kvIntStrSource{} }}
	kvStrFloatPool = sync.Pool{New: func() interface{} { return &kvStrFloatSource{} }}
)

func getKVIntFloat() *kvIntFloatSource {
	s := kvIntFloatPool.Get().(*kvIntFloatSource)
	if s.data == nil {
		s.data = make(map[int64]float64, sourceMapHint)
	} else {
		for k := range s.data {
			delete(s.data, k)
		}
	}
	return s
}

func (s *kvIntFloatSource) release() {
	if len(s.data) > maxPooledEntries {
		s.data = nil
	}
	kvIntFloatPool.Put(s)
}

func getKVIntStr() *kvIntStrSource {
	s := kvIntStrPool.Get().(*kvIntStrSource)
	if s.data == nil {
		s.data = make(map[int64]string, sourceMapHint)
	} else {
		for k := range s.data {
			delete(s.data, k)
		}
	}
	return s
}

func (s *kvIntStrSource) release() {
	if len(s.data) > maxPooledEntries {
		s.data = nil
	}
	kvIntStrPool.Put(s)
}

func getKVStrFloat() *kvStrFloatSource {
	s := kvStrFloatPool.Get().(*kvStrFloatSource)
	if s.data == nil {
		s.data = make(map[string]float64, sourceMapHint)
	} else {
		for k := range s.data {
			delete(s.data, k)
		}
	}
	return s
}

func (s *kvStrFloatSource) release() {
	if len(s.data) > maxPooledEntries {
		s.data = nil
	}
	kvStrFloatPool.Put(s)
}

var sessionFieldPool = sync.Pool{New: func() interface{} { return &sessionFieldSource{} }}

var (
	sessionPool = sync.Pool{New: func() interface{} { return &sessionSource{} }}
	vectorPool  = sync.Pool{New: func() interface{} { return &vectorSource{} }}
)

func getSession() *sessionSource { return sessionPool.Get().(*sessionSource) }

func (s *sessionSource) release() {
	if cap(s.data) > maxPooledEntries {
		s.data = nil
	}
	sessionPool.Put(s)
}

func getVector() *vectorSource { return vectorPool.Get().(*vectorSource) }

func (s *vectorSource) release() {
	if cap(s.data) > maxPooledEntries {
		s.data = nil
	}
	vectorPool.Put(s)
}

func getSessionField() *sessionFieldSource {
	s := sessionFieldPool.Get().(*sessionFieldSource)
	s.data = s.data[:0]
	return s
}

func (s *sessionFieldSource) release() {
	if cap(s.data) > maxPooledEntries {
		s.data = nil
	} else {
		for i := range s.data {
			s.data[i] = nil
		}
		s.data = s.data[:0]
	}
	sessionFieldPool.Put(s)
}

func nextField(s string, sep byte) (field, rest string) {
	if i := strings.IndexByte(s, sep); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func splitKV(field string) (k, v string, ok bool) {
	i := strings.IndexByte(field, ':')
	if i < 0 {
		return "", "", false
	}
	k, v = field[:i], field[i+1:]
	if j := strings.IndexByte(v, ':'); j >= 0 {
		v = v[:j]
	}
	return k, v, true
}

func splitKVFull(field string) (k, v string, ok bool) {
	i := strings.IndexByte(field, ':')
	if i < 0 {
		return "", "", false
	}
	return field[:i], field[i+1:], true
}

func fillUint64Float(raw string, m map[int64]float64) {
	var field string
	for len(raw) > 0 {
		field, raw = nextField(raw, ',')
		ks, vs, ok := splitKV(field)
		if !ok {
			continue
		}
		k, err := strconv.ParseInt(ks, 10, 64)
		if err != nil {
			continue // 单条解析失败跳过，不影响其余条目
		}
		v, err := strconv.ParseFloat(vs, 64)
		if err != nil {
			continue
		}
		m[k] = v
	}
}

func fillUint64String(raw string, m map[int64]string) {
	var field string
	for len(raw) > 0 {
		field, raw = nextField(raw, ',')
		ks, vs, ok := splitKV(field)
		if !ok {
			continue
		}
		k, err := strconv.ParseInt(ks, 10, 64)
		if err != nil {
			continue
		}
		m[k] = vs
	}
}

func fillStringFloat(raw string, m map[string]float64) {
	var field string
	for len(raw) > 0 {
		field, raw = nextField(raw, ',')
		ks, vs, ok := splitKV(field)
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(vs, 64)
		if err != nil {
			continue
		}
		m[ks] = v
	}
}
