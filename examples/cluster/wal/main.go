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
	examples.ActivateIgniteCluster(cli)

	cache, err := cli.GetOrCreateCacheWithConfiguration(
		context.Background(),
		ignite.CreateCacheConfiguration("wal-state-example-cache"))
	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	// Note, that WAL state can be changed only for persistent caches.
	fmt.Printf(">>> Disabling WAL state for cache `%s`\n", cache.Name())
	isWalStateChanged, err := cli.DisableWal(context.Background(), cache.Name())
	if err != nil {
		panic(fmt.Errorf("failed to change WAL state for the cache: %w", err))
	}
	if isWalStateChanged {
		fmt.Printf(">>> WAL was disable for cache `%s`\n", cache.Name())
	}
	requestCacheWalState(cli, cache.Name())

	fmt.Printf(">>> Enabling WAL state for cache `%s`\n", cache.Name())
	isWalStateChanged, err = cli.EnableWal(context.Background(), cache.Name())
	if err != nil {
		panic(fmt.Errorf("failed to change WAL state for the cache: %w", err))
	}
	if isWalStateChanged {
		fmt.Printf(">>> WAL was enabled for cache `%s`\n", cache.Name())
	}
	requestCacheWalState(cli, cache.Name())
}

func requestCacheWalState(cli *ignite.Client, cacheName string) {
	fmt.Printf(">>> Requesting whether WAL is enabled for cache `%s`\n", cacheName)
	isWalEnabled, err := cli.IsWalEnabled(context.Background(), cacheName)
	if err != nil {
		panic(fmt.Errorf("failed to request WAL state for the cache: %w", err))
	}
	fmt.Printf(">>> Is WAL enabled for cache `%s` == `%t`\n", cacheName, isWalEnabled)
}
