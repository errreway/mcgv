package ignite

import (
	"context"
	"errors"
	"sbt.ru/ignite-go/ignite/serdes"
)

//go:generate go run gen_req_resp.go

type Client interface {
	Version() (string, error)

	Close() error

	CacheNames() ([]string, error)

	CreateCache(name string) (Cache, error)

	CreateCacheWithConfiguration(ccfg *serdes.CacheConfiguration) (Cache, error)

	GetOrCreateCache(name string) (Cache, error)

	DestroyCache(name string) error
}

type ClientConfiguration struct {
	Addresses string
}

type ClientImpl struct {
	cfg ClientConfiguration
	ch  *serdes.Channel
}

func Start(cfg ClientConfiguration) (Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("addresses is empty")
	}

	ch, err := serdes.CreateChannel(cfg.Addresses)

	if err != nil {
		return nil, err
	}

	return &ClientImpl{cfg, ch}, nil
}

func (cli *ClientImpl) CacheNames() ([]string, error) {
	var err error = nil
	var resp serdes.CacheGetNamesResponse

	req := serdes.CacheGetNamesRequest{}
	cli.ch.Send(context.Background(), req.OpCode(), func(output serdes.BinaryWriter) {
		req.Write(output)
	}, func(input serdes.BinaryReader, err error) {
		resp = req.ReadResponse(input).(serdes.CacheGetNamesResponse)
	})

	if err != nil {
		return nil, err
	}
	var names []string
	for _, v := range resp.Caches {
		names = append(names, *v)
	}

	return names, nil
}

func (cli *ClientImpl) CreateCache(name string) (Cache, error) {
	var err error
	req := serdes.CacheCreateWithNameRequest{Cache: &name}
	cli.ch.Send(context.Background(), req.OpCode(), func(output serdes.BinaryWriter) {
		req.Write(output)
	}, func(output serdes.BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}

	return &CacheImpl{cli, name}, nil
}

func (cli *ClientImpl) CreateCacheWithConfiguration(config *serdes.CacheConfiguration) (Cache, error) {
	var err error
	req := serdes.CacheCreateWithConfigurationRequest{Config: *config}
	cli.ch.Send(context.Background(), req.OpCode(), func(output serdes.BinaryWriter) {
		req.Write(output)
	}, func(output serdes.BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}

	return &CacheImpl{cli, *config.Name}, nil
}

func (cli *ClientImpl) GetOrCreateCache(name string) (Cache, error) {
	var err error
	req := serdes.CacheGetOrCreateWithNameRequest{Cache: &name}
	cli.ch.Send(context.Background(), req.OpCode(), func(output serdes.BinaryWriter) {
		req.Write(output)
	}, func(output serdes.BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}

	return &CacheImpl{cli, name}, nil
}

func (cli *ClientImpl) DestroyCache(name string) error {
	var err error
	req := serdes.CacheDestroyRequest{CacheId: CacheId(name)}
	cli.ch.Send(context.Background(), req.OpCode(), func(output serdes.BinaryWriter) {
		req.Write(output)
	}, func(output serdes.BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	return err
}

func (cli *ClientImpl) Version() (string, error) {
	return "unimplemented", nil
}

func (cli *ClientImpl) Close() error {
	cli.ch.Close(nil)

	return nil
}
