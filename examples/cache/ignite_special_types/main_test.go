package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestIgniteSpecialTypesExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry for key == `keyEnumObject` was written to cache `ignite-special-types-example-cache`\n"+
			">>> Requesting value for key == `keyEnumObject` from cache `ignite-special-types-example-cache`\n"+
			">>> Value == Private\n"+
			">>> Entry for key == `keyEnumBinaryObject` was written to cache `ignite-special-types-example-cache`\n"+
			">>> Requesting value for key == `keyEnumBinaryObject` from cache `ignite-special-types-example-cache`\n"+
			">>> Value == BinaryEnum [type=`RCM_RANKS`, ordinal=1, name=MAYOR]\n"+
			">>> Entry for key == `keyIgniteUserCollection` was written to cache `ignite-special-types-example-cache`\n"+
			">>> Requesting value for key == `keyIgniteUserCollection` from cache `ignite-special-types-example-cache`\n"+
			">>> Value == [igniteCollectionKind=ArrayList, arr=[element]]\n"+
			">>> Entry for key == `keyIgniteUserMap` was written to cache `ignite-special-types-example-cache`\n"+
			">>> Requesting value for key == `keyIgniteUserMap` from cache `ignite-special-types-example-cache`\n"+
			">>> Value == [igniteMapKind=HashMap, map=map[mapKey:mapVal]]\n"+
			">>> Entry for key == `keyIgniteTime` was written to cache `ignite-special-types-example-cache`\n"+
			">>> Requesting value for key == `keyIgniteTime` from cache `ignite-special-types-example-cache`\n"+
			">>> Value == `04:17:46`\n"+
			">>> Entry for key == `keyIgniteDate` was written to cache `ignite-special-types-example-cache`\n"+
			">>> Requesting value for key == `keyIgniteDate` from cache `ignite-special-types-example-cache`\n"+
			">>> Value == `2009-07-02 04:17:46 +0000 UTC`\n")
}
