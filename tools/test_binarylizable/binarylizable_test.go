package test_binarylizable

import (
	"context"
	"fmt"
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gitverse.ru/sbertech/ignite-go-client"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type BinarylizableTestSuite struct {
	testing2.IgniteTestSuite
	cli *ignite.Client
}

func TestBinarylizables(t *testing.T) {
	suite.Run(t, new(BinarylizableTestSuite))
}

func (suite *BinarylizableTestSuite) TestBinarylizable() {
	cacheFactory := func(cli *ignite.Client) *ignite.Cache {
		cache, err := suite.cli.GetOrCreateCache(context.Background(), "test")
		require.NoError(suite.T(), err)
		return cache
	}
	suite.runTests(
		cacheFactory,
		testBinarylizablePrimitives,
		testBinarylizableSpecials,
		testBinarylizablePrimitiveArrays,
		testBinarylizableSpecialArrays,
		testBinarylizableObjectArrays,
		testBinarylizableCollections,
		testDotNetStruct,
		testEnumsBasic,
		testEnumsAsValues,
		testEnumsAsFields,
	)
}

func (suite *BinarylizableTestSuite) TestAffinityKey() {
	ctx := context.Background()
	cacheFactory := func(cli *ignite.Client) *ignite.Cache {
		ccfg := ignite.CreateCacheConfiguration(
			"COMPLEX",
			ignite.WithCacheKeyConfiguration("COMPLEX_KEY", "org_id"),
		)
		cache, err := suite.cli.GetOrCreateCacheWithConfiguration(ctx, ccfg)
		require.NoError(suite.T(), err)
		return cache
	}
	checkComplexStructsEqual := func(t *testing.T, val0, val1 interface{}) {
		switch realVal0 := val0.(type) {
		case *ComplexStruct:
			switch realVal1 := val1.(type) {
			case *ComplexStruct:
				require.Equal(suite.T(), realVal0.name, realVal1.name)
				require.Equal(t, realVal0.simple, realVal1.simple)
				checkBinaryObjectsEqual(t, realVal0.binaryObject, realVal1.binaryObject)
			default:
				require.Equal(t, val0, val1)
			}
		default:
			require.Equal(t, val0, val1)
		}
	}
	suite.runTests(
		cacheFactory,
		func(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
			err := RegisterCollectionsIgniteTypes(cli)
			require.NoError(t, err)

			key := &ComplexKey{
				id:    rand.Int(),
				orgId: rand.Int(),
			}
			val := &ComplexStruct{
				name:         testing2.MakeRandomString(100),
				simple:       createSimpleStruct(),
				binaryObject: createBinaryObject(t, cli),
			}
			// Put with binary object
			boKey, err := cli.CreateBinaryObject(ctx, "COMPLEX_KEY",
				ignite.WithAffinityKeyName("org_id"),
				ignite.WithField("id", key.id),
				ignite.WithField("org_id", key.orgId))
			require.NoError(suite.T(), err)
			boSimple, err := cli.CreateBinaryObject(ctx, "VALUE",
				ignite.WithField("id", val.simple.id),
				ignite.WithField("name", val.simple.name))
			require.NoError(suite.T(), err)
			boVal, err := cli.CreateBinaryObject(ctx, "COMPLEX_VALUE",
				ignite.WithField("name", val.name),
				ignite.WithField("simple_struct", boSimple),
				ignite.WithField("binary_object", val.binaryObject))
			require.NoError(suite.T(), err)
			err = cache.Put(context.Background(), boKey, boVal)
			require.NoError(suite.T(), err)

			cur, err := cache.Scan(ctx)
			require.NoError(suite.T(), err)
			require.True(t, cur.Next())
			var sKey, sVal ignite.Binarylizable
			err = cur.Scan(&sKey, &sVal)
			require.NoError(suite.T(), err)
			require.Equal(t, sKey, key)
			checkComplexStructsEqual(t, sVal, val)

			ok, err := cache.ContainsKey(context.Background(), &ComplexKey{
				id:    key.id,
				orgId: key.orgId,
			})
			require.NoError(t, err)
			require.True(t, ok)

			realVal, err := cache.Get(ctx, key)
			require.NoError(t, err)
			checkComplexStructsEqual(t, realVal, val)

			ok, err = cache.RemoveIfEquals(ctx, key, val)
			require.NoError(suite.T(), err)
			require.True(suite.T(), ok)
			ok, err = cache.ContainsKey(context.Background(), key)
			require.NoError(suite.T(), err)
			require.False(suite.T(), ok)
		},
	)
}

