package ignite

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	boolBytes  = 1
	byteBytes  = 1
	shortBytes = 2
	intBytes   = 4
	longBytes  = 8
	uuidBytes  = 16
)

type BinaryWriter interface {
	Data() []byte
	Position() int32
	Available() int32
	SetPosition(pos int32)
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
	WriteBytes(v []byte)
}

type BinaryReader interface {
	Position() int32
	Available() int32
	SetPosition(pos int32)
	ReadBool() bool
	ReadUInt8() uint8
	ReadInt8() int8
	ReadUInt16() uint16
	ReadInt16() int16
	ReadUInt32() uint32
	ReadInt32() int32
	ReadUInt64() uint64
	ReadInt64() int64
	ReadBytes(size int32) []byte
	IsNull() bool
}

type binaryWriterImpl struct {
	buffer   []byte
	position int32
}

type binaryReaderImpl struct {
	buffer   []byte
	offset   int32
	position int32
}

func NewBinaryWriter(length int) BinaryWriter {
	return &binaryWriterImpl{
		buffer: make([]byte, length),
	}
}

func NewBinaryReader(buffer []byte, offset int32) BinaryReader {
	return &binaryReaderImpl{
		buffer:   buffer,
		offset:   offset,
		position: offset,
	}
}

func (bw *binaryWriterImpl) Data() []byte {
	return bw.buffer[:bw.position]
}

func (bw *binaryWriterImpl) Available() int32 {
	return int32(len(bw.buffer)) - bw.position
}

func (bw *binaryWriterImpl) Position() int32 {
	return bw.position
}

func (bw *binaryWriterImpl) SetPosition(pos int32) {
	bw.position = pos
}

func (bw *binaryWriterImpl) ensureAvailable(size int32) {
	if math.MaxInt32-bw.position < size {
		panic(fmt.Sprintf("Buffer length overflow: position=%d, required size=%d", bw.position, size))
	}
	if bw.Available() < size {
		temp := make([]byte, bw.position+size)
		copy(temp, bw.buffer)
		bw.buffer = temp
	}
}

func (bw *binaryWriterImpl) WriteNull() {
	bw.WriteInt8(nullType)
}

func (bw *binaryWriterImpl) WriteBool(v bool) {
	bw.ensureAvailable(boolBytes)
	bw.writeBool(v)
}

func (bw *binaryWriterImpl) writeBool(v bool) {
	if v {
		bw.buffer[bw.position] = 1
	} else {
		bw.buffer[bw.position] = 0
	}
	bw.position += boolBytes
}

func (bw *binaryWriterImpl) WriteUInt8(v uint8) {
	bw.ensureAvailable(byteBytes)
	bw.writeByte(v)
}

func (bw *binaryWriterImpl) WriteInt8(v int8) {
	bw.ensureAvailable(byteBytes)
	bw.writeByte(uint8(v))
}

func (bw *binaryWriterImpl) writeByte(v byte) {
	bw.buffer[bw.position] = v
	bw.position += byteBytes
}

func (bw *binaryWriterImpl) WriteInt16(v int16) {
	bw.ensureAvailable(shortBytes)
	bw.writeShort(uint16(v))
}

func (bw *binaryWriterImpl) WriteUInt16(v uint16) {
	bw.ensureAvailable(shortBytes)
	bw.writeShort(v)
}

func (bw *binaryWriterImpl) writeShort(v uint16) {
	binary.LittleEndian.PutUint16(bw.buffer[bw.position:], v)
	bw.position += shortBytes
}

func (bw *binaryWriterImpl) WriteInt32(v int32) {
	bw.ensureAvailable(intBytes)
	bw.writeInt(uint32(v))
}

func (bw *binaryWriterImpl) WriteUInt32(v uint32) {
	bw.ensureAvailable(intBytes)
	bw.writeInt(v)
}

func (bw *binaryWriterImpl) writeInt(v uint32) {
	binary.LittleEndian.PutUint32(bw.buffer[bw.position:], v)
	bw.position += intBytes
}

func (bw *binaryWriterImpl) WriteUInt64(v uint64) {
	bw.ensureAvailable(longBytes)
	bw.writeLong(v)
}

func (bw *binaryWriterImpl) WriteInt64(v int64) {
	bw.ensureAvailable(longBytes)
	bw.writeLong(uint64(v))
}

func (bw *binaryWriterImpl) writeLong(v uint64) {
	binary.LittleEndian.PutUint64(bw.buffer[bw.position:], v)
	bw.position += longBytes
}

func (bw *binaryWriterImpl) WriteBytes(v []byte) {
	length := int32(len(v))
	bw.ensureAvailable(length)
	copy(bw.buffer[bw.position:], v)
	bw.position += length
}

func (br *binaryReaderImpl) Available() int32 {
	return int32(len(br.buffer)) - br.position
}

func (br *binaryReaderImpl) Position() int32 {
	return br.position
}

func (br *binaryReaderImpl) SetPosition(pos int32) {
	if pos < 0 {
		panic(fmt.Sprintf("negatige position passed: %d", pos))
	}
	if len(br.buffer) < int(pos) {
		panic(fmt.Sprintf("position %d is out of range, buffer length: %d", pos, len(br.buffer)))
	}
	br.position = pos
}

func (br *binaryReaderImpl) ReadBool() bool {
	ret := false
	if br.buffer[br.position] == 1 {
		ret = true
	}
	br.position += boolBytes
	return ret
}

func (br *binaryReaderImpl) ReadUInt8() uint8 {
	ret := br.buffer[br.position]
	br.position += byteBytes
	return ret
}

func (br *binaryReaderImpl) ReadInt8() int8 {
	return int8(br.ReadUInt8())
}

func (br *binaryReaderImpl) ReadUInt16() uint16 {
	r := binary.LittleEndian.Uint16(br.buffer[br.position:])
	br.position += shortBytes
	return r
}

func (br *binaryReaderImpl) ReadInt16() int16 {
	return int16(br.ReadUInt16())
}

func (br *binaryReaderImpl) ReadUInt32() uint32 {
	r := binary.LittleEndian.Uint32(br.buffer[br.position:])
	br.position += intBytes
	return r
}

func (br *binaryReaderImpl) ReadInt32() int32 {
	return int32(br.ReadUInt32())
}

func (br *binaryReaderImpl) ReadUInt64() uint64 {
	r := binary.LittleEndian.Uint64(br.buffer[br.position:])
	br.position += longBytes
	return r
}

func (br *binaryReaderImpl) IsNull() bool {
	if br.buffer[br.position] == byte(nullType) {
		br.position += byteBytes
		return true
	}
	return false
}

func (br *binaryReaderImpl) ReadInt64() int64 {
	return int64(br.ReadUInt64())
}

func (br *binaryReaderImpl) ReadBytes(size int32) []byte {
	ret := make([]byte, size)
	br.position += int32(copy(ret, br.buffer[br.position:]))
	return ret
}
