package catalog

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func AddTags(catalogPath, selector string, tags []string, updatedAt time.Time) (model.Paper, error) {
	normalized, err := requireTags(tags)
	if err != nil {
		return model.Paper{}, err
	}
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	existing := make(map[string]struct{}, len(paper.Tags)+len(normalized))
	for _, tag := range paper.Tags {
		existing[tag] = struct{}{}
	}
	changed := false
	for _, tag := range normalized {
		if _, exists := existing[tag]; exists {
			continue
		}
		paper.Tags = append(paper.Tags, tag)
		existing[tag] = struct{}{}
		changed = true
	}
	if !changed {
		return paper, nil
	}
	return writeTags(catalogPath, paper, updatedAt)
}

func RemoveTags(catalogPath, selector string, tags []string, updatedAt time.Time) (model.Paper, error) {
	normalized, err := requireTags(tags)
	if err != nil {
		return model.Paper{}, err
	}
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	remove := make(map[string]struct{}, len(normalized))
	for _, tag := range normalized {
		remove[tag] = struct{}{}
	}
	kept := make([]string, 0, len(paper.Tags))
	for _, tag := range paper.Tags {
		if _, found := remove[tag]; !found {
			kept = append(kept, tag)
		}
	}
	if len(kept) == len(paper.Tags) {
		return paper, nil
	}
	paper.Tags = kept
	return writeTags(catalogPath, paper, updatedAt)
}

func ListTags(catalogPath, selector string) (model.Paper, error) {
	return LoadPaper(catalogPath, selector)
}

func requireTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, errors.New("at least one tag is required")
	}
	normalized, err := model.NormalizeTags(tags)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func writeTags(catalogPath string, paper model.Paper, updatedAt time.Time) (model.Paper, error) {
	paper.UpdatedAt = updatedAt.UTC()
	path := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	if err := store.WritePaper(path, paper); err != nil {
		return model.Paper{}, fmt.Errorf("write paper tags: %w", err)
	}
	return paper, nil
}
