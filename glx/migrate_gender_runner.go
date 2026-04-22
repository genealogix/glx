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

// migrateGenderToSex renames the legacy `gender` person property to `sex`
// (and updates related assertions and inlined vocabularies) so that archives
// created before #528 align with the two-field-model split.
//
// It is opt-in via `glx migrate --rename-gender-to-sex`. The archive is
// mutated in place; the returned report counts the renames.
//
// Legacy detection: the rename is skipped entirely when the archive already
// carries post-split person data or assertions — a person with
// `properties.sex` set, a person with `gender == "nonbinary"`, or an
// assertion targeting `sex` or `gender == "nonbinary"`. Vocabulary-only
// signals are intentionally ignored because `mergeStandardVocabularies`
// unconditionally populates `sex_types` and the post-split
// `person_properties` at load time, so those would always look post-split.
// In post-split archives `gender` means identity, and treating it as sex
// would silently corrupt identity data.
func migrateGenderToSex(archive *glxlib.GLXFile, warnOut io.Writer) *MigrateReport {
	if warnOut == nil {
		warnOut = io.Discard
	}
	report := &MigrateReport{}

	if isPostSplitArchive(archive) {
		_, _ = fmt.Fprintln(warnOut,
			"Warning: archive appears to already use the post-split two-field model "+
				"(sex/gender) — skipping --rename-gender-to-sex. Running this migration "+
				"on a post-split archive would corrupt gender-identity data. Any legacy "+
				"`gender` values still in the archive must be migrated manually.")
		report.GenderRenameSkipped = true

		return report
	}

	report.PropertiesRenamed = renamePersonGenderProperties(archive)
	report.AssertionsRenamed = renameGenderAssertions(archive)
	report.VocabEntriesRenamed = renamePersonPropertyDefinition(archive) +
		movePreSplitGenderTypesVocab(archive)

	archive.InvalidateCache()

	return report
}

// isPostSplitArchive returns true when the archive carries data that is
// clearly post-#528 (two-field model already in use). Vocabulary presence is
// NOT a reliable signal — `LoadArchiveWithOptions` merges standard
// vocabularies at load time, so both `sex_types` and the post-split
// `person_properties` are always populated by that point. We look at actual
// person data and assertions instead:
//
//   - Any person has a `sex` property set (someone is already using the new
//     field — treating their `gender` values as sex would corrupt identity
//     data).
//   - Any person has a `gender` value of `nonbinary` (only exists in the
//     post-split gender vocabulary).
//   - Any assertion targets the `sex` property, or asserts `gender =
//     nonbinary`.
func isPostSplitArchive(archive *glxlib.GLXFile) bool {
	for _, person := range archive.Persons {
		if person == nil {
			continue
		}
		if _, hasSex := person.Properties[glxlib.PersonPropertySex]; hasSex {
			return true
		}
		if isIdentityOnlyGenderValue(person.Properties[glxlib.PersonPropertyGender]) {
			return true
		}
	}
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		// Only person-subject assertions count as post-split signals.
		// Non-person subjects (events, relationships) may legitimately
		// carry custom `sex`/`gender` properties unrelated to the split;
		// treating them as post-split would incorrectly skip migration.
		if assertion.Subject.Person == "" {
			continue
		}
		if assertion.Property == glxlib.PersonPropertySex {
			return true
		}
		if assertion.Property == glxlib.PersonPropertyGender &&
			assertion.Value == glxlib.GenderNonbinary {
			return true
		}
	}

	return false
}

// isIdentityOnlyGenderValue reports whether a gender property value contains
// `nonbinary` — the canonical "only exists post-split" marker. Handles the
// string, single-temporal-object, and temporal-list shapes a property can
// take.
func isIdentityOnlyGenderValue(val any) bool {
	switch v := val.(type) {
	case string:
		return v == glxlib.GenderNonbinary
	case map[string]any:
		if s, ok := v["value"].(string); ok {
			return s == glxlib.GenderNonbinary
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == glxlib.GenderNonbinary {
				return true
			}
			if m, ok := item.(map[string]any); ok {
				if s, ok := m["value"].(string); ok && s == glxlib.GenderNonbinary {
					return true
				}
			}
		}
	}

	return false
}

// renamePersonGenderProperties moves person.properties["gender"] → ["sex"].
// Any person already carrying `sex` would have tripped isPostSplitArchive and
// skipped the whole migration, so the iteration never observes a
// gender+sex conflict here.
func renamePersonGenderProperties(archive *glxlib.GLXFile) int {
	count := 0
	for _, personID := range sortedKeys(archive.Persons) {
		person := archive.Persons[personID]
		if person == nil || len(person.Properties) == 0 {
			continue
		}
		val, hasGender := person.Properties[glxlib.PersonPropertyGender]
		if !hasGender {
			continue
		}
		person.Properties[glxlib.PersonPropertySex] = val
		delete(person.Properties, glxlib.PersonPropertyGender)
		count++
	}

	return count
}

// renameGenderAssertions flips assertion.Property from "gender" to "sex"
// only for assertions whose subject is a person. Non-person subjects (events,
// relationships, etc.) may legitimately use a custom `gender` property and
// must not be touched by this migration.
func renameGenderAssertions(archive *glxlib.GLXFile) int {
	count := 0
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		if assertion.Subject.Person == "" {
			continue
		}
		if assertion.Property == glxlib.PersonPropertyGender {
			assertion.Property = glxlib.PersonPropertySex
			count++
		}
	}

	return count
}

// renamePersonPropertyDefinition renames person_properties["gender"] →
// ["sex"] when "sex" is absent, also flipping its vocabulary_type.
func renamePersonPropertyDefinition(archive *glxlib.GLXFile) int {
	if archive.PersonProperties == nil {
		return 0
	}
	def, ok := archive.PersonProperties[glxlib.PersonPropertyGender]
	if !ok || def == nil {
		return 0
	}
	if _, sexExists := archive.PersonProperties[glxlib.PersonPropertySex]; sexExists {
		return 0
	}
	if def.VocabularyType == glxlib.VocabGenderTypes {
		def.VocabularyType = glxlib.VocabSexTypes
	}
	archive.PersonProperties[glxlib.PersonPropertySex] = def
	delete(archive.PersonProperties, glxlib.PersonPropertyGender)

	return 1
}

// movePreSplitGenderTypesVocab moves an inlined pre-split gender_types
// vocabulary (contains "unknown", lacks "nonbinary") to sex_types so the
// archive stays self-contained after the rename. Existing sex_types entries
// are preserved — only keys not already present are copied over.
func movePreSplitGenderTypesVocab(archive *glxlib.GLXFile) int {
	if len(archive.GenderTypes) == 0 {
		return 0
	}
	if _, hasUnknown := archive.GenderTypes["unknown"]; !hasUnknown {
		return 0
	}
	if _, hasNonbinary := archive.GenderTypes["nonbinary"]; hasNonbinary {
		return 0
	}

	if archive.SexTypes == nil {
		archive.SexTypes = make(map[string]*glxlib.SexType, len(archive.GenderTypes))
	}
	for key, entry := range archive.GenderTypes {
		if entry == nil {
			continue
		}
		if _, exists := archive.SexTypes[key]; exists {
			continue
		}
		archive.SexTypes[key] = &glxlib.SexType{
			Label:       entry.Label,
			Description: entry.Description,
			GEDCOM:      entry.GEDCOM,
		}
	}
	archive.GenderTypes = nil

	return 1
}
