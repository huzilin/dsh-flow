package data_cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_CacheKey(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache", 7)
	key1 := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")
	key2 := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")
	key3 := cache.Key("000002", "A", "akshare", "2024-01-01", "2024-01-10")

	assert.Equal(t, key1, key2)
	assert.NotEqual(t, key1, key3)
	assert.Len(t, key1, 32)
}

func TestUnit_CacheFilePath(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache", 7)
	key := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")
	path := cache.FilePath(key)

	assert.Equal(t, filepath.Join("/tmp/test_cache", key+".csv"), path)
}

func TestUnit_CacheExists(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache_exists", 7)
	defer os.RemoveAll("/tmp/test_cache_exists")

	key := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")

	assert.False(t, cache.Exists(key))

	err := cache.Write(key, []byte("test data"))
	assert.NoError(t, err)

	assert.True(t, cache.Exists(key))
}

func TestUnit_CacheReadWrite(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache_rw", 7)
	defer os.RemoveAll("/tmp/test_cache_rw")

	key := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")
	data := []byte("test data content")

	err := cache.Write(key, data)
	assert.NoError(t, err)

	readData, err := cache.Read(key)
	assert.NoError(t, err)
	assert.Equal(t, data, readData)
}

func TestUnit_CacheIsExpired(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache_expire", 1)
	defer os.RemoveAll("/tmp/test_cache_expire")

	key := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")

	err := cache.Write(key, []byte("test data"))
	assert.NoError(t, err)

	assert.False(t, cache.IsExpired(key))

	path := cache.FilePath(key)
	pastTime := time.Now().Add(-2 * 24 * time.Hour)
	err = os.Chtimes(path, pastTime, pastTime)
	assert.NoError(t, err)

	assert.True(t, cache.IsExpired(key))
}

func TestUnit_CacheDelete(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache_delete", 7)
	defer os.RemoveAll("/tmp/test_cache_delete")

	key := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")

	err := cache.Write(key, []byte("test data"))
	assert.NoError(t, err)
	assert.True(t, cache.Exists(key))

	err = cache.Delete(key)
	assert.NoError(t, err)
	assert.False(t, cache.Exists(key))
}

func TestUnit_CacheGetOrFetch(t *testing.T) {
	cache := NewCacheWithOptions("/tmp/test_cache_fetch", 7)
	defer os.RemoveAll("/tmp/test_cache_fetch")

	key := cache.Key("000001", "A", "akshare", "2024-01-01", "2024-01-10")
	fetchCount := 0

	fetcher := func() ([]byte, error) {
		fetchCount++
		return []byte("fetched data"), nil
	}

	data1, err := cache.GetOrFetch(key, fetcher)
	assert.NoError(t, err)
	assert.Equal(t, []byte("fetched data"), data1)
	assert.Equal(t, 1, fetchCount)

	data2, err := cache.GetOrFetch(key, fetcher)
	assert.NoError(t, err)
	assert.Equal(t, []byte("fetched data"), data2)
	assert.Equal(t, 1, fetchCount)
}
