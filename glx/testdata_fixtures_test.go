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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invalidFixture is what one testdata/invalid archive must report.
type invalidFixture struct {
	// errors lists substrings the validation output must contain, and every
	// reported error line must match one of them. A fixture that also fails
	// for an unrelated reason (a field the schema later dropped, a vocabulary
	// it never defined) fails this test instead of passing by accident.
	errors []string
	// skip, when set, is why the fixture cannot fail yet.
	skip string
}

// invalidFixtures covers every directory under testdata/invalid. A new
// fixture without an entry here fails TestInvalidFixtures, so each one
// states the error it exists to produce.
var invalidFixtures = map[string]invalidFixture{
	"assertion-participant-and-property": {errors: []string{
		"/assertions/assertion-invalid: 'not' failed",
		"/assertions/assertion-invalid: 'allOf' failed",
		// property without value also trips the property/value pairing rule.
		"/assertions/assertion-invalid: missing property 'value'",
	}},
	"assertion-participant-and-value": {errors: []string{
		"/assertions/assertion-invalid: 'not' failed",
		"/assertions/assertion-invalid: 'allOf' failed",
	}},
	"assertion-participant-invalid-person": {errors: []string{
		"assertions[assertion-invalid].Person references non-existent persons: person-nonexistent",
	}},
	"assertion-participant-invalid-role": {errors: []string{
		"assertions[assertion-invalid].Role references non-existent participant_roles: invalid-role",
	}},
	"broken-references": {errors: []string{
		"assertions[assertion-john-birth-date].Citations[0] references non-existent citations: citation-nonexistent",
		"assertions[assertion-john-birth-date].Sources[0] references non-existent sources: source-nonexistent",
		"assertions[assertion-john-birth-date].Subject references non-existent events: event-nonexistent",
	}},
	"comprehensive-broken-references": {errors: []string{
		"citations[citation-birth].RepositoryID references non-existent repositories: repository-nonexistent",
		"citations[citation-birth].SourceID references non-existent sources: source-nonexistent",
		"events[event-birth].Person references non-existent persons: person-nonexistent",
		"events[event-birth].PlaceID references non-existent places: place-nonexistent",
		"places[place-leeds].ParentID references non-existent places: place-nonexistent-parent",
		"sources[source-register].RepositoryID references non-existent repositories: repository-nonexistent",
	}},
	"duplicate-ids": {errors: []string{
		"conflict persons ID: person-abc123",
	}},
	"invalid-assertion-properties": {
		skip: "the validator does not check assertion property names against a vocabulary yet (genealogix/glx#1224)",
	},
	"invalid-entity-ids": {errors: []string{
		"/persons: additional properties 'person_invalid_id' not allowed",
		"persons[person_invalid_id]: invalid entity ID",
	}},
	"invalid-relationship-participants": {errors: []string{
		"relationships[rel-marriage].Person references non-existent persons: person-nonexistent",
	}},
	"missing-required-fields": {errors: []string{
		"/assertions/assertion-missing-property: 'allOf' failed",
		"/assertions/assertion-missing-property: 'not' failed",
		"/assertions/assertion-missing-sources: 'allOf' failed",
		"/assertions/assertion-missing-sources: 'anyOf' failed",
		"/assertions/assertion-missing-sources: missing property 'citations'",
		"/assertions/assertion-missing-sources: missing property 'media'",
		"/assertions/assertion-missing-sources: missing property 'sources'",
		"/assertions/assertion-missing-subject: missing property 'subject'",
		"/citations/citation-missing-source: missing property 'source'",
		"/events/event-missing-type: missing property 'type'",
		"/media/media-missing-uri: missing property 'uri'",
		"/media/media-with-filepath: additional properties 'file_path' not allowed",
		"/media/media-with-filepath: missing property 'uri'",
		"/places/place-missing-name: missing property 'name'",
		"/relationships/rel-missing-persons: missing property 'participants'",
		"/relationships/rel-missing-type: missing property 'type'",
		"/repositories/repo-missing-name: missing property 'name'",
		"/sources/source-missing-title: missing property 'title'",
	}},
	"place-coords-half-set": {errors: []string{
		"/places/place-latitude-only: properties 'longitude' required, if 'latitude' exists",
		"/places/place-longitude-only: properties 'latitude' required, if 'longitude' exists",
	}},
	"temporal-properties-malformed": {errors: []string{
		"persons[person-invalid].properties.name[0]: temporal item missing required 'value' field",
	}},
}

// validationOutput validates path as a directory archive, listing every
// error, and returns everything it reported: output, error output and the
// returned error, which is where structural failures are listed.
func validationOutput(t *testing.T, path string) (string, error) {
	t.Helper()
	streams, out, errOut := TestIOStreams()
	err := validatePathsShowing(streams, []string{path}, 0)
	text := out.String() + errOut.String()
	if err != nil {
		text += err.Error()
	}

	return text, err
}

// reportedLines returns the lines of text that report one error or warning:
// the "- " bullets every validation stage prints.
func reportedLines(text string) []string {
	var lines []string
	for line := range strings.SplitSeq(text, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "- ") {
			lines = append(lines, trimmed)
		}
	}

	return lines
}

func TestInvalidFixtures(t *testing.T) {
	entries, err := os.ReadDir("testdata/invalid")
	require.NoError(t, err)

	seen := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			fixture, ok := invalidFixtures[name]
			require.True(t, ok, "testdata/invalid/%s has no entry in invalidFixtures: add the errors it must produce", name)
			if fixture.skip != "" {
				t.Skip(fixture.skip)
			}

			text, err := validationOutput(t, filepath.Join("testdata", "invalid", name))
			require.Error(t, err, "an invalid fixture must fail validation")

			for _, want := range fixture.errors {
				assert.Contains(t, text, want)
			}
			for _, line := range reportedLines(text) {
				matched := false
				for _, want := range fixture.errors {
					if strings.Contains(line, want) {
						matched = true

						break
					}
				}
				assert.True(t, matched, "unexpected error, so the fixture fails for another reason: %s", line)
			}
		})
	}
	for name := range invalidFixtures {
		assert.True(t, seen[name], "invalidFixtures lists %s, which is not a directory under testdata/invalid", name)
	}
}

// TestValidFixturesWithWarnings covers the testdata/valid archives that test a
// warning rather than a pass: each must validate, and report exactly the
// listed warnings.
func TestValidFixturesWithWarnings(t *testing.T) {
	cases := []struct {
		name     string
		warnings []string
	}{
		{"unknown-person-properties", []string{
			"persons[person-father]: unknown property 'another_invalid'",
			"persons[person-father]: unknown property 'invalid_property'",
		}},
		// No vocabularies at all: the standard vocabularies fill in, so a
		// person using standard properties needs none.
		{"missing-vocabularies", nil},
		// Should warn about the unknown assertion property once the validator
		// checks assertion property names (genealogix/glx#1224).
		{"assertion-unknown-property", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, err := validationOutput(t, filepath.Join("testdata", "valid", tc.name))
			require.NoError(t, err, text)

			lines := reportedLines(text)
			assert.Len(t, lines, len(tc.warnings), text)
			for _, want := range tc.warnings {
				assert.Contains(t, text, want)
			}
		})
	}
}
