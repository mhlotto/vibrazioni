package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

var ErrDuplicateContent = errors.New("duplicate paper content")

type DuplicateContentError struct {
	PaperID string
}

func (e *DuplicateContentError) Error() string {
	return fmt.Sprintf("%s: existing paper %s", ErrDuplicateContent, e.PaperID)
}

func (e *DuplicateContentError) Unwrap() error { return ErrDuplicateContent }

type AddOptions struct {
	Title   string
	Authors []string
	Source  string
	Tags    []string
}

var extensionPattern = regexp.MustCompile(`^[a-z0-9]{1,16}$`)

// AddLocal copies a local document into a newly staged paper directory and
// returns the durable paper record.
func AddLocal(catalogPath, sourcePath string, options AddOptions, addedAt time.Time) (model.Paper, error) {
	options.Title = strings.TrimSpace(options.Title)
	if options.Title == "" {
		return model.Paper{}, errors.New("paper title is required")
	}
	if sourcePath == "" {
		return model.Paper{}, errors.New("source document is required")
	}
	if err := normalizeOptions(&options); err != nil {
		return model.Paper{}, err
	}

	if _, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename)); err != nil {
		return model.Paper{}, fmt.Errorf("read catalog: %w", err)
	}
	papersPath := filepath.Join(catalogPath, PapersDirectory)
	if info, err := os.Stat(papersPath); err != nil {
		return model.Paper{}, fmt.Errorf("inspect papers directory: %w", err)
	} else if !info.IsDir() {
		return model.Paper{}, errors.New("papers path is not a directory")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return model.Paper{}, fmt.Errorf("open source document: %w", err)
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return model.Paper{}, fmt.Errorf("inspect source document: %w", err)
	}
	if !info.Mode().IsRegular() {
		return model.Paper{}, errors.New("source document is not a regular file")
	}

	stagePath, err := os.MkdirTemp(catalogPath, ".papersplz-import-*")
	if err != nil {
		return model.Paper{}, fmt.Errorf("create import staging directory: %w", err)
	}
	defer os.RemoveAll(stagePath)

	storedName := normalizedDocumentName(filepath.Base(sourcePath))
	size, digest, err := copyAndHash(source, filepath.Join(stagePath, storedName))
	if err != nil {
		return model.Paper{}, err
	}
	duplicateID, err := findDigest(papersPath, digest)
	if err != nil {
		return model.Paper{}, err
	}
	if duplicateID != "" {
		return model.Paper{}, &DuplicateContentError{PaperID: duplicateID}
	}

	id, finalPath, err := unusedPaperPath(papersPath)
	if err != nil {
		return model.Paper{}, err
	}
	timestamp := addedAt.UTC()
	paper := model.Paper{
		SchemaVersion: model.SchemaVersion,
		ID:            id,
		Title:         options.Title,
		Authors:       options.Authors,
		Source:        strings.TrimSpace(options.Source),
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         storedName,
			OriginalName: filepath.Base(sourcePath),
			Size:         size,
			SHA256:       digest,
		},
		Tags:     options.Tags,
		Review:   nil,
		Comments: []model.Comment{},
	}
	if err := store.WritePaper(filepath.Join(stagePath, store.RecordFilename), paper); err != nil {
		return model.Paper{}, fmt.Errorf("write paper record: %w", err)
	}
	if err := os.Chmod(stagePath, 0o755); err != nil {
		return model.Paper{}, fmt.Errorf("set paper directory permissions: %w", err)
	}
	if err := os.Rename(stagePath, finalPath); err != nil {
		return model.Paper{}, fmt.Errorf("complete paper import: %w", err)
	}
	return paper, nil
}

func normalizeOptions(options *AddOptions) error {
	for i, author := range options.Authors {
		options.Authors[i] = strings.TrimSpace(author)
		if options.Authors[i] == "" {
			return fmt.Errorf("author %d is empty", i)
		}
	}
	normalizedTags := make([]string, 0, len(options.Tags))
	seen := make(map[string]struct{}, len(options.Tags))
	for _, tag := range options.Tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalizedTags = append(normalizedTags, tag)
	}
	options.Tags = normalizedTags
	return nil
}

func normalizedDocumentName(originalName string) string {
	extension := strings.TrimPrefix(filepath.Ext(originalName), ".")
	extension = strings.ToLower(extension)
	if !extensionPattern.MatchString(extension) {
		extension = "bin"
	}
	return "paper." + extension
}

func copyAndHash(source io.Reader, destinationPath string) (int64, string, error) {
	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, "", fmt.Errorf("create stored document: %w", err)
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(destination, hash), source)
	closeErr := destination.Close()
	if copyErr != nil {
		return 0, "", fmt.Errorf("copy source document: %w", copyErr)
	}
	if closeErr != nil {
		return 0, "", fmt.Errorf("close stored document: %w", closeErr)
	}
	return size, hex.EncodeToString(hash.Sum(nil)), nil
}

func findDigest(papersPath, digest string) (string, error) {
	entries, err := os.ReadDir(papersPath)
	if err != nil {
		return "", fmt.Errorf("scan papers directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		recordPath := filepath.Join(papersPath, entry.Name(), store.RecordFilename)
		paper, err := store.ReadPaper(recordPath)
		if err != nil {
			return "", fmt.Errorf("read paper %s: %w", entry.Name(), err)
		}
		if paper.ID != entry.Name() {
			return "", fmt.Errorf("paper id %s does not match directory %s", paper.ID, entry.Name())
		}
		if paper.File.SHA256 == digest {
			return paper.ID, nil
		}
	}
	return "", nil
}

func unusedPaperPath(papersPath string) (string, string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		id, err := identity.NewPaperID()
		if err != nil {
			return "", "", err
		}
		path := filepath.Join(papersPath, id)
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return id, path, nil
		} else if err != nil {
			return "", "", fmt.Errorf("inspect generated paper path: %w", err)
		}
	}
	return "", "", errors.New("could not generate an unused paper id")
}
