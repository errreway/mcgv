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
}

type ClientConfiguration struct {
	Addresses string
}

type ClientImpl struct {
	cfg ClientConfiguration

	ch *serdes.Channel
}

func (cli ClientImpl) CacheNames() (*[]string, error) {
	resp, err := cli.ch.Send(serdes.CreateCacheGetNamesRequest())
	if err != nil {
		return nil, err
	}

	return resp.(serdes.CacheGetNamesResponse).Caches, nil
}

func (cli ClientImpl) Version() (string, error) {
	return "unimplemented", nil
}

func (cli ClientImpl) Close() error {
	return cli.ch.Close()
}

func Start(cfg ClientConfiguration) (Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("addresses is empty")
	}

	ch, err := serdes.CreateChannel(cfg.Addresses)

	if err != nil {
		return nil, err
	}

	return ClientImpl{cfg, ch}, nil
}
