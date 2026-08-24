package catalog

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

type ListedRelationship struct {
	Type    string
	PaperID string
	Title   string
}

func NormalizeRelationshipType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if err := model.ValidateRelationshipType(normalized); err != nil {
		return "", fmt.Errorf("relationship type: %w", err)
	}
	return normalized, nil
}

func AddRelationship(catalogPath, paperSelector, relationshipType, otherSelector string, updatedAt time.Time) (model.Paper, error) {
	paper, err := LoadPaper(catalogPath, paperSelector)
	if err != nil {
		return model.Paper{}, err
	}
	other, err := LoadPaper(catalogPath, otherSelector)
	if err != nil {
		return model.Paper{}, fmt.Errorf("resolve related paper: %w", err)
	}
	if paper.ID == other.ID {
		return model.Paper{}, errors.New("a paper cannot relate to itself")
	}
	relationshipType, err = NormalizeRelationshipType(relationshipType)
	if err != nil {
		return model.Paper{}, err
	}
	listed, err := ListRelationships(catalogPath, paper.ID)
	if err != nil {
		return model.Paper{}, err
	}
	for _, relationship := range listed {
		if relationship.Type == relationshipType && relationship.PaperID == other.ID {
			return paper, nil
		}
	}
	paper.Relationships = append(paper.Relationships, model.Relationship{Type: relationshipType, PaperID: other.ID})
	sortRelationships(paper.Relationships)
	paper.UpdatedAt = updatedAt.UTC()
	if err := writePaperRecord(catalogPath, paper); err != nil {
		return model.Paper{}, err
	}
	return paper, nil
}

func ListRelationships(catalogPath, paperSelector string) ([]ListedRelationship, error) {
	paper, err := LoadPaper(catalogPath, paperSelector)
	if err != nil {
		return nil, err
	}
	papers, err := ListPapers(catalogPath, ListOptions{})
	if err != nil {
		return nil, err
	}
	titles := make(map[string]string, len(papers))
	for _, candidate := range papers {
		titles[candidate.ID] = candidate.Title
	}
	listed := make([]ListedRelationship, 0)
	seen := make(map[string]struct{})
	appendRelationship := func(relationshipType, paperID string) {
		key := relationshipType + "\x00" + paperID
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		listed = append(listed, ListedRelationship{Type: relationshipType, PaperID: paperID, Title: titles[paperID]})
	}
	for _, relationship := range paper.Relationships {
		appendRelationship(relationship.Type, relationship.PaperID)
	}
	for _, candidate := range papers {
		for _, relationship := range candidate.Relationships {
			if relationship.PaperID == paper.ID {
				appendRelationship(inverseRelationshipType(relationship.Type), candidate.ID)
			}
		}
	}
	sort.Slice(listed, func(i, j int) bool {
		if listed[i].Type != listed[j].Type {
			return listed[i].Type < listed[j].Type
		}
		return listed[i].PaperID < listed[j].PaperID
	})
	return listed, nil
}

func RemoveRelationship(catalogPath, paperSelector, relationshipType, otherSelector string, updatedAt time.Time) (model.Paper, error) {
	paper, err := LoadPaper(catalogPath, paperSelector)
	if err != nil {
		return model.Paper{}, err
	}
	other, err := LoadPaper(catalogPath, otherSelector)
	if err != nil {
		return model.Paper{}, fmt.Errorf("resolve related paper: %w", err)
	}
	relationshipType, err = NormalizeRelationshipType(relationshipType)
	if err != nil {
		return model.Paper{}, err
	}
	if removeStoredRelationship(&paper, relationshipType, other.ID) {
		paper.UpdatedAt = updatedAt.UTC()
		if err := writePaperRecord(catalogPath, paper); err != nil {
			return model.Paper{}, err
		}
		return paper, nil
	}
	if removeStoredRelationship(&other, inverseRelationshipType(relationshipType), paper.ID) {
		other.UpdatedAt = updatedAt.UTC()
		if err := writePaperRecord(catalogPath, other); err != nil {
			return model.Paper{}, err
		}
		return paper, nil
	}
	return model.Paper{}, fmt.Errorf("relationship %s to %s not found", relationshipType, other.ID)
}

func inverseRelationshipType(relationshipType string) string {
	switch relationshipType {
	case model.RelationshipCites:
		return model.RelationshipCitedBy
	case model.RelationshipCitedBy:
		return model.RelationshipCites
	case model.RelationshipSupersedes:
		return model.RelationshipSupersededBy
	case model.RelationshipSupersededBy:
		return model.RelationshipSupersedes
	default:
		return model.RelationshipRelatedTo
	}
}

func removeStoredRelationship(paper *model.Paper, relationshipType, paperID string) bool {
	for i, relationship := range paper.Relationships {
		if relationship.Type == relationshipType && relationship.PaperID == paperID {
			paper.Relationships = append(paper.Relationships[:i], paper.Relationships[i+1:]...)
			return true
		}
	}
	return false
}

func sortRelationships(relationships []model.Relationship) {
	sort.Slice(relationships, func(i, j int) bool {
		if relationships[i].Type != relationships[j].Type {
			return relationships[i].Type < relationships[j].Type
		}
		return relationships[i].PaperID < relationships[j].PaperID
	})
}

func writePaperRecord(catalogPath string, paper model.Paper) error {
	path := filepath.Join(catalogPath, PapersDirectory, paper.ID, store.RecordFilename)
	if err := store.WritePaper(path, paper); err != nil {
		return fmt.Errorf("write paper %s: %w", paper.ID, err)
	}
	return nil
}
