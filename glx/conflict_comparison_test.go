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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// This is the same fixture consumed by the library, including #1227's qualifier
// boundaries. All three reporting commands must return the same verdict.
func TestConflictCommandParity(t *testing.T) {
	data, err := os.ReadFile("../go-glx/testdata/conflict-verdicts.json")
	require.NoError(t, err)
	var cases []struct {
		Name, Property, Kind, Reference, Vocabulary string
		Temporal                                    bool
		Width                                       *int
		Values                                      []glxlib.FactValue
		Verdict                                     glxlib.Verdict
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			def := &glxlib.PropertyDefinition{Temporal: &tc.Temporal, ValueType: tc.Kind, ReferenceType: tc.Reference, VocabularyType: tc.Vocabulary}
			subject := glxlib.EntityRef{Person: "p"}
			archive := &glxlib.GLXFile{
				Persons:          map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": "Person"}}},
				PersonProperties: map[string]*glxlib.PropertyDefinition{tc.Property: def},
				Places:           map[string]*glxlib.Place{"hartford": {Name: "Hartford County", ParentID: "connecticut"}, "connecticut": {Name: "Connecticut"}, "springfield-il": {Name: "Springfield"}, "springfield-ma": {Name: "Springfield"}},
				Assertions:       map[string]*glxlib.Assertion{},
			}
			for i, v := range tc.Values {
				archive.Assertions[string(rune('a'+i))] = &glxlib.Assertion{Subject: subject, Property: tc.Property, Value: v.Value, Date: v.Date, Status: v.Status}
			}
			opts := glxlib.ComparisonOptions{ApproximationYears: tc.Width}
			analysis := analyzeConflictsWithOptions(archive, opts)
			proof := detectProofConflicts(collectPersonProofAssertions(archive, "p"), archive, opts)
			evidence := collectEvidence(archive, subject, tc.Property, opts)
			severity := map[glxlib.Verdict]string{glxlib.VerdictDefinite: "high", glxlib.VerdictPossible: "medium", glxlib.VerdictDisputed: "low"}[tc.Verdict]
			if severity == "" {
				require.Empty(t, analysis)
				require.Empty(t, evidence.Conflicts)
				if tc.Verdict == glxlib.VerdictResolved {
					require.Len(t, proof, 1)
					require.True(t, proof[0].Resolved)
				} else {
					require.Empty(t, proof)
				}
			} else {
				require.Len(t, analysis, 1)
				require.Equal(t, severity, analysis[0].Severity)
				require.Len(t, proof, 1)
				require.Equal(t, tc.Verdict, proof[0].Verdict)
				require.Len(t, evidence.Conflicts, 1)
				require.Equal(t, tc.Verdict, evidence.Conflicts[0].Verdict)
			}
		})
	}
}

func TestConflictScopeDuplicateEventsAndRelationships(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}, "parent": {}},
		Events: map[string]*glxlib.Event{
			"birth-a": {Type: "birth", Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}, {Person: "parent", Role: "parent"}}},
			"birth-b": {Type: "birth", Date: "1852", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}, {Person: "parent", Role: "parent"}}},
		},
		Relationships: map[string]*glxlib.Relationship{"rel": {Type: "marriage", Participants: []glxlib.Participant{{Person: "p", Role: "spouse"}}}},
		Assertions: map[string]*glxlib.Assertion{
			"r1": {Subject: glxlib.EntityRef{Relationship: "rel"}, Property: "started_on", Value: "1900"},
			"r2": {Subject: glxlib.EntityRef{Relationship: "rel"}, Property: "started_on", Value: "1901"},
		},
	}
	issues := analyzeConflicts(archive)
	require.Len(t, issues, 2)
	for _, issue := range issues {
		require.Equal(t, "p", issue.Person)
	}
	proof := buildProof("p", archive.Persons["p"], "birth", archive)
	require.Len(t, proof.Conflicts, 1)
	require.Equal(t, proofConclusionConflicted, proof.Conclusion)
	archive.Events["birth-b"].Date = "1850-03-02"
	require.Len(t, analyzeConflicts(archive), 1)
}

