package ignite

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"sync"
	"testing"
	"time"
)

type TransactionsTestSuite struct {
	testing2.IgniteTestSuite
	client *Client
	cache  *Cache
}

func TestTransactionsTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionsTestSuite))
}

func (suite *TransactionsTestSuite) SetupSuite() {
	_, err := suite.StartIgnite()
	if err != nil {
		suite.T().Fatal("Failed to start ignite instance", err)
	}
	suite.client, err = StartTestClient(context.Background(), WithAddresses(defaultAddress), WithRequestTimeout(500*time.Millisecond))
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	suite.cache, err = suite.client.GetOrCreateCacheWithConfiguration(context.Background(),
		CreateCacheConfiguration("tx-test", WithCacheAtomicityMode(TransactionalAtomicityMode)))
	if err != nil {
		suite.T().Fatal("failed to create test cache", err)
	}
}

func (suite *TransactionsTestSuite) TearDownTest() {
	err := suite.cache.ClearAll(context.Background())
	if err != nil {
		suite.T().Fatal("failed to clear cache", err)
	}
	sz, err := suite.cache.Size(context.Background())
	if err != nil {
		suite.T().Fatal("failed to clear cache", err)
	}
	require.Equal(suite.T(), uint64(0), sz, fmt.Sprintf("cache is not empty: size=%d", sz))
}

func (suite *TransactionsTestSuite) TearDownSuite() {
	suite.KillAllGrids()
	if suite.client != nil {
		_ = suite.client.Close(context.Background())
		suite.client = nil
	}
}

type txTimeoutFixture struct {
	sleepAfterPut bool
	ctxTimeout    time.Duration
	txTimeout     time.Duration
	sleepTime     time.Duration
}

func (suite *TransactionsTestSuite) TestTxTimeout() {
	fixtures := make([]txTimeoutFixture, 0)
	for _, sleepAfterPut := range []bool{true, false} {
		fixtures = append(fixtures,
			txTimeoutFixture{
				sleepAfterPut: sleepAfterPut,
				ctxTimeout:    5 * time.Second,
				txTimeout:     500 * time.Millisecond,
				sleepTime:     1 * time.Second,
			},
		)
		fixtures = append(fixtures,
			txTimeoutFixture{
				sleepAfterPut: sleepAfterPut,
				ctxTimeout:    5 * time.Second,
				txTimeout:     500 * time.Millisecond,
				sleepTime:     100 * time.Millisecond,
			},
		)
		fixtures = append(fixtures,
			txTimeoutFixture{
				sleepAfterPut: sleepAfterPut,
				ctxTimeout:    5 * time.Second,
				txTimeout:     -1,
				sleepTime:     100 * time.Millisecond,
			},
		)
		// Should commit successfully if sleep after put since we retry commit even after ctx timeout.
		// Should fail with [ClientTimeoutError] otherwise.
		fixtures = append(fixtures,
			txTimeoutFixture{
				sleepAfterPut: sleepAfterPut,
				ctxTimeout:    100 * time.Millisecond,
				txTimeout:     0,
				sleepTime:     1 * time.Second,
			},
		)
		fixtures = append(fixtures,
			txTimeoutFixture{
				sleepAfterPut: sleepAfterPut,
				ctxTimeout:    100 * time.Millisecond,
				txTimeout:     5 * time.Second,
				sleepTime:     1 * time.Second,
			},
		)
	}
	for _, f := range fixtures {
		suite.T().Run(fmt.Sprintf("txTimeout=%v, sleepTime=%v, ctxTimeout=%v, sleepAfterPut=%t", f.txTimeout, f.sleepTime, f.ctxTimeout, f.sleepAfterPut),
			func(t *testing.T) {
				defer suite.TearDownTest()
				txOpts := make([]func(options *transactionOptions), 0)
				if f.txTimeout >= 0 {
					txOpts = append(txOpts, WithTimeout(f.txTimeout))
				}
				ctx, cancel := context.WithTimeout(context.Background(), f.ctxTimeout)
				defer cancel()
				err := suite.client.RunInTransaction(ctx, func(tCtx context.Context) error {
					if !f.sleepAfterPut {
						time.Sleep(f.sleepTime)
					}
					err0 := suite.cache.Put(tCtx, "test", "test")
					if f.sleepAfterPut {
						time.Sleep(f.sleepTime)
					}
					return err0
				}, txOpts...)

				ok, err0 := suite.cache.ContainsKey(context.Background(), "test")
				require.NoError(t, err0)

				if f.txTimeout < f.sleepTime && f.txTimeout > 0 {
					require.ErrorContains(t, err, "Cache transaction timed out")
					require.False(t, ok)
				} else if !f.sleepAfterPut && f.ctxTimeout < f.sleepTime {
					var err0 *ClientTimeoutError
					require.ErrorAs(t, err, &err0)
					require.ErrorContains(t, err, "request cancelled")
					require.False(t, ok)
				} else {
					require.NoError(t, err)
					require.True(t, ok)
				}
			},
		)
	}
}

