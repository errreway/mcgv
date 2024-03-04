package ignite

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sc/sbertech/ignite-go-client/internal/testing"
	"testing"
)

const (
	cacheName      = "new-cache"
	defaultAddress = "localhost"
)

type BasicTestSuite struct {
	suite.Suite
	grids []testing2.IgniteInstance
}

func TestBasicTestSuite(t *testing.T) {
	suite.Run(t, new(BasicTestSuite))
}

func (suite *BasicTestSuite) SetupSuite() {
	var err error
	suite.grids = make([]testing2.IgniteInstance, 0)
	ign, err := testing2.StartIgnite()
	if err != nil {
		suite.T().Errorf("Failed to startClient suite: %s", err.Error())
	}
	suite.grids = append(suite.grids, ign)
}

func (suite *BasicTestSuite) TearDownSuite() {
	if suite.grids != nil {
		for _, ign := range suite.grids {
			_ = ign.Kill()
		}
		clear(suite.grids)
	}
}

func (suite *BasicTestSuite) TearDownTest() {
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

func (suite *BasicTestSuite) TestCorrectAddresses() {
	cli, err := Start()

	assert.Nil(suite.T(), cli)
	assert.Error(suite.T(), err, "address supplier is nil")

	cli, err = Start(WithAddresses())

	assert.Nil(suite.T(), cli)
	assert.Error(suite.T(), err, "addresses are empty")
}

func (suite *BasicTestSuite) TestCacheSize() {
	cli, err := Start(WithAddresses(defaultAddress))

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	var cache Cache
	ctx := context.Background()
	cache, err = cli.GetOrCreateCache(ctx, cacheName)
	assert.Nil(suite.T(), err)

	var cacheSz uint64
	cacheSz, err = cache.Size(ctx)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), uint64(0), cacheSz)
	err = cache.Put(ctx, "test", "test")
	assert.Nil(suite.T(), err)
	val, err := cache.Get(ctx, "test")
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), "test", val)
	cacheSz, err = cache.Size(ctx)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), uint64(1), cacheSz)
}

func (suite *BasicTestSuite) TestCacheNames() {
	cli, err := Start(WithAddresses(defaultAddress))

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	var names []string
	ctx := context.Background()
	names, err = cli.CacheNames(ctx)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 0, len(names))

	var cache Cache
	cache, err = cli.CreateCache(ctx, cacheName)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)
	assert.Equal(suite.T(), cacheName, cache.Name())

	var cli0 Client
	cli0, err = Start(WithAddresses(defaultAddress))

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli0)
	defer func() {
		_ = cli0.Close()
	}()

	cache, err = cli0.CreateCache(ctx, cacheName)

	assert.NotNil(suite.T(), err)
	assert.Nil(suite.T(), cache)

	cache, err = cli0.GetOrCreateCache(ctx, cacheName)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)

	names, err = cli.CacheNames(ctx)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(names))
	assert.Equal(suite.T(), cacheName, (names)[0])
}

func (suite *BasicTestSuite) TestDestroyCache() {
	cli, err := Start(WithAddresses(defaultAddress))
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	ctx := context.Background()
	namesBefore, err := cli.CacheNames(ctx)

	toDestroy := "to-destroy"

	var cache Cache
	cache, err = cli.CreateCache(ctx, toDestroy)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)
	assert.Equal(suite.T(), toDestroy, cache.Name())

	err = cli.DestroyCache(ctx, toDestroy)
	assert.Nil(suite.T(), err)

	var names []string
	names, err = cli.CacheNames(ctx)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), len(namesBefore), len(names))
}

func (suite *BasicTestSuite) TestCacheConfig() {
	cli, err := Start(WithAddresses(defaultAddress))
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	ctx := context.Background()
	cfgTest := "config-test"

	cache, err := cli.CreateCache(ctx, cfgTest)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)

	readCcfg, err := cache.Configuration(ctx)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), cfgTest, readCcfg.Name())
	assert.Equal(suite.T(), 0, readCcfg.Backups())

	cfgTest += "-1"
	grpTest := "my-group"
	ccfg := CreateCacheConfiguration(cfgTest,
		WithCacheGroupName(grpTest),
		WithCacheMode(Partitioned),
		WithCacheAtomicityMode(Atomic),
		WithBackupsCount(1),
		WithQueryParallelism(1),
	)

	cache, err = cli.CreateCacheWithConfiguration(ctx, ccfg)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)

	readCcfg, err = cache.Configuration(ctx)

	assert.Equal(suite.T(), cfgTest, readCcfg.Name())
	assert.Equal(suite.T(), 1, readCcfg.Backups())
	assert.Equal(suite.T(), Atomic, readCcfg.CacheAtomicityMode())
	assert.Equal(suite.T(), grpTest, readCcfg.CacheGroupName())
}

func (suite *BasicTestSuite) TestQueryEntitiesConfig() {
	cli, err := Start(WithAddresses(defaultAddress))
	defer func() {
		_ = cli.Close()
	}()
	if !assert.Nil(suite.T(), err) || !assert.NotNil(suite.T(), cli) {
		suite.T().FailNow()
	}

	ctx := context.Background()
	testName := "qry-cache"
	grpName := "qry-grp"

	cfg := CreateCacheConfiguration(testName,
		WithCacheGroupName(grpName),
		WithCacheMode(Partitioned),
		WithCacheAtomicityMode(Atomic),
		WithBackupsCount(1),
		WithQueryParallelism(1),
		WithCacheKeyConfiguration("PersonId", "BUCKET_ID"),
		WithQueryEntity("PersonId", "Person", WithTableName("QUERY_CACHE"),
			WithQueryField("ID", "java.lang.Long", WithKey()),
			WithQueryField("BUCKET_ID", "java.lang.Long", WithKey()),
			WithQueryField("NAME", "java.lang.String", WithNotNull(), WithDefaultValue("")),
			WithQueryField("SALARY", "java.math.BigDecimal",
				WithPrecision(10), WithScale(2), WithNotNull()),
			WithQueryField("PERSON.AGE", "java.lang.Short", WithNotNull(), WithDefaultValue(2)),
			WithFieldAlias("PERSON.AGE", "PERSON_AGE"),
			WithIndex("NAME_IDX", WithIndexField(IndexField{Name: "NAME", Asc: true})),
		))
	cache, err := cli.CreateCacheWithConfiguration(ctx, cfg)
	if !assert.Nil(suite.T(), err) || !assert.NotNil(suite.T(), cache) {
		suite.T().FailNow()
	}
	cfg1, err := cache.Configuration(ctx)
	if !assert.Nil(suite.T(), err) || !assert.Equal(suite.T(), cfg.Name(), cfg1.Name()) {
		suite.T().FailNow()
	}
	cfg2 := cfg1.Copy(WithCacheName("test"))
	assert.Equal(suite.T(), cfg2.CacheGroupName(), cfg1.CacheGroupName())
}
