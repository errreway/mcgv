package ignite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
)

const (
	defaultPort = 10800
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
	addressesSupplier func() ([]string, error)
	user              string
	password          string
	tlsConfigSupplier func() (*tls.Config, error)
}

func WithAddressSupplier(supplier func() ([]string, error)) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		if supplier == nil {
			return errors.New("nil address supplier")
		}
		config.addressesSupplier = supplier
		return nil
	}
}

func WithAddresses(addresses ...string) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		if len(addresses) == 0 {
			return errors.New("empty addressed supplied")
		}
		preparedAddrs := make([]string, 0)
		for _, addr := range addresses {
			_, _, err := net.SplitHostPort(addr)
			if err != nil {
				addr = net.JoinHostPort(addr, strconv.Itoa(defaultPort))
			}
			_, _, err = net.SplitHostPort(addr)
			if err != nil {
				return err
			}
			preparedAddrs = append(preparedAddrs, addr)
		}
		config.addressesSupplier = func() ([]string, error) {
			return preparedAddrs, nil
		}
		return nil
	}
}

func WithCredentials(username string, password string) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		if len(username) != 0 && len(password) != 0 {
			config.user = username
			config.password = password
		}
		return nil
	}
}

func WithTls(supplier func() (*tls.Config, error)) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		if supplier == nil {
			return errors.New("nil tls configuration supplier")
		}
		config.tlsConfigSupplier = supplier
		return nil
	}
}

func Start(opts ...func(options *ClientConfiguration) error) (Client, error) {
	cfg := ClientConfiguration{}

	if len(opts) > 0 {
		for _, opt := range opts {
			if err := opt(&cfg); err != nil {
				return nil, err
			}
		}
	}
	return startClient(cfg)
}
