package model

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func validCatalog() Catalog {
	return Catalog{
		SchemaVersion: SchemaVersion,
		Name:          "Mathematics",
		Description:   "Papers and notes",
		CreatedAt:     time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC),
	}
}

func validPaper() Paper {
	created := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	return Paper{
		SchemaVersion: SchemaVersion,
		ID:            "a81f32c991b7",
		Title:         "Some Interesting Paper",
		Authors:       []string{"Alice Smith", "Bob Jones"},
		Source:        "Journal of Interesting Things",
		SourceURL:     "https://example.org/paper.pdf",
		AddedAt:       created,
		UpdatedAt:     updated,
		File: File{
			Name:         "paper.pdf",
			OriginalName: "smith-interesting-paper.pdf",
			Size:         481231,
			SHA256:       strings.Repeat("a", 64),
		},
		Tags: []string{"topology", "homotopy"},
		Review: &Review{
			Text:      "A useful review.",
			CreatedAt: created,
			UpdatedAt: updated,
		},
		Comments: []Comment{{
			ID:        "c91e84f2",
			Text:      "A working note.",
			CreatedAt: created,
			UpdatedAt: updated,
		}},
	}
}

func TestValidateCatalog(t *testing.T) {
	catalog := validCatalog()
	if err := ValidateCatalog(catalog); err != nil {
		t.Fatalf("ValidateCatalog() error = %v", err)
	}

	catalog.Name = ""
	if err := ValidateCatalog(catalog); err == nil {
		t.Fatal("ValidateCatalog() accepted a missing name")
	}
}

func TestValidatePaper(t *testing.T) {
	if err := ValidatePaper(validPaper()); err != nil {
		t.Fatalf("ValidatePaper() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Paper)
		want   string
	}{
		{name: "missing title", mutate: func(p *Paper) { p.Title = "" }, want: "title"},
		{name: "invalid id", mutate: func(p *Paper) { p.ID = "NOT-HEX" }, want: "id"},
		{name: "missing filename", mutate: func(p *Paper) { p.File.Name = "" }, want: "file name"},
		{name: "negative size", mutate: func(p *Paper) { p.File.Size = -1 }, want: "size"},
		{name: "invalid digest", mutate: func(p *Paper) { p.File.SHA256 = "abc" }, want: "sha256"},
		{name: "invalid tag", mutate: func(p *Paper) { p.Tags = []string{"Has Spaces"} }, want: "tag"},
		{name: "invalid comment", mutate: func(p *Paper) { p.Comments[0].Text = "" }, want: "comment"},
		{name: "updated before added", mutate: func(p *Paper) { p.UpdatedAt = p.AddedAt.Add(-time.Second) }, want: "precedes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paper := validPaper()
			tt.mutate(&paper)
			err := ValidatePaper(paper)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidatePaper() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestUnsupportedSchema(t *testing.T) {
	catalog := validCatalog()
	catalog.SchemaVersion = SchemaVersion + 1
	if err := ValidateCatalog(catalog); !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("ValidateCatalog() error = %v, want ErrUnsupportedSchema", err)
	}

	paper := validPaper()
	paper.SchemaVersion = SchemaVersion + 1
	if err := ValidatePaper(paper); !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("ValidatePaper() error = %v, want ErrUnsupportedSchema", err)
	}
}
