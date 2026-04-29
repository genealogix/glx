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
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"

	schema "github.com/genealogix/glx/specification/schema/v1"
)

// Sentinel errors used by the helpers in this file. err113 forbids dynamic
// errors at the call site; the helpers wrap these with %w to add context.
var (
	errYAMLRootNotMap        = errors.New("YAML root is not a map")
	errFragmentMergeConflict = errors.New("conflicting top-level key (non-map values cannot be merged)")
	errFragmentDuplicateID   = errors.New("duplicate entity ID across fragments")
)

// TestExampleArchivesRoundTrip exercises every example archive shipped under
// docs/examples/ through deserialize → re-serialize → re-deserialize, and
// validates that:
//
//  1. Every entity in the re-emitted archive passes its per-entity JSON-schema
//     check (person.schema.json, event.schema.json, etc.).
//  2. The set of keys/values present in the input is preserved on output
//     (catches omitempty drops at the YAML level — struct equality cannot,
//     because a dropped field never reaches the struct).
//
// See issue #296. We intentionally validate per-entity rather than against
// glx-file.schema.json: that top-level schema $refs the vocabulary schemas
// directly (e.g. `"$ref": "vocabularies/event-types.schema.json"`), so
// gojsonschema applies the whole vocabulary-schema document to each individual
// vocab entry — which fails. The CLI validator at glx/validator.go works
// around this by inlining $refs and extracting the inner pattern/additional
// property definitions; reproducing that ~200 lines of resolution logic is
// out of scope for this test. Per-entity schemas have no $refs (verified
// across all schemas in specification/schema/v1/) so they validate cleanly.
func TestExampleArchivesRoundTrip(t *testing.T) {
	const examplesRoot = "../docs/examples"

	examples, err := discoverExampleArchives(examplesRoot)
	require.NoError(t, err, "failed to discover example archives")
	require.NotEmpty(t, examples, "no example archives discovered under %s", examplesRoot)

	entitySchemas, err := loadEntitySchemas()
	require.NoError(t, err, "failed to load entity schemas")

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			roundTripExampleArchive(t, ex, entitySchemas)
		})
	}
}

// exampleArchive describes a discovered example archive.
type exampleArchive struct {
	name string // directory name, e.g. "minimal"
	dir  string // absolute path to the example dir
	// singleFilePath, when non-empty, points at <dir>/archive.glx and the example
	// is treated as single-file. Otherwise the example is multi-file and all .glx
	// files under <dir> (excluding vocabularies/) form the file map.
	singleFilePath string
}

// discoverExampleArchives walks examplesRoot one level deep and returns the
// archives. Single-file archives are detected by the presence of <dir>/archive.glx;
// multi-file archives are detected by the presence of any .glx file outside
// vocabularies/. Directories with no .glx files (e.g. westeros/) are skipped.
func discoverExampleArchives(examplesRoot string) ([]exampleArchive, error) {
	entries, err := os.ReadDir(examplesRoot)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", examplesRoot, err)
	}

	var archives []exampleArchive
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(examplesRoot, entry.Name())
		archivePath := filepath.Join(dir, "archive.glx")
		info, err := os.Lstat(archivePath)
		switch {
		case err == nil && info.Mode().IsRegular():
			archives = append(archives, exampleArchive{
				name:           entry.Name(),
				dir:            dir,
				singleFilePath: archivePath,
			})

			continue
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return nil, fmt.Errorf("lstat %s: %w", archivePath, err)
		}

		hasGLX, err := dirHasOwnGLXFiles(dir)
		if err != nil {
			return nil, err
		}
		if hasGLX {
			archives = append(archives, exampleArchive{name: entry.Name(), dir: dir})
		}
	}

	return archives, nil
}

// dirHasOwnGLXFiles reports whether dir contains at least one .glx file outside
// vocabularies/ (which holds symlinks to canonical spec vocabularies and is not
// part of the example's own content).
func dirHasOwnGLXFiles(dir string) (bool, error) {
	var found bool
	err := walkOwnGLXFiles(dir, func(_, _ string) error {
		found = true

		return filepath.SkipDir
	})
	if err != nil {
		return false, err
	}

	return found, nil
}

