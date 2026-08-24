package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mhlotto/vibrazioni/papersplz/internal/identity"
	"github.com/mhlotto/vibrazioni/papersplz/internal/model"
	"github.com/mhlotto/vibrazioni/papersplz/internal/store"
)

type DoctorProblem struct {
	Path    string
	Message string
}

type doctorRecord struct {
	directory string
	paper     model.Paper
}

func (p DoctorProblem) String() string {
	if p.Path == "" {
		return p.Message
	}
	return p.Path + ": " + p.Message
}

// Doctor performs a read-only consistency scan and returns every independent
// problem it can safely discover.
func Doctor(catalogPath string) []DoctorProblem {
	problems := make([]DoctorProblem, 0)
	catalogFile := filepath.Join(catalogPath, store.CatalogFilename)
	var metadata model.Catalog
	if err := decodeDoctorJSON(catalogFile, &metadata); err != nil {
		problems = append(problems, DoctorProblem{Path: store.CatalogFilename, Message: err.Error()})
	} else if err := model.ValidateCatalog(metadata); err != nil {
		problems = append(problems, DoctorProblem{Path: store.CatalogFilename, Message: err.Error()})
	}
	problems = append(problems, inspectDoctorRootTemporaries(catalogPath)...)

	papersPath := filepath.Join(catalogPath, PapersDirectory)
	entries, err := os.ReadDir(papersPath)
	if err != nil {
		return append(problems, DoctorProblem{Path: PapersDirectory, Message: err.Error()})
	}

	records := make([]doctorRecord, 0, len(entries))
	for _, entry := range entries {
		relativeDirectory := filepath.Join(PapersDirectory, entry.Name())
		if !entry.IsDir() {
			problems = append(problems, DoctorProblem{Path: relativeDirectory, Message: "unexpected entry; expected a paper directory"})
			continue
		}
		if !identity.Valid(entry.Name()) {
			problems = append(problems, DoctorProblem{Path: relativeDirectory, Message: "unexpected paper directory name; expected a lowercase hexadecimal ID"})
		}
		recordPath := filepath.Join(papersPath, entry.Name(), store.RecordFilename)
		var paper model.Paper
		if err := decodeDoctorJSON(recordPath, &paper); err != nil {
			problems = append(problems, DoctorProblem{Path: filepath.Join(relativeDirectory, store.RecordFilename), Message: err.Error()})
			problems = append(problems, inspectDoctorDirectory(papersPath, entry.Name(), "")...)
			continue
		}
		records = append(records, doctorRecord{directory: entry.Name(), paper: paper})
		if err := model.ValidatePaper(paper); err != nil {
			problems = append(problems, DoctorProblem{Path: filepath.Join(relativeDirectory, store.RecordFilename), Message: err.Error()})
		}
		problems = append(problems, doctorSchemaConsistency(metadata.SchemaVersion, relativeDirectory, paper)...)
		if paper.ID != entry.Name() {
			problems = append(problems, DoctorProblem{Path: relativeDirectory, Message: fmt.Sprintf("paper ID %q does not match directory name", paper.ID)})
		}
		problems = append(problems, inspectDoctorDocument(papersPath, entry.Name(), paper)...)
		problems = append(problems, inspectDoctorDirectory(papersPath, entry.Name(), paper.File.Name)...)
	}
	problems = append(problems, duplicateDoctorProblems(records)...)
	problems = append(problems, relationshipDoctorProblems(records)...)
	return problems
}

func doctorSchemaConsistency(catalogSchema int, relativeDirectory string, paper model.Paper) []DoctorProblem {
	path := filepath.Join(relativeDirectory, store.RecordFilename)
	problems := make([]DoctorProblem, 0, 2)
	if paper.SchemaVersion == model.PaperSchemaVersion1 && len(paper.Relationships) != 0 {
		problems = append(problems, DoctorProblem{Path: path, Message: "schema version 1 record contains relationship metadata; upgrade to schema version 2 is required"})
	}
	if catalogSchema == model.CatalogSchemaVersion1 && paper.SchemaVersion == model.PaperSchemaVersion2 {
		problems = append(problems, DoctorProblem{Path: path, Message: "paper schema version 2 is incompatible with catalog schema version 1"})
	}
	if catalogSchema == model.CatalogSchemaVersion2 && paper.SchemaVersion == model.PaperSchemaVersion1 {
		problems = append(problems, DoctorProblem{Path: path, Message: "paper schema version 1 is pending upgrade in schema version 2 catalog"})
	}
	return problems
}

func relationshipDoctorProblems(records []doctorRecord) []DoctorProblem {
	ids := make(map[string]struct{}, len(records))
	for _, record := range records {
		if identity.Valid(record.paper.ID) {
			ids[record.paper.ID] = struct{}{}
		}
	}
	problems := make([]DoctorProblem, 0)
	for _, record := range records {
		for _, relationship := range record.paper.Relationships {
			if _, exists := ids[relationship.PaperID]; !exists && identity.Valid(relationship.PaperID) {
				problems = append(problems, DoctorProblem{
					Path:    filepath.Join(PapersDirectory, record.directory, store.RecordFilename),
					Message: fmt.Sprintf("relationship %q references missing paper %s", relationship.Type, relationship.PaperID),
				})
			}
		}
	}
	return problems
}

