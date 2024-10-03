package ignite

import (
	"context"
	"fmt"
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"math/rand"
	"strings"
	"testing"
	"time"
)

type ScanQueryTestSuite struct {
	testing2.IgniteTestSuite
	client *Client
	cache  *Cache
}

func TestScanQueryTestSuite(t *testing.T) {
	suite.Run(t, new(ScanQueryTestSuite))
}

func (suite *ScanQueryTestSuite) SetupSuite() {
	var err error
	for i := 0; i < 3; i++ {
		_, err = suite.StartIgnite()
		if err != nil {
			suite.T().Fatal("Failed to start ignite instance", err)
		}
	}
	suite.client, err = StartTestClient(context.Background(),
		WithAddresses(defaultAddress+":10800", defaultAddress+":10801", defaultAddress+":10802"))
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	suite.cache, err = suite.client.GetOrCreateCache(context.Background(), "test")
	if err != nil {
		suite.T().Fatal("failed to create test cache", err)
	}
}

func (suite *ScanQueryTestSuite) TearDownTest() {
	err := suite.cache.ClearAll(context.Background())
	if err != nil {
		suite.T().Fatal("failed to clear cache", err)
	}
	sz, err := suite.cache.Size(context.Background())
	if err != nil {
		suite.T().Fatal("failed to clear cache", err)
	}
	require.Equal(suite.T(), uint64(0), sz, fmt.Sprintf("cache is not empty: size=%d", sz))
}

func (suite *ScanQueryTestSuite) TearDownSuite() {
	suite.KillAllGrids()
	if suite.client != nil {
		_ = suite.client.Close(context.Background())
		suite.client = nil
	}
}

type scanQueryFixture struct {
	name             string
	keyFactory       func(i int) interface{}
	valueFactory     func(i int) interface{}
	queryOptsFactory func() []func(*scanQueryOpts)
	scanChecker      func(cursor Cursor)
	rowsCountChecker func(int)
}

func (suite *ScanQueryTestSuite) TestScanWithDifferentCursorSize() {
	recordsCount := 2048
	ctx := context.Background()
	fixtureFactory := func(pageSize int) scanQueryFixture {
		return scanQueryFixture{
			name: fmt.Sprintf("pageSize=%d", pageSize),
			keyFactory: func(i int) interface{} {
				return int32(i)
			},
			valueFactory: func(i int) interface{} {
				val, err := suite.client.CreateBinaryObject(ctx, "Value", WithField("name", fmt.Sprintf("name-%d", i)))
				require.NoError(suite.T(), err)
				return val
			},
			scanChecker: func(cursor Cursor) {
				scanChecker[int32, BinaryObject](suite, cursor)
			},
			queryOptsFactory: func() []func(*scanQueryOpts) {
				return []func(opts *scanQueryOpts){WithScanQueryPageSize(pageSize)}
			},
			rowsCountChecker: func(actualCount int) {
				require.Equal(suite.T(), recordsCount, actualCount)
			},
		}
	}
	fixtures := make([]scanQueryFixture, 0)
	for pageSz := 0; pageSz <= 256; pageSz = pageSz + 32 {
		fixtures = append(fixtures, fixtureFactory(pageSz))
	}
	runScanQueryFixtures(suite, recordsCount, fixtures)
}

func (suite *ScanQueryTestSuite) TestScanWithPartitions() {
	recordsCount := 2048
	ctx := context.Background()
	fixtureFactory := func(name string, rowsCount int, qOpts ...func(*scanQueryOpts)) scanQueryFixture {
		return scanQueryFixture{
			name: name,
			keyFactory: func(i int) interface{} {
				return int32(i)
			},
			valueFactory: func(i int) interface{} {
				val, err := suite.client.CreateBinaryObject(ctx, "Value", WithField("name", fmt.Sprintf("name-%d", i)))
				require.NoError(suite.T(), err)
				return val
			},
			scanChecker: func(cursor Cursor) {
				scanChecker[int32, BinaryObject](suite, cursor)
			},
			queryOptsFactory: func() []func(*scanQueryOpts) {
				return append(qOpts, WithScanQueryPageSize(10))
			},
			rowsCountChecker: func(actualCount int) {
				require.Less(suite.T(), actualCount, rowsCount+100)
				require.Greater(suite.T(), actualCount, rowsCount-100)
			},
		}
	}
	fixtures := []scanQueryFixture{
		fixtureFactory("localQuery", recordsCount/suite.GridsCount(), WithScanQueryLocal()),
		fixtureFactory("withPartition", recordsCount/1024, WithScanQueryPartition(512)),
	}
	runScanQueryFixtures(suite, recordsCount, fixtures)
}

