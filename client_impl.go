package ignite

import (
	"context"
	"gitverse.ru/sbertech/ignite-go-client/internal"
)

const (
	opCacheGetNames              int16 = 1050
	opCacheCreateWithName        int16 = 1051
	opCacheGetOrCreateWithName   int16 = 1052
	opCacheCreateWithConfig      int16 = 1053
	opCacheGetOrCreateWithConfig int16 = 1054
	opCacheDestroy               int16 = 1056
)

type clientImpl struct {
	cfg   *ClientConfiguration
	ch    Channel
	marsh marshaller
}

func startClient(cfg ClientConfiguration) (Client, error) {
	ch, err := CreateReliableChannel(&cfg)
	if err != nil {
		return nil, err
	}
	cli := clientImpl{cfg: &cfg, ch: ch}
	cli.marsh = newMarshaller(&cli)
	return &cli, err
}

func (cli *clientImpl) CacheNames(ctx context.Context) ([]string, error) {
	var err error = nil
	var names []string
	cli.ch.Send(ctx, opCacheGetNames, func(output BinaryWriter) error {
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		sz := int(input.ReadInt32())
		names = make([]string, sz)
		for i := 0; i < sz; i++ {
			var name string
			name, err0 = unmarshalString(input, false)
			if err0 != nil {
				err = err0
				return
			}
			names[i] = name
		}
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}

func (cli *clientImpl) CreateCache(ctx context.Context, name string) (Cache, error) {
	var err error
	cli.ch.Send(ctx, opCacheCreateWithName, func(output BinaryWriter) error {
		marshalString(output, name)
		return nil
	}, func(output BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}

	return cli.newCache(name), nil
}

func (cli *clientImpl) CreateCacheWithConfiguration(ctx context.Context, config CacheConfiguration) (Cache, error) {
	var err error
	cli.ch.Send(ctx, opCacheCreateWithConfig, func(output BinaryWriter) error {
		if err0 := config.marshall(ctx, cli.marsh, output); err0 != nil {
			return err0
		}
		return nil
	}, func(output BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}
	return cli.newCache(config.Name()), nil
}

func (cli *clientImpl) GetOrCreateCache(ctx context.Context, name string) (Cache, error) {
	var err error
	cli.ch.Send(ctx, opCacheGetOrCreateWithName, func(output BinaryWriter) error {
		if err0 := cli.marsh.marshal(ctx, output, name); err0 != nil {
			return err0
		}
		return nil
	}, func(output BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}
	return cli.newCache(name), nil
}

func (cli *clientImpl) GetOrCreateCacheWithConfiguration(ctx context.Context, config CacheConfiguration) (Cache, error) {
	var err error
	cli.ch.Send(ctx, opCacheGetOrCreateWithConfig, func(output BinaryWriter) error {
		if err0 := config.marshall(ctx, cli.marsh, output); err0 != nil {
			return err0
		}
		return nil
	}, func(output BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	if err != nil {
		return nil, err
	}

	return cli.newCache(config.Name()), nil
}

func (cli *clientImpl) DestroyCache(ctx context.Context, name string) error {
	var err error
	cli.ch.Send(ctx, opCacheDestroy, func(output BinaryWriter) error {
		output.WriteInt32(internal.HashCode(name))
		return nil
	}, func(output BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		}
	})
	return err
}

func (cli *clientImpl) newCache(name string) *cacheImpl {
	return &cacheImpl{
		cli:  cli,
		name: name,
		id:   internal.HashCode(name),
	}
}

func (cli *clientImpl) Version() (string, error) {
	ver := cli.ch.ProtocolContext().Version()
	return ver.String(), nil
}

func (cli *clientImpl) Close() error {
	cli.ch.Close()
	return nil
}
