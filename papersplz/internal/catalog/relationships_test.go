package catalog

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

func TestRelationshipCRUDAndDerivedInverses(t *testing.T) {
	catalogPath := newTestCatalog(t)
	a := testInspectionPaper("aaaa00000000", "A", nil, nil)
	b := testInspectionPaper("bbbb00000000", "B", nil, nil)
	c := testInspectionPaper("cccc00000000", "C", nil, nil)
	writeCatalogPaper(t, catalogPath, a)
	writeCatalogPaper(t, catalogPath, b)
	writeCatalogPaper(t, catalogPath, c)

	addedAt := a.UpdatedAt.Add(time.Hour)
	updated, err := AddRelationship(catalogPath, "aaaa", " CITES ", "bbbb", addedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.UpdatedAt.Equal(addedAt) || !reflect.DeepEqual(updated.Relationships, []model.Relationship{{Type: model.RelationshipCites, PaperID: b.ID}}) {
		t.Fatalf("updated paper = %#v", updated)
	}
	storedB, err := LoadPaper(catalogPath, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storedB.Relationships) != 0 {
		t.Fatalf("inverse was redundantly stored: %#v", storedB.Relationships)
	}

	fromA, err := ListRelationships(catalogPath, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	fromB, err := ListRelationships(catalogPath, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromA, []ListedRelationship{{Type: model.RelationshipCites, PaperID: b.ID, Title: b.Title}}) {
		t.Fatalf("A relationships = %#v", fromA)
	}
	if !reflect.DeepEqual(fromB, []ListedRelationship{{Type: model.RelationshipCitedBy, PaperID: a.ID, Title: a.Title}}) {
		t.Fatalf("B relationships = %#v", fromB)
	}

	unchanged, err := AddRelationship(catalogPath, b.ID, model.RelationshipCitedBy, a.ID, addedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged.UpdatedAt.Equal(b.UpdatedAt) || len(unchanged.Relationships) != 0 {
		t.Fatalf("inverse duplicate changed B: %#v", unchanged)
	}

	removedAt := addedAt.Add(2 * time.Hour)
	if _, err := RemoveRelationship(catalogPath, b.ID, model.RelationshipCitedBy, a.ID, removedAt); err != nil {
		t.Fatal(err)
	}
	storedA, err := LoadPaper(catalogPath, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storedA.Relationships) != 0 || !storedA.UpdatedAt.Equal(removedAt) {
		t.Fatalf("stored A after inverse removal = %#v", storedA)
	}
}

func TestRelationshipValidationAndOrdering(t *testing.T) {
	catalogPath := newTestCatalog(t)
	a := testInspectionPaper("aaaa00000000", "A", nil, nil)
	b := testInspectionPaper("bbbb00000000", "B", nil, nil)
	c := testInspectionPaper("cccc00000000", "C", nil, nil)
	writeCatalogPaper(t, catalogPath, a)
	writeCatalogPaper(t, catalogPath, b)
	writeCatalogPaper(t, catalogPath, c)

	if _, err := AddRelationship(catalogPath, a.ID, "mentions", b.ID, time.Now()); err == nil || !strings.Contains(err.Error(), "unknown relationship type") {
		t.Fatalf("invalid type error = %v", err)
	}
	if _, err := AddRelationship(catalogPath, a.ID, model.RelationshipCites, a.ID, time.Now()); err == nil || !strings.Contains(err.Error(), "cannot relate to itself") {
		t.Fatalf("self relationship error = %v", err)
	}
	if _, err := AddRelationship(catalogPath, a.ID, model.RelationshipSupersedes, c.ID, a.UpdatedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := AddRelationship(catalogPath, a.ID, model.RelationshipRelatedTo, b.ID, a.UpdatedAt.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	listed, err := ListRelationships(catalogPath, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []ListedRelationship{
		{Type: model.RelationshipRelatedTo, PaperID: b.ID, Title: b.Title},
		{Type: model.RelationshipSupersedes, PaperID: c.ID, Title: c.Title},
	}
	if !reflect.DeepEqual(listed, want) {
		t.Fatalf("relationships = %#v, want %#v", listed, want)
	}
}