// walkOwnGLXFiles invokes fn for each .glx file under dir that is not inside a
// vocabularies/ subdirectory. fn receives the absolute path and the path
// relative to dir; returning filepath.SkipDir or another error stops the walk
// in the standard filepath.Walk way.
//
// vocabularies/ subtrees are pruned at the directory boundary so we don't
// stat the (often-symlinked) entries inside them. Non-regular files
// (symlinks, devices, sockets) at the .glx level are also skipped — defensive
// in case a non-regular .glx ever lands outside vocabularies/.
func walkOwnGLXFiles(dir string, fn func(absPath, rel string) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if filepath.Base(path) == "vocabularies" {
				return filepath.SkipDir
			}

			return nil
		}
		if filepath.Ext(path) != ".glx" || !info.Mode().IsRegular() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		return fn(path, rel)
	})
}

// roundTripExampleArchive runs the schema and semantic-equality assertions
// for one example.
func roundTripExampleArchive(t *testing.T, ex exampleArchive, entitySchemas map[string]*gojsonschema.Schema) {
	t.Helper()

	serializer := NewSerializer(&SerializerOptions{Validate: false, Pretty: true})

	// Step 1: read input bytes; produce the first struct snapshot and the
	// canonical "input map" used for the semantic-equality check.
	var (
		glx1     *GLXFile
		inputMap map[string]any
	)
	if ex.singleFilePath != "" {
		bytes, err := os.ReadFile(ex.singleFilePath)
		require.NoError(t, err, "read %s", ex.singleFilePath)

		glx1, err = serializer.DeserializeSingleFileBytes(bytes)
		require.NoError(t, err, "deserialize single-file %s", ex.singleFilePath)

		inputMap, err = parseYAMLAsMap(bytes)
		require.NoError(t, err, "parse input YAML as map: %s", ex.singleFilePath)
	} else {
		files, err := readMultiFileArchive(ex.dir)
		require.NoError(t, err, "read multi-file archive at %s", ex.dir)
		require.NotEmpty(t, files, "multi-file archive %s contained no .glx fragments", ex.dir)

		var conflicts []string
		glx1, conflicts, err = serializer.DeserializeMultiFileFromMap(files)
		require.NoError(t, err, "deserialize multi-file %s", ex.dir)
		require.Empty(t, conflicts, "duplicate entity IDs across fragments in %s", ex.dir)

		inputMap, err = mergeYAMLFragments(files)
		require.NoError(t, err, "merge input fragments to map: %s", ex.dir)
	}

	// Step 2: re-serialize as single-file. Single-file YAML is the canonical
	// representation of the merged archive and is what we compare against the
	// merged input map.
	remarshaled, err := serializer.SerializeSingleFileBytes(glx1)
	require.NoError(t, err, "re-serialize %s", ex.name)
	require.NotEmpty(t, remarshaled, "re-serialized output is empty for %s", ex.name)

	remarshaledMap, err := parseYAMLAsMap(remarshaled)
	require.NoError(t, err, "parse re-serialized YAML as map")

	// --- Assertion 1: per-entity schema validation ---
	validateEntitiesAgainstSchemas(t, ex.name, remarshaledMap, entitySchemas)

	// --- Assertion 2: semantic equality at the YAML map level ---
	// This is the omitempty-drop detector. Struct equality cannot serve here:
	// a field dropped on (de)serialization never enters the struct, so a
	// struct compare passes vacuously. Comparing the parsed-input map to the
	// parsed-output map catches the drop.
	assert.Equal(t, inputMap, remarshaledMap,
		"re-serialized YAML differs from input at the map level for %s "+
			"(possible omitempty drop, scalar coercion, or key reordering)", ex.name)
}

// entitySchemaFiles maps a top-level YAML key under a GLX archive to the
// schema file that validates each entry under that key.
var entitySchemaFiles = map[string]string{
	EntityTypePersons:       "person.schema.json",
	EntityTypeEvents:        "event.schema.json",
	EntityTypeRelationships: "relationship.schema.json",
	EntityTypePlaces:        "place.schema.json",
	EntityTypeSources:       "source.schema.json",
	EntityTypeCitations:     "citation.schema.json",
	EntityTypeRepositories:  "repository.schema.json",
	EntityTypeMedia:         "media.schema.json",
	EntityTypeAssertions:    "assertion.schema.json",
}

// loadEntitySchemas compiles the per-entity schemas once for reuse across
// every sub-test. Failures here halt the parent test before any sub-tests run.
func loadEntitySchemas() (map[string]*gojsonschema.Schema, error) {
	out := make(map[string]*gojsonschema.Schema, len(entitySchemaFiles))
	for entityType, filename := range entitySchemaFiles {
		bytes, err := schema.EntitySchemas.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read embedded schema %s: %w", filename, err)
		}
		compiled, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(bytes))
		if err != nil {
			return nil, fmt.Errorf("compile %s: %w", filename, err)
		}
		out[entityType] = compiled
	}

	return out, nil
}

