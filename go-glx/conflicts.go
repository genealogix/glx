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
	"fmt"
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

// ApproximationWidth returns the effective width or ErrInvalidApproximation.
func (o ComparisonOptions) ApproximationWidth() (int, error) {
	if err := o.Validate(); err != nil {
		return 0, err
	}

	return o.approximationWidth(), nil
}

// Validate rejects widths outside 0..10000. A nil width selects the default.
func (o ComparisonOptions) Validate() error {
	if o.ApproximationYears != nil && (*o.ApproximationYears < 0 || *o.ApproximationYears > glxdate.MaxApproximationYears) {
		return fmt.Errorf("%w: must be between 0 and %d", ErrInvalidApproximation, glxdate.MaxApproximationYears)
	}

	return nil
}

func (o ComparisonOptions) approximationWidth() int {
	if o.ApproximationYears == nil {
		return DefaultApproximationYears
	}

	return *o.ApproximationYears
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

// CompareFacts validates options and compares all pairs independently. A nil
// definition explicitly means a non-temporal free-text property; archive-level
// callers should use the research workflows for vocabulary defaults. Invalid
// approximation widths return ErrInvalidApproximation. Compatibility is deliberately
// not transitive: a broad date or ancestor place must not bridge two conflicting
// precise claims. This pure function never modifies its inputs.
// A singleton known dispute is represented by a self-pair (Left == Right).
// Explicit disputes remain visible even when a temporal claim has no known
// period; missing timing only prevents inferring conflicts between values.
func CompareFacts(values []FactValue, definition *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) ([]FactComparison, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	return compareFacts(values, definition, places, opts), nil
}

func compareFacts(values []FactValue, definition *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) []FactComparison {
	var result []FactComparison
	for i, left := range values {
		for j, right := range values {
			if j <= i {
				continue
			}
			c := compareFactPair(left, right, definition, places, opts)
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
			x, y := ad.ValueBounds(opts.approximationWidth()), bd.ValueBounds(opts.approximationWidth())
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

// placeAncestor reports whether ancestor is child or one of its ancestors,
// following parent edges of any period (#225).
func placeAncestor(ancestor, child string, places map[string]*Place) bool {
	if child == "" {
		return false
	}
	if child == ancestor {
		return true
	}

	return (&GLXFile{Places: places}).PlaceHasAncestor(child, ancestor)
}

// ConflictProperty returns vocabulary semantics, including structural event
// fields, whose definitions do not live in the property vocabulary.
func ConflictProperty(archive *GLXFile, subject EntityRef, property string) *PropertyDefinition {
	if archive == nil {
		return nil
	}
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
	if def, defined := definitions[property]; defined {
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
	if subject.Person != "" {
		switch _, field := legacyVitalProperty(property); field {
		case conflictDateType:
			return &PropertyDefinition{ValueType: conflictDateType}
		case eventFieldPlace:
			return &PropertyDefinition{ReferenceType: EntityTypePlaces.String()}
		}
	}

	return nil
}

// AssertionFact preserves the value, period and researcher status of an assertion.
// A nil assertion returns the zero FactValue.
func AssertionFact(a *Assertion) FactValue {
	if a == nil {
		return FactValue{}
	}

	return FactValue{Value: a.Value, Date: a.Date, Confidence: a.Confidence, Status: a.Status}
}
