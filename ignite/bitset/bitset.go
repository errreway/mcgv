package bitset

import "encoding/binary"

const (
	addressBitsPerWord uint = 6
)

type BitSet struct {
	words      []uint64
	wordsInUse uint
}

func New() *BitSet {
	return &BitSet{
		words: make([]uint64, 1),
	}
}

func (bs *BitSet) Test(idx uint) bool {
	wordIdx := wordIndex(idx)
	if wordIdx >= bs.wordsInUse {
		return false
	}
	return (bs.words[wordIdx] & (uint64(1) << idx)) != 0
}

func (bs *BitSet) Set(idx uint) {
	wordIdx := wordIndex(idx)
	bs.expandTo(wordIdx)
	bs.words[wordIdx] |= uint64(1) << idx
}

func (bs *BitSet) Clear(idx uint) {
	wordIdx := wordIndex(idx)
	if wordIdx >= bs.wordsInUse {
		return
	}
	bs.words[wordIdx] &= ^(uint64(1) << idx)
	bs.recalculateWordsInUse()
}

func FromBytes(data []byte) *BitSet {
	if len(data) == 0 {
		return New()
	}
	if x := len(data) % 8; x > 0 {
		for i := x; i < 8; i++ {
			data = append(data, 0)
		}
	}

	length := len(data) >> 3
	words := make([]uint64, length)

	for i := 0; i < length; i++ {
		words[i] = binary.LittleEndian.Uint64(data[8*i:])
	}

	return &BitSet{
		words:      words,
		wordsInUse: uint(length),
	}

}

func (bs *BitSet) Bytes() []byte {
	// Simplify serialization, don't care about excessive bytes.
	n := bs.wordsInUse
	if n == 0 {
		return []byte{}
	}
	res := make([]byte, 8*n)
	for i := uint(0); i < n; i++ {
		binary.LittleEndian.PutUint64(res[8*i:], bs.words[i])
	}
	return res
}

func (bs *BitSet) expandTo(wordIdx uint) {
	wordsRequired := wordIdx + 1
	if bs.wordsInUse < wordsRequired {
		bs.ensureCapacity(wordsRequired)
		bs.wordsInUse = wordsRequired
	}
}

func (bs *BitSet) ensureCapacity(wordsRequired uint) {
	if len(bs.words) < int(wordsRequired) {
		tmp := make([]uint64, max(wordsRequired, uint(len(bs.words))*2))
		copy(tmp, bs.words)
		bs.words = tmp
	}
}

func (bs *BitSet) recalculateWordsInUse() {
	var i uint
	for i = bs.wordsInUse - 1; i >= 0; i-- {
		if bs.words[i] != 0 {
			break
		}
	}
	bs.wordsInUse = i + 1
}

func wordIndex(bitIndex uint) uint {
	return bitIndex >> addressBitsPerWord
}

func maxUint(x uint, y uint) uint {
	if x > y {
		return x
	} else {
		return y
	}
}

func minUint(x uint, y uint) uint {
	if x < y {
		return x
	} else {
		return y
	}
}
