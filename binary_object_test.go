package ignite

import (
	"context"
	"fmt"
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"math"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type BinaryObjectTestSuite struct {
	testing2.IgniteTestSuite
	cli *Client
}

func TestBinaryObjectsCacheOperations(t *testing.T) {
	suite.Run(t, new(BinaryObjectTestSuite))
}

func (suite *BinaryObjectTestSuite) TestBasic() {
	suite.runTests([]struct {
		f         func(*testing.T, *Client, *Cache)
		isIndexed bool
	}{
		{testPutGetAllBinaryObject, true},
		{testNestedBinaryObject, false},
		{testLargeBinaryObject, false},
		{testDifferentFieldTypes, false},
		{testUnregisteredTypes, false},
		{testBinaryEnumsAsFields, false},
		{testBinaryEnumsAsValues, false},
	}...)
}

func (suite *BinaryObjectTestSuite) TestMergeMetadata() {
	suite.runTests(struct {
		f         func(*testing.T, *Client, *Cache)
		isIndexed bool
	}{testMergeMetadata_WithClearRegistry, false})
	suite.runTests(struct {
		f         func(*testing.T, *Client, *Cache)
		isIndexed bool
	}{testMergeMetadata_WithoutClearRegistry, false})
}

func testPutGetAllBinaryObject(t *testing.T, cli *Client, cache *Cache) {
	keyValues := make([]KeyValue, 0)
	keyValuesMap := make(map[int]KeyValue)
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		key, err := cli.CreateBinaryObject(ctx, "PERSON_KEY",
			WithField("id", i),
			WithField("bucket_id", i%2),
			WithAffinityKeyName("bucket_id"),
		)
		require.NoError(t, err)
		value, err := cli.CreateBinaryObject(ctx, "PERSON",
			WithField("name", fmt.Sprintf("name_%d", i)),
			WithField("age", int16(10)),
		)
		require.NoError(t, err)
		keyValues = append(keyValues, KeyValue{key, value})
		keyValuesMap[i] = KeyValue{key, value}
	}
	err := cache.PutAll(ctx, keyValues...)
	require.NoError(t, err)
	cfg, err := cache.Configuration(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	sz, err := cache.Size(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1000), sz)

	keys := make([]interface{}, 0)
	for i := 0; i < 1000; i++ {
		keys = append(keys, keyValues[i].Key)
	}

	cli.marsh.clearRegistry() // Test retrieving binary info from server

	keyValues0, err := cache.GetAll(ctx, keys...)
	require.NoError(t, err)
	require.Equal(t, 1000, len(keyValues0))

	for _, kv0 := range keyValues0 {
		key0 := kv0.Key.(BinaryObject)
		value0 := kv0.Value.(BinaryObject)
		var id0 int64
		err = key0.ScanField(ctx, "id", &id0)
		require.NoError(t, err)

		var pid0 *int64
		err = key0.ScanField(ctx, "id0", &pid0)
		require.NoError(t, err)
		require.Nil(t, pid0)

		kv, ok := keyValuesMap[int(id0)]
		require.True(t, ok)
		key := kv.Key.(BinaryObject)
		value := kv.Value.(BinaryObject)

		RequireBinaryObjectsEqual(t, key0, key)
		RequireBinaryObjectsEqual(t, value0, value)
	}
}

func testLargeBinaryObject(t *testing.T, cli *Client, cache *Cache) {
	for _, sz := range []int{0, 1024, 10 * 1024, 100 * 1024} {
		t.Run(fmt.Sprintf("%s-%d", t.Name(), sz), func(t *testing.T) {
			var val BinaryObject
			var err error
			ctx := context.Background()
			if sz != 0 {
				val, err = cli.CreateBinaryObject(ctx, "BLOB",
					WithField("data0", testing2.MakeByteArrayPayload(sz)),
					WithField("data1", testing2.MakeByteArrayPayload(sz)),
				)
			} else {
				val, err = cli.CreateBinaryObject(ctx, "BLOB", WithNullField("data0", ByteArrayType))
			}
			require.NoError(t, err)
			err = cache.Put(ctx, "blob", val)
			require.NoError(t, err)
			val0, err := cache.Get(ctx, "blob")
			require.NoError(t, err)
			RequireBinaryObjectsEqual(t, val, val0.(BinaryObject))
		})
	}
}

