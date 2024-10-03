//go:build testing

package test_binarylizable

import (
	"context"
	"github.com/cockroachdb/apd/v3"
	"github.com/stretchr/testify/require"
	"gitverse.ru/sbertech/ignite-go-client"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"gitverse.ru/sbertech/ignite-go-client/logger"
	"log"
	"math/rand"
	"os"
	"testing"
)

const testArrSize = 1 << 11

var defaultAddress = "127.0.0.1"

func startTestClient(ctx context.Context, opts ...ignite.ClientConfigurationOption) (*ignite.Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	opts = append(opts, ignite.WithAddresses(defaultAddress))
	sink, _ := logger.NewSink(log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds), logger.DebugLevel) // level is ok, error can be ignored.
	opts = append(opts, ignite.WithLoggingSink(sink))
	return ignite.Start(ctx, opts...)
}

func createSimpleStruct() *SimpleStruct {
	return &SimpleStruct{
		id:   rand.Int(),
		name: testing2.MakeRandomString(100),
	}
}

func createSimpleKey() *SimpleKey {
	return &SimpleKey{
		id:    rand.Int(),
		orgId: rand.Int(),
	}
}

func createBinaryObject(t *testing.T, cli *ignite.Client) ignite.BinaryObject {
	bo, err := cli.CreateBinaryObject(context.Background(), "SIMPLE_BINARY_OBJECT",
		ignite.WithField("id", rand.Int63()), ignite.WithField("name", testing2.MakeRandomString(100)))
	require.NoError(t, err)
	return bo
}

func checkCollectionsEqual(t *testing.T, coll0, coll1 ignite.Collection) {
	require.Equal(t, coll0.Kind(), coll1.Kind())
	require.Equal(t, coll0.Size(), coll1.Size())
	if coll0.Size() > 0 {
		require.False(t, coll0.IsNull())
	}
	if coll1.Size() > 0 {
		require.False(t, coll1.IsNull())
	}
	require.Equal(t, coll0.IsNull(), coll1.IsNull())
	checkObjectArraysEqual(t, coll0.Values(), coll1.Values())
}

func checkMapsEqual(t *testing.T, map0, map1 ignite.Map) {
	require.Equal(t, map0.Kind(), map1.Kind())
	require.Equal(t, map0.Size(), map1.Size())
	if map0.Size() > 0 {
		require.False(t, map0.IsNull())
	}
	if map1.Size() > 0 {
		require.False(t, map1.IsNull())
	}
	require.Equal(t, map0.IsNull(), map1.IsNull())
	checkObjectArraysEqual(t, map0.Entries(), map1.Entries())
}

func checkObjectArraysEqual[T any](t *testing.T, arr0, arr1 []T) {
	require.Equal(t, len(arr0), len(arr1))
	for i, el0 := range arr0 {
		checkElementsEqual(t, el0, arr1[i])
	}
}

func checkElementsEqual(t *testing.T, el0, el1 interface{}) {
	switch realEl0 := el0.(type) {
	case ignite.BinaryObject:
		realEl1, ok := el1.(ignite.BinaryObject)
		require.True(t, ok)
		checkBinaryObjectsEqual(t, realEl0, realEl1)
	case ignite.KeyValue:
		realEl1, ok := el1.(ignite.KeyValue)
		require.True(t, ok)
		checkElementsEqual(t, realEl0.Key, realEl1.Key)
		checkElementsEqual(t, realEl0.Value, realEl1.Value)
	case *apd.Decimal:
		switch realEl1 := el1.(type) {
		case apd.Decimal:
			require.Equal(t, realEl0, &realEl1)
		default:
			require.Equal(t, realEl0, realEl1)
		}
	case apd.Decimal:
		switch realEl1 := el1.(type) {
		case *apd.Decimal:
			require.Equal(t, &realEl0, realEl1)
		default:
			require.Equal(t, el0, el1)
		}
	case ignite.Collection:
		switch realEl1 := el1.(type) {
		case ignite.Collection:
			checkCollectionsEqual(t, realEl0, realEl1)
		default:
			require.Equal(t, el0, realEl1)
		}
	case ignite.Map:
		switch realEl1 := el1.(type) {
		case ignite.Map:
			checkMapsEqual(t, realEl0, realEl1)
		default:
			require.Equal(t, el0, realEl1)
		}
	default:
		require.Equal(t, el0, el1)
	}
}

func checkBinaryObjectArraysEqual(t *testing.T, arr0, arr1 []ignite.BinaryObject) {
	require.Equal(t, len(arr0), len(arr1))
	for i, el0 := range arr0 {
		checkElementsEqual(t, el0, arr1[i])
	}
}

func checkBinaryObjectsEqual(t *testing.T, obj0 ignite.BinaryObject, obj1 ignite.BinaryObject) {
	require.Equal(t, obj0.HashCode(), obj1.HashCode())
	require.Equal(t, obj0.Data(), obj1.Data())
	ctx := context.Background()

	type1, err := obj0.Type(ctx)
	require.NoError(t, err)
	require.NotNil(t, type1)
	type2, err := obj1.Type(ctx)
	require.NoError(t, err)
	require.NotNil(t, type2)

	require.Equal(t, type1.TypeId(), type2.TypeId())
	require.Equal(t, type1.TypeName(), type2.TypeName())
	require.Equal(t, type1.AffinityKeyName(), type2.AffinityKeyName())
	require.Equal(t, type1.IsEnum(), type2.IsEnum())
	require.Equal(t, type1.Fields(), type2.Fields())

	for _, fldName := range type1.Fields() {
		fld1, err := obj0.Field(ctx, fldName)
		require.NoError(t, err)
		fld2, err := obj1.Field(ctx, fldName)
		require.NoError(t, err)
		checkElementsEqual(t, fld1, fld2)
	}
}

func checkGoMapsEqual[K comparable, V any](t *testing.T, map0, map1 map[K]V) {
	require.Equal(t, len(map0), len(map1))
	if map0 != nil && map1 != nil {
		for k0, v0 := range map0 {
			v1, ok := map1[k0]
			require.True(t, ok)
			checkElementsEqual(t, v0, v1)
		}
	}
}

func createRandPrimitiveArray[T testing2.Primitives](randFactory func() T) []T {
	return testing2.MakeRandomPrimitiveArray[T](testArrSize, randFactory)
}
