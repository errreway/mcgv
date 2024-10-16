package ignite

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	opSqlQuery         int16 = 2004
	opSqlQueryNextPage int16 = 2005
)

type sqlQueryOpts struct {
	args                []interface{}
	timeout             time.Duration
	isLocal             bool
	isColocated         bool
	isJoinOrderEnforced bool
	isDistributedJoins  bool
	pageSize            int
	partitions          []int
	schema              string
	updateBatchSize     int
}

type sqlQueryCursor struct {
	baseCursor
	columns []string
}

func (cursor *sqlQueryCursor) Columns() []string {
	return cursor.columns
}

type SqlQueryOption func(*sqlQueryOpts) error

// WithSqlQueryArguments Sets query arguments.
func WithSqlQueryArguments(args ...interface{}) SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.args = args
		return nil
	}
}

// WithSqlQueryTimeout Sets query execution timeout.
func WithSqlQueryTimeout(timeout time.Duration) SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.timeout = timeout
		return nil
	}
}

// WithSqlQueryLocal Sets flag indicating that query must be executed only on the node to which current client is connected.
func WithSqlQueryLocal() SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.isLocal = true
		return nil
	}
}

// WithSqlQueryColocated Sets hint flag indicating that the elements of query selection are colocated together on the same node.
func WithSqlQueryColocated() SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.isColocated = true
		return nil
	}
}

// WithSqlQueryJoinOrderEnforced Sets a flag indicating that the query optimizer is not allowed to reorder table joins.
func WithSqlQueryJoinOrderEnforced() SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.isJoinOrderEnforced = true
		return nil
	}
}

// WithSqlQueryDistributedJoins Sets a flag indicating that distributed joins are allowed.
func WithSqlQueryDistributedJoins() SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.isDistributedJoins = true
		return nil
	}
}

// WithSqlQueryPageSize Sets query result page size.
func WithSqlQueryPageSize(pageSize int) SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		if pageSize <= 0 {
			return errors.New("page size must be greater than zero")
		}
		queryOpts.pageSize = pageSize
		return nil
	}
}

// WithSqlQueryPartitions Sets list of partitions. The query will be executed only on nodes which are primary for specified partitions.
func WithSqlQueryPartitions(partitions ...int) SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		queryOpts.partitions = partitions
		return nil
	}
}

// WithSqlQuerySchema Sets SQL Schema name.
func WithSqlQuerySchema(schema string) SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		schema = strings.TrimSpace(schema)
		if len(schema) == 0 {
			return errors.New("schema cannot be empty")
		}
		queryOpts.schema = schema
		return nil
	}
}

// WithSqlQueryUpdateBatchSize Sets update internal batch size.
func WithSqlQueryUpdateBatchSize(updateBatchSize int) SqlQueryOption {
	return func(queryOpts *sqlQueryOpts) error {
		if updateBatchSize < 1 {
			return errors.New("update batch size must be greater than one")
		}
		queryOpts.updateBatchSize = updateBatchSize
		return nil
	}
}

func newSqlQueryCursor(ctx context.Context, ch channel, marsh marshaller, curId int64) *sqlQueryCursor {
	return &sqlQueryCursor{
		baseCursor: baseCursor{
			ctx:        ctx,
			ch:         ch,
			curId:      curId,
			currIdx:    -1,
			opPageNext: opSqlQueryNextPage,
			marsh:      marsh,
		},
	}
}

func (cursor *sqlQueryCursor) readDataColumns(input BinaryInputStream) error {
	columns, err := readSlice(input,
		func(_ int, stream BinaryInputStream) (string, error) {
			return unmarshalString(stream)
		},
	)
	if err != nil {
		return err
	}
	cursor.columns = columns
	cursor.rowSize = len(columns)
	return nil
}

func writeQueryPartitions(output BinaryOutputStream, partitions []int) error {
	partsCnt := len(partitions)

	if partsCnt > 0 {
		return writeSequence(output, partsCnt,
			func(output BinaryOutputStream, idx int) error {
				output.WriteInt32(int32(partitions[idx]))
				return nil
			},
		)
	} else {
		output.WriteInt32(-1)
		return nil
	}
}

func writeQueryArguments(ctx context.Context, output BinaryOutputStream, marsh marshaller, args []interface{}) error {
	return writeSequence(output, len(args),
		func(output BinaryOutputStream, idx int) error {
			return marsh.marshal(ctx, output, args[idx])
		},
	)
}
