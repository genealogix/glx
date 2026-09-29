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
	"unicode"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// ComparisonOptions configures every consumer of the conflict engine.
// A nil ApproximationYears uses the default ±2 years; a pointer to zero disables widening.
type ComparisonOptions struct{ ApproximationYears *int }

// DefaultApproximationYears is the default tolerance for approximate date values.
const DefaultApproximationYears = 2

const (
	conflictDateType   = "date"
	conflictPlaceField = "place"
)

// ApproximationWidth returns the effective, non-negative approximation width.
func (o ComparisonOptions) ApproximationWidth() int {
	if o.ApproximationYears == nil {
		return DefaultApproximationYears
	}

	return min(glxdate.MaxApproximationYears, max(0, *o.ApproximationYears))
}

// FactValue is a recorded claim, including the period during which it is true.
type FactValue struct {
	Value      string     `json:"value"`
	Date       DateString `json:"date,omitempty"`
	Confidence string     `json:"confidence,omitempty"`
	Status     string     `json:"status,omitempty"`
}

// Verdict describes the relationship between two recorded values.
type Verdict string

// Shared comparison verdicts.
const (
	VerdictAgree      Verdict = "agree"
	VerdictRefinement Verdict = "refinement"
	VerdictHistory    Verdict = "history"
	VerdictDefinite   Verdict = "definite"
	VerdictPossible   Verdict = "possible"
	VerdictDisputed   Verdict = "known-dispute"
	VerdictResolved   Verdict = "resolved"
)

// FactComparison identifies a pair in the input and its verdict. Definite
// preserves certainty for acknowledged disputes, which still block a proof.
type FactComparison struct {
	Left     int     `json:"left"`
	Right    int     `json:"right"`
	Verdict  Verdict `json:"verdict"`
	Definite bool    `json:"definite,omitempty"`
}

// IsConflict reports whether this is an unresolved disagreement.
func (c FactComparison) IsConflict() bool {
	return c.Verdict == VerdictDefinite || c.Verdict == VerdictPossible || c.Verdict == VerdictDisputed
}

// CompareFacts compares all pairs independently. Compatibility is deliberately
// not transitive: a broad date or ancestor place must not bridge two conflicting
// precise claims. This pure function never modifies its inputs.
func CompareFacts(values []FactValue, definition *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) []FactComparison {
	var result []FactComparison
	for i := range values {
		for j := i + 1; j < len(values); j++ {
			c := compareFactPair(values[i], values[j], definition, places, opts)
			c.Left, c.Right = i, j
			result = append(result, c)
		}
	}
	for i, value := range values {
		if !strings.EqualFold(value.Status, "disputed") {
			continue
		}
		covered := false
		for _, c := range result {
			if c.IsConflict() && (c.Left == i || c.Right == i) {
				covered = true

				break
			}
		}
		if covered {
			continue
		}
		definite := !IsTemporalProperty(definition)
		if !definite {
			date, _ := value.Date.Parse()
			if !date.Timing().Known {
				continue
			}
		}
		result = append(result, FactComparison{Left: i, Right: i, Verdict: VerdictDisputed, Definite: definite})
	}

	return result
}

func compareFactPair(a, b FactValue, def *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) FactComparison {
	verdict := compareValues(a.Value, b.Value, def, places, opts)
	if verdict == VerdictAgree || verdict == VerdictRefinement {
		return FactComparison{Verdict: verdict}
	}
	if IsTemporalProperty(def) {
		ad, _ := glxdate.Parse(string(a.Date)) // Best-effort bounds also cover non-canonical dates.
		bd, _ := glxdate.Parse(string(b.Date))
		switch glxdate.CompareTiming(ad, bd) {
		case glxdate.NoOverlap:
			return FactComparison{Verdict: VerdictHistory}
		case glxdate.PossibleOverlap:
			verdict = VerdictPossible
		case glxdate.DefiniteOverlap:
			verdict = VerdictDefinite
		}
	}
	definite := verdict == VerdictDefinite
	if strings.EqualFold(a.Status, "disproven") || strings.EqualFold(b.Status, "disproven") {
		return FactComparison{Verdict: VerdictResolved}
	}
	if strings.EqualFold(a.Status, "disputed") || strings.EqualFold(b.Status, "disputed") {
		verdict = VerdictDisputed
	}

	return FactComparison{Verdict: verdict, Definite: definite}
}