func (suite *ScanQueryTestSuite) TestScanFilter() {
	recordsCount := 2048
	ctx := context.Background()
	fixtures := make([]scanQueryFixture, 0)
	for _, setCloPlatform := range []bool{true, false} {
		for _, filter := range []string{
			"ru.gitverse.sbertech.client.filters.PersonByNameBinaryObjectFilter",
			"ru.gitverse.sbertech.client.filters.PersonByNamePojoFilter",
		} {
			testName := "int64->"
			if strings.Contains(filter, "BinaryObject") {
				testName += "BinaryObject"
			} else {
				testName += "POJO"
			}
			fixtures = append(fixtures, scanQueryFixture{
				name: fmt.Sprintf("%s,set-platform-closure=%t", testName, setCloPlatform),
				keyFactory: func(i int) interface{} {
					return int64(i)
				},
				valueFactory: func(i int) interface{} {
					val, err := suite.client.CreateBinaryObject(ctx, "ru.gitverse.sbertech.client.filters.Person",
						WithField("name", fmt.Sprintf("name-%d", i)), WithField("age", int16(i)))
					require.NoError(suite.T(), err)
					return val
				},
				queryOptsFactory: func() []func(*scanQueryOpts) {
					ret := make([]func(*scanQueryOpts), 0)
					filterOpts := []func(opts *closureOpts){WithClosureField("name", fmt.Sprintf("name-%d", 10))}
					if setCloPlatform {
						filterOpts = append(filterOpts, WithServerClosurePlatform(JavaServerClosure))
					}
					ret = append(ret, WithScanQueryFilter(filter, filterOpts...))
					if strings.Contains(filter, "BinaryObject") {
						ret = append(ret, WithScanQueryKeepBinary())
					}
					return ret
				},
				scanChecker: func(cursor Cursor) {
					scanChecker[int64, BinaryObject](suite, cursor)
				},
				rowsCountChecker: func(actualCnt int) {
					require.Equal(suite.T(), 1, actualCnt)
				},
			})
		}
	}
	runScanQueryFixtures(suite, recordsCount, fixtures)
}

