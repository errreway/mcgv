package ignite

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"gitverse.ru/sbertech/ignite-go-client/logger"
	"log"
	"os"
	"testing"
)

const (
	cacheName      = "new-cache"
	defaultAddress = "localhost"
)

type BasicTestSuite struct {
	testing2.IgniteTestSuite
}

func StartTestClient(opts ...func(options *ClientConfiguration) error) (Client, error) {
	dummyCfg := ClientConfiguration{}
	for _, opt := range opts {
		_ = opt(&dummyCfg)
	}
	if dummyCfg.addressesSupplier == nil {
		opts = append(opts, WithAddresses(defaultAddress))
	}
	if dummyCfg.logger == nil {
		sink, _ := logger.NewSink(log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds), logger.DebugLevel) // level is ok, error can be ignored.
		opts = append(opts, WithLoggingSink(sink))
	}
	return Start(opts...)
}

func TestBasicTestSuite(t *testing.T) {
	suite.Run(t, new(BasicTestSuite))
}

func (suite *BasicTestSuite) SetupSuite() {
	_, err := suite.StartIgnite()
	if err != nil {
		suite.T().Fatal("Failed to start ignite instance", err)
	}
}

func (suite *BasicTestSuite) TearDownSuite() {
	suite.KillAllGrids()
}

func (suite *BasicTestSuite) TearDownTest() {
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

func (suite *BasicTestSuite) TestCorrectAddresses() {
	cli, err := Start()
	require.Nil(suite.T(), cli)
	require.Error(suite.T(), err, "address supplier is nil")

	cli, err = Start(WithAddresses())
	require.Nil(suite.T(), cli)
	require.Error(suite.T(), err, "addresses are empty")
}

func (suite *BasicTestSuite) TestCacheSize() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	var cache Cache
	ctx := context.Background()
	cache, err = cli.GetOrCreateCache(ctx, cacheName)
	if err != nil {
		suite.T().Fatal("failed to obtain cache", err)
	}

	var cacheSz uint64
	cacheSz, err = cache.Size(ctx)
	require.Nil(suite.T(), err)
	require.Equal(suite.T(), uint64(0), cacheSz)
	err = cache.Put(ctx, "test", "test")
	require.Nil(suite.T(), err)
	val, err := cache.Get(ctx, "test")
	require.Nil(suite.T(), err)
	require.Equal(suite.T(), "test", val)
	cacheSz, err = cache.Size(ctx)
	require.Nil(suite.T(), err)
	require.Equal(suite.T(), uint64(1), cacheSz)
}

func (suite *BasicTestSuite) TestCacheNames() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	defer func() {
		_ = cli.Close()
	}()

	var names []string
	ctx := context.Background()
	names, err = cli.CacheNames(ctx)

	require.Nil(suite.T(), err)
	require.Equal(suite.T(), 0, len(names))

	var cache Cache
	cache, err = cli.CreateCache(ctx, cacheName)

	require.Nil(suite.T(), err)
	require.NotNil(suite.T(), cache)
	require.Equal(suite.T(), cacheName, cache.Name())

	var cli0 Client
	cli0, err = StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}

	defer func() {
		_ = cli0.Close()
	}()

	cache, err = cli0.CreateCache(ctx, cacheName)

	require.NotNil(suite.T(), err)
	require.Nil(suite.T(), cache)

	cache, err = cli0.GetOrCreateCache(ctx, cacheName)

	require.Nil(suite.T(), err)
	require.NotNil(suite.T(), cache)

	names, err = cli.CacheNames(ctx)

	require.Nil(suite.T(), err)
	require.Equal(suite.T(), 1, len(names))
	require.Equal(suite.T(), cacheName, (names)[0])
}

func (suite *BasicTestSuite) TestDestroyCache() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	require.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	ctx := context.Background()
	namesBefore, err := cli.CacheNames(ctx)
	if err != nil {
		suite.T().Fatal(err)
	}

	toDestroy := "to-destroy"

	var cache Cache
	cache, err = cli.CreateCache(ctx, toDestroy)
	if err != nil {
		suite.T().Fatal(err)
	}

	require.NotNil(suite.T(), cache)
	require.Equal(suite.T(), toDestroy, cache.Name())

	err = cli.DestroyCache(ctx, toDestroy)
	if err != nil {
		suite.T().Fatal(err)
	}

	var names []string
	names, err = cli.CacheNames(ctx)
	if err != nil {
		suite.T().Fatal(err)
	}

	require.Equal(suite.T(), len(namesBefore), len(names))
}

func (suite *BasicTestSuite) TestCacheConfig() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	require.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	ctx := context.Background()
	cfgTest := "config-test"

	cache, err := cli.CreateCache(ctx, cfgTest)
	if err != nil {
		suite.T().Fatal(err)
	}
	require.NotNil(suite.T(), cache)

	readCcfg, err := cache.Configuration(ctx)
	if err != nil {
		suite.T().Fatal(err)
	}
	require.Equal(suite.T(), cfgTest, readCcfg.Name())
	require.Equal(suite.T(), 0, readCcfg.Backups())

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
	if err != nil {
		suite.T().Fatal(err)
	}
	require.NotNil(suite.T(), cache)

	readCcfg, err = cache.Configuration(ctx)
	if err != nil {
		suite.T().Fatal(err)
	}

	require.Equal(suite.T(), cfgTest, readCcfg.Name())
	require.Equal(suite.T(), 1, readCcfg.Backups())
	require.Equal(suite.T(), Atomic, readCcfg.CacheAtomicityMode())
	require.Equal(suite.T(), grpTest, readCcfg.CacheGroupName())
}

func (suite *BasicTestSuite) TestQueryEntitiesConfig() {
	cli, err := StartTestClient()
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
	defer func() {
		_ = cli.Close()
	}()

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
	require.Nil(suite.T(), err)
	require.NotNil(suite.T(), cache)
	cfg1, err := cache.Configuration(ctx)
	require.Nil(suite.T(), err)
	require.Equal(suite.T(), cfg.Name(), cfg1.Name())

	cfg2 := cfg1.Copy(WithCacheName("test"))
	require.Equal(suite.T(), cfg2.CacheGroupName(), cfg1.CacheGroupName())
}
