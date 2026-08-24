package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestLoadPaperUsesPrefixResolution(t *testing.T) {
	catalogPath := newTestCatalog(t)
	writeCatalogPaper(t, catalogPath, testInspectionPaper("a81f32c991b7", "Paper", []string{"Alice"}, []string{"math"}))

	paper, err := LoadPaper(catalogPath, "a81f32")
	if err != nil {
		t.Fatalf("LoadPaper() error = %v", err)
	}
	if paper.ID != "a81f32c991b7" {
		t.Fatalf("LoadPaper() id = %q", paper.ID)
	}
	if _, err := LoadPaper(catalogPath, "ffff"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("LoadPaper() missing error = %v", err)
	}
}

func TestListPapersOrdersAndFilters(t *testing.T) {
	catalogPath := newTestCatalog(t)
	for _, paper := range []model.Paper{
		testInspectionPaper("aaaa00000000", "Zoo", []string{"Carol Example"}, []string{"physics"}),
		testInspectionPaper("bbbb00000000", "alpha", []string{"Alice Smith"}, []string{"math"}),
		testInspectionPaper("cccc00000000", "Beta", []string{"Bob Jones", "Alice Cooper"}, []string{"math", "topology"}),
	} {
		writeCatalogPaper(t, catalogPath, paper)
	}

	tests := []struct {
		name    string
		options ListOptions
		want    []string
	}{
		{name: "title order", want: []string{"alpha", "Beta", "Zoo"}},
		{name: "tag normalized exact match", options: ListOptions{Tag: " MATH "}, want: []string{"alpha", "Beta"}},
		{name: "author case insensitive substring", options: ListOptions{Author: "ALICE"}, want: []string{"alpha", "Beta"}},
		{name: "combined filters", options: ListOptions{Tag: "topology", Author: "cooper"}, want: []string{"Beta"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			papers, err := ListPapers(catalogPath, tt.options)
			if err != nil {
				t.Fatal(err)
			}
			titles := make([]string, len(papers))
			for i, paper := range papers {
				titles[i] = paper.Title
			}
			if !reflect.DeepEqual(titles, tt.want) {
				t.Fatalf("titles = %v, want %v", titles, tt.want)
			}
		})
	}
}

func TestDocumentPath(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("a81f32c991b7", "Paper", nil, nil)
	writeCatalogPaper(t, catalogPath, paper)
	want := filepath.Join(catalogPath, PapersDirectory, paper.ID, paper.File.Name)
	got, err := DocumentPath(catalogPath, "a81f32")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("DocumentPath() = %q, want %q", got, want)
	}
}

func TestGetInfoEmptyAndPopulated(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		catalogPath := newTestCatalog(t)
		info, err := GetInfo(catalogPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Metadata.Name != "Test Catalog" || info.Path != catalogPath || info.PaperCount != 0 || info.TagCount != 0 || info.LastAdded != nil {
			t.Fatalf("GetInfo() = %#v", info)
		}
	})

	t.Run("populated counts distinct tags and latest addition", func(t *testing.T) {
		catalogPath := newTestCatalog(t)
		firstAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
		lastAt := time.Date(2026, 8, 25, 9, 30, 0, 0, time.UTC)
		first := testInspectionPaper("aaaa00000000", "First", nil, []string{"math", "topology"})
		first.AddedAt, first.UpdatedAt = firstAt, firstAt
		last := testInspectionPaper("bbbb00000000", "Last", nil, []string{"math", "physics"})
		last.AddedAt, last.UpdatedAt = lastAt, lastAt
		writeCatalogPaper(t, catalogPath, last)
		writeCatalogPaper(t, catalogPath, first)

		info, err := GetInfo(catalogPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.PaperCount != 2 || info.TagCount != 3 || info.LastAdded == nil || !info.LastAdded.Equal(lastAt) {
			t.Fatalf("GetInfo() = %#v", info)
		}
	})
}

func testInspectionPaper(id, title string, authors, tags []string) model.Paper {
	timestamp := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	return model.Paper{
		SchemaVersion: model.SchemaVersion,
		ID:            id,
		Title:         title,
		Authors:       authors,
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         "paper.pdf",
			OriginalName: title + ".pdf",
			Size:         8,
			SHA256:       strings.Repeat("a", 64),
		},
		Tags:     tags,
		Comments: []model.Comment{},
	}
}

func writeCatalogPaper(t *testing.T, catalogPath string, paper model.Paper) {
	t.Helper()
	directory := filepath.Join(catalogPath, PapersDirectory, paper.ID)
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, paper.File.Name), []byte("document"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.WritePaper(filepath.Join(directory, store.RecordFilename), paper); err != nil {
		t.Fatal(err)
	}
}
