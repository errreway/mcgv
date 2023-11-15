package serdes

import (
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
)

const (
	DefaultBufferSize int32 = 1024 * 1024
	ShortBytes              = 2
	IntBytes                = 4
	LongBytes               = 8

	String      byte = 9
	UUID        byte = 10
	ByteArray   byte = 12
	StringArray byte = 20
	MAP         byte = 25
	Null        byte = 101
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

func (buf *IgniteBuffer) writeBytes(v *[]byte) {
	length := len(*v)
	copy(buf.buf[buf.idx:(buf.idx+length)], *v)
	buf.incrementIndex(length)
}

func (buf *IgniteBuffer) WriteInt(v int) {
	binary.LittleEndian.PutUint32(buf.buf[buf.idx:(buf.idx+IntBytes)], uint32(v))
	buf.incrementIndex(IntBytes)
}

func (buf *IgniteBuffer) WriteInt16(v int16) {
	binary.LittleEndian.PutUint16(buf.buf[buf.idx:(buf.idx+ShortBytes)], uint16(v))
	buf.incrementIndex(ShortBytes)
}

func (buf *IgniteBuffer) WriteInt32(v int32) {
	binary.LittleEndian.PutUint32(buf.buf[buf.idx:(buf.idx+IntBytes)], uint32(v))
	buf.incrementIndex(IntBytes)
}

func (buf *IgniteBuffer) WriteInt64(v int64) {
	binary.LittleEndian.PutUint64(buf.buf[buf.idx:(buf.idx+LongBytes)], uint64(v))
	buf.incrementIndex(LongBytes)
}

func (buf *IgniteBuffer) WriteString(v *string) {
	if v == nil {
		buf.WriteByte(Null)
		return
	}

	bytes := []byte(*v)

	buf.WriteByte(String)
	buf.WriteInt(len(bytes))
	buf.writeBytes(&bytes)
}

func (buf *IgniteBuffer) WriteByteArray(v *[]byte) {
	if v == nil {
		buf.WriteByte(Null)
		return
	}
	buf.WriteByte(ByteArray)
	buf.WriteInt(len(*v))
	buf.writeBytes(v)
}

func (buf *IgniteBuffer) WriteStringArray(v *[]string) {
	if v == nil {
		buf.WriteByte(Null)
		return
	}
	buf.WriteByte(StringArray)
	buf.WriteInt(len(*v))
	for _, str := range *v {
		buf.WriteString(&str)
	}
}

func (buf *IgniteBuffer) WriteUuid(v *uuid.UUID) {
	if v == nil {
		buf.WriteByte(Null)
		return
	}
	buf.WriteByte(UUID)
	bytes, err := v.MarshalBinary()
	if err != nil {
		panic(err)
	}
	buf.writeBytes(&bytes)
}

func (buf *IgniteBuffer) ReadBoolean() bool {
	return buf.ReadByte() == byte(1)
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
	res := int16(binary.LittleEndian.Uint16(buf.buf[buf.idx:(buf.idx + ShortBytes)]))
	buf.incrementIndex(ShortBytes)
	return res
}

func (buf *IgniteBuffer) ReadInt32() int32 {
	res := int32(binary.LittleEndian.Uint32(buf.buf[buf.idx:(buf.idx + IntBytes)]))
	buf.incrementIndex(IntBytes)
	return res
}

func (buf *IgniteBuffer) ReadInt64() int64 {
	res := int64(binary.LittleEndian.Uint64(buf.buf[buf.idx:(buf.idx + LongBytes)]))
	buf.incrementIndex(LongBytes)
	return res
}

func (buf *IgniteBuffer) ReadString() *string {
	tp := buf.ReadByte()

	if tp == Null {
		return nil
	} else if tp != String {
		panic(fmt.Sprintf("wrong type. [expecting=%d, actual=%d]", String, tp))
	}

	length := buf.ReadInt()
	res := string(buf.buf[buf.idx:(buf.idx + length)])
	buf.incrementIndex(length)

	return &res
}

func (buf *IgniteBuffer) ReadByteArray() *[]byte {
	tp := buf.ReadByte()

	if tp == Null {
		return nil
	} else if tp != ByteArray {
		panic(fmt.Sprintf("wrong type. [expecting=%d, actual=%d]", ByteArray, tp))
	}

	length := buf.ReadInt()
	res := buf.buf[buf.idx:(buf.idx + length)]
	buf.incrementIndex(length)
	return &res
}

func (buf *IgniteBuffer) ReadStringArray() *[]string {
	tp := buf.ReadByte()

	if tp == Null {
		return nil
	} else if tp != StringArray {
		panic(fmt.Sprintf("wrong type. [expecting=%d, actual=%d]", StringArray, tp))
	}

	return buf.ReadStringArrayWithoutType()
}

func (buf *IgniteBuffer) ReadStringArrayWithoutType() *[]string {
	length := buf.ReadInt()
	strings := make([]string, length)
	for i := 0; i < length; i++ {
		strings[i] = *buf.ReadString()
	}
	return &strings
}

func (buf *IgniteBuffer) ReadUuid() *uuid.UUID {
	tp := buf.ReadByte()

	if tp == Null {
		return nil
	} else if tp != UUID {
		panic(fmt.Sprintf("wrong type. [expecting=%d, actual=%d]", UUID, tp))
	}

	mostSigBits := buf.ReadInt64()
	leastSigBits := buf.ReadInt64()

	bytes := make([]byte, 16)

	binary.BigEndian.PutUint64(bytes, uint64(mostSigBits))
	binary.BigEndian.PutUint64(bytes[8:], uint64(leastSigBits))

	res, err := uuid.FromBytes(bytes)
	if err != nil {
		panic(err)
	}

	return &res
}

func (buf *IgniteBuffer) Reset() {
	buf.Position(0)
	buf.Limit(len(buf.buf))
}

func (buf *IgniteBuffer) Position(pos int) {
	buf.idx = pos
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

func (buf *IgniteBuffer) HasMore() bool {
	return buf.idx < buf.limit
}

func CreateIgniteBuffer() IgniteBuffer {
	return IgniteBuffer{make([]byte, DefaultBufferSize), 0, int(DefaultBufferSize)}
}
