package feature

import "sync"

// releasable returns a parsed data container to its source pool.
type releasable interface {
	release()
}

const (
	maxPooledEntries = 4096
	sourceMapHint    = 64
)

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