func (suite *BinarylizableTestSuite) runTests(cacheFactory func(*ignite.Client) *ignite.Cache, fixtures ...func(*testing.T, *ignite.Client, *ignite.Cache)) {
	defer func() {
		suite.KillAllGrids()
	}()
	for _, compactFooter := range []bool{true, false} {
		_, err := suite.StartIgnite(testing2.WithCompactFooter(compactFooter))
		require.NoError(suite.T(), err)
		suite.cli, err = startTestClient(context.Background())
		require.NoError(suite.T(), err)
		for _, fixture := range fixtures {
			fName := runtime.FuncForPC(reflect.ValueOf(fixture).Pointer()).Name()
			suite.T().Run(fmt.Sprintf("%s/compactFooter-%t", fName, compactFooter), func(t *testing.T) {
				ctx := context.Background()
				cache := cacheFactory(suite.cli)
				fixture(t, suite.cli, cache)
				err = suite.cli.DestroyCache(ctx, cache.Name())
				require.NoError(suite.T(), err)
			})
		}
		_ = suite.cli.Close(context.Background())
		suite.KillAllGrids()
	}
}

func testBinarylizablePrimitives(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterPrimitives(cli)
	require.NoError(t, err)
	ctx := context.Background()

	empty := &Primitives{}
	err = cache.Put(ctx, 1, empty)
	require.NoError(t, err)

	val, err := cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, empty, val)

	full := genFullPrimitives()
	err = cache.Put(ctx, 1, full)
	require.NoError(t, err)
	val, err = cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, full, val)
}

func testBinarylizablePrimitiveArrays(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterPrimitiveArrays(cli)
	require.NoError(t, err)
	ctx := context.Background()

	empty := &PrimitiveArrays{}
	err = cache.Put(ctx, 1, empty)
	require.NoError(t, err)

	val, err := cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, empty, val)

	full := genFullPrimitiveArrays()
	err = cache.Put(ctx, 1, full)
	require.NoError(t, err)
	val, err = cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, full, val)
}

func testBinarylizableSpecials(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterSpecialsIgniteTypes(cli)
	require.NoError(t, err)
	ctx := context.Background()

	empty := &Specials{
		fTimestamp: time.Now().Truncate(0), // remove monotonic part
	}
	err = cache.Put(ctx, 1, empty)
	require.NoError(t, err)

	val, err := cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, empty, val)

	full := genFullSpecials()
	err = cache.Put(ctx, 1, full)
	require.NoError(t, err)
	val, err = cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, full, val)
}

func testBinarylizableSpecialArrays(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterSpecialsIgniteTypes(cli)
	require.NoError(t, err)
	ctx := context.Background()

	empty := &SpecialArrays{}
	err = cache.Put(ctx, 1, empty)
	require.NoError(t, err)

	val, err := cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, empty, val)

	full := genFullSpecialsArrays()
	err = cache.Put(ctx, 1, full)
	require.NoError(t, err)
	val, err = cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, full, val)
}

func testBinarylizableObjectArrays(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterCollectionsIgniteTypes(cli)
	require.NoError(t, err)
	ctx := context.Background()

	empty := &ObjectArrays{}
	err = cache.Put(ctx, 1, empty)
	require.NoError(t, err)

	val, err := cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, empty, val)

	full := genFullObjectArrays(t, cli)
	err = cache.Put(ctx, 1, full)
	require.NoError(t, err)
	val, err = cache.Get(ctx, 1)
	require.NoError(t, err)
	valReal := val.(*ObjectArrays)
	checkObjectArraysEqual(t, full.ifaceArray, valReal.ifaceArray)
	checkBinaryObjectArraysEqual(t, full.boArray, valReal.boArray)
	require.Equal(t, full.simpleStructArray, valReal.simpleStructArray)
}

