package ignite

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=TypeDesc

type marshaller interface {
	binaryMetadataRegistry
	marshal(ctx context.Context, writer BinaryOutputStream, payload interface{}) error
	unmarshal(ctx context.Context, reader BinaryInputStream) (interface{}, error)
	protocolContext() *ProtocolContext
	isCompactFooter() bool
	binaryIdMapper() BinaryIdMapper
	registerBinarylizable(factory func() Binarylizable)
	registerEnum(factory func(ord int) Enum)
	clearTypeFactories() // only for testing
}

type MarshallerPlatform int8

const (
	JavaMarshaller MarshallerPlatform = iota
	DotNetMarshaller
)

type TypeDesc int8

const (
	objectType TypeDesc = iota - 1
	unregisteredType
	ByteType
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
	ShortArrayType
	IntArrayType
	LongArrayType
	FloatArrayType
	DoubleArrayType
	CharArrayType
	BoolArrayType
	StringArrayType
	UuidArrayType
	DateArrayType
	ObjectArrayType
	CollectionType
	MapType
	wrappedObjectType  TypeDesc = 27
	EnumType           TypeDesc = 28
	EnumArrayType      TypeDesc = 29
	DecimalType        TypeDesc = 30
	DecimalArrayType   TypeDesc = 31
	ClassType          TypeDesc = 32
	TimestampType      TypeDesc = 33
	TimestampArrayType TypeDesc = 34
	ProxyType          TypeDesc = 35
	TimeType           TypeDesc = 36
	TimeArrayType      TypeDesc = 37
	BinaryEnumType     TypeDesc = 38
	NullType           TypeDesc = 101
	handleType         TypeDesc = 102 //lint:ignore U1000 reserved for future
	BinaryObjectType   TypeDesc = 103
)

const (
	opGetBinaryConfiguration int16 = 3004
)

