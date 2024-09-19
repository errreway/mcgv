package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestTransactionExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Transaction started\n"+
			">>> Entry [key=`key`, val=`val`] was written to cache `example-cache`\n"+
			">>> Requesting value for key == 'key' from cache `example-cache`\n"+
			">>> Value == `val`\n"+
			">>> Commiting transaction\n"+
			">>> Transaction completed [err=<nil>]\n"+
			">>> Requesting value for key == 'key' from cache `example-cache`\n"+
			">>> Value == `val`\n"+
			">>> Transaction started\n"+
			">>> Entry [key=`key`, val=`updated-value`] was written to cache `example-cache`\n"+
			">>> Requesting value for key == 'key' from cache `example-cache`\n"+
			">>> Value == `updated-val`]\n"+
			">>> Aborting transaction\n"+
			">>> Transaction completed [err=transaction aborted]\n"+
			">>> Requesting value for key == 'key' from cache `example-cache`\n"+
			">>> Value == `val`\n")
}
