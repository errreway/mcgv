//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"gitverse.ru/sbertech/ignite-go-client/examples"
	"time"
)

// Automatically generates Enum interface method implementations for all types marked as `ignite:enum`
// and writes generated code to `ranks_enum.go` file. Note that const variables that represents enum elements must be
// marked with ignite:enum as well.
//`with_register_func` flag tells the generator to create a utility method that helps to registers all generated enums by one call.
//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -output_filename=ranks_enum.go -with_register_func

//go:generate go run golang.org/x/tools/cmd/stringer -tags=testing -type=Rank

// ignite:enum,typename="RANK"
type Rank int

const (
	Private    Rank = iota // ignite:enum,name="PRIVATE"
	Lieutenant             // ignite:enum,name="LIEUTENANT"
)

func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}
	defer func() {
		_ = cli.Close(context.Background())
	}()
	examples.ActivateIgniteCluster(cli)

	cache, err := cli.GetOrCreateCacheWithConfiguration(
		context.Background(),
		ignite.CreateCacheConfiguration("ignite-special-types-example-cache"))
	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	// Apache Ignite can store entries which type is related to specific Java build-in data structures that have no
	// equivalent in Golang. Ignite Go client provides representations of Java-specific types that allow easy conversion
	// to Golang data structures and vice versa.
	// Here is examples of usage of the mentioned above structures.

	// Enums.
	// Working with Enums is based on the same approaches as with Binary Objects.
	// Enum elements can be created manually or by using a struct that implements Enum interface.
	// In either way enum type must be registered before first use.
	err = RegisterRank(cli) // Utility enum registration method created by the generator.
	if err != nil {
		panic(fmt.Errorf("failed to register enum types: %w", err))
	}
	rank := doPutGet(cache, "keyEnumObject", Private)
	fmt.Printf(">>> Value == %v\n", rank)

	err = cli.RegisterEnumMetadata(context.Background(), "RCM_RANKS", map[string]int{"CORPORAL": 0, "MAYOR": 1}) // Manual enum type registration.
	if err != nil {
		panic(fmt.Errorf("failed to register enum types: %w", err))
	}
	binEnumElem, err := cli.CreateBinaryObject(context.Background(), "RCM_RANKS", ignite.WithEnumOrdinal(1)) // Note that the enum type name used was previously registered.
	if err != nil {
		panic(fmt.Errorf("failed to create binary enum element: %w", err))
	}
	obj := doPutGet(cache, "keyEnumBinaryObject", binEnumElem)
	binObj := obj.(ignite.BinaryObject)
	fmt.Printf(">>> Value == %v\n", examples.ToString(binObj))

	// Collections and Maps.
	m := make(map[string]string)
	m["key"] = "val"
	_ = ignite.ToUserMap(m)

	igniteUserCollection := ignite.NewUserCollection("element")
	igniteUserMap := ignite.NewUserMap([]ignite.KeyValue{{Key: "mapKey", Value: "mapVal"}}...)
	_ = ignite.NewHashMap([]ignite.KeyValue{{Key: "mapKey", Value: "mapVal"}}...)
	_ = ignite.NewSingletonList("element")
	_ = ignite.NewArrayList("element")
	_ = ignite.NewLinkedList("element")
	_ = ignite.NewHashSet("element")
	_ = ignite.NewLinkedHashSet("element")

	obj = doPutGet(cache, "keyIgniteUserCollection", igniteUserCollection)
	igniteColObj := obj.(ignite.Collection)
	goSlice, err := ignite.ToSlice[string](igniteColObj)
	if err != nil {
		panic(fmt.Errorf("failed to convert ignite.Collection to slice: %w", err))
	}
	fmt.Printf(">>> Value == [igniteCollectionKind=%v, arr=%v]\n", igniteColObj.Kind(), goSlice)

	obj = doPutGet(cache, "keyIgniteUserMap", igniteUserMap)
	igniteMapObj := obj.(ignite.Map)
	goMap, err := ignite.ToMap[string, string](igniteMapObj)
	if err != nil {
		panic(fmt.Errorf("failed to convert ignite.Map to map: %w", err))
	}
	fmt.Printf(">>> Value == [igniteMapKind=%v, map=%v]\n", igniteMapObj.Kind(), goMap)

	// Date and Time.
	igniteDate := ignite.NewDate(time.Unix(1246508266, 0))
	igniteTime := ignite.NewTime(time.Unix(1246508266, 0))
	obj = doPutGet(cache, "keyIgniteTime", igniteTime)
	igniteTimeObj := obj.(ignite.Time)
	fmt.Printf(">>> Value == `%s`\n", igniteTimeObj.Time().UTC().Format("15:04:05"))

	obj = doPutGet(cache, "keyIgniteDate", igniteDate)
	igniteDateObj := obj.(ignite.Date)
	fmt.Printf(">>> Value == `%v`\n", igniteDateObj.Time().UTC())
}

func doPutGet(cache *ignite.Cache, key string, val interface{}) interface{} {
	err := cache.Put(context.Background(), key, val)
	if err != nil {
		panic(fmt.Errorf("failed to write entry to the cache: %w", err))
	}
	fmt.Printf(">>> Entry for key == `%s` was written to cache `%s`\n", key, cache.Name())

	fmt.Printf(">>> Requesting value for key == `%s` from cache `%s`\n", key, cache.Name())
	obj, err := cache.Get(context.Background(), key)
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}

	return obj
}
