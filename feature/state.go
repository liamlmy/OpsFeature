package feature

import "sort"

// State holds intermediate values and caches for one sample.
// Call Reset before reusing it for a different sample; do not share it concurrently.
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
