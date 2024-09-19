//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"gitverse.ru/sbertech/ignite-go-client/examples"
)

func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}

	defer func() {
		_ = cli.Close(context.Background())
	}()

	cache, err := cli.GetOrCreateCache(context.Background(), "example-cache")

	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	// Binary Object - special Ignite platform independent format for storing and accessing complex objects.
	key, err := cli.CreateBinaryObject(context.Background(), "PERSON_KEY",
		ignite.WithField("id", 0),
		ignite.WithField("organization_id", "RCM"),
		ignite.WithAffinityKeyName("organization_id"))
	if err != nil {
		panic(fmt.Errorf("failed to create binary object: %w", err))
	}

	val, err := cli.CreateBinaryObject(context.Background(), "PERSON",
		ignite.WithField("name", "Cuno"), // Note that field value itself can also be a Binary Object.
		ignite.WithField("age", int16(12)))
	if err != nil {
		panic(fmt.Errorf("failed to create binary object: %w", err))
	}

	err = cache.Put(context.Background(), key, val)
	if err != nil {
		panic(fmt.Errorf("failed to write entry to the cache: %w", err))
	}
	fmt.Printf(">>> Entry [key=%s, val=%s] was written to cache `example-cache`\n", examples.ToString(key), examples.ToString(val))

	fmt.Printf(">>> Requesting value for key == %s from cache `example-cache`\n", examples.ToString(key))
	obj, err := cache.Get(context.Background(), key)
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}
	binObj := obj.(ignite.BinaryObject)

	objType, err := binObj.Type(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to get Binary Object type: %w", err))
	}

	// Note, client does not perform Binary Object unmarshalling when receiving it. Binary Object stores raw data and
	// does unmarshalling when a particular field is accessed.
	nameField, err := binObj.Field(context.Background(), "name")
	if err != nil {
		panic(fmt.Errorf("failed to get Binary Object field [name=`name`]: %w", err))
	}

	// Field Getter automatically checks and casts unmarshalled field value to the specified type.
	var ageField int16
	err = ignite.NewFieldGetter[int16](binObj, "age").Get(context.Background(), &ageField)
	if err != nil {
		panic(fmt.Errorf("failed to get Binary Object field [name=`age`]: %w", err))
	}

	fmt.Printf(">>> Value == BinaryObject [type=%v, name=%v, age=%v]\n", objType.TypeName(), nameField, ageField)
}
