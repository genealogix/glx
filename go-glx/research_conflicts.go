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
	"slices"
)

// FactEvaluation is the aggregate verdict for one fact. Comparisons and Selected
// index the original input. Selected includes unresolved disagreements, or the
// resolved disagreements when no unresolved ones remain. Surviving excludes
// disproven claims. Compatibility is never collapsed transitively.
type FactEvaluation struct {
	Comparisons []FactComparison `json:"comparisons"`
	Selected    []int            `json:"selected"`
	Surviving   []int            `json:"surviving"`
	Verdict     Verdict          `json:"verdict"`
	Definite    bool             `json:"definite"`
	Resolved    bool             `json:"resolved"`
	Resolution  string           `json:"resolution,omitempty"`
}

// EvaluateFacts compares one fact and reports its aggregate resolution. It
// validates options, preserves input order, never mutates inputs, and returns
// owned slices. A singleton explicit dispute has a self-pair (Left == Right).
func EvaluateFacts(values []FactValue, definition *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) (FactEvaluation, error) {
	if err := opts.Validate(); err != nil {
		return FactEvaluation{}, err
	}

	return evaluateFacts(values, definition, places, opts), nil
}

func evaluateFacts(values []FactValue, definition *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) FactEvaluation {
	result := FactEvaluation{Comparisons: compareFacts(values, definition, places, opts), Verdict: VerdictAgree}
	selected := make([]bool, len(values))
	for _, c := range result.Comparisons {
		if !c.IsConflict() {
			continue
		}
		selected[c.Left], selected[c.Right] = true, true
		if verdictRank(c.Verdict) > verdictRank(result.Verdict) {
			result.Verdict = c.Verdict
		}
		result.Definite = result.Definite || c.Definite
	}
	if verdictRank(result.Verdict) == 0 {
		for _, c := range result.Comparisons {
			switch c.Verdict {
			case VerdictResolved:
				selected[c.Left], selected[c.Right] = true, true
				result.Verdict = VerdictResolved
				result.Resolved = true
			case VerdictHistory:
				if !result.Resolved {
					result.Verdict = c.Verdict
				}
			case VerdictRefinement:
				if result.Verdict == VerdictAgree {
					result.Verdict = c.Verdict
				}
			}
		}
	}
	var survivingValues []string
	for i, v := range values {
		if selected[i] {
			result.Selected = append(result.Selected, i)
		}
		if !equalStatus(v.Status, statusDisproven) {
			result.Surviving = append(result.Surviving, i)
			if !slices.Contains(survivingValues, v.Value) {
				survivingValues = append(survivingValues, v.Value)
			}
		}
	}
	if result.Resolved && len(survivingValues) == 1 {
		result.Resolution = survivingValues[0]
	}

	return result
}

// ResearchFact is an owned snapshot of a recorded claim and its subject context.
// FactKey and FactProperty identify the comparison group, including legacy vital
// aliases; Subject and Property retain the original assertion context.
// Synthetic marks structural date/place facts compared with legacy assertions or
// duplicate vital events; their IDs identify event fields rather than assertions.
type ResearchFact struct {
	ID                string    `json:"id"`
	Subject           EntityRef `json:"subject"`
	Property          string    `json:"property,omitempty"`
	Fact              FactValue `json:"fact"`
	EventType         string    `json:"event_type,omitempty"`
	RelationshipType  string    `json:"relationship_type,omitempty"`
	PersonRole        string    `json:"person_role,omitempty"`
	FactKey           string    `json:"fact_key"`
	FactProperty      string    `json:"fact_property"`
	Synthetic         bool      `json:"synthetic,omitempty"`
	ParticipantPerson string    `json:"participant_person,omitempty"`
	ParticipantRole   string    `json:"participant_role,omitempty"`
	Citations         []string  `json:"citations,omitempty"`
	Sources           []string  `json:"sources,omitempty"`
	Media             []string  `json:"media,omitempty"`
	Notes             []string  `json:"notes,omitempty"`
}

// CollectPersonFacts gathers claims on an exact person ID and their events and
// relationships, including vital fields needed for legacy and duplicate-event
// comparisons. It does not mutate the archive; modifying returned facts cannot
// alter the archive.
func CollectPersonFacts(archive *GLXFile, personID string) ([]ResearchFact, error) {
	if archive == nil {
		return nil, ErrNilArchive
	}
	if err := requirePerson(archive, personID, "person"); err != nil {
		return nil, err
	}
	collected := collectPersonProofAssertions(archive, personID)
	out := make([]ResearchFact, 0, len(collected))
	for i := range collected {
		out = append(out, researchFact(&collected[i]))
	}

	return out, nil
}

// ConflictAnalysisOptions selects all persons by default, or one exact PersonID.
type ConflictAnalysisOptions struct {
	PersonID   string
	Comparison ComparisonOptions
}

// ConflictFinding identifies an unresolved group affecting one person. Severity
// is high for definite conflicts, medium for possible ones, low for known disputes.
type ConflictFinding struct {
	PersonID string        `json:"person_id"`
	Severity string        `json:"severity"`
	Conflict ConflictGroup `json:"conflict"`
}

// AnalyzeConflicts reports unresolved conflicts in stable person/fact order.
// Missing standard vocabularies are supplied read-only. Unknown selected persons,
// nil archives and invalid options return errors. Resolved groups are omitted.
func AnalyzeConflicts(archive *GLXFile, opts ConflictAnalysisOptions) ([]ConflictFinding, error) {
	if err := opts.Comparison.Validate(); err != nil {
		return nil, err
	}
	prepared, err := researchArchive(archive)
	if err != nil {
		return nil, err
	}
	ids := sortedKeys(prepared.Persons)
	if opts.PersonID != "" {
		if err := requirePerson(prepared, opts.PersonID, "person"); err != nil {
			return nil, err
		}
		ids = []string{opts.PersonID}
	}
	index := newPersonFactIndex(prepared, ids)
	var findings []ConflictFinding
	for _, id := range ids {
		if prepared.Persons[id] == nil {
			continue
		}
		groups := detectProofConflicts(index.collect(prepared, id), prepared, opts.Comparison)
		for i := range groups {
			group := &groups[i]
			if group.Resolved {
				continue
			}
			severity := severityHigh
			if group.Verdict == VerdictPossible {
				severity = severityMedium
			}
			if group.Verdict == VerdictDisputed {
				severity = "low"
			}
			findings = append(findings, ConflictFinding{PersonID: id, Severity: severity, Conflict: *group})
		}
	}

	return findings, nil
}

func researchFact(pa *proofAssertion) ResearchFact {
	f := ResearchFact{ID: pa.id, Subject: pa.a.Subject, Property: pa.a.Property, Fact: AssertionFact(pa.a), EventType: pa.eventType, RelationshipType: pa.relType, PersonRole: pa.personRole, FactKey: pa.factKey, Synthetic: pa.synthetic, Citations: slices.Clone(pa.a.Citations), Sources: slices.Clone(pa.a.Sources), Notes: slices.Clone([]string(pa.a.Notes))}
	f.Media = slices.Clone(pa.a.Media)
	f.FactProperty = pa.factProperty
	if f.FactProperty == "" {
		f.FactProperty = pa.a.Property
	}
	if f.FactKey == "" {
		f.FactKey = pa.a.Subject.Type().String() + ":" + pa.subjectID
	}
	if pa.a.Participant != nil {
		f.ParticipantPerson = pa.a.Participant.Person
		f.ParticipantRole = pa.a.Participant.Role
	}

	return f
}
