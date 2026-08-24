package model

import (
	"fmt"
	"regexp"
	"strings"
)

var tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]*$`)

// NormalizeTag applies the v1 tag normalization and validation rules.
func NormalizeTag(tag string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(tag))
	if !tagPattern.MatchString(normalized) {
		return "", fmt.Errorf("invalid tag %q: must match [a-z0-9][a-z0-9._+-]*", tag)
	}
	return normalized, nil
}

// NormalizeTags normalizes tags, removes duplicates, and preserves their
// first-occurrence order.
func NormalizeTags(tags []string) ([]string, error) {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		value, err := NormalizeTag(tag)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized, nil
}
