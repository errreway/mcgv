package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestPutGetExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry [key=`key`, val=`val`] was written to cache `example-cache`\n"+
			">>> Requesting value for key == 'key' from cache `example-cache`\n"+
			">>> Value == `val`\n")
}
