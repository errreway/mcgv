package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestWalStateExample(t *testing.T) {
	igniteTesting.TestExampleWithRegularExpression(t, []func(params *igniteTesting.IgniteParams){igniteTesting.WithPersistence()}, main,
		">>> Ignite cluster state was changed to `Active`\n"+
			">>> Disabling WAL state for cache `wal-state-example-cache`\n"+
			">>> WAL was disable for cache `wal-state-example-cache`\n"+
			">>> Requesting whether WAL is enabled for cache `wal-state-example-cache`\n"+
			">>> Is WAL enabled for cache `wal-state-example-cache` == `false`\n"+
			">>> Enabling WAL state for cache `wal-state-example-cache`\n"+
			">>> WAL was enabled for cache `wal-state-example-cache`\n"+
			">>> Requesting whether WAL is enabled for cache `wal-state-example-cache`\n"+
			">>> Is WAL enabled for cache `wal-state-example-cache` == `true`\n")
}