type marshallerImpl struct {
	cli           *Client
	reg           binaryMetadataRegistry
	compactFooter bool
	idMapper      BinaryIdMapper
	typeFactories sync.Map
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

func (m *marshallerImpl) putMetadata(ctx context.Context, meta *binaryMetadata) error {
	return m.reg.putMetadata(ctx, meta)
}

func (m *marshallerImpl) getClassName(ctx context.Context, platform MarshallerPlatform, typeId int32) (string, error) {
	return m.reg.getClassName(ctx, platform, typeId)
}

func (m *marshallerImpl) registerClassName(ctx context.Context, platform MarshallerPlatform, typeId int32, className string) error {
	return m.reg.registerClassName(ctx, platform, typeId, className)
}

func (m *marshallerImpl) clearRegistry() {
	m.reg.clearRegistry()
}

func (m *marshallerImpl) registerBinarylizable(factory func() Binarylizable) {
	empty := factory()
	m.registerTypeFactory(empty, factory)
}

func (m *marshallerImpl) registerEnum(factory func(ord int) Enum) {
	empty := factory(0)
	m.registerTypeFactory(empty, factory)
}

func (m *marshallerImpl) registerTypeFactory(emptyVal interface{}, factory interface{}) {
	typeName, err := typeNameByReflection(emptyVal)
	if err != nil {
		panic(fmt.Errorf("failed to register type %T: %w", emptyVal, err))
	}
	typeId := m.idMapper.TypeId(typeName)
	m.typeFactories.Store(typeId, factory)
}

func (m *marshallerImpl) getBinarylizableFactory(typeId int32) (func() Binarylizable, bool) {
	val, ok := m.typeFactories.Load(typeId)
	if ok {
		factory, ok := val.(func() Binarylizable)
		if !ok {
			return nil, false
		}
		return factory, ok
	} else {
		return nil, false
	}
}

func (m *marshallerImpl) getEnumFactory(typeId int32) (func(int) Enum, bool) {
	val, ok := m.typeFactories.Load(typeId)
	if ok {
		factory, ok := val.(func(int) Enum)
		if !ok {
			return nil, false
		}
		return factory, ok
	} else {
		return nil, false
	}
}

func (m *marshallerImpl) clearTypeFactories() {
	m.typeFactories.Range(func(key, value interface{}) bool {
		m.typeFactories.Delete(key)
		return true
	})
}

func (m *marshallerImpl) checkBinaryConfiguration(ctx context.Context) error {
	var err error
	pCtx := m.protocolContext()
	if pCtx != nil && pCtx.SupportsAttributeFeature(BinaryConfigurationFeature) {
		var srvCompactFooter bool
		m.cli.ch.send(ctx, opGetBinaryConfiguration, func(_ channel, output BinaryOutputStream) error {
			return nil
		}, func(_ channel, input BinaryInputStream, err0 error) {
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

func (m *marshallerImpl) marshal(ctx context.Context, writer BinaryOutputStream, payload interface{}) error {
	if m.isNil(payload) {
		writer.WriteNull()
		return nil
	}

	switch val := payload.(type) {
	case bool:
		{
			writer.WriteType(BoolType)
			writer.WriteBool(val)
		}
	case uint8:
		{
			writer.WriteType(ByteType)
			writer.WriteUInt8(val)
		}
	case int8:
		{
			writer.WriteType(ByteType)
			writer.WriteInt8(val)
		}
	case uint16:
		{
			writer.WriteType(CharType)
			writer.WriteUInt16(val)
		}
	case int16:
		{
			writer.WriteType(ShortType)
			writer.WriteInt16(val)
		}
	case uint32:
		{
			writer.WriteType(IntType)
			writer.WriteUInt32(val)
		}
	case int32:
		{
			writer.WriteType(IntType)
			writer.WriteInt32(val)
		}
	case uint:
		{
			writer.WriteType(LongType)
			writer.WriteUInt64(uint64(val))
		}
	case int:
		{
			writer.WriteType(LongType)
			writer.WriteInt64(int64(val))
		}
	case uint64:
		{
			writer.WriteType(LongType)
			writer.WriteUInt64(val)
		}
	case int64:
		{
			writer.WriteType(LongType)
			writer.WriteInt64(val)
		}
	case float32:
		{
			writer.WriteType(FloatType)
			writer.WriteFloat32(val)
		}
	case float64:
		{
			writer.WriteType(DoubleType)
			writer.WriteFloat64(val)
		}
	case []bool:
		{
			writer.WriteType(BoolArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteBoolSlice(val)
		}
	case []byte:
		{
			marshalByteArray(writer, val)
		}
	case []int8:
		{
			writer.WriteType(ByteArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteInt8Slice(val)
		}
	case []uint16:
		{
			writer.WriteType(CharArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteUInt16Slice(val)
		}
	case []int16:
		{
			writer.WriteType(ShortArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteInt16Slice(val)
		}
	case []uint32:
		{
			writer.WriteType(IntArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteUInt32Slice(val)
		}
	case []int32:
		{
			writer.WriteType(IntArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteInt32Slice(val)
		}
	case []uint:
		{
			writer.WriteType(LongArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteUIntSlice(val)
		}
	case []int:
		{
			writer.WriteType(LongArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteIntSlice(val)
		}
	case []uint64:
		{
			writer.WriteType(LongArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteUInt64Slice(val)
		}
	case []int64:
		{
			writer.WriteType(LongArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteInt64Slice(val)
		}
	case []float32:
		{
			writer.WriteType(FloatArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteFloat32Slice(val)
		}
	case []float64:
		{
			writer.WriteType(DoubleArrayType)
			writer.WriteInt32(int32(len(val)))
			writer.WriteFloat64Slice(val)
		}
	case string:
		{
			marshalString(writer, val)
		}
	case []string:
		{
			writer.WriteType(StringArrayType)
			err := writeIgniteTypedArray(writer, StringType, val, func(output BinaryOutputStream, el *string) error {
				writeString(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []*string:
		{
			writer.WriteType(StringArrayType)
			err := writeIgniteTypedArrayP(writer, StringType, val, func(output BinaryOutputStream, el *string) error {
				writeString(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case uuid.UUID:
		{
			marshalUuid(writer, &val)
		}
	case []uuid.UUID:
		{
			writer.WriteType(UuidArrayType)
			err := writeIgniteTypedArray(writer, UuidType, val, func(output BinaryOutputStream, el *uuid.UUID) error {
				writeUuid(output, el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []*uuid.UUID:
		{
			writer.WriteType(UuidArrayType)
			err := writeIgniteTypedArrayP(writer, UuidType, val, func(output BinaryOutputStream, el *uuid.UUID) error {
				writeUuid(output, el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case Time:
		{
			writer.WriteType(TimeType)
			writeTime(writer, val)
		}
	case []Time:
		{
			writer.WriteType(TimeArrayType)
			err := writeIgniteTypedArray(writer, TimeType, val, func(output BinaryOutputStream, el *Time) error {
				writeTime(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []*Time:
		{
			writer.WriteType(TimeArrayType)
			err := writeIgniteTypedArrayP(writer, TimeType, val, func(output BinaryOutputStream, el *Time) error {
				writeTime(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case Date:
		{
			writer.WriteType(DateType)
			writeDate(writer, val)
		}
	case []Date:
		{
			writer.WriteType(DateArrayType)
			err := writeIgniteTypedArray(writer, DateType, val, func(output BinaryOutputStream, el *Date) error {
				writeDate(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []*Date:
		{
			writer.WriteType(DateArrayType)
			err := writeIgniteTypedArrayP(writer, DateType, val, func(output BinaryOutputStream, el *Date) error {
				writeDate(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case time.Time:
		{
			writer.WriteType(TimestampType)
			writeTimestamp(writer, val)
		}
	case []time.Time:
		{
			writer.WriteType(TimestampArrayType)
			err := writeIgniteTypedArray(writer, TimestampType, val, func(output BinaryOutputStream, el *time.Time) error {
				writeTimestamp(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []*time.Time:
		{
			writer.WriteType(TimestampArrayType)
			err := writeIgniteTypedArrayP(writer, TimestampType, val, func(output BinaryOutputStream, el *time.Time) error {
				writeTimestamp(output, *el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case *apd.Decimal:
		{
			writer.WriteType(DecimalType)
			writeDecimal(writer, val)
		}
	case apd.Decimal:
		{
			writer.WriteType(DecimalType)
			writeDecimal(writer, &val)
		}
	case []*apd.Decimal:
		{
			writer.WriteType(DecimalArrayType)
			err := writeIgniteTypedArrayP(writer, DecimalType, val, func(output BinaryOutputStream, el *apd.Decimal) error {
				writeDecimal(output, el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []apd.Decimal:
		{
			writer.WriteType(DecimalArrayType)
			err := writeIgniteTypedArray(writer, DecimalType, val, func(output BinaryOutputStream, el *apd.Decimal) error {
				writeDecimal(output, el)
				return nil
			})
			if err != nil {
				return err
			}
		}
	case []interface{}:
		{
			if err := m.marshallObjectArray(ctx, writer, val); err != nil {
				return err
			}
		}
	case Collection:
		{
			if err := m.marshallCollection(ctx, writer, val); err != nil {
				return err
			}
		}
	case Map:
		{
			if err := m.marshallMap(ctx, writer, val); err != nil {
				return err
			}
		}
	case BinaryObject:
		{
			writer.WriteBytes(val.Data())
		}
	case Binarylizable:
		{
			if err := marshalBinarylizable(ctx, m, writer, val); err != nil {
				return err
			}
		}
	case BinaryEnumArray:
		{
			if val.IsNull() {
				writer.WriteType(NullType)
			} else {
				writer.WriteType(EnumArrayType)
				writeClass(writer, &igniteType{
					typeId:   val.elType.TypeId(),
					typeName: val.elType.TypeName(),
				})
				err := writeSequence(writer, len(val.data), func(output BinaryOutputStream, idx int) error {
					enum := val.data[idx]
					if enum != nil {
						writer.WriteBytes(enum.Data())
					} else {
						writer.WriteType(NullType)
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
		}
	case Enum:
		if err := marshalEnum(ctx, m, writer, val); err != nil {
			return err
		}
	default:
		t := reflect.TypeOf(val)
		switch t.Kind() {
		case reflect.Map:
			{
				if err := m.marshallReflectMap(ctx, writer, reflect.ValueOf(val)); err != nil {
					return err
				}
			}
		case reflect.Slice:
			{
				if err := m.marshallReflectSlice(ctx, writer, reflect.ValueOf(val)); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("type '%T' is not supported", val)
		}
	}
	return nil
}

func (m *marshallerImpl) isNil(payload any) bool {
	if payload == nil {
		return true
	}
	v := reflect.ValueOf(payload)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func getTypeId(val interface{}) (TypeDesc, error) {
	if val == nil {
		return NullType, nil
	}
	switch t := val.(type) {
	case bool:
		{
			return BoolType, nil
		}
	case uint8, int8:
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
	case uint32, int32:
		{
			return IntType, nil
		}
	case uint64, int64, uint, int:
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
	case []byte, []int8:
		{
			return ByteArrayType, nil
		}
	case []int16:
		{
			return ShortArrayType, nil
		}
	case []uint16:
		{
			return CharArrayType, nil
		}
	case []uint32, []int32:
		{
			return IntArrayType, nil
		}
	case []uint64, []int64, []uint, []int:
		{
			return LongArrayType, nil
		}
	case []float32:
		{
			return FloatArrayType, nil
		}
	case []float64:
		{
			return DoubleArrayType, nil
		}
	case string:
		{
			return StringType, nil
		}
	case []string, []*string:
		{
			return StringArrayType, nil
		}
	case uuid.UUID:
		{
			return UuidType, nil
		}
	case []uuid.UUID, []*uuid.UUID:
		{
			return UuidArrayType, nil
		}
	case Time:
		{
			return TimeType, nil
		}
	case []Time, []*Time:
		{
			return TimeArrayType, nil
		}
	case Date:
		{
			return DateType, nil
		}
	case []Date, []*Date:
		{
			return DateArrayType, nil
		}
	case time.Time:
		{
			return TimestampType, nil
		}
	case []time.Time, []*time.Time:
		{
			return TimestampArrayType, nil
		}
	case *apd.Decimal, apd.Decimal:
		{
			return DecimalType, nil
		}
	case []*apd.Decimal, []apd.Decimal:
		{
			return DecimalArrayType, nil
		}
	case BinaryObject:
		{
			if t.IsEnum() {
				return EnumType, nil
			} else {
				return BinaryObjectType, nil
			}
		}
	case Binarylizable:
		{
			return BinaryObjectType, nil
		}
	case BinaryEnumArray:
		{
			return EnumArrayType, nil
		}
	case Enum:
		{
			return EnumType, nil
		}
	case []interface{}:
		{
			return ObjectArrayType, nil
		}
	case Map:
		{
			return MapType, nil
		}
	case Collection:
		{
			return CollectionType, nil
		}
	default:
		{
			t := reflect.TypeOf(val)
			switch t.Kind() {
			case reflect.Slice:
				_, ok := reflect.Zero(t.Elem()).Interface().(Enum)
				if ok {
					return EnumArrayType, nil
				}
				return ObjectArrayType, nil
			case reflect.Map:
				return MapType, nil
			default:
				return -1, fmt.Errorf("type '%T' is not supported", t)
			}
		}
	}

}

func (m *marshallerImpl) unmarshal(ctx context.Context, reader BinaryInputStream) (interface{}, error) {
	err := ensureAvailable(reader, byteBytes)
	if err != nil {
		return nil, err
	}
	payloadType := TypeDesc(reader.ReadInt8())
	switch payloadType {
	case NullType:
		{
			return nil, nil
		}
	case BoolType:
		{
			err = ensureAvailable(reader, boolBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadBool(), nil
		}
	case ByteType:
		{
			err = ensureAvailable(reader, byteBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt8(), nil
		}
	case ShortType:
		{
			err = ensureAvailable(reader, shortBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt16(), nil
		}
	case CharType:
		{
			err = ensureAvailable(reader, charBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadUInt16(), nil
		}
	case IntType:
		{
			err = ensureAvailable(reader, intBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt32(), nil
		}
	case LongType:
		{
			err = ensureAvailable(reader, longBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadInt64(), nil
		}
	case FloatType:
		{
			err = ensureAvailable(reader, intBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadFloat32(), nil

		}
	case DoubleType:
		{
			err = ensureAvailable(reader, longBytes)
			if err != nil {
				return nil, err
			}
			return reader.ReadFloat64(), nil
		}
	case ByteArrayType:
		{
			return readByteArray(reader)
		}
	case BoolArrayType, ShortArrayType, CharArrayType, IntArrayType, LongArrayType, FloatArrayType, DoubleArrayType:
		{
			var elemSz int
			switch payloadType {
			case BoolArrayType:
				elemSz = boolBytes
			case ShortArrayType:
				elemSz = shortBytes
			case CharArrayType:
				elemSz = charBytes
			case IntArrayType, FloatArrayType:
				elemSz = intBytes
			case LongArrayType, DoubleArrayType:
				elemSz = longBytes
			default:
				panic("impossible condition")
			}
			err = ensureAvailable(reader, intBytes)
			if err != nil {
				return nil, err
			}
			sz := int(reader.ReadInt32())
			err = ensureAvailable(reader, sz*elemSz)
			if err != nil {
				return nil, err
			}
			switch payloadType {
			case BoolArrayType:
				return reader.ReadBoolSlice(sz), nil
			case ShortArrayType:
				return reader.ReadInt16Slice(sz), nil
			case CharArrayType:
				return reader.ReadUInt16Slice(sz), nil
			case IntArrayType:
				return reader.ReadInt32Slice(sz), nil
			case LongArrayType:
				return reader.ReadInt64Slice(sz), nil
			case FloatArrayType:
				return reader.ReadFloat32Slice(sz), nil
			case DoubleArrayType:
				return reader.ReadFloat64Slice(sz), nil
			default:
				panic("impossible condition")
			}
		}
	case StringType:
		{
			return readString(reader)
		}
	case StringArrayType:
		{
			return readIgniteTypedArray[string](reader, StringType, func(_ int, reader BinaryInputStream) (*string, error) {
				ret, err := readString(reader)
				if err != nil {
					return nil, err
				}
				return &ret, err
			})
		}
	case UuidType:
		{
			return readUuid(reader)
		}
	case UuidArrayType:
		{
			return readIgniteTypedArray[uuid.UUID](reader, UuidType, func(_ int, reader BinaryInputStream) (*uuid.UUID, error) {
				ret, err := readUuid(reader)
				if err != nil {
					return nil, err
				}
				return &ret, err
			})
		}
	case TimeType:
		{
			return readTime(reader)
		}
	case TimeArrayType:
		{
			return readIgniteTypedArray[Time](reader, TimeType, func(_ int, reader BinaryInputStream) (*Time, error) {
				ret, err := readTime(reader)
				if err != nil {
					return nil, err
				}
				return &ret, err
			})
		}
	case DateType:
		{
			return readDate(reader)
		}
	case DateArrayType:
		{
			return readIgniteTypedArray[Date](reader, DateType, func(_ int, reader BinaryInputStream) (*Date, error) {
				ret, err := readDate(reader)
				if err != nil {
					return nil, err
				}
				return &ret, err
			})
		}
	case TimestampType:
		{
			return readTimestamp(reader)
		}
	case TimestampArrayType:
		{
			return readIgniteTypedArray[time.Time](reader, TimestampType, func(_ int, reader BinaryInputStream) (*time.Time, error) {
				ret, err := readTimestamp(reader)
				if err != nil {
					return nil, err
				}
				return &ret, err
			})
		}
	case DecimalType:
		{
			return readDecimal(reader)
		}
	case DecimalArrayType:
		{
			return readIgniteTypedArray[apd.Decimal](reader, DecimalType, func(_ int, reader BinaryInputStream) (*apd.Decimal, error) {
				return readDecimal(reader)
			})
		}
	case ObjectArrayType:
		{
			return m.readIgniteObjectArray(ctx, reader)
		}
	case CollectionType:
		{
			return m.readIgniteCollection(ctx, reader)
		}
	case MapType:
		{
			return m.readIgniteMap(ctx, reader)
		}
	case BinaryObjectType:
		{
			bo, err := m.readBinaryObject(reader, false)
			if err != nil {
				return nil, err
			}
			return m.tryConvertToBinarylizable(ctx, bo)
		}
	case EnumType, BinaryEnumType, EnumArrayType:
		{
			startPos := reader.Position() - 1
			enumCls, err := readClass(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to read enum cls: %w", err)
			}
			factory, hasFactory := m.getEnumFactory(enumCls.typeId)
			if payloadType == EnumArrayType {
				if hasFactory {
					return m.readEnumArray(reader, factory)
				}
				return m.readBinaryEnumArray(ctx, reader, &enumCls)
			} else {
				if hasFactory {
					return m.readEnum(reader, factory)
				}
				return m.readBinaryEnum(ctx, reader, &enumCls, startPos)
			}
		}
	case wrappedObjectType:
		{
			bo, err := m.readBinaryObject(reader, true)
			if err != nil {
				return nil, err
			}
			return m.tryConvertToBinarylizable(ctx, bo)
		}
	case ClassType:
		return readClass(reader)
	default:
		return nil, fmt.Errorf("type %d is not supported", payloadType)
	}
}

func (m *marshallerImpl) tryConvertToBinarylizable(ctx context.Context, bo BinaryObject) (interface{}, error) {
	typeId, err := bo.getTypeId()
	if err != nil {
		return nil, fmt.Errorf("failed to get binary object type: %w", err)
	}
	factory, ok := m.getBinarylizableFactory(typeId)
	if !ok {
		return bo, nil
	}
	meta, err := m.getMetadata(ctx, typeId)
	if err != nil {
		return nil, fmt.Errorf("failed to get binary object metadata: %w", err)
	}
	reader, err := newBinaryReaderImpl(bo, meta)
	if err != nil {
		return nil, fmt.Errorf("failed to create binary object reader: %w", err)
	}
	ret := factory()
	if err = ret.Read(ctx, reader); err != nil {
		return nil, fmt.Errorf("failed to read %T object: %w", ret, err)
	}
	return ret, nil
}

func (m *marshallerImpl) readBinaryObject(reader BinaryInputStream, wrapped bool) (BinaryObject, error) {
	var err error
	var data []byte
	if wrapped {
		err = ensureAvailable(reader, intBytes)
		if err != nil {
			return nil, err
		}
		sz := int(reader.ReadInt32())
		err = ensureAvailable(reader, sz+intBytes)
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

func (m *marshallerImpl) readBinaryEnumArray(ctx context.Context, reader BinaryInputStream, cls *igniteType) (arr BinaryEnumArray, err error) {
	typeId := cls.typeId
	if typeId == int32(unregisteredType) {
		typeId = m.binaryIdMapper().TypeId(cls.typeName)
	}
	meta, err0 := m.getMetadata(ctx, typeId)
	if err0 != nil {
		err = fmt.Errorf("failed to read enum array: %w", err)
		return
	}
	if meta == nil {
		err = fmt.Errorf("no metadata found for typeId=%d", typeId)
		return
	}
	arr.elType = meta
	arr.data, err = readSlice[BinaryObject](reader, func(i int, stream BinaryInputStream) (BinaryObject, error) {
		if err0 := ensureAvailable(reader, byteBytes); err0 != nil {
			return nil, err0
		}
		t := TypeDesc(reader.ReadInt8())
		switch t {
		case NullType:
			return nil, nil
		case EnumType, BinaryEnumType:
			startPos := reader.Position() - 1
			enumCls, err0 := readClass(reader)
			if err0 != nil {
				return nil, fmt.Errorf("failed to read enum array: %w", err0)
			}
			return m.readBinaryEnum(ctx, stream, &enumCls, startPos)
		default:
			return nil, fmt.Errorf("failed to read enum array, type is not expected: %s", t)
		}
	})
	if err == nil {
		arr.isNotNull = true
	}
	return
}

func (m *marshallerImpl) readBinaryEnum(ctx context.Context, reader BinaryInputStream, enumCls *igniteType, startPos int) (BinaryObject, error) {
	enum := &binaryEnum{
		typeId:   enumCls.typeId,
		typeName: enumCls.typeName,
		marsh:    m,
	}
	if err := ensureAvailable(reader, intBytes); err != nil {
		return nil, err
	}
	enum.ord = reader.ReadInt32()
	size := reader.Position() - startPos
	reader.SetPosition(startPos)
	enum.data = reader.ReadBytes(size)
	enum.data[0] = byte(EnumType)
	meta, err := m.getMetadata(ctx, enum.typeId)
	if err != nil {
		return nil, fmt.Errorf("failed to get binary object metadata: %w", err)
	}
	if !meta.isEnum {
		return nil, fmt.Errorf("invalid data from server: type id=%d is not an enum: %s", enum.typeId, enum.typeName)
	}
	enum.name = meta.ordToName[enum.ord]
	return enum, nil
}

func (m *marshallerImpl) readEnumArray(reader BinaryInputStream, factory func(int) Enum) (ret interface{}, err error) {
	if err = ensureAvailable(reader, intBytes); err != nil {
		return
	}
	sz := int(reader.ReadInt32())
	empty := factory(0)
	enumT := reflect.TypeOf(empty)
	slice := reflect.MakeSlice(reflect.SliceOf(enumT), 0, sz)
	zero := reflect.Zero(enumT)

	err = readSequence(reader, sz, func(i int, stream BinaryInputStream) error {
		if err0 := ensureAvailable(reader, byteBytes); err0 != nil {
			return err0
		}
		t := TypeDesc(reader.ReadInt8())
		switch t {
		case NullType:
			slice = reflect.Append(slice, zero)
			return nil
		case EnumType, BinaryEnumType:
			_, err0 := readClass(reader)
			if err0 != nil {
				return fmt.Errorf("failed to read enum array: %w", err0)
			}
			enum, err0 := m.readEnum(reader, factory)
			if err0 != nil {
				return fmt.Errorf("failed to read enum array: %w", err0)
			}
			slice = reflect.Append(slice, reflect.ValueOf(enum))
			return nil
		default:
			return fmt.Errorf("failed to read enum array, type is not expected: %s", t)
		}
	})
	if err == nil {
		ret = slice.Interface()
	}
	return
}

func (m *marshallerImpl) readEnum(reader BinaryInputStream, factory func(int) Enum) (Enum, error) {
	if err := ensureAvailable(reader, intBytes); err != nil {
		return nil, err
	}
	ord := int(reader.ReadInt32())
	return factory(ord), nil
}

func (m *marshallerImpl) readIgniteObjectArray(ctx context.Context, reader BinaryInputStream) ([]interface{}, error) {
	if _, err := readClass(reader); err != nil {
		return nil, fmt.Errorf("failed to read object array: %w", err)
	}
	return readSlice(reader, func(_ int, reader BinaryInputStream) (interface{}, error) {
		return m.unmarshal(ctx, reader)
	})
}

func (m *marshallerImpl) readIgniteCollection(ctx context.Context, reader BinaryInputStream) (coll Collection, err error) {
	if err = ensureAvailable(reader, intBytes); err != nil {
		return
	}
	sz := int(reader.ReadInt32())
	if err = ensureAvailable(reader, byteBytes); err != nil {
		return
	}
	coll.kind = CollectionKind(reader.ReadInt8())
	coll.values = make([]interface{}, sz)
	err = readSequence(reader, sz, func(idx int, reader0 BinaryInputStream) error {
		el, err0 := m.unmarshal(ctx, reader0)
		if err0 != nil {
			return err0
		}
		coll.values[idx] = el
		return nil
	})
	if err == nil {
		coll.isNotNull = true
	}
	return
}

func (m *marshallerImpl) readIgniteMap(ctx context.Context, reader BinaryInputStream) (coll Map, err error) {
	if err = ensureAvailable(reader, intBytes); err != nil {
		return
	}
	sz := int(reader.ReadInt32())
	if err = ensureAvailable(reader, byteBytes); err != nil {
		return
	}
	coll.kind = MapKind(reader.ReadInt8())
	coll.entries = make([]KeyValue, sz)
	err = readSequence(reader, sz, func(idx int, reader0 BinaryInputStream) error {
		var key, val interface{}
		var err0 error
		if key, err0 = m.unmarshal(ctx, reader0); err0 != nil {
			return err0
		}
		if val, err0 = m.unmarshal(ctx, reader0); err0 != nil {
			return err0
		}
		coll.entries[idx] = KeyValue{key, val}
		return nil
	})
	if err == nil {
		coll.isNotNull = true
	}
	return
}

func (m *marshallerImpl) marshallObjectArray(ctx context.Context, writer BinaryOutputStream, objArr []interface{}) error {
	if objArr == nil {
		writer.WriteType(NullType)
		return nil
	}
	writer.WriteType(ObjectArrayType)
	writer.WriteInt32(int32(objectType))
	return writeSequence(writer, len(objArr), func(output BinaryOutputStream, idx int) error {
		if err := m.marshal(ctx, output, objArr[idx]); err != nil {
			return err
		}
		return nil
	})
}

func (m *marshallerImpl) marshallReflectSlice(ctx context.Context, writer BinaryOutputStream, val reflect.Value) error {
	elType := val.Type().Elem()
	zVal, ok := reflect.Zero(elType).Interface().(Enum)
	if ok {
		writer.WriteType(EnumArrayType)

		typeName, err := typeNameByReflection(zVal)
		if err != nil {
			return fmt.Errorf("failed to marshal enum array: %w", err)
		}
		typeId := m.binaryIdMapper().TypeId(typeName)
		writeClass(writer, &igniteType{
			typeId:   typeId,
			typeName: typeName,
		})
	} else {
		writer.WriteType(ObjectArrayType)
		writer.WriteInt32(int32(objectType))
	}
	return writeSequence(writer, val.Len(), func(output BinaryOutputStream, idx int) error {
		value := val.Index(idx).Interface()
		return m.marshal(ctx, output, value)
	})
}

func (m *marshallerImpl) marshallCollection(ctx context.Context, writer BinaryOutputStream, collection Collection) error {
	if collection.IsNull() {
		writer.WriteType(NullType)
		return nil
	}
	writer.WriteType(CollectionType)
	values := collection.Values()
	return writeSequenceWithKind(writer, len(values), int8(collection.Kind()), func(output BinaryOutputStream, idx int) error {
		if err := m.marshal(ctx, output, values[idx]); err != nil {
			return err
		}
		return nil
	})
}

func (m *marshallerImpl) marshallMap(ctx context.Context, writer BinaryOutputStream, collection Map) error {
	if collection.IsNull() {
		writer.WriteType(NullType)
		return nil
	}
	writer.WriteType(MapType)
	entries := collection.Entries()
	return writeSequenceWithKind(writer, len(entries), int8(collection.Kind()), func(output BinaryOutputStream, idx int) error {
		entry := entries[idx]
		if err := m.marshal(ctx, output, entry.Key); err != nil {
			return err
		}
		if err := m.marshal(ctx, output, entry.Value); err != nil {
			return err
		}
		return nil
	})
}

func (m *marshallerImpl) marshallReflectMap(ctx context.Context, writer BinaryOutputStream, collection reflect.Value) error {
	if collection.IsNil() {
		writer.WriteType(NullType)
		return nil
	}
	writer.WriteType(MapType)
	keys := collection.MapKeys()
	return writeSequenceWithKind(writer, len(keys), int8(HashMap), func(output BinaryOutputStream, idx int) error {
		key := keys[idx]
		if err := m.marshal(ctx, output, key.Interface()); err != nil {
			return err
		}
		if err := m.marshal(ctx, output, collection.MapIndex(key).Interface()); err != nil {
			return err
		}
		return nil
	})
}

func marshalString(writer BinaryOutputStream, val string) {
	writer.WriteType(StringType)
	writeString(writer, val)
}

func writeString(writer BinaryOutputStream, val string) {
	bytes := []byte(val)
	writer.WriteInt32(int32(len(bytes)))
	writer.WriteBytes(bytes)
}

func marshalEmptyStringAsNull(writer BinaryOutputStream, val string) {
	if len(val) == 0 {
		writer.WriteNull()
	} else {
		marshalString(writer, val)
	}
}

func marshalByteArray(writer BinaryOutputStream, val []byte) {
	if val == nil {
		writer.WriteNull()
		return
	}
	writer.WriteType(ByteArrayType)
	writer.WriteInt32(int32(len(val)))
	writer.WriteBytes(val)
}

func marshalUuid(writer BinaryOutputStream, val *uuid.UUID) {
	if val == nil {
		writer.WriteNull()
	} else {
		writer.WriteType(UuidType)
		writer.WriteUInt64(binary.BigEndian.Uint64(val[:8]))
		writer.WriteUInt64(binary.BigEndian.Uint64(val[8:]))
	}
}

func writeUuid(writer BinaryOutputStream, val *uuid.UUID) {
	writer.WriteUInt64(binary.BigEndian.Uint64(val[:8]))
	writer.WriteUInt64(binary.BigEndian.Uint64(val[8:]))
}

func writeDate(writer BinaryOutputStream, val Date) {
	writer.WriteInt64(int64(val))
}

func writeTime(writer BinaryOutputStream, val Time) {
	writer.WriteInt64(int64(val))
}

func writeTimestamp(writer BinaryOutputStream, val time.Time) {
	millis := val.Unix() * 1000
	nanos := val.Nanosecond()
	millis += int64(nanos / int(time.Millisecond))
	nanos %= int(time.Millisecond)
	writer.WriteInt64(millis)
	writer.WriteInt32(int32(nanos))
}

func writeDecimal(writer BinaryOutputStream, val *apd.Decimal) {
	writer.WriteInt32(-val.Exponent)
	coeff := val.Coeff.Bytes()
	if len(coeff) == 0 || coeff[0] > 0x7F {
		tmp := make([]byte, len(coeff)+1)
		copy(tmp[1:], coeff)
		coeff = tmp
	}
	if val.Negative {
		coeff[0] |= 0x80
	}
	writer.WriteInt32(int32(len(coeff)))
	writer.WriteBytes(coeff)
}

func checkNotNull(reader BinaryInputStream, expected TypeDesc) (bool, error) {
	var err error
	if err = ensureAvailable(reader, byteBytes); err != nil {
		return false, err
	}
	t := TypeDesc(reader.ReadInt8())
	switch t {
	case NullType:
		{
			return false, nil
		}
	case expected:
		{
			return true, nil
		}
	default:
		{
			return false, fmt.Errorf("unexpected type %s in stream", t)
		}
	}
}

func readByteArray(reader BinaryInputStream) ([]byte, error) {
	if err := ensureAvailable(reader, intBytes); err != nil {
		return nil, err
	}
	bytesSz := int(reader.ReadInt32())
	if err := ensureAvailable(reader, bytesSz); err != nil {
		return nil, err
	}
	return reader.ReadBytes(bytesSz), nil
}

func unmarshalByteArray(reader BinaryInputStream) ([]byte, error) {
	isNotNull, err := checkNotNull(reader, ByteArrayType)
	if err != nil || !isNotNull {
		return nil, err
	}
	return readByteArray(reader)
}

func readString(reader BinaryInputStream) (string, error) {
	if err := ensureAvailable(reader, intBytes); err != nil {
		return "", err
	}
	strSz := int(reader.ReadInt32())
	if err := ensureAvailable(reader, strSz); err != nil {
		return "", err
	}
	return bytesToString(reader.ReadBytes(strSz)), nil
}

func unmarshalString(reader BinaryInputStream) (string, error) {
	isNotNull, err := checkNotNull(reader, StringType)
	if err != nil || !isNotNull {
		return "", err
	}
	return readString(reader)
}

func readUuid(reader BinaryInputStream) (uuid.UUID, error) {
	if err := ensureAvailable(reader, 2*longBytes); err != nil {
		return uuid.Nil, err
	}
	ret := uuid.UUID{}
	binary.BigEndian.PutUint64(ret[:8], reader.ReadUInt64())
	binary.BigEndian.PutUint64(ret[8:], reader.ReadUInt64())
	return ret, nil
}

func unmarshalUuid(reader BinaryInputStream) (uuid.UUID, error) {
	isNotNull, err := checkNotNull(reader, UuidType)
	if err != nil || !isNotNull {
		return uuid.Nil, err
	}
	return readUuid(reader)
}

func readClass(reader BinaryInputStream) (ret igniteType, err error) {
	err = ensureAvailable(reader, intBytes)
	if err != nil {
		return
	}
	ret.typeId = reader.ReadInt32()
	if ret.typeId == int32(unregisteredType) {
		ret.typeName, err = unmarshalString(reader)
		if err != nil {
			err = fmt.Errorf("failed to read className for typeId %d: %w", ret.typeId, err)
			return
		}
	}
	return
}

func writeClass(writer BinaryOutputStream, cls *igniteType) {
	writer.WriteInt32(cls.typeId)
	if cls.typeId == int32(unregisteredType) {
		marshalString(writer, cls.typeName)
	}
}

func readTime(reader BinaryInputStream) (Time, error) {
	if err := ensureAvailable(reader, longBytes); err != nil {
		return 0, err
	}
	return Time(reader.ReadInt64()), nil
}

func readDate(reader BinaryInputStream) (Date, error) {
	if err := ensureAvailable(reader, longBytes); err != nil {
		return 0, err
	}
	return Date(reader.ReadInt64()), nil
}

func readTimestamp(reader BinaryInputStream) (time.Time, error) {
	if err := ensureAvailable(reader, intBytes+longBytes); err != nil {
		return time.Time{}, err
	}
	millis := reader.ReadInt64()
	nanos := reader.ReadInt32() + int32((millis%1000)*int64(time.Millisecond))
	return time.Unix(millis/1000, int64(nanos)), nil
}

func readDecimal(reader BinaryInputStream) (*apd.Decimal, error) {
	if err := ensureAvailable(reader, intBytes); err != nil {
		return nil, err
	}
	exp := -reader.ReadInt32()
	coefSz := int(reader.ReadInt32())
	if coefSz == 0 {
		return nil, errors.New("invalid coef value: empty")
	}
	coefData := reader.ReadBytes(coefSz)
	negative := coefData[0]&0x80 == 0x80
	if negative {
		coefData[0] &= 0x7F
	}
	coef := new(apd.BigInt)
	coef.SetBytes(coefData)
	if negative {
		coef.Neg(coef)
	}
	return apd.NewWithBigInt(coef, exp), nil
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

func readIgniteTypedArray[T any](reader BinaryInputStream, elType TypeDesc, elemReader func(idx int, reader BinaryInputStream) (*T, error)) ([]*T, error) {
	return readSlice(reader, func(idx int, input BinaryInputStream) (*T, error) {
		isNotNull, err := checkNotNull(input, elType)
		if err != nil || !isNotNull {
			return nil, err
		}
		return elemReader(idx, input)
	})
}

func writeIgniteTypedArrayP[T any](writer BinaryOutputStream, elType TypeDesc, arr []*T, elemWriter func(writer BinaryOutputStream, el *T) error) error {
	return writeSequence(writer, len(arr), func(output BinaryOutputStream, idx int) error {
		el := arr[idx]
		if el == nil {
			writer.WriteNull()
		} else {
			writer.WriteType(elType)
			return elemWriter(writer, el)
		}
		return nil
	})
}

func writeIgniteTypedArray[T any](writer BinaryOutputStream, elType TypeDesc, arr []T, elemWriter func(writer BinaryOutputStream, el *T) error) error {
	return writeSequence(writer, len(arr), func(output BinaryOutputStream, idx int) error {
		el := arr[idx]
		writer.WriteType(elType)
		return elemWriter(writer, &el)
	})
}
