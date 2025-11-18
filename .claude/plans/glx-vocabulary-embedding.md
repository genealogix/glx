# GLX Vocabulary Embedding Strategy

**Date**: 2025-11-18
**Status**: Implementation Plan
**Decision**: Embed vocabularies in binary using go:embed

---

## Overview

Standard GLX vocabularies will be embedded in the binary using Go's `embed` package. This ensures that `glx import` and other commands can always write complete, valid archives without requiring external vocabulary files.

---

## Directory Structure

```
glx/
├── standard-vocabularies/
│   ├── event-types.glx
│   ├── relationship-types.glx
│   ├── place-types.glx
│   ├── source-types.glx
│   ├── repository-types.glx
│   ├── media-types.glx
│   ├── participant-roles.glx
│   ├── confidence-levels.glx
│   └── quality-ratings.glx
└── (existing files...)
```

---

## Implementation

### 1. Embed Package

```go
// lib/vocabularies.go

package lib

import (
	_ "embed"
	"fmt"
	"path/filepath"
)

// Standard vocabulary files embedded at compile time
//go:embed ../glx/standard-vocabularies/event-types.glx
var eventTypesYAML string

//go:embed ../glx/standard-vocabularies/relationship-types.glx
var relationshipTypesYAML string

//go:embed ../glx/standard-vocabularies/place-types.glx
var placeTypesYAML string

//go:embed ../glx/standard-vocabularies/source-types.glx
var sourceTypesYAML string

//go:embed ../glx/standard-vocabularies/repository-types.glx
var repositoryTypesYAML string

//go:embed ../glx/standard-vocabularies/media-types.glx
var mediaTypesYAML string

//go:embed ../glx/standard-vocabularies/participant-roles.glx
var participantRolesYAML string

//go:embed ../glx/standard-vocabularies/confidence-levels.glx
var confidenceLevelsYAML string

//go:embed ../glx/standard-vocabularies/quality-ratings.glx
var qualityRatingsYAML string

// StandardVocabularies returns a map of vocabulary name to YAML content
func StandardVocabularies() map[string]string {
	return map[string]string{
		"event-types":        eventTypesYAML,
		"relationship-types": relationshipTypesYAML,
		"place-types":        placeTypesYAML,
		"source-types":       sourceTypesYAML,
		"repository-types":   repositoryTypesYAML,
		"media-types":        mediaTypesYAML,
		"participant-roles":  participantRolesYAML,
		"confidence-levels":  confidenceLevelsYAML,
		"quality-ratings":    qualityRatingsYAML,
	}
}

// WriteStandardVocabularies writes all standard vocabularies to a directory
func WriteStandardVocabularies(outputDir string) error {
	vocabDir := filepath.Join(outputDir, "vocabularies")

	// Create vocabularies directory
	if err := os.MkdirAll(vocabDir, 0755); err != nil {
		return fmt.Errorf("failed to create vocabularies directory: %w", err)
	}

	// Write each vocabulary file
	vocabs := StandardVocabularies()
	for name, content := range vocabs {
		filename := filepath.Join(vocabDir, name+".glx")
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write vocabulary %s: %w", name, err)
		}
	}

	return nil
}
```

### 2. Alternative: Embed FS (More Flexible)

If we want more flexibility (like listing files dynamically):

```go
// lib/vocabularies.go

package lib

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed ../glx/standard-vocabularies/*.glx
var vocabulariesFS embed.FS

// WriteStandardVocabularies writes all embedded vocabularies to a directory
func WriteStandardVocabularies(outputDir string) error {
	vocabDir := filepath.Join(outputDir, "vocabularies")

	// Create vocabularies directory
	if err := os.MkdirAll(vocabDir, 0755); err != nil {
		return fmt.Errorf("failed to create vocabularies directory: %w", err)
	}

	// Walk embedded FS and write files
	err := fs.WalkDir(vocabulariesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read embedded file
		content, err := vocabulariesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Write to output directory
		// Strip the "../glx/standard-vocabularies/" prefix from path
		filename := filepath.Base(path)
		outputPath := filepath.Join(vocabDir, filename)

		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}

		return nil
	})

	return err
}

// GetStandardVocabulary returns the content of a specific vocabulary
func GetStandardVocabulary(name string) (string, error) {
	filename := fmt.Sprintf("../glx/standard-vocabularies/%s.glx", name)
	content, err := vocabulariesFS.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("vocabulary not found: %s", name)
	}
	return string(content), nil
}

// ListStandardVocabularies returns a list of all embedded vocabulary names
func ListStandardVocabularies() ([]string, error) {
	var names []string

	err := fs.WalkDir(vocabulariesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".glx" {
			name := filepath.Base(path)
			name = name[:len(name)-4] // Remove .glx extension
			names = append(names, name)
		}
		return nil
	})

	return names, err
}
```

**Recommendation**: Use embed.FS approach - more flexible, easier to maintain

---

## Usage in Serializer

