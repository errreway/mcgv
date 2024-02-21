package ignite

import (
	"context"
	"time"
)

type CachePeekMode uint8

const (
	All CachePeekMode = iota
	Near
	Primary
	Backup
	OnHeap
	OffHeap
)

const (
	DurationUnchanged time.Duration = -2
	DurationEternal   time.Duration = -1
	DurationZero      time.Duration = 0
)

type Cache interface {
	WithExpirePolicy(creation time.Duration, access time.Duration, update time.Duration) Cache
	Name() string
	Get(ctx context.Context, key interface{}) (interface{}, error)
	Put(ctx context.Context, key interface{}, value interface{}) error
	Size(ctx context.Context, peekModes ...CachePeekMode) (uint64, error)
	Configuration(ctx context.Context) (CacheConfiguration, error)
}

type ExpirePolicy interface {
	Creation() time.Duration
	Access() time.Duration
	Update() time.Duration
}
