package ignite

import (
	"context"
	"fmt"
	"time"
)

const (
	keepBinaryMask    uint8 = 0x01
	transactionalMask uint8 = 0x02
	expiryPolicyMask  uint8 = 0x04
)

const (
	opCacheGet              int16 = 1000
	opCachePut              int16 = 1001
	opCacheGetSize          int16 = 1020
	opCacheGetConfiguration int16 = 1055
)

type expirePolicyImpl struct {
	creation time.Duration
	access   time.Duration
	update   time.Duration
}

func (e *expirePolicyImpl) Creation() time.Duration {
	return e.creation
}

func (e *expirePolicyImpl) Access() time.Duration {
	return e.access
}

func (e *expirePolicyImpl) Update() time.Duration {
	return e.update
}

type cacheImpl struct {
	cli          *clientImpl
	expiryPolicy *expirePolicyImpl
	name         string
	id           int32
}

func (cache *cacheImpl) Name() string {
	return cache.name
}

func (cache *cacheImpl) Get(ctx context.Context, key interface{}) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (cache *cacheImpl) Put(ctx context.Context, key interface{}, value interface{}) error {
	var err error = nil
	cache.cli.ch.Send(ctx, opCachePut, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.Marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.Marshal(ctx, output, value)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}

	})

	return err
}

func (cache *cacheImpl) Size(ctx context.Context, peekModes ...CachePeekMode) (uint64, error) {
	var size uint64 = 0
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheGetSize, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		peekModesSz := int32(len(peekModes))
		output.WriteInt32(peekModesSz)
		if peekModesSz > 0 {
			for _, peekMode := range peekModes {
				output.WriteInt8(int8(peekMode))
			}
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		size = input.ReadUInt64()
	})

	return size, err
}

func (cache *cacheImpl) Configuration(ctx context.Context) (CacheConfiguration, error) {
	var err error
	var cfg CacheConfiguration
	cache.cli.ch.Send(ctx, opCacheGetConfiguration, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		return nil
	}, func(output BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
		} else {
			cfg, err = unmarshall(ctx, cache.cli.marsh, output)
		}
	})

	return cfg, err
}

func (cache *cacheImpl) writeCacheInfo(protoCtx ProtocolContext, output BinaryWriter) error {
	output.WriteInt32(cache.id)
	var flag = keepBinaryMask
	if cache.expiryPolicy != nil {
		if !protoCtx.SupportsExpiryPolicy() {
			return fmt.Errorf("expiry policies are not supported for protocol %v", protoCtx.Version())
		}
		flag |= expiryPolicyMask
	}
	output.WriteUInt8(flag)
	if flag&expiryPolicyMask != 0 {
		output.WriteInt64(durationToMillis(cache.expiryPolicy.Creation()))
		output.WriteInt64(durationToMillis(cache.expiryPolicy.Update()))
		output.WriteInt64(durationToMillis(cache.expiryPolicy.Access()))
	}
	return nil
}

func durationToMillis(dur time.Duration) int64 {
	if dur > 0 {
		return dur.Milliseconds()
	} else if dur <= DurationUnchanged {
		return int64(DurationUnchanged)
	} else {
		return int64(dur)
	}
}

func millisToDuration(millis int64) time.Duration {
	if millis > 0 {
		return time.Duration(millis) * time.Millisecond
	} else {
		return time.Duration(millis)
	}
}