func testNestedBinaryObject(t *testing.T, cli *Client, cache *Cache) {
	ctx := context.Background()
	inner, err := cli.CreateBinaryObject(ctx, "INNER", WithField("id", 10))
	require.NoError(t, err)
	outer, err := cli.CreateBinaryObject(ctx, "OUTER", WithField("id", 10), WithField("obj", inner))
	require.NoError(t, err)
	err = cache.Put(ctx, "outer", outer)
	require.NoError(t, err)
	cli.marsh.clearRegistry()

	val, err := cache.Get(ctx, "outer")
	require.NoError(t, err)

	outer1 := val.(BinaryObject)
	outer1Type, err := outer1.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, "OUTER", outer1Type.TypeName())
	require.Equal(t, []string{"id", "obj"}, outer1Type.Fields())

	val, err = outer1.Field(ctx, "obj")
	require.NoError(t, err)

	inner1 := val.(BinaryObject)
	inner1Type, err := inner1.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, "INNER", inner1Type.TypeName())
	require.Equal(t, []string{"id"}, inner1Type.Fields())

	innerFldVal, err := inner1.Field(ctx, "id")
	require.NoError(t, err)
	require.Equal(t, int64(10), innerFldVal)
}

func testDifferentFieldTypes(t *testing.T, cli *Client, cache *Cache) {
	timestamp := time.Now().Truncate(0) // remove monotonic part.
	fields := []struct {
		name  string
		value interface{}
	}{
		{"bool", true}, {"byte", byte(10)}, {"int8", int8(10)},
		{"char", uint16(10)}, {"short", int16(10)}, {"int", int32(10)},
		{"uint", uint32(10)}, {"long", int64(10)}, {"ulong", uint64(10)},
		{"goInt", 10}, {"goUInt", uint(10)},
		{"float", float32(10.0)}, {"double", float64(10.0)},
		{"boolArray", createRandPrimitiveArray(func() bool { return rand.Intn(3) != 0 })},
		{"byteArray", createRandPrimitiveArray(func() byte { return byte(rand.Intn(1<<8 - 1)) })},
		{"signedByteArray", createRandPrimitiveArray(func() int8 { return int8(rand.Intn(1<<8 - 1)) })},
		{"shortArray", createRandPrimitiveArray(func() int16 { return int16(rand.Intn(1<<16 - 1)) })},
		{"charArray", createRandPrimitiveArray(func() uint16 { return uint16(rand.Intn(1<<16 - 1)) })},
		{"uintArray", createRandPrimitiveArray(func() uint { return uint(rand.Uint64()) })},
		{"intArray", createRandPrimitiveArray(func() int { return int(rand.Int63()) })},
		{"uint32Array", createRandPrimitiveArray(func() uint32 { return rand.Uint32() })},
		{"int32Array", createRandPrimitiveArray(func() int32 { return rand.Int31() })},
		{"uint64Array", createRandPrimitiveArray(func() uint64 { return rand.Uint64() })},
		{"int64Array", createRandPrimitiveArray(func() int64 { return rand.Int63() })},
		{"floatArray", createRandPrimitiveArray(func() float32 { return rand.Float32() })},
		{"doubleArray", createRandPrimitiveArray(func() float64 { return rand.Float64() })},
		{"string", "test"}, {"uuid", uuid.New()}, {"time", NewTime(timestamp)},
		{"date", NewDate(timestamp)}, {"timestamp", timestamp},
		{"decimal", apd.New(100500, -3)},
		{"stringPArray", testing2.CreatePSlice("test1", "test2", "test3")},
		{"stringArray", []string{"test1", "test2", "test3"}},
		{"uuidPArray", testing2.CreatePSlice(uuid.New(), uuid.New(), uuid.New())},
		{"uuidArray", []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}},
		{"timePArray", testing2.CreatePSlice(NewTime(timestamp), NewTime(timestamp.Add(time.Minute*10)), NewTime(timestamp.Add(time.Minute*20)))},
		{"timeArray", []Time{NewTime(timestamp), NewTime(timestamp.Add(time.Minute * 10)), NewTime(timestamp.Add(time.Minute * 20))}},
		{"datePArray", testing2.CreatePSlice(NewDate(timestamp), NewDate(timestamp.Add(time.Minute*10)), NewDate(timestamp.Add(time.Minute*20)))},
		{"dateArray", []Date{NewDate(timestamp), NewDate(timestamp.Add(time.Minute * 10)), NewDate(timestamp.Add(time.Minute * 20))}},
		{"timeStampPArray", testing2.CreatePSlice(timestamp, timestamp.Add(time.Minute*10), timestamp.Add(time.Minute*20))},
		{"timeStampArray", []time.Time{timestamp, timestamp.Add(time.Minute * 10), timestamp.Add(time.Minute * 20)}},
		{"decimalPArray", []*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)}},
		{"decimalArray", testing2.FromPSlice([]*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)})},
		{"binaryObject", createTestBinaryObject(t, cli, 100500)},
		{"singletonList", NewSingletonList("test")},
		{"objectArray", []BinaryObject{createTestBinaryObject(t, cli, 10), createTestBinaryObject(t, cli, 20), nil}},
		{"objectArrayMixed", []interface{}{createTestBinaryObject(t, cli, 10), "test", nil}},
		{"arrayList", NewArrayList("test", "test")},
		{"arrayListMixed", NewArrayList([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...)},
		{"linkedList", NewLinkedList("test", "test")},
		{"linkedListMixed", NewLinkedList([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...)},
		{"hashSet", NewHashSet("test1", "test2")},
		{"hashSetMixed", NewHashSet([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...)},
		{"linkedHashSet", NewHashSet("test1", "test2")},
		{"linkedHashSetMixed", NewHashSet([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...)},
		{"hashMap", ToHashMap(map[string]int32{"test1": 1, "test2": 2})},
		{"hashMapMixed", NewHashMap([]KeyValue{{"test1", createTestBinaryObject(t, cli, 10)}, {int32(10), "test"}}...)},
		{"linkedHashMap", ToLinkedHashMap(map[string]int32{"test1": 1, "test2": 2})},
		{"linkedHashMapMixed", NewLinkedHashMap([]KeyValue{{"test1", createTestBinaryObject(t, cli, 10)}, {int32(10), "test"}}...)},
		{"hashMapGo", map[string]BinaryObject{"test1": createTestBinaryObject(t, cli, 10), "test2": createTestBinaryObject(t, cli, 20)}},
		{"enum", createTestEnum(t, cli)},
		{"enumArray", createTestEnumArray(t, cli)},
	}
	ctx := context.Background()
	opts := make([]BinaryObjectOption, 0)
	for _, field := range fields {
		opts = append(opts, WithField(field.name, field.value))
	}
	obj, err := cli.CreateBinaryObject(ctx, "MANYFIELDS", opts...)
	require.NoError(t, err)
	for _, field := range fields {
		val, err := obj.Field(ctx, field.name)
		require.NoError(t, err, field.name)
		t.Logf("testing field %s: expected=%v, actual=%v", field.name, field.value, val)
		RequireIgniteTypesEqual(t, field.value, val)
	}

	err = cache.Put(ctx, "key", obj)
	require.NoError(t, err)

	cli.marsh.clearRegistry()

	obj1, err := cache.Get(ctx, "key")
	require.NoError(t, err)
	RequireBinaryObjectsEqual(t, obj, obj1.(BinaryObject))
}

func testMergeMetadata_WithClearRegistry(t *testing.T, cli *Client, cache *Cache) {
	testMergeMetadata(t, cli, cache, true)
}

func testMergeMetadata_WithoutClearRegistry(t *testing.T, cli *Client, cache *Cache) {
	testMergeMetadata(t, cli, cache, false)
}

func testUnregisteredTypes(t *testing.T, cli *Client, cache *Cache) {
	ctx := context.Background()
	typeName := "UNREGISTERED"
	expTypeId := cli.marsh.binaryIdMapper().TypeId(typeName)
	exp, err := cli.CreateBinaryObject(ctx, typeName, WithField("id", uuid.New()), func(options *binaryObjectOptions) {
		options.skipTypeRegistration = true
	})
	require.Equal(t, int32(unregisteredType), exp.(*binaryObjectImpl).getRawTypeId())
	require.NoError(t, err)
	typ, err := exp.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, typeName, typ.TypeName())
	require.Equal(t, expTypeId, typ.TypeId())

	err = cache.Put(ctx, "key", exp)
	require.NoError(t, err)
	val, err := cache.Get(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, int32(unregisteredType), val.(*binaryObjectImpl).getRawTypeId())
	RequireBinaryObjectsEqual(t, exp, val.(BinaryObject))
}

func testMergeMetadata(t *testing.T, cli *Client, _ *Cache, clearRegistry bool) {
	ctx := context.Background()
	objChecker := func(obj BinaryObject, typeName string, fields []string, fieldVals ...struct {
		name  string
		value interface{}
	}) {
		objT, err := obj.Type(ctx)
		require.NoError(t, err)
		require.Equal(t, typeName, objT.TypeName())
		require.Equal(t, fields, objT.Fields())
		for _, fv := range fieldVals {
			val, err := obj.Field(ctx, fv.name)
			require.NoError(t, err)
			require.Equal(t, fv.value, val)
		}
	}
	objSchemaSizeChecker := func(obj BinaryObject, expectedSz int) {
		objT, err := obj.Type(ctx)
		require.NoError(t, err)
		meta, err := cli.marsh.getMetadata(ctx, objT.TypeId())
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Equal(t, expectedSz, len(meta.schemas))
	}

	runChecks := func(schemaCnt int, fields []string, checks []func(int, []string)) {
		for _, check := range checks {
			check(schemaCnt, fields)
		}
	}
	checks := make([]func(int, []string), 0)

	// Empty object, schema with no fields
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	objEmpty, err := cli.CreateBinaryObject(ctx, "MERGED")
	require.NoError(t, err)
	checks = append(checks, func(schemaCnt int, fields []string) {
		objSchemaSizeChecker(objEmpty, schemaCnt)
		objChecker(objEmpty, "MERGED", fields, []struct {
			name  string
			value interface{}
		}{
			{name: "id", value: nil},
			{"name", nil},
		}...)
	})
	runChecks(1, []string{}, checks)

	// New schema should be added since new field was added
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	objOneField, err := cli.CreateBinaryObject(ctx, "MERGED", WithField("id", 10))
	require.NoError(t, err)
	checks = append(checks, func(schemaCnt int, fields []string) {
		objSchemaSizeChecker(objOneField, schemaCnt)
		objChecker(objOneField, "MERGED", fields, []struct {
			name  string
			value interface{}
		}{
			{name: "id", value: int64(10)},
			{"name", nil},
		}...)
	})
	runChecks(2, []string{"id"}, checks)

	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	_, err = cli.CreateBinaryObject(ctx, "MERGED", WithField("id", 10), WithAffinityKeyName("id"))
	require.Error(t, err) // should fail if affinity key name was changed
	runChecks(2, []string{"id"}, checks)

	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	_, err = cli.CreateBinaryObject(ctx, "MERGED", WithField("id", "test"))
	require.Error(t, err) // should fail if field type was changed
	runChecks(2, []string{"id"}, checks)

	// New schema should be added since new field was added
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	objTwoField, err := cli.CreateBinaryObject(ctx, "MERGED", WithField("id", 10), WithField("name", "name"))
	require.NoError(t, err)
	checks = append(checks, func(schemaCnt int, fields []string) {
		objSchemaSizeChecker(objTwoField, schemaCnt)
		objChecker(objTwoField, "MERGED", fields, []struct {
			name  string
			value interface{}
		}{
			{name: "id", value: int64(10)},
			{"name", "name"},
		}...)
	})
	runChecks(3, []string{"id", "name"}, checks)

	// No new schema should be added.
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	objOneFieldAfterTwo, err := cli.CreateBinaryObject(ctx, "MERGED", WithField("id", 10))
	require.NoError(t, err)
	checks = append(checks, func(schemaCnt int, fields []string) {
		objSchemaSizeChecker(objOneFieldAfterTwo, schemaCnt)
		objChecker(objOneFieldAfterTwo, "MERGED", fields, []struct {
			name  string
			value interface{}
		}{
			{name: "id", value: int64(10)},
			{"name", nil},
		}...)
	})
	runChecks(3, []string{"id", "name"}, checks)

	// New schema should be added if field order is different.
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	objTwoFieldDiffOrder, err := cli.CreateBinaryObject(ctx, "MERGED", WithField("name", "name"), WithField("id", 10))
	require.NoError(t, err)
	checks = append(checks, func(schemaCnt int, fields []string) {
		objSchemaSizeChecker(objTwoFieldDiffOrder, schemaCnt)
		objChecker(objTwoField, "MERGED", fields, []struct {
			name  string
			value interface{}
		}{
			{name: "id", value: int64(10)},
			{"name", "name"},
		}...)
	})
	runChecks(4, []string{"id", "name"}, checks)

	// Check changing to enum fail
	if clearRegistry {
		cli.marsh.clearRegistry()
	}

	err = cli.RegisterEnumMetadata(ctx, "MERGED", map[string]int{"VAL1": 0})
	require.ErrorContains(t, err, "has been already registered as non enum")

	// Check first time creation of enum without supplying enum names
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	_, err = cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(0))
	require.ErrorContains(t, err, "no previous binary metadata was registered for ENUM_MERGED")

	// Check creating already registered enum without supplying enum ordinals
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	err = cli.RegisterEnumMetadata(ctx, "ENUM_MERGED", map[string]int{"VAL1": 0})
	require.NoError(t, err)
	enum0, err := cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(0))
	require.NoError(t, err)
	require.Equal(t, enum0.EnumOrdinal(), 0)
	require.Equal(t, enum0.EnumName(), "VAL1")
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	enum1, err := cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(0))
	require.NoError(t, err)
	RequireBinaryObjectsEqual(t, enum0, enum1)

	// Check invalid default ordinal
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	_, err = cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(math.MinInt32))
	require.ErrorContains(t, err, fmt.Sprintf("invalid ordinal %d", math.MinInt32))

	// Check merging with conflicting enum names.
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	err = cli.RegisterEnumMetadata(ctx, "ENUM_MERGED", map[string]int{"VAL0": 0})
	require.ErrorContains(t, err, "conflicting enum names for ordinal 0: old VAL1 vs new VAL0")

	// Check merging with conflicting enum ordinals.
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	err = cli.RegisterEnumMetadata(ctx, "ENUM_MERGED", map[string]int{"VAL1": 1})
	require.ErrorContains(t, err, "conflicting enum ordinals for name VAL1: old 0 vs new 1")

	// Check merging -- adding field
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	_, err = cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithField("id", 10))
	require.ErrorContains(t, err, "has been already registered as enum")

	// OK merging
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	err = cli.RegisterEnumMetadata(ctx, "ENUM_MERGED", map[string]int{"VAL1": 0, "VAL2": 1})
	require.NoError(t, err)
	enum2, err := cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(0))
	require.NoError(t, err)
	RequireBinaryObjectsEqual(t, enum2, enum1)
	RequireBinaryObjectsEqual(t, enum2, enum0)

	enumT, err := enum0.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, enumT.EnumNames(), map[string]int{"VAL1": 0, "VAL2": 1})

	// OK merging -- only add new name to ordinal
	if clearRegistry {
		cli.marsh.clearRegistry()
	}
	err = cli.RegisterEnumMetadata(ctx, "ENUM_MERGED", map[string]int{"VAL3": 2})
	require.NoError(t, err)
	enum3, err := cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(0))
	require.NoError(t, err)
	RequireBinaryObjectsEqual(t, enum3, enum2)
	RequireBinaryObjectsEqual(t, enum3, enum1)
	RequireBinaryObjectsEqual(t, enum3, enum0)

	enumT, err = enum0.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, enumT.EnumNames(), map[string]int{"VAL1": 0, "VAL2": 1, "VAL3": 2})

	for i := 0; i < 3; i++ {
		enum, err := cli.CreateBinaryObject(ctx, "ENUM_MERGED", WithEnumOrdinal(i))
		require.NoError(t, err)
		require.Equal(t, enum.EnumOrdinal(), i)
		require.Equal(t, enum.EnumName(), enumT.EnumName(i))
	}
}

