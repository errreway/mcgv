package ignite

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
	"time"
)

type TtlTestSuite struct {
	testing2.IgniteTestSuite
}

func TestTtlTestSuite(t *testing.T) {
	suite.Run(t, new(TtlTestSuite))
}

func (suite *TtlTestSuite) SetupSuite() {
	_, err := suite.StartIgnite(testing2.WithInstanceIndex(0))
	if err != nil {
		suite.T().Fatal("Failed to start ignite instance", err)
	}
}

func (suite *TtlTestSuite) TearDownSuite() {
	suite.KillAllGrids()
}

func (suite *TtlTestSuite) TearDownTest() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	defer func() {
		_ = cli.Close()
	}()
	ctx := context.Background()
	caches, err := cli.CacheNames(ctx)
	require.Nil(suite.T(), err)
	for _, cache := range caches {
		err = cli.DestroyCache(ctx, cache)
		require.Nil(suite.T(), err)
	}
}

func (suite *TtlTestSuite) TestCreationPolicy() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	fixtures := []struct {
		name     string
		supplier func() (Cache, error)
	}{
		{"cache_config", func() (Cache, error) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer func() {
				cancel()
			}()
			cache, err0 := cli.GetOrCreateCache(ctx, "test")
			if err0 != nil {
				return nil, err0
			}
			return cache.WithExpirePolicy(1*time.Second, DurationZero, DurationZero), nil
		}},
		{"cache_decorator", func() (Cache, error) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer func() {
				cancel()
			}()
			return cli.CreateCacheWithConfiguration(ctx,
				CreateCacheConfiguration("test", WithExpirePolicy(1*time.Second, DurationZero, DurationZero)))
		}},
	}

	for _, fixture := range fixtures {
		suite.T().Run(fixture.name, func(t *testing.T) {
			ctx := context.Background()
			cache, err := fixture.supplier()
			require.Nil(suite.T(), err)
			err = cache.Put(ctx, "test", "test")
			defer func() {
				_ = cli.DestroyCache(ctx, "test")
			}()
			require.Nil(suite.T(), err)
			<-time.After(1200 * time.Millisecond)
			contains, err := cache.ContainsKey(ctx, "test")
			require.Nil(suite.T(), err)
			require.False(suite.T(), contains)
		})
	}
}
