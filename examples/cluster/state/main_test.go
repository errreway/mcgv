package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestClusterStateExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Changing state of the Ignite cluster to `Active`\n"+
			">>> State of the Ignite cluster changed to `Active`\n"+
			">>> Requesting state of the Ignite cluster\n"+
			">>> Ignite cluster state == `Active`\n")
}
