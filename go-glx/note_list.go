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
	"strings"

	"gopkg.in/yaml.v3"
)

// NoteList holds one or more notes. It deserializes from either a YAML scalar
// string (single note) or a YAML sequence (multiple notes), ensuring backwards
// compatibility with existing archives that use `notes: "single string"`.
//
// On serialization, a single-element list emits as a plain scalar string,
// and a multi-element list emits as a YAML sequence.
type NoteList []string

// UnmarshalYAML accepts both scalar strings and string sequences.
func (n *NoteList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Value != "" {
			*n = NoteList{node.Value}
		} else {
			*n = nil
		}
		return nil
	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return err
		}
		*n = NoteList(list)
		return nil
	default:
		// Tolerate unexpected types — try scalar decode
		*n = NoteList{node.Value}
		return nil
	}
}

// MarshalYAML emits a scalar for single notes and a sequence for multiple.
func (n NoteList) MarshalYAML() (any, error) {
	if len(n) == 0 {
		return nil, nil
	}
	if len(n) == 1 {
		return n[0], nil
	}
	return []string(n), nil
}

// String returns all notes joined with double newlines (for display).
func (n NoteList) String() string {
	return strings.Join(n, "\n\n")
}

// IsEmpty returns true if there are no notes.
func (n NoteList) IsEmpty() bool {
	return len(n) == 0
}
