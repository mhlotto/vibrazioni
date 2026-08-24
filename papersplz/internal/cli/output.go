package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

type ShowOutput struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Authors          []string  `json:"authors"`
	Source           string    `json:"source"`
	SourceURL        string    `json:"source_url"`
	Tags             []string  `json:"tags"`
	AddedAt          time.Time `json:"added_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	StoredFilename   string    `json:"stored_filename"`
	OriginalFilename string    `json:"original_filename"`
	FileSize         int64     `json:"file_size"`
	ReviewStatus     string    `json:"review_status"`
	CommentCount     int       `json:"comment_count"`
}

type ListOutput struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type TagListOutput struct {
	PaperID string   `json:"paper_id"`
	Tags    []string `json:"tags"`
}

type ReviewOutput struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommentOutput struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newShowOutput(paper model.Paper) ShowOutput {
	reviewStatus := "none"
	if paper.Review != nil {
		reviewStatus = "present"
	}
	return ShowOutput{
		ID:               paper.ID,
		Title:            paper.Title,
		Authors:          nonNilStrings(paper.Authors),
		Source:           paper.Source,
		SourceURL:        paper.SourceURL,
		Tags:             nonNilStrings(paper.Tags),
		AddedAt:          paper.AddedAt,
		UpdatedAt:        paper.UpdatedAt,
		StoredFilename:   paper.File.Name,
		OriginalFilename: paper.File.OriginalName,
		FileSize:         paper.File.Size,
		ReviewStatus:     reviewStatus,
		CommentCount:     len(paper.Comments),
	}
}

func newListOutput(papers []model.Paper) []ListOutput {
	output := make([]ListOutput, len(papers))
	for i, paper := range papers {
		output[i] = ListOutput{
			ID:      paper.ID,
			Title:   paper.Title,
			Authors: nonNilStrings(paper.Authors),
		}
	}
	return output
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeShow(writer io.Writer, paper model.Paper) {
	output := newShowOutput(paper)
	fmt.Fprintf(writer, "ID: %s\n", output.ID)
	fmt.Fprintf(writer, "Title: %s\n", output.Title)
	fmt.Fprintf(writer, "Authors: %s\n", displayList(output.Authors))
	fmt.Fprintf(writer, "Source: %s\n", displayValue(output.Source))
	fmt.Fprintf(writer, "Source URL: %s\n", displayValue(output.SourceURL))
	fmt.Fprintf(writer, "Tags: %s\n", displayList(output.Tags))
	fmt.Fprintf(writer, "Added: %s\n", output.AddedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "Updated: %s\n", output.UpdatedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "Stored filename: %s\n", output.StoredFilename)
	fmt.Fprintf(writer, "Original filename: %s\n", output.OriginalFilename)
	fmt.Fprintf(writer, "File size: %d bytes\n", output.FileSize)
	fmt.Fprintf(writer, "Review: %s\n", output.ReviewStatus)
	fmt.Fprintf(writer, "Comments: %d\n", output.CommentCount)
}

func writeList(writer io.Writer, papers []model.Paper) error {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "ID\tTITLE\tAUTHORS"); err != nil {
		return err
	}
	for _, paper := range papers {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\n", abbreviatedID(paper.ID), paper.Title, strings.Join(paper.Authors, ", ")); err != nil {
			return err
		}
	}
	return table.Flush()
}

func writeTags(writer io.Writer, tags []string) {
	if len(tags) == 0 {
		fmt.Fprintln(writer, "No tags.")
		return
	}
	for _, tag := range tags {
		fmt.Fprintln(writer, tag)
	}
}

func writeReview(writer io.Writer, review model.Review) {
	fmt.Fprintf(writer, "Created: %s\n", review.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "Updated: %s\n", review.UpdatedAt.Format(time.RFC3339))
	fmt.Fprintln(writer, "Text:")
	fmt.Fprintln(writer, review.Text)
}

func newCommentOutput(comment model.Comment) CommentOutput {
	return CommentOutput(comment)
}

func newCommentListOutput(comments []model.Comment) []CommentOutput {
	output := make([]CommentOutput, len(comments))
	for i, comment := range comments {
		output[i] = newCommentOutput(comment)
	}
	return output
}

func writeComment(writer io.Writer, comment model.Comment) {
	fmt.Fprintf(writer, "ID: %s\n", comment.ID)
	fmt.Fprintf(writer, "Created: %s\n", comment.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "Updated: %s\n", comment.UpdatedAt.Format(time.RFC3339))
	fmt.Fprintln(writer, "Text:")
	fmt.Fprintln(writer, comment.Text)
}

func writeComments(writer io.Writer, comments []model.Comment) {
	if len(comments) == 0 {
		fmt.Fprintln(writer, "No comments.")
		return
	}
	for i, comment := range comments {
		if i > 0 {
			fmt.Fprintln(writer)
		}
		writeComment(writer, comment)
	}
}

func abbreviatedID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func displayList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
