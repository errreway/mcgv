package ignite

import (
	"errors"
	"sbt.ru/ignite-go/ignite/ignite/serdes"
)

type IgniteClient interface {
	Version() (string, error)

	Close() error
}

type ClientConfiguration struct {
	Addresses string
}

type igniteClientImpl struct {
	cfg ClientConfiguration

	ch *serdes.Channel
}

func (cli igniteClientImpl) Version() (string, error) {
	return "unimplemented", nil
}

func (cli igniteClientImpl) Close() error {
	return cli.ch.Close()
}

//go:generate go run gen_req_resp.go

func Start(cfg ClientConfiguration) (IgniteClient, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("addresses is empty")
	}

	ch, err := serdes.CreateChannel(cfg.Addresses)

	if err != nil {
		return nil, err
	}

	return igniteClientImpl{cfg, ch}, nil
}
