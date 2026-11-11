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
	"maps"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	vocabularies "github.com/genealogix/glx/specification/5-standard-vocabularies"
)

// cachedVocabs holds pre-parsed standard vocabularies. Parsed once on first use
// via sync.Once. LoadStandardVocabulariesIntoGLX shallow-clones each map so
// callers can safely add or remove keys without affecting other GLXFiles.
// The map values (struct pointers) are still shared — callers must not mutate
// the pointed-to VocabularyEntry structs.
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
		// Remove .glx extension (if present) safely.
		name := strings.TrimSuffix(filename, ".glx")
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
// Vocabularies are parsed once and cached; each call shallow-clones the cached
// maps so callers can safely add or remove keys without affecting other GLXFiles.
// The map values (struct pointers) are still shared and must not be mutated.
func LoadStandardVocabulariesIntoGLX(glx *GLXFile) error {
	cachedVocabsOnce.Do(func() {
		cachedVocabs = &GLXFile{}
		cachedVocabsErr = LoadVocabulariesFromMap(vocabularies.Files, cachedVocabs)
	})
	if cachedVocabsErr != nil {
		return cachedVocabsErr
	}

	cachedTypes := cachedVocabs.vocabFields()
	for i, f := range glx.vocabFields() {
		*f.Map = maps.Clone(*cachedTypes[i].Map)
	}
	cachedProps := cachedVocabs.propertyFields()
	for i, f := range glx.propertyFields() {
		*f.Map = maps.Clone(*cachedProps[i].Map)
	}

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
func loadVocabulary(filename string, data []byte, glx *GLXFile) error {
	for _, f := range glx.vocabFields() {
		if f.Filename() == filename {
			return unmarshalVocab(filename, data, f.Key, f.Map)
		}
	}
	for _, f := range glx.propertyFields() {
		if f.Filename() == filename {
			return unmarshalVocab(filename, data, f.Key, f.Map)
		}
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
