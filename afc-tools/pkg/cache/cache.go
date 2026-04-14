package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrNotFound = errors.New("cache entry not found")
	ErrStale    = errors.New("cache entry is stale")
)

const DefaultMaxAge = 2 * time.Hour

type Entry struct {
	Key     string
	Path    string
	ModTime time.Time
	Size    int64
}

type Cache struct {
	dir string
}

func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	return filepath.Join(home, ".afctoolcache"), nil
}

func New(dir string) *Cache {
	return &Cache{dir: dir}
}

func NewDefault() (*Cache, error) {
	dir, err := DefaultDir()
	if err != nil {
		return nil, err
	}

	return New(dir), nil
}

func (c *Cache) Dir() string {
	return c.dir
}

func (c *Cache) Get(key string) ([]byte, error) {
	return c.GetIfFresh(key, DefaultMaxAge)
}

func (c *Cache) GetIfFresh(key string, maxAge time.Duration) ([]byte, error) {
	entry, err := c.Stat(key)
	if err != nil {
		return nil, err
	}

	if maxAge > 0 && time.Since(entry.ModTime) > maxAge {
		return nil, ErrStale
	}

	data, err := os.ReadFile(entry.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read cache entry %q: %w", key, err)
	}

	return data, nil
}

func (c *Cache) Stat(key string) (Entry, error) {
	path := c.pathForKey(key)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Entry{}, ErrNotFound
		}
		return Entry{}, fmt.Errorf("stat cache entry %q: %w", key, err)
	}

	return Entry{
		Key:     key,
		Path:    path,
		ModTime: info.ModTime(),
		Size:    info.Size(),
	}, nil
}

func (c *Cache) Put(key string, data []byte) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("create cache dir %q: %w", c.dir, err)
	}

	path := c.pathForKey(key)
	tmp, err := os.CreateTemp(c.dir, "cache-*")
	if err != nil {
		return fmt.Errorf("create temp cache file for %q: %w", key, err)
	}

	tmpName := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp cache file for %q: %w", key, err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp cache file for %q: %w", key, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("replace cache entry %q: %w", key, err)
	}

	return nil
}

func (c *Cache) Delete(key string) error {
	err := os.Remove(c.pathForKey(key))
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("delete cache entry %q: %w", key, err)
}

func (c *Cache) Invalidate(key string) error {
	return c.Delete(key)
}

func (c *Cache) pathForKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	name := hex.EncodeToString(sum[:]) + ".cache"
	return filepath.Join(c.dir, name)
}
