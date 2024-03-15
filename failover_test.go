package ignite

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

type FailoverTestSuite struct {
	testing2.IgniteTestSuite
}

func TestFailoverTestSuiteSuite(t *testing.T) {
	suite.Run(t, new(FailoverTestSuite))
}

func (suite *FailoverTestSuite) TearDownSuite() {
	suite.KillAllGrids()
}

func (suite *FailoverTestSuite) AfterTest() {
	suite.KillAllGrids()
}

func (suite *FailoverTestSuite) TestFailover() {
	_, err := suite.StartIgnite()
	if err != nil {
		suite.T().Fatal("failed to start first grid", err)
	}
	_, err = suite.StartIgnite()
	if err != nil {
		suite.T().Fatalf("failed to start second grid")
	}

	cli, err := Start(WithAddresses(defaultAddress+":10800", defaultAddress+":10801"), WithShuffleAddresses(false))
	if err != nil {
		suite.T().Fatal("Failed to start client", err)
		return
	}
	defer func() {
		_ = cli.Close()
	}()
	cache, err := cli.GetOrCreateCache(context.Background(), "test")
	if err != nil {
		suite.T().Fatal("Failed to create cache", err)
		return
	}
	var errCnt int64
	var successCnt int64
	stopCh := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		LOOP:
			for {
				select {
				case <-stopCh:
					return
				default:
					key := rnd.Int63n(1 << 15)
					err := cache.Put(context.Background(), fmt.Sprintf("key-%d", key), "test")
					if err != nil {
						atomic.AddInt64(&errCnt, 1)
					} else {
						atomic.AddInt64(&successCnt, 1)
					}
					continue LOOP
				}
			}
		}()
	}
	testing2.WaitForCondition(func() bool {
		return atomic.LoadInt64(&successCnt) >= 1000
	}, 3*time.Second)
	_ = suite.KillIgnite(0)
	currOk := atomic.LoadInt64(&successCnt)
	testing2.WaitForCondition(func() bool {
		return atomic.LoadInt64(&successCnt) > 2*currOk
	}, 3*time.Second)
	close(stopCh)
	suite.Assert().True(atomic.LoadInt64(&errCnt) <= 10,
		fmt.Sprintf("Total errors %d, total ok %d", atomic.LoadInt64(&errCnt), atomic.LoadInt64(&successCnt)))
}
