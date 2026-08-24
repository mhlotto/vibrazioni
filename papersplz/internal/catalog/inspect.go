package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

type ListOptions struct {
	Tag     string
	Author  string
	Sort    string
	Reverse bool
	Limit   int
}

const (
	ListSortTitle  = "title"
	ListSortAdded  = "added"
	ListSortAuthor = "author"
)

type Info struct {
	Metadata   model.Catalog
	Path       string
	PaperCount int
	TagCount   int
	LastAdded  *time.Time
}

// GetInfo returns catalog-level metadata and summary counts without modifying
// the catalog.
func GetInfo(catalogPath string) (Info, error) {
	metadata, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename))
	if err != nil {
		return Info{}, fmt.Errorf("read catalog: %w", err)
	}
	papers, err := ListPapers(catalogPath, ListOptions{})
	if err != nil {
		return Info{}, err
	}
	tags := make(map[string]struct{})
	var lastAdded *time.Time
	for _, paper := range papers {
		for _, tag := range paper.Tags {
			tags[tag] = struct{}{}
		}
		if lastAdded == nil || paper.AddedAt.After(*lastAdded) {
			addedAt := paper.AddedAt
			lastAdded = &addedAt
		}
	}
	return Info{
		Metadata:   metadata,
		Path:       catalogPath,
		PaperCount: len(papers),
		TagCount:   len(tags),
		LastAdded:  lastAdded,
	}, nil
}

// LoadPaper resolves selector and loads its validated record.
func LoadPaper(catalogPath, selector string) (model.Paper, error) {
	id, err := ResolvePaperID(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	paper, err := store.ReadPaper(filepath.Join(catalogPath, PapersDirectory, id, store.RecordFilename))
	if err != nil {
		return model.Paper{}, fmt.Errorf("read paper %s: %w", id, err)
	}
	if paper.ID != id {
		return model.Paper{}, fmt.Errorf("paper id %s does not match directory %s", paper.ID, id)
	}
	return paper, nil
}

// ListPapers loads validated records, applies metadata filters, and orders and
// limits the result. The zero-value options retain title ordering with no limit.
func ListPapers(catalogPath string, options ListOptions) ([]model.Paper, error) {
	sortBy := strings.ToLower(strings.TrimSpace(options.Sort))
	if sortBy == "" {
		sortBy = ListSortTitle
	}
	switch sortBy {
	case ListSortTitle, ListSortAdded, ListSortAuthor:
	default:
		return nil, fmt.Errorf("unknown list sort %q (want title, added, or author)", options.Sort)
	}
	if options.Limit < 0 {
		return nil, errors.New("list limit must not be negative")
	}
	if _, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename)); err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	papersPath := filepath.Join(catalogPath, PapersDirectory)
	entries, err := os.ReadDir(papersPath)
	if err != nil {
		return nil, fmt.Errorf("read papers directory: %w", err)
	}
	tag := strings.ToLower(strings.TrimSpace(options.Tag))
	if tag != "" {
		var err error
		tag, err = model.NormalizeTag(tag)
		if err != nil {
			return nil, err
		}
	}
	author := strings.ToLower(strings.TrimSpace(options.Author))
	papers := make([]model.Paper, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		paper, err := store.ReadPaper(filepath.Join(papersPath, entry.Name(), store.RecordFilename))
		if err != nil {
			return nil, fmt.Errorf("read paper %s: %w", entry.Name(), err)
		}
		if paper.ID != entry.Name() {
			return nil, fmt.Errorf("paper id %s does not match directory %s", paper.ID, entry.Name())
		}
		if tag != "" && !hasTag(paper.Tags, tag) {
			continue
		}
		if author != "" && !hasAuthor(paper.Authors, author) {
			continue
		}
		papers = append(papers, paper)
	}
	sort.Slice(papers, func(i, j int) bool {
		comparison := compareListedPapers(papers[i], papers[j], sortBy)
		if options.Reverse {
			return comparison > 0
		}
		return comparison < 0
	})
	if options.Limit > 0 && len(papers) > options.Limit {
		papers = papers[:options.Limit]
	}
	return papers, nil
}

func compareListedPapers(left, right model.Paper, sortBy string) int {
	switch sortBy {
	case ListSortAdded:
		if left.AddedAt.Before(right.AddedAt) {
			return -1
		}
		if left.AddedAt.After(right.AddedAt) {
			return 1
		}
	case ListSortAuthor:
		if len(left.Authors) == 0 && len(right.Authors) != 0 {
			return 1
		}
		if len(left.Authors) != 0 && len(right.Authors) == 0 {
			return -1
		}
		if comparison := compareFolded(strings.Join(left.Authors, "\x00"), strings.Join(right.Authors, "\x00")); comparison != 0 {
			return comparison
		}
	}
	if comparison := compareFolded(left.Title, right.Title); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.ID, right.ID)
}

func compareFolded(left, right string) int {
	if comparison := strings.Compare(strings.ToLower(left), strings.ToLower(right)); comparison != 0 {
		return comparison
	}
	return strings.Compare(left, right)
}

// DocumentPath returns the path recorded for a paper's catalog-owned copy.
func DocumentPath(catalogPath, selector string) (string, error) {
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return "", err
	}
	documentPath := filepath.Join(catalogPath, PapersDirectory, paper.ID, paper.File.Name)
	info, err := os.Stat(documentPath)
	if err != nil {
		return "", fmt.Errorf("inspect stored document: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("stored document is not a regular file")
	}
	return documentPath, nil
}

func hasTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if tag == wanted {
			return true
		}
	}
	return false
}

func hasAuthor(authors []string, wanted string) bool {
	for _, author := range authors {
		if strings.Contains(strings.ToLower(author), wanted) {
			return true
		}
	}
	return false
}
