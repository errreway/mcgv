package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestScanQueryExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Entry [key=`1`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-1`, age=`51`]] was written to cache `example-cache`\n"+
			">>> Entry [key=`2`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-2`, age=`52`]] was written to cache `example-cache`\n"+
			">>> Entry [key=`3`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-3`, age=`53`]] was written to cache `example-cache`\n"+
			">>> Entry [key=`4`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-4`, age=`54`]] was written to cache `example-cache`\n"+
			">>> Entry [key=`5`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-5`, age=`55`]] was written to cache `example-cache`\n"+
			">>> Executing Scan Query [pageSize=2, isLoc=true] for cache `example-cache`\n"+
			">>> Scan Query cursor returned value [key=`1`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-1`, age=`51`]]\n"+
			">>> Scan Query cursor returned value [key=`2`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-2`, age=`52`]]\n"+
			">>> Scan Query cursor returned value [key=`3`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-3`, age=`53`]]\n"+
			">>> Scan Query cursor returned value [key=`4`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-4`, age=`54`]]\n"+
			">>> Scan Query cursor returned value [key=`5`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-5`, age=`55`]]\n"+
			">>> Executing Scan Query [filter = FilterByFieldName[name=Filippe-3]] for cache `example-cache`\n"+
			">>> Scan Query cursor returned value [key=`3`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-3`, age=`53`]]\n"+
			">>> Executing Scan Query [filter=FilterByFieldName[name=Filippe-2], keepBinary=true] for cache `example-cache`\n"+
			">>> Scan Query cursor returned value [key=`2`, val=BinaryObject [type=`ru.gitverse.sbertech.client.filters.Person`, name=`Filippe-2`, age=`52`]]\n")
}
