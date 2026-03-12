// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package glx

import (
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"

	vocabularies "github.com/genealogix/glx/specification/5-standard-vocabularies"
)

// cachedVocabs holds pre-parsed standard vocabularies. Parsed once on first use
// via sync.Once, then vocabulary map pointers are shared across all GLXFiles.
// This is safe because vocabularies are read-only after loading.
var (
	cachedVocabs     *GLXFile
	cachedVocabsOnce sync.Once
	cachedVocabsErr  error
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
// Vocabularies are parsed once and cached; subsequent calls share the same map
// pointers (safe because vocabularies are read-only after loading).
func LoadStandardVocabulariesIntoGLX(glx *GLXFile) error {
	cachedVocabsOnce.Do(func() {
		cachedVocabs = &GLXFile{}
		cachedVocabsErr = LoadVocabulariesFromMap(vocabularies.Files, cachedVocabs)
	})
	if cachedVocabsErr != nil {
		return cachedVocabsErr
	}

	glx.EventTypes = cachedVocabs.EventTypes
	glx.RelationshipTypes = cachedVocabs.RelationshipTypes
	glx.PlaceTypes = cachedVocabs.PlaceTypes
	glx.SourceTypes = cachedVocabs.SourceTypes
	glx.RepositoryTypes = cachedVocabs.RepositoryTypes
	glx.ParticipantRoles = cachedVocabs.ParticipantRoles
	glx.MediaTypes = cachedVocabs.MediaTypes
	glx.ConfidenceLevels = cachedVocabs.ConfidenceLevels
	glx.PersonProperties = cachedVocabs.PersonProperties
	glx.EventProperties = cachedVocabs.EventProperties
	glx.RelationshipProperties = cachedVocabs.RelationshipProperties
	glx.PlaceProperties = cachedVocabs.PlaceProperties
	glx.MediaProperties = cachedVocabs.MediaProperties
	glx.RepositoryProperties = cachedVocabs.RepositoryProperties
	glx.CitationProperties = cachedVocabs.CitationProperties
	glx.SourceProperties = cachedVocabs.SourceProperties

	return nil
}

// LoadVocabulariesFromMap loads vocabularies from a map of filenames to content into a GLXFile.
// This populates the vocabulary maps (EventTypes, RelationshipTypes, etc.) from the provided files.
// Returns error if vocabulary parsing fails.
func LoadVocabulariesFromMap(vocabFiles map[string][]byte, glx *GLXFile) error {
	for filename, data := range vocabFiles {
		if err := loadVocabulary(filename, data, glx); err != nil {
			return err
		}
	}

	return nil
}

// loadVocabulary parses a single vocabulary file and assigns its contents to the
// appropriate field on the GLXFile. Unrecognized filenames are silently skipped.
//
//nolint:gocyclo
func loadVocabulary(filename string, data []byte, glx *GLXFile) error {
	switch filename {
	case "event-types.glx":
		return unmarshalVocab(filename, data, "event_types", &glx.EventTypes)
	case "relationship-types.glx":
		return unmarshalVocab(filename, data, "relationship_types", &glx.RelationshipTypes)
	case "place-types.glx":
		return unmarshalVocab(filename, data, "place_types", &glx.PlaceTypes)
	case "source-types.glx":
		return unmarshalVocab(filename, data, "source_types", &glx.SourceTypes)
	case "repository-types.glx":
		return unmarshalVocab(filename, data, "repository_types", &glx.RepositoryTypes)
	case "participant-roles.glx":
		return unmarshalVocab(filename, data, "participant_roles", &glx.ParticipantRoles)
	case "media-types.glx":
		return unmarshalVocab(filename, data, "media_types", &glx.MediaTypes)
	case "confidence-levels.glx":
		return unmarshalVocab(filename, data, "confidence_levels", &glx.ConfidenceLevels)
	case "person-properties.glx":
		return unmarshalVocab(filename, data, "person_properties", &glx.PersonProperties)
	case "event-properties.glx":
		return unmarshalVocab(filename, data, "event_properties", &glx.EventProperties)
	case "relationship-properties.glx":
		return unmarshalVocab(filename, data, "relationship_properties", &glx.RelationshipProperties)
	case "place-properties.glx":
		return unmarshalVocab(filename, data, "place_properties", &glx.PlaceProperties)
	case "media-properties.glx":
		return unmarshalVocab(filename, data, "media_properties", &glx.MediaProperties)
	case "repository-properties.glx":
		return unmarshalVocab(filename, data, "repository_properties", &glx.RepositoryProperties)
	case "citation-properties.glx":
		return unmarshalVocab(filename, data, "citation_properties", &glx.CitationProperties)
	case "source-properties.glx":
		return unmarshalVocab(filename, data, "source_properties", &glx.SourceProperties)
	}

	return nil
}

// unmarshalVocab is a generic helper that unmarshals a YAML vocabulary file into
// the target map pointer. The yamlKey is the top-level YAML key (e.g. "event_types").
func unmarshalVocab[T any](filename string, data []byte, yamlKey string, target *map[string]*T) error {
	// Unmarshal into a generic wrapper keyed by the YAML top-level key
	var raw map[string]map[string]*T
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing %s: %w", filename, err)
	}
	entries, ok := raw[yamlKey]
	if !ok {
		return fmt.Errorf("parsing %s: missing expected key %q", filename, yamlKey)
	}
	*target = entries

	return nil
}
