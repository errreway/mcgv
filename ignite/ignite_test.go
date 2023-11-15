package ignite

import (
	"github.com/stretchr/testify/assert"
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

func TestCacheNames(t *testing.T) {
	cli, err := Start(ClientConfiguration{DefaultAddress})

	assert.Nil(t, err)
	assert.NotNil(t, cli)

	var names *[]string
	names, err = cli.CacheNames()

	assert.Nil(t, err)
	assert.Equal(t, 0, len(*names))

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
	assert.Equal(t, 1, len(*names))
	assert.Equal(t, cacheName, (*names)[0])
}

func TestDestroyCache(t *testing.T) {
	cli, err := Start(ClientConfiguration{DefaultAddress})

	assert.Nil(t, err)
	assert.NotNil(t, cli)

	toDestroy := "to-destroy"

	var cache Cache
	cache, err = cli.CreateCache(toDestroy)

	assert.Nil(t, err)
	assert.NotNil(t, cache)
	assert.Equal(t, toDestroy, cache.Name())

	err = cli.DestroyCache(toDestroy)

	var names *[]string
	names, err = cli.CacheNames()

	assert.Nil(t, err)
	assert.Equal(t, 0, len(*names))
}
