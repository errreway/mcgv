package ignite

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TransactionConcurrency sets transaction concurrency control.
type TransactionConcurrency int8

const (
	// OptimisticConcurrency sets optimistic concurrency control. This is default one.
	OptimisticConcurrency TransactionConcurrency = iota
	// PessimisticConcurrency sets pessimistic concurrency control.
	PessimisticConcurrency
)

// TransactionIsolationLevel defines different cache transaction isolation levels.
type TransactionIsolationLevel int8

const (
	// ReadCommittedLevel corresponds to read committed isolation level.
	ReadCommittedLevel TransactionIsolationLevel = iota
	// RepeatableReadLevel corresponds to repeatable read isolation level.  This is default one.
	RepeatableReadLevel
	// SerializableLevel corresponds to repeatable read isolation level.  This is default one.
	SerializableLevel
)

const (
	opTxStart int16 = 4000
	opTxEnd   int16 = 4001
)

type txKey struct{}

type txSession struct {
	txId int32
	ch   channel
}

type transactionOptions struct {
	isoLvl      TransactionIsolationLevel
	concurrency TransactionConcurrency
	label       string
	timeout     time.Duration
}

// WithIsolationLevel sets transaction's isolation level option.
func WithIsolationLevel(lvl TransactionIsolationLevel) func(*transactionOptions) {
	return func(o *transactionOptions) {
		o.isoLvl = lvl
	}
}

// WithConcurrency sets transaction's concurrency level option.
func WithConcurrency(concurrency TransactionConcurrency) func(*transactionOptions) {
	return func(o *transactionOptions) {
		o.concurrency = concurrency
	}
}

// WithLabel sets transaction's label option.
func WithLabel(label string) func(*transactionOptions) {
	return func(o *transactionOptions) {
		o.label = label
	}
}

// WithTimeout sets transaction's timeout option.
func WithTimeout(timeout time.Duration) func(*transactionOptions) {
	return func(o *transactionOptions) {
		o.timeout = timeout
	}
}

// RunInTransaction specific closure inside transaction.
//
// Example of usage:
//
//	 ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	 defer cancel()
//	 err := client.RunInTransaction(ctx, func(tCtx context.Context) error {
//			return cache.Put(tCtx, "test", "test")
//		}, WithIsolationLevel(RepeatableReadLevel), WithLabel("test-tx"))
func (cli *Client) RunInTransaction(ctx context.Context, txFunc func(tCtx context.Context) error, opts ...func(options *transactionOptions)) error {
	txOpts := transactionOptions{
		isoLvl:      cli.cfg.dfltTxIsolationLvl,
		concurrency: cli.cfg.dfltTxConcurrency,
		label:       "",
		timeout:     cli.cfg.dfltTxTimeout,
	}
	for _, opt := range opts {
		opt(&txOpts)
	}
	closed := false
	txSess, err := cli.startTx(ctx, &txOpts)
	defer func() {
		// close if txFunc panics, last resort.
		if txSess != nil && !closed {
			_ = cli.closeTx(context.Background(), txSess, false)
		}
	}()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	ctx = context.WithValue(ctx, txKey{}, txSess)
	txErr := txFunc(ctx)
	shouldCommit := txErr == nil

	err = cli.closeTx(ctx, txSess, shouldCommit)
	// retry tx close if context timed out
	var timeoutErr *ClientTimeoutError
	if errors.As(err, &timeoutErr) {
		err = cli.closeTx(context.Background(), txSess, shouldCommit)
	}
	if err != nil {
		if shouldCommit {
			err = fmt.Errorf("failed to commit transaction: %w", err)
		} else {
			err = fmt.Errorf("failed to rollback transaction: %w, original cause of rollback: %s", err, txErr.Error())
		}
	} else {
		// return txFunc error
		err = txErr
	}
	closed = true
	return err
}

func (cli *Client) startTx(ctx context.Context, opts *transactionOptions) (*txSession, error) {
	var tx *txSession
	ch, err := cli.ch.defaultChannel(ctx)
	if err != nil {
		return nil, err
	}
	ch.send(ctx, opTxStart,
		func(currCh channel, output BinaryOutputStream) error {
			protoCtx := currCh.protocolContext()
			if !protoCtx.SupportsTransactions() {
				return fmt.Errorf("transactions are not supported for protocol %v", protoCtx.Version())
			}
			output.WriteInt8(int8(opts.concurrency))
			output.WriteInt8(int8(opts.isoLvl))
			output.WriteInt64(opts.timeout.Milliseconds())
			if len(opts.label) > 0 {
				marshalString(output, opts.label)
			} else {
				output.WriteNull()
			}
			return nil
		},
		func(currCh channel, input BinaryInputStream, err0 error) {
			if err0 != nil {
				err = err0
				return
			}
			tx = &txSession{txId: input.ReadInt32(), ch: currCh}
		},
	)
	return tx, err
}

func (cli *Client) closeTx(ctx context.Context, tx *txSession, committed bool) error {
	var err error
	tx.ch.send(ctx, opTxEnd,
		func(_ channel, output BinaryOutputStream) error {
			output.WriteInt32(tx.txId)
			output.WriteBool(committed)
			return nil
		},
		func(_ channel, _ BinaryInputStream, err0 error) {
			if err0 != nil {
				err = err0
			}
		},
	)
	return err
}
