package ignite

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client/internal"
	"reflect"
	"sort"
	"strings"
)

const (
	protoVersion       int8   = 1
	flagsPos                  = 2
	typeIdPos                 = 4
	hashCodePos               = 8
	lenPos                    = 12
	schemaIdPos               = 16
	schemaOffsetPos           = 20
	headerLength              = 24
	flagUserType       uint16 = 0x0001
	flagHasSchema      uint16 = 0x0002
	flagHasRaw         uint16 = 0x0004
	flagOffsetOneByte  uint16 = 0x0008
	flagOffsetTwoBytes uint16 = 0x0010
	flagCompactFooter  uint16 = 0x0020
)

// BinaryObject is a wrapper around byte slice that contains serialized data in Apache Ignite binary format.
type BinaryObject interface {
	// Type returns binary object metadata
	Type(ctx context.Context) (BinaryType, error)
	// Field returns value of field with specified name.
	Field(ctx context.Context, name string) (interface{}, error)
	// ScanField set value of field to dest.
	ScanField(ctx context.Context, fldName string, dest interface{}) error
	// Size returns the underlying byte slice size.
	Size() int
	// Data returns the underlying byte slice
	Data() []byte
	// HashCode returns hash code according to ApacheIgnite binary format specification.
	HashCode() int32
	// IsEnum returns true if this is a binary enum object, false otherwise.
	IsEnum() bool
	// EnumOrdinal returns enum ordinal of this instance if this is a binary enum object, 0 otherwise.
	EnumOrdinal() int
	// EnumName returns enum name of this instance if this is a binary enum object, empty string otherwise.
	EnumName() string
	// internal method
	getTypeId() (int32, error)
}

type BinaryIdMapper interface {
	TypeId(typeName string) int32
	FieldId(typeId int32, fldName string) int32
}

type BinaryBasicIdMapper struct {
}

func (b BinaryBasicIdMapper) TypeId(typeName string) int32 {
	return internal.HashCode(strings.ToLower(typeName))
}

func (b BinaryBasicIdMapper) FieldId(_ int32, fldName string) int32 {
	return internal.HashCode(strings.ToLower(fldName))
}

type binaryObjectImpl struct {
	typeId int32
	data   []byte
	marsh  marshaller
}

func (b *binaryObjectImpl) Type(ctx context.Context) (BinaryType, error) {
	typeId, err := b.getTypeId()
	if err != nil {
		return nil, err
	}
	meta, err := b.marsh.getMetadata(ctx, typeId)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("no metadata found for typeId=%d", typeId)
	}
	return meta, nil
}

func (b *binaryObjectImpl) Field(ctx context.Context, fldName string) (interface{}, error) {
	flags := b.flags()
	if flags&flagHasSchema != flagHasSchema {
		return nil, nil
	}

	bIdMapper := b.marsh.binaryIdMapper()
	typeId, err := b.getTypeId()
	if err != nil {
		return nil, err
	}
	schemaId := b.schemaId()
	fieldId := bIdMapper.FieldId(typeId, fldName)

	isCompactFooter := flags&flagCompactFooter == flagCompactFooter
	footer, fldSize := b.footer()
	var fldOffsetIdx int
	if isCompactFooter {
		schema, err := b.marsh.getSchema(ctx, typeId, schemaId)
		if err != nil {
			return nil, err
		}
		if schema == nil {
			return nil, fmt.Errorf("schema not fount for typeId=%d and schemaId=%d", typeId, schemaId)
		}
		fieldIdx := -1
		for idx, id := range schema.fieldIds {
			if id == fieldId {
				fieldIdx = idx
			}
		}
		if fieldIdx == -1 {
			return nil, nil
		}
		fldOffsetIdx = fieldIdx * fldSize
	} else {
		fldOffsetIdx = -1
		for pos := 0; pos < len(footer); {
			id := int32(binary.LittleEndian.Uint32(footer[pos : pos+intBytes]))
			if id == fieldId {
				fldOffsetIdx = pos + 4
				break
			}
			pos += 4 + fldSize
		}
		if fldOffsetIdx == -1 {
			return nil, nil
		}
	}
	var fldOffset int
	if fldSize == 1 {
		fldOffset = int(footer[fldOffsetIdx])
	} else if fldSize == 2 {
		fldOffset = int(binary.LittleEndian.Uint16(footer[fldOffsetIdx : fldOffsetIdx+shortBytes]))
	} else {
		fldOffset = int(binary.LittleEndian.Uint32(footer[fldOffsetIdx : fldOffsetIdx+intBytes]))
	}
	return b.marsh.unmarshal(ctx, NewBinaryInputStream(b.data, fldOffset))
}

