package lib

import (
	"fmt"
	"os"
	"path/filepath"

	vocabularies "github.com/genealogix/spec/specification/5-standard-vocabularies"
	"gopkg.in/yaml.v3"
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

// LoadStandardVocabulariesIntoGLX loads all standard vocabularies into a GLXFile.
// This populates the vocabulary maps (EventTypes, RelationshipTypes, etc.) so that
// validation can check references against the standard vocabulary values.
// Returns error if vocabulary parsing fails.
func LoadStandardVocabulariesIntoGLX(glx *GLXFile) error {
	vocabs := vocabularies.Files

	// Load event types
	if data, ok := vocabs["event-types.glx"]; ok {
		var doc struct {
			EventTypes map[string]*EventType `yaml:"event_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.EventTypes = doc.EventTypes
		}
	}

	// Load relationship types
	if data, ok := vocabs["relationship-types.glx"]; ok {
		var doc struct {
			RelationshipTypes map[string]*RelationshipType `yaml:"relationship_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.RelationshipTypes = doc.RelationshipTypes
		}
	}

	// Load place types
	if data, ok := vocabs["place-types.glx"]; ok {
		var doc struct {
			PlaceTypes map[string]*PlaceType `yaml:"place_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.PlaceTypes = doc.PlaceTypes
		}
	}

	// Load source types
	if data, ok := vocabs["source-types.glx"]; ok {
		var doc struct {
			SourceTypes map[string]*SourceType `yaml:"source_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.SourceTypes = doc.SourceTypes
		}
	}

	// Load repository types
	if data, ok := vocabs["repository-types.glx"]; ok {
		var doc struct {
			RepositoryTypes map[string]*RepositoryType `yaml:"repository_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.RepositoryTypes = doc.RepositoryTypes
		}
	}

	// Load participant roles
	if data, ok := vocabs["participant-roles.glx"]; ok {
		var doc struct {
			ParticipantRoles map[string]*ParticipantRole `yaml:"participant_roles"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.ParticipantRoles = doc.ParticipantRoles
		}
	}

	// Load media types
	if data, ok := vocabs["media-types.glx"]; ok {
		var doc struct {
			MediaTypes map[string]*MediaType `yaml:"media_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.MediaTypes = doc.MediaTypes
		}
	}

	// Load confidence levels
	if data, ok := vocabs["confidence-levels.glx"]; ok {
		var doc struct {
			ConfidenceLevels map[string]*ConfidenceLevel `yaml:"confidence_levels"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.ConfidenceLevels = doc.ConfidenceLevels
		}
	}

	// Load quality ratings
	if data, ok := vocabs["quality-ratings.glx"]; ok {
		var doc struct {
			QualityRatings map[string]*QualityRating `yaml:"quality_ratings"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.QualityRatings = doc.QualityRatings
		}
	}

	// Load property vocabularies
	if data, ok := vocabs["person-properties.glx"]; ok {
		var doc struct {
			PersonProperties map[string]*PropertyDefinition `yaml:"person_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.PersonProperties = doc.PersonProperties
		}
	}

	if data, ok := vocabs["event-properties.glx"]; ok {
		var doc struct {
			EventProperties map[string]*PropertyDefinition `yaml:"event_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.EventProperties = doc.EventProperties
		}
	}

	if data, ok := vocabs["relationship-properties.glx"]; ok {
		var doc struct {
			RelationshipProperties map[string]*PropertyDefinition `yaml:"relationship_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.RelationshipProperties = doc.RelationshipProperties
		}
	}

	if data, ok := vocabs["place-properties.glx"]; ok {
		var doc struct {
			PlaceProperties map[string]*PropertyDefinition `yaml:"place_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			glx.PlaceProperties = doc.PlaceProperties
		}
	}

	return nil
}
