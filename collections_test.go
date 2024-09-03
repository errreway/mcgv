package ignite

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
)

func (suite *CacheTestSuite) TestCollections() {
	factories := []struct {
		name    string
		factory func(elems []interface{}) Collection
	}{
		{
			"ArrayList",
			func(elems []interface{}) Collection {
				return NewArrayList(elems...)
			},
		},
		{
			"LinkedList",
			func(elems []interface{}) Collection {
				return NewLinkedList(elems...)
			},
		},
		{
			"HashSet",
			func(elems []interface{}) Collection {
				return NewHashSet(elems...)
			},
		},
		{
			"LinkedHashSet",
			func(elems []interface{}) Collection {
				return NewLinkedHashSet(elems...)
			},
		},
		{
			"UserCollection",
			func(elems []interface{}) Collection {
				return NewUserCollection(elems...)
			},
		},
	}
	fixtures := [][]interface{}{
		{"test", "bucket"},
		{int32(10), int32(20)},
		{createTestBinaryObject(suite.T(), suite.client, 10), createTestBinaryObject(suite.T(), suite.client, 10)},
		{createTestBinaryObject(suite.T(), suite.client, 10), createTestBinaryObject(suite.T(), suite.client, 20)},
		{nil, "test", int32(10), createTestBinaryObject(suite.T(), suite.client, 20)},
	}

	for _, factory := range factories {
		for _, fixture := range fixtures {
			suite.T().Run(fmt.Sprintf("%s: %s", factory.name, fixture), func(t *testing.T) {
				val := factory.factory(fixture)
				suite.putGetCollectionTest(suite.T(), "col-key", val)
			})
		}
	}

	for _, fixture := range fixtures {
		suite.T().Run(fmt.Sprintf("ObjectArray: %s", fixture), func(t *testing.T) {
			suite.putGetCollectionTest(suite.T(), "obj-arr-key", fixture)
		})
	}

	boFixtures := [][]BinaryObject{
		{createTestBinaryObject(suite.T(), suite.client, 10), createTestBinaryObject(suite.T(), suite.client, 10)},
		{createTestBinaryObject(suite.T(), suite.client, 10), createTestBinaryObject(suite.T(), suite.client, 20)},
		{createTestBinaryObject(suite.T(), suite.client, 10), createTestBinaryObject(suite.T(), suite.client, 20), nil},
	}

	for _, fixture := range boFixtures {
		suite.T().Run(fmt.Sprintf("ObjectArray from slice: %s", fixture), func(t *testing.T) {
			suite.putGetCollectionTest(suite.T(), "obj-arr-key", fixture)
		})
	}

}

func (suite *CacheTestSuite) TestSpecialArrays() {
	timestamp := time.Now().Truncate(0) // remove monotonic part.
	fixtures := []struct {
		name string
		arr  interface{}
	}{
		{"StringPArray", createPSlice("test1", "test2", "test3")},
		{"StringArray", []string{"test1", "test2", "test3"}},
		{"UuidPArray", createPSlice(uuid.New(), uuid.New(), uuid.New())},
		{"UuidArray", []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}},
		{"TimePArray", createPSlice(NewTime(timestamp), NewTime(timestamp.Add(time.Minute*10)), NewTime(timestamp.Add(time.Minute*20)))},
		{"TimeArray", []Time{NewTime(timestamp), NewTime(timestamp.Add(time.Minute * 10)), NewTime(timestamp.Add(time.Minute * 20))}},
		{"DatePArray", createPSlice(NewDate(timestamp), NewDate(timestamp.Add(time.Minute*10)), NewDate(timestamp.Add(time.Minute*20)))},
		{"DateArray", []Date{NewDate(timestamp), NewDate(timestamp.Add(time.Minute * 10)), NewDate(timestamp.Add(time.Minute * 20))}},
		{"TimeStampPArray", createPSlice(timestamp, timestamp.Add(time.Minute*10), timestamp.Add(time.Minute*20))},
		{"TimeStampArray", []time.Time{timestamp, timestamp.Add(time.Minute * 10), timestamp.Add(time.Minute * 20)}},
		{"DecimalPArray", []*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)}},
		{"DecimalArray", fromPSlice([]*apd.Decimal{apd.New(100500, -3), apd.New(0, 3), apd.New(31415926, 7)})},
	}

	for _, fixture := range fixtures {
		suite.T().Run(fmt.Sprintf("%s:%s", fixture.name, fixture.arr), func(t *testing.T) {
			suite.putGetCollectionTest(t, "special-arr-key", fixture.arr)
		})
	}
}