func (b *binaryObjectImpl) ScanField(ctx context.Context, fldName string, dest interface{}) error {
	src, err := b.Field(ctx, fldName)
	if err != nil {
		return err
	}
	return convertAssign(dest, src)
}

func (b *binaryObjectImpl) Size() int {
	return len(b.data)
}

func (b *binaryObjectImpl) Data() []byte {
	return b.data
}

func (b *binaryObjectImpl) IsEnum() bool {
	return false
}

func (b *binaryObjectImpl) EnumOrdinal() int {
	return 0
}

func (b *binaryObjectImpl) EnumName() string {
	return ""
}

func (b *binaryObjectImpl) getTypeId() (int32, error) {
	if len(b.data) < headerLength {
		return 0, fmt.Errorf("invalid binary object, data is corrupted")
	}
	typeId := b.typeId
	if typeId != 0 {
		return typeId, nil
	}
	typeId = b.getRawTypeId()
	if typeId == int32(unregisteredType) {
		clsName, err := unmarshalString(NewBinaryInputStream(b.data, headerLength))
		if err != nil {
			return 0, fmt.Errorf("invalid binary object, data is corrupted: %w", err)
		}
		typeId = b.marsh.binaryIdMapper().TypeId(clsName)
	}
	b.typeId = typeId
	return typeId, nil
}

func (b *binaryObjectImpl) getRawTypeId() int32 {
	if len(b.data) < headerLength {
		panic("impossible condition")
	}
	return int32(binary.LittleEndian.Uint32(b.data[typeIdPos : typeIdPos+intBytes]))
}

func (b *binaryObjectImpl) schemaId() int32 {
	if len(b.data) < headerLength {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b.data[schemaIdPos : schemaIdPos+intBytes]))
}

func (b *binaryObjectImpl) footer() ([]byte, int) {
	if len(b.data) < headerLength {
		return nil, 0
	}
	flags := b.flags()
	if flags&flagHasSchema != flagHasSchema {
		return nil, 0
	}
	var fldSize int
	if flags&flagOffsetOneByte == flagOffsetOneByte {
		fldSize = 1
	} else if flags&flagOffsetTwoBytes == flagOffsetTwoBytes {
		fldSize = 2
	} else {
		fldSize = 4
	}
	schemaStart := int(binary.LittleEndian.Uint32(b.data[schemaOffsetPos : schemaOffsetPos+intBytes]))
	schemaLen := len(b.data) - schemaStart
	if flags&flagHasRaw == flagHasRaw {
		schemaLen -= 4
	}
	return b.data[schemaStart : schemaStart+schemaLen], fldSize
}

func (b *binaryObjectImpl) flags() uint16 {
	if len(b.data) < headerLength {
		return 0
	}
	return binary.LittleEndian.Uint16(b.data[flagsPos : flagsPos+shortBytes])
}

func (b *binaryObjectImpl) HashCode() int32 {
	if len(b.data) < headerLength {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b.data[hashCodePos : hashCodePos+intBytes]))
}

type boField struct {
	typeId int32
	value  interface{}
}

type binaryObjectOptions struct {
	isEnum               bool
	ord                  int32
	typeName             string
	affKeyName           string
	fields               map[string]*boField
	fieldsOrder          []string
	platform             MarshallerPlatform
	skipTypeRegistration bool // default is false, true is only for testing.
}

