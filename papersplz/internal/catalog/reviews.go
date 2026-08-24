package catalog

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

var ErrReviewNotFound = errors.New("paper has no review")

func SetReview(catalogPath, selector, text string, updatedAt time.Time) (model.Paper, error) {
	if strings.TrimSpace(text) == "" {
		return model.Paper{}, errors.New("review text is required")
	}
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	timestamp := updatedAt.UTC()
	createdAt := timestamp
	if paper.Review != nil {
		createdAt = paper.Review.CreatedAt
	}
	paper.Review = &model.Review{
		Text:      text,
		CreatedAt: createdAt,
		UpdatedAt: timestamp,
	}
	paper.UpdatedAt = timestamp
	if err := writeReviewPaper(catalogPath, &paper); err != nil {
		return model.Paper{}, err
	}
	return paper, nil
}

func ShowReview(catalogPath, selector string) (model.Paper, error) {
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	if paper.Review == nil {
		return model.Paper{}, ErrReviewNotFound
	}
	return paper, nil
}

func RemoveReview(catalogPath, selector string, updatedAt time.Time) (model.Paper, error) {
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, err
	}
	if paper.Review == nil {
		return paper, nil
	}
	paper.Review = nil
	paper.UpdatedAt = updatedAt.UTC()
	if err := writeReviewPaper(catalogPath, &paper); err != nil {
		return model.Paper{}, err
	}
	return paper, nil
}

func writeReviewPaper(catalogPath string, paper *model.Paper) error {
	if err := writePaperRecord(catalogPath, paper); err != nil {
		return fmt.Errorf("write paper review: %w", err)
	}
	return nil
}
