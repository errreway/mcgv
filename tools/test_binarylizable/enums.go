package test_binarylizable

//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -with_register_func

// ignite:enum,typename="Simple$Enum"
type SimpleEnum int8

// ignite:enum,typename="Skipped$Enum"
type SkippedEnum string

const (
	SimpleEnumVal1  SimpleEnum  = iota // ignite:enum,name="VAL1"
	SimpleEnumVal2                     // ignite:enum,name="VAL2"
	BlaBla          int         = iota
	SimpleEnumVal3  SimpleEnum  = iota + 5 // ignite:enum,name="VAL3"
	SkippedEnumVal1 SkippedEnum = "bla"
	SimpleEnumVal4  SimpleEnum  = iota
	SkippedEnumVal2 SkippedEnum = "bla"
)

// ignite:enum,typename="ru.gitverse.sbertech.client.filters.TestEnum$Enum"
type Enum int

const (
	EnumVal1 Enum = iota // ignite:enum,name="VAL1"
	EnumVal2             // ignite:enum,name="VAL2"
	EnumVal3             // ignite:enum,name="VAL3"
)
