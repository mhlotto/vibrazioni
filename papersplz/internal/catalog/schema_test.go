package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func TestOrdinarySchemaVersionOneCatalogRemainsReadableAndWritable(t *testing.T) {
	catalogPath := newTestCatalog(t)
	paper := testInspectionPaper("aaaa00000000", "Original", []string{"Alice"}, []string{"to-read"})
	writeCatalogPaper(t, catalogPath, paper)
	setPaperSchema(t, catalogPath, paper.ID, model.PaperSchemaVersion1)
	setCatalogSchema(t, catalogPath, model.CatalogSchemaVersion1)

	papers, err := ListPapers(catalogPath, ListOptions{})
	if err != nil || len(papers) != 1 || papers[0].SchemaVersion != model.PaperSchemaVersion1 {
		t.Fatalf("ListPapers() = %#v, %v", papers, err)
	}
	title := "Edited"
	edited, err := EditPaper(catalogPath, paper.ID, EditOptions{Title: &title}, paper.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if edited.SchemaVersion != model.PaperSchemaVersion1 || edited.Title != title {
		t.Fatalf("edited paper = %#v", edited)
	}
	metadata := readTestCatalog(t, catalogPath)
	if metadata.SchemaVersion != model.CatalogSchemaVersion1 {
		t.Fatalf("ordinary edit upgraded catalog to v%d", metadata.SchemaVersion)
	}
	source := filepath.Join(t.TempDir(), "new.pdf")
	if err := os.WriteFile(source, []byte("new v1 paper"), 0o644); err != nil {
		t.Fatal(err)
	}
	added, err := AddLocal(catalogPath, source, AddOptions{Title: "Added to v1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if added.SchemaVersion != model.PaperSchemaVersion1 {
		t.Fatalf("paper added to v1 catalog has schema %d", added.SchemaVersion)
	}
}

func TestNewCatalogAndAddedPaperUseSchemaVersionTwo(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := Initialize(catalogPath, "New", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := readTestCatalog(t, catalogPath).SchemaVersion; got != model.CatalogSchemaVersion2 {
		t.Fatalf("new catalog schema = %d", got)
	}
	source := filepath.Join(t.TempDir(), "paper.pdf")
	if err := os.WriteFile(source, []byte("schema two paper"), 0o644); err != nil {
		t.Fatal(err)
	}
	paper, err := AddLocal(catalogPath, source, AddOptions{Title: "Paper"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if paper.SchemaVersion != model.PaperSchemaVersion2 {
		t.Fatalf("new paper schema = %d", paper.SchemaVersion)
	}
}

func TestRelationshipAddUpgradesSchemaOneCatalogBeforeStoringEdge(t *testing.T) {
	catalogPath, first, second := schemaOneCatalogWithTwoPapers(t)
	updated, err := AddRelationship(catalogPath, first.ID, model.RelationshipCites, second.ID, first.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if readTestCatalog(t, catalogPath).SchemaVersion != model.CatalogSchemaVersion2 {
		t.Fatal("catalog was not upgraded to schema version 2")
	}
	for _, id := range []string{first.ID, second.ID} {
		paper, err := LoadPaper(catalogPath, id)
		if err != nil {
			t.Fatal(err)
		}
		if paper.SchemaVersion != model.PaperSchemaVersion2 {
			t.Fatalf("paper %s schema = %d", id, paper.SchemaVersion)
		}
	}
	if !reflect.DeepEqual(updated.Relationships, []model.Relationship{{Type: model.RelationshipCites, PaperID: second.ID}}) {
		t.Fatalf("relationships = %#v", updated.Relationships)
	}
}

func TestRelationshipMigrationOrderingAndRestart(t *testing.T) {
	catalogPath, first, second := schemaOneCatalogWithTwoPapers(t)
	wantErr := errors.New("interrupted paper upgrade")
	paperWriteCalled := false
	err := ensureRelationshipSchema(
		catalogPath,
		store.WriteCatalog,
		func(string, model.Paper) error {
			paperWriteCalled = true
			if got := readTestCatalog(t, catalogPath).SchemaVersion; got != model.CatalogSchemaVersion2 {
				t.Fatalf("paper upgrade began while catalog schema = %d", got)
			}
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) || !paperWriteCalled {
		t.Fatalf("migration error = %v, paperWriteCalled = %v", err, paperWriteCalled)
	}
	if readTestCatalog(t, catalogPath).SchemaVersion != model.CatalogSchemaVersion2 {
		t.Fatal("interrupted migration did not leave catalog protected at v2")
	}
	for _, id := range []string{first.ID, second.ID} {
		paper, readErr := LoadPaper(catalogPath, id)
		if readErr != nil || paper.SchemaVersion != model.PaperSchemaVersion1 {
			t.Fatalf("mixed state paper %s = %#v, %v", id, paper, readErr)
		}
	}

	if err := EnsureRelationshipSchema(catalogPath); err != nil {
		t.Fatalf("resume migration: %v", err)
	}
	if err := EnsureRelationshipSchema(catalogPath); err != nil {
		t.Fatalf("idempotent migration: %v", err)
	}
	for _, id := range []string{first.ID, second.ID} {
		paper, err := LoadPaper(catalogPath, id)
		if err != nil || paper.SchemaVersion != model.PaperSchemaVersion2 {
			t.Fatalf("resumed paper %s = %#v, %v", id, paper, err)
		}
	}
}

func TestPrereleaseSchemaOneRelationshipsSurviveMetadataEdit(t *testing.T) {
	catalogPath, first, second := schemaOneCatalogWithTwoPapers(t)
	first.Relationships = []model.Relationship{{Type: model.RelationshipCites, PaperID: second.ID}}
	writeRawPaper(t, catalogPath, first)

	loaded, err := LoadPaper(catalogPath, first.ID)
	if err != nil || len(loaded.Relationships) != 1 {
		t.Fatalf("pre-release record = %#v, %v", loaded, err)
	}
	title := "Preserved relationship"
	edited, err := EditPaper(catalogPath, first.ID, EditOptions{Title: &title}, first.UpdatedAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if edited.SchemaVersion != model.PaperSchemaVersion2 || !reflect.DeepEqual(edited.Relationships, first.Relationships) {
		t.Fatalf("edited record = %#v", edited)
	}
	if readTestCatalog(t, catalogPath).SchemaVersion != model.CatalogSchemaVersion2 {
		t.Fatal("pre-release relationship edit did not upgrade catalog")
	}
}

func TestMigrationRejectsFuturePaperBeforeChangingCatalog(t *testing.T) {
	catalogPath, first, _ := schemaOneCatalogWithTwoPapers(t)
	first.SchemaVersion = model.CurrentPaperSchemaVersion + 1
	writeRawPaper(t, catalogPath, first)
	if err := EnsureRelationshipSchema(catalogPath); !errors.Is(err, model.ErrUnsupportedSchema) {
		t.Fatalf("EnsureRelationshipSchema() error = %v", err)
	}
	if readTestCatalog(t, catalogPath).SchemaVersion != model.CatalogSchemaVersion1 {
		t.Fatal("future paper schema changed catalog before being rejected")
	}
}

func schemaOneCatalogWithTwoPapers(t *testing.T) (string, model.Paper, model.Paper) {
	t.Helper()
	catalogPath := newTestCatalog(t)
	first := testInspectionPaper("aaaa00000000", "First", nil, nil)
	second := testInspectionPaper("bbbb00000000", "Second", nil, nil)
	writeCatalogPaper(t, catalogPath, first)
	writeCatalogPaper(t, catalogPath, second)
	setPaperSchema(t, catalogPath, first.ID, model.PaperSchemaVersion1)
	setPaperSchema(t, catalogPath, second.ID, model.PaperSchemaVersion1)
	first.SchemaVersion = model.PaperSchemaVersion1
	second.SchemaVersion = model.PaperSchemaVersion1
	setCatalogSchema(t, catalogPath, model.CatalogSchemaVersion1)
	return catalogPath, first, second
}

func setCatalogSchema(t *testing.T, catalogPath string, version int) {
	t.Helper()
	metadata := readTestCatalog(t, catalogPath)
	metadata.SchemaVersion = version
	if err := store.WriteCatalog(filepath.Join(catalogPath, store.CatalogFilename), metadata); err != nil {
		t.Fatal(err)
	}
}

func readTestCatalog(t *testing.T, catalogPath string) model.Catalog {
	t.Helper()
	metadata, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename))
	if err != nil {
		t.Fatal(err)
	}
	return metadata
}

func setPaperSchema(t *testing.T, catalogPath, paperID string, version int) {
	t.Helper()
	path := filepath.Join(catalogPath, PapersDirectory, paperID, store.RecordFilename)
	paper, err := store.ReadPaper(path)
	if err != nil {
		t.Fatal(err)
	}
	paper.SchemaVersion = version
	if err := store.WritePaper(path, paper); err != nil {
		t.Fatal(err)
	}
}

func writeRawPaper(t *testing.T, catalogPath string, paper model.Paper) {
	t.Helper()
	encoded, err := json.MarshalIndent(paper, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	path := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}
