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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SerializerOptions.Indent sets the indentation of every file written (#1427).
func TestSerializerHonoursIndent(t *testing.T) {
	archive := func() *GLXFile {
		return &GLXFile{Persons: map[string]*Person{"person-a": {Properties: map[string]any{"name": "Ada"}}}}
	}
	twoSpace := "persons:\n  person-a:\n    properties:\n      name: Ada\n"
	fourSpace := "persons:\n    person-a:\n        properties:\n            name: Ada\n"

	for _, tc := range []struct {
		name, indent, want string
	}{
		{"default", DefaultSerializerOptions().Indent, twoSpace},
		{"empty means the default", "", twoSpace},
		{"four spaces", "    ", fourSpace},
		{"not spaces falls back to the default", "\t", twoSpace},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSerializer(&SerializerOptions{Indent: tc.indent})

			single, err := s.SerializeSingleFileBytes(archive())
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(single))

			files, err := s.SerializeMultiFileToMap(archive())
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(files["persons/person-a.glx"]))
		})
	}
}
