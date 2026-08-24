package catalog

import (
	"fmt"
	"path/filepath"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

// MetadataExport is a read-only snapshot of all catalog-managed metadata.
type MetadataExport struct {
	Catalog model.Catalog
	Papers  []model.Paper
}

// ExportMetadata reads and validates catalog metadata and paper records. Paper
// records are returned in the same deterministic title order as the default
// list command.
func ExportMetadata(catalogPath string) (MetadataExport, error) {
	metadata, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename))
	if err != nil {
		return MetadataExport{}, fmt.Errorf("read catalog: %w", err)
	}
	papers, err := ListPapers(catalogPath, ListOptions{})
	if err != nil {
		return MetadataExport{}, err
	}
	return MetadataExport{Catalog: metadata, Papers: papers}, nil
}
