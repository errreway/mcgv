package ignite

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"math"
	"unsafe"
)

type marshaller interface {
	marshal(ctx context.Context, writer BinaryWriter, payload interface{}) error
	unmarshall(ctx context.Context, reader BinaryReader) (interface{}, error)
	protocolContext() ProtocolContext
}

type typeDesc = int8
type mapTypeDesc = int8
type collectionTypeDesc = int8 //lint:ignore U1000 reserved for future

const (
	byteType typeDesc = iota + 1
	shortType
	intType
	longType
	floatType
	doubleType
	charType
	boolType
	stringType
	uuidType
	dateType //lint:ignore U1000 reserved for future
	byteArrayType
	shortArrayType  //lint:ignore U1000 reserved for future
	intArrayType    //lint:ignore U1000 reserved for future
	longArrayType   //lint:ignore U1000 reserved for future
	floatArrayType  //lint:ignore U1000 reserved for future
	doubleArrayType //lint:ignore U1000 reserved for future
	charArrayType   //lint:ignore U1000 reserved for future
	boolArrayType   //lint:ignore U1000 reserved for future
	stringArrayType //lint:ignore U1000 reserved for future
	uuidArrayType   //lint:ignore U1000 reserved for future
	dateArrayType   //lint:ignore U1000 reserved for future
	objectArrayType //lint:ignore U1000 reserved for future
	collectionType  //lint:ignore U1000 reserved for future
	mapType
	nullType typeDesc = 101
)

const (
	hashMap       mapTypeDesc = iota + 1
	linkedHashMap             //lint:ignore U1000 reserved for future
)

const (
	arrayList     collectionTypeDesc = iota + 1 //lint:ignore U1000 reserved for future
	linkedList                                  //lint:ignore U1000 reserved for future
	hashSet                                     //lint:ignore U1000 reserved for future
	linkedHashSet                               //lint:ignore U1000 reserved for future
	singletonList                               //lint:ignore U1000 reserved for future
)

type marshallerImpl struct {
	cli *clientImpl
}

func newMarshaller(cli *clientImpl) marshaller {
	return &marshallerImpl{
		cli: cli,
	}
}

func (m *marshallerImpl) protocolContext() ProtocolContext {
	return m.cli.ch.ProtocolContext()
}

func (m *marshallerImpl) marshal(ctx context.Context, writer BinaryWriter, payload interface{}) error {
	if payload == nil {
		writer.WriteInt8(nullType)
		return nil
	}
	switch val := payload.(type) {
	case bool:
		{
			writer.WriteInt8(boolType)
			writer.WriteBool(val)
		}
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
	case int:
		{
			writer.WriteInt8(intType)
			writer.WriteInt32(int32(val))
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
		return fmt.Errorf("type '%T' is not supported", val)
	}
	return nil
}

func (m *marshallerImpl) unmarshall(_ context.Context, reader BinaryReader) (interface{}, error) {
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
	case boolType:
		{
			err = ensureAvailable(reader, 1)
			if err != nil {
				return nil, err
			}
			return reader.ReadBool(), nil
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
			if ret, err = unmarshalBytes(reader, true); err != nil {
				return nil, err
			}
			return ret, nil
		}
	case stringType:
		{
			var ret string
			if ret, err = unmarshalString(reader, true); err != nil {
				return "", err
			}
			return ret, nil
		}
	case uuidType:
		{
			var ret uuid.UUID
			if ret, err = unmarshalUuid(reader, true); err != nil {
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

func unmarshalBytes(reader BinaryReader, skipHeader bool) ([]byte, error) {
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

func unmarshalString(reader BinaryReader, skipHeader bool) (string, error) {
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

func unmarshalUuid(reader BinaryReader, skipHeader bool) (uuid.UUID, error) {
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
	ret := uuid.UUID{}
	copy(ret[:], reader.ReadBytes(16))
	return ret, nil
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
