package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestExpiryExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry [key=`key`, val=`val`] was written to cache `expiry-example-cache` with TTL [type=creation, duration=1s]\n"+
			">>> Requesting value for key == `key` from cache `expiry-example-cache`\n"+
			">>> Value == `val`\n"+
			">>> Waiting for 2 seconds...\n"+
			">>> Checking whether the key == `key` is in the cache `expiry-example-cache`\n"+
			">>> Result == false\n")
}
