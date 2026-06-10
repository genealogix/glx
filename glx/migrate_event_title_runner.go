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
	glxlib "github.com/genealogix/glx/go-glx"
)

// migrateStripStaleEventTitles removes the stale " (YEAR)" suffix that GEDCOM
// imports before #1026 appended to auto-generated event titles (including the
// buggy day-as-year form tracked in #1025, e.g. "Birth of John Smith (15)").
// Titles are stored, not regenerated, so archives imported before that fix
// keep the suffix until migrated.
//
// It is opt-in via `glx migrate --strip-event-title-year`. The archive is
// mutated in place; the returned report counts only titles actually changed.
//
// Safety: the strip is gated by glxlib.StripStaleEventTitleYear — a title is
// only touched when, with the suffix removed, it is byte-for-byte identical to
// the title import would auto-generate for that event from its type and
// participant names. Hand-authored titles are never modified, and a second run
// is a no-op (a cleaned title no longer carries a suffix to match).
func migrateStripStaleEventTitles(archive *glxlib.GLXFile) *MigrateReport {
	report := &MigrateReport{}

	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || event.Title == "" {
			continue
		}
		cleaned := glxlib.StripStaleEventTitleYear(
			event.Title, event.Type, eventTitleParticipantNames(archive, event))
		if cleaned != event.Title {
			event.Title = cleaned
			report.EventTitleYearsStripped++
		}
	}

	archive.InvalidateCache()

	return report
}

// eventTitleParticipantNames returns the display names import titled the event
// with: principal participants (individual events) and spouse participants
// (family events), in stored order. ASSO-derived roles (witness, officiant, …)
// are excluded — import never used them for titles, and including them could
// gate a strip on the wrong name.
func eventTitleParticipantNames(archive *glxlib.GLXFile, event *glxlib.Event) []string {
	var names []string
	for _, participant := range event.Participants {
		if participant.Role != glxlib.ParticipantRolePrincipal &&
			participant.Role != glxlib.ParticipantRoleSpouse {
			continue
		}
		names = append(names, glxlib.PersonDisplayName(archive.Persons[participant.Person]))
	}

	return names
}
