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
	opCacheGet               int16 = 1000
	opCachePut               int16 = 1001
	opCachePutIfAbsent       int16 = 1002
	opCacheGetAll            int16 = 1003
	opCachePutAll            int16 = 1004
	opCacheGetAndPut         int16 = 1005
	opCacheGetAndReplace     int16 = 1006
	opCacheGetAndRemove      int16 = 1007
	opCacheGetAndPutIfAbsent int16 = 1008
	opCacheReplace           int16 = 1009
	opCacheReplaceIfEquals   int16 = 1010
	opCacheContainsKey       int16 = 1011
	opCacheContainsKeys      int16 = 1012
	opCacheClear             int16 = 1013
	opCacheClearKey          int16 = 1014
	opCacheClearKeys         int16 = 1015
	opCacheRemoveKey         int16 = 1016
	opCacheRemoveIfEquals    int16 = 1017
	opCacheRemoveKeys        int16 = 1018
	opCacheRemoveAll         int16 = 1019
	opCacheGetSize           int16 = 1020
	opCacheGetConfiguration  int16 = 1055
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

func (cache *cacheImpl) Get(ctx context.Context, key interface{}) (interface{}, error) {
	if key == nil {
		return nil, fmt.Errorf("nil key")
	}
	var err error = nil
	var ret interface{}
	cache.cli.ch.Send(ctx, opCacheGet, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret, err = cache.cli.marsh.unmarshall(ctx, input)
	})
	return ret, err
}

func (cache *cacheImpl) GetAndPut(ctx context.Context, key interface{}, value interface{}) (interface{}, error) {
	if key == nil {
		return nil, fmt.Errorf("nil key")
	}
	if value == nil {
		return nil, fmt.Errorf("nil value")
	}
	var ret interface{}
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheGetAndPut, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, value)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret, err = cache.cli.marsh.unmarshall(ctx, input)
	})
	return ret, err
}

func (cache *cacheImpl) GetAndReplace(ctx context.Context, key interface{}, value interface{}) (interface{}, error) {
	if key == nil {
		return nil, fmt.Errorf("nil key")
	}
	if value == nil {
		return nil, fmt.Errorf("nil value")
	}
	var ret interface{} = nil
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheGetAndReplace, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, value)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret, err = cache.cli.marsh.unmarshall(ctx, input)
	})
	return ret, err
}

func (cache *cacheImpl) Put(ctx context.Context, key interface{}, value interface{}) error {
	if key == nil {
		return fmt.Errorf("nil key")
	}
	if value == nil {
		return fmt.Errorf("nil value")
	}
	var err error = nil
	cache.cli.ch.Send(ctx, opCachePut, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, value)
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

func (cache *cacheImpl) PutIfAbsent(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	if value == nil {
		return false, fmt.Errorf("nil value")
	}
	var ret bool
	var err error = nil
	cache.cli.ch.Send(ctx, opCachePutIfAbsent, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, value)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) GetAndPutIfAbsent(ctx context.Context, key interface{}, value interface{}) (interface{}, error) {
	if key == nil {
		return nil, fmt.Errorf("nil key")
	}
	if value == nil {
		return nil, fmt.Errorf("nil value")
	}
	var ret interface{}
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheGetAndPutIfAbsent, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, value)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret, err = cache.cli.marsh.unmarshall(ctx, input)
	})
	return ret, err
}

