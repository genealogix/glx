package glx

import (
	"fmt"

	vocabularies "github.com/genealogix/glx/specification/5-standard-vocabularies"
	"gopkg.in/yaml.v3"
)

// StandardVocabularies returns a map of vocabulary filename to content bytes.
// Uses embedded vocabularies from the specification package.
func StandardVocabularies() map[string][]byte {
	return vocabularies.Files
}

// ListStandardVocabularies returns a list of all embedded vocabulary names (without .glx extension).
func ListStandardVocabularies() []string {
	names := make([]string, 0, len(vocabularies.Files))
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
		return nil, fmt.Errorf("%w: %s", ErrVocabularyNotFound, name)
	}

	return content, nil
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
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing event-types.glx: %w", err)
		}
		glx.EventTypes = doc.EventTypes
	}

	// Load relationship types
	if data, ok := vocabs["relationship-types.glx"]; ok {
		var doc struct {
			RelationshipTypes map[string]*RelationshipType `yaml:"relationship_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing relationship-types.glx: %w", err)
		}
		glx.RelationshipTypes = doc.RelationshipTypes
	}

	// Load place types
	if data, ok := vocabs["place-types.glx"]; ok {
		var doc struct {
			PlaceTypes map[string]*PlaceType `yaml:"place_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing place-types.glx: %w", err)
		}
		glx.PlaceTypes = doc.PlaceTypes
	}

	// Load source types
	if data, ok := vocabs["source-types.glx"]; ok {
		var doc struct {
			SourceTypes map[string]*SourceType `yaml:"source_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing source-types.glx: %w", err)
		}
		glx.SourceTypes = doc.SourceTypes
	}

	// Load repository types
	if data, ok := vocabs["repository-types.glx"]; ok {
		var doc struct {
			RepositoryTypes map[string]*RepositoryType `yaml:"repository_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing repository-types.glx: %w", err)
		}
		glx.RepositoryTypes = doc.RepositoryTypes
	}

	// Load participant roles
	if data, ok := vocabs["participant-roles.glx"]; ok {
		var doc struct {
			ParticipantRoles map[string]*ParticipantRole `yaml:"participant_roles"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing participant-roles.glx: %w", err)
		}
		glx.ParticipantRoles = doc.ParticipantRoles
	}

	// Load media types
	if data, ok := vocabs["media-types.glx"]; ok {
		var doc struct {
			MediaTypes map[string]*MediaType `yaml:"media_types"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing media-types.glx: %w", err)
		}
		glx.MediaTypes = doc.MediaTypes
	}

	// Load confidence levels
	if data, ok := vocabs["confidence-levels.glx"]; ok {
		var doc struct {
			ConfidenceLevels map[string]*ConfidenceLevel `yaml:"confidence_levels"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing confidence-levels.glx: %w", err)
		}
		glx.ConfidenceLevels = doc.ConfidenceLevels
	}

	// Load property vocabularies
	if data, ok := vocabs["person-properties.glx"]; ok {
		var doc struct {
			PersonProperties map[string]*PropertyDefinition `yaml:"person_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing person-properties.glx: %w", err)
		}
		glx.PersonProperties = doc.PersonProperties
	}

	if data, ok := vocabs["event-properties.glx"]; ok {
		var doc struct {
			EventProperties map[string]*PropertyDefinition `yaml:"event_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing event-properties.glx: %w", err)
		}
		glx.EventProperties = doc.EventProperties
	}

	if data, ok := vocabs["relationship-properties.glx"]; ok {
		var doc struct {
			RelationshipProperties map[string]*PropertyDefinition `yaml:"relationship_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing relationship-properties.glx: %w", err)
		}
		glx.RelationshipProperties = doc.RelationshipProperties
	}

	if data, ok := vocabs["place-properties.glx"]; ok {
		var doc struct {
			PlaceProperties map[string]*PropertyDefinition `yaml:"place_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing place-properties.glx: %w", err)
		}
		glx.PlaceProperties = doc.PlaceProperties
	}

	if data, ok := vocabs["media-properties.glx"]; ok {
		var doc struct {
			MediaProperties map[string]*PropertyDefinition `yaml:"media_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing media-properties.glx: %w", err)
		}
		glx.MediaProperties = doc.MediaProperties
	}

	if data, ok := vocabs["repository-properties.glx"]; ok {
		var doc struct {
			RepositoryProperties map[string]*PropertyDefinition `yaml:"repository_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing repository-properties.glx: %w", err)
		}
		glx.RepositoryProperties = doc.RepositoryProperties
	}

	if data, ok := vocabs["citation-properties.glx"]; ok {
		var doc struct {
			CitationProperties map[string]*PropertyDefinition `yaml:"citation_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing citation-properties.glx: %w", err)
		}
		glx.CitationProperties = doc.CitationProperties
	}

	if data, ok := vocabs["source-properties.glx"]; ok {
		var doc struct {
			SourceProperties map[string]*PropertyDefinition `yaml:"source_properties"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing source-properties.glx: %w", err)
		}
		glx.SourceProperties = doc.SourceProperties
	}

	return nil
}

// LoadVocabulariesFromMap loads vocabularies from a map of filenames to content into a GLXFile.
// This populates the vocabulary maps (EventTypes, RelationshipTypes, etc.) from the provided files.
// Returns error if vocabulary parsing fails.
func LoadVocabulariesFromMap(vocabFiles map[string][]byte, glx *GLXFile) error {
	for filename, data := range vocabFiles {
		// Parse based on filename
		switch filename {
		case "event-types.glx":
			var doc struct {
				EventTypes map[string]*EventType `yaml:"event_types"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.EventTypes = doc.EventTypes
		case "relationship-types.glx":
			var doc struct {
				RelationshipTypes map[string]*RelationshipType `yaml:"relationship_types"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.RelationshipTypes = doc.RelationshipTypes
		case "place-types.glx":
			var doc struct {
				PlaceTypes map[string]*PlaceType `yaml:"place_types"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.PlaceTypes = doc.PlaceTypes
		case "source-types.glx":
			var doc struct {
				SourceTypes map[string]*SourceType `yaml:"source_types"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.SourceTypes = doc.SourceTypes
		case "repository-types.glx":
			var doc struct {
				RepositoryTypes map[string]*RepositoryType `yaml:"repository_types"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.RepositoryTypes = doc.RepositoryTypes
		case "participant-roles.glx":
			var doc struct {
				ParticipantRoles map[string]*ParticipantRole `yaml:"participant_roles"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.ParticipantRoles = doc.ParticipantRoles
		case "media-types.glx":
			var doc struct {
				MediaTypes map[string]*MediaType `yaml:"media_types"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.MediaTypes = doc.MediaTypes
		case "confidence-levels.glx":
			var doc struct {
				ConfidenceLevels map[string]*ConfidenceLevel `yaml:"confidence_levels"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.ConfidenceLevels = doc.ConfidenceLevels
		case "person-properties.glx":
			var doc struct {
				PersonProperties map[string]*PropertyDefinition `yaml:"person_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.PersonProperties = doc.PersonProperties
		case "event-properties.glx":
			var doc struct {
				EventProperties map[string]*PropertyDefinition `yaml:"event_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.EventProperties = doc.EventProperties
		case "relationship-properties.glx":
			var doc struct {
				RelationshipProperties map[string]*PropertyDefinition `yaml:"relationship_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.RelationshipProperties = doc.RelationshipProperties
		case "place-properties.glx":
			var doc struct {
				PlaceProperties map[string]*PropertyDefinition `yaml:"place_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.PlaceProperties = doc.PlaceProperties
		case "media-properties.glx":
			var doc struct {
				MediaProperties map[string]*PropertyDefinition `yaml:"media_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.MediaProperties = doc.MediaProperties
		case "repository-properties.glx":
			var doc struct {
				RepositoryProperties map[string]*PropertyDefinition `yaml:"repository_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.RepositoryProperties = doc.RepositoryProperties
		case "citation-properties.glx":
			var doc struct {
				CitationProperties map[string]*PropertyDefinition `yaml:"citation_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.CitationProperties = doc.CitationProperties
		case "source-properties.glx":
			var doc struct {
				SourceProperties map[string]*PropertyDefinition `yaml:"source_properties"`
			}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing %s: %w", filename, err)
			}
			glx.SourceProperties = doc.SourceProperties
		}
	}

	return nil
}