func (suite *TransactionsTestSuite) TestMultipleCacheTransactions() {
	ctx := context.Background()
	cache1 := suite.cache
	cache2, err := suite.client.GetOrCreateCacheWithConfiguration(ctx,
		CreateCacheConfiguration("tx-test-2", WithCacheAtomicityMode(TransactionalAtomicityMode)))
	require.NoError(suite.T(), err)
	defer func() {
		err := suite.client.DestroyCache(ctx, cache2.Name())
		require.NoError(suite.T(), err)
	}()

	for _, commit := range []bool{true, false} {
		suite.T().Run(fmt.Sprintf("commit=%t", commit), func(t *testing.T) {
			defer func() {
				_ = cache1.ClearAll(ctx)
				_ = cache2.ClearAll(ctx)
			}()
			sz, err := cache1.Size(ctx)
			require.NoError(suite.T(), err)
			require.Equal(t, uint64(0), sz)
			sz, err = cache2.Size(ctx)
			require.NoError(suite.T(), err)
			require.Equal(t, uint64(0), sz)

			err = suite.client.RunInTransaction(ctx, func(tCtx context.Context) error {
				err := cache1.Put(tCtx, "test", "test")
				require.NoError(suite.T(), err)
				err = cache2.Put(tCtx, "test", "test")
				require.NoError(suite.T(), err)
				if commit {
					return nil
				}
				return errors.New("rollback")
			})
			if err != nil {
				ok, err := cache1.ContainsKey(context.Background(), "test")
				require.NoError(suite.T(), err)
				require.False(suite.T(), ok)
				ok, err = cache2.ContainsKey(context.Background(), "test")
				require.NoError(suite.T(), err)
				require.False(suite.T(), ok)
			} else {
				ok, err := cache1.ContainsKey(context.Background(), "test")
				require.NoError(suite.T(), err)
				require.True(suite.T(), ok)
				ok, err = cache2.ContainsKey(context.Background(), "test")
				require.NoError(suite.T(), err)
				require.True(suite.T(), ok)
			}
		})
	}
}

func (suite *TransactionsTestSuite) TestPanicInTransaction() {
	var txSess *txSession
	defer func() {
		r := recover()
		require.NotNil(suite.T(), r)
		require.NotNil(suite.T(), txSess)

		// try to commit already closed tx
		closeErr := suite.client.closeTx(context.Background(), txSess, true)
		require.NotNil(suite.T(), closeErr)
		ok, err0 := suite.cache.ContainsKey(context.Background(), "test")
		require.NoError(suite.T(), err0)
		require.False(suite.T(), ok)
	}()
	_ = suite.client.RunInTransaction(context.Background(), func(tCtx context.Context) error {
		txSess = tCtx.Value(txKey{}).(*txSession)
		err0 := suite.cache.Put(tCtx, "test", "test")
		require.NoError(suite.T(), err0)
		panic("panics inside tx")
	})
}

func (suite *TransactionsTestSuite) TestContextTimeoutInTransaction() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var txSess *txSession
	defer func() {
		require.NotNil(suite.T(), txSess)

		// try to commit already closed tx
		closeErr := suite.client.closeTx(context.Background(), txSess, true)
		require.NotNil(suite.T(), closeErr)
		ok, err0 := suite.cache.ContainsKey(context.Background(), "test")
		require.NoError(suite.T(), err0)
		require.False(suite.T(), ok)
	}()
	_ = suite.client.RunInTransaction(ctx, func(tCtx context.Context) error {
		txSess = tCtx.Value(txKey{}).(*txSession)
		time.Sleep(1 * time.Second)
		err0 := suite.cache.Put(tCtx, "test", "test")
		require.NotNil(suite.T(), err0)
		return err0
	})
}

func (suite *TransactionsTestSuite) TestOptimisticError() {
	var slowPessimistic sync.WaitGroup
	var optimisticStart sync.WaitGroup
	var optimisticEnd sync.WaitGroup
	optimisticStart.Add(1)
	optimisticEnd.Add(1)
	slowPessimistic.Add(1)

	go func() {
		defer optimisticEnd.Done()
		ctx := context.Background()
		_ = suite.client.RunInTransaction(ctx, func(tCtx context.Context) error {
			err := suite.cache.Put(tCtx, "test", "test")
			require.NoError(suite.T(), err)
			optimisticStart.Done()
			slowPessimistic.Wait()
			return nil
		}, WithConcurrency(PessimisticConcurrency), WithIsolationLevel(SerializableLevel))
	}()

	optimisticStart.Wait()
	// start optimistic transaction, must return lock conflict
	err := suite.client.RunInTransaction(context.Background(), func(tCtx context.Context) error {
		return suite.cache.Put(tCtx, "test", "test")
	},
		WithConcurrency(OptimisticConcurrency),
		WithIsolationLevel(SerializableLevel),
		WithLabel("failed-optimistic"),
	)
	require.ErrorContains(suite.T(), err, "Failed to prepare transaction (lock conflict)")
	require.ErrorContains(suite.T(), err, "lb=failed-optimistic")
	// waiting for the end of optimistic tx goroutine
	slowPessimistic.Done()
	optimisticEnd.Wait()
}
