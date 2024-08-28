package ignite

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"math"
	"time"
	"unsafe"
)

type marshaller interface {
	binaryMetadataRegistry
	marshal(ctx context.Context, writer BinaryOutputStream, payload interface{}) error
	unmarshal(ctx context.Context, reader BinaryInputStream) (interface{}, error)
	protocolContext() *ProtocolContext
	isCompactFooter() bool
	binaryIdMapper() BinaryIdMapper
}

type TypeDesc = int8
type mapTypeDesc = int8
type collectionTypeDesc = int8 //lint:ignore U1000 reserved for future

const (
	ByteType TypeDesc = iota + 1
	ShortType
	IntType
	LongType
	FloatType
	DoubleType
	CharType
	BoolType
	StringType
	UuidType
	DateType
	ByteArrayType
	ShortArrayType  //lint:ignore U1000 reserved for future
	IntArrayType    //lint:ignore U1000 reserved for future
	LongArrayType   //lint:ignore U1000 reserved for future
	FloatArrayType  //lint:ignore U1000 reserved for future
	DoubleArrayType //lint:ignore U1000 reserved for future
	CharArrayType   //lint:ignore U1000 reserved for future
	BoolArrayType   //lint:ignore U1000 reserved for future
	StringArrayType //lint:ignore U1000 reserved for future
	UuidArrayType   //lint:ignore U1000 reserved for future
	DateArrayType   //lint:ignore U1000 reserved for future
	ObjectArrayType //lint:ignore U1000 reserved for future
	CollectionType  //lint:ignore U1000 reserved for future
	mapType
	wrappedObjectType TypeDesc = 27
	TimestampType     TypeDesc = 33
	TimeType          TypeDesc = 36
	NullType          TypeDesc = 101
	handleType        TypeDesc = 102 //lint:ignore U1000 reserved for future
	BinaryObjectType  TypeDesc = 103
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

const (
	opGetBinaryConfiguration int16 = 3004
)

type marshallerImpl struct {
	cli           *Client
	reg           binaryMetadataRegistry
	compactFooter bool
	idMapper      BinaryIdMapper
}

func (m *marshallerImpl) protocolContext() *ProtocolContext {
	return m.cli.ch.protocolContext()
}

func (m *marshallerImpl) isCompactFooter() bool {
	return m.compactFooter
}

func (m *marshallerImpl) binaryIdMapper() BinaryIdMapper {
	return m.idMapper
}

func (m *marshallerImpl) getMetadata(ctx context.Context, typeId int32) (*binaryMetadata, error) {
	return m.reg.getMetadata(ctx, typeId)
}

func (m *marshallerImpl) getSchema(ctx context.Context, typeId int32, schemaId int32) (*binarySchema, error) {
	return m.reg.getSchema(ctx, typeId, schemaId)
}

func (m *marshallerImpl) putMetadata(ctx context.Context, typeId int32, meta *binaryMetadata) error {
	return m.reg.putMetadata(ctx, typeId, meta)
}

func (m *marshallerImpl) clearRegistry() {
	m.reg.clearRegistry()
}

func (m *marshallerImpl) checkBinaryConfiguration(ctx context.Context) error {
	var err error
	pCtx := m.protocolContext()
	if pCtx != nil && pCtx.SupportsAttributeFeature(BinaryConfigurationFeature) {
		var srvCompactFooter bool
		m.cli.ch.send(ctx, opGetBinaryConfiguration, func(output BinaryOutputStream) error {
			return nil
		}, func(input BinaryInputStream, err0 error) {
			if err0 != nil {
				err = err0
				return
			}
			srvCompactFooter = input.ReadBool()
			input.ReadInt8() // Ignored
		})
		if err == nil {
			m.compactFooter = srvCompactFooter
		}
	}
	return err
}

func (m *marshallerImpl) marshal(_ context.Context, writer BinaryOutputStream, payload interface{}) error {
	if payload == nil {
		writer.WriteInt8(NullType)
		return nil
	}
	typeId, err := GetTypeId(payload)
	if err != nil {
		return err
	}
	if typeId != BinaryObjectType {
		writer.WriteInt8(typeId)
	}

	switch val := payload.(type) {
	case bool:
		{
			writer.WriteBool(val)
		}
	case uint8:
		{
			writer.WriteUInt8(val)
		}
	case int8:
		{
			writer.WriteInt8(val)
		}
	case uint16:
		{
			writer.WriteUInt16(val)
		}
	case int16:
		{
			writer.WriteInt16(val)
		}
	case uint32:
		{
			writer.WriteUInt32(val)
		}
	case int32:
		{
			writer.WriteInt32(val)
		}
	case int:
		{
			writer.WriteInt32(int32(val))
		}
	case uint64:
		{
			writer.WriteUInt64(val)
		}
	case int64:
		{
			writer.WriteInt64(val)
		}
	case float32:
		{
			writer.WriteUInt32(math.Float32bits(val))
		}
	case float64:
		{
			writer.WriteUInt64(math.Float64bits(val))
		}
	case []byte:
		{
			marshalBytes0(writer, val)
		}
	case string:
		{
			marshalString0(writer, val)
		}
	case uuid.UUID:
		{
			marshalUuid0(writer, val)
		}
	case Time:
		{
			writer.WriteInt64(int64(val))
		}
	case Date:
		{
			writer.WriteInt64(int64(val))
		}
	case time.Time:
		{
			marshalTimestamp0(writer, val)
		}
	case BinaryObject:
		{
			writer.WriteBytes(val.Data())
		}
	default:
		return fmt.Errorf("type '%T' is not supported", val)
	}
	return nil
}

func GetTypeId(val interface{}) (TypeDesc, error) {
	if val == nil {
		return NullType, nil
	}
	switch t := val.(type) {
	case bool:
		{
			return BoolType, nil
		}
	case uint8:
		{
			return ByteType, nil
		}
	case int8:
		{
			return ByteType, nil
		}
	case uint16:
		{
			return CharType, nil
		}
	case int16:
		{
			return ShortType, nil
		}
	case uint32:
		{
			return IntType, nil
		}
	case int32:
		{
			return IntType, nil
		}
	case int:
		{
			return IntType, nil
		}
	case uint64:
		{
			return LongType, nil
		}
	case int64:
		{
			return LongType, nil
		}
	case float32:
		{
			return FloatType, nil
		}
	case float64:
		{
			return DoubleType, nil
		}
	case []byte:
		{
			return ByteArrayType, nil
		}
	case string:
		{
			return StringType, nil
		}
	case uuid.UUID:
		{
			return UuidType, nil
		}
	case Time:
		{
			return TimeType, nil
		}
	case Date:
		{
			return DateType, nil
		}
	case time.Time:
		{
			return TimestampType, nil
		}
	case BinaryObject:
		{
			return BinaryObjectType, nil
		}
	default:
		return -1, fmt.Errorf("type '%T' is not supported", t)
	}

}

func (m *marshallerImpl) unmarshal(_ context.Context, reader BinaryInputStream) (interface{}, error) {
	err := ensureAvailable(reader, 1)
	if err != nil {
		return nil, err
	}
	payloadType := reader.ReadInt8()
	switch payloadType {
	case NullType:
		{
			return nil, nil
		}
	case BoolType:
		{
			err = ensureAvailable(reader, 1)
			if err != nil {
				return nil, err
			}
			return reader.ReadBool(), nil
		}
	case ByteType:
		{
			err = ensureAvailable(reader, 1)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt8(), nil
		}
	case ShortType:
		{
			err = ensureAvailable(reader, 2)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt16(), nil
		}
	case CharType:
		{
			err = ensureAvailable(reader, 2)
			if err != nil {
				return nil, err
			}
			return reader.ReadUInt16(), nil
		}
	case IntType:
		{
			err = ensureAvailable(reader, 4)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt32(), nil
		}
	case LongType:
		{
			err = ensureAvailable(reader, 8)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt64(), nil
		}
	case FloatType:
		{
			err = ensureAvailable(reader, 4)
			if err != nil {
				return nil, err
			}
			return math.Float32frombits(reader.ReadUInt32()), nil

		}
	case DoubleType:
		{
			err = ensureAvailable(reader, 8)
			if err != nil {
				return nil, err
			}
			return math.Float64frombits(reader.ReadUInt64()), nil
		}
	case ByteArrayType:
		{
			return unmarshalBytes(reader, true)
		}
	case StringType:
		{
			return unmarshalString(reader, true)
		}
	case UuidType:
		{
			return unmarshalUuid(reader, true)
		}
	case TimeType:
		{
			return unmarshalTime(reader, true)
		}
	case DateType:
		{
			return unmarshalDate(reader, true)
		}
	case TimestampType:
		{
			return unmarshalTimestamp(reader, true)
		}
	case BinaryObjectType:
		{
			return unmarshalBinaryObject(m, reader, false)
		}
	case wrappedObjectType:
		{
			return unmarshalBinaryObject(m, reader, true)
		}
	default:
		return nil, fmt.Errorf("type %d is not supported", payloadType)
	}
}

func marshalString(writer BinaryOutputStream, val string) {
	writer.WriteInt8(StringType)
	marshalString0(writer, val)
}

func marshalString0(writer BinaryOutputStream, val string) {
	bytes := []byte(val)
	writer.WriteInt32(int32(len(bytes)))
	writer.WriteBytes(bytes)
}

//lint:ignore U1000 reserved for future
func marshalUuid(writer BinaryOutputStream, val uuid.UUID) {
	writer.WriteInt8(UuidType)
	marshalUuid0(writer, val)
}

func marshalUuid0(writer BinaryOutputStream, val uuid.UUID) {
	writer.WriteUInt64(binary.BigEndian.Uint64(val[:8]))
	writer.WriteUInt64(binary.BigEndian.Uint64(val[8:]))
}

//lint:ignore U1000 reserved for future
func marshalTimestamp(writer BinaryOutputStream, val time.Time) {
	writer.WriteInt8(TimestampType)
	marshalTimestamp0(writer, val)
}

func marshalTimestamp0(writer BinaryOutputStream, val time.Time) {
	millis := val.Unix() * 1000
	nanos := val.Nanosecond()
	millis += int64(nanos / int(time.Millisecond))
	nanos %= int(time.Millisecond)
	writer.WriteInt64(millis)
	writer.WriteInt32(int32(nanos))
}

func marshalBytes(writer BinaryOutputStream, val []byte) {
	if val == nil {
		writer.WriteInt8(NullType)
		return
	}
	writer.WriteInt8(ByteArrayType)
	marshalBytes0(writer, val)
}

func marshalBytes0(writer BinaryOutputStream, val []byte) {
	writer.WriteInt32(int32(len(val)))
	writer.WriteBytes(val)
}

func unmarshalBytes(reader BinaryInputStream, skipHeader bool) ([]byte, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return nil, err
		}
		t := reader.ReadInt8()
		switch t {
		case NullType:
			{
				return nil, nil
			}
		case ByteArrayType:
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
	bytesSz := int(reader.ReadInt32())
	if err = ensureAvailable(reader, bytesSz); err != nil {
		return nil, err
	}
	return reader.ReadBytes(bytesSz), nil
}

func unmarshalString(reader BinaryInputStream, skipHeader bool) (string, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return "", err
		}
		t := reader.ReadInt8()
		switch t {
		case NullType:
			{
				return "", nil
			}
		case StringType:
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
	strSz := int(reader.ReadInt32())
	if err = ensureAvailable(reader, strSz); err != nil {
		return "", err
	}
	return bytesToString(reader.ReadBytes(strSz)), nil
}

func unmarshalBinaryObject(m marshaller, reader BinaryInputStream, wrapped bool) (BinaryObject, error) {
	var err error
	var data []byte
	if wrapped {
		err = ensureAvailable(reader, 4)
		if err != nil {
			return nil, err
		}
		sz := int(reader.ReadInt32())
		err = ensureAvailable(reader, sz+4)
		if err != nil {
			return nil, err
		}
		data = reader.ReadBytes(sz)
		offset := reader.ReadInt32()
		data = data[offset:]
	} else {
		if err = ensureAvailable(reader, headerLength); err != nil {
			return nil, err
		}
		startPos := reader.Position() - 1 // First byte of header has been already read
		reader.SetPosition(startPos + lenPos)
		sz := int(reader.ReadInt32())
		reader.SetPosition(startPos)
		if err = ensureAvailable(reader, sz); err != nil {
			return nil, err
		}
		data = reader.ReadBytes(sz)
	}
	return &binaryObjectImpl{
		data:  data,
		marsh: m,
	}, nil
}

func unmarshalUuid(reader BinaryInputStream, skipHeader bool) (uuid.UUID, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return uuid.Nil, err
		}
		t := reader.ReadInt8()
		switch t {
		case NullType:
			{
				return uuid.Nil, nil
			}
		case UuidType:
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
	binary.BigEndian.PutUint64(ret[:8], reader.ReadUInt64())
	binary.BigEndian.PutUint64(ret[8:], reader.ReadUInt64())
	return ret, nil
}

func unmarshalTime(reader BinaryInputStream, skipHeader bool) (Time, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return 0, err
		}
		t := reader.ReadInt8()
		switch t {
		case NullType:
			{
				return 0, nil
			}
		case TimeType:
			{
				break
			}
		default:
			{
				return 0, fmt.Errorf("unexpected type %d in stream", t)
			}
		}
	}
	err = ensureAvailable(reader, 8)
	if err != nil {
		return 0, err
	}
	return Time(reader.ReadInt64()), nil
}

