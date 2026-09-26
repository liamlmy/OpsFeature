package feature

// Base configuration accessors used by operator implementations.
func (b *Base) Class() string     { return b.class }
func (b *Base) SlotID() int       { return b.slotID }
func (b *Base) Depends() []string { return b.depends }
func (b *Base) Args() []string    { return b.args }
func (b *Base) AddCol() bool      { return b.addcol }
func (b *Base) ParseHashArgs(hashVersionIndex, coeffIndex int) error {
	return b.parseHashArgs(hashVersionIndex, coeffIndex)
}
func (b *Base) ArgInt(index int) (int, error)       { return b.argInt(index) }
func (b *Base) ArgFloat(index int) (float64, error) { return b.argFloat(index) }
func (b *Base) ArgUint64(index int) (uint64, error) { return b.argUint64(index) }
func (b *Base) ArgStr(index int, defaultValue string) string {
	return b.argStr(index, defaultValue)
}
func (b *Base) RequireDepends(count int) error { return b.requireDepends(count) }
func (b *Base) RequireDependsExactly(count int) error {
	return b.requireDependsExactly(count)
}
func (b *Base) RequireArgs(count int) error { return b.requireArgs(count) }

type FakeSlotRange = fakeSlotRange

func (r fakeSlotRange) Start() int { return r.start }
func (r fakeSlotRange) End() int   { return r.end }
func (r fakeSlotRange) Num() int   { return r.num }
func (b *Base) InitFakeSlots() (FakeSlotRange, error) {
	return b.initFakeSlots()
}

func (b *Base) DependRaw(name string, in *Input, st *State) (interface{}, bool) {
	return b.dependRaw(name, in, st)
}
func (b *Base) DependFloat64(name string, in *Input, st *State) float64 {
	return b.dependFloat64(name, in, st)
}
func (b *Base) DependString(name string, in *Input, st *State) string {
	return b.dependString(name, in, st)
}
func (b *Base) DependStringAt(index int, in *Input, st *State) string {
	return b.dependStringAt(index, in, st)
}
func (b *Base) DependInt(index int, in *Input, st *State) int {
	return b.dependInt(index, in, st)
}
func (b *Base) DependStringList(name string, in *Input, st *State) []string {
	return b.dependStringList(name, in, st)
}

type SourceKind = sourceKind

const (
	SrcKVIntFloat   = srcKVIntFloat
	SrcKVIntStr     = srcKVIntStr
	SrcKVStrFloat   = srcKVStrFloat
	SrcSession      = srcSession
	SrcVector       = srcVector
	SrcSessionField = srcSessionField
)

type KVIntFloatSource = kvIntFloatSource
type KVIntStrSource = kvIntStrSource
type KVStrFloatSource = kvStrFloatSource
type SessionSource = sessionSource
type VectorSource = vectorSource
type SessionFieldSource = sessionFieldSource

func (b *Base) Source(kind SourceKind, depend string, in *Input, st *State) Source {
	return b.source(kind, depend, in, st)
}
func (s *sessionSource) Data() []uint64           { return s.data }
func (s *vectorSource) Data() []float64           { return s.data }
func (s *sessionFieldSource) Data() []interface{} { return s.data }

type EmitCtx = emitCtx

func NewEmitCtx(st *State, out *[]Fid, addCol bool) *EmitCtx {
	return &emitCtx{st: st, out: out, addcol: addCol}
}
func (c *emitCtx) EnableDedup(count int) { c.enableDedup(count) }
func (c *emitCtx) Reserve(count int)     { c.reserve(count) }

func (b *Base) EmitFloat64(value float64, slot int, ctx *EmitCtx) {
	b.emitFloat64(value, slot, ctx)
}
func (b *Base) EmitString(value string, slot int, ctx *EmitCtx) {
	b.emitString(value, slot, ctx)
}
func (b *Base) EmitInt64(value int64, slot int, ctx *EmitCtx) {
	b.emitInt64(value, slot, ctx)
}
func (b *Base) EmitUint64(value uint64, slot int, ctx *EmitCtx) {
	b.emitUint64(value, slot, ctx)
}
func (b *Base) EmitIface(value interface{}, slot int, ctx *EmitCtx) {
	b.emitIface(value, slot, ctx)
}
func (b *Base) Flush(ctx *EmitCtx) { b.flush(ctx) }

func NextField(value string, separator byte) (field, rest string) {
	return nextField(value, separator)
}
func ToFloat64(value interface{}) (float64, bool) { return toFloat64(value) }
func Logf(format string, args ...interface{})     { logf(format, args...) }
