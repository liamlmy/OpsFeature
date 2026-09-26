package utils

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"unsafe"
)

const (
	k0 = uint64(0xc3a5c85c97cb3127)
	k1 = uint64(0xb492b66fbe98f273)
	k2 = uint64(0x9ae16a3b2f90404f)
)

const (
	c1 = uint32(0xcc9e2d51)
	c2 = uint32(0x1b873593)
)

func CombineSlotAndVal(slotID int, x uint64) uint64 {
	hashLow48 := x & 0xFFFFFFFFFFFF // 取低48位
	fid := (uint64(slotID) << 48) | hashLow48
	return fid
}

func CombineSlotAndHash(slotID int, v interface{}) uint64 {
	return CombineSlotWithHash(slotID, CityHash64Any(v))
}

// 下面这组 typed 函数是 CityHash64Any 各分支的**唯一实现**,CityHash64Any 只做类型
// 分发。这样调用方可以直接调 typed 版本而无需把值装箱成 interface{}(装箱是一次堆
// 分配,alloc_objects profile 里各算子调用 GetFid 时的装箱占全进程分配对象数的 8%),
// 同时哈希结果与走 interface{} 完全同源,不可能漂移。

func CityHash64String(x string) uint64 { return Hash64([]byte(x)) }

func CityHash64Bytes(x []byte) uint64 { return Hash64(x) }

func CityHash64Uint64(x uint64) uint64 {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, x)
	return Hash64(buf)
}

// CityHash64Int 与 CityHash64Int64 共用 8 字节小端表示,和原 case int / case int64 一致
func CityHash64Int(x int) uint64 { return CityHash64Uint64(uint64(x)) }

func CityHash64Int64(x int64) uint64 { return CityHash64Uint64(uint64(x)) }

func CityHash64Uint32(x uint32) uint64 {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, x)
	return Hash64(buf)
}

func CityHash64Int32(x int32) uint64 { return CityHash64Uint32(uint32(x)) }

func CityHash64Float64(x float64) uint64 { return CityHash64Uint64(math.Float64bits(x)) }

func CityHash64Float32(x float32) uint64 { return CityHash64Uint32(math.Float32bits(x)) }

func CityHash64Bool(x bool) uint64 {
	if x {
		return Hash64([]byte{1})
	}
	return Hash64([]byte{0})
}

// CombineSlotWithHash 把已经算好的 hash 与 slot 合并 —— CombineSlotAndHash 的后半段,
// 供已经自己算好哈希的 typed 调用方复用(否则就得把值再装箱一次去走 interface{} 版本)。
//
// 与 CombineSlotAndVal 是同一个位运算,只是入参含义不同(一个是哈希结果、一个是原值),
// 所以保留两个名字让调用点读起来准确,但实现只有一份。
func CombineSlotWithHash(slotID int, hashvalue uint64) uint64 {
	return CombineSlotAndVal(slotID, hashvalue)
}

func CityHash64Any(v interface{}) uint64 {
	switch x := v.(type) {
	case nil:
		return 0
	case []byte:
		return CityHash64Bytes(x)
	case string:
		return CityHash64String(x)
	case int:
		return CityHash64Int(x)
	case int32:
		return CityHash64Int32(x)
	case int64:
		return CityHash64Int64(x)
	case uint64:
		return CityHash64Uint64(x)
	case float32:
		return CityHash64Float32(x)
	case float64:
		return CityHash64Float64(x)
	case bool:
		return CityHash64Bool(x)
	default:
		// 兜底：将字符串表示 hash
		return Hash64([]byte(fmt.Sprintf("%v", x)))
	}
}

func Hash64(s []byte) uint64 {
	n := uint64(len(s))
	if n <= 32 {
		if n <= 16 {
			return hash64Len0to16(s)
		}
		return hash64Len17to32(s)
	} else if n <= 64 {
		return hash64Len33to64(s)
	}

	x := fetch64(s[n-40:])
	y := fetch64(s[n-16:]) + fetch64(s[n-56:])
	z := hash64Len16(fetch64(s[n-48:])+n, fetch64(s[n-24:]))

	v1, v2 := weakHashLen32WithSeeds(s[n-64:], n, z)
	w1, w2 := weakHashLen32WithSeeds(s[n-32:], y+k1, x)
	x = x*k1 + fetch64(s)

	n = (n - 1) &^ 63
	for {
		x = ror64(x+y+v1+fetch64(s[8:]), 37) * k1
		y = ror64(y+v2+fetch64(s[48:]), 42) * k1
		x ^= w2
		y += v1 + fetch64(s[40:])
		z = ror64(z+w1, 33) * k1
		v1, v2 = weakHashLen32WithSeeds(s, v2*k1, x+w1)
		w1, w2 = weakHashLen32WithSeeds(s[32:], z+w2, y+fetch64(s[16:]))
		z, x = x, z
		s = s[64:]
		n -= 64
		if n == 0 {
			break
		}
	}
	return hash64Len16(hash64Len16(v1, w1)+shiftMix(y)*k1+z, hash64Len16(v2, w2)+x)
}

