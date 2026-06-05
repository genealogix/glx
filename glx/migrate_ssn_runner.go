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
	"fmt"
	"io"

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	// legacyPersonPropertySsn is the pre-#532 person-property key for a national
	// identification number. It was renamed to nationalIDProperty because "ssn"
	// (a US Social Security Number) read as US-centric even though the property
	// always meant the generic concept.
	legacyPersonPropertySsn = "ssn"
	// nationalIDProperty is the post-#532 internationalized key. Both keys map
	// to the GEDCOM SSN tag, so GEDCOM round-tripping is unaffected.
	nationalIDProperty = "national_id"
)

// migrateSsnToNationalID renames the legacy `ssn` person property (and any
// related person-subject assertions and inlined vocabulary definition) to
// `national_id`, so that archives created before #532 align with the
// internationalized property name. GEDCOM compatibility is unaffected — both
// keys carry `gedcom: "SSN"`, so import/export of the GEDCOM SSN tag is
// unchanged.
//
// It is opt-in via `glx migrate --rename-ssn-to-national-id`. The archive is
// mutated in place; the returned report counts the renames.
//
// A person, assertion, or vocabulary definition that already carries
// `national_id` is never overwritten: when both keys are present on the same
// person (or in the vocabulary) the legacy `ssn` value is left in place and a
// warning is emitted so the user can reconcile by hand. That person's `ssn`
// assertions are also left untouched — repointing them to `national_id` would
// attach legacy evidence to the wrong field and leave the kept `ssn` property
// inconsistent with its assertions. Only person-subject assertions are ever
// touched — a non-person subject (event, relationship) may legitimately carry a
// custom `ssn` property unrelated to this migration.
func migrateSsnToNationalID(archive *glxlib.GLXFile, warnOut io.Writer) *MigrateReport {
	if warnOut == nil {
		warnOut = io.Discard
	}
	report := &MigrateReport{}

	// renamePersonSsnProperties records every person it skipped because the
	// person already carried `national_id` (a manual conflict). Those persons'
	// `ssn` assertions must also be left as `ssn` — repointing them would
	// attach legacy evidence to `national_id` and leave the kept `ssn` property
	// inconsistent with its assertions.
	conflicted := make(map[string]bool)
	report.SsnPropertiesRenamed = renamePersonSsnProperties(archive, warnOut, conflicted)
	report.SsnAssertionsRenamed = renameSsnAssertions(archive, conflicted)
	report.SsnVocabEntriesRenamed = renamePersonSsnPropertyDefinition(archive, warnOut)

	archive.InvalidateCache()

	return report
}

// renamePersonSsnProperties moves person.properties["ssn"] → ["national_id"].
// A person already carrying `national_id` is skipped (and recorded in
// conflicted) with a warning rather than overwritten. Persons are visited in
// sorted order so warnings are deterministic.
func renamePersonSsnProperties(archive *glxlib.GLXFile, warnOut io.Writer, conflicted map[string]bool) int {
	count := 0
	for _, personID := range sortedKeys(archive.Persons) {
		person := archive.Persons[personID]
		if person == nil || len(person.Properties) == 0 {
			continue
		}
		val, hasSsn := person.Properties[legacyPersonPropertySsn]
		if !hasSsn {
			continue
		}
		if _, hasNationalID := person.Properties[nationalIDProperty]; hasNationalID {
			conflicted[personID] = true
			_, _ = fmt.Fprintf(warnOut,
				"Warning: person %s carries both legacy `ssn` and `national_id` "+
					"properties — leaving `ssn` (and its assertions) in place to "+
					"avoid overwriting `national_id`. Reconcile manually.\n", personID)

			continue
		}
		person.Properties[nationalIDProperty] = val
		delete(person.Properties, legacyPersonPropertySsn)
		count++
	}

	return count
}

// renameSsnAssertions flips assertion.Property from "ssn" to "national_id" for
// assertions whose subject is a person. Non-person subjects are left untouched,
// as is any assertion whose subject person is in conflicted (kept both keys) —
// repointing those would misattribute the legacy value to `national_id`.
func renameSsnAssertions(archive *glxlib.GLXFile, conflicted map[string]bool) int {
	count := 0
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		if assertion.Subject.Person == "" {
			continue
		}
		if conflicted[assertion.Subject.Person] {
			continue
		}
		if assertion.Property == legacyPersonPropertySsn {
			assertion.Property = nationalIDProperty
			count++
		}
	}

	return count
}

// renamePersonSsnPropertyDefinition renames an inlined person_properties["ssn"]
// definition → ["national_id"] when "national_id" is absent. This matters for
// single-file archives, which carry their vocabulary inline; multi-file
// archives regenerate the standard vocabulary from the embedded definitions on
// write, so the on-disk file is corrected there regardless. An existing
// `national_id` definition is never overwritten.
func renamePersonSsnPropertyDefinition(archive *glxlib.GLXFile, warnOut io.Writer) int {
	if archive.PersonProperties == nil {
		return 0
	}
	def, ok := archive.PersonProperties[legacyPersonPropertySsn]
	if !ok || def == nil {
		return 0
	}
	if _, exists := archive.PersonProperties[nationalIDProperty]; exists {
		_, _ = fmt.Fprintln(warnOut,
			"Warning: vocabulary defines both legacy `ssn` and `national_id` "+
				"person properties — leaving the `ssn` definition in place. "+
				"Reconcile manually.")

		return 0
	}
	archive.PersonProperties[nationalIDProperty] = def
	delete(archive.PersonProperties, legacyPersonPropertySsn)

	return 1
}
