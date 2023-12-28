package ignite

import (
	"context"
	"sbt.ru/ignite-go/ignite/serdes"
)

type Cache interface {
	Name() string

	Configuration() (*serdes.CacheConfiguration, error)
}

type CacheImpl struct {
	cli *ClientImpl

	name string
}

func (cache *CacheImpl) Name() string {
	return cache.name
}

func (cache *CacheImpl) Configuration() (*serdes.CacheConfiguration, error) {
	var err error
	req := serdes.CacheGetConfigurationRequest{CacheId: CacheId(cache.name), Flag: 0}
	var cfg *serdes.CacheConfiguration = nil
	cache.cli.ch.Send(context.Background(), req.OpCode(), func(output serdes.BinaryWriter) {
		req.Write(output)
	}, func(output serdes.BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		} else {
			resp, _ := req.ReadResponse(output).(serdes.CacheGetConfigurationResponse)
			cfg = &resp.CacheConfiguration
		}
	})

	return cfg, err
}
