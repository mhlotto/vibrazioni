package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

// RemovePaper resolves a paper selector and removes its complete catalog-owned
// directory. It returns the removed record for user-facing confirmation.
func RemovePaper(catalogPath, selector string) (model.Paper, error) {
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	papers, err := ListPapers(catalogPath, ListOptions{})
	if err != nil {
		return model.Paper{}, err
	}
	updatedAt := time.Now().UTC()
	for _, candidate := range papers {
		if candidate.ID == paper.ID {
			continue
		}
		kept := candidate.Relationships[:0]
		for _, relationship := range candidate.Relationships {
			if relationship.PaperID != paper.ID {
				kept = append(kept, relationship)
			}
		}
		if len(kept) == len(candidate.Relationships) {
			continue
		}
		candidate.Relationships = kept
		candidateUpdatedAt := updatedAt
		if !candidateUpdatedAt.After(candidate.UpdatedAt) {
			candidateUpdatedAt = candidate.UpdatedAt.Add(time.Nanosecond)
		}
		candidate.UpdatedAt = candidateUpdatedAt
		if err := writePaperRecord(catalogPath, candidate); err != nil {
			return model.Paper{}, fmt.Errorf("remove inbound relationship from %s: %w", candidate.ID, err)
		}
	}
	directory := filepath.Join(catalogPath, PapersDirectory, paper.ID)
	if err := os.RemoveAll(directory); err != nil {
		return model.Paper{}, fmt.Errorf("remove paper %s: %w", paper.ID, err)
	}
	return paper, nil
}
