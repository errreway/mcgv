//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"time"
)

func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}

	defer func() {
		_ = cli.Close(context.Background())
	}()

	cache, err := cli.GetOrCreateCacheWithConfiguration(
		context.Background(),
		ignite.CreateCacheConfiguration("example-cache"))
	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	// Apache Ignite can store entries which type is related to specific Java build-in data structures that have no
	// equivalent in Golang. Ignite Go client provides representations of Java-specific types that allow easy conversion
	// to Golang data structures and vice versa.
	// Here is examples of usage of the mentioned above structures.
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
	igniteDate := ignite.NewDate(time.Unix(1246508266, 0))
	igniteTime := ignite.NewTime(time.Unix(1246508266, 0))

	obj := doPutGet(cache, "keyIgniteUserCollection", igniteUserCollection)
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
