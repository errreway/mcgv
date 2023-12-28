package serdes

import (
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"math"
	"unsafe"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=TypeDesc

type TypeDesc int8

const (
	BoolBytes  = 1
	ByteBytes  = 1
	ShortBytes = 2
	IntBytes   = 4
	LongBytes  = 8
	UuidBytes  = 16

	String       TypeDesc = 9
	Uuid         TypeDesc = 10
	ByteArray    TypeDesc = 12
	ShortArray   TypeDesc = 13
	IntArray     TypeDesc = 14
	LongArray    TypeDesc = 15
	FloatArray   TypeDesc = 16
	DoubleArray  TypeDesc = 17
	CharArray    TypeDesc = 18
	BooleanArray TypeDesc = 19
	StringArray  TypeDesc = 20
	Map          TypeDesc = 25
	Null         TypeDesc = 101
)

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type Integer interface {
	Signed | Unsigned
}

type BinaryWriter interface {
	Data() []byte
	Position() int32
	Available() int32
	SetPosition(pos int32)
	WriteBool(v bool)
	WriteByte(v byte)
	WriteUInt8(v uint8)
	WriteInt8(v int8)
	WriteUInt16(v uint16)
	WriteInt16(v int16)
	WriteUInt32(v uint32)
	WriteInt32(v int32)
	WriteUInt64(v uint64)
	WriteInt64(v int64)
	WriteString(v *string)
	WriteByteArray(v []byte)
	WriteInt8Array(v []int8)
	WriteStringArray(v []string)
	WriteNullableStringArray(v []*string)
	WriteUuid(v *uuid.UUID)
}

type BinaryReader interface {
	Position() int32
	Available() int32
	SetPosition(pos int32)
	ReadBool() bool
	ReadByte() byte
	ReadUInt8() uint8
	ReadInt8() int8
	ReadUInt16() uint16
	ReadInt16() int16
	ReadUInt32() uint32
	ReadInt32() int32
	ReadUInt64() uint64
	ReadInt64() int64
	ReadString() *string
	ReadByteArray() []byte
	ReadInt8Array() []int8
	ReadStringArray() []string
	ReadNullableStringArray() []*string
	ReadUuid() *uuid.UUID
}

type BinaryWriterImpl struct {
	buffer   []byte
	position int32
}

type BinaryReaderImpl struct {
	buffer   []byte
	offset   int32
	position int32
}

func NewBinaryWriter(length int) BinaryWriter {
	return &BinaryWriterImpl{
		buffer: make([]byte, length),
	}
}

func NewBinaryReader(buffer []byte, offset int32) BinaryReader {
	return &BinaryReaderImpl{
		buffer:   buffer,
		offset:   offset,
		position: offset,
	}
}

func (bw *BinaryWriterImpl) Data() []byte {
	return bw.buffer[:bw.position]
}

func (bw *BinaryWriterImpl) Available() int32 {
	return int32(len(bw.buffer)) - bw.position
}

func (bw *BinaryWriterImpl) Position() int32 {
	return bw.position
}

func (bw *BinaryWriterImpl) SetPosition(pos int32) {
	bw.position = pos
}

func (bw *BinaryWriterImpl) ensureAvailable(size int32) {
	if math.MaxInt32-bw.position < size {
		panic(fmt.Sprintf("Buffer length overflow: position=%d, required size=%d", bw.position, size))
	}
	if bw.Available() < size {
		temp := make([]byte, bw.position+size)
		copy(temp, bw.buffer)
		bw.buffer = temp
	}
}

func (bw *BinaryWriterImpl) WriteBool(v bool) {
	bw.ensureAvailable(BoolBytes)
	bw.writeBool(v)
}

func (bw *BinaryWriterImpl) writeBool(v bool) {
	if v {
		bw.buffer[bw.position] = 1
	} else {
		bw.buffer[bw.position] = 0
	}
	bw.position += BoolBytes
}

func (bw *BinaryWriterImpl) WriteByte(v byte) {
	bw.WriteUInt8(v)
}

func (bw *BinaryWriterImpl) WriteUInt8(v uint8) {
	bw.ensureAvailable(ByteBytes)
	bw.writeByte(v)
}

func (bw *BinaryWriterImpl) WriteInt8(v int8) {
	bw.ensureAvailable(ByteBytes)
	bw.writeByte(uint8(v))
}

func (bw *BinaryWriterImpl) writeByte(v byte) {
	bw.buffer[bw.position] = v
	bw.position += ByteBytes
}

func (bw *BinaryWriterImpl) WriteInt16(v int16) {
	bw.ensureAvailable(ShortBytes)
	bw.writeShort(uint16(v))
}

func (bw *BinaryWriterImpl) WriteUInt16(v uint16) {
	bw.ensureAvailable(ShortBytes)
	bw.writeShort(v)
}

func (bw *BinaryWriterImpl) writeShort(v uint16) {
	binary.LittleEndian.PutUint16(bw.buffer[bw.position:], v)
	bw.position += ShortBytes
}

func (bw *BinaryWriterImpl) WriteInt32(v int32) {
	bw.ensureAvailable(IntBytes)
	bw.writeInt(uint32(v))
}

func (bw *BinaryWriterImpl) WriteUInt32(v uint32) {
	bw.ensureAvailable(IntBytes)
	bw.writeInt(v)
}

func (bw *BinaryWriterImpl) writeInt(v uint32) {
	binary.LittleEndian.PutUint32(bw.buffer[bw.position:], v)
	bw.position += IntBytes
}

func (bw *BinaryWriterImpl) WriteUInt64(v uint64) {
	bw.ensureAvailable(LongBytes)
	bw.writeLong(v)
}

func (bw *BinaryWriterImpl) WriteInt64(v int64) {
	bw.ensureAvailable(LongBytes)
	bw.writeLong(uint64(v))
}

func (bw *BinaryWriterImpl) writeLong(v uint64) {
	binary.LittleEndian.PutUint64(bw.buffer[bw.position:], v)
	bw.position += LongBytes
}

func (bw *BinaryWriterImpl) writeNull() {
	bw.ensureAvailable(ByteBytes)
	bw.WriteInt8(int8(Null))
}

func (bw *BinaryWriterImpl) WriteString(v *string) {
	if v == nil {
		bw.writeNull()
		return
	}
	length := int32(len(*v))
	bw.WriteInt8(int8(String))
	bw.WriteInt32(length)
	if length > 0 {
		bw.ensureAvailable(length)
		copy(bw.buffer[bw.position:], *v)
		bw.position += length
	}
}

func (bw *BinaryWriterImpl) WriteByteArray(v []byte) {
	if v == nil {
		bw.writeNull()
		return
	}
	bw.WriteInt8(int8(ByteArray))
	length := int32(len(v))
	bw.WriteInt32(length)
	bw.ensureAvailable(length)
	copy(bw.buffer[bw.position:], v)
	bw.position += length
}

func (bw *BinaryWriterImpl) WriteInt8Array(v []int8) {
	if v == nil {
		bw.writeNull()
		return
	}
	bw.WriteInt8(int8(ByteArray))
	length := int32(len(v))
	bw.WriteInt32(length)
	bw.ensureAvailable(length)
	for _, sByte := range v {
		bw.writeByte(uint8(sByte))
	}
}

func (bw *BinaryWriterImpl) WriteStringArray(v []string) {
	if v == nil {
		bw.writeNull()
		return
	}
	bw.WriteInt8(int8(StringArray))
	bw.WriteInt32(int32(len(v)))
	for _, str := range v {
		bw.WriteString(&str)
	}
}

func (bw *BinaryWriterImpl) WriteNullableStringArray(v []*string) {
	if v == nil {
		bw.writeNull()
		return
	}
	bw.WriteInt8(int8(StringArray))
	bw.WriteInt32(int32(len(v)))
	for _, str := range v {
		bw.WriteString(str)
	}
}

func (bw *BinaryWriterImpl) WriteUuid(v *uuid.UUID) {
	if v == nil {
		bw.writeNull()
		return
	}
	bw.ensureAvailable(UuidBytes + ByteBytes)
	bw.writeByte(byte(Uuid))
	bw.position += int32(copy(bw.buffer[bw.position:], v[:]))
}

func (br *BinaryReaderImpl) Available() int32 {
	return int32(len(br.buffer)) - br.position
}

func (br *BinaryReaderImpl) Position() int32 {
	return br.position
}

func (br *BinaryReaderImpl) SetPosition(pos int32) {
	if pos < 0 {
		panic(fmt.Sprintf("negatige position passed: %d", pos))
	}
	if len(br.buffer) < int(pos) {
		panic(fmt.Sprintf("position %d is out of range, buffer length: %d", pos, len(br.buffer)))
	}
	br.position = pos
}

func (br *BinaryReaderImpl) assertAvailable(k int32) {
	if int32(len(br.buffer)) < br.position+k {
		panic(fmt.Sprintf("cannot read %d bytes", k))
	}
}

func (br *BinaryReaderImpl) ReadBool() bool {
	br.assertAvailable(ByteBytes)
	ret := false
	if br.buffer[br.position] == 1 {
		ret = true
	}
	br.position += BoolBytes
	return ret
}

func (br *BinaryReaderImpl) ReadByte() byte {
	return br.ReadUInt8()
}

func (br *BinaryReaderImpl) ReadUInt8() uint8 {
	br.assertAvailable(ByteBytes)
	ret := br.buffer[br.position]
	br.position += ByteBytes
	return ret
}

func (br *BinaryReaderImpl) ReadInt8() int8 {
	return int8(br.ReadUInt8())
}

func (br *BinaryReaderImpl) ReadUInt16() uint16 {
	br.assertAvailable(ShortBytes)
	r := binary.LittleEndian.Uint16(br.buffer[br.position:])
	br.position += ShortBytes
	return r
}

func (br *BinaryReaderImpl) ReadInt16() int16 {
	return int16(br.ReadUInt16())
}

func (br *BinaryReaderImpl) ReadUInt32() uint32 {
	br.assertAvailable(IntBytes)
	r := binary.LittleEndian.Uint32(br.buffer[br.position:])
	br.position += IntBytes
	return r
}

func (br *BinaryReaderImpl) ReadInt32() int32 {
	return int32(br.ReadUInt32())
}

func (br *BinaryReaderImpl) ReadUInt64() uint64 {
	br.assertAvailable(LongBytes)
	r := binary.LittleEndian.Uint64(br.buffer[br.position:])
	br.position += LongBytes
	return r
}

func (br *BinaryReaderImpl) isNull() bool {
	br.assertAvailable(ByteBytes)
	if br.buffer[br.position] == byte(Null) {
		br.position += ByteBytes
		return true
	}
	return false
}

func (br *BinaryReaderImpl) ReadInt64() int64 {
	return int64(br.ReadUInt64())
}

func (br *BinaryReaderImpl) ReadString() *string {
	if br.isNull() {
		return nil
	}
	if dsc := br.ReadInt8(); dsc != int8(String) {
		panic(fmt.Errorf("expected %d for string, got %d instead", String, TypeDesc(dsc)))
	}
	length := br.ReadInt32()
	br.assertAvailable(length)
	ret := bytesToString(br.buffer[br.position : br.position+length])
	br.position += length
	return &ret
}

func (br *BinaryReaderImpl) ReadByteArray() []byte {
	if br.isNull() {
		return nil
	}
	if dsc := br.ReadInt8(); dsc != int8(ByteArray) {
		panic(fmt.Errorf("expected %d for bytearray, got %d instead", ByteArray, dsc))
	}
	length := br.ReadInt32()
	br.assertAvailable(length)
	ret := br.buffer[br.position : br.position+length]
	return ret
}

func (br *BinaryReaderImpl) ReadInt8Array() []int8 {
	if br.isNull() {
		return nil
	}
	if dsc := br.ReadInt8(); dsc != int8(ByteArray) {
		panic(fmt.Errorf("expected %d for bytearray, got %d instead", ByteArray, dsc))
	}
	length := br.ReadInt32()
	br.assertAvailable(length)
	ret := make([]int8, length)
	for i := int32(0); i < length; i++ {
		ret[i] = br.ReadInt8()
	}
	return ret
}

func (br *BinaryReaderImpl) ReadStringArray() []string {
	//TODO implement me
	panic("implement me")
}

func (br *BinaryReaderImpl) ReadNullableStringArray() []*string {
	//TODO implement me
	panic("implement me")
}

func (br *BinaryReaderImpl) ReadUuid() *uuid.UUID {
	if br.isNull() {
		return nil
	}
	br.assertAvailable(UuidBytes + ByteBytes)
	res := uuid.UUID{}
	br.position += int32(copy(res[:], br.buffer[br.position:br.position+UuidBytes]))
	return &res
}

func bytesToString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}
