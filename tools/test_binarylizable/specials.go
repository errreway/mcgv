//go:build testing

package test_binarylizable

import (
	dec "github.com/cockroachdb/apd/v3"
	uid "github.com/google/uuid"
	ign "gitverse.ru/sbertech/ignite-go-client"
	t "time"
)

//go:generate go run gitverse.ru/sbertech/ignite-go-client/tools/generator -build_tags=testing -with_register_func

// ignite:binarylizable
type Specials struct {
	fString    string
	pString    *string
	fUuid      uid.UUID
	pUuid      *uid.UUID
	fDate      ign.Date
	pDate      *ign.Date
	fTime      ign.Time
	pTime      *ign.Time
	fTimestamp t.Time
	pTimestamp *t.Time
	fDecimal   dec.Decimal
	pDecimal   *dec.Decimal
}

// ignite:binarylizable
type SpecialArrays struct {
	stringArray     []string
	stringPArray    []*string
	uuidArray       []uid.UUID
	uuidPArray      []*uid.UUID
	dateArray       []ign.Date
	datePArray      []*ign.Date
	timeArray       []ign.Time
	timePArray      []*ign.Time
	timeStampArray  []t.Time
	timeStampPArray []*t.Time
	decimalArray    []dec.Decimal
	decimalPArray   []*dec.Decimal
}