func ror64(val, shift uint64) uint64 {
	// Avoid shifting by 64: doing so yields an undefined result.
	if shift != 0 {
		return val>>shift | val<<(64-shift)
	}
	return val
}

func ror32(val, shift uint32) uint32 {
	// Avoid shifting by 32: doing so yields an undefined result.
	if shift != 0 {
		return val>>shift | val<<(32-shift)
	}
	return val
}

func shiftMix(val uint64) uint64 { return val ^ (val >> 47) }

func hash64Len0to16(s []byte) uint64 {
	n := uint64(len(s))
	if n >= 8 {
		mul := k2 + n*2
		a := fetch64(s) + k2
		b := fetch64(s[n-8:])
		c := ror64(b, 37)*mul + a
		d := (ror64(a, 25) + b) * mul
		return hash64Len16Mul(c, d, mul)
	}
	if n >= 4 {
		mul := k2 + n*2
		a := uint64(fetch32(s))
		return hash64Len16Mul(n+(a<<3), uint64(fetch32(s[n-4:])), mul)
	}
	if n > 0 {
		a := s[0]
		b := s[n>>1]
		c := s[n-1]
		y := uint32(a) + uint32(b)<<8
		z := uint32(n) + uint32(c)<<2
		return shiftMix(uint64(y)*k2^uint64(z)*k0) * k2
	}
	return k2
}

func hash64Len17to32(s []byte) uint64 {
	n := uint64(len(s))
	mul := k2 + n*2
	a := fetch64(s) * k1
	b := fetch64(s[8:])
	c := fetch64(s[n-8:]) * mul
	d := fetch64(s[n-16:]) * k2
	return hash64Len16Mul(ror64(a+b, 43)+ror64(c, 30)+d, a+ror64(b+k2, 18)+c, mul)
}

func bswap64(x uint64) uint64 {
	return bits.ReverseBytes64(x)
}

func hash64Len33to64(s []byte) uint64 {
	n := uint64(len(s))
	mul := k2 + n*2
	a := fetch64(s) * k2
	b := fetch64(s[8:])
	c := fetch64(s[n-24:])
	d := fetch64(s[n-32:])
	e := fetch64(s[16:]) * k2
	f := fetch64(s[24:]) * 9
	g := fetch64(s[n-8:])
	h := fetch64(s[n-16:]) * mul
	u := ror64(a+g, 43) + (ror64(b, 30)+c)*9
	v := ((a + g) ^ d) + f + 1
	w := bswap64((u+v)*mul) + h
	x := ror64(e+f, 42) + c
	y := (bswap64((v+w)*mul) + g) * mul
	z := e + f + c
	a = bswap64((x+z)*mul+y) + b
	b = shiftMix((z+a)*mul+d+h) * mul
	return b + x
}

func hash64Len16Mul(u, v, mul uint64) uint64 {
	// Murmur-inspired hashing.
	a := (u ^ v) * mul
	a ^= (a >> 47)
	b := (v ^ a) * mul
	b ^= (b >> 47)
	b *= mul
	return b
}

func hash64Len16(u, v uint64) uint64 { return hash128to64(u, v) }

func hash128to64(lo, hi uint64) uint64 {
	// Murmur-inspired hashing.
	const mul = uint64(0x9ddfea08eb382d69)
	a := (lo ^ hi) * mul
	a ^= (a >> 47)
	b := (hi ^ a) * mul
	b ^= (b >> 47)
	b *= mul
	return b
}

func fetch64(b []byte) uint64 { return *(*uint64)(unsafe.Pointer(&b[0])) }
func fetch32(b []byte) uint32 { return *(*uint32)(unsafe.Pointer(&b[0])) }

func weakHashLen32WithSeeds(s []byte, a, b uint64) (uint64, uint64) {
	w := fetch64(s)
	x := fetch64(s[8:])
	y := fetch64(s[16:])
	z := fetch64(s[24:])

	a += w
	b = ror64(b+a+z, 21)
	c := a
	a += x
	a += y
	b += ror64(a, 44)
	return a + z, b + c
}

func HashBucketID(token string, numBuckets int64) int64 {
	return int64(Hash64([]byte(token))%uint64(numBuckets-1)) + 1
}
