package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestClusterGroupExample(t *testing.T) {
	igniteTesting.TestExampleWithRegularExpression(t, nil, main,
		">>> Requesting nodes that are present in cluster topology and meet the Cluster Group criteria \\[`ForServers`, `ForOldest`\\]\\n"+
			">>> Received ClusterNode \\[id=[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}, consistentId=srv_0, addresses=\\[127\\.0\\.0\\.1\\], hostNames=\\[\\], order=1, version=[0-9]+.[0-9]+.[0-9]+#[0-9]{8}-sha1:[0-9a-f]{8}, isClient=false, isLocal=true\\]\\n")
}