func testBinaryEnumsAsFields(t *testing.T, cli *Client, cache *Cache) {
	ctx := context.Background()
	err := cli.RegisterEnumMetadata(ctx, "ru.gitverse.sbertech.client.filters.TestEnum$Enum",
		map[string]int{"VAL1": 0, "VAL2": 1, "VAL3": 2})
	require.NoError(t, err)
	enumFactory := func(ord int) BinaryObject {
		enum, err := cli.CreateBinaryObject(ctx, "ru.gitverse.sbertech.client.filters.TestEnum$Enum",
			WithEnumOrdinal(ord))
		require.NoError(t, err)
		return enum
	}
	enumType, err := enumFactory(0).Type(ctx)
	require.NoError(t, err)
	binaryObjectFactory := func(id int, ord int, arr []int) BinaryObject {
		opts := []BinaryObjectOption{WithField("id", id)}
		opts = append(opts, WithField("enumField", enumFactory(ord)))
		if len(arr) == 0 {
			opts = append(opts, WithNullField("enumArrayField", EnumArrayType))
		} else {
			enums := make([]BinaryObject, len(arr))
			for i, ord := range arr {
				enums[i] = enumFactory(ord)
			}
			enumArray, err := NewEnumArray(enumType, enums...)
			require.NoError(t, err)
			opts = append(opts, WithField("enumArrayField", enumArray))
		}

		ret, err := cli.CreateBinaryObject(ctx, "ru.gitverse.sbertech.client.filters.TestEnum", opts...)
		require.NoError(t, err)
		return ret
	}
	for i := 0; i < 6; i++ {
		var ords []int
		if i%2 == 0 {
			ords = []int{0, 1}
		}
		val := binaryObjectFactory(i, i%3, ords)
		err = cache.Put(ctx, i, val)
		require.NoError(t, err)
	}

	scanOpts := [][]ScanQueryOption{
		{
			WithScanQueryKeepBinary(),
			WithScanQueryFilter("ru.gitverse.sbertech.client.filters.TestEnumBinaryObjectFilter", WithClosureField("val", enumFactory(0))),
		},
		{
			WithScanQueryFilter("ru.gitverse.sbertech.client.filters.TestEnumFilter", WithClosureField("val", enumFactory(0))),
		},
	}
	for _, opts := range scanOpts {
		cur, err := cache.Scan(ctx, opts...)
		require.NoError(t, err)

		cnt := 0
		for cur.Next() {
			var id int64
			var bo BinaryObject
			err = cur.Scan(&id, &bo)
			require.NoError(t, err)

			val, err := cache.Get(ctx, id)
			require.NoError(t, err)
			realBo := val.(BinaryObject)
			RequireIgniteTypesEqual(t, bo, realBo)
			cnt++
		}
		require.NoError(t, cur.Err())
		require.GreaterOrEqual(t, cnt, 1)
	}
}

