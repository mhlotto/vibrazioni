package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestEditPaperChangesSelectedMetadataAndPreservesRecordIdentity(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Original", []string{"Alice"}, []string{"topology"})
	paper.Source = "Old Journal"
	paper.SourceURL = "https://old.example/paper"
	timestamp := paper.UpdatedAt
	paper.Review = &model.Review{Text: "Review", CreatedAt: timestamp, UpdatedAt: timestamp}
	paper.Comments = []model.Comment{{ID: "abcd0000", Text: "Note", CreatedAt: timestamp, UpdatedAt: timestamp}}
	writeCatalogPaper(t, catalogPath, paper)
	documentPath := filepath.Join(catalogPath, PapersDirectory, paper.ID, paper.File.Name)
	documentBefore, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}

	title := "  Corrected Title  "
	source := "  New Journal  "
	sourceURL := "  https://new.example/paper  "
	updatedAt := timestamp.Add(time.Hour)
	updated, err := EditPaper(catalogPath, "a81f32", EditOptions{
		Title: stringPointer(title), Authors: []string{" Bob ", "Carol"}, AuthorsSet: true,
		Source: stringPointer(source), SourceURL: stringPointer(sourceURL),
	}, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Corrected Title" || !reflect.DeepEqual(updated.Authors, []string{"Bob", "Carol"}) || updated.Source != "New Journal" || updated.SourceURL != "https://new.example/paper" {
		t.Fatalf("edited metadata = %#v", updated)
	}
	if updated.ID != paper.ID || !updated.AddedAt.Equal(paper.AddedAt) || updated.File != paper.File || !reflect.DeepEqual(updated.Tags, paper.Tags) || !reflect.DeepEqual(updated.Review, paper.Review) || !reflect.DeepEqual(updated.Comments, paper.Comments) {
		t.Fatalf("edit changed preserved fields\nbefore: %#v\nafter:  %#v", paper, updated)
	}
	if !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated_at = %v, want %v", updated.UpdatedAt, updatedAt)
	}
	stored, err := store.ReadPaper(filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored, updated) {
		t.Fatalf("stored paper = %#v, want %#v", stored, updated)
	}
	documentAfter, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(documentAfter, documentBefore) {
		t.Fatal("edit changed stored document")
	}
}

func TestEditPaperPartialEditAndExplicitClearing(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Original", []string{"Alice"}, []string{"math"})
	paper.Source = "Journal"
	paper.SourceURL = "https://example.org/paper"
	writeCatalogPaper(t, catalogPath, paper)

	title := "Replacement"
	updated, err := EditPaper(catalogPath, paper.ID, EditOptions{Title: &title}, paper.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != title || !reflect.DeepEqual(updated.Authors, paper.Authors) || updated.Source != paper.Source || updated.SourceURL != paper.SourceURL {
		t.Fatalf("partial edit = %#v", updated)
	}

	empty := ""
	updated, err = EditPaper(catalogPath, paper.ID, EditOptions{Source: &empty, SourceURL: &empty}, updated.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Source != "" || updated.SourceURL != "" || updated.Title != title || !reflect.DeepEqual(updated.Authors, paper.Authors) {
		t.Fatalf("clearing edit = %#v", updated)
	}

	clearedAt := updated.UpdatedAt.Add(time.Hour)
	updated, err = EditPaper(catalogPath, paper.ID, EditOptions{Authors: []string{}, AuthorsSet: true}, clearedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Authors == nil || len(updated.Authors) != 0 || updated.Title != title || updated.Source != "" || updated.SourceURL != "" || !updated.UpdatedAt.Equal(clearedAt) {
		t.Fatalf("clear authors edit = %#v", updated)
	}
}

func TestEditPaperRejectsMissingInvalidAndAmbiguousSelection(t *testing.T) {
	catalogPath := newTestCatalog(t)
	for _, id := range []string{"abcd0000", "abcd1111"} {
		writeCatalogPaper(t, catalogPath, testInspectionPaper(id, id, nil, nil))
	}
	title := "Changed"
	if _, err := EditPaper(catalogPath, "abcd", EditOptions{Title: &title}, time.Now()); !errors.Is(err, identity.ErrAmbiguous) {
		t.Fatalf("ambiguous edit error = %v", err)
	}
	if _, err := EditPaper(catalogPath, "not-hex", EditOptions{Title: &title}, time.Now()); !errors.Is(err, identity.ErrInvalid) {
		t.Fatalf("invalid edit error = %v", err)
	}
	if _, err := EditPaper(catalogPath, "abcd0000", EditOptions{}, time.Now()); err == nil {
		t.Fatal("empty edit succeeded")
	}
	empty := "  "
	if _, err := EditPaper(catalogPath, "abcd0000", EditOptions{Title: &empty}, time.Now()); err == nil {
		t.Fatal("empty title edit succeeded")
	}
	unchanged, err := LoadPaper(catalogPath, "abcd0000")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Title != "abcd0000" {
		t.Fatalf("failed edits changed title to %q", unchanged.Title)
	}
}

func stringPointer(value string) *string { return &value }
