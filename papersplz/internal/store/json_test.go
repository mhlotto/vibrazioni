package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

func testCatalog() model.Catalog {
	return model.Catalog{
		SchemaVersion: model.CurrentCatalogSchemaVersion,
		Name:          "Mathematics",
		Description:   "Papers and notes",
		CreatedAt:     time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC),
	}
}

func testPaper() model.Paper {
	timestamp := time.Date(2026, 8, 24, 15, 31, 0, 0, time.UTC)
	return model.Paper{
		SchemaVersion: model.CurrentPaperSchemaVersion,
		ID:            "a81f32c991b7",
		Title:         "Some Interesting Paper",
		Authors:       []string{"Alice Smith", "Bob Jones"},
		Source:        "Journal of Interesting Things",
		SourceURL:     "https://example.org/paper.pdf",
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         "paper.pdf",
			OriginalName: "smith-interesting-paper.pdf",
			Size:         481231,
			SHA256:       strings.Repeat("a", 64),
		},
		Tags:     []string{"topology", "homotopy"},
		Review:   nil,
		Comments: []model.Comment{},
	}
}

func TestCatalogRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), CatalogFilename)
	want := testCatalog()
	if err := WriteCatalog(path, want); err != nil {
		t.Fatalf("WriteCatalog() error = %v", err)
	}
	got, err := ReadCatalog(path)
	if err != nil {
		t.Fatalf("ReadCatalog() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadCatalog() = %#v, want %#v", got, want)
	}
	assertReadableJSON(t, path)
}

func TestPaperRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), RecordFilename)
	want := testPaper()
	if err := WritePaper(path, want); err != nil {
		t.Fatalf("WritePaper() error = %v", err)
	}
	got, err := ReadPaper(path)
	if err != nil {
		t.Fatalf("ReadPaper() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadPaper() = %#v, want %#v", got, want)
	}
	assertReadableJSON(t, path)
}

func TestReadsSchemaVersionOneAndTwo(t *testing.T) {
	directory := t.TempDir()
	for _, version := range []int{model.CatalogSchemaVersion1, model.CatalogSchemaVersion2} {
		catalog := testCatalog()
		catalog.SchemaVersion = version
		path := filepath.Join(directory, fmt.Sprintf("catalog-%d.json", version))
		if err := WriteCatalog(path, catalog); err != nil {
			t.Fatal(err)
		}
		if got, err := ReadCatalog(path); err != nil || got.SchemaVersion != version {
			t.Fatalf("ReadCatalog(v%d) = %#v, %v", version, got, err)
		}
	}
	for _, version := range []int{model.PaperSchemaVersion1, model.PaperSchemaVersion2} {
		paper := testPaper()
		paper.SchemaVersion = version
		path := filepath.Join(directory, fmt.Sprintf("paper-%d.json", version))
		if err := WritePaper(path, paper); err != nil {
			t.Fatal(err)
		}
		if got, err := ReadPaper(path); err != nil || got.SchemaVersion != version {
			t.Fatalf("ReadPaper(v%d) = %#v, %v", version, got, err)
		}
	}
}

func TestWriteRejectsSchemaVersionOneRelationships(t *testing.T) {
	path := filepath.Join(t.TempDir(), RecordFilename)
	paper := testPaper()
	paper.SchemaVersion = model.PaperSchemaVersion1
	paper.Relationships = []model.Relationship{{Type: model.RelationshipCites, PaperID: "bbbb0000"}}
	if err := WritePaper(path, paper); err == nil || !strings.Contains(err.Error(), "schema version 1 cannot store relationships") {
		t.Fatalf("WritePaper() error = %v", err)
	}
}

func TestReadRejectsMalformedAndStructurallyInvalidJSON(t *testing.T) {
	directory := t.TempDir()
	tests := []struct {
		name string
		data string
	}{
		{name: "malformed", data: `{"schema_version": 1`},
		{name: "multiple values", data: `{}` + "\n" + `{}`},
		{name: "missing required fields", data: `{"schema_version":1}`},
		{name: "bad timestamp", data: `{"schema_version":1,"name":"Catalog","created_at":"yesterday"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(directory, strings.ReplaceAll(tt.name, " ", "-")+".json")
			if err := os.WriteFile(path, []byte(tt.data), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadCatalog(path); err == nil {
				t.Fatal("ReadCatalog() accepted invalid JSON")
			}
		})
	}
}

func TestUnsupportedSchemaDoesNotReplaceExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), CatalogFilename)
	original := []byte("original metadata\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	catalog := testCatalog()
	catalog.SchemaVersion = model.CurrentCatalogSchemaVersion + 1
	err := WriteCatalog(path, catalog)
	if !errors.Is(err, model.ErrUnsupportedSchema) {
		t.Fatalf("WriteCatalog() error = %v, want ErrUnsupportedSchema", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("file after failed write = %q, want %q", got, original)
	}
}

func TestReadRejectsUnsupportedSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), CatalogFilename)
	data := `{"schema_version":3,"name":"Future","created_at":"2026-08-24T15:30:00Z"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadCatalog(path)
	if !errors.Is(err, model.ErrUnsupportedSchema) {
		t.Fatalf("ReadCatalog() error = %v, want ErrUnsupportedSchema", err)
	}
}

func TestReadRejectsUnsupportedPaperSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), RecordFilename)
	paper := testPaper()
	paper.SchemaVersion = model.CurrentPaperSchemaVersion + 1
	encoded, err := json.Marshal(paper)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPaper(path); !errors.Is(err, model.ErrUnsupportedSchema) {
		t.Fatalf("ReadPaper() error = %v, want ErrUnsupportedSchema", err)
	}
}

func TestAtomicReplaceFailureLeavesExistingFileIntact(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, CatalogFilename)
	original := []byte("complete original metadata\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("simulated write failure")
	err := atomicReplace(path, func(writer io.Writer) error {
		if _, err := io.WriteString(writer, "partial replacement"); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("atomicReplace() error = %v, want simulated failure", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("file after failed replacement = %q, want %q", got, original)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".papersplz-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func assertReadableJSON(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") || !strings.Contains(string(data), "\n  \"") {
		t.Fatalf("metadata is not indented, newline-terminated JSON:\n%s", data)
	}
}