// validateEntitiesAgainstSchemas walks the top-level entity maps in the
// re-serialized archive and validates each entry against its per-entity
// schema. Vocabulary blocks (event_types, participant_roles, …) and property
// definition blocks (person_properties, …) are intentionally skipped — those
// would require resolving cross-file $refs from glx-file.schema.json, which
// is the CLI validator's territory.
func validateEntitiesAgainstSchemas(t *testing.T, exampleName string, archiveMap map[string]any, schemas map[string]*gojsonschema.Schema) {
	t.Helper()

	for entityType, schemaForType := range schemas {
		raw, ok := archiveMap[entityType]
		if !ok {
			continue
		}
		entities, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("%s: %s should be a map, got %T", exampleName, entityType, raw)

			continue
		}
		for entityID, entity := range entities {
			entityJSON, err := json.Marshal(entity)
			if err != nil {
				t.Errorf("%s: %s/%s: marshal to JSON for validation: %v",
					exampleName, entityType, entityID, err)

				continue
			}
			result, err := schemaForType.Validate(gojsonschema.NewBytesLoader(entityJSON))
			if err != nil {
				t.Errorf("%s: %s/%s: schema validator errored: %v",
					exampleName, entityType, entityID, err)

				continue
			}
			if !result.Valid() {
				var msgs []string
				for _, e := range result.Errors() {
					msgs = append(msgs, e.String())
				}
				t.Errorf("%s: %s/%s failed schema validation:\n  %s",
					exampleName, entityType, entityID, strings.Join(msgs, "\n  "))
			}
		}
	}
}

// readMultiFileArchive walks a multi-file example directory and returns a path→bytes
// map suitable for DeserializeMultiFileFromMap. Vocabulary symlinks are excluded.
func readMultiFileArchive(dir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := walkOwnGLXFiles(dir, func(absPath, rel string) error {
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", absPath, readErr)
		}
		files[filepath.ToSlash(rel)] = data

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// parseYAMLAsMap parses YAML bytes into a normalized map[string]any. The
// normalization step converts map[any]any (which gopkg.in/yaml.v3 may produce
// for nested maps with non-string keys) into map[string]any so that gojsonschema
// can ingest the value via NewGoLoader and so that map equality works as
// expected with assert.Equal.
func parseYAMLAsMap(data []byte) (map[string]any, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		return map[string]any{}, nil
	}
	normalized := normalizeYAML(doc)
	m, ok := normalized.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: got %T", errYAMLRootNotMap, normalized)
	}

	return m, nil
}

// normalizeYAML recursively converts map[any]any into map[string]any.
// Mirrors normalizeYAMLMap from glx/validator.go, kept private to this test
// file to avoid expanding the go-glx public API for a test-only helper.
func normalizeYAML(val any) any {
	switch v := val.(type) {
	case map[any]any:
		result := make(map[string]any, len(v))
		for key, value := range v {
			result[fmt.Sprintf("%v", key)] = normalizeYAML(value)
		}

		return result
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, value := range v {
			result[key] = normalizeYAML(value)
		}

		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeYAML(item)
		}

		return result
	default:
		return v
	}
}

// mergeYAMLFragments parses every fragment into a map and merges them by
// top-level key (e.g. all `persons:` maps from every fragment combine into one
// `persons:` map). Mirrors what DeserializeMultiFileFromMap does internally.
func mergeYAMLFragments(files map[string][]byte) (map[string]any, error) {
	merged := make(map[string]any)
	for path, data := range files {
		fragmentMap, err := parseYAMLAsMap(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		for topKey, topValue := range fragmentMap {
			existing, ok := merged[topKey]
			if !ok {
				merged[topKey] = topValue

				continue
			}
			// Same top-level key in two fragments: both must be maps, and we
			// merge them shallowly. The example archives don't (and shouldn't)
			// duplicate entity IDs across fragments — that's enforced by the
			// duplicate-detection in DeserializeMultiFileFromMap.
			existingMap, eOK := existing.(map[string]any)
			incomingMap, iOK := topValue.(map[string]any)
			if !eOK || !iOK {
				return nil, fmt.Errorf("%w: %q in %s",
					errFragmentMergeConflict, topKey, path)
			}
			for k, v := range incomingMap {
				if _, dup := existingMap[k]; dup {
					return nil, fmt.Errorf("%w: %q under %q (file %s)",
						errFragmentDuplicateID, k, topKey, path)
				}
				existingMap[k] = v
			}
		}
	}

	return merged, nil
}