func testBinarylizableCollections(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterCollectionsIgniteTypes(cli)
	require.NoError(t, err)
	boFactory := func() ignite.BinaryObject {
		return createBinaryObject(t, cli)
	}

	ctx := context.Background()

	empty := &Collections{}
	err = cache.Put(ctx, 1, empty)
	require.NoError(t, err)

	val, err := cache.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, empty, val)

	full := &Collections{
		skipMap0: make(map[ignite.Binarylizable]string),
		skipMap1: make(map[SimpleKey]string),
		skipMap2: make(map[*SimpleKey]string),
		skipMap3: make(map[ignite.BinaryObject]string),
		skipMap4: make(map[*apd.Decimal]string),
	}
	full.list = ignite.NewArrayList[interface{}]("string", createSimpleStruct(), createSimpleKey())
	full.ignMap = ignite.NewLinkedHashMap(
		ignite.KeyValue{Key: createSimpleKey(), Value: createSimpleStruct()},
		ignite.KeyValue{Key: createSimpleKey(), Value: nil},
		ignite.KeyValue{Key: createSimpleKey(), Value: boFactory()},
		ignite.KeyValue{Key: createSimpleKey(), Value: "test"},
		ignite.KeyValue{Key: "test", Value: uuid.New()},
		ignite.KeyValue{Key: createSimpleKey(), Value: boFactory()},
		ignite.KeyValue{Key: boFactory(), Value: createSimpleStruct()},
		ignite.KeyValue{Key: "123", Value: nil},
	)
	full.simpleMap = map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
	}
	dec0, _, _ := apd.NewFromString("100.53")
	dec := *dec0
	timestamp := time.Now().Truncate(0) // remove monotonic part.
	full.iMap = map[string]interface{}{
		"bool":           rand.Intn(2) == 0,
		"byte":           int8(rand.Intn(255)),
		"short":          int16(rand.Intn(1 << 15)),
		"char":           uint16(rand.Intn(1 << 15)),
		"int32":          rand.Int31(),
		"int64":          rand.Int63(),
		"float":          rand.Float32(),
		"double":         rand.Float64(),
		"string":         testing2.MakeRandomString(200),
		"pDecimal":       &dec,
		"decimal":        dec,
		"stringArray":    testing2.CreatePSlice(testing2.MakeRandomString(100), testing2.MakeRandomString(100), testing2.MakeRandomString(100)),
		"uuidArray":      testing2.CreatePSlice(uuid.New(), uuid.New(), uuid.New()),
		"timeArray":      testing2.CreatePSlice(ignite.NewTime(timestamp), ignite.NewTime(timestamp.Add(time.Minute*10)), ignite.NewTime(timestamp.Add(time.Minute*20))),
		"dateArray":      testing2.CreatePSlice(ignite.NewDate(timestamp), ignite.NewDate(timestamp.Add(time.Minute*10)), ignite.NewDate(timestamp.Add(time.Minute*20))),
		"timeStampArray": testing2.CreatePSlice(timestamp, timestamp.Add(time.Minute*10), timestamp.Add(time.Minute*20)),
		"decimalArray":   []*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)},
		"binaryObject":   boFactory(),
		"simpleSruct":    createSimpleStruct(),
		"objArray":       []interface{}{"string", createSimpleStruct(), createSimpleKey(), nil},
		"arrayList":      ignite.NewArrayList[interface{}]("string", createSimpleStruct(), createSimpleKey()),
		"map": ignite.NewLinkedHashMap(
			ignite.KeyValue{Key: createSimpleKey(), Value: createSimpleStruct()},
			ignite.KeyValue{Key: "test", Value: nil},
			ignite.KeyValue{Key: createSimpleKey(), Value: boFactory()},
			ignite.KeyValue{Key: createSimpleKey(), Value: "test"}),
	}
	full.decMap = map[apd.Decimal]*SimpleStruct{
		dec: createSimpleStruct(),
	}
	full.boMap = map[string]ignite.BinaryObject{
		"key1": boFactory(),
		"key2": boFactory(),
		"key3": nil,
	}
	full.biMap = map[string]ignite.Binarylizable{
		"key1": createSimpleStruct(),
		"key2": createSimpleStruct(),
		"key3": nil,
	}
	err = cache.Put(ctx, 1, full)
	require.NoError(t, err)
	val, err = cache.Get(ctx, 1)
	require.NoError(t, err)
	valReal := val.(*Collections)
	checkCollectionsEqual(t, full.list, valReal.list)
	checkMapsEqual(t, full.ignMap, valReal.ignMap)
	checkGoMapsEqual(t, full.simpleMap, valReal.simpleMap)
	checkGoMapsEqual(t, full.boMap, valReal.boMap)
	checkGoMapsEqual(t, full.biMap, valReal.biMap)
	checkGoMapsEqual(t, full.iMap, valReal.iMap)
	checkGoMapsEqual(t, full.decMap, valReal.decMap)
	require.Nil(t, valReal.skipMap0)
	require.Nil(t, valReal.skipMap1)
	require.Nil(t, valReal.skipMap2)
	require.Nil(t, valReal.skipMap3)
	require.Nil(t, valReal.skipMap4)
}

