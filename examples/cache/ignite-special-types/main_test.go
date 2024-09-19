package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestIgniteSpecialTypesExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry for key == `keyIgniteUserCollection` was written to cache `example-cache`\n"+
			">>> Requesting value for key == `keyIgniteUserCollection` from cache `example-cache`\n"+
			">>> Value == [igniteCollectionKind=1, arr=[element]]\n"+
			">>> Entry for key == `keyIgniteUserMap` was written to cache `example-cache`\n"+
			">>> Requesting value for key == `keyIgniteUserMap` from cache `example-cache`\n"+
			">>> Value == [igniteMapKind=1, map=map[mapKey:mapVal]]\n"+
			">>> Entry for key == `keyIgniteTime` was written to cache `example-cache`\n"+
			">>> Requesting value for key == `keyIgniteTime` from cache `example-cache`\n"+
			">>> Value == `2009-07-02 04:17:46 +0000 UTC`\n"+
			">>> Entry for key == `keyIgniteDate` was written to cache `example-cache`\n"+
			">>> Requesting value for key == `keyIgniteDate` from cache `example-cache`\n"+
			">>> Value == `2009-07-02 04:17:46 +0000 UTC`\n")
}