func (suite *CacheTestSuite) TestMaps() {
	fixtures := []struct {
		name string
		m    interface{}
	}{
		{"UserMap-fromKV", NewUserMap([]KeyValue{{"test1", int32(1)}, {"test2", int32(2)}, {int32(3), createTestBinaryObject(suite.T(), suite.client, 3)}}...)},
		{"HashMap-fromKV", NewHashMap([]KeyValue{{"test1", int32(1)}, {"test2", int32(2)}, {int32(3), createTestBinaryObject(suite.T(), suite.client, 3)}}...)},
		{"LinkedHashMap-fromKV", NewLinkedHashMap([]KeyValue{{"test1", int32(1)}, {"test2", int32(2)}, {int32(3), createTestBinaryObject(suite.T(), suite.client, 3)}}...)},
		{"UserMap-fromMap", ToUserMap(map[string]int32{"test1": 1, "test2": 2})},
		{"HashMap-fromMap", ToHashMap(map[string]int32{"test1": 1, "test2": 2})},
		{"LinkedHashMap-fromMap", ToLinkedHashMap(map[string]int32{"test1": 1, "test2": 2})},
		{"GoMap", map[string]int32{"test1": 1, "test2": 2}},
		{"GoMap-BinaryObject", map[string]BinaryObject{"test1": createTestBinaryObject(suite.T(), suite.client, 10), "test2": nil}},
	}

	for _, fixture := range fixtures {
		suite.T().Run(fmt.Sprintf("%s:%s", fixture.name, fixture.m), func(t *testing.T) {
			suite.putGetCollectionTest(t, "map-key", fixture.m)
		})
	}
}

func (suite *CacheTestSuite) TestIgniteMapToGoMapConversions() {
	goMap := map[string]BinaryObject{"test1": createTestBinaryObject(suite.T(), suite.client, 10), "test2": nil}
	transFunc := func(ignMap Map) map[string]BinaryObject {
		ret, err := ToMap[string, BinaryObject](ignMap)
		require.NoError(suite.T(), err)
		return ret
	}
	require.Equal(suite.T(), goMap, transFunc(ToUserMap(goMap)))
	require.Equal(suite.T(), goMap, transFunc(ToHashMap(goMap)))
	require.Equal(suite.T(), goMap, transFunc(ToLinkedHashMap(goMap)))
}

func (suite *CacheTestSuite) putGetCollectionTest(t *testing.T, key interface{}, val interface{}) {
	defer func() {
		_ = suite.cache.ClearAll(context.Background())
	}()
	sz, err := suite.cache.Size(context.Background())
	require.Nil(t, err)
	require.Equal(t, uint64(0), sz, "cache must be empty")

	ok, err := suite.cache.ContainsKey(context.Background(), key)
	require.Nil(t, err)
	require.False(t, ok)

	err = suite.cache.Put(context.Background(), key, val)
	require.Nil(t, err, "failed to put value", err)

	ok, err = suite.cache.ContainsKey(context.Background(), key)
	require.Nil(t, err)
	require.True(t, ok)

	actualVal, err := suite.cache.Get(context.Background(), key)
	require.NoError(t, err, "failed to get value", err)

	if coll, ok := val.(Collection); ok {
		if coll.Kind() == UserCollection {
			require.Equal(t, ArrayList, actualVal.(Collection).kind) // quirk of ignite serialization.
		} else {
			require.Equal(t, coll.Kind(), actualVal.(Collection).kind)
		}
		RequireArraysEqual(t, coll.values, actualVal.(Collection).values)
	} else if arr, ok := val.([]interface{}); ok {
		RequireArraysEqual(t, arr, actualVal.([]interface{}))
	} else if expectedMap, ok := val.(Map); ok {
		actualMap, ok := actualVal.(Map)
		require.True(t, ok)
		RequireMapEqual(t, expectedMap, actualMap)
	} else if reflect.ValueOf(val).Kind() == reflect.Map {
		expectedMap := reflect.ValueOf(val)
		actualMap, ok := actualVal.(Map)
		require.True(t, ok)
		RequireMapEqualGoMap(t, expectedMap, actualMap)
	} else if reflect.ValueOf(val).Kind() == reflect.Slice {
		expectedSlice := reflect.ValueOf(val)
		actualSlice := reflect.ValueOf(actualVal)
		RequireArraysReflectionEqual(t, expectedSlice, actualSlice)
	} else {
		require.FailNow(t, "unexpected type", reflect.TypeOf(val))
	}
}

