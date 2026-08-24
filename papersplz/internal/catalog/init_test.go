package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestInitializeCreatesCatalogLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "catalog")
	createdAt := time.Date(2026, 8, 24, 15, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	if err := Initialize(path, "Mathematics", "Papers and notes", createdAt); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != store.CatalogFilename || entries[1].Name() != PapersDirectory {
		t.Fatalf("catalog entries = %v, want catalog.json and papers", entryNames(entries))
	}
	info, err := os.Stat(filepath.Join(path, PapersDirectory))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatal("papers is not a directory")
	}

	metadata, err := store.ReadCatalog(filepath.Join(path, store.CatalogFilename))
	if err != nil {
		t.Fatalf("ReadCatalog() error = %v", err)
	}
	want := model.Catalog{
		SchemaVersion: model.CurrentCatalogSchemaVersion,
		Name:          "Mathematics",
		Description:   "Papers and notes",
		CreatedAt:     createdAt.UTC(),
	}
	if metadata != want {
		t.Fatalf("catalog metadata = %#v, want %#v", metadata, want)
	}
}

func TestInitializeAcceptsExistingEmptyDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(path, "Empty Start", "", time.Now()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}

func TestInitializeRefusesExistingCatalogWithoutOverwriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog")
	createdAt := time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC)
	if err := Initialize(path, "Original", "Keep this", createdAt); err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(path, store.CatalogFilename)
	before, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}

	err = Initialize(path, "Replacement", "Do not write", createdAt.Add(time.Hour))
	if !errors.Is(err, ErrCatalogExists) {
		t.Fatalf("Initialize() error = %v, want ErrCatalogExists", err)
	}
	after, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("repeated initialization changed catalog.json")
	}
}

func TestInitializeRefusesUnrelatedNonEmptyDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-catalog")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	unrelatedPath := filepath.Join(path, "notes.txt")
	if err := os.WriteFile(unrelatedPath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Initialize(path, "No", "", time.Now())
	if !errors.Is(err, ErrDirectoryNotEmpty) {
		t.Fatalf("Initialize() error = %v, want ErrDirectoryNotEmpty", err)
	}
	data, err := os.ReadFile(unrelatedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep me\n" {
		t.Fatal("initialization changed unrelated data")
	}
	if _, err := os.Stat(filepath.Join(path, store.CatalogFilename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("catalog.json unexpectedly exists: %v", err)
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}
