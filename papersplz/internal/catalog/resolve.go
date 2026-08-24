package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

// ResolvePaperID resolves an exact paper ID or unambiguous prefix within a
// valid catalog.
func ResolvePaperID(catalogPath, selector string) (string, error) {
	if _, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename)); err != nil {
		return "", fmt.Errorf("read catalog: %w", err)
	}
	entries, err := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
	if err != nil {
		return "", fmt.Errorf("read papers directory: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !identity.Valid(entry.Name()) {
			return "", fmt.Errorf("invalid paper directory id %q", entry.Name())
		}
		ids = append(ids, entry.Name())
	}
	resolved, err := identity.Resolve(selector, ids)
	if err != nil {
		return "", fmt.Errorf("resolve paper id: %w", err)
	}
	return resolved, nil
}
