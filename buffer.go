package ignite

import (
	"encoding/binary"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client/internal"
	"math"
)

const (
	boolBytes  = 1
	byteBytes  = 1
	shortBytes = 2
	charBytes  = 2
	intBytes   = 4
	longBytes  = 8
)

type BinaryOutputStream interface {
	Data() []byte
	Position() int
	Available() int
	SetPosition(pos int)
	WriteNull()
	WriteBool(v bool)
	WriteUInt8(v uint8)
	WriteInt8(v int8)
	WriteUInt16(v uint16)
	WriteInt16(v int16)
	WriteUInt32(v uint32)
	WriteInt32(v int32)
	WriteUInt64(v uint64)
	WriteInt64(v int64)
	WriteFloat32(v float32)
	WriteFloat64(v float64)
	WriteBytes(v []byte)
	HashCode(start int, end int) int32
}

type BinaryInputStream interface {
	Position() int
	Available() int
	SetPosition(pos int)
	ReadBool() bool
	ReadUInt8() uint8
	ReadInt8() int8
	ReadUInt16() uint16
	ReadInt16() int16
	ReadUInt32() uint32
	ReadInt32() int32
	ReadUInt64() uint64
	ReadInt64() int64
	ReadFloat32() float32
	ReadFloat64() float64
	ReadBytes(size int) []byte
	IsNull() bool
}

type binaryOutputStreamImpl struct {
	buffer   []byte
	position int
}

type binaryInputStreamImpl struct {
	buffer   []byte
	offset   int
	position int
}

func NewBinaryOutputStream(length int) BinaryOutputStream {
	return &binaryOutputStreamImpl{
		buffer: make([]byte, length),
	}
}

func NewBinaryReader(buffer []byte, offset int) BinaryInputStream {
	return &binaryInputStreamImpl{
		buffer:   buffer,
		offset:   offset,
		position: offset,
	}
}

func (bw *binaryOutputStreamImpl) Data() []byte {
	return bw.buffer[:bw.position]
}

func (bw *binaryOutputStreamImpl) Available() int {
	return len(bw.buffer) - bw.position
}

func (bw *binaryOutputStreamImpl) Position() int {
	return bw.position
}

func (bw *binaryOutputStreamImpl) SetPosition(pos int) {
	bw.position = pos
}

func (bw *binaryOutputStreamImpl) ensureAvailable(size int) {
	if math.MaxInt32-bw.position < size {
		panic(fmt.Sprintf("Buffer length overflow: position=%d, required size=%d", bw.position, size))
	}
	if bw.Available() < size {
		temp := make([]byte, bw.position+size)
		copy(temp, bw.buffer)
		bw.buffer = temp
	}
}

func (bw *binaryOutputStreamImpl) WriteNull() {
	bw.WriteInt8(NullType)
}

func (bw *binaryOutputStreamImpl) WriteBool(v bool) {
	bw.ensureAvailable(boolBytes)
	bw.writeBool(v)
}

func (bw *binaryOutputStreamImpl) writeBool(v bool) {
	if v {
		bw.buffer[bw.position] = 1
	} else {
		bw.buffer[bw.position] = 0
	}
	bw.position += boolBytes
}

func (bw *binaryOutputStreamImpl) WriteUInt8(v uint8) {
	bw.ensureAvailable(byteBytes)
	bw.writeByte(v)
}

func (bw *binaryOutputStreamImpl) WriteInt8(v int8) {
	bw.ensureAvailable(byteBytes)
	bw.writeByte(uint8(v))
}

func (bw *binaryOutputStreamImpl) writeByte(v byte) {
	bw.buffer[bw.position] = v
	bw.position += byteBytes
}

func (bw *binaryOutputStreamImpl) WriteInt16(v int16) {
	bw.ensureAvailable(shortBytes)
	bw.writeShort(uint16(v))
}

func (bw *binaryOutputStreamImpl) WriteUInt16(v uint16) {
	bw.ensureAvailable(shortBytes)
	bw.writeShort(v)
}

func (bw *binaryOutputStreamImpl) writeShort(v uint16) {
	binary.LittleEndian.PutUint16(bw.buffer[bw.position:], v)
	bw.position += shortBytes
}

func (bw *binaryOutputStreamImpl) WriteInt32(v int32) {
	bw.ensureAvailable(intBytes)
	bw.writeInt(uint32(v))
}

func (bw *binaryOutputStreamImpl) WriteUInt32(v uint32) {
	bw.ensureAvailable(intBytes)
	bw.writeInt(v)
}

func (bw *binaryOutputStreamImpl) writeInt(v uint32) {
	binary.LittleEndian.PutUint32(bw.buffer[bw.position:], v)
	bw.position += intBytes
}

func (bw *binaryOutputStreamImpl) WriteUInt64(v uint64) {
	bw.ensureAvailable(longBytes)
	bw.writeLong(v)
}

func (bw *binaryOutputStreamImpl) WriteInt64(v int64) {
	bw.ensureAvailable(longBytes)
	bw.writeLong(uint64(v))
}

func (bw *binaryOutputStreamImpl) writeLong(v uint64) {
	binary.LittleEndian.PutUint64(bw.buffer[bw.position:], v)
	bw.position += longBytes
}

