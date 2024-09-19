//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"time"
)

const DurationZero = time.Duration(0)

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

	// Note that current example uses a TTL that is tied to the time the cache entry was created.
	// Expiry Policy can also be configured based on access/update time of the cache entry.
	cache = cache.WithExpiryPolicy(1*time.Second, DurationZero, DurationZero)

	err = cache.Put(context.Background(), "key", "val")
	if err != nil {
		panic(fmt.Errorf("failed to write entry to the cache: %w", err))
	}
	fmt.Println(">>> Entry [key=`key`, val=`val`] was written to cache `example-cache` with TTL [type=creation, duration=1s]")

	fmt.Println(">>> Requesting value for key == `key` from cache `example-cache`")
	val, err := cache.Get(context.Background(), "key")
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}
	fmt.Printf(">>> Value == `%s`\n", val)

	fmt.Println(">>> Waiting for 2 seconds...")
	<-time.After(2 * time.Second)

	// Cache entry that we stored previously with TTL=1s should have expired by now.
	fmt.Println(">>> Checking whether the key == `key` is in the cache `example-cache`")
	isPresent, err := cache.ContainsKey(context.Background(), "key")
	if err != nil {
		panic(fmt.Errorf("failed to check for key presence: %w", err))
	}

	fmt.Printf(">>> Result == %t", isPresent)
}