func (suite *ScanQueryTestSuite) TestCursorScan() {
	timestamp := time.Now().Truncate(0) // remove monotonic part.
	cli := suite.client
	fixtures := []struct {
		name    string
		checker func(t *testing.T)
	}{
		{"bool", func(t *testing.T) { testCursorScan[bool](suite, true) }},
		{"int8", func(t *testing.T) { testCursorScan[int8](suite, 127) }},
		{"int16", func(t *testing.T) { testCursorScan[int16](suite, 10) }},
		{"uint16", func(t *testing.T) { testCursorScan[uint16](suite, 'a') }},
		{"int32", func(t *testing.T) { testCursorScan[int32](suite, -1<<20) }},
		{"int64", func(t *testing.T) { testCursorScan[int64](suite, 1<<20) }},
		{"float32", func(t *testing.T) { testCursorScan[float32](suite, 1.0*1<<20) }},
		{"float64", func(t *testing.T) { testCursorScan[float64](suite, 1.0*1<<20) }},
		{"boolArray", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() bool { return rand.Intn(3) != 0 }))
		}},
		{"byteArray", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() byte { return byte(rand.Intn(1<<8 - 1)) }))
		}},
		{"shortArray", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() int16 { return int16(rand.Intn(1<<16 - 1)) }))
		}},
		{"charArray", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() uint16 { return uint16(rand.Intn(1<<16 - 1)) }))
		}},
		{"int32Array", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() int32 { return rand.Int31() }))
		}},
		{"int64Array", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() int64 { return rand.Int63() }))
		}},
		{"floatArray", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() float32 { return rand.Float32() }))
		}},
		{"doubleArray", func(t *testing.T) {
			testCursorScan(suite, createRandPrimitiveArray(func() float64 { return rand.Float64() }))
		}},
		{"string", func(t *testing.T) { testCursorScan(suite, "test") }},
		{"uuid", func(t *testing.T) { testCursorScan(suite, uuid.New()) }},
		{"time", func(t *testing.T) { testCursorScan(suite, NewTime(timestamp)) }},
		{"date", func(t *testing.T) { testCursorScan(suite, NewDate(timestamp)) }},
		{"timestamp", func(t *testing.T) { testCursorScan(suite, timestamp) }},
		{"decimal", func(t *testing.T) { testCursorScan(suite, apd.New(100500, -3)) }},
		{"stringArray", func(t *testing.T) { testCursorScan(suite, testing2.CreatePSlice("test1", "test2")) }},
		{"uuidArray", func(t *testing.T) { testCursorScan(suite, testing2.CreatePSlice(uuid.New(), uuid.New(), uuid.New())) }},
		{"timeArray", func(t *testing.T) {
			testCursorScan(suite, testing2.CreatePSlice(NewTime(timestamp), NewTime(timestamp.Add(time.Minute*10)), NewTime(timestamp.Add(time.Minute*20))))
		}},
		{"dateArray", func(t *testing.T) {
			testCursorScan(suite, testing2.CreatePSlice(NewDate(timestamp), NewDate(timestamp.Add(time.Minute*10)), NewDate(timestamp.Add(time.Minute*20))))
		}},
		{"timeStampArray", func(t *testing.T) {
			testCursorScan(suite, testing2.CreatePSlice(timestamp, timestamp.Add(time.Minute*10), timestamp.Add(time.Minute*20)))
		}},
		{"decimalArray", func(t *testing.T) {
			testCursorScan(suite, []*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)})
		}},
		{"binaryObject", func(t *testing.T) { testCursorScan(suite, createTestBinaryObject(t, cli, 100500)) }},
		{"objectArray", func(t *testing.T) {
			testCursorScan(suite, []interface{}{createTestBinaryObject(t, cli, 10), createTestBinaryObject(t, cli, 20), nil})
		}},
		{"objectArrayMixed", func(t *testing.T) { testCursorScan(suite, []interface{}{createTestBinaryObject(t, cli, 10), "test"}) }},
		{"arrayList", func(t *testing.T) { testCursorScan(suite, NewArrayList("test", "test")) }},
		{"arrayListMixed", func(t *testing.T) {
			testCursorScan(suite, NewArrayList([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...))
		}},
		{"linkedList", func(t *testing.T) { testCursorScan(suite, NewLinkedList("test", "test")) }},
		{"linkedListMixed", func(t *testing.T) {
			testCursorScan(suite, NewLinkedList([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...))
		}},
		{"hashSet", func(t *testing.T) { testCursorScan(suite, NewHashSet("test1", "test2")) }},
		{"hashSetMixed", func(t *testing.T) {
			testCursorScan(suite, NewHashSet([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...))
		}},
		{"linkedHashSet", func(t *testing.T) { testCursorScan(suite, NewHashSet("test1")) }},
		{"linkedHashSetMixed", func(t *testing.T) {
			testCursorScan(suite, NewHashSet([]interface{}{createTestBinaryObject(t, cli, 10), nil, "test"}...))
		}},
		{"hashMap", func(t *testing.T) {
			testCursorScan(suite, ToHashMap(map[string]int32{"test1": 1, "test2": 2}))
		}},
		{"hashMapMixed", func(t *testing.T) {
			testCursorScan(suite, NewHashMap([]KeyValue{{"test1", createTestBinaryObject(t, cli, 10)}, {int32(10), "test"}}...))
		}},
		{"linkedHashMap", func(t *testing.T) {
			testCursorScan(suite, ToLinkedHashMap(map[string]int32{"test1": 1, "test2": 2}))
		}},
		{"linkedHashMapMixed", func(t *testing.T) {
			testCursorScan(suite, NewLinkedHashMap([]KeyValue{{"test1", createTestBinaryObject(t, cli, 10)}, {int32(10), "test"}}...))
		}},
	}
	for _, fixture := range fixtures {
		suite.T().Run(fixture.name, func(t *testing.T) {
			fixture.checker(t)
		})
	}
}

func testCursorScan[T any](suite *ScanQueryTestSuite, exp T) {
	defer suite.TearDownTest()
	ctx := context.Background()
	expKey := "val-key"
	err := suite.cache.Put(ctx, expKey, exp)
	require.NoError(suite.T(), err)

	rows, err := suite.cache.Scan(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), rows.Next())

	var key string
	var val T
	err = rows.Scan(&key, &val)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), expKey, key)
	RequireIgniteTypesEqual(suite.T(), exp, val)

	var pkey *string
	var pval *T
	err = rows.Scan(&pkey, &pval)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), expKey)
	require.Equal(suite.T(), expKey, *pkey)
	require.NotNil(suite.T(), pval)
	RequireIgniteTypesEqual(suite.T(), exp, *pval)

	var ikey interface{}
	var ival interface{}
	err = rows.Scan(&ikey, &ival)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), expKey, ikey)
	RequireIgniteTypesEqual(suite.T(), exp, ival)
}

