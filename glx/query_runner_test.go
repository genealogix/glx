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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryPersons_BasicFamily(t *testing.T) {
	err := queryEntities("persons", queryOpts{Archive: "../docs/examples/basic-family"})
	require.NoError(t, err)
}

func TestQueryPersons_NameFilter(t *testing.T) {
	err := queryEntities("persons", queryOpts{
		Archive: "../docs/examples/complete-family",
		Name:    "John",
	})
	require.NoError(t, err)
}

func TestQueryPersons_BornBefore(t *testing.T) {
	err := queryEntities("persons", queryOpts{
		Archive:    "../docs/examples/complete-family",
		BornBefore: 1860,
	})
	require.NoError(t, err)
}

func TestQueryPersons_BornAfter(t *testing.T) {
	err := queryEntities("persons", queryOpts{
		Archive:   "../docs/examples/complete-family",
		BornAfter: 1870,
	})
	require.NoError(t, err)
}

func TestQueryEvents_TypeFilter(t *testing.T) {
	err := queryEntities("events", queryOpts{
		Archive: "../docs/examples/complete-family",
		Type:    "birth",
	})
	require.NoError(t, err)
}

func TestQueryEvents_BeforeFilter(t *testing.T) {
	err := queryEntities("events", queryOpts{
		Archive: "../docs/examples/complete-family",
		Before:  1860,
	})
	require.NoError(t, err)
}

func TestQueryAssertions(t *testing.T) {
	err := queryEntities("assertions", queryOpts{
		Archive:    "../docs/examples/complete-family",
		Confidence: "high",
	})
	require.NoError(t, err)
}

func TestQuerySources(t *testing.T) {
	err := queryEntities("sources", queryOpts{Archive: "../docs/examples/complete-family"})
	require.NoError(t, err)
}

func TestQueryRelationships(t *testing.T) {
	err := queryEntities("relationships", queryOpts{Archive: "../docs/examples/complete-family"})
	require.NoError(t, err)
}

func TestQueryPlaces(t *testing.T) {
	err := queryEntities("places", queryOpts{Archive: "../docs/examples/complete-family"})
	require.NoError(t, err)
}

func TestQueryCitations(t *testing.T) {
	err := queryEntities("citations", queryOpts{Archive: "../docs/examples/complete-family"})
	require.NoError(t, err)
}

func TestQueryRepositories(t *testing.T) {
	err := queryEntities("repositories", queryOpts{Archive: "../docs/examples/complete-family"})
	require.NoError(t, err)
}

func TestQueryMedia(t *testing.T) {
	err := queryEntities("media", queryOpts{Archive: "../docs/examples/complete-family"})
	require.NoError(t, err)
}

