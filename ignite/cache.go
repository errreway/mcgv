package ignite

import "sbt.ru/ignite-go/ignite/ignite/serdes"

type Cache interface {
	Name() string

	Configuration() (*serdes.CacheConfiguration, error)
}

type CacheImpl struct {
	cli *ClientImpl

	name *string
}

func (cache CacheImpl) Name() string {
	return *cache.name
}

func (cache CacheImpl) Configuration() (*serdes.CacheConfiguration, error) {
	resp, err := cache.cli.ch.Send(serdes.CacheGetConfigurationRequest{CacheId: CacheId(*cache.name), Flag: 0})
	if err != nil {
		return nil, err
	}

	ccfg := resp.(serdes.CacheGetConfigurationResponse).CacheConfiguration

	return &ccfg, nil
}
