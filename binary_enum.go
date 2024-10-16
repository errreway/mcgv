package ignite

import (
	"context"
	"fmt"
)

type BinaryEnumArray struct {
	elType    BinaryType
	data      []BinaryObject
	isNotNull bool
}

func NewEnumArray(elType BinaryType, elements ...BinaryObject) (ret BinaryEnumArray, err error) {
	if elType == nil {
		err = fmt.Errorf("NewEnumArray called with nil elType")
		return
	}
	if !elType.IsEnum() {
		err = fmt.Errorf("type [id=%d, name=%s] is not enum, enum type is required", elType.TypeId(), elType.TypeName())
		return
	}
	ret = BinaryEnumArray{
		elType: elType,
		data:   make([]BinaryObject, len(elements)),
	}
	for i, el := range elements {
		if el == nil {
			ret.data[i] = nil
			continue
		}
		if !el.IsEnum() {
			err = fmt.Errorf("non enum element is inserting, %v", el)
			return
		}
		el0 := el.(*binaryEnum)
		if el0.typeId != elType.TypeId() || el0.typeName != elType.TypeName() {
			err = fmt.Errorf("inserting elements with different typeId or typeName: element typeId %d vs %d, typeName %s vs %s",
				el0.typeId, elType.TypeId(), el0.typeName, elType.TypeName())
			return
		}
		ret.data[i] = el
	}
	ret.isNotNull = true
	return
}

func (arr *BinaryEnumArray) IsNull() bool {
	return !arr.isNotNull
}

func (arr *BinaryEnumArray) Size() int {
	return len(arr.data)
}

func (arr *BinaryEnumArray) Type() BinaryType {
	return arr.elType
}

type binaryEnum struct {
	ord      int32
	name     string
	typeId   int32
	typeName string
	marsh    marshaller
	data     []byte
}

func (b *binaryEnum) Type(ctx context.Context) (BinaryType, error) {
	meta, err := b.marsh.getMetadata(ctx, b.typeId)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("no metadata found for typeId=%d", b.typeId)
	}
	return meta, nil
}

func (b *binaryEnum) Field(context.Context, string) (interface{}, error) {
	return nil, fmt.Errorf("not supported for binary enums")
}

func (b *binaryEnum) ScanField(context.Context, string, interface{}) error {
	return fmt.Errorf("not supported for binary enums")
}

func (b *binaryEnum) Size() int {
	return len(b.data)
}

func (b *binaryEnum) Data() []byte {
	return b.data
}

func (b *binaryEnum) HashCode() int32 {
	return 31*b.typeId + b.ord
}

func (b *binaryEnum) IsEnum() bool {
	return true
}

func (b *binaryEnum) EnumOrdinal() int {
	return int(b.ord)
}

func (b *binaryEnum) EnumName() string {
	return b.name
}

func (b *binaryEnum) getTypeId() (int32, error) {
	return b.typeId, nil
}

func newBinaryEnum(ctx context.Context, marsh marshaller, opts *binaryObjectOptions) (ret BinaryObject, err error) {
	if !opts.isEnum {
		err = fmt.Errorf("invalid options for creating enums: %v", opts)
		return
	}
	typeName := opts.typeName
	ord := opts.ord

	bIdMapper := marsh.binaryIdMapper()
	typeId := bIdMapper.TypeId(typeName)

	meta, err := marsh.getMetadata(ctx, typeId)
	if err != nil {
		err = fmt.Errorf("failed to get metadata: %w", err)
		return
	}
	if meta == nil {
		err = fmt.Errorf("no previous binary metadata was registered for %s", typeName)
		return
	} else if !meta.isEnum {
		err = fmt.Errorf("type %s with typeId %d has been already registered as non enum", typeName, typeId)
		return
	}

	if _, ok := meta.ordToName[ord]; !ok {
		err = fmt.Errorf("invalid ordinal %d for enum typeId=%d", ord, typeId)
		return
	}

	if opts.skipTypeRegistration {
		typeId = int32(unregisteredType)
	} else {
		if err = marsh.registerClassName(ctx, opts.platform, typeId, opts.typeName); err != nil {
			return
		}
	}

	outStream := NewBinaryOutputStream(5)
	outStream.WriteType(EnumType)
	outStream.WriteInt32(typeId)
	if typeId == int32(unregisteredType) {
		writeString(outStream, meta.typeName)
	}
	outStream.WriteInt32(ord)

	ret = &binaryEnum{
		ord:      ord,
		name:     meta.ordToName[ord],
		typeId:   typeId,
		marsh:    marsh,
		typeName: meta.typeName,
		data:     outStream.Slice(0, outStream.Position()),
	}

	return
}

func registerEnumMeta(ctx context.Context, marsh marshaller, typeName string, nameToOrd map[string]int) error {
	typeId := marsh.binaryIdMapper().TypeId(typeName)

	oldMeta, err := marsh.getMetadata(ctx, typeId)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}
	if oldMeta != nil {
		if typeName != oldMeta.typeName {
			return fmt.Errorf("types have the same typeId %d: old %s vs new %s", typeId, oldMeta.typeName, typeName)
		}
		if !oldMeta.isEnum {
			return fmt.Errorf("type %s with typeId %d has been already registered as non enum", typeName, typeId)
		}
	}

	nameToOrd0 := make(map[string]int32)
	for name, ord := range nameToOrd {
		nameToOrd0[name] = int32(ord)
	}
	newMeta := newEnumMetadata(typeId, typeName, nameToOrd0)

	return marsh.putMetadata(ctx, newMeta)
}

func marshalEnum(ctx context.Context, marsh marshaller, outStream BinaryOutputStream, value Enum) (err error) {
	bIdMapper := marsh.binaryIdMapper()

	typeName, err := typeNameByReflection(value)
	if err != nil {
		return fmt.Errorf("invalid enum value %v: %w", value, err)
	}
	typeId := bIdMapper.TypeId(typeName)

	meta, err := marsh.getMetadata(ctx, typeId)
	if err != nil {
		err = fmt.Errorf("failed to get metadata: %w", err)
		return
	}
	if meta == nil {
		nameToOrd := make(map[string]int32)
		for _, enum := range value.Values() {
			nameToOrd[enum.Name()] = int32(enum.Ordinal())
		}
		meta = newEnumMetadata(typeId, typeName, nameToOrd)

		if err = marsh.putMetadata(ctx, meta); err != nil {
			return
		}
	}

	ord := int32(value.Ordinal())
	if _, ok := meta.ordToName[ord]; !ok {
		err = fmt.Errorf("invalid ordinal %d for enum typeId=%d", ord, typeId)
		return
	}

	outStream.WriteType(EnumType)
	outStream.WriteInt32(typeId)
	if typeId == int32(unregisteredType) {
		writeString(outStream, typeName)
	}
	outStream.WriteInt32(ord)

	if typeId != int32(unregisteredType) {
		marshPlatform := JavaMarshaller
		if marshPlatformAwareVal, ok := value.(MarshallerPlatformAware); ok {
			marshPlatform = marshPlatformAwareVal.Platform()
		}
		if err = marsh.registerClassName(ctx, marshPlatform, typeId, typeName); err != nil {
			return
		}
	}
	return
}
