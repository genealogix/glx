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
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// errUnexpectedNoteKind is returned when notes contains an unexpected YAML node type.
var errUnexpectedNoteKind = errors.New("notes: expected string or list")

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
		if node.Tag == "!!null" || node.Value == "" {
			*n = nil
		} else {
			*n = NoteList{node.Value}
		}

		return nil
	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return err
		}
		if len(list) == 0 {
			*n = nil
		} else {
			*n = NoteList(list)
		}

		return nil
	case yaml.AliasNode:
		if node.Alias != nil {
			return n.UnmarshalYAML(node.Alias)
		}
		*n = nil

		return nil
	default:
		return fmt.Errorf("%w, got YAML node kind %d", errUnexpectedNoteKind, node.Kind)
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
