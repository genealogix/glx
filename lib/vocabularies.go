package lib

import (
	"fmt"
	"os"
	"path/filepath"

	vocabularies "github.com/genealogix/spec/specification/5-standard-vocabularies"
)

// StandardVocabularies returns a map of vocabulary filename to content bytes.
// Uses embedded vocabularies from the specification package.
func StandardVocabularies() map[string][]byte {
	return vocabularies.Files
}

// ListStandardVocabularies returns a list of all embedded vocabulary names (without .glx extension).
func ListStandardVocabularies() []string {
	var names []string
	for filename := range vocabularies.Files {
		// Remove .glx extension
		name := filename[:len(filename)-4]
		names = append(names, name)
	}
	return names
}

// GetStandardVocabulary returns the content of a specific vocabulary by name (without .glx extension).
// Returns error if vocabulary not found.
func GetStandardVocabulary(name string) ([]byte, error) {
	filename := name + ".glx"
	content, ok := vocabularies.Files[filename]
	if !ok {
		return nil, fmt.Errorf("vocabulary not found: %s", name)
	}
	return content, nil
}

// WriteStandardVocabularies writes all standard vocabularies to a directory.
// Creates a "vocabularies" subdirectory in the specified output directory.
// Returns error if directory creation or file writing fails.
func WriteStandardVocabularies(outputDir string) error {
	vocabDir := filepath.Join(outputDir, "vocabularies")

	// Create vocabularies directory
	if err := os.MkdirAll(vocabDir, 0755); err != nil {
		return fmt.Errorf("failed to create vocabularies directory: %w", err)
	}

	// Write each vocabulary file
	for filename, content := range vocabularies.Files {
		outputPath := filepath.Join(vocabDir, filename)
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write vocabulary %s: %w", filename, err)
		}
	}

	return nil
}

// WriteVocabulariesToFile writes all standard vocabularies to a single file.
// Each vocabulary is separated by a comment header.
// Useful for single-file archives that want to include vocabularies.
func WriteVocabulariesToFile(outputPath string) error {
	var content []byte

	// Add header
	content = append(content, []byte("# GLX Standard Vocabularies\n\n")...)

	// Write each vocabulary
	for filename, vocabContent := range vocabularies.Files {
		// Add vocabulary header
		header := fmt.Sprintf("# %s\n\n", filename)
		content = append(content, []byte(header)...)

		// Add vocabulary content
		content = append(content, vocabContent...)

		// Add separator
		content = append(content, []byte("\n\n")...)
	}

	// Write to file
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write vocabularies file: %w", err)
	}

	return nil
}
