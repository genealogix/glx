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
func migrateGenderToSex(archive *glxlib.GLXFile, warnOut io.Writer) (*MigrateReport, error) {
	if warnOut == nil {
		warnOut = io.Discard
	}
	report := &MigrateReport{}

	report.PropertiesRenamed = renamePersonGenderProperties(archive, warnOut)
	report.AssertionsRenamed = renameGenderAssertions(archive)
	report.VocabEntriesRenamed = renamePersonPropertyDefinition(archive) +
		movePreSplitGenderTypesVocab(archive)

	archive.InvalidateCache()

	return report, nil
}

// renamePersonGenderProperties moves person.properties["gender"] → ["sex"]
// when "sex" is absent. Person IDs are iterated in sorted order so conflict
// warnings are deterministic.
func renamePersonGenderProperties(archive *glxlib.GLXFile, warnOut io.Writer) int {
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
		if _, hasSex := person.Properties[glxlib.PersonPropertySex]; hasSex {
			_, _ = fmt.Fprintf(warnOut,
				"Warning: person %q has both 'gender' and 'sex' properties; leaving 'gender' untouched\n",
				personID)

			continue
		}
		person.Properties[glxlib.PersonPropertySex] = val
		delete(person.Properties, glxlib.PersonPropertyGender)
		count++
	}

	return count
}

// renameGenderAssertions flips assertion.Property from "gender" to "sex".
func renameGenderAssertions(archive *glxlib.GLXFile) int {
	count := 0
	for _, assertion := range archive.Assertions {
		if assertion == nil {
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
