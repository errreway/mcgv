package test_binarylizable

//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -with_register_func

// ignite:enum
type Enum1 int

const (
	Enum1Val1 Enum1 = iota // ignite:enum,name="VAL1"
	Enum2Val2              // ignite:enum,name="VAL2"
	Enum3Val3              // ignite:enum,name="VAL3"
)

// ignite:binarylizable,typename="ru.gitverse.sbertech.client.filters.TestEnum"
type TestEnum struct {
	id             int
	enumField      Enum
	enumArrayField []Enum
}