func testBinaryEnumsAsValues(t *testing.T, cli *Client, cache *Cache) {
	ctx := context.Background()
	err := cli.RegisterEnumMetadata(ctx, "ru.gitverse.sbertech.client.filters.TestEnum$Enum",
		map[string]int{"VAL1": 0, "VAL2": 1, "VAL3": 2})
	require.NoError(t, err)
	enumFactory := func(ord int) BinaryObject {
		enum, err := cli.CreateBinaryObject(ctx, "ru.gitverse.sbertech.client.filters.TestEnum$Enum", WithEnumOrdinal(ord))
		require.NoError(t, err)
		return enum
	}
	enumType, err := enumFactory(0).Type(ctx)
	require.NoError(t, err)
	enumArrayFactory := func(arr []int) BinaryEnumArray {
		enums := make([]BinaryObject, len(arr))
		for i, ord := range arr {
			enums[i] = enumFactory(ord)
		}
		enumArray, err := NewEnumArray(enumType, enums...)
		require.NoError(t, err)
		return enumArray
	}

	for _, isArray := range []bool{false, true} {
		testName := t.Name()
		if isArray {
			testName += "-enumArray"
		} else {
			testName += "-enum"
		}
		t.Run(testName, func(t *testing.T) {
			for i := 0; i < 6; i++ {
				var val interface{}
				if isArray {
					var ords []int
					if i%2 == 0 {
						ords = []int{i % 3, 0}
					}
					val = enumArrayFactory(ords)
				} else {
					val = enumFactory(i % 3)
				}
				err = cache.Put(ctx, i, val)
				require.NoError(t, err)
			}

			var scanOpts [][]ScanQueryOption
			if isArray {
				scanOpts = [][]ScanQueryOption{
					{
						WithScanQueryKeepBinary(),
						WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumArrayBinaryObjectFilter", WithClosureField("val", enumFactory(0))),
					},
					{
						WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumArrayFilter", WithClosureField("val", enumFactory(0))),
					},
				}
			} else {
				scanOpts = [][]ScanQueryOption{
					{
						WithScanQueryKeepBinary(),
						WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumBinaryObjectFilter", WithClosureField("val", enumFactory(0))),
					},
					{
						WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumFilter", WithClosureField("val", enumFactory(0))),
					},
				}
			}
			for _, opts := range scanOpts {
				cur, err := cache.Scan(ctx, opts...)
				require.NoError(t, err)

				cnt := 0
				for cur.Next() {
					var id int64
					var val interface{}
					err = cur.Scan(&id, &val)
					require.NoError(t, err)

					realVal, err := cache.Get(ctx, id)
					require.NoError(t, err)
					RequireIgniteTypesEqual(t, val, realVal)
					cnt++
				}
				require.NoError(t, cur.Err())
				require.GreaterOrEqual(t, cnt, 1)
			}
		})
	}
}

