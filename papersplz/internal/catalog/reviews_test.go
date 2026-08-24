package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestSetShowAndRemoveReview(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Reviewed Paper", nil, nil)
	writeCatalogPaper(t, catalogPath, paper)
	createdAt := paper.UpdatedAt.Add(time.Hour)

	updated, err := SetReview(catalogPath, "a81f32", "First review\n", createdAt)
	if err != nil {
		t.Fatalf("SetReview() error = %v", err)
	}
	if updated.Review == nil || updated.Review.Text != "First review\n" {
		t.Fatalf("review after set = %#v", updated.Review)
	}
	if !updated.Review.CreatedAt.Equal(createdAt) || !updated.Review.UpdatedAt.Equal(createdAt) || !updated.UpdatedAt.Equal(createdAt) {
		t.Fatalf("timestamps after set: paper=%v review=%#v", updated.UpdatedAt, updated.Review)
	}

	replacedAt := createdAt.Add(time.Hour)
	updated, err = SetReview(catalogPath, paper.ID, "Replacement", replacedAt)
	if err != nil {
		t.Fatalf("SetReview() replacement error = %v", err)
	}
	if !updated.Review.CreatedAt.Equal(createdAt) {
		t.Fatalf("replacement created_at = %v, want %v", updated.Review.CreatedAt, createdAt)
	}
	if !updated.Review.UpdatedAt.Equal(replacedAt) || !updated.UpdatedAt.Equal(replacedAt) {
		t.Fatalf("replacement timestamps: paper=%v review=%#v", updated.UpdatedAt, updated.Review)
	}

	shown, err := ShowReview(catalogPath, "a81f32")
	if err != nil || shown.Review == nil || shown.Review.Text != "Replacement" {
		t.Fatalf("ShowReview() = %#v, %v", shown.Review, err)
	}

	removedAt := replacedAt.Add(time.Hour)
	removed, err := RemoveReview(catalogPath, "a81f32", removedAt)
	if err != nil {
		t.Fatalf("RemoveReview() error = %v", err)
	}
	if removed.Review != nil || !removed.UpdatedAt.Equal(removedAt) {
		t.Fatalf("paper after remove = %#v", removed)
	}
	stored, err := store.ReadPaper(filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Review != nil || !stored.UpdatedAt.Equal(removedAt) {
		t.Fatalf("stored paper after remove = %#v", stored)
	}
	if _, err := ShowReview(catalogPath, paper.ID); !errors.Is(err, ErrReviewNotFound) {
		t.Fatalf("ShowReview() error = %v, want ErrReviewNotFound", err)
	}
}

func TestEmptyReviewDoesNotMutateRecord(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Reviewed Paper", nil, nil)
	writeCatalogPaper(t, catalogPath, paper)
	recordPath := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	before, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SetReview(catalogPath, paper.ID, " \n\t", paper.UpdatedAt.Add(time.Hour)); err == nil {
		t.Fatal("SetReview() accepted empty text")
	}
	after, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("empty review changed record.json")
	}
}

func TestRemoveMissingReviewIsIdempotent(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Reviewed Paper", nil, nil)
	writeCatalogPaper(t, catalogPath, paper)
	got, err := RemoveReview(catalogPath, paper.ID, paper.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdatedAt.Equal(paper.UpdatedAt) {
		t.Fatalf("updated_at changed to %v", got.UpdatedAt)
	}
}
