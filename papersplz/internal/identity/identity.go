// Package identity generates and resolves papersplz identifiers.
package identity

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const byteLength = 16

var (
	ErrInvalid   = errors.New("invalid id selector")
	ErrNotFound  = errors.New("id not found")
	ErrAmbiguous = errors.New("ambiguous id prefix")
)

func NewPaperID() (string, error) {
	return generate(rand.Reader)
}

func NewCommentID() (string, error) {
	return generate(rand.Reader)
}

func generate(reader io.Reader) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := io.ReadFull(reader, bytes); err != nil {
		return "", fmt.Errorf("generate random id: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// Resolve returns an exact candidate or a candidate identified by a unique
// prefix. It never selects one of multiple matches arbitrarily.
func Resolve(selector string, candidates []string) (string, error) {
	if !Valid(selector) {
		return "", fmt.Errorf("%w: %q must be lowercase hexadecimal", ErrInvalid, selector)
	}

	exactMatches := 0
	for _, candidate := range candidates {
		if candidate == selector {
			exactMatches++
		}
	}
	if exactMatches == 1 {
		return selector, nil
	}
	if exactMatches > 1 {
		return "", fmt.Errorf("%w %q", ErrAmbiguous, selector)
	}

	var match string
	matches := 0
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, selector) {
			match = candidate
			matches++
		}
	}
	switch matches {
	case 0:
		return "", fmt.Errorf("%w: %q", ErrNotFound, selector)
	case 1:
		return match, nil
	default:
		return "", fmt.Errorf("%w %q (%d matches)", ErrAmbiguous, selector, matches)
	}
}

func Valid(id string) bool {
	if id == "" {
		return false
	}
	for _, character := range id {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}
