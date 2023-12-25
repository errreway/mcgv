package serdes

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumbers(t *testing.T) {
	bw := NewBinaryWriter(0)

	nums0 := []int32{0, -1, -2, 42, 1000000, math.MaxInt32, math.MinInt32}
	nums1 := []int16{0, -1, -2, 42, 10000, math.MaxInt16, math.MinInt16}
	nums2 := []byte{0, 1, 2, 42, 100}

	for _, num := range nums0 {
		bw.WriteInt32(num)
	}

	for _, num := range nums1 {
		bw.WriteInt16(num)
	}

	for _, num := range nums2 {
		bw.WriteByte(num)
	}

	br := NewBinaryReader(bw.Data(), 0)

	for _, num := range nums0 {
		assert.Equal(t, num, br.ReadInt32(), "Must read same number as written")
	}

	for _, num := range nums1 {
		assert.Equal(t, num, br.ReadInt16(), "Must read same number as written")
	}

	for _, num := range nums2 {
		assert.Equal(t, num, br.ReadByte(), "Must read same number as written")
	}

	bw = NewBinaryWriter(0)
	bw.WriteByteArray(nums2)
	br = NewBinaryReader(bw.Data(), 0)

	assert.Equal(t, nums2, br.ReadByteArray(), "Must read same bytes as written")
}

func TestStrings(t *testing.T) {
	bw := NewBinaryWriter(0)
	str := "test"
	bw.WriteString(&str)

	br := NewBinaryReader(bw.Data(), 0)

	assert.Equal(t, str, *br.ReadString(), "Must read written string")
}

func TestNilString(t *testing.T) {
	bw := NewBinaryWriter(0)
	bw.WriteString(nil)

	br := NewBinaryReader(bw.Data(), 0)
	assert.Nil(t, br.ReadString())
}