func runScanQueryFixtures(suite *ScanQueryTestSuite, recordsCount int, fixtures []scanQueryFixture) {
	ctx := context.Background()
	for _, fixture := range fixtures {
		suite.T().Run(fixture.name, func(t *testing.T) {
			defer suite.TearDownTest()
			for i := 0; i < recordsCount; i++ {
				key := fixture.keyFactory(i)
				val := fixture.valueFactory(i)
				err := suite.cache.Put(ctx, key, val)
				require.NoError(suite.T(), err)
			}

			queryOpts := fixture.queryOptsFactory()
			rows, err := suite.cache.Scan(ctx, queryOpts...)
			require.NoError(t, err)
			require.NotNil(t, rows)
			defer func() {
				require.NotNil(suite.T(), rows)
				err = rows.Close()
				require.NoError(suite.T(), err)
			}()

			cnt := 0
			for rows.Next() {
				fixture.scanChecker(rows)
				cnt++
			}
			fixture.rowsCountChecker(cnt)
			require.NoError(suite.T(), rows.Err())
		})
	}
}

func scanChecker[K any, V any](suite *ScanQueryTestSuite, rows Cursor) {
	ctx := context.Background()
	var key K
	var val V
	err := rows.Scan(&key, &val)
	require.NoError(suite.T(), err)
	var ikey interface{}
	var ival interface{}
	err = rows.Scan(&ikey, &ival)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), key, ikey)
	require.Equal(suite.T(), val, ival)

	val1, err := suite.cache.Get(ctx, key)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), val, val1)
	ival1, err := suite.cache.Get(ctx, ikey)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), val, ival1)
}

func TestConvertAssign(t *testing.T) {
	var pStr *int32
	err := convertAssign(&pStr, nil)
	require.NoError(t, err)
	require.Nil(t, pStr)
	err = convertAssign(&pStr, int32(100))
	require.NoError(t, err)
	require.Equal(t, int32(100), *pStr)

	var m Map
	err = convertAssign(&m, nil)
	require.NoError(t, err)
	require.True(t, m.IsNull())

	var pm *Map
	err = convertAssign(&pm, nil)
	require.NoError(t, err)
	require.Nil(t, pm)

	var col Collection
	err = convertAssign(&col, nil)
	require.NoError(t, err)
	require.True(t, col.IsNull())

	var pcol *Collection
	err = convertAssign(&pcol, nil)
	require.NoError(t, err)
	require.Nil(t, pcol)
}