func TestQueryUnknownEntityType(t *testing.T) {
	err := queryEntities("foobar", queryOpts{Archive: "../docs/examples/basic-family"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown entity type")
}

func TestQueryUnsupportedFlag(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		opts       queryOpts
	}{
		{"born-before on events", "events", queryOpts{Archive: "../docs/examples/basic-family", BornBefore: 1850}},
		{"name on citations", "citations", queryOpts{Archive: "../docs/examples/complete-family", Name: "foo"}},
		{"type on persons", "persons", queryOpts{Archive: "../docs/examples/basic-family", Type: "birth"}},
		{"confidence on places", "places", queryOpts{Archive: "../docs/examples/complete-family", Confidence: "high"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := queryEntities(tt.entityType, tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not supported for entity type")
		})
	}
}

func TestQueryNonexistentArchive(t *testing.T) {
	err := queryEntities("persons", queryOpts{Archive: filepath.Join(t.TempDir(), "does-not-exist")})
	require.Error(t, err)
}

func TestAssertionReferencesSource_DirectSource(t *testing.T) {
	archive := &glxlib.GLXFile{}
	a := &glxlib.Assertion{
		Sources: []string{"source-abc", "source-def"},
	}

	assert.True(t, assertionReferencesSource(a, archive, "source-abc"))
	assert.True(t, assertionReferencesSource(a, archive, "source-def"))
	assert.False(t, assertionReferencesSource(a, archive, "source-xyz"))
}

func TestAssertionReferencesSource_ViaCitation(t *testing.T) {
	archive := &glxlib.GLXFile{
		Citations: map[string]*glxlib.Citation{
			"cit-1": {SourceID: "source-abc"},
			"cit-2": {SourceID: "source-def"},
		},
	}
	a := &glxlib.Assertion{
		Citations: []string{"cit-1", "cit-2"},
	}

	assert.True(t, assertionReferencesSource(a, archive, "source-abc"))
	assert.True(t, assertionReferencesSource(a, archive, "source-def"))
	assert.False(t, assertionReferencesSource(a, archive, "source-xyz"))
}

func TestAssertionReferencesSource_MissingCitation(t *testing.T) {
	archive := &glxlib.GLXFile{
		Citations: map[string]*glxlib.Citation{},
	}
	a := &glxlib.Assertion{
		Citations: []string{"cit-nonexistent"},
	}

	assert.False(t, assertionReferencesSource(a, archive, "source-abc"))
}

func TestQueryAssertions_SourceFilter(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject: glxlib.EntityRef{Person: "person-1"},
				Sources: []string{"source-abc"},
			},
			"a-2": {
				Subject:   glxlib.EntityRef{Person: "person-2"},
				Citations: []string{"cit-1"},
			},
			"a-3": {
				Subject: glxlib.EntityRef{Person: "person-3"},
				Sources: []string{"source-other"},
			},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-1": {SourceID: "source-abc"},
		},
	}

	// Capture stdout to verify filtering
	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	t.Cleanup(func() { r.Close() })
	os.Stdout = w

	err := queryAssertions(archive, queryOpts{Source: "source-abc"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "2 assertion(s) found")
}

func TestQueryAssertions_CitationFilter(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:   glxlib.EntityRef{Person: "person-1"},
				Citations: []string{"cit-1", "cit-2"},
			},
			"a-2": {
				Subject:   glxlib.EntityRef{Person: "person-2"},
				Citations: []string{"cit-3"},
			},
		},
	}

	// Capture stdout to verify filtering
	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	t.Cleanup(func() { r.Close() })
	os.Stdout = w

	err := queryAssertions(archive, queryOpts{Citation: "cit-1"})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "1 assertion(s) found")
}

func TestQueryUnsupportedFlag_SourceOnPersons(t *testing.T) {
	err := queryEntities("persons", queryOpts{
		Archive: "../docs/examples/basic-family",
		Source:  "source-abc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported for entity type")
}

func TestExtractDateYear(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1850", 1850},
		{"1850-01-15", 1850},
		{"ABT 1850", 1850},
		{"BEF 1920-01-15", 1920},
		{"BET 1880 AND 1890", 1880},
		{"", 0},
		{"unknown", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractDateYear(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractPersonName(t *testing.T) {
	tests := []struct {
		name  string
		props map[string]any
		want  string
	}{
		{
			name:  "simple string",
			props: map[string]any{"name": "John Smith"},
			want:  "John Smith",
		},
		{
			name:  "structured map",
			props: map[string]any{"name": map[string]any{"value": "Jane Doe", "fields": map[string]any{"given": "Jane"}}},
			want:  "Jane Doe",
		},
		{
			name:  "temporal list",
			props: map[string]any{"name": []any{map[string]any{"value": "First Name"}}},
			want:  "First Name",
		},
		{
			name:  "primary_name fallback",
			props: map[string]any{"primary_name": "Bob Clark"},
			want:  "Bob Clark",
		},
		{
			name:  "no name property",
			props: map[string]any{"gender": "male"},
			want:  "(unnamed)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			person := &glxlib.Person{Properties: tt.props}
			got := extractPersonName(person)
			assert.Equal(t, tt.want, got)
		})
	}
}
