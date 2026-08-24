package catalog

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestAddAndRemoveTags(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Tagged Paper", []string{"Alice"}, []string{"math"})
	writeCatalogPaper(t, catalogPath, paper)
	firstUpdate := paper.UpdatedAt.Add(time.Hour)

	updated, err := AddTags(catalogPath, "a81f32", []string{" Topology ", "MATH", "qft+notes"}, firstUpdate)
	if err != nil {
		t.Fatalf("AddTags() error = %v", err)
	}
	if !reflect.DeepEqual(updated.Tags, []string{"math", "topology", "qft+notes"}) {
		t.Fatalf("tags after add = %v", updated.Tags)
	}
	if !updated.UpdatedAt.Equal(firstUpdate) {
		t.Fatalf("updated_at = %v, want %v", updated.UpdatedAt, firstUpdate)
	}

	secondUpdate := firstUpdate.Add(time.Hour)
	updated, err = RemoveTags(catalogPath, paper.ID, []string{" MATH ", "missing"}, secondUpdate)
	if err != nil {
		t.Fatalf("RemoveTags() error = %v", err)
	}
	if !reflect.DeepEqual(updated.Tags, []string{"topology", "qft+notes"}) {
		t.Fatalf("tags after remove = %v", updated.Tags)
	}
	if !updated.UpdatedAt.Equal(secondUpdate) {
		t.Fatalf("updated_at = %v, want %v", updated.UpdatedAt, secondUpdate)
	}

	stored, err := store.ReadPaper(filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored.Tags, updated.Tags) || !stored.UpdatedAt.Equal(secondUpdate) {
		t.Fatalf("stored paper = %#v", stored)
	}
}

func TestInvalidTagDoesNotMutateRecord(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Tagged Paper", nil, []string{"math"})
	writeCatalogPaper(t, catalogPath, paper)
	recordPath := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	before, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := AddTags(catalogPath, paper.ID, []string{"bad tag"}, paper.UpdatedAt.Add(time.Hour)); err == nil {
		t.Fatal("AddTags() accepted an invalid tag")
	}
	after, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("invalid tag changed record.json")
	}
}

func TestIdempotentTagMutationPreservesUpdatedAt(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Tagged Paper", nil, []string{"math"})
	writeCatalogPaper(t, catalogPath, paper)

	unchanged, err := AddTags(catalogPath, paper.ID, []string{"MATH"}, paper.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged.UpdatedAt.Equal(paper.UpdatedAt) {
		t.Fatalf("idempotent add changed updated_at to %v", unchanged.UpdatedAt)
	}
	unchanged, err = RemoveTags(catalogPath, paper.ID, []string{"missing"}, paper.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged.UpdatedAt.Equal(paper.UpdatedAt) {
		t.Fatalf("idempotent remove changed updated_at to %v", unchanged.UpdatedAt)
	}
}

func TestSetReadingStatus(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Reading Paper", nil, []string{"topology", "unread", "read"})
	writeCatalogPaper(t, catalogPath, paper)

	statuses := []struct {
		name string
		tag  string
	}{
		{name: "unread", tag: "unread"},
		{name: "reading", tag: "reading"},
		{name: "read", tag: "read"},
	}
	updatedAt := paper.UpdatedAt
	for index, status := range statuses {
		updatedAt = updatedAt.Add(time.Hour)
		updated, err := SetReadingStatus(catalogPath, paper.ID[:6], status.tag, updatedAt)
		if err != nil {
			t.Fatalf("SetReadingStatus(%s) error = %v", status.name, err)
		}
		want := []string{"topology", status.tag}
		if !reflect.DeepEqual(updated.Tags, want) {
			t.Fatalf("SetReadingStatus(%s) tags = %v, want %v", status.name, updated.Tags, want)
		}
		if !updated.UpdatedAt.Equal(updatedAt) {
			t.Fatalf("SetReadingStatus(%s) updated_at = %v, want %v", status.name, updated.UpdatedAt, updatedAt)
		}
		if index == len(statuses)-1 {
			unchanged, err := SetReadingStatus(catalogPath, paper.ID, status.tag, updatedAt.Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			if !unchanged.UpdatedAt.Equal(updatedAt) {
				t.Fatalf("repeated mark changed updated_at to %v", unchanged.UpdatedAt)
			}
		}
	}
}

func TestSetReadingStatusRejectsUnknownTag(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Reading Paper", nil, []string{"topology"})
	writeCatalogPaper(t, catalogPath, paper)

	if _, err := SetReadingStatus(catalogPath, paper.ID, "finished", time.Now()); err == nil {
		t.Fatal("SetReadingStatus accepted an unknown status tag")
	}
	stored, err := LoadPaper(catalogPath, paper.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored.Tags, paper.Tags) {
		t.Fatalf("invalid status changed tags to %v", stored.Tags)
	}
}

func TestListTagUsageCountsAndOrders(t *testing.T) {
	catalogPath := newTestCatalog(t)
	papers := []struct {
		id   string
		tags []string
	}{
		{id: "aaaa0000", tags: []string{"topology", "to-read", "homotopy"}},
		{id: "bbbb0000", tags: []string{"topology", "to-read", "algebraic-topology"}},
		{id: "cccc0000", tags: []string{"topology", "homotopy", "algebraic-topology"}},
	}
	for _, fixture := range papers {
		writeCatalogPaper(t, catalogPath, testInspectionPaper(fixture.id, fixture.id, nil, fixture.tags))
	}

	got, err := ListTagUsage(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []TagUsage{
		{Tag: "topology", Count: 3},
		{Tag: "algebraic-topology", Count: 2},
		{Tag: "homotopy", Count: 2},
		{Tag: "to-read", Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListTagUsage() = %#v, want %#v", got, want)
	}
}

func TestListTagUsageEmptyCatalog(t *testing.T) {
	got, err := ListTagUsage(newTestCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("ListTagUsage() = %#v, want empty non-nil slice", got)
	}
}
