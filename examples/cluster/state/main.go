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

	fmt.Printf(">>> Changing state of the Ignite cluster to `%v`\n", ignite.Active)
	err = cli.ChangeClusterState(context.Background(), ignite.Active)
	if err != nil {
		panic(fmt.Errorf("failed to change state of the Ignite cluster: %w", err))
	}
	fmt.Printf(">>> State of the Ignite cluster changed to `%v`\n", ignite.Active)

	fmt.Println(">>> Requesting state of the Ignite cluster")
	state, err := cli.GetClusterState(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to request cluster state: %w", err))
	}
	fmt.Printf(">>> Ignite cluster state == `%s`\n", state)
}