func testDotNetStruct(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterCollectionsIgniteTypes(cli)
	require.NoError(t, err)
	ctx := context.Background()

	expVal := &DotNetStruct{name: "test"}
	boValue, err := cli.CreateBinaryObject(
		ctx, "DOT_NET_VALUE",
		ignite.WithMarshallerPlatform(ignite.DotNetMarshaller),
		ignite.WithField("name", "test"))
	require.NoError(t, err)

	err = cache.Put(ctx, "test", boValue)
	require.NoError(t, err)

	realVal, err := cache.Get(ctx, "test")
	require.NoError(t, err)
	require.Equal(t, expVal, realVal)
}

func testEnumsBasic(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterEnumsIgniteTypes(cli)
	require.NoError(t, err)

	ctx := context.Background()
	err = cache.Put(ctx, "test", SimpleEnumVal2)
	require.NoError(t, err)

	realVal, err := cache.Get(ctx, "test")
	require.NoError(t, err)
	checkElementsEqual(t, SimpleEnumVal2, realVal)

	enumArr := []ignite.Enum{SimpleEnumVal2, SimpleEnumVal1, SimpleEnumVal4, nil}
	err = cache.Put(ctx, "test", enumArr)
	require.NoError(t, err)
	realVal, err = cache.Get(ctx, "test")
	require.NoError(t, err)
	realArr, ok := realVal.([]interface{})
	require.True(t, ok)
	for i, v := range realArr {
		checkElementsEqual(t, enumArr[i], v)
	}
}

func testEnumsAsValues(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	ctx := context.Background()
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
					var ords []Enum
					if i%2 == 0 {
						ords = []Enum{Enum(i % 3), EnumVal1}
					} else {
						ords = []Enum{EnumVal3}
					}
					val = ords
				} else {
					val = Enum(i % 3)
				}
				err := cache.Put(ctx, i, val)
				require.NoError(t, err)

				realVal, err := cache.Get(ctx, i)
				require.NoError(t, err)
				if isArray {
					arr, ok := val.([]Enum)
					require.True(t, ok)
					realArr, ok := realVal.([]interface{})
					require.True(t, ok)
					require.Equal(t, len(arr), len(realArr))
					for i, v := range arr {
						checkElementsEqual(t, v, realArr[i])
					}
				} else {
					checkElementsEqual(t, val, realVal)
				}
				require.NoError(t, err)
			}

			var scanOpts [][]ignite.ScanQueryOption
			if isArray {
				scanOpts = [][]ignite.ScanQueryOption{
					{
						ignite.WithScanQueryKeepBinary(),
						ignite.WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumArrayBinaryObjectFilter",
							ignite.WithClosureField("val", EnumVal1)),
					},
					{
						ignite.WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumArrayFilter",
							ignite.WithClosureField("val", EnumVal1)),
					},
				}
			} else {
				scanOpts = [][]ignite.ScanQueryOption{
					{
						ignite.WithScanQueryKeepBinary(),
						ignite.WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumBinaryObjectFilter",
							ignite.WithClosureField("val", EnumVal1)),
					},
					{
						ignite.WithScanQueryFilter("ru.gitverse.sbertech.client.filters.EnumFilter",
							ignite.WithClosureField("val", EnumVal1)),
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
					checkElementsEqual(t, val, realVal)
					cnt++
				}
				require.NoError(t, cur.Err())
				require.GreaterOrEqual(t, cnt, 1)
			}
		})
	}
}

