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
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	yamlBatchSize  = 128
	wasmGCInterval = 4096
)

// SerializeSingleFileBatchedBytes serializes a single GLX archive in small YAML
// batches for memory-constrained callers. Mapping keys have a deterministic
// order, which may differ from SerializeSingleFileBytes.
func (s *DefaultSerializer) SerializeSingleFileBatchedBytes(archive *GLXFile) ([]byte, error) {
	if archive == nil {
		return nil, ErrGLXFileNil
	}

	fields := reflect.ValueOf(archive).Elem()
	for _, field := range fields.Fields() {
		if field.Kind() != reflect.Map || field.Len() <= yamlBatchSize {
			continue
		}
		if s.Options.Validate {
			if err := validateGLXFile(archive); err != nil {
				return nil, fmt.Errorf("validation failed: %w", err)
			}
		}

		return marshalArchiveInBatches(reflect.ValueOf(withoutEmptyMetadata(archive)).Elem())
	}

	return s.SerializeSingleFileBytes(archive)
}

// go-yaml retains an emitter slot for each event until the encoder finishes.
// Start a new encoder for each batch so the slots for a whole archive do not
// have to fit in memory. All batches still form one YAML mapping document.
func marshalArchiveInBatches(fields reflect.Value) ([]byte, error) {
	var output bytes.Buffer
	for descriptor, field := range fields.Fields() {
		tag := descriptor.Tag.Get("yaml")
		if tag == "" || field.IsZero() {
			continue
		}
		key, _, _ := strings.Cut(tag, ",")
		if field.Kind() == reflect.Map && field.Len() > yamlBatchSize {
			if err := appendYAMLCollection(&output, key, field); err != nil {
				return nil, err
			}

			continue
		}
		if field.Kind() == reflect.Map && field.Len() == 0 {
			continue
		}
		encoded, err := yaml.Marshal(map[string]any{key: field.Interface()})
		if err != nil {
			return nil, err
		}
		output.Write(encoded)
	}

	return output.Bytes(), nil
}

func appendYAMLCollection(output *bytes.Buffer, key string, collection reflect.Value) error {
	keys := collection.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })

	for offset := 0; offset < len(keys); offset += yamlBatchSize {
		chunk := keys[offset:min(offset+yamlBatchSize, len(keys))]
		batch := reflect.MakeMapWithSize(collection.Type(), len(chunk))
		for _, id := range chunk {
			batch.SetMapIndex(id, collection.MapIndex(id))
		}

		encoded, err := yaml.Marshal(map[string]any{key: batch.Interface()})
		if err != nil {
			return err
		}
		if offset > 0 {
			// The first line is the collection key. Each batch's entries
			// already have the indentation for that collection.
			_, encoded, _ = bytes.Cut(encoded, []byte("\n"))
		}
		output.Write(encoded)
		// A long synchronous js/wasm callback can allocate faster than
		// concurrent GC can finish. Reclaim the completed batches together.
		if runtime.GOARCH == "wasm" && offset%wasmGCInterval == 0 {
			runtime.GC()
		}
	}

	return nil
}
