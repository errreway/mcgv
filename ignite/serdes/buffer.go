package serdes

import (
	"encoding/binary"
	"fmt"
)

const (
	dfltBufSz  int32 = 1024 * 1024
	shortBytes       = 2
	intBytes         = 4
	longBytes        = 8

	STRING     byte = 9
	BYTE_ARRAY byte = 12
	MAP        byte = 25
	NULL       byte = 101
)

type IgniteBuffer struct {
	buf []byte

	idx int

	limit int
}

func (buf *IgniteBuffer) WriteByte(v byte) {
	buf.buf[buf.idx] = v
	buf.incrementIndex(1)
}

func (buf *IgniteBuffer) writeBytes(v []byte) {
	length := len(v)
	copy(buf.buf[buf.idx:(buf.idx+length)], v)
	buf.incrementIndex(length)
}

func (buf *IgniteBuffer) WriteInt(v int) {
	binary.LittleEndian.PutUint32(buf.buf[buf.idx:(buf.idx+intBytes)], uint32(v))
	buf.incrementIndex(intBytes)
}

func (buf *IgniteBuffer) WriteInt16(v int16) {
	binary.LittleEndian.PutUint16(buf.buf[buf.idx:(buf.idx+shortBytes)], uint16(v))
	buf.incrementIndex(shortBytes)
}

func (buf *IgniteBuffer) WriteInt32(v int32) {
	binary.LittleEndian.PutUint32(buf.buf[buf.idx:(buf.idx+intBytes)], uint32(v))
	buf.incrementIndex(intBytes)
}

func (buf *IgniteBuffer) WriteInt64(v int64) {
	binary.LittleEndian.PutUint64(buf.buf[buf.idx:(buf.idx+longBytes)], uint64(v))
	buf.incrementIndex(longBytes)
}

func (buf *IgniteBuffer) WriteString(v string) {
	bytes := []byte(v)

	buf.WriteByte(STRING)
	buf.WriteInt(len(bytes))
	buf.writeBytes(bytes)
}

func (buf *IgniteBuffer) WriteByteArray(v []byte) {
	buf.WriteByte(BYTE_ARRAY)
	buf.WriteInt(len(v))
	buf.writeBytes(v)
}

func (buf *IgniteBuffer) ReadByte() byte {
	res := buf.buf[buf.idx]
	buf.incrementIndex(1)
	return res
}

func (buf *IgniteBuffer) ReadInt() int {
	return int(buf.ReadInt32())
}

func (buf *IgniteBuffer) ReadInt16() int16 {
	res := int16(binary.LittleEndian.Uint16(buf.buf[buf.idx:(buf.idx + shortBytes)]))
	buf.incrementIndex(shortBytes)
	return res
}

func (buf *IgniteBuffer) ReadInt32() int32 {
	res := int32(binary.LittleEndian.Uint32(buf.buf[buf.idx:(buf.idx + intBytes)]))
	buf.incrementIndex(intBytes)
	return res
}

func (buf *IgniteBuffer) ReadInt64() int64 {
	res := int64(binary.LittleEndian.Uint64(buf.buf[buf.idx:(buf.idx + longBytes)]))
	buf.incrementIndex(longBytes)
	return res
}

func (buf *IgniteBuffer) ReadString() string {
	if buf.ReadByte() != STRING {
		panic("not a string")
	}

	length := buf.ReadInt()
	res := string(buf.buf[buf.idx:(buf.idx + length)])
	buf.incrementIndex(length)

	return res
}

func (buf *IgniteBuffer) ReadByteArray() interface{} {
	if buf.ReadByte() != BYTE_ARRAY {
		panic("not a byte array")
	}

	length := buf.ReadInt()
	res := buf.buf[buf.idx:(buf.idx + length)]
	buf.incrementIndex(length)
	return res
}

func (buf *IgniteBuffer) Reset() {
	buf.idx = 0
}

func (buf *IgniteBuffer) Limit(limit int) {
	buf.limit = limit
}

func (buf *IgniteBuffer) incrementIndex(cnt int) {
	buf.idx += cnt
	if buf.idx > buf.limit {
		panic(fmt.Sprintf("buffer overflow[idx=%d,max_idx=%d]", buf.idx, buf.limit))
	}
}

func (buf *IgniteBuffer) HashMore() bool {
	return buf.idx < buf.limit
}

func CreateIgniteBuffer() IgniteBuffer {
	return IgniteBuffer{make([]byte, dfltBufSz), 0, int(dfltBufSz)}
}