func newBinaryObject(ctx context.Context, marsh marshaller, opts *binaryObjectOptions) (ret BinaryObject, err error) {
	bIdMapper := marsh.binaryIdMapper()
	typeId := bIdMapper.TypeId(opts.typeName)

	oldMeta, err := marsh.getMetadata(ctx, typeId)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	if oldMeta != nil {
		if opts.typeName != oldMeta.typeName {
			return nil, fmt.Errorf("types have the same typeId %d: old %s vs new %s", typeId, oldMeta.typeName,
				opts.typeName)
		}
		if opts.affKeyName != oldMeta.affKeyName {
			return nil, fmt.Errorf("type %s with typeId %d has different affinity key field: old %s vs new %s",
				opts.typeName, typeId, oldMeta.affKeyName, opts.affKeyName)
		}
		if oldMeta.isEnum {
			return nil, fmt.Errorf("type %s with typeId %d has been already registered as enum", opts.typeName, typeId)
		}
	}
	// For testing of class name registration
	if opts.skipTypeRegistration {
		typeId = int32(unregisteredType)
	}

	newMeta := newBinaryMetadata(typeId, opts.typeName, opts.affKeyName)
	outStream := NewBinaryOutputStream(headerLength)
	bWriter := newBinaryWriterImpl(marsh, outStream, newMeta, oldMeta)

	for _, fldName := range opts.fieldsOrder {
		field := opts.fields[fldName]
		if err = bWriter.writeField0(ctx, fldName, field); err != nil {
			return
		}
	}
	if err = bWriter.finalize(ctx); err != nil {
		return
	}
	if typeId != int32(unregisteredType) {
		if err = marsh.registerClassName(ctx, opts.platform, typeId, opts.typeName); err != nil {
			return
		}
	}
	ret = &binaryObjectImpl{
		data:  outStream.Slice(bWriter.startPos, outStream.Position()),
		marsh: bWriter.marsh,
	}
	return
}

func marshalBinarylizable(ctx context.Context, marsh marshaller, outStream BinaryOutputStream, value Binarylizable) error {
	bIdMapper := marsh.binaryIdMapper()
	typeName, err := typeNameByReflection(value)
	if err != nil {
		return fmt.Errorf("invalid binarylizable %v: %w", value, err)
	}
	affKeyName := ""
	if affKewAwareVal, ok := value.(AffinityKeyAware); ok {
		affKeyName = affKewAwareVal.AffinityKeyName()
	}
	typeId := bIdMapper.TypeId(typeName)

	oldMeta, err := marsh.getMetadata(ctx, typeId)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}
	if oldMeta != nil {
		if typeName != oldMeta.typeName {
			return fmt.Errorf("types have the same typeId %d: old %s vs new %s", typeId, oldMeta.typeName,
				typeName)
		}
		if affKeyName != oldMeta.affKeyName {
			return fmt.Errorf("type %s with typeId %d has different affinity key field: old %s vs new %s",
				typeName, typeId, oldMeta.affKeyName, affKeyName)
		}
		if oldMeta.isEnum {
			return fmt.Errorf("type %s with typeId %d has been already registered as enum", typeName, typeId)
		}
	}

	newMeta := newBinaryMetadata(typeId, typeName, affKeyName)
	bWriter := newBinaryWriterImpl(marsh, outStream, newMeta, oldMeta)

	if err = value.Write(ctx, bWriter); err != nil {
		return err
	}
	if err = bWriter.finalize(ctx); err != nil {
		return err
	}
	marshPlatform := JavaMarshaller
	if marshPlatformAwareVal, ok := value.(MarshallerPlatformAware); ok {
		marshPlatform = marshPlatformAwareVal.Platform()
	}
	if err = marsh.registerClassName(ctx, marshPlatform, typeId, typeName); err != nil {
		return err
	}
	return nil
}

