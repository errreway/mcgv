package ignite

import (
	"github.com/stretchr/testify/assert"
	"sbt.ru/ignite-go/ignite/ignite/serdes"
	"testing"
)

const (
	cacheName      = "new-cache"
	DefaultAddress = "localhost:10800"
)

func TestCorrectAddresses(t *testing.T) {
	cli, err := Start(ClientConfiguration{})

	assert.Nil(t, cli)
	assert.Error(t, err, "Addresses is empty!")

	cli, err = Start(ClientConfiguration{""})

	assert.Nil(t, cli)
	assert.Error(t, err, "Addresses is empty!")
}

func Cleanup(t *testing.T) {
	cli, err := Start(ClientConfiguration{DefaultAddress})
	assert.Nil(t, err)
	caches, err := cli.CacheNames()
	assert.Nil(t, err)
	for _, cache := range caches {
		err = cli.DestroyCache(cache)
		assert.Nil(t, err)
	}
	defer func() {
		_ = cli.Close()
	}()
}

func TestCacheNames(t *testing.T) {
	t.Cleanup(func() {
		Cleanup(t)
	})

	cli, err := Start(ClientConfiguration{DefaultAddress})

	assert.Nil(t, err)
	assert.NotNil(t, cli)

	var names []string
	names, err = cli.CacheNames()

	assert.Nil(t, err)
	assert.Equal(t, 0, len(names))

	var cache Cache
	cache, err = cli.CreateCache(cacheName)

	assert.Nil(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, cacheName, cache.Name())

	var cli0 Client
	cli0, err = Start(ClientConfiguration{DefaultAddress})

	assert.Nil(t, err)
	assert.NotNil(t, cli0)

	cache, err = cli0.CreateCache(cacheName)

	assert.NotNil(t, err)
	assert.Nil(t, cache)

	cache, err = cli0.GetOrCreateCache(cacheName)

	assert.Nil(t, err)
	assert.NotNil(t, cache)

	names, err = cli.CacheNames()

	assert.Nil(t, err)
	assert.Equal(t, 1, len(names))
	assert.Equal(t, cacheName, (names)[0])
}

func TestDestroyCache(t *testing.T) {
	t.Cleanup(func() {
		Cleanup(t)
	})
	cli, err := Start(ClientConfiguration{DefaultAddress})

	assert.Nil(t, err)
	assert.NotNil(t, cli)

	namesBefore, err := cli.CacheNames()

	toDestroy := "to-destroy"

	var cache Cache
	cache, err = cli.CreateCache(toDestroy)

	assert.Nil(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, toDestroy, cache.Name())

	err = cli.DestroyCache(toDestroy)
	assert.Nil(t, err)

	var names []string
	names, err = cli.CacheNames()

	assert.Nil(t, err)
	assert.Equal(t, len(namesBefore), len(names))
}

func TestCacheConfig(t *testing.T) {
	t.Cleanup(func() {
		Cleanup(t)
	})

	cli, err := Start(ClientConfiguration{DefaultAddress})

	assert.Nil(t, err)
	assert.NotNil(t, cli)

	cfgTest := "config-test"

	cache, err := cli.CreateCache(cfgTest)
	assert.Nil(t, err)

	readCcfg, err := cache.Configuration()

	assert.Nil(t, err)
	assert.Equal(t, cfgTest, *readCcfg.Name)
	assert.Equal(t, int32(0), readCcfg.Backups)

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
	assert.Nil(t, err)
	assert.NotNil(t, cache)

	readCcfg, err = cache.Configuration()

	assert.Equal(t, cfgTest, *readCcfg.Name)
	assert.Equal(t, int32(1), readCcfg.Backups)
	assert.Equal(t, int32(1), readCcfg.AtomicityMode)
	assert.Equal(t, grpTest, *readCcfg.GroupName)
}