func testEnumsAsFields(t *testing.T, cli *ignite.Client, cache *ignite.Cache) {
	err := RegisterEnumsIgniteTypes(cli)
	require.NoError(t, err)
	err = RegisterTestEnumsIgniteTypes(cli)
	require.NoError(t, err)

	ctx := context.Background()

	for i := 0; i < 6; i++ {
		var enumArray []Enum
		if i%2 == 0 {
			enumArray = []Enum{EnumVal1, EnumVal2}
		}
		val := &TestEnum{
			id:             i,
			enumField:      Enum(i % 3),
			enumArrayField: enumArray,
		}
		err = cache.Put(ctx, i, val)
		require.NoError(t, err)

		realVal, err := cache.Get(ctx, i)
		require.NoError(t, err)
		checkElementsEqual(t, val, realVal)
	}

	scanOpts := [][]ignite.ScanQueryOption{
		{
			ignite.WithScanQueryKeepBinary(),
			ignite.WithScanQueryFilter("ru.gitverse.sbertech.client.filters.TestEnumBinaryObjectFilter",
				ignite.WithClosureField("val", EnumVal1)),
		},
		{
			ignite.WithScanQueryFilter("ru.gitverse.sbertech.client.filters.TestEnumFilter",
				ignite.WithClosureField("val", EnumVal1)),
		},
	}
	for _, opts := range scanOpts {
		cur, err := cache.Scan(ctx, opts...)
		require.NoError(t, err)

		cnt := 0
		for cur.Next() {
			var id int64
			var val *TestEnum
			err = cur.Scan(&id, &val)
			require.NoError(t, err)

			realVal, err := cache.Get(ctx, id)
			require.NoError(t, err)
			checkElementsEqual(t, val, realVal)
			cnt++
		}
		require.NoError(t, cur.Err())
		require.GreaterOrEqual(t, cnt, 1)
	}
}

func genFullPrimitives() *Primitives {
	rndBool := rand.Intn(2) == 0
	rndByte := uint8(rand.Intn(255))
	rndSByte := int8(rand.Intn(255))
	rndShort := int16(rand.Intn(1 << 15))
	rndChar := uint16(rand.Intn(1 << 15))
	rndInt32 := rand.Int31()
	rndUint32 := uint32(rand.Int31())
	rndInt64 := rand.Int63()
	rndUint64 := rand.Uint64()
	rndInt := rand.Int()
	rndUint := uint(rand.Int())
	rndFloat32 := rand.Float32()
	rndFloat64 := rand.Float64()
	return &Primitives{
		fBool:    rndBool,
		pBool:    &rndBool,
		fByte:    rndByte,
		pByte:    &rndByte,
		fUByte:   rndByte,
		pUByte:   &rndByte,
		fSByte:   rndSByte,
		pSByte:   &rndSByte,
		fShort:   rndShort,
		pShort:   &rndShort,
		fChar:    rndChar,
		pChar:    &rndChar,
		fInt32:   rndInt32,
		pInt32:   &rndInt32,
		fUint32:  rndUint32,
		pUint32:  &rndUint32,
		fInt64:   rndInt64,
		pInt64:   &rndInt64,
		fUint64:  rndUint64,
		pUint64:  &rndUint64,
		fInt:     rndInt,
		pInt:     &rndInt,
		fUint:    rndUint,
		pUint:    &rndUint,
		fFloat32: rndFloat32,
		pFloat32: &rndFloat32,
		fFloat64: rndFloat64,
		pFloat64: &rndFloat64,
	}
}

func genFullSpecials() *Specials {
	ts := time.Now().Truncate(0) // remove monotonic part
	str := testing2.MakeRandomString(120)
	uid := uuid.New()
	dec, _, _ := apd.NewFromString("100.500")
	ignTime := ignite.NewTime(ts)
	ignDate := ignite.NewDate(ts)
	return &Specials{
		fString:    str,
		pString:    &str,
		fUuid:      uid,
		pUuid:      &uid,
		fDate:      ignDate,
		pDate:      &ignDate,
		fTime:      ignTime,
		pTime:      &ignTime,
		fTimestamp: ts,
		pTimestamp: &ts,
		fDecimal:   *dec,
		pDecimal:   dec,
	}
}

