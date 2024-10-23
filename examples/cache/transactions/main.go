//go:build testing

package main

import (
	"context"
	"errors"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"gitverse.ru/sbertech/ignite-go-client/examples"
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
	examples.ActivateIgniteCluster(cli)

	// Note, transactions are only supported by caches with Transactional Atomicity mode.
	cache, err := cli.GetOrCreateCacheWithConfiguration(
		context.Background(),
		ignite.CreateCacheConfiguration("transactions-example-cache", ignite.WithCacheAtomicityMode(ignite.TransactionalAtomicityMode)))
	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Local transaction timeout.
	defer cancel()

	err = cli.RunInTransaction(ctx,
		func(txCtx context.Context) error {
			fmt.Println(">>> Transaction started")

			// Note that cache operations must reuse the transaction context in order to be run within a transaction
			// or to see previous cache entry updates made as part of the transaction itself.
			txErr := cache.Put(txCtx, "key", "val")
			if txErr != nil {
				return fmt.Errorf("failed to write entry to cache: %w", err)
			}
			fmt.Printf(">>> Entry [key=`key`, val=`val`] was written to cache `%s`\n", cache.Name())

			fmt.Printf(">>> Requesting value for key == 'key' from cache `%s`\n", cache.Name())
			val, txErr := cache.Get(txCtx, "key")
			if txErr != nil {
				return fmt.Errorf("failed to get value from the cache: %w", err)
			}
			fmt.Printf(">>> Value == `%s`\n", val)

			fmt.Println(">>> Commiting transaction")

			return nil // nil means - commit, otherwise - rollback.
		},
		ignite.WithConcurrency(ignite.PessimisticConcurrency),
		ignite.WithIsolationLevel(ignite.RepeatableReadLevel),
		ignite.WithTimeout(1*time.Second)) // Transaction timeout, which is handled on the Ignite cluster side.

	// err == nil means that tx was successfully committed.
	fmt.Printf(">>> Transaction completed [err=%v]\n", err)

	requestCacheValue(cache, "key")

	err = cli.RunInTransaction(ctx,
		func(txCtx context.Context) error {
			fmt.Println(">>> Transaction started")

			txErr := cache.Put(txCtx, "key", "updated-val")
			if txErr != nil {
				return fmt.Errorf("failed to write entry to cache: %w", err)
			}
			fmt.Printf(">>> Entry [key=`key`, val=`updated-value`] was written to cache `%s`\n", cache.Name())

			fmt.Printf(">>> Requesting value for key == 'key' from cache `%s`\n", cache.Name())
			val, txErr := cache.Get(txCtx, "key")
			if txErr != nil {
				return fmt.Errorf("failed to get value from the cache: %w", err)
			}
			fmt.Printf(">>> Value == `%s`]\n", val)

			fmt.Println(">>> Aborting transaction")
			return errors.New("transaction aborted") // Explicit rollback.
		})

	// err != nil means that the transaction was aborted or even failed to start for any reasons.
	fmt.Printf(">>> Transaction completed [err=%s]\n", fmt.Errorf("%w", err))

	requestCacheValue(cache, "key")
}

func requestCacheValue(cache *ignite.Cache, key interface{}) {
	fmt.Printf(">>> Requesting value for key == 'key' from cache `%s`\n", cache.Name())
	val, err := cache.Get(context.Background(), key)
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}
	fmt.Printf(">>> Value == `%s`\n", val)
}
