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

func TestGenerateTestData(t *testing.T) {
	tests := []struct {
		name      string
		numPeople int
		wantErr   bool
	}{
		{
			name:      "generate for 10 people",
			numPeople: 10,
			wantErr:   false,
		},
		{
			name:      "generate for 0 people",
			numPeople: 0,
			wantErr:   false,
		},
		{
			name:      "generate for 1 person",
			numPeople: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glxFile, err := GenerateTestData(tt.numPeople)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, glxFile)

				// Verify the number of persons
				assert.Len(t, glxFile.Persons, tt.numPeople, "should generate the correct number of persons")

				// For each person, a birth event should be created
				assert.Len(t, glxFile.Events, tt.numPeople, "should generate a birth event for each person")

				// For each birth event, there should be an assertion, citation, and source.
				assert.Len(t, glxFile.Assertions, tt.numPeople, "should generate an assertion for each birth event")
				assert.Len(t, glxFile.Citations, tt.numPeople, "should generate a citation for each birth event")
				assert.Len(t, glxFile.Sources, tt.numPeople, "should generate a source for each birth event")

				// A single repository should be created
				if tt.numPeople > 0 {
					assert.Len(t, glxFile.Repositories, 1, "should generate a single repository")
				}
			}
		})
	}
}