func decodeDoctorJSON(path string, destination any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("malformed JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("malformed JSON: multiple JSON values")
		}
		return fmt.Errorf("malformed JSON: %w", err)
	}
	return nil
}

func inspectDoctorDocument(papersPath, directory string, paper model.Paper) []DoctorProblem {
	if paper.File.Name == "" || filepath.Base(paper.File.Name) != paper.File.Name || strings.Contains(paper.File.Name, `\`) {
		return nil
	}
	relativePath := filepath.Join(PapersDirectory, directory, paper.File.Name)
	path := filepath.Join(papersPath, directory, paper.File.Name)
	info, err := os.Lstat(path)
	if err != nil {
		return []DoctorProblem{{Path: relativePath, Message: fmt.Sprintf("referenced document is unavailable: %v", err)}}
	}
	if !info.Mode().IsRegular() {
		return []DoctorProblem{{Path: relativePath, Message: "referenced document is not a regular file"}}
	}
	problems := make([]DoctorProblem, 0, 2)
	if info.Size() != paper.File.Size {
		problems = append(problems, DoctorProblem{Path: relativePath, Message: fmt.Sprintf("size is %d bytes; record says %d", info.Size(), paper.File.Size)})
	}
	file, err := os.Open(path)
	if err != nil {
		return append(problems, DoctorProblem{Path: relativePath, Message: fmt.Sprintf("cannot read document: %v", err)})
	}
	digest := sha256.New()
	_, copyErr := io.Copy(digest, file)
	closeErr := file.Close()
	if copyErr != nil {
		return append(problems, DoctorProblem{Path: relativePath, Message: fmt.Sprintf("cannot hash document: %v", copyErr)})
	}
	if closeErr != nil {
		return append(problems, DoctorProblem{Path: relativePath, Message: fmt.Sprintf("cannot close document after hashing: %v", closeErr)})
	}
	actual := hex.EncodeToString(digest.Sum(nil))
	if actual != paper.File.SHA256 {
		problems = append(problems, DoctorProblem{Path: relativePath, Message: fmt.Sprintf("SHA-256 is %s; record says %s", actual, paper.File.SHA256)})
	}
	return problems
}

func inspectDoctorDirectory(papersPath, directory, documentName string) []DoctorProblem {
	entries, err := os.ReadDir(filepath.Join(papersPath, directory))
	if err != nil {
		return []DoctorProblem{{Path: filepath.Join(PapersDirectory, directory), Message: fmt.Sprintf("cannot read paper directory: %v", err)}}
	}
	problems := make([]DoctorProblem, 0)
	for _, entry := range entries {
		if entry.Name() != store.RecordFilename && entry.Name() != documentName {
			message := "unexpected entry in paper directory"
			if temporaryArtifactKind(entry.Name()) == "temporary metadata" {
				message = "abandoned papersplz temporary metadata artifact"
			}
			problems = append(problems, DoctorProblem{
				Path:    filepath.Join(PapersDirectory, directory, entry.Name()),
				Message: message,
			})
		}
	}
	return problems
}

func inspectDoctorRootTemporaries(catalogPath string) []DoctorProblem {
	entries, err := os.ReadDir(catalogPath)
	if err != nil {
		return nil
	}
	problems := make([]DoctorProblem, 0)
	for _, entry := range entries {
		kind := temporaryArtifactKind(entry.Name())
		if kind == "" {
			continue
		}
		problems = append(problems, DoctorProblem{
			Path:    entry.Name(),
			Message: fmt.Sprintf("abandoned papersplz %s artifact", kind),
		})
	}
	return problems
}

func temporaryArtifactKind(name string) string {
	if strings.HasPrefix(name, ".papersplz-import-") {
		return "import staging"
	}
	if strings.HasPrefix(name, ".papersplz-") && strings.HasSuffix(name, ".tmp") {
		return "temporary metadata"
	}
	return ""
}

func duplicateDoctorProblems(records []doctorRecord) []DoctorProblem {
	ids := make(map[string][]string)
	digests := make(map[string][]string)
	for _, record := range records {
		if record.paper.ID != "" {
			ids[record.paper.ID] = append(ids[record.paper.ID], record.directory)
		}
		if validDoctorDigest(record.paper.File.SHA256) {
			digests[record.paper.File.SHA256] = append(digests[record.paper.File.SHA256], record.directory)
		}
	}
	problems := duplicateMapProblems(ids, "duplicate paper ID %s appears in directories %s")
	problems = append(problems, duplicateMapProblems(digests, "duplicate content SHA-256 %s appears in directories %s")...)
	return problems
}

func duplicateMapProblems(values map[string][]string, format string) []DoctorProblem {
	keys := make([]string, 0)
	for key, locations := range values {
		if len(locations) > 1 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	problems := make([]DoctorProblem, 0, len(keys))
	for _, key := range keys {
		locations := values[key]
		sort.Strings(locations)
		problems = append(problems, DoctorProblem{Path: PapersDirectory, Message: fmt.Sprintf(format, key, strings.Join(locations, ", "))})
	}
	return problems
}

func validDoctorDigest(digest string) bool {
	if len(digest) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil && digest == strings.ToLower(digest)
}
