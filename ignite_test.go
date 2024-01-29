package ignite

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	testing2 "sbt.ru/ignite-go/ignite/internal/testing"
	"sbt.ru/ignite-go/ignite/serdes"
	"testing"
)

const (
	cacheName      = "new-cache"
	DefaultAddress = "localhost:10800"
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
		suite.T().Errorf("Failed to start suite: %s", err.Error())
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
	cli, err := Start(ClientConfiguration{DefaultAddress})
	assert.Nil(suite.T(), err)
	defer func() {
		_ = cli.Close()
	}()
	caches, err := cli.CacheNames()
	assert.Nil(suite.T(), err)
	for _, cache := range caches {
		err = cli.DestroyCache(cache)
		assert.Nil(suite.T(), err)
	}
}

func (suite *BasicTestSuite) TestCorrectAddresses() {
	cli, err := Start(ClientConfiguration{})

	assert.Nil(suite.T(), cli)
	assert.Error(suite.T(), err, "Addresses is empty!")

	cli, err = Start(ClientConfiguration{""})

	assert.Nil(suite.T(), cli)
	assert.Error(suite.T(), err, "Addresses is empty!")
}

func (suite *BasicTestSuite) TestCacheNames() {
	cli, err := Start(ClientConfiguration{DefaultAddress})

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	var names []string
	names, err = cli.CacheNames()

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 0, len(names))

	var cache Cache
	cache, err = cli.CreateCache(cacheName)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)
	assert.Equal(suite.T(), cacheName, cache.Name())

	var cli0 Client
	cli0, err = Start(ClientConfiguration{DefaultAddress})

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli0)
	defer func() {
		_ = cli0.Close()
	}()

	cache, err = cli0.CreateCache(cacheName)

	assert.NotNil(suite.T(), err)
	assert.Nil(suite.T(), cache)

	cache, err = cli0.GetOrCreateCache(cacheName)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)

	names, err = cli.CacheNames()

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 1, len(names))
	assert.Equal(suite.T(), cacheName, (names)[0])
}

func (suite *BasicTestSuite) TestDestroyCache() {
	cli, err := Start(ClientConfiguration{DefaultAddress})
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	namesBefore, err := cli.CacheNames()

	toDestroy := "to-destroy"

	var cache Cache
	cache, err = cli.CreateCache(toDestroy)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)
	assert.Equal(suite.T(), toDestroy, cache.Name())

	err = cli.DestroyCache(toDestroy)
	assert.Nil(suite.T(), err)

	var names []string
	names, err = cli.CacheNames()

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), len(namesBefore), len(names))
}

func (suite *BasicTestSuite) TestCacheConfig() {
	cli, err := Start(ClientConfiguration{DefaultAddress})
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cli)
	defer func() {
		_ = cli.Close()
	}()

	cfgTest := "config-test"

	cache, err := cli.CreateCache(cfgTest)
	assert.Nil(suite.T(), err)

	readCcfg, err := cache.Configuration()

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), cfgTest, *readCcfg.Name)
	assert.Equal(suite.T(), int32(0), readCcfg.Backups)

	ccfg := serdes.CacheConfiguration{}

	cfgTest += "-1"
	ccfg.Name = &cfgTest
	ccfg.Backups = 1
	ccfg.AtomicityMode = 1
	ccfg.QueryParallelism = 1
	ccfg.CacheMode = 2
	ccfg.RebalanceBatchSize = 512 * 1024
	ccfg.RebalanceBatchesPrefetchCount = 3
	ccfg.RebalanceTimeout = 10000

	grpTest := "my-group"

	ccfg.GroupName = &grpTest

	cache, err = cli.CreateCacheWithConfiguration(&ccfg)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), cache)

	readCcfg, err = cache.Configuration()

	assert.Equal(suite.T(), cfgTest, *readCcfg.Name)
	assert.Equal(suite.T(), int32(1), readCcfg.Backups)
	assert.Equal(suite.T(), int32(1), readCcfg.AtomicityMode)
	assert.Equal(suite.T(), grpTest, *readCcfg.GroupName)
}
