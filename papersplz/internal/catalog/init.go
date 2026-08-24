package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

var (
	ErrCatalogExists     = errors.New("papersplz catalog already exists")
	ErrDirectoryNotEmpty = errors.New("directory is not empty")
)

const PapersDirectory = "papers"

// Initialize creates a version 1 catalog at path. The path may be absent or
// may name an existing empty directory.
func Initialize(path, name, description string, createdAt time.Time) error {
	createdRoot, err := prepareEmptyDirectory(path)
	if err != nil {
		return err
	}

	papersPath := filepath.Join(path, PapersDirectory)
	if err := os.Mkdir(papersPath, 0o755); err != nil {
		if createdRoot {
			os.Remove(path)
		}
		return fmt.Errorf("create papers directory: %w", err)
	}

	metadata := model.Catalog{
		SchemaVersion: model.CurrentCatalogSchemaVersion,
		Name:          name,
		Description:   description,
		CreatedAt:     createdAt.UTC(),
	}
	metadataPath := filepath.Join(path, store.CatalogFilename)
	if err := store.WriteCatalog(metadataPath, metadata); err != nil {
		os.Remove(papersPath)
		if createdRoot {
			os.Remove(path)
		}
		return fmt.Errorf("write catalog metadata: %w", err)
	}
	return nil
}

func prepareEmptyDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return false, fmt.Errorf("create catalog directory: %w", err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect catalog path: %w", err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("catalog path is not a directory: %s", path)
	}

	if _, err := os.Stat(filepath.Join(path, store.CatalogFilename)); err == nil {
		return false, fmt.Errorf("%w: %s", ErrCatalogExists, path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect catalog metadata: %w", err)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("read catalog directory: %w", err)
	}
	if len(entries) != 0 {
		return false, fmt.Errorf("%w: %s", ErrDirectoryNotEmpty, path)
	}
	return false, nil
}
