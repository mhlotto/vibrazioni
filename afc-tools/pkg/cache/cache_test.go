package cache

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestCachePutAndGetIfFresh(t *testing.T) {
	t.Parallel()

	c := New(t.TempDir())
	key := "results-and-fixtures"
	want := []byte("arsenal")

	if err := c.Put(key, want); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := c.GetIfFresh(key, time.Hour)
	if err != nil {
		t.Fatalf("GetIfFresh() error = %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("GetIfFresh() = %q, want %q", got, want)
	}
}

func TestCacheGetUsesDefaultMaxAge(t *testing.T) {
	t.Parallel()

	c := New(t.TempDir())
	key := "results-and-fixtures"
	want := []byte("arsenal")

	if err := c.Put(key, want); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("Get() = %q, want %q", got, want)
	}
}

func TestCacheGetIfFreshNotFound(t *testing.T) {
	t.Parallel()

	c := New(t.TempDir())

	_, err := c.GetIfFresh("missing", time.Hour)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetIfFresh() error = %v, want ErrNotFound", err)
	}
}

func TestCacheGetIfFreshStale(t *testing.T) {
	t.Parallel()

	c := New(t.TempDir())
	key := "fixtures"

	if err := c.Put(key, []byte("old")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	entry, err := c.Stat(key)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(entry.Path, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	_, err = c.GetIfFresh(key, time.Hour)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("GetIfFresh() error = %v, want ErrStale", err)
	}
}

func TestCacheDelete(t *testing.T) {
	t.Parallel()

	c := New(t.TempDir())
	key := "fixtures"

	if err := c.Put(key, []byte("data")); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	if err := c.Delete(key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := c.Stat(key)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat() error = %v, want ErrNotFound", err)
	}
}
