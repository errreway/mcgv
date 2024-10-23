package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestBinaryObjectExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry [key=BinarylizablePersonKey [identifier=`0`, organization=`RCM`], val=Employee [name=`Kim`, salary=`1000`]] was written to cache `binary-object-example-cache`\n"+
			">>> Requesting value for key == BinarylizablePersonKey [identifier=`0`, organization=`RCM`] from cache `binary-object-example-cache`\n"+
			">>> Value == Employee [name=`Kim`, salary=`1000`]\n"+
			">>> Entry [key=BinaryObject [type=`PERSON_KEY`, id=`1`, organization_id=`RCM`], val=BinaryObject [type=`PERSON`, name=`Cuno`, age=`12`]] was written to cache `binary-object-example-cache`\n"+
			">>> Requesting value for key == BinaryObject [type=`PERSON_KEY`, id=`1`, organization_id=`RCM`] from cache `binary-object-example-cache`\n"+
			">>> Value == BinaryObject [type=`PERSON`, name=`Cuno`, age=`12`]\n"+
			">>> Requesting value for key == BinarylizablePersonKey [identifier=`1`, organization=`RCM`] from cache `binary-object-example-cache`\n"+
			">>> Value == BinaryObject [type=`PERSON`, name=`Cuno`, age=`12`]\n")
}
