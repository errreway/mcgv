package ignite

import (
	"context"
)

const (
	opQueryScan         int16 = 2000
	opQueryScanNextPage int16 = 2001
)

type scanCursor struct {
	baseCursor
}

func newScanCursor(ctx context.Context, ch channel, marsh marshaller, curId int64) *scanCursor {
	return &scanCursor{
		baseCursor{
			ctx:        ctx,
			ch:         ch,
			curId:      curId,
			currIdx:    -1,
			rowSize:    2,
			opPageNext: opQueryScanNextPage,
			marsh:      marsh,
		},
	}
}

func (sc *scanCursor) Columns() []string {
	return []string{"KEY", "VALUE"}
}
