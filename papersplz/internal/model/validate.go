package model

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
)

var ErrUnsupportedSchema = errors.New("unsupported schema version")

var (
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const (
	RelationshipRelatedTo    = "related-to"
	RelationshipCites        = "cites"
	RelationshipCitedBy      = "cited-by"
	RelationshipSupersedes   = "supersedes"
	RelationshipSupersededBy = "superseded-by"
)

func ValidateCatalog(c Catalog) error {
	if err := validateSchema(c.SchemaVersion); err != nil {
		return err
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("catalog name is required")
	}
	if err := validateTime("catalog created_at", c.CreatedAt); err != nil {
		return err
	}
	return nil
}

func ValidatePaper(p Paper) error {
	if err := validateSchema(p.SchemaVersion); err != nil {
		return err
	}
	if !identity.Valid(p.ID) {
		return errors.New("paper id must be lowercase hexadecimal")
	}
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("paper title is required")
	}
	if err := validateTime("paper added_at", p.AddedAt); err != nil {
		return err
	}
	if err := validateTime("paper updated_at", p.UpdatedAt); err != nil {
		return err
	}
	if p.UpdatedAt.Before(p.AddedAt) {
		return errors.New("paper updated_at precedes added_at")
	}
	if err := validateFile(p.File); err != nil {
		return err
	}
	for i, author := range p.Authors {
		if strings.TrimSpace(author) == "" {
			return fmt.Errorf("author %d is empty", i)
		}
	}
	seenTags := make(map[string]struct{}, len(p.Tags))
	for _, tag := range p.Tags {
		normalized, err := NormalizeTag(tag)
		if err != nil {
			return err
		}
		if normalized != tag {
			return fmt.Errorf("stored tag %q is not normalized", tag)
		}
		if _, exists := seenTags[tag]; exists {
			return fmt.Errorf("duplicate tag %q", tag)
		}
		seenTags[tag] = struct{}{}
	}
	seenRelationships := make(map[string]struct{}, len(p.Relationships))
	for i, relationship := range p.Relationships {
		if err := ValidateRelationship(relationship); err != nil {
			return fmt.Errorf("relationship %d: %w", i, err)
		}
		if relationship.PaperID == p.ID {
			return fmt.Errorf("relationship %d references its own paper", i)
		}
		key := relationship.Type + "\x00" + relationship.PaperID
		if _, exists := seenRelationships[key]; exists {
			return fmt.Errorf("duplicate relationship %q to %s", relationship.Type, relationship.PaperID)
		}
		seenRelationships[key] = struct{}{}
	}
	if p.Review != nil {
		if err := validateReview(*p.Review); err != nil {
			return err
		}
	}
	seenComments := make(map[string]struct{}, len(p.Comments))
	for i, comment := range p.Comments {
		if err := validateComment(comment); err != nil {
			return fmt.Errorf("comment %d: %w", i, err)
		}
		if _, exists := seenComments[comment.ID]; exists {
			return fmt.Errorf("duplicate comment id %q", comment.ID)
		}
		seenComments[comment.ID] = struct{}{}
	}
	return nil
}

func ValidateRelationship(relationship Relationship) error {
	if err := ValidateRelationshipType(relationship.Type); err != nil {
		return err
	}
	if !identity.Valid(relationship.PaperID) {
		return errors.New("paper_id must be lowercase hexadecimal")
	}
	return nil
}

func ValidateRelationshipType(relationshipType string) error {
	switch relationshipType {
	case RelationshipRelatedTo, RelationshipCites, RelationshipCitedBy, RelationshipSupersedes, RelationshipSupersededBy:
	default:
		return fmt.Errorf("unknown relationship type %q", relationshipType)
	}
	return nil
}

func validateSchema(version int) error {
	if version == 0 {
		return errors.New("schema_version is required")
	}
	if version != SchemaVersion {
		return fmt.Errorf("%w: %d", ErrUnsupportedSchema, version)
	}
	return nil
}

func validateFile(file File) error {
	if file.Name == "" {
		return errors.New("file name is required")
	}
	if file.Name == "." || file.Name == ".." || filepath.Base(file.Name) != file.Name || strings.Contains(file.Name, `\`) {
		return errors.New("file name must not contain a directory path")
	}
	if file.OriginalName == "" {
		return errors.New("file original_name is required")
	}
	if file.Size < 0 {
		return errors.New("file size must not be negative")
	}
	if !sha256Pattern.MatchString(file.SHA256) {
		return errors.New("file sha256 must be 64 lowercase hexadecimal characters")
	}
	return nil
}

func validateReview(review Review) error {
	if strings.TrimSpace(review.Text) == "" {
		return errors.New("review text is required")
	}
	if err := validateTime("review created_at", review.CreatedAt); err != nil {
		return err
	}
	if err := validateTime("review updated_at", review.UpdatedAt); err != nil {
		return err
	}
	if review.UpdatedAt.Before(review.CreatedAt) {
		return errors.New("review updated_at precedes created_at")
	}
	return nil
}

func validateComment(comment Comment) error {
	if !identity.Valid(comment.ID) {
		return errors.New("id must be lowercase hexadecimal")
	}
	if strings.TrimSpace(comment.Text) == "" {
		return errors.New("text is required")
	}
	if err := validateTime("created_at", comment.CreatedAt); err != nil {
		return err
	}
	if err := validateTime("updated_at", comment.UpdatedAt); err != nil {
		return err
	}
	if comment.UpdatedAt.Before(comment.CreatedAt) {
		return errors.New("updated_at precedes created_at")
	}
	return nil
}

func validateTime(field string, value time.Time) error {
	if value.IsZero() {
		return fmt.Errorf("%s is required", field)
	}
	_, offset := value.Zone()
	if offset != 0 {
		return fmt.Errorf("%s must be UTC", field)
	}
	return nil
}
