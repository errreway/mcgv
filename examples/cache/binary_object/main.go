//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"gitverse.ru/sbertech/ignite-go-client/examples"
)

// Automatically generates Binarylizable interface method implementations for all structs marked as `ignite:binarylizable`
// and writes generated code to `binarylizable.go` file.
// Tags can be used on struct fields to pass to generator Binary Object specific field options.
//`with_register_func` flag tells the generator to create a utility method that helps to registers all generated types by one call.
//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -output_filename=binarylizable.go -with_register_func

// ignite:binarylizable,typename="PERSON_KEY"
type BinarylizablePersonKey struct {
	identifier   int    `ignite:"id"`
	organization string `ignite:"organization_id,affinityKey"`
}

func (k *BinarylizablePersonKey) String() string {
	return fmt.Sprintf("BinarylizablePersonKey [identifier=`%d`, organization=`%s`]", k.identifier, k.organization)
}

// ignite:binarylizable,typename="EMPLOYEE"
type Employee struct {
	name   string
	salary int16
}

func (p *Employee) String() string {
	return fmt.Sprintf("Employee [name=`%s`, salary=`%d`]", p.name, p.salary)
}

func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}
	defer func() {
		_ = cli.Close(context.Background())
	}()
	examples.ActivateIgniteCluster(cli)

	cache, err := cli.GetOrCreateCache(context.Background(), "binary-object-example-cache")
	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	// Binary Object - special Ignite platform independent format for storing and accessing complex objects.
	// To write a complex object to a cache, you can create Binary Object manually or create Golang struct with
	// desired fields that implements Binarylizable interface.

	// Here is example of working with Ignite cache complex key and value created as Binarylizable objects.
	// Note, that custom types that implement Binarylizable interface must be registered before first use.
	err = RegisterMainIgniteTypes(cli) // Utility type registration method created by the generator.
	if err != nil {
		panic(fmt.Errorf("failed to register Binarylizable types: %w", err))
	}
	binarylizableKey := &BinarylizablePersonKey{identifier: 0, organization: "RCM"}
	binarylizableVal := &Employee{name: "Kim", salary: 1000}

	err = cache.Put(context.Background(), binarylizableKey, binarylizableVal)
	if err != nil {
		panic(fmt.Errorf("failed to write entry to the cache: %w", err))
	}
	fmt.Printf(">>> Entry [key=%v, val=%v] was written to cache `%s`\n", binarylizableKey, binarylizableVal, cache.Name())

	fmt.Printf(">>> Requesting value for key == %v from cache `%s`\n", binarylizableKey, cache.Name())
	obj, err := cache.Get(context.Background(), binarylizableKey)
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}
	fmt.Printf(">>> Value == %v\n", obj.(*Employee))

	// Here is example of working with Ignite cache complex key and value created manually as Binary Objects.
	key, err := cli.CreateBinaryObject(context.Background(), "PERSON_KEY",
		ignite.WithField("id", 1),
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
	fmt.Printf(">>> Entry [key=%s, val=%s] was written to cache `%s`\n", examples.ToString(key), examples.ToString(val), cache.Name())

	fmt.Printf(">>> Requesting value for key == %s from cache `%s`\n", examples.ToString(key), cache.Name())
	obj, err = cache.Get(context.Background(), key)
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

	// ScanField automatically checks and casts unmarshalled field value to the specified type.
	var ageField int16
	err = binObj.ScanField(context.Background(), "age", &ageField)
	if err != nil {
		panic(fmt.Errorf("failed to get Binary Object field [name=`age`]: %w", err))
	}

	fmt.Printf(">>> Value == BinaryObject [type=`%v`, name=`%v`, age=`%v`]\n", objType.TypeName(), nameField, ageField)

	// The same as Binary Object key that we created manually but defined via Binarylizable object.
	binarylizableKey = &BinarylizablePersonKey{identifier: 1, organization: "RCM"}

	// Here we are getting value of the entry that was previously written to the cache with a key created as Binary
	// Object manually. But now we are using BinarylizablePersonKey object as a key.
	fmt.Printf(">>> Requesting value for key == %v from cache `%s`\n", binarylizableKey, cache.Name())
	obj, err = cache.Get(context.Background(), binarylizableKey)
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}
	fmt.Printf(">>> Value == %v\n", examples.ToString(obj.(ignite.BinaryObject)))
}