// IsTemporalProperty reports whether the vocabulary permits a property to change.
func IsTemporalProperty(def *PropertyDefinition) bool {
	return def != nil && def.Temporal != nil && *def.Temporal
}

func compareValues(a, b string, def *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) Verdict {
	if a == b {
		return VerdictAgree
	}
	if def != nil {
		switch {
		case def.ReferenceType == EntityTypePlaces.String():
			if placeAncestor(a, b, places) || placeAncestor(b, a, places) {
				return VerdictRefinement
			}

			return VerdictDefinite
		case def.ReferenceType != "", def.VocabularyType != "":
			return VerdictDefinite
		case def.ValueType == conflictDateType:
			ad, _ := glxdate.Parse(a)
			bd, _ := glxdate.Parse(b)
			x, y := ad.ValueBounds(opts.ApproximationWidth()), bd.ValueBounds(opts.ApproximationWidth())
			if ad.Equal(bd) {
				return VerdictAgree
			}
			if !x.Known || !y.Known || x.Reversed || y.Reversed || ad.Calendar() != bd.Calendar() || ad.CalendarName() != bd.CalendarName() {
				return VerdictPossible
			}
			if x.Outer.Intersects(y.Outer) {
				return VerdictRefinement
			}

			return VerdictDefinite
		}
	}
	if foldFactText(a) == foldFactText(b) {
		return VerdictAgree
	}

	return VerdictDefinite
}

func foldFactText(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if !unicode.IsPunct(r) {
			b.WriteRune(r)
		}
	}

	return strings.Join(strings.Fields(b.String()), " ")
}

func placeAncestor(ancestor, child string, places map[string]*Place) bool {
	seen := make(map[string]bool)
	for child != "" && !seen[child] {
		if child == ancestor {
			return true
		}
		seen[child] = true
		p := places[child]
		if p == nil {
			return false
		}
		child = p.ParentID
	}

	return false
}

// ConflictProperty returns vocabulary semantics, including structural event
// fields, whose definitions do not live in the property vocabulary.
func ConflictProperty(archive *GLXFile, subject EntityRef, property string) *PropertyDefinition {
	var definitions map[string]*PropertyDefinition
	switch subject.Type() {
	case EntityTypePersons:
		definitions = archive.PersonProperties
	case EntityTypeEvents:
		definitions = archive.EventProperties
	case EntityTypeRelationships:
		definitions = archive.RelationshipProperties
	case EntityTypePlaces:
		definitions = archive.PlaceProperties
	}
	if def := definitions[property]; def != nil {
		return def
	}
	if subject.Event != "" {
		switch property {
		case conflictDateType:
			return &PropertyDefinition{ValueType: conflictDateType}
		case conflictPlaceField:
			return &PropertyDefinition{ReferenceType: EntityTypePlaces.String()}
		}
	}
	// Legacy assertions remain readable after the event-fact migration.
	switch property {
	case "born_on", "died_on", "buried_on", "birth_date", "death_date", "burial_date":
		return &PropertyDefinition{ValueType: conflictDateType}
	case "born_at", "died_at", "buried_at", "birth_place", "death_place", "burial_place":
		return &PropertyDefinition{ReferenceType: EntityTypePlaces.String()}
	}

	return nil
}

// AssertionFact preserves the value, period and researcher status of an assertion.
func AssertionFact(a *Assertion) FactValue {
	return FactValue{Value: a.Value, Date: a.Date, Confidence: a.Confidence, Status: a.Status}
}
