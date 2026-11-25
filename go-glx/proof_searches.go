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
	"slices"
	"sort"
	"strings"
)

// ResearchLogQuestionProperty is the research-log property that names the
// proof question (or questions) a log serves, e.g. `question: parentage` or
// `question: [marriage, children]`. Proof lists a log's searches only under
// the questions it serves.
const ResearchLogQuestionProperty = "question"

// coverageLabelProbate is the coverage checklist's label for a probate record.
const coverageLabelProbate = "Probate/will"

// researchLogQuestions returns the proof questions a research log serves for
// the person, and whether the log concerns the person at all. An explicit
// `question` property wins. Otherwise the log's subject decides: a marriage
// serves marriage (a search for a couple's marriage record does not test for
// children, so a marriage log serves children only when its question names
// them), a parent_child relationship parentage (for the child) or children
// (for the parent), a possibly_same_person link identity, and an event the
// question of its type for its principals (and children for a birth's parent).
// A log about an event the person only witnessed, sponsored or informed is not
// about them. A log about the person with no question serves none, and its
// searches are reported as other research.
func researchLogQuestions(log *ResearchLog, personID string, archive *GLXFile) ([]string, bool) {
	if log == nil || log.Subject == nil {
		return nil, false
	}
	var about bool
	var inferred []string
	switch {
	case log.Subject.Person != "":
		about = log.Subject.Person == personID
	case log.Subject.Relationship != "":
		inferred, about = relationshipLogQuestions(archive.Relationships[log.Subject.Relationship], personID)
	case log.Subject.Event != "":
		inferred, about = eventLogQuestions(archive.Events[log.Subject.Event], personID)
	}
	if !about {
		return nil, false
	}
	if explicit := logQuestionProperty(log); len(explicit) > 0 {
		return explicit, true
	}

	return inferred, true
}

// relationshipLogQuestions infers the questions a log about a relationship
// serves for one of its participants.
func relationshipLogQuestions(rel *Relationship, personID string) ([]string, bool) {
	if rel == nil {
		return nil, false
	}
	about := false
	var out []string
	for _, p := range rel.Participants {
		if p.Person != personID {
			continue
		}
		about = true
		switch {
		case rel.Type == RelationshipTypeMarriage || rel.Type == RelationshipTypePartner:
			out = append(out, topicMarriage)
		case IsParentChildRelationshipType(rel.Type) && IsChildSideRole(p.Role):
			out = append(out, topicParentage)
		case IsParentChildRelationshipType(rel.Type) && IsParentSideRole(p.Role):
			out = append(out, topicChildren)
		case rel.Type == RelationshipTypePossiblySamePerson:
			out = append(out, topicIdentity)
		}
	}

	return out, about
}

// eventLogQuestions infers the questions a log about an event serves for one
// of its participants.
func eventLogQuestions(event *Event, personID string) ([]string, bool) {
	if event == nil {
		return nil, false
	}
	about := false
	var out []string
	for _, p := range event.Participants {
		if p.Person != personID {
			continue
		}
		switch {
		case isBirthEventType(event.Type) && isVitalPrincipal(event.Type, p.Role):
			out = append(out, topicBirth, topicParentage)
		case isBirthEventType(event.Type) && IsParentSideRole(p.Role) && !IsStepRole(p.Role):
			out = append(out, topicChildren)
		case isDeathEventType(event.Type) && isVitalPrincipal(event.Type, p.Role):
			out = append(out, topicDeath)
		case isMarriageEventType(event.Type) && isMarriagePrincipal(p.Role):
			out = append(out, topicMarriage)
		default:
			continue
		}
		about = true
	}

	return out, about
}

// logQuestionProperty reads the log's `question` property: a question key or
// alias, or a list of them. Unknown values are ignored.
func logQuestionProperty(log *ResearchLog) []string {
	var raw []string
	switch v := log.Properties[ResearchLogQuestionProperty].(type) {
	case string:
		raw = strings.Split(v, ",")
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				raw = append(raw, s)
			}
		}
	case []string:
		raw = v
	}
	var out []string
	for _, q := range raw {
		if key, ok := canonicalQuestion(q); ok && !slices.Contains(out, key) {
			out = append(out, key)
		}
	}

	return out
}

