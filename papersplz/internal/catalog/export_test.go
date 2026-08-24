package catalog

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestExportMetadataIncludesAllMetadataInDeterministicOrder(t *testing.T) {
	catalogPath := newTestCatalog(t)
	timestamp := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	zoo := testInspectionPaper("aaaa00000000", "Zoo", []string{"Zoe"}, []string{"to-read"})
	alpha := testInspectionPaper("bbbb00000000", "Alpha", []string{"Alice", "Bob"}, []string{"math", "topology"})
	alpha.Source = "Journal"
	alpha.SourceURL = "https://example.org/alpha.pdf"
	alpha.Review = &model.Review{Text: "Strong result", CreatedAt: timestamp, UpdatedAt: timestamp}
	alpha.Comments = []model.Comment{{ID: "cccc0000", Text: "Check proof", CreatedAt: timestamp, UpdatedAt: timestamp}}
	writeCatalogPaper(t, catalogPath, zoo)
	writeCatalogPaper(t, catalogPath, alpha)

	catalogFile := filepath.Join(catalogPath, store.CatalogFilename)
	recordFile := filepath.Join(catalogPath, PapersDirectory, alpha.ID, store.RecordFilename)
	catalogBefore, err := os.ReadFile(catalogFile)
	if err != nil {
		t.Fatal(err)
	}
	recordBefore, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatal(err)
	}

	export, err := ExportMetadata(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if export.Catalog.Name != "Test Catalog" {
		t.Fatalf("catalog = %#v", export.Catalog)
	}
	if !reflect.DeepEqual(export.Papers, []model.Paper{alpha, zoo}) {
		t.Fatalf("papers = %#v, want Alpha then Zoo with complete metadata", export.Papers)
	}

	catalogAfter, err := os.ReadFile(catalogFile)
	if err != nil {
		t.Fatal(err)
	}
	recordAfter, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(catalogBefore, catalogAfter) || !bytes.Equal(recordBefore, recordAfter) {
		t.Fatal("ExportMetadata modified catalog metadata")
	}
}

func TestExportMetadataEmptyCatalogUsesEmptyPaperList(t *testing.T) {
	export, err := ExportMetadata(newTestCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if export.Papers == nil || len(export.Papers) != 0 {
		t.Fatalf("papers = %#v, want non-nil empty list", export.Papers)
	}
}
