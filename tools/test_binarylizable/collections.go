//go:build testing

package test_binarylizable

import (
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	ign "gitverse.ru/sbertech/ignite-go-client"
)

//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -with_register_func

// ignite:binarylizable
type ObjectArrays struct {
	ifaceArray        []interface{}
	boArray           []ign.BinaryObject
	binArray          []ign.Binarylizable
	simpleStructArray []*SimpleStruct
}

// ignite:binarylizable
type Collections struct {
	list      ign.Collection
	ignMap    ign.Map
	simpleMap map[string]string
	boMap     map[string]ign.BinaryObject
	biMap     map[string]ign.Binarylizable
	decMap    map[apd.Decimal]*SimpleStruct
	iMap      map[string]interface{}
	uuidMap   map[uuid.UUID]string
	skipMap0  map[ign.Binarylizable]string
	skipMap1  map[SimpleKey]string
	skipMap2  map[*SimpleKey]string
	skipMap3  map[ign.BinaryObject]string
	skipMap4  map[*apd.Decimal]string
}

// ignite:binarylizable,typename="KEY"
type SimpleKey struct {
	id    int
	orgId int `ignite:"affinityKey"`
}

// ignite:binarylizable,typename="VALUE"
type SimpleStruct struct {
	id   int
	name string
}

// ignite:binarylizable,typename="COMPLEX_KEY"
type ComplexKey struct {
	id    int `ignite:"id"`
	orgId int `ignite:"org_id,affinityKey"`
}

// ignite:binarylizable,typename="COMPLEX_VALUE"
type ComplexStruct struct {
	name         string           `ignite:"name"`
	simple       *SimpleStruct    `ignite:"simple_struct"`
	binaryObject ign.BinaryObject `ignite:"binary_object"`
}

// ignite:binarylizable,typename="DOT_NET_VALUE",platform="dotnet"
type DotNetStruct struct {
	name string
}
