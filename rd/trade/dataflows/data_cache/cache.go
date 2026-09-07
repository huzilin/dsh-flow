package data_cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huzilin/trade/dataflows/config"
	"github.com/pkg/errors"
)

const (
	defaultCacheDir   = "./data_cache"
	defaultExpireDays = 7
)

type Cache struct {
	cacheDir   string
	expireDays int
}

func NewCache() *Cache {
	cfg := config.Get()
	cacheDir := cfg.DataVendors.CoreStockAPIS
	if cacheDir == "" {
		cacheDir = defaultCacheDir
	}
	return &Cache{
		cacheDir:   cacheDir,
		expireDays: defaultExpireDays,
	}
}

func NewCacheWithOptions(cacheDir string, expireDays int) *Cache {
	if cacheDir == "" {
		cacheDir = defaultCacheDir
	}
	if expireDays < 0 {
		expireDays = defaultExpireDays
	}
	return &Cache{
		cacheDir:   cacheDir,
		expireDays: expireDays,
	}
}

func (c *Cache) GetCacheDir() string {
	return c.cacheDir
}

func (c *Cache) Key(symbol, market, source, startDate, endDate string) string {
	key := fmt.Sprintf("%s-%s-%s-%s-%s", symbol, market, source, startDate, endDate)
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (c *Cache) FilePath(key string) string {
	return filepath.Join(c.cacheDir, key+".csv")
}

func (c *Cache) Exists(key string) bool {
	path := c.FilePath(key)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (c *Cache) IsExpired(key string) bool {
	path := c.FilePath(key)
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	expireTime := info.ModTime().AddDate(0, 0, c.expireDays)
	return time.Now().After(expireTime)
}

func (c *Cache) Read(key string) ([]byte, error) {
	path := c.FilePath(key)
	return os.ReadFile(path)
}

func (c *Cache) Write(key string, data []byte) error {
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create cache directory %s", c.cacheDir)
	}
	path := c.FilePath(key)
	return errors.Wrapf(os.WriteFile(path, data, 0644), "failed to write cache file %s", path)
}

func (c *Cache) Delete(key string) error {
	path := c.FilePath(key)
	return os.Remove(path)
}

func (c *Cache) Clear() error {
	return os.RemoveAll(c.cacheDir)
}

func (c *Cache) ClearExpired() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".csv") {
			continue
		}

		path := filepath.Join(c.cacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		expireTime := info.ModTime().AddDate(0, 0, c.expireDays)
		if now.After(expireTime) {
			os.Remove(path)
		}
	}

	return nil
}

func (c *Cache) GetOrFetch(key string, fetcher func() ([]byte, error)) ([]byte, error) {
	if c.Exists(key) && !c.IsExpired(key) {
		data, err := c.Read(key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read cached data for key %s", key)
		}
		return data, nil
	}

	data, err := fetcher()
	if err != nil {
		return nil, errors.Wrapf(err, "fetcher failed for key %s", key)
	}

	if err := c.Write(key, data); err != nil {
		return data, nil
	}

	return data, nil
}
