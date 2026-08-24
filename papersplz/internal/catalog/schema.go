package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func paperSchemaForCatalog(catalogSchema int) int {
	if catalogSchema == model.CatalogSchemaVersion1 {
		return model.PaperSchemaVersion1
	}
	return model.PaperSchemaVersion2
}

// EnsureRelationshipSchema upgrades a catalog before relationship metadata is
// written. catalog.json is raised first so a v1-only binary refuses the catalog
// throughout any partial paper-record migration. Repeating this operation
// resumes safely after interruption.
func EnsureRelationshipSchema(catalogPath string) error {
	return ensureRelationshipSchema(catalogPath, store.WriteCatalog, store.WritePaper)
}

func ensureRelationshipSchema(
	catalogPath string,
	writeCatalog func(string, model.Catalog) error,
	writePaper func(string, model.Paper) error,
) error {
	catalogFile := filepath.Join(catalogPath, store.CatalogFilename)
	metadata, err := store.ReadCatalog(catalogFile)
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	entries, err := os.ReadDir(filepath.Join(catalogPath, PapersDirectory))
	if err != nil {
		return fmt.Errorf("read papers directory: %w", err)
	}
	type migrationRecord struct {
		path  string
		paper model.Paper
	}
	records := make([]migrationRecord, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		recordPath := filepath.Join(catalogPath, PapersDirectory, entry.Name(), store.RecordFilename)
		paper, err := store.ReadPaper(recordPath)
		if err != nil {
			return fmt.Errorf("read paper %s during schema upgrade: %w", entry.Name(), err)
		}
		if paper.ID != entry.Name() {
			return fmt.Errorf("paper id %s does not match directory %s during schema upgrade", paper.ID, entry.Name())
		}
		records = append(records, migrationRecord{path: recordPath, paper: paper})
	}

	if metadata.SchemaVersion == model.CatalogSchemaVersion1 {
		metadata.SchemaVersion = model.CatalogSchemaVersion2
		if err := writeCatalog(catalogFile, metadata); err != nil {
			return fmt.Errorf("upgrade catalog schema: %w", err)
		}
	}
	for _, record := range records {
		if record.paper.SchemaVersion == model.PaperSchemaVersion2 {
			continue
		}
		record.paper.SchemaVersion = model.PaperSchemaVersion2
		if err := writePaper(record.path, record.paper); err != nil {
			return fmt.Errorf("upgrade paper %s schema: %w", record.paper.ID, err)
		}
	}
	return nil
}

// preparePaperForWrite keeps ordinary v1 mutations at v1, upgrades a selected
// paper in a v2 catalog, and resumes a migration when relationship data or a v2
// paper is discovered beneath a v1 catalog.
func preparePaperForWrite(catalogPath string, paper *model.Paper) error {
	metadata, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename))
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	needsCatalogUpgrade := metadata.SchemaVersion == model.CatalogSchemaVersion1 &&
		(paper.SchemaVersion == model.PaperSchemaVersion2 || len(paper.Relationships) != 0)
	if needsCatalogUpgrade {
		if err := EnsureRelationshipSchema(catalogPath); err != nil {
			return err
		}
		metadata.SchemaVersion = model.CatalogSchemaVersion2
	}
	if metadata.SchemaVersion == model.CatalogSchemaVersion2 {
		paper.SchemaVersion = model.PaperSchemaVersion2
	}
	return nil
}

func writePaperRecord(catalogPath string, paper *model.Paper) error {
	if err := preparePaperForWrite(catalogPath, paper); err != nil {
		return err
	}
	path := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	if err := store.WritePaper(path, *paper); err != nil {
		return fmt.Errorf("write paper %s: %w", paper.ID, err)
	}
	return nil
}
