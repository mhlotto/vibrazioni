// Package store reads and atomically writes papersplz JSON metadata.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
)

const (
	CatalogFilename = "catalog.json"
	RecordFilename  = "record.json"
)

func ReadCatalog(path string) (model.Catalog, error) {
	var catalog model.Catalog
	if err := readJSON(path, &catalog); err != nil {
		return model.Catalog{}, err
	}
	if err := model.ValidateCatalog(catalog); err != nil {
		return model.Catalog{}, fmt.Errorf("validate %s: %w", path, err)
	}
	return catalog, nil
}

func WriteCatalog(path string, catalog model.Catalog) error {
	if err := model.ValidateCatalog(catalog); err != nil {
		return fmt.Errorf("validate catalog: %w", err)
	}
	return writeJSONAtomic(path, catalog)
}

func ReadPaper(path string) (model.Paper, error) {
	var paper model.Paper
	if err := readJSON(path, &paper); err != nil {
		return model.Paper{}, err
	}
	if err := model.ValidatePaper(paper); err != nil {
		return model.Paper{}, fmt.Errorf("validate %s: %w", path, err)
	}
	return paper, nil
}

func WritePaper(path string, paper model.Paper) error {
	if err := model.ValidatePaper(paper); err != nil {
		return fmt.Errorf("validate paper: %w", err)
	}
	return writeJSONAtomic(path, paper)
}

func readJSON(path string, destination any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode %s: multiple JSON values", path)
		}
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	return atomicReplace(path, func(writer io.Writer) error {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(value)
	})
}

func atomicReplace(path string, write func(io.Writer) error) (err error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".papersplz-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary metadata file: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() {
		temporary.Close()
		os.Remove(temporaryName)
	}()

	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set temporary metadata permissions: %w", err)
	}
	if err := write(temporary); err != nil {
		return fmt.Errorf("write temporary metadata: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary metadata: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