func (cache *cacheImpl) ContainsKey(ctx context.Context, key interface{}) (bool, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	var err error = nil
	var ret bool
	cache.cli.ch.Send(ctx, opCacheContainsKey, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) ContainsKeys(ctx context.Context, keys ...interface{}) (bool, error) {
	if len(keys) == 0 {
		return true, nil
	}
	var err error
	var ret bool
	cache.cli.ch.Send(ctx, opCacheContainsKeys, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = writeCollection(output, keys, func(output0 BinaryWriter, el interface{}) error {
			return cache.cli.marsh.marshal(ctx, output0, el)
		})
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) GetAll(ctx context.Context, keys ...interface{}) ([]KeyValue, error) {
	ret := make([]KeyValue, 0)
	if len(keys) == 0 {
		return ret, nil
	}
	var err error
	cache.cli.ch.Send(ctx, opCacheGetAll, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = writeCollection(output, keys, func(output0 BinaryWriter, el interface{}) error {
			return cache.cli.marsh.marshal(ctx, output0, el)
		})
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		sz := input.ReadInt32()
		if sz == 0 {
			return
		}
		for i := 0; i < int(sz); i++ {
			key, err0 := cache.cli.marsh.unmarshall(ctx, input)
			if err0 != nil {
				err = err0
				return
			}
			value, err0 := cache.cli.marsh.unmarshall(ctx, input)
			if err0 != nil {
				err = err0
				return
			}
			ret = append(ret, KeyValue{key, value})
		}
	})
	return ret, err
}

func (cache *cacheImpl) PutAll(ctx context.Context, keysAndValues ...KeyValue) error {
	lenData := len(keysAndValues)
	if lenData == 0 {
		return nil
	}
	var err error
	cache.cli.ch.Send(ctx, opCachePutAll, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = writeCollection(output, keysAndValues, func(output0 BinaryWriter, el KeyValue) error {
			if el.Key == nil {
				return fmt.Errorf("nil key")
			}
			if el.Value == nil {
				return fmt.Errorf("nil value")
			}
			var err0 error
			err0 = cache.cli.marsh.marshal(ctx, output0, el.Key)
			if err0 != nil {
				return err0
			}
			err0 = cache.cli.marsh.marshal(ctx, output0, el.Value)
			if err0 != nil {
				return err0
			}
			return nil
		})
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

func (cache *cacheImpl) ReplaceIfEquals(ctx context.Context, key interface{}, oldValue interface{}, newValue interface{}) (bool, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	if oldValue == nil {
		return false, fmt.Errorf("nil oldValue")
	}
	if newValue == nil {
		return false, fmt.Errorf("nil newValue")
	}
	var ret bool
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheReplaceIfEquals, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, oldValue)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, newValue)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) Replace(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	if value == nil {
		return false, fmt.Errorf("nil value")
	}
	var ret bool
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheReplace, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, value)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) Remove(ctx context.Context, key interface{}) (bool, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	var ret bool
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheRemoveKey, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) GetAndRemove(ctx context.Context, key interface{}) (interface{}, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	var ret interface{} = nil
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheGetAndRemove, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret, err = cache.cli.marsh.unmarshall(ctx, input)
	})
	return ret, err
}

func (cache *cacheImpl) RemoveIfEquals(ctx context.Context, key interface{}, oldValue interface{}) (bool, error) {
	if key == nil {
		return false, fmt.Errorf("nil key")
	}
	if oldValue == nil {
		return false, fmt.Errorf("nil oldValue")
	}
	var ret bool
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheRemoveIfEquals, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, oldValue)
		if err != nil {
			return err
		}
		return nil
	}, func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		ret = input.ReadBool()
	})
	return ret, err
}

func (cache *cacheImpl) RemoveKeys(ctx context.Context, keys ...interface{}) error {
	if len(keys) == 0 {
		return nil
	}
	var err error
	cache.cli.ch.Send(ctx, opCacheRemoveKeys, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = writeCollection(output, keys, func(output0 BinaryWriter, el interface{}) error {
			return cache.cli.marsh.marshal(ctx, output0, el)
		})
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

func (cache *cacheImpl) RemoveAll(ctx context.Context) error {
	var err error
	cache.cli.ch.Send(ctx, opCacheRemoveAll, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
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

func (cache *cacheImpl) Clear(ctx context.Context, key interface{}) error {
	if key == nil {
		return fmt.Errorf("nil key")
	}
	var err error = nil
	cache.cli.ch.Send(ctx, opCacheClearKey, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = cache.cli.marsh.marshal(ctx, output, key)
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

func (cache *cacheImpl) ClearKeys(ctx context.Context, keys ...interface{}) error {
	if len(keys) == 0 {
		return nil
	}
	var err error
	cache.cli.ch.Send(ctx, opCacheClearKeys, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
		if err != nil {
			return err
		}
		err = writeCollection(output, keys, func(output0 BinaryWriter, el interface{}) error {
			return cache.cli.marsh.marshal(ctx, output0, el)
		})
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

func (cache *cacheImpl) ClearAll(ctx context.Context) error {
	var err error
	cache.cli.ch.Send(ctx, opCacheClear, func(output BinaryWriter) error {
		err = cache.writeCacheInfo(cache.cli.ch.ProtocolContext(), output)
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

func (cache *cacheImpl) Name() string {
	return cache.name
}

func (cache *cacheImpl) WithExpirePolicy(creation time.Duration, access time.Duration, update time.Duration) Cache {
	return &cacheImpl{
		cli: cache.cli,
		expiryPolicy: &expirePolicyImpl{
			creation: creation,
			access:   access,
			update:   update,
		},
		name: cache.name,
		id:   cache.id,
	}
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
		for _, peekMode := range peekModes {
			output.WriteInt8(int8(peekMode))
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

func writeCollection[T any](output BinaryWriter, values []T, valueWriter func(output BinaryWriter, value T) error) error {
	colSz := len(values)
	output.WriteInt32(int32(colSz))
	if colSz == 0 {
		return nil
	}
	var err error
	for _, value := range values {
		err = valueWriter(output, value)
		if err != nil {
			break
		}
	}
	return err
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
