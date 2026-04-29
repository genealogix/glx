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
// `properties.sex` set, a person whose `gender` carries a non-legacy-sex
// value (e.g. `nonbinary`, `two-spirit`, or any custom identity vocab key),
// or an assertion expressing either signal. Vocabulary-only signals are
// intentionally ignored because `mergeStandardVocabularies` unconditionally
// populates `sex_types` and the post-split `person_properties` at load
// time, so those would always look post-split. In post-split archives
// `gender` means identity, and treating it as sex would silently corrupt
// identity data.
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
//   - Any person has a meaningful `sex` value (someone is already using the
//     new field — treating their `gender` values as sex would corrupt
//     identity data). Empty or placeholder shapes like `sex: ""`, `sex: {}`,
//     or `sex: []` do NOT count — those are malformed/partial and the legacy
//     data still lives in `gender`.
//   - Any person has a `gender` value that is NOT one of the legacy sex
//     vocabulary keys (male/female/unknown/other/not_recorded). The
//     standard `nonbinary` and any custom identity values (e.g.
//     `two-spirit`, `fluid`) all qualify — moving them into `sex` would
//     corrupt identity data.
//   - Any assertion expresses either signal above for a person subject.
func isPostSplitArchive(archive *glxlib.GLXFile) bool {
	for _, person := range archive.Persons {
		if person == nil {
			continue
		}
		if propertyScalar(person.Properties[glxlib.PersonPropertySex]) != "" {
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
			assertion.Value != "" && !isLegacySexValue(assertion.Value) {
			return true
		}
	}

	return false
}

// isIdentityOnlyGenderValue reports whether a gender property value carries a
// post-split-only identity — any non-empty value that is NOT one of the legacy
// sex vocabulary keys (male/female/unknown/other/not_recorded). Examples:
// `nonbinary`, `two-spirit`, `fluid`, or any custom identity vocab key. Handles
// the string, single-temporal-map, and temporal-list shapes a property can
// take; in a list, ANY non-legacy scalar flags the whole value.
func isIdentityOnlyGenderValue(val any) bool {
	isIdentity := func(s string) bool {
		return s != "" && !isLegacySexValue(s)
	}
	switch v := val.(type) {
	case string:
		return isIdentity(v)
	case map[string]any:
		if s, ok := v["value"].(string); ok {
			return isIdentity(s)
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && isIdentity(s) {
				return true
			}
			if m, ok := item.(map[string]any); ok {
				if s, ok := m["value"].(string); ok && isIdentity(s) {
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

	// Always allocate a fresh map. After mergeStandardVocabularies,
	// archive.SexTypes may alias the embedded standard vocabulary's map
	// (the loader assigns it by reference when the archive doesn't inline
	// its own), so mutating it in place would leak this archive's entries
	// into the shared standard for any other archive loaded by the same
	// process. Copy any existing entries (entry pointers are shared, which
	// is fine — the entries themselves are not mutated).
	existing := archive.SexTypes
	archive.SexTypes = make(map[string]*glxlib.VocabularyEntry, len(existing)+len(archive.GenderTypes))
	for k, v := range existing {
		archive.SexTypes[k] = v
	}
	for key, entry := range archive.GenderTypes {
		if entry == nil {
			continue
		}
		if _, exists := archive.SexTypes[key]; exists {
			continue
		}
		// Clone the full entry so optional fields (Category, AppliesTo,
		// MimeType) on user-supplied entries survive the move.
		cloned := *entry
		if entry.AppliesTo != nil {
			cloned.AppliesTo = append([]string(nil), entry.AppliesTo...)
		}
		archive.SexTypes[key] = &cloned
	}
	archive.GenderTypes = nil

	return 1
}
