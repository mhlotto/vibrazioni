package catalog

import (
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

func TestCommentCRUDOrderingPrefixesAndTimestamps(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Commented Paper", nil, nil)
	writeCatalogPaper(t, catalogPath, paper)
	firstAt := paper.UpdatedAt.Add(time.Hour)

	updated, first, err := AddComment(catalogPath, "a81f32", "First", firstAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.ID) != 32 || first.Text != "First" || !first.CreatedAt.Equal(firstAt) || !first.UpdatedAt.Equal(firstAt) || !updated.UpdatedAt.Equal(firstAt) {
		t.Fatalf("first comment/paper = %#v / %#v", first, updated)
	}
	secondAt := firstAt.Add(time.Hour)
	_, second, err := AddComment(catalogPath, paper.ID, "Second", secondAt)
	if err != nil {
		t.Fatal(err)
	}

	_, comments, err := ListComments(catalogPath, paper.ID)
	if err != nil || len(comments) != 2 || comments[0].ID != first.ID || comments[1].ID != second.ID {
		t.Fatalf("ListComments() = %#v, %v", comments, err)
	}
	prefix := first.ID[:8]
	_, shown, err := ShowComment(catalogPath, "a81f32", prefix)
	if err != nil || shown.ID != first.ID {
		t.Fatalf("ShowComment(prefix) = %#v, %v", shown, err)
	}

	editedAt := secondAt.Add(time.Hour)
	updated, edited, err := EditComment(catalogPath, paper.ID, prefix, "Edited", editedAt)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Text != "Edited" || !edited.CreatedAt.Equal(firstAt) || !edited.UpdatedAt.Equal(editedAt) || !updated.UpdatedAt.Equal(editedAt) {
		t.Fatalf("edited comment/paper = %#v / %#v", edited, updated)
	}
	if updated.Comments[1].ID != second.ID || updated.Comments[1].Text != "Second" {
		t.Fatalf("edit affected second comment: %#v", updated.Comments[1])
	}

	removedAt := editedAt.Add(time.Hour)
	updated, removed, err := RemoveComment(catalogPath, paper.ID, second.ID[:8], removedAt)
	if err != nil {
		t.Fatal(err)
	}
	if removed.ID != second.ID || len(updated.Comments) != 1 || updated.Comments[0].ID != first.ID || !updated.UpdatedAt.Equal(removedAt) {
		t.Fatalf("remove result = %#v / %#v", removed, updated)
	}
}

func TestCommentOrderingHasDeterministicTieBreak(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Commented Paper", nil, nil)
	timestamp := paper.UpdatedAt.Add(time.Hour)
	paper.Comments = []model.Comment{
		{ID: "bbbb0000", Text: "Later ID", CreatedAt: timestamp, UpdatedAt: timestamp},
		{ID: "aaaa0000", Text: "Earlier ID", CreatedAt: timestamp, UpdatedAt: timestamp},
	}
	writeCatalogPaper(t, catalogPath, paper)
	_, comments, err := ListComments(catalogPath, paper.ID)
	if err != nil {
		t.Fatal(err)
	}
	if comments[0].ID != "aaaa0000" || comments[1].ID != "bbbb0000" {
		t.Fatalf("comments = %#v", comments)
	}
}

func TestAmbiguousCommentPrefixDoesNotMutate(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Commented Paper", nil, nil)
	timestamp := paper.UpdatedAt.Add(time.Hour)
	paper.Comments = []model.Comment{
		{ID: "abcd0000", Text: "One", CreatedAt: timestamp, UpdatedAt: timestamp},
		{ID: "abcd1111", Text: "Two", CreatedAt: timestamp, UpdatedAt: timestamp},
	}
	writeCatalogPaper(t, catalogPath, paper)
	if _, _, err := EditComment(catalogPath, paper.ID, "abcd", "Changed", timestamp.Add(time.Hour)); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("EditComment() error = %v", err)
	}
	stored, err := LoadPaper(catalogPath, paper.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Comments[0].Text != "One" || stored.Comments[1].Text != "Two" {
		t.Fatalf("ambiguous edit mutated comments: %#v", stored.Comments)
	}
}

func TestEmptyCommentTextIsRejected(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Commented Paper", nil, nil)
	writeCatalogPaper(t, catalogPath, paper)
	if _, _, err := AddComment(catalogPath, paper.ID, " \n", time.Now()); err == nil {
		t.Fatal("AddComment() accepted empty text")
	}
}
