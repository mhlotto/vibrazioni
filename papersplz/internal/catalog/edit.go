package catalog

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

type EditOptions struct {
	Title      *string
	Authors    []string
	AuthorsSet bool
	Source     *string
	SourceURL  *string
}

// EditPaper changes only explicitly selected descriptive metadata fields.
func EditPaper(catalogPath, selector string, options EditOptions, updatedAt time.Time) (model.Paper, error) {
	if options.Title == nil && !options.AuthorsSet && options.Source == nil && options.SourceURL == nil {
		return model.Paper{}, errors.New("at least one metadata field is required")
	}
	if options.Title != nil {
		title := strings.TrimSpace(*options.Title)
		if title == "" {
			return model.Paper{}, errors.New("paper title is required")
		}
		options.Title = &title
	}
	if options.AuthorsSet {
		options.Authors = append([]string{}, options.Authors...)
		for i := range options.Authors {
			options.Authors[i] = strings.TrimSpace(options.Authors[i])
			if options.Authors[i] == "" {
				return model.Paper{}, fmt.Errorf("author %d is empty", i)
			}
		}
	}
	if options.Source != nil {
		source := strings.TrimSpace(*options.Source)
		options.Source = &source
	}
	if options.SourceURL != nil {
		sourceURL := strings.TrimSpace(*options.SourceURL)
		options.SourceURL = &sourceURL
	}

	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	if options.Title != nil {
		paper.Title = *options.Title
	}
	if options.AuthorsSet {
		paper.Authors = options.Authors
	}
	if options.Source != nil {
		paper.Source = *options.Source
	}
	if options.SourceURL != nil {
		paper.SourceURL = *options.SourceURL
	}
	paper.UpdatedAt = updatedAt.UTC()
	if err := writePaperRecord(catalogPath, &paper); err != nil {
		return model.Paper{}, fmt.Errorf("write paper metadata: %w", err)
	}
	return paper, nil
}
