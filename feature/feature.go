package feature

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// =============================================================================
// 常量
// =============================================================================

// hashVersion取值，对应fid生产规则。
const (
	hashVersionCityHash  = 0 // CityHash64(值) 低 48 位 + slot_id 高 16 位
	hashVersionRawValue  = 1 // 原值直接当 fid，无 slot 位
	hashVersionSlotValue = 2 // 原值低 48 位 + slot_id 高 16 位
	hashVersionFloatBits = 3 // float32 位模式运算，无 slot 位
	hashVersionShareSlot = 4 // CityHash64(值的字符串形式) + share_slot 高 16 位
)

const (
	coeffEpsilon = 1e-6
	maxSlotID    = 32768
	minFakeSlot  = 32768
)

type Operator interface {
	Conf() *Base
	Init() error
	Emit(in *Input, st *State, out *[]Fid) error
}

type Base struct {
	name    string
	class   string
	slotID  int
	depends []string
	args    []string

	hasSlotID bool
	addcol    bool
	listCache bool
	strOut    bool

	shareSlot   int
	hashVersion int
	coeff       float64
	isSeq       bool

	argsVariant string
}

func (b *Base) Conf() *Base { return b }

func (b *Base) Name() string { return b.name }

func (b *Base) load(conf map[string]string) error {
	for k, v := range conf {
		var err error
		switch k {
		case "name":
			b.name = v
		case "class":
			b.class = v
		case "slot_id":
			b.slotID, err = atoiField(k, v)
			b.hasSlotID = true
		case "share_slot":
			b.shareSlot, err = atoiField(k, v)
		case "depend":
			b.depends = splitTrim(v)
		case "args":
			b.args = splitTrim(v)
			if len(b.args) > 4 {
				b.argsVariant = strings.Join(b.args[4:], "_")
			}
		case "addcol":
			b.addcol, err = atobField(k, v)
		case "list_cache":
			b.listCache, err = atobField(k, v)
		case "enable_str_out":
			b.strOut, err = atobField(k, v)
		default:
			return fmt.Errorf("operator: unknown feature conf key %q (value %q)", k, v)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Base) validate() error {
	if b.name == "" {
		return fmt.Errorf("operator: feature name is empty (class %q)", b.class)
	}
	if b.class == "" {
		return fmt.Errorf("operator: feature %q has no class", b.name)
	}
	if !b.hasSlotID {
		return fmt.Errorf("operator: feature %q has no slot_id", b.name)
	}
	if b.slotID < 0 || b.slotID > maxSlotID {
		return fmt.Errorf("operator: feature %q slot_id %d out of range [0, %d]",
			b.name, b.slotID, maxSlotID)
	}
	switch b.hashVersion {
	case hashVersionCityHash, hashVersionRawValue, hashVersionSlotValue,
		hashVersionFloatBits:
	case hashVersionShareSlot:
		if b.shareSlot <= 0 || b.shareSlot >= maxSlotID {
			return fmt.Errorf("operator: feature %q hash_version=4 requires share_slot in (1, %d), got %d",
				b.name, maxSlotID, b.shareSlot)
		}
	default:
		return fmt.Errorf("operator: feature %q unknown hash_version %d", b.name, b.hashVersion)
	}
	return nil
}

func (b *Base) parseHashArgs(hvIdx, coeffIdx int) error {
	if v, ok := b.arg(hvIdx); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("operator: feature %q args[%d] hash_version %q is not an int: %v",
				b.name, hvIdx, v, err)
		}
		b.hashVersion = n
	}
	if v, ok := b.arg(coeffIdx); ok && v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("operator: feature %q args[%d] coeff %q is not a float: %v",
				b.name, coeffIdx, v, err)
		}
		b.coeff = f
	}
	return nil
}

func (b *Base) arg(i int) (string, bool) {
	if i < 0 || i >= len(b.args) {
		return "", false
	}
	return b.args[i], true
}

func (b *Base) argInt(i int) (int, error) {
	v, ok := b.arg(i)
	if !ok || v == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("operator: feature %q args[%d] %q is not an int: %v", b.name, i, v, err)
	}
	return n, nil
}

func (b *Base) argFloat(i int) (float64, error) {
	v, ok := b.arg(i)
	if !ok || v == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("operator: feature %q args[%d] %q is not a float: %v", b.name, i, v, err)
	}
	return f, nil
}

func (b *Base) argUint64(i int) (uint64, error) {
	v, ok := b.arg(i)
	if !ok || v == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("operator: feature %q args[%d] %q is not a uint64: %v", b.name, i, v, err)
	}
	return n, nil
}

func (b *Base) argStr(i int, def string) string {
	if v, ok := b.arg(i); ok && v != "" {
		return v
	}
	return def
}

func (b *Base) requireDepends(n int) error {
	if len(b.depends) < n {
		return fmt.Errorf("operator: feature %q (class %s) needs at least %d depend columns, got %d",
			b.name, b.class, n, len(b.depends))
	}
	return nil
}

func (b *Base) requireDependsExactly(n int) error {
	if len(b.depends) != n {
		return fmt.Errorf("operator: feature %q (class %s) needs exactly %d depend columns, got %d",
			b.name, b.class, n, len(b.depends))
	}
	return nil
}

func (b *Base) requireArgs(n int) error {
	if len(b.args) < n {
		return fmt.Errorf("operator: feature %q (class %s) needs at least %d args, got %d",
			b.name, b.class, n, len(b.args))
	}
	return nil
}

type fakeSlotRange struct {
	start int
	end   int
	num   int
}

func (b *Base) initFakeSlots() (fakeSlotRange, error) {
	var r fakeSlotRange
	if err := b.requireArgs(5); err != nil {
		return r, err
	}
	var err error
	if r.start, err = b.argInt(2); err != nil {
		return r, err
	}
	if r.end, err = b.argInt(3); err != nil {
		return r, err
	}
	if r.num, err = b.argInt(4); err != nil {
		return r, err
	}

	if r.num <= 0 {
		return r, fmt.Errorf("operator: feature %q args[4] num must be > 0, got %d", b.name, r.num)
	}
	if r.start < minFakeSlot {
		return r, fmt.Errorf("operator: feature %q args[2] slot_start %d must be >= %d (fake slot range)",
			b.name, r.start, minFakeSlot)
	}
	if r.end-r.start+1 != r.num {
		return r, fmt.Errorf("operator: feature %q fake slot range [%d,%d] holds %d slots but num is %d",
			b.name, r.start, r.end, r.end-r.start+1, r.num)
	}

	b.isSeq = true
	return r, nil
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func atoiField(key, v string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("operator: conf %s=%q is not an int: %v", key, v, err)
	}
	return n, nil
}

func atobField(key, v string) (bool, error) {
	n, err := atoiField(key, v)
	if err != nil {
		return false, err
	}
	return n != 0, nil
}

func logf(format string, args ...interface{}) {
	log.Printf("OpsFeature/operator||"+format, args...)
}
