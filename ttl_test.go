package ignite

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sc/sbertech/ignite-go-client/internal/testing"
	"testing"
	"time"
)

type TtlTestSuite struct {
	suite.Suite
	grids []testing2.IgniteInstance
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(TtlTestSuite))
}

func (suite *TtlTestSuite) SetupSuite() {
	var err error
	suite.grids = make([]testing2.IgniteInstance, 0)
	ign, err := testing2.StartIgnite()
	if err != nil {
		suite.T().Errorf("Failed to startClient suite: %s", err.Error())
	}
	suite.grids = append(suite.grids, ign)
}

func (suite *TtlTestSuite) TearDownSuite() {
	if suite.grids != nil {
		for _, ign := range suite.grids {
			_ = ign.Kill()
		}
		clear(suite.grids)
	}
}

func (suite *TtlTestSuite) TearDownTest() {
	cli, err := Start(WithAddresses(defaultAddress))
	assert.Nil(suite.T(), err)
	defer func() {
		_ = cli.Close()
	}()
	ctx := context.Background()
	caches, err := cli.CacheNames(ctx)
	assert.Nil(suite.T(), err)
	for _, cache := range caches {
		err = cli.DestroyCache(ctx, cache)
		assert.Nil(suite.T(), err)
	}
}

func (suite *TtlTestSuite) TestCreationPolicy() {
	cli, err := Start(WithAddresses(defaultAddress))
	assert.Nil(suite.T(), err)
	defer func() {
		_ = cli.Close()
	}()

	ctx := context.Background()
	cache, err := cli.CreateCache(ctx, "test")
	cache = cache.WithExpirePolicy(1*time.Second, DurationZero, DurationZero)
	err = cache.Put(ctx, "test", "test")
	assert.Nil(suite.T(), err)
	<-time.After(1200 * time.Millisecond)
	contains, err := cache.ContainsKey(ctx, "test")
	assert.Nil(suite.T(), err)
	assert.False(suite.T(), contains)
}
