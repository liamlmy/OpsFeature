package feature

// fidCache caches encoded values for a single feature within one sample.
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

// dedupSet removes repeated values within one multi-value feature.
type dedupSet struct {
	hint int // 调用方在循环前设好的条数，首次建 map 时作容量提示

	f   map[float64]struct{}
	s   map[string]struct{}
	i64 map[int64]struct{}
	u64 map[uint64]struct{}
	ifc map[interface{}]struct{}
}

func (d *dedupSet) reset(hint int) {
	d.hint = hint
	if d.f != nil {
		if len(d.f) > maxKeptBuf {
			d.f = nil
		} else {
			for k := range d.f {
				delete(d.f, k)
			}
		}
	}
	if d.s != nil {
		if len(d.s) > maxKeptBuf {
			d.s = nil
		} else {
			for k := range d.s {
				delete(d.s, k)
			}
		}
	}
	if d.i64 != nil {
		if len(d.i64) > maxKeptBuf {
			d.i64 = nil
		} else {
			for k := range d.i64 {
				delete(d.i64, k)
			}
		}
	}
	if d.u64 != nil {
		if len(d.u64) > maxKeptBuf {
			d.u64 = nil
		} else {
			for k := range d.u64 {
				delete(d.u64, k)
			}
		}
	}
	if d.ifc != nil {
		if len(d.ifc) > maxKeptBuf {
			d.ifc = nil
		} else {
			for k := range d.ifc {
				delete(d.ifc, k)
			}
		}
	}
}

func (d *dedupSet) seen(v value) bool {
	switch v.k {
	case kindFloat64:
		if d.f == nil {
			d.f = make(map[float64]struct{}, d.hint)
		}
		_, ok := d.f[v.f]
		d.f[v.f] = struct{}{}
		return ok
	case kindString:
		if d.s == nil {
			d.s = make(map[string]struct{}, d.hint)
		}
		_, ok := d.s[v.s]
		d.s[v.s] = struct{}{}
		return ok
	case kindInt64:
		if d.i64 == nil {
			d.i64 = make(map[int64]struct{}, d.hint)
		}
		k := int64(v.u)
		_, ok := d.i64[k]
		d.i64[k] = struct{}{}
		return ok
	case kindUint64, kindBool:
		if d.u64 == nil {
			d.u64 = make(map[uint64]struct{}, d.hint)
		}
		_, ok := d.u64[v.u]
		d.u64[v.u] = struct{}{}
		return ok
	default:
		if d.ifc == nil {
			d.ifc = make(map[interface{}]struct{}, d.hint)
		}
		_, ok := d.ifc[v.iface]
		d.ifc[v.iface] = struct{}{}
		return ok
	}
}