func unmarshalDate(reader BinaryInputStream, skipHeader bool) (Date, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return 0, err
		}
		t := reader.ReadInt8()
		switch t {
		case NullType:
			{
				return 0, nil
			}
		case DateType:
			{
				break
			}
		default:
			{
				return 0, fmt.Errorf("unexpected type %d in stream", t)
			}
		}
	}
	err = ensureAvailable(reader, 8)
	if err != nil {
		return 0, err
	}
	return Date(reader.ReadInt64()), nil
}

func unmarshalTimestamp(reader BinaryInputStream, skipHeader bool) (time.Time, error) {
	var err error
	if !skipHeader {
		if err = ensureAvailable(reader, 1); err != nil {
			return time.Time{}, err
		}
		t := reader.ReadInt8()
		switch t {
		case NullType:
			{
				return time.Time{}, nil
			}
		case TimestampType:
			{
				break
			}
		default:
			{
				return time.Time{}, fmt.Errorf("unexpected type %d in stream", t)
			}
		}
	}
	err = ensureAvailable(reader, 12)
	if err != nil {
		return time.Time{}, err
	}
	millis := reader.ReadInt64()
	nanos := reader.ReadInt32() + int32((millis%1000)*int64(time.Millisecond))
	return time.Unix(millis/1000, int64(nanos)), nil
}

