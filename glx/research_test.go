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

	parents := person.Research["parents"]
	require.NotNil(t, parents)
	assert.Equal(t, "open", parents.Status)
	assert.Equal(t, "Parents unknown.", parents.Summary)
	require.Len(t, parents.Leads, 2)
	assert.Equal(t, "Candidate A", parents.Leads[0].Description)
	assert.Equal(t, "medium-high", parents.Leads[0].Confidence)
	assert.Equal(t, []string{"Check census"}, parents.Leads[0].NextSteps)
	assert.Equal(t, "eliminated", parents.Leads[1].Confidence)
	assert.Len(t, parents.CompletedResearch, 2)
}
