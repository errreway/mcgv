package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestBinaryObjectExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry [key=BinaryObject [type=`PERSON_KEY`, id=`0`, organization_id=`RCM`], val=BinaryObject [type=`PERSON`, name=`Cuno`, age=`12`]] was written to cache `example-cache`\n"+
			">>> Requesting value for key == BinaryObject [type=`PERSON_KEY`, id=`0`, organization_id=`RCM`] from cache `example-cache`\n"+
			">>> Value == BinaryObject [type=PERSON, name=Cuno, age=12]\n")
}
