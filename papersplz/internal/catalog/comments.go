package catalog

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

func AddComment(catalogPath, selector, text string, createdAt time.Time) (model.Paper, model.Comment, error) {
	if strings.TrimSpace(text) == "" {
		return model.Paper{}, model.Comment{}, errors.New("comment text is required")
	}
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	id, err := identity.NewCommentID()
	if err != nil {
		return model.Paper{}, model.Comment{}, fmt.Errorf("generate comment id: %w", err)
	}
	for _, existing := range paper.Comments {
		if existing.ID == id {
			return model.Paper{}, model.Comment{}, errors.New("generated duplicate comment id")
		}
	}
	timestamp := createdAt.UTC()
	comment := model.Comment{ID: id, Text: text, CreatedAt: timestamp, UpdatedAt: timestamp}
	paper.Comments = append(paper.Comments, comment)
	paper.UpdatedAt = timestamp
	if err := writeCommentPaper(catalogPath, paper); err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	return paper, comment, nil
}

func ListComments(catalogPath, selector string) (model.Paper, []model.Comment, error) {
	paper, err := LoadPaper(catalogPath, selector)
	if err != nil {
		return model.Paper{}, nil, err
	}
	comments := append([]model.Comment(nil), paper.Comments...)
	sort.Slice(comments, func(i, j int) bool {
		if !comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
			return comments[i].CreatedAt.Before(comments[j].CreatedAt)
		}
		return comments[i].ID < comments[j].ID
	})
	return paper, comments, nil
}

func ShowComment(catalogPath, paperSelector, commentSelector string) (model.Paper, model.Comment, error) {
	paper, err := LoadPaper(catalogPath, paperSelector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	index, err := resolveCommentIndex(paper, commentSelector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	return paper, paper.Comments[index], nil
}

func EditComment(catalogPath, paperSelector, commentSelector, text string, updatedAt time.Time) (model.Paper, model.Comment, error) {
	if strings.TrimSpace(text) == "" {
		return model.Paper{}, model.Comment{}, errors.New("comment text is required")
	}
	paper, err := LoadPaper(catalogPath, paperSelector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	index, err := resolveCommentIndex(paper, commentSelector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	timestamp := updatedAt.UTC()
	paper.Comments[index].Text = text
	paper.Comments[index].UpdatedAt = timestamp
	paper.UpdatedAt = timestamp
	if err := writeCommentPaper(catalogPath, paper); err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	return paper, paper.Comments[index], nil
}

func RemoveComment(catalogPath, paperSelector, commentSelector string, updatedAt time.Time) (model.Paper, model.Comment, error) {
	paper, err := LoadPaper(catalogPath, paperSelector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	index, err := resolveCommentIndex(paper, commentSelector)
	if err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	removed := paper.Comments[index]
	paper.Comments = append(paper.Comments[:index], paper.Comments[index+1:]...)
	paper.UpdatedAt = updatedAt.UTC()
	if err := writeCommentPaper(catalogPath, paper); err != nil {
		return model.Paper{}, model.Comment{}, err
	}
	return paper, removed, nil
}

func resolveCommentIndex(paper model.Paper, selector string) (int, error) {
	id, err := model.ResolveCommentID(paper, selector)
	if err != nil {
		return 0, err
	}
	for i := range paper.Comments {
		if paper.Comments[i].ID == id {
			return i, nil
		}
	}
	return 0, errors.New("resolved comment is missing")
}

func writeCommentPaper(catalogPath string, paper model.Paper) error {
	path := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	if err := store.WritePaper(path, paper); err != nil {
		return fmt.Errorf("write paper comments: %w", err)
	}
	return nil
}
