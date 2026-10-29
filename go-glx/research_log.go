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
	"strings"
)

// PersonIDs returns the IDs of every person the log is about: its subject
// (when the subject is a person) followed by each lead's candidate persons, in
// order of first appearance and without duplicates.
func (r *ResearchLog) PersonIDs() []string {
	if r == nil {
		return nil
	}
	var ids []string
	if r.Subject != nil && r.Subject.Person != "" {
		ids = append(ids, r.Subject.Person)
	}
	for i := range r.Leads {
		for _, id := range r.Leads[i].Persons {
			if id != "" && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

// ReferencesEntity reports whether id (compared case-insensitively) is the
// log's subject, of any type, or a candidate person in one of its leads.
func (r *ResearchLog) ReferencesEntity(id string) bool {
	if r == nil || id == "" {
		return false
	}
	if r.Subject != nil && strings.EqualFold(r.Subject.ID(), id) {
		return true
	}
	for i := range r.Leads {
		for _, p := range r.Leads[i].Persons {
			if strings.EqualFold(p, id) {
				return true
			}
		}
	}

	return false
}

// CountSearchResults returns how many of the log's searches have the given
// result (e.g. SearchResultNotSearched for outstanding planned searches).
func (r *ResearchLog) CountSearchResults(result string) int {
	if r == nil {
		return 0
	}
	n := 0
	for i := range r.Searches {
		if r.Searches[i].Result == result {
			n++
		}
	}

	return n
}

// IsOpen reports whether the lead is still under investigation: neither
// eliminated nor confirmed. A lead with no status counts as open, since
// nothing has ruled it in or out yet.
func (l *ResearchLead) IsOpen() bool {
	return l.Status != LeadStatusEliminated && l.Status != LeadStatusConfirmed
}
