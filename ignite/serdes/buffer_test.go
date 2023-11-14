package serdes

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumbers(t *testing.T) {
	buf := CreateIgniteBuffer()

	nums0 := []int32{0, -1, -2, 42, 1000000, math.MaxInt32, math.MinInt32}
	nums1 := []int16{0, -1, -2, 42, 10000, math.MaxInt16, math.MinInt16}
	nums2 := []byte{0, 1, 2, 42, 100}

	for _, num := range nums0 {
		buf.WriteInt32(num)
	}

	for _, num := range nums1 {
		buf.WriteInt16(num)
	}

	for _, num := range nums2 {
		buf.WriteByte(num)
	}

	buf.Reset()

	for _, num := range nums0 {
		assert.Equal(t, num, buf.ReadInt32(), "Must read same number as written")
	}

	for _, num := range nums1 {
		assert.Equal(t, num, buf.ReadInt16(), "Must read same number as written")
	}

	for _, num := range nums2 {
		assert.Equal(t, num, buf.ReadByte(), "Must read same number as written")
	}

	buf.Reset()
	buf.WriteByteArray(&nums2)
	buf.Reset()

	assert.Equal(t, nums2, buf.ReadByteArray(), "Must read same bytes as written")
}

func TestStrings(t *testing.T) {
	buf := CreateIgniteBuffer()

	str := "test"

	buf.WriteString(&str)
	buf.Reset()

	assert.Equal(t, str, buf.ReadString(), "Must read written string")
}
