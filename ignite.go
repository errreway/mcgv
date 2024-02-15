package ignite

import (
	"context"
	"errors"
)

type Client interface {
	Version() (string, error)

	Close() error

	CacheNames(ctx context.Context) ([]string, error)

	CreateCache(ctx context.Context, name string) (Cache, error)

	CreateCacheWithConfiguration(ctx context.Context, ccfg CacheConfiguration) (Cache, error)

	GetOrCreateCache(ctx context.Context, name string) (Cache, error)

	DestroyCache(ctx context.Context, name string) error
}

type ClientConfiguration struct {
	Addresses string
}

func Start(cfg ClientConfiguration) (Client, error) {
	if len(cfg.Addresses) == 0 {
		return nil, errors.New("addresses is empty")
	}

	return startClient(cfg)
}