func genFullPrimitiveArrays() *PrimitiveArrays {
	return &PrimitiveArrays{
		boolArray:    createRandPrimitiveArray(func() bool { return rand.Intn(2) != 0 }),
		byteArray:    createRandPrimitiveArray(func() byte { return byte(rand.Intn(1<<8 - 1)) }),
		sByteArray:   createRandPrimitiveArray(func() int8 { return int8(rand.Intn(1<<8 - 1)) }),
		uByteArray:   createRandPrimitiveArray(func() uint8 { return uint8(rand.Intn(1<<8 - 1)) }),
		shortArray:   createRandPrimitiveArray(func() int16 { return int16(rand.Intn(1<<8*2 - 1)) }),
		charArray:    createRandPrimitiveArray(func() uint16 { return uint16(rand.Intn(1<<8*2 - 1)) }),
		int32Array:   createRandPrimitiveArray(func() int32 { return rand.Int31() }),
		uint32Array:  createRandPrimitiveArray(func() uint32 { return rand.Uint32() }),
		int64Array:   createRandPrimitiveArray(func() int64 { return rand.Int63() }),
		uint64Array:  createRandPrimitiveArray(func() uint64 { return rand.Uint64() }),
		intArray:     createRandPrimitiveArray(func() int { return rand.Int() }),
		uintArray:    createRandPrimitiveArray(func() uint { return uint(rand.Uint64()) }),
		float32Array: createRandPrimitiveArray(func() float32 { return rand.Float32() }),
		float64Array: createRandPrimitiveArray(func() float64 { return rand.Float64() }),
	}
}

func genFullSpecialsArrays() *SpecialArrays {
	timestamp := time.Now().Truncate(0) // remove monotonic part.
	return &SpecialArrays{
		stringPArray:    testing2.CreatePSlice(testing2.MakeRandomString(100), testing2.MakeRandomString(100), testing2.MakeRandomString(100)),
		stringArray:     []string{testing2.MakeRandomString(100), testing2.MakeRandomString(100), testing2.MakeRandomString(100)},
		uuidPArray:      testing2.CreatePSlice(uuid.New(), uuid.New(), uuid.New()),
		uuidArray:       []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
		timePArray:      testing2.CreatePSlice(ignite.NewTime(timestamp), ignite.NewTime(timestamp.Add(time.Minute*10)), ignite.NewTime(timestamp.Add(time.Minute*20))),
		timeArray:       []ignite.Time{ignite.NewTime(timestamp), ignite.NewTime(timestamp.Add(time.Minute * 10)), ignite.NewTime(timestamp.Add(time.Minute * 20))},
		datePArray:      testing2.CreatePSlice(ignite.NewDate(timestamp), ignite.NewDate(timestamp.Add(time.Minute*10)), ignite.NewDate(timestamp.Add(time.Minute*20))),
		dateArray:       []ignite.Date{ignite.NewDate(timestamp), ignite.NewDate(timestamp.Add(time.Minute * 10)), ignite.NewDate(timestamp.Add(time.Minute * 20))},
		timeStampPArray: testing2.CreatePSlice(timestamp, timestamp.Add(time.Minute*10), timestamp.Add(time.Minute*20)),
		timeStampArray:  []time.Time{timestamp, timestamp.Add(time.Minute * 10), timestamp.Add(time.Minute * 20)},
		decimalPArray:   []*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)},
		decimalArray:    testing2.FromPSlice([]*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)}),
	}
}

func genFullObjectArrays(t *testing.T, cli *ignite.Client) *ObjectArrays {
	boFactory := func() ignite.BinaryObject {
		return createBinaryObject(t, cli)
	}
	return &ObjectArrays{
		ifaceArray:        []interface{}{createSimpleStruct(), "test", boFactory(), nil},
		boArray:           []ignite.BinaryObject{boFactory(), boFactory(), nil},
		simpleStructArray: []*SimpleStruct{createSimpleStruct(), createSimpleStruct(), nil},
	}
}