// collectScopedProofSearches splits the person's logged searches into those
// serving the question and the rest of the research about the person.
func collectScopedProofSearches(personID, topic string, archive *GLXFile) (scoped, other []ProofSearch) {
	for _, logID := range sortedKeys(archive.ResearchLogs) {
		log := archive.ResearchLogs[logID]
		questions, about := researchLogQuestions(log, personID, archive)
		if !about {
			continue
		}
		// Research about the person that names no question still bears on
		// who they were, so it serves identity and no narrower question.
		matches := slices.Contains(questions, topic) || len(questions) == 0 && topic == topicIdentity
		for i := range log.Searches {
			s := &log.Searches[i]
			search := ProofSearch{
				Log:        logID,
				Query:      s.Query,
				Collection: s.Collection,
				Repository: s.RepositoryID,
				Source:     s.SourceID,
				SourceType: proofSearchSourceType(s, archive),
				Result:     s.Result,
				Notes:      strings.Join(s.Notes, "; "),
			}
			if matches {
				scoped = append(scoped, search)
			} else {
				other = append(other, search)
			}
		}
	}

	return scoped, other
}

func proofSearchSourceType(s *Search, archive *GLXFile) string {
	if src := archive.Sources[s.SourceID]; src != nil {
		return src.Type
	}
	if cit := archive.Citations[s.CitationID]; cit != nil {
		if src := archive.Sources[cit.SourceID]; src != nil {
			return src.Type
		}
	}

	return ""
}

// gapSourceTypes returns the source types whose search answers a coverage gap.
func gapSourceTypes(gap *ProofGap) []string {
	switch {
	case gap.Label == "Church records":
		return []string{SourceTypeChurchRegister}
	case strings.HasPrefix(gap.Label, "Birth record"), strings.HasPrefix(gap.Label, "Death record"), strings.HasPrefix(gap.Label, "Marriage record"):
		return []string{SourceTypeVitalRecord, SourceTypeChurchRegister}
	case gap.Label == coverageLabelProbate:
		return []string{SourceTypeProbate}
	case gap.Label == "Land records":
		return []string{SourceTypeLand}
	case gap.Label == "Military records":
		return []string{SourceTypeMilitary}
	case strings.Contains(gap.Label, "census"), strings.Contains(gap.Label, "Census"):
		return []string{SourceTypeCensus}
	default:
		return nil
	}
}

// markSearchedGaps records, on each gap, the question's searches in a source of
// the gap's type, whatever they found. A searched gap is reported with its
// results and no longer heads the next steps.
func markSearchedGaps(gaps []ProofGap, searches []ProofSearch) {
	for i := range gaps {
		types := gapSourceTypes(&gaps[i])
		if len(types) == 0 {
			continue
		}
		counts := map[string]int{}
		for j := range searches {
			if !slices.Contains(types, searches[j].SourceType) {
				continue
			}
			result := searches[j].Result
			if result == SearchResultNotSearched {
				// A planned search covers nothing yet.
				continue
			}
			if result == "" {
				result = "searched"
			}
			counts[result]++
		}
		if len(counts) == 0 {
			continue
		}
		results := make([]string, 0, len(counts))
		for result := range counts {
			results = append(results, result)
		}
		sort.Strings(results)
		total := 0
		parts := make([]string, len(results))
		for k, result := range results {
			total += counts[result]
			parts[k] = fmt.Sprintf("%d %s", counts[result], result)
		}
		noun := "searches"
		if total == 1 {
			noun = "search"
		}
		gaps[i].Searched = fmt.Sprintf("%d %s: %s", total, noun, strings.Join(parts, ", "))
	}
}

// negativeSearchSummary describes a question's searches when nothing answers
// it: all negative (a negative conclusion), or some still outstanding. A search
// not yet run, needing a visit, with an unclear result or in an inaccessible
// source tested nothing, so it is outstanding rather than negative.
func negativeSearchSummary(searches []ProofSearch) (string, bool) {
	if len(searches) == 0 {
		return "", false
	}
	outstanding := 0
	for i := range searches {
		switch searches[i].Result {
		case SearchResultNotSearched, SearchResultRequiresVisit, SearchResultInconclusive, SearchResultUnavailable, "":
			outstanding++
		case SearchResultFound, SearchResultPartial:
			return "", false
		}
	}
	if outstanding > 0 {
		return fmt.Sprintf("%d of %d logged searches still outstanding.", outstanding, len(searches)), false
	}

	return fmt.Sprintf("None found in %d logged searches.", len(searches)), true
}
