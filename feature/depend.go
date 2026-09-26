package feature

import (
	"strconv"
)

func (b *Base) lookup(name string, in *Input, st *State) (interface{}, bool) {
	if v, ok := in.get(name); ok {
		return v, true
	}
	if st != nil {
		if v, ok := st.inner[name]; ok { // 读 nil map 合法
			return v, true
		}
	}
	return nil, false
}

func (b *Base) lookupAt(index int, in *Input, st *State) (interface{}, bool) {
	if index < 0 || index >= len(b.depends) {
		return nil, false
	}
	return b.lookup(b.depends[index], in, st)
}

func (b *Base) dependRaw(name string, in *Input, st *State) (interface{}, bool) {
	return b.lookup(name, in, st)
}

func (b *Base) dependFloat64(name string, in *Input, st *State) float64 {
	raw, ok := b.lookup(name, in, st)
	if !ok {
		return 0
	}
	f, ok := toFloat64(raw)
	if !ok {
		logf("depend not convertible to float64, feature=%s class=%s depend=%s type=%T",
			b.name, b.class, name, raw)
		return 0
	}
	return f
}

func (b *Base) dependString(name string, in *Input, st *State) string {
	raw, ok := b.lookup(name, in, st)
	if !ok {
		return ""
	}
	s, ok := toStringScalar(raw)
	if !ok {
		logf("depend not convertible to string, feature=%s class=%s depend=%s type=%T",
			b.name, b.class, name, raw)
		return ""
	}
	return s
}

func (b *Base) dependStringAt(index int, in *Input, st *State) string {
	raw, ok := b.lookupAt(index, in, st)
	if !ok {
		return ""
	}
	s, ok := toStringScalar(raw)
	if !ok {
		logf("depend not convertible to string, feature=%s class=%s index=%d type=%T",
			b.name, b.class, index, raw)
		return ""
	}
	return s
}

func (b *Base) dependInt(index int, in *Input, st *State) int {
	raw, ok := b.lookupAt(index, in, st)
	if !ok {
		return 0
	}
	n, ok := toInt(raw)
	if !ok {
		logf("depend not convertible to int, feature=%s class=%s index=%d type=%T",
			b.name, b.class, index, raw)
		return 0
	}
	return n
}

func (b *Base) dependStringList(name string, in *Input, st *State) []string {
	raw, ok := b.lookup(name, in, st)
	if !ok {
		return nil
	}
	list, ok := toStringList(raw)
	if !ok {
		logf("depend not convertible to []string, feature=%s class=%s depend=%s type=%T",
			b.name, b.class, name, raw)
		return nil
	}
	return list
}

// 类型转换：纯函数，用于单测
func toFloat64(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint64:
		return float64(x), true
	case string:
		f, e := strconv.ParseFloat(x, 64)
		if e == nil {
			return f, true
		} else {
			return 0, false
		}
	case []string:
		if len(x) == 0 {
			return 0, true
		}
		f, e := strconv.ParseFloat(x[0], 64)
		if e == nil {
			return f, true
		} else {
			return 0, false
		}
	default:
		return 0, false
	}
}

func toInt(v interface{}) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	case string:
		n, _ := strconv.Atoi(x) // 吞错理由同 toFloat64
		return n, true
	case []string:
		if len(x) == 0 {
			return 0, true
		}
		n, _ := strconv.Atoi(x[0])
		return n, true
	default:
		return 0, false
	}
}

func toStringScalar(v interface{}) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case int:
		return strconv.Itoa(x), true
	case int32:
		return strconv.FormatInt(int64(x), 10), true
	case int64:
		return strconv.FormatInt(x, 10), true
	case uint64:
		return strconv.FormatUint(x, 10), true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32), true
	case bool:
		if x {
			return "1", true
		}
		return "0", true
	case []string:
		if len(x) == 0 {
			return "", true
		}
		return x[0], true
	default:
		return "", false
	}
}

func toStringList(v interface{}) ([]string, bool) {
	switch x := v.(type) {
	case []string:
		return x, true
	case []interface{}:
		res := make([]string, 0, len(x))
		for _, item := range x {
			s, ok := toStringScalar(item)
			if !ok {
				return nil, false
			}
			res = append(res, s)
		}
		return res, true
	default:
		// 单值视为长度 1 的列表，与原实现一致
		if s, ok := toStringScalar(v); ok {
			return []string{s}, true
		}
		return nil, false
	}
}
