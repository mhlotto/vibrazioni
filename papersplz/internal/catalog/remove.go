package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

// RemovePaper resolves a paper selector and removes its complete catalog-owned
// directory. It returns the removed record for user-facing confirmation.
func RemovePaper(catalogPath, selector string) (model.Paper, error) {
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	directory := filepath.Join(catalogPath, PapersDirectory, paper.ID)
	if err := os.RemoveAll(directory); err != nil {
		return model.Paper{}, fmt.Errorf("remove paper %s: %w", paper.ID, err)
	}
	return paper, nil
}