func TestConflictHistoryAndPossibleProof(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": "Mary"}}}, Assertions: map[string]*glxlib.Assertion{
		"a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Smith", Date: "1850", Confidence: "high"},
		"b": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Jones", Date: "1850", Confidence: "high"},
		"c": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Brown"},
	}}
	require.NoError(t, glxlib.LoadStandardVocabulariesIntoGLX(archive))
	proof := buildProof("p", archive.Persons["p"], "identity", archive)
	require.NotEqual(t, proofConclusionConflicted, proof.Conclusion)
	require.NotEqual(t, proofConclusionProven, proof.Conclusion)
	require.Len(t, proof.Undated, 1)
	require.Len(t, proof.Conflicts, 1)
	evidence := collectEvidence(archive, glxlib.EntityRef{Person: "p"}, "name")
	require.Len(t, evidence.Groups, 2)
	require.Len(t, evidence.Undated, 1)
	require.Empty(t, evidence.BestEvidence)
	archive.Assertions["b"].Date = "1870"
	require.Empty(t, analyzeConflicts(archive))
	evidence = collectEvidence(archive, glxlib.EntityRef{Person: "p"}, "name")
	require.Equal(t, glxlib.DateString("1850"), evidence.Groups[0].Date)
	require.Empty(t, evidence.Conflicts)
	require.Empty(t, evidence.Groups[0].BestEvidence)
}

func TestConflictDisprovenSurvivorAndNamesakes(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}}, Assertions: map[string]*glxlib.Assertion{
		"a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_on", Value: "1850", Status: "proven"},
		"b": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_on", Value: "1851", Status: "disproven"},
		"c": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_on", Value: "1852", Status: "disproven"},
	}}
	require.Empty(t, analyzeConflicts(archive))
	result := buildProof("p", archive.Persons["p"], "birth", archive)
	require.Len(t, result.Conflicts, 1)
	require.True(t, result.Conflicts[0].Resolved)
	archive.Places = map[string]*glxlib.Place{"il": {Name: "Springfield"}, "ma": {Name: "Springfield"}}
	archive.Assertions = map[string]*glxlib.Assertion{
		"a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_at", Value: "il"},
		"b": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_at", Value: "ma"},
	}
	evidence := collectEvidence(archive, glxlib.EntityRef{Person: "p"}, "born_at")
	require.Len(t, evidence.Groups, 2)
	require.NotEqual(t, evidence.Groups[0].Value, evidence.Groups[1].Value)
}

func TestProofNonPropertyDisputes(t *testing.T) {
	for _, participant := range []bool{false, true} {
		for _, status := range []string{statusDisputed, statusProven, statusDisproven} {
			t.Run(fmt.Sprintf("participant=%t/status=%s", participant, status), func(t *testing.T) {
				archive := &glxlib.GLXFile{
					Persons:       map[string]*glxlib.Person{"child": {Properties: map[string]any{"name": "Child"}}, "parent": {Properties: map[string]any{"name": "Alleged Father"}}},
					Relationships: map[string]*glxlib.Relationship{"rel": {Type: glxlib.RelationshipTypeParentChild, Participants: []glxlib.Participant{{Person: "child", Role: "child"}, {Person: "parent", Role: "parent"}}}},
					Assertions:    map[string]*glxlib.Assertion{"claim": {Subject: glxlib.EntityRef{Relationship: "rel"}, Status: status, Confidence: "high"}},
				}
				if participant {
					archive.Assertions["claim"].Participant = &glxlib.Participant{Person: "parent", Role: "parent"}
				}
				result := buildProof("child", archive.Persons["child"], topicParentage, archive)
				issues := analyzeConflicts(archive)
				if status == statusDisputed {
					require.Equal(t, proofConclusionConflicted, result.Conclusion)
					require.Len(t, result.Conflicts, 1)
					require.Equal(t, glxlib.VerdictDisputed, result.Conflicts[0].Verdict)
					require.True(t, result.Conflicts[0].Definite)
					require.NotEmpty(t, issues)
					for _, issue := range issues {
						require.Equal(t, severityLow, issue.Severity)
					}
					if participant {
						require.Contains(t, result.Conflicts[0].Values[0].Value, "Alleged Father")
					}
				} else {
					require.Empty(t, result.Conflicts)
					require.Empty(t, issues)
					if status == statusProven {
						require.Equal(t, proofConclusionProven, result.Conclusion)
					} else {
						require.Equal(t, proofConclusionInsufficient, result.Conclusion)
					}
				}
			})
		}
	}
}

