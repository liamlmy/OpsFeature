package feature

import "sort"

// 本文件定义抽取过程中流转的三个数据结构体：
//   Input —— 条件样本的只读输入
//   State —— 条件样本的可变中间态（链式依赖 / 缓存），跨算子复用
//   Fid   —— 条件抽取结果

type Fid struct {
	Slot int    // 输出槽位
	Val  uint64 // 特征 id
}

type Input struct {
	Global map[string]interface{} // 请求级公共特征
	Score  map[string]interface{} // 当前候选的特征
}

func (in *Input) get(name string) (interface{}, bool) {
	if in == nil {
		return nil, false
	}
	if v, ok := in.Global[name]; ok {
		return v, true
	}
	if v, ok := in.Score[name]; ok {
		return v, true
	}
	return nil, false
}

type State struct {
	inner   map[string]interface{}
	fids    map[string]*fidCache
	sources map[sourceKey]Source
	dedup   dedupSet

	keyBuf    []string
	keyBufI64 []int64
}

func NewState() *State {
	s := &State{}
	s.ensure()
	return s
}

func (s *State) ensure() {
	if s.inner == nil {
		s.inner = make(map[string]interface{}, 32)
	}
	if s.fids == nil {
		s.fids = make(map[string]*fidCache, 8)
	}
	if s.sources == nil {
		s.sources = make(map[sourceKey]Source, 8)
	}
}

// maxKeptBuf 是跨样本保留的缓冲区容量上限。
const maxKeptBuf = 4096

func (s *State) Reset() {
	if s == nil {
		return
	}
	for k := range s.inner {
		delete(s.inner, k)
	}
	for k := range s.fids {
		delete(s.fids, k)
	}

	for _, v := range s.sources {
		if r, ok := v.(releasable); ok {
			r.release()
		}
	}
	for k := range s.sources {
		delete(s.sources, k)
	}

	if cap(s.keyBuf) > maxKeptBuf {
		s.keyBuf = nil
	}
	if cap(s.keyBufI64) > maxKeptBuf {
		s.keyBufI64 = nil
	}
}

func sortedStringKeys(m map[string]float64, s *State) []string {
	buf := s.keyBuf[:0]
	for k := range m {
		buf = append(buf, k)
	}
	s.keyBuf = buf
	sortStrings(buf)
	return buf
}

func sortedInt64Keys(m map[int64]float64, s *State) []int64 {
	buf := s.keyBufI64[:0]
	for k := range m {
		buf = append(buf, k)
	}
	s.keyBufI64 = buf
	sortInt64s(buf)
	return buf
}

const insertionSortMax = 24

func sortStrings(a []string) {
	if len(a) > insertionSortMax {
		sort.Strings(a)
		return
	}
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

func sortInt64s(a []int64) {
	if len(a) > insertionSortMax {
		sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
		return
	}
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

type fidCache struct {
	byFloat  map[float64]uint64
	byString map[string]uint64
	byInt64  map[int64]uint64
	byUint64 map[uint64]uint64
	byBool   map[bool]uint64
	byIface  map[interface{}]uint64
}

func (c *fidCache) get(v value) (uint64, bool) {
	switch v.k {
	case kindFloat64:
		fid, ok := c.byFloat[v.f]
		return fid, ok
	case kindString:
		fid, ok := c.byString[v.s]
		return fid, ok
	case kindInt64:
		fid, ok := c.byInt64[int64(v.u)]
		return fid, ok
	case kindUint64:
		fid, ok := c.byUint64[v.u]
		return fid, ok
	case kindBool:
		fid, ok := c.byBool[v.u == 1]
		return fid, ok
	default:
		fid, ok := c.byIface[v.iface]
		return fid, ok
	}
}

func (c *fidCache) put(v value, fid uint64) {
	switch v.k {
	case kindFloat64:
		if c.byFloat == nil {
			c.byFloat = make(map[float64]uint64)
		}
		c.byFloat[v.f] = fid
	case kindString:
		if c.byString == nil {
			c.byString = make(map[string]uint64)
		}
		c.byString[v.s] = fid
	case kindInt64:
		if c.byInt64 == nil {
			c.byInt64 = make(map[int64]uint64)
		}
		c.byInt64[int64(v.u)] = fid
	case kindUint64:
		if c.byUint64 == nil {
			c.byUint64 = make(map[uint64]uint64)
		}
		c.byUint64[v.u] = fid
	case kindBool:
		if c.byBool == nil {
			c.byBool = make(map[bool]uint64)
		}
		c.byBool[v.u == 1] = fid
	default:
		if c.byIface == nil {
			c.byIface = make(map[interface{}]uint64)
		}
		c.byIface[v.iface] = fid
	}
}
