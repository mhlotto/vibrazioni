package catalog

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

type TagUsage struct {
	Tag   string
	Count int
}

// ListTagUsage returns every distinct catalog tag ordered by usage count
// descending and then tag name ascending.
func ListTagUsage(catalogPath string) ([]TagUsage, error) {
	papers, err := ListPapers(catalogPath, ListOptions{})
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, paper := range papers {
		for _, tag := range paper.Tags {
			counts[tag]++
		}
	}
	usage := make([]TagUsage, 0, len(counts))
	for tag, count := range counts {
		usage = append(usage, TagUsage{Tag: tag, Count: count})
	}
	sort.Slice(usage, func(i, j int) bool {
		if usage[i].Count != usage[j].Count {
			return usage[i].Count > usage[j].Count
		}
		return usage[i].Tag < usage[j].Tag
	})
	return usage, nil
}

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
	if err := writePaperRecord(catalogPath, &paper); err != nil {
		return model.Paper{}, fmt.Errorf("write paper tags: %w", err)
	}
	return paper, nil
}
