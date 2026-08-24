package catalog

import (
	"strings"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

type SearchOptions struct {
	Terms  []string
	Tag    string
	Author string
}

// SearchPapers performs a linear, case-insensitive plain-text scan of managed
// paper metadata. Every term must match at least one searchable field.
func SearchPapers(catalogPath string, options SearchOptions) ([]model.Paper, error) {
	papers, err := ListPapers(catalogPath, ListOptions{Tag: options.Tag, Author: options.Author})
	if err != nil {
		return nil, err
	}
	terms := make([]string, len(options.Terms))
	for i, term := range options.Terms {
		terms[i] = strings.ToLower(term)
	}
	results := make([]model.Paper, 0, len(papers))
	for _, paper := range papers {
		if paperMatchesTerms(paper, terms) {
			results = append(results, paper)
		}
	}
	return results, nil
}

func paperMatchesTerms(paper model.Paper, terms []string) bool {
	fields := []string{paper.Title, paper.Source}
	fields = append(fields, paper.Authors...)
	fields = append(fields, paper.Tags...)
	if paper.Review != nil {
		fields = append(fields, paper.Review.Text)
	}
	for _, comment := range paper.Comments {
		fields = append(fields, comment.Text)
	}
	for i := range fields {
		fields[i] = strings.ToLower(fields[i])
	}
	for _, term := range terms {
		matched := false
		for _, field := range fields {
			if strings.Contains(field, term) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
