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
// mutated in place; the returned report counts the renames. Person IDs are
// iterated in sorted order so conflict warnings are deterministic.
func migrateGenderToSex(archive *glxlib.GLXFile, warnOut io.Writer) (*MigrateReport, error) {
	report := &MigrateReport{}

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
			fmt.Fprintf(warnOut,
				"Warning: person %q has both 'gender' and 'sex' properties; leaving 'gender' untouched\n",
				personID)
			continue
		}
		person.Properties[glxlib.PersonPropertySex] = val
		delete(person.Properties, glxlib.PersonPropertyGender)
		report.PropertiesRenamed++
	}

	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		if assertion.Property == glxlib.PersonPropertyGender {
			assertion.Property = glxlib.PersonPropertySex
			report.AssertionsRenamed++
		}
	}

	if archive.PersonProperties != nil {
		if def, ok := archive.PersonProperties[glxlib.PersonPropertyGender]; ok && def != nil {
			if _, sexExists := archive.PersonProperties[glxlib.PersonPropertySex]; !sexExists {
				if def.VocabularyType == glxlib.VocabGenderTypes {
					def.VocabularyType = glxlib.VocabSexTypes
				}
				archive.PersonProperties[glxlib.PersonPropertySex] = def
				delete(archive.PersonProperties, glxlib.PersonPropertyGender)
				report.VocabEntriesRenamed++
			}
		}
	}

	// Pre-split gender_types vocabulary (contains "unknown" but not "nonbinary")
	// moves to sex_types to keep the archive self-contained after the rename.
	if len(archive.GenderTypes) > 0 && len(archive.SexTypes) == 0 {
		if _, hasUnknown := archive.GenderTypes["unknown"]; hasUnknown {
			if _, hasNonbinary := archive.GenderTypes["nonbinary"]; !hasNonbinary {
				archive.SexTypes = make(map[string]*glxlib.SexType, len(archive.GenderTypes))
				for key, entry := range archive.GenderTypes {
					if entry == nil {
						continue
					}
					archive.SexTypes[key] = &glxlib.SexType{
						Label:       entry.Label,
						Description: entry.Description,
						GEDCOM:      entry.GEDCOM,
					}
				}
				archive.GenderTypes = nil
				report.VocabEntriesRenamed++
			}
		}
	}

	archive.InvalidateCache()

	return report, nil
}
