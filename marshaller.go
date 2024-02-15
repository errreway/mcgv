package ignite

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"math"
	"unsafe"
)

type Marshaller interface {
	Marshal(ctx context.Context, writer BinaryWriter, payload interface{}) error
	Unmarshall(ctx context.Context, reader BinaryReader) (interface{}, error)
	ProtocolContext() ProtocolContext
}

const (
	byteType int8 = iota + 1
	shortType
	intType
	longType
	floatType
	doubleType
	charType
	boolType
	stringType
	uuidType
	dateType
	byteArrayType
	shortArrayType
	intArrayType
	longArrayType
	floatArrayType
	doubleArrayType
	charArrayType
	boolArrayType
	stringArrayType
	uuidArrayType
	dateArrayType
	objectArrayType
	nullType int8 = 101
)

type marshallerImpl struct {
	cli *clientImpl
}

func NewMarshaller(cli *clientImpl) Marshaller {
	return &marshallerImpl{
		cli: cli,
	}
}

func (m *marshallerImpl) ProtocolContext() ProtocolContext {
	return m.cli.ch.ProtocolContext()
}

func (m *marshallerImpl) Marshal(ctx context.Context, writer BinaryWriter, payload interface{}) error {
	if payload == nil {
		writer.WriteInt8(nullType)
	}
	switch val := payload.(type) {
	case uint8:
		{
			writer.WriteInt8(byteType)
			writer.WriteUInt8(val)
		}
	case int8:
		{
			writer.WriteInt8(byteType)
			writer.WriteInt8(val)
		}
	case uint16:
		{
			writer.WriteInt8(charType)
			writer.WriteUInt16(val)
		}
	case int16:
		{
			writer.WriteInt8(shortType)
			writer.WriteInt16(val)
		}
	case uint32:
		{
			writer.WriteInt8(intType)
			writer.WriteUInt32(val)
		}
	case int32:
		{
			writer.WriteInt8(intType)
			writer.WriteInt32(val)
		}
	case uint64:
		{
			writer.WriteInt8(longType)
			writer.WriteUInt64(val)
		}
	case int64:
		{
			writer.WriteInt8(longType)
			writer.WriteInt64(val)
		}
	case float32:
		{
			writer.WriteInt8(floatType)
			writer.WriteUInt32(math.Float32bits(val))
		}
	case float64:
		{
			writer.WriteInt8(doubleType)
			writer.WriteUInt64(math.Float64bits(val))
		}
	case []byte:
		{
			marshalBytes(writer, val)
		}
	case string:
		{
			marshalString(writer, val)
		}
	case uuid.UUID:
		{
			writer.WriteInt8(uuidType)
			writer.WriteBytes(val[:])
		}
	default:
		return fmt.Errorf("type %t is not supported", val)
	}
	return nil
}

func (m *marshallerImpl) Unmarshall(_ context.Context, reader BinaryReader) (interface{}, error) {
	err := ensureAvailable(reader, 1)
	if err != nil {
		return nil, err
	}
	payloadType := reader.ReadInt8()
	switch payloadType {
	case nullType:
		{
			return nil, nil
		}
	case byteType:
		{
			err = ensureAvailable(reader, 1)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt8(), nil
		}
	case shortType:
		{
			err = ensureAvailable(reader, 2)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt16(), nil
		}
	case charType:
		{
			err = ensureAvailable(reader, 2)
			if err != nil {
				return nil, err
			}
			return reader.ReadUInt16(), nil
		}
	case intType:
		{
			err = ensureAvailable(reader, 4)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt32(), nil
		}
	case longType:
		{
			err = ensureAvailable(reader, 8)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt64(), nil
		}
	case floatType:
		{
			err = ensureAvailable(reader, 4)
			if err != nil {
				return nil, err
			}
			return math.Float32frombits(reader.ReadUInt32()), nil

		}
	case doubleType:
		{
			err = ensureAvailable(reader, 8)
			if err != nil {
				return nil, err
			}
			return math.Float64frombits(reader.ReadUInt64()), nil
		}
	case byteArrayType:
		{
			var ret []byte = nil
			if ret, err = unmarshallBytes(reader, true); err != nil {
				return nil, err
			}
			return ret, nil
		}
	case stringType:
		{
			var ret string
			if ret, err = unmarshallString(reader, true); err != nil {
				return "", err
			}
			return ret, nil
		}
	case uuidType:
		{
			var ret uuid.UUID
			if ret, err = unmarshallUuid(reader, true); err != nil {
				return uuid.Nil, err
			}
			return ret, nil
		}
	default:
		return nil, fmt.Errorf("type %d is not supported", payloadType)
	}
}

func marshalString(writer BinaryWriter, val string) {
	writer.WriteInt8(stringType)
	bytes := []byte(val)
	writer.WriteInt32(int32(len(bytes)))
	writer.WriteBytes(bytes)
}

func marshalBytes(writer BinaryWriter, val []byte) {
	if val == nil {
		writer.WriteInt8(nullType)
		return
	}
	writer.WriteInt8(byteArrayType)
	writer.WriteInt32(int32(len(val)))
	writer.WriteBytes(val)
}

func unmarshallBytes(reader BinaryReader, skipHeader bool) ([]byte, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return nil, err
		}
		t := reader.ReadInt8()
		switch t {
		case nullType:
			{
				return nil, nil
			}
		case byteArrayType:
			{
				break
			}
		default:
			{
				return nil, fmt.Errorf("unexpected type %d in stream", t)
			}
		}
	}
	if err = ensureAvailable(reader, 4); err != nil {
		return nil, err
	}
	bytesSz := reader.ReadInt32()
	if err = ensureAvailable(reader, bytesSz); err != nil {
		return nil, err
	}
	return reader.ReadBytes(bytesSz), nil
}

func unmarshallString(reader BinaryReader, skipHeader bool) (string, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return "", err
		}
		t := reader.ReadInt8()
		switch t {
		case nullType:
			{
				return "", nil
			}
		case stringType:
			{
				break
			}
		default:
			{
				return "", fmt.Errorf("unexpected type %d in stream", t)
			}
		}
	}
	if err = ensureAvailable(reader, 4); err != nil {
		return "", err
	}
	strSz := reader.ReadInt32()
	if err = ensureAvailable(reader, strSz); err != nil {
		return "", err
	}
	return bytesToString(reader.ReadBytes(strSz)), nil
}

func unmarshallUuid(reader BinaryReader, skipHeader bool) (uuid.UUID, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return uuid.Nil, err
		}
		t := reader.ReadInt8()
		switch t {
		case nullType:
			{
				return uuid.Nil, nil
			}
		case uuidType:
			{
				break
			}
		default:
			{
				return uuid.Nil, fmt.Errorf("unexpected type %d in stream", t)
			}
		}
	}
	if err = ensureAvailable(reader, 16); err != nil {
		return uuid.Nil, err
	}
	return uuid.UUID(reader.ReadBytes(16)), nil
}

func ensureAvailable(reader BinaryReader, nBytes int32) error {
	if reader.Available() < nBytes {
		return fmt.Errorf("invalid binary stream")
	}
	return nil
}

func bytesToString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}
