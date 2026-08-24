// Package model defines the version 1 on-disk metadata model.
package model

import "time"

const (
	CatalogSchemaVersion1       = 1
	CatalogSchemaVersion2       = 2
	CurrentCatalogSchemaVersion = CatalogSchemaVersion2
	PaperSchemaVersion1         = 1
	PaperSchemaVersion2         = 2
	CurrentPaperSchemaVersion   = PaperSchemaVersion2
)

type Catalog struct {
	SchemaVersion int       `json:"schema_version"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Paper struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Authors       []string       `json:"authors"`
	Source        string         `json:"source,omitempty"`
	SourceURL     string         `json:"source_url,omitempty"`
	AddedAt       time.Time      `json:"added_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	File          File           `json:"file"`
	Tags          []string       `json:"tags"`
	Relationships []Relationship `json:"relationships,omitempty"`
	Review        *Review        `json:"review"`
	Comments      []Comment      `json:"comments"`
}

type Relationship struct {
	Type    string `json:"type"`
	PaperID string `json:"paper_id"`
}

type File struct {
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size"`
	SHA256       string `json:"sha256"`
}

type Review struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Comment struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
