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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestResearchBlockDeserialization(t *testing.T) {
	data := []byte(`persons:
  person-test:
    properties:
      name:
        value: "Mary Green"
    research:
      parents:
        status: "open"
        summary: "Parents unknown."
        leads:
          - description: "Candidate A"
            details: "Strong evidence"
            confidence: medium-high
            next_steps:
              - "Check census"
          - description: "Candidate B"
            confidence: eliminated
        completed_research:
          - "Pension file reviewed"
          - "1880 census checked"
`)
	var glx glxlib.GLXFile
	err := yaml.Unmarshal(data, &glx)
	require.NoError(t, err)

	person := glx.Persons["person-test"]
	require.NotNil(t, person)
	require.NotNil(t, person.Research)

	parents, ok := person.Research["parents"]
	require.True(t, ok, "parents research topic should exist")
	assert.Equal(t, "open", parents.Status)
	assert.Equal(t, "Parents unknown.", parents.Summary)
	require.Len(t, parents.Leads, 2)
	assert.Equal(t, "Candidate A", parents.Leads[0].Description)
	assert.Equal(t, "medium-high", parents.Leads[0].Confidence)
	assert.Equal(t, []string{"Check census"}, parents.Leads[0].NextSteps)
	assert.Equal(t, "eliminated", parents.Leads[1].Confidence)
	assert.Len(t, parents.CompletedResearch, 2)
}
