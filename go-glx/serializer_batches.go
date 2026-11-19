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
	"io"
	"reflect"
	"runtime"
	"slices"
	"strings"
)

const (
	yamlBatchSize  = 128
	wasmGCInterval = 4096
)

// SerializeSingleFileBatchedBytes serializes a single GLX archive in small YAML
// batches for memory-constrained callers. The bytes are the same as
// SerializeSingleFileBytes produces.
func (s *DefaultSerializer) SerializeSingleFileBatchedBytes(archive *GLXFile) ([]byte, error) {
	var output bytes.Buffer
	if err := s.WriteSingleFileBatched(archive, &output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// WriteSingleFileBatched writes the bytes SerializeSingleFileBatchedBytes
// returns to output batch by batch, so a caller that streams them elsewhere
// never holds the whole encoding in one buffer.
func (s *DefaultSerializer) WriteSingleFileBatched(archive *GLXFile, output io.Writer) error {
	if archive == nil {
		return ErrGLXFileNil
	}

	fields := reflect.ValueOf(archive).Elem()
	for _, field := range fields.Fields() {
		if field.Kind() != reflect.Map || field.Len() <= yamlBatchSize {
			continue
		}
		if s.Options.Validate {
			if err := validateGLXFile(archive); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
		}

		return marshalArchiveInBatches(reflect.ValueOf(withoutEmptyMetadata(archive)).Elem(), output)
	}

	encoded, err := s.SerializeSingleFileBytes(archive)
	if err != nil {
		return err
	}
	_, err = output.Write(encoded)

	return err
}

// go-yaml retains an emitter slot for each event until the encoder finishes.
// Start a new encoder for each batch so the slots for a whole archive do not
// have to fit in memory. All batches still form one YAML mapping document.
func marshalArchiveInBatches(fields reflect.Value, output io.Writer) error {
	for descriptor, field := range fields.Fields() {
		tag := descriptor.Tag.Get("yaml")
		if tag == "" || field.IsZero() {
			continue
		}
		key, _, _ := strings.Cut(tag, ",")
		if field.Kind() == reflect.Map && field.Len() > yamlBatchSize {
			if err := appendYAMLCollection(output, key, field); err != nil {
				return err
			}

			continue
		}
		if field.Kind() == reflect.Map && field.Len() == 0 {
			continue
		}
		encoded, err := marshalYAML(map[string]any{key: field.Interface()})
		if err != nil {
			return err
		}
		if _, err := output.Write(encoded); err != nil {
			return err
		}
	}

	return nil
}

func appendYAMLCollection(output io.Writer, key string, collection reflect.Value) error {
	keys := collection.MapKeys()
	// Split in go-yaml's own key order, so the batches concatenate to the
	// same listing as one encoding of the whole map.
	slices.SortFunc(keys, func(a, b reflect.Value) int {
		switch {
		case yamlKeyLess(a.String(), b.String()):
			return -1
		case yamlKeyLess(b.String(), a.String()):
			return 1
		default:
			return 0
		}
	})

	for offset := 0; offset < len(keys); offset += yamlBatchSize {
		chunk := keys[offset:min(offset+yamlBatchSize, len(keys))]
		batch := reflect.MakeMapWithSize(collection.Type(), len(chunk))
		for _, id := range chunk {
			batch.SetMapIndex(id, collection.MapIndex(id))
		}

		encoded, err := marshalYAML(map[string]any{key: batch.Interface()})
		if err != nil {
			return err
		}
		if offset > 0 {
			// The first line is the collection key. Each batch's entries
			// already have the indentation for that collection.
			_, encoded, _ = bytes.Cut(encoded, []byte("\n"))
		}
		if _, err := output.Write(encoded); err != nil {
			return err
		}
		// A long synchronous js/wasm callback can allocate faster than
		// concurrent GC can finish. Reclaim the completed batches together,
		// once there are some to reclaim.
		if runtime.GOARCH == "wasm" && offset > 0 && offset%wasmGCInterval == 0 {
			runtime.GC()
		}
	}

	return nil
}
