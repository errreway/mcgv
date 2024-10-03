//go:build testing

package test_binarylizable

//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -with_register_func

// ignite:binarylizable
type PrimitiveArrays struct {
	boolArray    []bool
	byteArray    []byte
	sByteArray   []int8
	uByteArray   []uint8
	shortArray   []int16
	charArray    []uint16
	int32Array   []int32
	uint32Array  []uint32
	int64Array   []int64
	uint64Array  []uint64
	intArray     []int
	uintArray    []uint
	float32Array []float32
	float64Array []float64
}
