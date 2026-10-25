package utils

const (
	m    uint = 0x5bd1e995
	r    int  = 24
	seed int  = 97
)

func MurMurHash(key []rune, length int) uint {
	var h uint = uint(seed ^ length)
	var data uint
	data = uint(key[0])
	for length > 4 {
		k := data
		k *= m
		k ^= k >> r
		k *= m
		h *= m
		h ^= k
		data += 4
		length -= 4
	}
	switch length {
	case 3:
		h ^= uint(key[2]) << 16
	case 2:
		h ^= uint(key[1]) << 8
	case 1:
		h ^= uint(key[0])
	}
	h *= m
	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}

func BKDRHash(str string) uint {
	var bSeed uint = 31
	var hash uint = 0
	for _, ele := range str {
		hash = hash*bSeed + uint(ele)
	}
	return hash & 0x7FFFFFFF
}