```go
// lib/serializer.go

func (s *DefaultSerializer) SerializeMultiFile(glx *GLXFile, outputDir string) error {
	// ... create directory structure ...

	// Write vocabularies if requested
	if s.IncludeVocabularies {
		if err := WriteStandardVocabularies(outputDir); err != nil {
			return fmt.Errorf("failed to write vocabularies: %w", err)
		}
	}

	// ... write entities ...

	return nil
}
```

---

## Single-File Format

For single-file archives, vocabularies can optionally be included at the top:

```go
func (s *DefaultSerializer) SerializeSingleFileBytes(glx *GLXFile) ([]byte, error) {
	// If vocabularies requested, prepend them to the YAML
	if s.IncludeVocabularies {
		vocabs := StandardVocabularies()

		// Build combined YAML with vocabularies first
		var buf bytes.Buffer

		// Write vocabularies section
		buf.WriteString("# Standard Vocabularies\n\n")
		for name, content := range vocabs {
			buf.WriteString(fmt.Sprintf("# %s\n", name))
			buf.WriteString(content)
			buf.WriteString("\n\n")
		}

		// Write entities
		buf.WriteString("# Entities\n\n")
		entityYAML, err := yaml.Marshal(glx)
		if err != nil {
			return nil, err
		}
		buf.Write(entityYAML)

		return buf.Bytes(), nil
	}

	// Without vocabularies, just serialize GLXFile
	return yaml.Marshal(glx)
}
```

**Note**: For single-file, vocabularies are optional since the format is typically used for smaller archives or when vocabularies are already defined elsewhere.

---

## Testing

### Test Embedded Vocabularies

```go
// lib/vocabularies_test.go

func TestStandardVocabulariesEmbedded(t *testing.T) {
	vocabs := StandardVocabularies()

	// Check all expected vocabularies are present
	expected := []string{
		"event-types",
		"relationship-types",
		"place-types",
		"source-types",
		"repository-types",
		"media-types",
		"participant-roles",
		"confidence-levels",
		"quality-ratings",
	}

	for _, name := range expected {
		content, ok := vocabs[name]
		if !ok {
			t.Errorf("Missing vocabulary: %s", name)
		}
		if len(content) == 0 {
			t.Errorf("Empty vocabulary: %s", name)
		}
	}
}

func TestWriteStandardVocabularies(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Write vocabularies
	err := WriteStandardVocabularies(tmpDir)
	if err != nil {
		t.Fatalf("Failed to write vocabularies: %v", err)
	}

	// Check files exist
	vocabDir := filepath.Join(tmpDir, "vocabularies")
	files, err := os.ReadDir(vocabDir)
	if err != nil {
		t.Fatalf("Failed to read vocabulary directory: %v", err)
	}

	// Should have 9 vocabulary files
	if len(files) != 9 {
		t.Errorf("Expected 9 vocabulary files, got %d", len(files))
	}

	// Check each file has content
	for _, file := range files {
		path := filepath.Join(vocabDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file.Name(), err)
		}
		if len(data) == 0 {
			t.Errorf("Empty vocabulary file: %s", file.Name())
		}
	}
}
```

---

## Build Considerations

### Binary Size Impact

Embedding vocabularies will increase binary size:
- Each vocabulary: ~1-5 KB
- Total: ~20-30 KB
- Negligible impact on binary size

### Build Tags (Optional)

If we want to optionally exclude vocabularies for minimal builds:

```go
//go:build !novocab

package lib

//go:embed ../glx/standard-vocabularies/*.glx
var vocabulariesFS embed.FS
```

```bash
# Build with vocabularies (default)
go build ./cmd/glx

# Build without vocabularies (minimal)
go build -tags novocab ./cmd/glx
```

**Recommendation**: Always include vocabularies - the size impact is minimal and the convenience is high.

---

## Future Enhancements

1. **Custom Vocabularies**
   - Allow users to provide custom vocabulary directory
   - Merge custom with standard vocabularies

2. **Vocabulary Validation**
   - Validate vocabulary YAML structure at compile time
   - Generate vocabulary types from YAML

3. **Vocabulary Versioning**
   - Include vocabulary version in embedded data
   - Support multiple vocabulary versions

4. **Vocabulary Updates**
   - Command to update vocabularies: `glx vocab update`
   - Download latest vocabularies from web

---

## Implementation Checklist

- [ ] Create lib/vocabularies.go
- [ ] Implement embed.FS approach
- [ ] Implement WriteStandardVocabularies()
- [ ] Implement GetStandardVocabulary()
- [ ] Implement ListStandardVocabularies()
- [ ] Add unit tests for embedded vocabularies
- [ ] Add integration test for writing vocabularies
- [ ] Integrate with serializer
- [ ] Test build size impact
- [ ] Update documentation

---

## Summary

**Decision**: Use embed.FS to embed all standard vocabularies in binary
**Approach**: Write vocabularies to multi-file archives, optionally include in single-file
**Benefits**: Self-contained binary, always available, minimal size impact
**Implementation**: ~100 lines of code + tests
