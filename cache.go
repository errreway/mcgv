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
	Configuration(ctx context.Context) (CacheConfiguration, error)
	Get(ctx context.Context, key interface{}) (interface{}, error)
	GetAndPut(ctx context.Context, key interface{}, value interface{}) (interface{}, error)
	GetAndReplace(ctx context.Context, key interface{}, value interface{}) (interface{}, error)
	Put(ctx context.Context, key interface{}, value interface{}) error
	PutIfAbsent(ctx context.Context, key interface{}, value interface{}) (bool, error)
	GetAndPutIfAbsent(ctx context.Context, key interface{}, value interface{}) (interface{}, error)
	ContainsKey(ctx context.Context, key interface{}) (bool, error)
	ContainsKeys(ctx context.Context, keys ...interface{}) (bool, error)
	GetAll(ctx context.Context, keys ...interface{}) ([]KeyValue, error)
	PutAll(ctx context.Context, keysAndValues ...KeyValue) error
	ReplaceIfEquals(ctx context.Context, key interface{}, oldValue interface{}, newValue interface{}) (bool, error)
	Replace(ctx context.Context, key interface{}, value interface{}) (bool, error)
	Remove(ctx context.Context, key interface{}) (bool, error)
	GetAndRemove(ctx context.Context, key interface{}) (interface{}, error)
	RemoveIfEquals(ctx context.Context, key interface{}, oldValue interface{}) (bool, error)
	RemoveKeys(ctx context.Context, keys ...interface{}) error
	RemoveAll(ctx context.Context) error
	Clear(ctx context.Context, key interface{}) error
	ClearKeys(ctx context.Context, keys ...interface{}) error
	ClearAll(ctx context.Context) error
	Size(ctx context.Context, peekModes ...CachePeekMode) (uint64, error)
}

type KeyValue struct {
	Key   interface{}
	Value interface{}
}

type ExpirePolicy interface {
	Creation() time.Duration
	Access() time.Duration
	Update() time.Duration
}
