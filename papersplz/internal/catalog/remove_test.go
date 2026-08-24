package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

func TestRemovePaperDeletesCatalogCopyAndPreservesSource(t *testing.T) {
	catalogPath := newTestCatalog(t)
	sourcePath := filepath.Join(t.TempDir(), "original.pdf")
	contents := []byte("original source contents\n")
	if err := os.WriteFile(sourcePath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	paper, err := AddLocal(catalogPath, sourcePath, AddOptions{Title: "Remove Me"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	paperDirectory := filepath.Join(catalogPath, PapersDirectory, paper.ID)
	if err := os.WriteFile(filepath.Join(paperDirectory, "catalog-owned-extra"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := RemovePaper(catalogPath, paper.ID[:8])
	if err != nil {
		t.Fatalf("RemovePaper() error = %v", err)
	}
	if removed.ID != paper.ID {
		t.Fatalf("removed id = %q, want %q", removed.ID, paper.ID)
	}
	if _, err := os.Stat(paperDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("paper directory still exists: %v", err)
	}
	got, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("original source missing: %v", err)
	}
	if string(got) != string(contents) {
		t.Fatalf("original source = %q, want %q", got, contents)
	}
}

func TestRemovePaperCleansInboundRelationships(t *testing.T) {
	catalogPath := newTestCatalog(t)
	removed := testInspectionPaper("aaaa00000000", "Removed", nil, nil)
	remaining := testInspectionPaper("bbbb00000000", "Remaining", nil, nil)
	remaining.Relationships = []model.Relationship{{Type: model.RelationshipCites, PaperID: removed.ID}}
	writeCatalogPaper(t, catalogPath, removed)
	writeCatalogPaper(t, catalogPath, remaining)

	if _, err := RemovePaper(catalogPath, removed.ID); err != nil {
		t.Fatal(err)
	}
	stored, err := LoadPaper(catalogPath, remaining.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Relationships) != 0 || !stored.UpdatedAt.After(remaining.UpdatedAt) {
		t.Fatalf("remaining paper = %#v", stored)
	}
}
