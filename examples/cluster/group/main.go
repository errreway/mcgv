//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
)

func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}
	defer func() {
		_ = cli.Close(context.Background())
	}()

	// Cluster group can be considered as a composite filter that is sequentially applied whenever Ignite cluster nodes
	// are requested. You can use set of predefined filters or create a custom one using `ForPredicate`. It is acceptable
	// if filters are not used at all - in this case the cluster Group will be mapped to the full Ignite
	// topology, including server and client nodes.
	// The following example uses two filters - `ForServer` and `ForOldest`, which filter out all nodes except one, which
	// in Ignite terminology is called as `coordinator`.
	group, err := cli.GetClusterGroup(ignite.ForServers(), ignite.ForOldest())
	if err != nil {
		panic(fmt.Errorf("failed to get cluster group: %w", err))
	}

	fmt.Println(">>> Requesting nodes that are present in cluster topology and meet the Cluster Group criteria [`ForServers`, `ForOldest`]")
	nodes, err := group.Nodes(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to change state of the Ignite cluster: %w", err))
	}
	for _, node := range nodes {
		fmt.Printf(">>> Received %v\n", node)
	}
}
