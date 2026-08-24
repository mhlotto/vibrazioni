package model

import (
	"fmt"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
)

// ResolveCommentID resolves an exact comment ID or unambiguous prefix within
// one paper.
func ResolveCommentID(paper Paper, selector string) (string, error) {
	ids := make([]string, len(paper.Comments))
	for i, comment := range paper.Comments {
		ids[i] = comment.ID
	}
	resolved, err := identity.Resolve(selector, ids)
	if err != nil {
		return "", fmt.Errorf("resolve comment id: %w", err)
	}
	return resolved, nil
}