func TestProofMarkdownUndatedDetails(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": "Mary Smith"}}},
		Sources: map[string]*glxlib.Source{"source": {Title: "Name register"}},
		Assertions: map[string]*glxlib.Assertion{"undated-name": {
			Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Green", Status: statusDisputed, Confidence: "low",
			Sources: []string{"source"}, Notes: glxlib.NoteList{"Name recorded without a date."},
		}},
	}
	require.NoError(t, glxlib.LoadStandardVocabulariesIntoGLX(archive))
	result := buildProof("p", archive.Persons["p"], topicIdentity, archive)
	require.Empty(t, result.Evidence)
	require.Len(t, result.Undated, 1)
	streams, out, _ := TestIOStreams()
	printProofMarkdown(streams, result)
	_, undated, found := strings.Cut(out.String(), "## Undated")
	require.True(t, found)
	for _, text := range []string{"Name register (source)", "name = Mary Green", "confidence: low", "status: disputed", "Name recorded without a date."} {
		require.Contains(t, undated, text)
	}
}

func TestComparisonCommandHandlersForwardWidth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	original := []byte(`persons:
  p:
    properties: {name: Mary, birth_fact: ABT 1850}
  q:
    properties: {name: Mary, birth_fact: "1853"}
person_properties:
  birth_fact: {label: Birth fact, value_type: date}
assertions:
  a: {subject: {person: p}, property: born_on, value: ABT 1850}
  b: {subject: {person: p}, property: born_on, value: "1853"}
`)
	require.NoError(t, os.WriteFile(path, original, 0o600))
	setString := func(target *string, value string) {
		old := *target
		t.Cleanup(func() { *target = old })
		*target = value
	}
	for _, target := range []*string{&analyzeArchive, &evidenceArchive, &proofArchive, &mergePersonsArchive} {
		setString(target, path)
	}
	setString(&analyzeCheck, conflictsCheck)
	setString(&analyzeFormat, "json")
	setString(&analyzeCountry, "")
	setString(&censusCountryFallback, "")
	setString(&evidenceFormat, "json")
	setString(&proofFormat, "json")
	setString(&proofQuestion, topicBirth)
	setString(&mergePersonsNotesStrategy, "append")
	oldDryRun, oldNewest, oldOldest := mergePersonsDryRun, mergePersonsKeepNewest, mergePersonsKeepOldest
	t.Cleanup(func() {
		mergePersonsDryRun, mergePersonsKeepNewest, mergePersonsKeepOldest = oldDryRun, oldNewest, oldOldest
	})
	mergePersonsDryRun, mergePersonsKeepNewest, mergePersonsKeepOldest = true, false, false
	cases := []struct {
		name     string
		run      func(*cobra.Command, []string) error
		args     []string
		conflict string
	}{
		{"analyze", runAnalyze, []string{"p"}, `"conflict": 1`},
		{"proof", runProof, []string{"p"}, `"conclusion": "CONFLICTED"`},
		{"evidence", runEvidence, []string{"p", "born_on"}, `"verdict": "definite"`},
		{"merge-persons", runMergePersons, []string{"p", "q"}, `Conflict on "birth_fact"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, width := range []string{"", "0", "3", "-1", "10001"} {
				t.Run("width="+width, func(t *testing.T) {
					cmd := &cobra.Command{}
					cmd.Flags().Int(approximationFlag, glxlib.DefaultApproximationYears, "")
					if width != "" {
						require.NoError(t, cmd.ParseFlags([]string{"--" + approximationFlag, width}))
					}
					var runErr error
					var stdout string
					stderr := captureStderr(t, func() { stdout = captureStdout(t, func() { runErr = tc.run(cmd, tc.args) }) })
					if width == "-1" || width == "10001" {
						require.ErrorIs(t, runErr, ErrInvalidApproximation)
						require.Empty(t, stdout+stderr)
					} else {
						require.NoError(t, runErr)
						if width == "3" {
							require.NotContains(t, stdout+stderr, tc.conflict)
						} else {
							require.Contains(t, stdout+stderr, tc.conflict)
						}
					}
					after, err := os.ReadFile(path)
					require.NoError(t, err)
					require.Equal(t, original, after)
				})
			}
			require.Error(t, tc.run(&cobra.Command{}, nil), "a missing flag registration must return before accessing arguments")
		})
	}
}
