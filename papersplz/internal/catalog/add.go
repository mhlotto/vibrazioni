package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

var ErrDuplicateContent = errors.New("duplicate paper content")

// RemoteDownloadTimeout is the overall limit for an HTTP or HTTPS import,
// including reading the response body.
const RemoteDownloadTimeout = 5 * time.Minute

var defaultDownloadHTTPClient = &http.Client{Timeout: RemoteDownloadTimeout}

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

// Add imports a local document or a direct HTTP or HTTPS URL.
func Add(catalogPath, source string, options AddOptions, addedAt time.Time) (model.Paper, error) {
	return AddWithHTTPClient(catalogPath, source, options, addedAt, defaultDownloadHTTPClient)
}

// AddWithHTTPClient is Add with an explicit client for controlled callers and
// tests. A nil client uses the default finite download timeout. A supplied
// client without a timeout is copied and given that same timeout.
func AddWithHTTPClient(catalogPath, source string, options AddOptions, addedAt time.Time, client *http.Client) (model.Paper, error) {
	parsed, isURL := directDocumentURL(source)
	if !isURL {
		return AddLocal(catalogPath, source, options, addedAt)
	}
	client = downloadHTTPClient(client)
	return addURL(catalogPath, source, parsed, options, addedAt, client)
}

func downloadHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return defaultDownloadHTTPClient
	}
	if client.Timeout > 0 {
		return client
	}
	copy := *client
	copy.Timeout = RemoteDownloadTimeout
	return &copy
}

// AddLocal copies a local document into a newly staged paper directory.
func AddLocal(catalogPath, sourcePath string, options AddOptions, addedAt time.Time) (model.Paper, error) {
	papersPath, options, err := prepareAdd(catalogPath, sourcePath, options)
	if err != nil {
		return model.Paper{}, err
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
	originalName := filepath.Base(sourcePath)
	return addFromReader(catalogPath, papersPath, source, originalName, "", options, addedAt)
}

func addURL(catalogPath, suppliedURL string, parsed *url.URL, options AddOptions, addedAt time.Time, client *http.Client) (model.Paper, error) {
	papersPath, options, err := prepareAdd(catalogPath, suppliedURL, options)
	if err != nil {
		return model.Paper{}, err
	}
	request, err := http.NewRequest(http.MethodGet, suppliedURL, nil)
	if err != nil {
		return model.Paper{}, fmt.Errorf("create download request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return model.Paper{}, fmt.Errorf("download source document: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return model.Paper{}, fmt.Errorf("download source document: HTTP status %s", response.Status)
	}
	originalName := path.Base(parsed.Path)
	if strings.HasSuffix(parsed.Path, "/") || originalName == "." || originalName == "/" || originalName == "" {
		originalName = "download"
	}
	return addFromReader(catalogPath, papersPath, response.Body, originalName, suppliedURL, options, addedAt)
}

func prepareAdd(catalogPath, source string, options AddOptions) (string, AddOptions, error) {
	options.Title = strings.TrimSpace(options.Title)
	if options.Title == "" {
		return "", AddOptions{}, errors.New("paper title is required")
	}
	if source == "" {
		return "", AddOptions{}, errors.New("source document is required")
	}
	if err := normalizeOptions(&options); err != nil {
		return "", AddOptions{}, err
	}

	if _, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename)); err != nil {
		return "", AddOptions{}, fmt.Errorf("read catalog: %w", err)
	}
	papersPath := filepath.Join(catalogPath, PapersDirectory)
	if info, err := os.Stat(papersPath); err != nil {
		return "", AddOptions{}, fmt.Errorf("inspect papers directory: %w", err)
	} else if !info.IsDir() {
		return "", AddOptions{}, errors.New("papers path is not a directory")
	}
	return papersPath, options, nil
}

func addFromReader(catalogPath, papersPath string, source io.Reader, originalName, sourceURL string, options AddOptions, addedAt time.Time) (model.Paper, error) {
	stagePath, err := os.MkdirTemp(catalogPath, ".papersplz-import-*")
	if err != nil {
		return model.Paper{}, fmt.Errorf("create import staging directory: %w", err)
	}
	defer os.RemoveAll(stagePath)

	storedName := normalizedDocumentName(originalName)
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
	metadata, err := store.ReadCatalog(filepath.Join(catalogPath, store.CatalogFilename))
	if err != nil {
		return model.Paper{}, fmt.Errorf("read catalog before writing paper: %w", err)
	}
	paper := model.Paper{
		SchemaVersion: paperSchemaForCatalog(metadata.SchemaVersion),
		ID:            id,
		Title:         options.Title,
		Authors:       options.Authors,
		Source:        strings.TrimSpace(options.Source),
		SourceURL:     sourceURL,
		AddedAt:       timestamp,
		UpdatedAt:     timestamp,
		File: model.File{
			Name:         storedName,
			OriginalName: originalName,
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

func directDocumentURL(source string) (*url.URL, bool) {
	parsed, err := url.Parse(source)
	if err != nil || parsed.Host == "" {
		return nil, false
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, false
	}
	return parsed, true
}

func normalizeOptions(options *AddOptions) error {
	for i, author := range options.Authors {
		options.Authors[i] = strings.TrimSpace(author)
		if options.Authors[i] == "" {
			return fmt.Errorf("author %d is empty", i)
		}
	}
	normalizedTags, err := model.NormalizeTags(options.Tags)
	if err != nil {
		return err
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
