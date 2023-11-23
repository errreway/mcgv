package ignite

import (
	"errors"
	"sbt.ru/ignite-go/ignite/ignite/serdes"
)

//go:generate go run gen_req_resp.go

type Client interface {
	Version() (string, error)

	Close() error

	CacheNames() (*[]string, error)

	CreateCache(name string) (Cache, error)

	CreateCacheWithConfiguration(ccfg serdes.CacheConfiguration) (Cache, error)

	GetOrCreateCache(name string) (Cache, error)

	DestroyCache(name string) error
}

type ClientConfiguration struct {
	Addresses string
}

type ClientImpl struct {
	cfg    ClientConfiguration
	ch     *serdes.Channel
	caches map[string]CacheImpl
}

func Start(cfg ClientConfiguration) (Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("addresses is empty")
	}

	ch, err := serdes.CreateChannel(cfg.Addresses)

	if err != nil {
		return nil, err
	}

	return ClientImpl{cfg, ch, make(map[string]CacheImpl)}, nil
}

func (cli ClientImpl) CacheNames() (*[]string, error) {
	resp, err := cli.ch.Send(serdes.CacheGetNamesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.(serdes.CacheGetNamesResponse).Caches, nil
}

func (cli ClientImpl) CreateCache(name string) (Cache, error) {
	_, contains := cli.caches[name]
	if contains {
		return nil, errors.New("cache already exists: " + name)
	}

	_, err := cli.ch.Send(serdes.CacheCreateWithNameRequest{Cache: &name})
	if err != nil {
		return nil, err
	}

	return CacheImpl{&cli, &name}, nil
}

func (cli ClientImpl) CreateCacheWithConfiguration(ccfg serdes.CacheConfiguration) (Cache, error) {
	_, contains := cli.caches[*ccfg.Name]
	if contains {
		return nil, errors.New("cache already exists: " + *ccfg.Name)
	}

	_, err := cli.ch.Send(serdes.CacheCreateWithConfigurationRequest{Config: ccfg})
	if err != nil {
		return nil, err
	}

	return CacheImpl{&cli, ccfg.Name}, nil
}

func (cli ClientImpl) GetOrCreateCache(name string) (Cache, error) {
	cache, contains := cli.caches[name]
	if contains {
		return cache, nil
	}

	_, err := cli.ch.Send(serdes.CacheGetOrCreateWithNameRequest{Cache: &name})
	if err != nil {
		return nil, err
	}

	return CacheImpl{&cli, &name}, nil
}

func (cli ClientImpl) DestroyCache(name string) error {
	_, err := cli.ch.Send(serdes.CacheDestroyRequest{CacheId: CacheId(name)})
	if err != nil {
		return err
	}

	delete(cli.caches, name)

	return nil
}

func (cli ClientImpl) Version() (string, error) {
	return "unimplemented", nil
}

func (cli ClientImpl) Close() error {
	return cli.ch.Close()
}
