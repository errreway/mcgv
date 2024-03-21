//go:build testing

package benchmarks

import (
	"context"
	"gitverse.ru/sbertech/ignite-go-client"
	"os"
	"strconv"
	"testing"
)

const EnvWarmupCount = "WARMUPS"

func CacheBenchmarker(b *testing.B, cliCreate func() (ignite.Client, ignite.Cache), fixture func(c ignite.Cache), f func(b *testing.B, c ignite.Cache)) {
	warmups := warmupCount()
	if warmups > 0 {
		b.Logf("Warmups: %d", warmups)
	}
	runner := func(b *testing.B, smart bool) {
		cli, cache := cliCreate()
		defer func() {
			if err := cli.Close(); err != nil {
				b.Log("Test warning, client not shutdown", err)
			}
		}()
		if fixture != nil {
			fixture(cache)
		}
		b.ResetTimer()
		f(b, cache)
	}
	warmJvmUp := func() {
		client, cache := cliCreate()
		defer func() {
			_ = client.Close()
		}()
		for i := 0; i < warmups; i++ {
			f(b, cache)
		}
		if err := cache.RemoveAll(context.Background()); err != nil {
			b.Error("failed to remove all data")
		}
	}
	warmJvmUp()
	runner(b, true)
}

func warmupCount() int {
	if s := os.Getenv(EnvWarmupCount); s != "" {
		if i, err := strconv.ParseInt(s, 10, 32); err != nil {
			panic(err)
		} else {
			return int(i)
		}
	}
	return 3
}