func (suite *BinaryObjectTestSuite) createIndexedCache() *Cache {
	cfg := CreateCacheConfiguration("PERSON", WithCacheKeyConfiguration("PERSON_KEY", "bucket_id"),
		WithQueryEntity("PERSON_KEY", "PERSON", WithTableName("PERSON"),
			WithQueryField("ID", "java.lang.Long", WithKey()),
			WithQueryField("BUCKET_ID", "java.lang.Long", WithKey()),
			WithQueryField("NAME", "java.lang.String", WithNotNull(), WithDefaultValue("")),
			WithQueryField("AGE", "java.lang.Short", WithNotNull(), WithDefaultValue(int16(2))),
			WithIndex("NAME_IDX", WithIndexType(Sorted), WithInlineSize(200), WithIndexField(IndexField{Name: "NAME", Asc: true})),
			WithIndex("NAME_AGE_IDX", WithIndexField(IndexField{Name: "AGE", Asc: true}), WithIndexField(IndexField{Name: "NAME", Asc: true})),
		))
	cache, err := suite.cli.CreateCacheWithConfiguration(context.Background(), cfg)
	require.NoError(suite.T(), err)
	return cache
}

func (suite *BinaryObjectTestSuite) createSimpleCache() *Cache {
	cache, err := suite.cli.CreateCache(context.Background(), "SIMPLE")
	require.NoError(suite.T(), err)
	return cache
}

func (suite *BinaryObjectTestSuite) runTests(tests ...struct {
	f         func(*testing.T, *Client, *Cache)
	isIndexed bool
}) {
	defer func() {
		suite.KillAllGrids()
	}()
	for _, compactFooter := range []bool{true, false} {
		_, err := suite.StartIgnite(testing2.WithCompactFooter(compactFooter))
		require.NoError(suite.T(), err)
		suite.cli, err = StartTestClient(context.Background())
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), compactFooter, suite.cli.marsh.isCompactFooter())
		for _, fixture := range tests {
			fName := runtime.FuncForPC(reflect.ValueOf(fixture.f).Pointer()).Name()
			suite.T().Run(fmt.Sprintf("%s/compactFooter-%t", fName, compactFooter), func(t *testing.T) {
				var cache *Cache
				if fixture.isIndexed {
					cache = suite.createIndexedCache()
				} else {
					cache = suite.createSimpleCache()
				}
				fixture.f(t, suite.cli, cache)
				DestroyAllCaches(t, suite.cli)
			})
		}
		_ = suite.cli.Close(context.Background())
		suite.KillAllGrids()
	}
}