func RequireMapEqualGoMap(t *testing.T, expected reflect.Value, actual Map) {
	require.Equal(t, HashMap, actual.Kind())
	require.Equal(t, expected.Len(), actual.Size())
	for _, actualEl := range actual.Entries() {
		expValue := expected.MapIndex(reflect.ValueOf(actualEl.Key)).Interface()
		require.True(t, ElementEqual(expValue, actualEl.Value),
			"elements are not equal", expValue, actualEl.Value, actualEl.Key)
	}
}

func RequireMapEqual(t *testing.T, expected Map, actual Map) {
	if expected.Kind() == UserCollection {
		require.Equal(t, HashMap, actual.Kind())
	} else {
		require.Equal(t, expected.Kind(), actual.Kind())
	}
loop:
	for _, el := range expected.Entries() {
		for _, actualEl := range actual.Entries() {
			if ElementEqual(el.Key, actualEl.Key) {
				require.True(t, ElementEqual(el.Value, actualEl.Value), "elements are not equal", el.Value, actualEl.Value)
				continue loop
			}
		}
		t.Errorf("failed to find element in returned collection %s", el)
		t.FailNow()
	}
}

func RequireArraysReflectionEqual(t *testing.T, expected reflect.Value, actual reflect.Value) {
	require.Equal(t, expected.Len(), actual.Len())
loop:
	for idx := 0; idx < expected.Len(); idx++ {
		el := getValue(expected.Index(idx))
		for idx0 := 0; idx0 < actual.Len(); idx0++ {
			actualVal := getValue(actual.Index(idx0))
			if ElementEqual(el, actualVal) {
				continue loop
			}
		}
		t.Errorf("failed to find element in returned collection %s", el)
		t.FailNow()
	}
}

func getValue(val reflect.Value) interface{} {
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		return val.Elem().Interface()
	}
	return val.Interface()
}

func RequireArraysEqual(t *testing.T, expected, actual []interface{}) {
loop:
	for _, el := range expected {
		for _, actualEl := range actual {
			if ElementEqual(el, actualEl) {
				continue loop
			}
		}
		t.Errorf("failed to find element in returned collection %s", el)
		t.FailNow()
	}
}

func ElementEqual(expected, actual interface{}) bool {
	expectedBO, ok := expected.(BinaryObject)
	if ok {
		if actualBO, ok := actual.(BinaryObject); ok {
			return expectedBO.HashCode() == actualBO.HashCode() && bytes.Equal(expectedBO.Data(), actualBO.Data())
		}
	}
	expectedKV, ok := expected.(KeyValue)
	if ok {
		if actualKV, ok := actual.(KeyValue); ok {
			return ElementEqual(expectedKV.Key, actualKV.Key) && ElementEqual(expectedKV.Value, actualKV.Value)
		}
	}
	return reflect.DeepEqual(expected, actual)
}

func createTestBinaryObject(t *testing.T, cli *Client, id int32) BinaryObject {
	ret, err := cli.CreateBinaryObject(context.Background(), "TEST_VALUE",
		WithField("id", id), WithField("name", fmt.Sprintf("name_%d", id)))
	require.NoError(t, err)
	return ret
}

func createPSlice[T any](vals ...T) []*T {
	ret := make([]*T, len(vals))
	for i, v := range vals {
		ret[i] = &v
	}
	return ret
}

func fromPSlice[T any](pSlice []*T) []T {
	ret := make([]T, len(pSlice))
	for i, v := range pSlice {
		if v != nil {
			ret[i] = *v
		}
	}
	return ret
}