func ensureAvailable(reader BinaryInputStream, nBytes int) error {
	if reader.Available() < nBytes {
		return fmt.Errorf("invalid binary stream")
	}
	return nil
}

func bytesToString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}

func readCollection[T any](reader BinaryInputStream, elemReader func(reader BinaryInputStream) (T, error)) ([]T, error) {
	sz := reader.ReadInt32()
	coll := make([]T, sz)
	for i := 0; i < int(sz); i++ {
		el, err := elemReader(reader)
		if err != nil {
			return nil, err
		}
		coll[i] = el
	}
	return coll, nil
}

func readMap[K comparable, V any](reader BinaryInputStream, kvReader func(BinaryInputStream) (K, V, error)) (map[K]V, error) {
	sz := reader.ReadInt32()
	outMap := make(map[K]V, sz)
	for i := 0; i < int(sz); i++ {
		k, v, err := kvReader(reader)
		if err != nil {
			return nil, err
		}
		outMap[k] = v
	}
	return outMap, nil
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

func writeCollection[T any](writer BinaryOutputStream, values []T, valueWriter func(output BinaryOutputStream, value T) error) error {
	writer.WriteInt32(int32(len(values)))
	for _, value := range values {
		err := valueWriter(writer, value)
		if err != nil {
			return err
		}
	}
	return nil
}
