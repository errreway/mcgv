package ignite

type Cache interface {
	Name() string
}

type CacheImpl struct {
	cli *ClientImpl

	name *string
}

func (cache CacheImpl) Name() string {
	return *cache.name
}
