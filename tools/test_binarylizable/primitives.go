//go:build testing

package test_binarylizable

//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -with_register_func

// ignite:binarylizable
type Primitives struct {
	fBool bool
	pBool *bool

	fByte  byte
	pByte  *byte
	fSByte int8
	pSByte *int8
	fUByte uint8
	pUByte *uint8

	fShort int16
	pShort *int16

	fChar uint16
	pChar *uint16

	fInt32  int32
	pInt32  *int32
	fUint32 uint32
	pUint32 *uint32

	fInt64  int64
	pInt64  *int64
	fUint64 uint64
	pUint64 *uint64
	fInt    int
	pInt    *int
	fUint   uint
	pUint   *uint

	fFloat32 float32
	pFloat32 *float32

	fFloat64 float64
	pFloat64 *float64
}
