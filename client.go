package ignite

import (
	"context"
	"crypto/tls"
	"errors"
	"gitverse.ru/sbertech/ignite-go-client/logger"
	"net"
	"strconv"
	"time"
)

const (
	defaultPort = 10800
)

type Client interface {
	Version() (string, error)
	Close() error
	CacheNames(ctx context.Context) ([]string, error)
	CreateCache(ctx context.Context, name string) (Cache, error)
	CreateCacheWithConfiguration(ctx context.Context, config CacheConfiguration) (Cache, error)
	GetOrCreateCache(ctx context.Context, name string) (Cache, error)
	GetOrCreateCacheWithConfiguration(ctx context.Context, config CacheConfiguration) (Cache, error)
	DestroyCache(ctx context.Context, name string) error
}

type ClientConfiguration struct {
	addressesSupplier func() ([]string, error)
	shuffleAddresses  bool
	user              string
	password          string
	attrs             map[string]string
	tlsConfigSupplier func() (*tls.Config, error)
	requestTimeout    time.Duration
	retryLimit        int
	logger            *logger.Logger
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

func WithShuffleAddresses(shuffle bool) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		config.shuffleAddresses = shuffle
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

func WithRequestTimeout(timeout time.Duration) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		config.requestTimeout = timeout
		return nil
	}
}

func WithClientAttribute(key string, value string) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		if len(key) != 0 && len(value) != 0 {
			if config.attrs == nil {
				config.attrs = make(map[string]string)
			}
			config.attrs[key] = value
		}
		return nil
	}
}

func WithLoggingSink(sink logger.Sink) func(config *ClientConfiguration) error {
	return func(config *ClientConfiguration) error {
		if sink == nil {
			return nil
		}
		config.logger = &logger.Logger{Sink: sink}
		return nil
	}
}

const (
	defaultTimeout    = 1 * time.Second
	defaultRetryLimit = 0
)

func Start(opts ...func(options *ClientConfiguration) error) (Client, error) {
	cfg := ClientConfiguration{
		requestTimeout:   defaultTimeout,
		retryLimit:       defaultRetryLimit,
		shuffleAddresses: true,
	}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	if cfg.logger == nil {
		dfltSink, _ := logger.NewSink(nil, logger.OffLevel)
		cfg.logger = &logger.Logger{Sink: dfltSink}
	}
	return startClient(cfg)
}