type binaryWriterImpl struct {
	newMeta   *binaryMetadata
	oldMeta   *binaryMetadata
	marsh     marshaller
	schemaBld *binarySchemaBuilder
	outStream BinaryOutputStream
	startPos  int
	offset    int
	flags     uint16
}

func newBinaryWriterImpl(marsh marshaller, outStream BinaryOutputStream, newMeta *binaryMetadata, oldMeta *binaryMetadata) *binaryWriterImpl {
	schemaBuilder := newBinarySchemaBuilder()

	outStream.EnsureAvailable(headerLength)
	startPos := outStream.Position()
	outStream.SetPosition(startPos + headerLength)

	flags := flagUserType
	if marsh.isCompactFooter() {
		flags |= flagCompactFooter
	}

	var offset int
	if newMeta.typeId == int32(unregisteredType) {
		marshalString(outStream, newMeta.TypeName())
		offset = outStream.Position() - startPos
	} else {
		offset = headerLength
	}

	return &binaryWriterImpl{
		newMeta:   newMeta,
		oldMeta:   oldMeta,
		marsh:     marsh,
		schemaBld: schemaBuilder,
		outStream: outStream,
		startPos:  startPos,
		offset:    offset,
		flags:     flags,
	}
}

func (bWriter *binaryWriterImpl) WriteField(ctx context.Context, fldName string, value interface{}) error {
	return bWriter.writeField0(ctx, fldName, &boField{
		typeId: -1,
		value:  value,
	})
}

func (bWriter *binaryWriterImpl) WriteNullField(ctx context.Context, fldName string, typeId TypeDesc) error {
	return bWriter.writeField0(ctx, fldName, &boField{
		typeId: int32(typeId),
		value:  nil,
	})
}

func (bWriter *binaryWriterImpl) writeField0(ctx context.Context, fldName string, field *boField) error {
	var err error
	if bWriter.flags&flagHasSchema != flagHasSchema {
		bWriter.flags |= flagHasSchema
	}
	newMeta := bWriter.newMeta
	bIdMapper := bWriter.marsh.binaryIdMapper()
	typeId := bWriter.newMeta.TypeId()
	fieldId := bIdMapper.FieldId(typeId, fldName)
	// set old field metadata if exists
	var oldFieldMeta *binaryFieldMeta = nil
	oldMeta := bWriter.oldMeta
	if oldMeta != nil {
		if fMeta, ok := oldMeta.fields[fldName]; ok {
			oldFieldMeta = &fMeta
		}
	}
	// set field type if not have been set already
	if field.typeId < 0 {
		if field.value != nil {
			fldTypeId, err := getTypeId(field.value)
			if err != nil {
				return fmt.Errorf("failed to get type id for field %s: %w", fldName, err)
			}
			field.typeId = int32(fldTypeId)
		} else if oldFieldMeta != nil {
			field.typeId = oldFieldMeta.typeId
		} else {
			field.typeId = int32(BinaryObjectType)
		}
	}
	// check type compliance for oldMeta
	if oldFieldMeta != nil && field.typeId != oldFieldMeta.typeId {
		return fmt.Errorf("type %s with typeId %d has different type for field %s: old %d vs %d",
			newMeta.typeName, typeId, fldName, oldFieldMeta.typeId, typeId)
	}
	// add new field to meta if oldFieldMeta is nil
	if oldFieldMeta == nil {
		newMeta.addField(fldName, field.typeId, fieldId)
	}
	outStream := bWriter.outStream
	bWriter.schemaBld.AddField(fieldId, int32(outStream.Position()-bWriter.startPos))
	if err = bWriter.marsh.marshal(ctx, outStream, field.value); err != nil {
		return fmt.Errorf("failed to marshall field %s: %w", fldName, err)
	}
	return nil
}

