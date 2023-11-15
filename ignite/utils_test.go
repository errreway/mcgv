package ignite

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCacheId(t *testing.T) {
	assert.Equal(t, int32(1544803905), CacheId("default"))
	assert.Equal(t, int32(1482644790), CacheId("myCache"))
	assert.Equal(t, int32(-1671596858), CacheId("ABCDEFGHIJKL"))
	assert.Equal(t, int32(-1172872323), CacheId("Cache1234567890"))
	assert.Equal(t, int32(1449161031), CacheId("Cache-123"))
}
