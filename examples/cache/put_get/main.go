//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
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
		ignite.CreateCacheConfiguration("example-cache",
			ignite.WithCacheAtomicityMode(ignite.AtomicAtomicityMode),
			ignite.WithCacheMode(ignite.ReplicatedCacheMode)))

	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	err = cache.Put(context.Background(), "key", "val")
	if err != nil {
		panic(fmt.Errorf("failed to write entry to the cache: %w", err))
	}
	fmt.Printf(">>> Entry [key=`key`, val=`val`] was written to cache `%s`\n", cache.Name())

	fmt.Printf(">>> Requesting value for key == 'key' from cache `%s`\n", cache.Name())
	val, err := cache.Get(context.Background(), "key")
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}
	fmt.Printf(">>> Value == `%s`\n", val)
}