func (bWriter *binaryWriterImpl) finalize(ctx context.Context) error {
	outStream := bWriter.outStream
	schemaBld := bWriter.schemaBld
	marsh := bWriter.marsh
	if bWriter.flags&flagHasSchema == flagHasSchema {
		bWriter.offset = outStream.Position() - bWriter.startPos
		fldSize := schemaBld.WriteFooter(outStream, marsh.isCompactFooter())
		if fldSize == 1 {
			bWriter.flags |= flagOffsetOneByte
		} else if fldSize == 2 {
			bWriter.flags |= flagOffsetTwoBytes
		}
	}
	schema := schemaBld.Build()
	retPos := outStream.Position()
	outStream.SetPosition(bWriter.startPos)
	outStream.WriteType(BinaryObjectType)
	outStream.WriteInt8(protoVersion)
	outStream.WriteUInt16(bWriter.flags)
	outStream.WriteInt32(bWriter.newMeta.TypeId())
	outStream.SetPosition(bWriter.startPos + lenPos)
	outStream.WriteInt32(int32(retPos - bWriter.startPos))
	outStream.WriteInt32(schema.schemaId)
	outStream.WriteInt32(int32(bWriter.offset))
	// Add schema to registry
	bWriter.newMeta.addSchema(schema)

	meta := bWriter.newMeta
	if meta.typeId == int32(unregisteredType) {
		meta = meta.copy(func(cpy *binaryMetadata) {
			cpy.typeId = marsh.binaryIdMapper().TypeId(meta.TypeName())
		})
	}
	if err := marsh.putMetadata(ctx, meta); err != nil {
		return err
	}
	// Calculate and write hashcode
	hashCode := outStream.HashCode(bWriter.startPos+headerLength, bWriter.startPos+bWriter.offset)
	outStream.SetPosition(bWriter.startPos + hashCodePos)
	outStream.WriteInt32(hashCode)
	// Restore final stream position
	outStream.SetPosition(retPos)
	return nil
}

type binaryReaderImpl struct {
	bo     BinaryObject
	fields []string
}

func newBinaryReaderImpl(bo BinaryObject, meta *binaryMetadata) (BinaryReader, error) {
	fields := meta.Fields()
	sort.Strings(fields)
	return &binaryReaderImpl{bo: bo, fields: fields}, nil
}

func (bReader *binaryReaderImpl) ReadField(ctx context.Context, fldName string, dest any) error {
	fields := bReader.fields
	_, ok := sort.Find(len(fields), func(i int) int {
		return strings.Compare(fldName, fields[i])
	})
	if ok {
		return bReader.bo.ScanField(ctx, fldName, dest)
	}
	return fmt.Errorf("field %s not found", fldName)
}

type BinaryReader interface {
	ReadField(ctx context.Context, fldName string, dest interface{}) error
}

type BinaryWriter interface {
	WriteField(ctx context.Context, fldName string, data interface{}) error
	WriteNullField(ctx context.Context, fldName string, typeId TypeDesc) error
}

type MarshallerPlatformAware interface {
	Platform() MarshallerPlatform
}

type AffinityKeyAware interface {
	AffinityKeyName() string
}

type TypeNameAware interface {
	TypeName() string
}

type Binarylizable interface {
	Write(ctx context.Context, writer BinaryWriter) error
	Read(ctx context.Context, reader BinaryReader) error
}

type Enum interface {
	Ordinal() int
	Name() string
	Values() []Enum
}

func typeNameByReflection(obj interface{}) (typeName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch err0 := r.(type) {
			case string:
				err = fmt.Errorf("failed to get type by reflection: %s", err0)
			case error:
				err = fmt.Errorf("failed to get type by reflection: %s", err0)
			default:
				err = errors.New("failed to get type by reflection")
			}
			typeName = ""
		}
	}()
	if typeNameAware, ok := obj.(TypeNameAware); ok {
		typeName = typeNameAware.TypeName()
	} else {
		value := reflect.ValueOf(obj)
		if value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface {
			typeName = value.Elem().Type().String()
		} else {
			typeName = value.Type().String()
		}
	}
	return
}
