package ignite

func CacheId(name string) int32 {
	strLen := len(name)

	if strLen == 1 {
		return 0
	}

	var hash int32 = 0

	for i := 0; i < strLen; i++ {
		hash = 31*hash + int32((name)[i])
	}

	return hash
}
