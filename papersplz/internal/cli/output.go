package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/catalog"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

type InfoOutput struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Path        string     `json:"path"`
	PaperCount  int        `json:"paper_count"`
	TagCount    int        `json:"tag_count"`
	LastAdded   *time.Time `json:"last_added,omitempty"`
}

type ShowOutput struct {
	ID               string               `json:"id"`
	Title            string               `json:"title"`
	Authors          []string             `json:"authors"`
	Source           string               `json:"source"`
	SourceURL        string               `json:"source_url"`
	Tags             []string             `json:"tags"`
	AddedAt          time.Time            `json:"added_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	StoredFilename   string               `json:"stored_filename"`
	OriginalFilename string               `json:"original_filename"`
	FileSize         int64                `json:"file_size"`
	ReviewStatus     string               `json:"review_status"`
	CommentCount     int                  `json:"comment_count"`
	Relationships    []RelationshipOutput `json:"relationships"`
}

type ListOutput struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type SearchOutput struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type TagListOutput struct {
	PaperID string   `json:"paper_id"`
	Tags    []string `json:"tags"`
}

type TagUsageOutput struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
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

type ExportOutput struct {
	Catalog model.Catalog `json:"catalog"`
	Papers  []model.Paper `json:"papers"`
}

type RelationshipOutput struct {
	Type    string `json:"type"`
	PaperID string `json:"paper_id"`
	Title   string `json:"title"`
}

func newInfoOutput(info catalog.Info) InfoOutput {
	return InfoOutput{
		Name:        info.Metadata.Name,
		Description: info.Metadata.Description,
		Path:        info.Path,
		PaperCount:  info.PaperCount,
		TagCount:    info.TagCount,
		LastAdded:   info.LastAdded,
	}
}

func newExportOutput(export catalog.MetadataExport) ExportOutput {
	papers := export.Papers
	if papers == nil {
		papers = []model.Paper{}
	}
	return ExportOutput{Catalog: export.Catalog, Papers: papers}
}

func writeInfo(writer io.Writer, info catalog.Info) {
	output := newInfoOutput(info)
	fmt.Fprintf(writer, "Name: %s\n", output.Name)
	fmt.Fprintf(writer, "Description: %s\n", displayValue(output.Description))
	fmt.Fprintf(writer, "Path: %s\n", output.Path)
	fmt.Fprintf(writer, "Papers: %d\n", output.PaperCount)
	fmt.Fprintf(writer, "Tags: %d\n", output.TagCount)
	if output.LastAdded != nil {
		fmt.Fprintf(writer, "Last added: %s\n", output.LastAdded.Format("2006-01-02"))
	}
}

func newShowOutput(paper model.Paper, relationships []catalog.ListedRelationship) ShowOutput {
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
		Relationships:    newRelationshipOutput(relationships),
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

func newSearchOutput(papers []model.Paper) []SearchOutput {
	output := make([]SearchOutput, len(papers))
	for i, paper := range papers {
		output[i] = SearchOutput{ID: paper.ID, Title: paper.Title, Authors: nonNilStrings(paper.Authors)}
	}
	return output
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeShow(writer io.Writer, paper model.Paper, relationships []catalog.ListedRelationship) {
	output := newShowOutput(paper, relationships)
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
	fmt.Fprintln(writer, "Relationships:")
	writeRelationships(writer, output.Relationships)
}

func newRelationshipOutput(relationships []catalog.ListedRelationship) []RelationshipOutput {
	output := make([]RelationshipOutput, len(relationships))
	for i, relationship := range relationships {
		output[i] = RelationshipOutput{Type: relationship.Type, PaperID: relationship.PaperID, Title: relationship.Title}
	}
	return output
}

func writeRelationships(writer io.Writer, relationships []RelationshipOutput) {
	if len(relationships) == 0 {
		fmt.Fprintln(writer, "  none")
		return
	}
	for _, relationship := range relationships {
		fmt.Fprintf(writer, "  %s %s %s\n", relationship.Type, abbreviatedID(relationship.PaperID), relationship.Title)
	}
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

func newTagUsageOutput(usage []catalog.TagUsage) []TagUsageOutput {
	output := make([]TagUsageOutput, len(usage))
	for i, item := range usage {
		output[i] = TagUsageOutput{Tag: item.Tag, Count: item.Count}
	}
	return output
}

func writeTagUsage(writer io.Writer, usage []catalog.TagUsage) error {
	if len(usage) == 0 {
		_, err := fmt.Fprintln(writer, "No tags.")
		return err
	}
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	for _, item := range usage {
		if _, err := fmt.Fprintf(table, "%s\t%d\n", item.Tag, item.Count); err != nil {
			return err
		}
	}
	return table.Flush()
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