func (bw *binaryOutputStreamImpl) WriteFloat32(v float32) {
	bw.WriteUInt32(math.Float32bits(v))
}

func (bw *binaryOutputStreamImpl) WriteFloat64(v float64) {
	bw.WriteUInt64(math.Float64bits(v))
}

func (bw *binaryOutputStreamImpl) WriteBytes(v []byte) {
	length := len(v)
	bw.ensureAvailable(length)
	copy(bw.buffer[bw.position:], v)
	bw.position += length
}

func (bw *binaryOutputStreamImpl) HashCode(start int, end int) int32 {
	bufLen := len(bw.buffer)
	if start > bufLen || start < 0 || end > bufLen || end < 0 || start > end {
		panic(fmt.Sprintf("invalid start=%d, end=%d", start, end))
	}
	return internal.SliceHashCode(bw.buffer[start:end])
}

func (br *binaryInputStreamImpl) Available() int {
	return len(br.buffer) - br.position
}

func (br *binaryInputStreamImpl) Position() int {
	return br.position
}

func (br *binaryInputStreamImpl) SetPosition(pos int) {
	if pos < 0 {
		panic(fmt.Sprintf("negatige position passed: %d", pos))
	}
	if len(br.buffer) < int(pos) {
		panic(fmt.Sprintf("position %d is out of range, buffer length: %d", pos, len(br.buffer)))
	}
	br.position = pos
}

func (br *binaryInputStreamImpl) ReadBool() bool {
	ret := false
	if br.buffer[br.position] == 1 {
		ret = true
	}
	br.position += boolBytes
	return ret
}

func (br *binaryInputStreamImpl) ReadUInt8() uint8 {
	ret := br.buffer[br.position]
	br.position += byteBytes
	return ret
}

func (br *binaryInputStreamImpl) ReadInt8() int8 {
	return int8(br.ReadUInt8())
}

func (br *binaryInputStreamImpl) ReadUInt16() uint16 {
	r := binary.LittleEndian.Uint16(br.buffer[br.position:])
	br.position += shortBytes
	return r
}

func (br *binaryInputStreamImpl) ReadInt16() int16 {
	return int16(br.ReadUInt16())
}

func (br *binaryInputStreamImpl) ReadUInt32() uint32 {
	r := binary.LittleEndian.Uint32(br.buffer[br.position:])
	br.position += intBytes
	return r
}

func (br *binaryInputStreamImpl) ReadInt32() int32 {
	return int32(br.ReadUInt32())
}

func (br *binaryInputStreamImpl) ReadUInt64() uint64 {
	r := binary.LittleEndian.Uint64(br.buffer[br.position:])
	br.position += longBytes
	return r
}

func (br *binaryInputStreamImpl) IsNull() bool {
	if br.buffer[br.position] == byte(NullType) {
		br.position += byteBytes
		return true
	}
	return false
}

func (br *binaryInputStreamImpl) ReadInt64() int64 {
	return int64(br.ReadUInt64())
}

func (br *binaryInputStreamImpl) ReadFloat32() float32 {
	return math.Float32frombits(br.ReadUInt32())
}

func (br *binaryInputStreamImpl) ReadFloat64() float64 {
	return math.Float64frombits(br.ReadUInt64())
}

func (br *binaryInputStreamImpl) ReadBytes(size int) []byte {
	ret := make([]byte, size)
	br.position += copy(ret, br.buffer[br.position:])
	return ret
}

func readSlice[T any](reader BinaryInputStream, elemReader func(int, BinaryInputStream) (T, error)) ([]T, error) {
	sz := int(reader.ReadInt32())
	coll := make([]T, sz)
	err := readSequence(reader, sz, func(idx int, reader BinaryInputStream) error {
		el, err := elemReader(idx, reader)
		if err != nil {
			return err
		}
		coll[idx] = el
		return nil
	})
	return coll, err
}

func readSequence(reader BinaryInputStream, length int, elemReader func(idx int, reader BinaryInputStream) error) error {
	for i := 0; i < length; i++ {
		if err := elemReader(i, reader); err != nil {
			return err
		}
	}
	return nil
}

func writeSequence(writer BinaryOutputStream, length int, valueWriter func(output BinaryOutputStream, idx int) error) error {
	writer.WriteInt32(int32(length))
	for idx := 0; idx < length; idx++ {
		err := valueWriter(writer, idx)
		if err != nil {
			return err
		}
	}
	return nil
}

func readMap[K comparable, V any](reader BinaryInputStream, kvReader func(BinaryInputStream) (K, V, error)) (map[K]V, error) {
	length := int(reader.ReadInt32())
	outMap := make(map[K]V)
	err := readSequence(reader, length, func(_ int, reader BinaryInputStream) error {
		k, v, err := kvReader(reader)
		if err != nil {
			return err
		}
		outMap[k] = v
		return nil
	})
	return outMap, err
}

func writeMap[K comparable, V any](writer BinaryOutputStream, inMap map[K]V, kvWriter func(BinaryOutputStream, K, V) error) error {
	writer.WriteInt32(int32(len(inMap)))
	for k, v := range inMap {
		err := kvWriter(writer, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
