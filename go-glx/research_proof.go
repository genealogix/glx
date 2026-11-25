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

// Proof conclusion levels, ordered from strongest to weakest. These summarize
// where the evidence stands relative to the BCG Genealogical Proof Standard.
const (
	proofConclusionProven       = "PROVEN"
	proofConclusionProbable     = "PROBABLE"
	proofConclusionPossible     = "POSSIBLE"
	proofConclusionInsufficient = "INSUFFICIENT EVIDENCE"
	proofConclusionConflicted   = "CONFLICTED"
)

// Canonical research-question topic keys.
const (
	topicParentage  = "parentage"
	topicBirth      = "birth"
	topicDeath      = "death"
	topicMarriage   = "marriage"
	topicChildren   = "children"
	topicOccupation = "occupation"
	topicIdentity   = "identity"
)

// Assertion status values that drive conflict resolution. These are free-text
// in the spec but the standard vocabulary uses these tokens. (statusDisputed is
// declared in migrate_confidence_runner.go and reused here.)
const (
	statusProven    = "proven"
	statusDisproven = "disproven"
)

// Support scores used to rank how strongly the evidence backs the answer.
const (
	supportNone = iota
	supportWeak
	supportModerate
	supportStrong
)

// proofSupport.Kind values, distinguishing how a piece of support was attached
// to the assertion (a citation referencing a source, or a source directly).
const (
	proofKindCitation = "citation"
	proofKindSource   = "source"
	proofKindMedia    = "media"
)

// ProofSupport is a single citation, source or media object backing an assertion,
// resolved to human-readable form (GPS element 2 — complete citations).
type ProofSupport struct {
	Ref         string `json:"ref"`                    // citation, source or media ID
	Kind        string `json:"kind"`                   // "citation", "source" or "media"
	SourceID    string `json:"source_id,omitempty"`    // source ID, including one linked by citation/media
	SourceTitle string `json:"source_title,omitempty"` // resolved source or media title
	Locator     string `json:"locator,omitempty"`      // citation locator (page, entry, certificate no.)
}

// ProofEvidence is one assertion gathered as evidence for the research question
// (GPS element 3 — analysis of evidence).
type ProofEvidence struct {
	// Participant fields retain raw claim identity alongside the Value label.
	ParticipantPerson string         `json:"participant_person,omitempty"`
	ParticipantRole   string         `json:"participant_role,omitempty"`
	AssertionID       string         `json:"assertion_id"`
	Subject           string         `json:"subject,omitempty"` // subject context (e.g. "birth event (1955)")
	Property          string         `json:"property,omitempty"`
	Value             string         `json:"value,omitempty"`
	Date              string         `json:"date,omitempty"`
	Confidence        string         `json:"confidence,omitempty"`
	Status            string         `json:"status,omitempty"`
	Notes             string         `json:"notes,omitempty"`
	Support           []ProofSupport `json:"support,omitempty"`
	// Counterparts are the people the assertion relates the person to: the
	// parent of a parent_child line, the spouse of a marriage line.
	Counterparts []string `json:"counterparts,omitempty"`
}

// ProofGap is a missing record that would strengthen the proof (GPS element 1 —
// reasonably exhaustive search), sourced from the coverage checklist.
type ProofGap struct {
	Label       string `json:"label"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
	// Searched summarizes this question's logged searches in a source of the
	// gap's type, e.g. "12 searches: 11 not_found, 1 requires_visit". A
	// searched gap is still listed but no longer heads the next steps.
	Searched string `json:"searched,omitempty"`
	// relationshipID is the couple relationship a marriage-record gap is for.
	relationshipID string
}

// ConflictValue is one competing value within a conflict.
type ConflictValue struct {
	Value      string `json:"value"`
	Confidence string `json:"confidence,omitempty"`
	Status     string `json:"status,omitempty"`
}

// ConflictGroup reports disagreements on one fact, including singleton known
// disputes. Subject and Values are display labels; Facts retain raw identifiers
// and claims. Evaluation.Selected identifies the participating claims in Facts.
type ConflictGroup struct {
	// Facts retain raw IDs, values and provenance; Evaluation indexes this slice.
	Facts      []ResearchFact  `json:"facts"`
	Evaluation FactEvaluation  `json:"evaluation"`
	Verdict    Verdict         `json:"verdict"`
	Definite   bool            `json:"definite,omitempty"`
	Subject    string          `json:"subject,omitempty"`
	Property   string          `json:"property"`
	Values     []ConflictValue `json:"values"`
	Resolved   bool            `json:"resolved"`
	Resolution string          `json:"resolution,omitempty"`
}

// ProofSearch is one logged search, including searches that found nothing
// (negative evidence supporting GPS element 1).
type ProofSearch struct {
	Log        string `json:"log,omitempty"`
	SourceType string `json:"source_type,omitempty"`
	Query      string `json:"query,omitempty"`
	Collection string `json:"collection,omitempty"`
	Repository string `json:"repository,omitempty"`
	Source     string `json:"source,omitempty"`
	Result     string `json:"result,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// ProofExcludedAlternative is a candidate answer the evidence rules out (every
// assertion about it disproven) or leaves contested (every surviving assertion
// disputed): a parent, a spouse, a child, a same-person link. It remains
// visible apart from the answers, and never counts toward the conclusion.
type ProofExcludedAlternative struct {
	PersonID     string   `json:"person_id,omitempty"`
	Name         string   `json:"name"`
	Relationship string   `json:"relationship,omitempty"`
	Event        string   `json:"event,omitempty"`
	Role         string   `json:"role,omitempty"`
	Value        string   `json:"value,omitempty"`
	Status       string   `json:"status,omitempty"`
	Assertions   []string `json:"assertions,omitempty"`
	Citations    []string `json:"citations,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// ProofResult is the full structured proof argument for one research question.
type ProofResult struct {
	Answers  []ProofAnswer              `json:"answers,omitempty"`
	Excluded []ProofExcludedAlternative `json:"alternatives_excluded,omitempty"`
	// OtherSearches are the searches of research logs about the person that
	// do not serve this question (see ResearchLogQuestionProperty).
	OtherSearches []ProofSearch   `json:"other_searches,omitempty"`
	Undated       []ProofEvidence `json:"undated,omitempty"`
	PersonID      string          `json:"person_id"`
	PersonName    string          `json:"person_name"`
	Question      string          `json:"question"`
	QuestionText  string          `json:"question_text"`
	Evidence      []ProofEvidence `json:"evidence"`
	Gaps          []ProofGap      `json:"gaps,omitempty"`
	Conflicts     []ConflictGroup `json:"conflicts"`
	Searches      []ProofSearch   `json:"searches,omitempty"`
	Conclusion    string          `json:"conclusion"`
	Summary       string          `json:"summary,omitempty"`
}

// buildProof assembles the proof argument for a person and research topic.
func buildProof(personID string, person *Person, topic string, archive *GLXFile, opts ProofOptions) *ProofResult {
	name := PersonDisplayName(person)
	if name == "" {
		name = personID
	}

	all := collectPersonProofAssertions(archive, personID)
	all = appendClaimOnlyParticipations(all, archive, personID)
	all = appendProxyVitalProofFields(all, archive, personID, topic)
	standings := NewRelationshipStandingIndex(archive)
	var parentLinks []ParentChildLink
	if topic == topicParentage || topic == topicChildren {
		parentLinks = ParentChildLinks(archive, standings)
	}

	var relevant []proofAssertion
	for i := range all {
		if assertionRelevant(topic, &all[i], personID) && (topic != topicParentage || parentageLinkRelevant(&all[i], personID, parentLinks, archive)) {
			relevant = append(relevant, all[i])
		}
	}
	candidates := buildProofCandidates(topic, personID, archive, relevant, parentLinks, standings)

	evidence := make([]ProofEvidence, 0, len(relevant))
	var undated []ProofEvidence
	for i := range relevant {
		if relevant[i].synthetic {
			continue
		}
		ev := buildProofEvidence(&relevant[i], personID, archive)
		def := ConflictProperty(archive, relevant[i].a.Subject, relevant[i].a.Property)
		date, _ := relevant[i].a.Date.Parse()
		if IsTemporalProperty(def) && !date.Timing().Known {
			undated = append(undated, ev)
		} else {
			evidence = append(evidence, ev)
		}
	}

	sort.SliceStable(evidence, func(i, j int) bool {
		a, _ := DateString(evidence[i].Date).Parse()
		b, _ := DateString(evidence[j].Date).Parse()

		return a.Timing().Outer.Start < b.Timing().Outer.Start
	})
	// A rejected candidate's assertions stay in the evidence, but its own
	// status already reports them: they never enter conflict detection, so an
	// excluded alternative cannot leave the question CONFLICTED.
	conflicts := detectProofConflicts(withoutRejected(relevant, candidates.rejected), archive, opts.Comparison)
	if topic == topicDeath {
		conflicts = append(conflicts, vitalChronologyConflicts(archive, personID, all, opts.Comparison)...)
	}
	conflicts = append(conflicts, candidates.conflicts...)
	conclusionEvidence := relevant
	excluded := candidates.alternatives
	if topic == topicParentage {
		parents, _ := parentageCandidates(personID, archive, parentLinks, standings)
		parents = slices.DeleteFunc(parents, func(c parentCandidate) bool { return !answersInclude(candidates.answers, c.personID) })
		conflicts = append(conflicts, parentageConflicts(personID, parents, archive, opts.Comparison)...)
		conclusionEvidence = parentageSupport(relevant, personID, parentLinks, standings)
	}
	conclusionEvidence = slices.DeleteFunc(slices.Clone(conclusionEvidence), func(pa proofAssertion) bool { return candidates.rejected[pa.a] })
	gaps := collectProofGaps(personID, person, topic, archive, opts.Coverage)
	if gap := proxyVitalProofGap(topic, relevant, archive, personID); gap != nil {
		gaps = append(gaps, *gap)
	}
	if topic == topicMarriage {
		drop := excludedMarriageRelationships(&candidates)
		gaps = slices.DeleteFunc(gaps, func(g ProofGap) bool { return g.relationshipID != "" && drop[g.relationshipID] })
	}
	searches, otherSearches := collectScopedProofSearches(personID, topic, archive)
	markSearchedGaps(gaps, searches)

	conclusion, summary := concludeProof(topic, personID, archive, conclusionEvidence, conflicts, gaps, &candidates, searches)

	return &ProofResult{
		PersonID:      personID,
		PersonName:    name,
		Question:      topic,
		QuestionText:  questionText(topic, name),
		Evidence:      evidence,
		Undated:       undated,
		Gaps:          gaps,
		Conflicts:     conflicts,
		Answers:       candidates.answers,
		Excluded:      excluded,
		Searches:      searches,
		OtherSearches: otherSearches,
		Conclusion:    conclusion,
		Summary:       summary,
	}
}

// answersInclude reports whether a person is among the surviving answers.
func answersInclude(answers []ProofAnswer, personID string) bool {
	for i := range answers {
		if answers[i].PersonID == personID {
			return true
		}
	}

	return false
}

// withoutRejected returns the assertions no rejected candidate holds.
func withoutRejected(relevant []proofAssertion, rejected map[*Assertion]bool) []proofAssertion {
	if len(rejected) == 0 {
		return relevant
	}

	return slices.DeleteFunc(slices.Clone(relevant), func(pa proofAssertion) bool { return rejected[pa.a] })
}

// appendClaimOnlyParticipations adds the assertions that name the person as a
// participant of an event or relationship that does not list them: a claim
// that they took part, or, disproven, a record ruled out as theirs.
func appendClaimOnlyParticipations(all []proofAssertion, archive *GLXFile, personID string) []proofAssertion {
	seen := make(map[string]bool, len(all))
	for i := range all {
		seen[all[i].id] = true
	}
	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil || seen[id] || a.Participant == nil || a.Participant.Person != personID {
			continue
		}
		switch {
		case a.Subject.Event != "":
			if event := archive.Events[a.Subject.Event]; event != nil {
				all = append(all, proofAssertion{id: id, a: a, subjectID: a.Subject.Event, eventType: event.Type, personRole: a.Participant.Role})
			}
		case a.Subject.Relationship != "":
			if rel := archive.Relationships[a.Subject.Relationship]; rel != nil {
				all = append(all, proofAssertion{id: id, a: a, subjectID: a.Subject.Relationship, relType: rel.Type, personRole: a.Participant.Role})
			}
		}
	}

	return all
}

// proofTopic defines a supported research question and the input aliases that
// map to it.
type proofTopic struct {
	key     string
	aliases []string
}

// proofTopics lists supported research questions in display order. Question
// strings are matched case-insensitively against the key and aliases.
var proofTopics = []proofTopic{
	{key: topicParentage, aliases: []string{"parents", ParticipantRoleParent, "father", "mother", "ancestry"}},
	{key: topicBirth, aliases: []string{"born", "birthdate"}},
	{key: topicDeath, aliases: []string{"died", "deathdate", EventTypeBurial}},
	{key: topicMarriage, aliases: []string{ParticipantRoleSpouse, "marriages"}},
	{key: topicChildren, aliases: []string{ParticipantRoleChild, "family", "issue", "offspring"}},
	{key: topicOccupation, aliases: []string{"office", "profession", "trade"}},
	{key: topicIdentity, aliases: []string{PersonPropertyName, "who"}},
}

// canonicalQuestion maps an input question to its canonical topic key.
func canonicalQuestion(input string) (string, bool) {
	q := strings.ToLower(strings.TrimSpace(input))
	if q == "" {
		return "", false
	}
	for _, t := range proofTopics {
		if t.key == q || slices.Contains(t.aliases, q) {
			return t.key, true
		}
	}

	return "", false
}

// proofQuestionKeys returns the canonical question keys for error messages.
func proofQuestionKeys() []string {
	keys := make([]string, len(proofTopics))
	for i, t := range proofTopics {
		keys[i] = t.key
	}

	return keys
}

// questionText renders the human-readable research question for a person.
func questionText(topic, name string) string {
	switch topic {
	case topicParentage:
		return "Who are the parents of " + name + "?"
	case topicBirth:
		return "When and where was " + name + " born?"
	case topicDeath:
		return "When and where did " + name + " die?"
	case topicMarriage:
		return "Whom did " + name + " marry?"
	case topicChildren:
		return "Who were the children of " + name + "?"
	case topicOccupation:
		return "What occupation or office did " + name + " hold?"
	case topicIdentity:
		return "Who was " + name + "?"
	default:
		return name
	}
}

// proofAssertion couples an assertion with its resolved subject context so
// relevance and conflict logic can reason about it without re-resolving.
type proofAssertion struct {
	id           string
	a            *Assertion
	subjectID    string // person, event, or relationship ID the assertion is about
	eventType    string // set when the subject is an event the person participates in
	relType      string // set when the subject is a relationship the person participates in
	factKey      string // shared identity of a person's vital fact
	factProperty string // canonical date/place field, retaining a.Property as provenance
	personRole   string // the person's role in the subject event or relationship
	synthetic    bool   // structural comparison input; never proof evidence or support
}

// subjectIsPerson reports whether the assertion's subject is the target person
// directly (rather than one of their events or relationships).
func (pa *proofAssertion) subjectIsPerson() bool {
	return pa.eventType == "" && pa.relType == ""
}

// collectPersonProofAssertions gathers every assertion whose subject is the
// person, or an event/relationship the person participates in.
func collectPersonProofAssertions(archive *GLXFile, personID string) []proofAssertion {
	return newPersonFactIndex(archive, []string{personID}).collect(archive, personID)
}

// Property-name sets for matching legacy direct-on-person assertions to a topic.
var (
	parentPropertyNames = map[string]bool{
		parentageFather: true, parentageMother: true, ParticipantRoleParent: true, "parents": true,
		"father_name": true, "mother_name": true,
	}
)

// assertionRelevant reports whether an assertion is evidence for the topic.
func assertionRelevant(topic string, pa *proofAssertion, personID string) bool {
	if (topic == topicBirth || topic == topicDeath) && pa.a.Participant != nil && pa.a.Participant.Person != personID {
		return false
	}
	legacyKind, _ := legacyVitalProperty(pa.a.Property)
	switch topic {
	case topicParentage:
		return parentageAssertionRelevant(pa)
	case topicBirth:
		if isBirthEventType(pa.eventType) {
			return isVitalPrincipal(pa.eventType, pa.personRole)
		}

		return pa.subjectIsPerson() && legacyKind == EventTypeBirth
	case topicDeath:
		if isDeathEventType(pa.eventType) {
			return isVitalPrincipal(pa.eventType, pa.personRole)
		}

		return pa.subjectIsPerson() && (legacyKind == EventTypeDeath || legacyKind == EventTypeBurial)
	case topicMarriage:
		if pa.relType == RelationshipTypeMarriage || pa.relType == RelationshipTypePartner {
			return isMarriagePrincipal(pa.personRole)
		}

		return isMarriageEventType(pa.eventType) && isMarriagePrincipal(pa.personRole)
	case topicChildren:
		return childrenAssertionRelevant(pa)
	case topicOccupation:
		// A claim on the person, or on their participation in an event: the
		// assertion format puts an event participant's occupation under
		// participant.properties, not in property.
		if pa.subjectIsPerson() {
			return pa.a.Property == PersonPropertyOccupation
		}
		p := pa.a.Participant

		return pa.eventType != "" && p != nil && p.Person == personID &&
			(pa.a.Property == PersonPropertyOccupation || p.Properties[PersonPropertyOccupation] != nil)
	case topicIdentity:
		// Who a person was rests on everything recorded about them: their own
		// assertions, the events they take part in (in whatever role), and the
		// possibly_same_person links that join or separate them from a namesake.
		return pa.subjectIsPerson() ||
			pa.eventType != "" && (pa.a.Participant == nil || pa.a.Participant.Person == personID) ||
			pa.relType == RelationshipTypePossiblySamePerson
	default:
		return false
	}
}

// childrenAssertionRelevant reports whether an assertion bears on the person's
// children: a non-step parent_child link where the person is the parent, or a
// birth or baptism in which the person is recorded as parent.
func childrenAssertionRelevant(pa *proofAssertion) bool {
	if IsParentChildRelationshipType(pa.relType) {
		return !strings.EqualFold(pa.relType, RelationshipTypeStepParent) && IsParentSideRole(pa.personRole) && !IsStepRole(pa.personRole) &&
			(pa.a.Property == "" || pa.a.Property == relationshipPropertyType)
	}

	return isBirthEventType(pa.eventType) && IsParentSideRole(pa.personRole) && !IsStepRole(pa.personRole)
}

func parentageAssertionRelevant(pa *proofAssertion) bool {
	if IsParentChildRelationshipType(pa.relType) && IsChildSideRole(pa.personRole) {
		return true
	}
	if isBirthEventType(pa.eventType) {
		// A baptism or christening names the parents as a birth record does.
		if !isVitalPrincipal(pa.eventType, pa.personRole) {
			return false
		}
		if parentPropertyNames[pa.a.Property] {
			return true
		}

		return isVitalPrincipal(pa.eventType, pa.personRole) && pa.a.Participant != nil && pa.a.Participant.Role == ParticipantRoleParent
	}

	return pa.subjectIsPerson() && parentPropertyNames[pa.a.Property]
}

func isBirthEventType(t string) bool {
	return t == EventTypeBirth || t == EventTypeBaptism || t == EventTypeChristening
}

func isDeathEventType(t string) bool {
	return t == EventTypeDeath || t == EventTypeBurial || t == EventTypeCremation
}

func isMarriageEventType(t string) bool {
	switch t {
	case EventTypeMarriage, EventTypeMarriageLicense,
		EventTypeMarriageBanns, EventTypeMarriageContract,
		EventTypeEngagement:
		return true
	default:
		return false
	}
}

// buildProofEvidence converts a relevant assertion into a display-ready
// evidence record with resolved citations and values.
func buildProofEvidence(pa *proofAssertion, personID string, archive *GLXFile) ProofEvidence {
	a := pa.a
	subject, counterparts := describeProofEvidenceSubject(pa, personID, archive)

	evidence := ProofEvidence{
		AssertionID:  pa.id,
		Subject:      subject,
		Counterparts: counterparts,
		Property:     a.Property,
		Value:        proofAssertionValue(a, archive),
		Date:         string(a.Date),
		Confidence:   a.Confidence,
		Status:       a.Status,
		Notes:        firstLine(a.Notes.String()),
		Support:      buildProofSupport(a, archive),
	}
	if a.Participant != nil {
		evidence.ParticipantPerson = a.Participant.Person
		evidence.ParticipantRole = a.Participant.Role
	}

	return evidence
}

// describeProofSubject returns a short label for an assertion's subject when it
// is an event or relationship; the person subject needs no label.
func describeProofSubject(pa *proofAssertion, archive *GLXFile) string {
	switch {
	case pa.eventType != "":
		label := pa.eventType + " event"
		if ev := archive.Events[pa.subjectID]; ev != nil && ev.Date != "" {
			label += " (" + string(ev.Date) + ")"
		}

		return label
	case pa.relType != "":
		return pa.relType + " relationship"
	default:
		return ""
	}
}

// describeProofEvidenceSubject labels an evidence line's subject from the
// person's side, naming the other party a relationship assertion is about:
// "child of Hans Roth (p-hans) [rel-pc]", "marriage to Eva (p-eva) [rel-m]".
// Event subjects keep their date and add the person's role when it is not a
// principal one. It also returns the counterparts' person IDs.
func describeProofEvidenceSubject(pa *proofAssertion, personID string, archive *GLXFile) (string, []string) {
	if pa.eventType != "" {
		label := describeProofSubject(pa, archive)
		if pa.a.Participant == nil && pa.personRole != "" && !isVitalPrincipal(pa.eventType, pa.personRole) && !isMarriagePrincipal(pa.personRole) {
			label += " as " + pa.personRole
		}

		return label, nil
	}
	rel := archive.Relationships[pa.subjectID]
	if pa.relType == "" || rel == nil {
		return describeProofSubject(pa, archive), nil
	}
	prefix, want := relationshipCounterpartRule(rel.Type, pa.personRole)
	if want == nil {
		return describeProofSubject(pa, archive), nil
	}
	ids := relationshipCounterparts(pa, personID, rel, want)
	if len(ids) == 0 {
		return describeProofSubject(pa, archive), nil
	}
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = personName(archive, id)
		if names[i] != id {
			names[i] += " (" + id + ")"
		}
	}

	return prefix + strings.Join(names, " and ") + " [" + pa.subjectID + "]", ids
}

// relationshipCounterpartRule returns how an evidence line names the other
// party of a relationship from the person's side, and which participants count
// as that party; nil when the relationship type has no such reading.
func relationshipCounterpartRule(relType, personRole string) (string, func(Participant) bool) {
	anyone := func(Participant) bool { return true }
	switch {
	case IsParentChildRelationshipType(relType) && IsChildSideRole(personRole):
		return "child of ", func(p Participant) bool { return IsParentSideRole(p.Role) }
	case IsParentChildRelationshipType(relType) && IsParentSideRole(personRole):
		return "parent of ", func(p Participant) bool { return IsChildSideRole(p.Role) }
	case relType == RelationshipTypeMarriage:
		return "marriage to ", anyone
	case relType == RelationshipTypePartner:
		return "partnership with ", anyone
	case relType == RelationshipTypePossiblySamePerson:
		return "possibly same person as ", anyone
	default:
		return "", nil
	}
}

// relationshipCounterparts returns the other party an assertion is about: its
// participant when that is someone on the other side, otherwise every
// participant on the other side of the relationship. A participant on the
// person's own side (the co-parent of a child) is not a counterpart.
func relationshipCounterparts(pa *proofAssertion, personID string, rel *Relationship, want func(Participant) bool) []string {
	if p := pa.a.Participant; p != nil && p.Person != "" && p.Person != personID && want(*p) {
		return []string{p.Person}
	}
	var ids []string
	for _, p := range rel.Participants {
		if p.Person != "" && p.Person != personID && want(p) && !slices.Contains(ids, p.Person) {
			ids = append(ids, p.Person)
		}
	}

	return ids
}

// buildProofSupport resolves an assertion's citations, sources and media into complete,
// human-readable references.
func buildProofSupport(a *Assertion, archive *GLXFile) []ProofSupport {
	support := make([]ProofSupport, 0, len(a.Citations)+len(a.Sources)+len(a.Media))

	for _, citID := range a.Citations {
		s := ProofSupport{Ref: citID, Kind: proofKindCitation}
		if cit, ok := archive.Citations[citID]; ok && cit != nil {
			s.SourceID = cit.SourceID
			s.SourceTitle = resolveSourceTitle(cit.SourceID, archive)
			s.Locator = citationProperty(cit, "locator")
		}
		support = append(support, s)
	}

	for _, srcID := range a.Sources {
		support = append(support, ProofSupport{
			Ref:         srcID,
			Kind:        proofKindSource,
			SourceID:    srcID,
			SourceTitle: resolveSourceTitle(srcID, archive),
		})
	}
	for _, mediaID := range a.Media {
		s := ProofSupport{Ref: mediaID, Kind: proofKindMedia}
		if media := archive.Media[mediaID]; media != nil {
			s.SourceTitle = media.Title
			s.SourceID = media.Source
		}
		support = append(support, s)
	}

	return support
}

// resolveProofValue renders only references declared for the assertion's property.
func resolveProofValue(value string, subject EntityRef, property string, archive *GLXFile) string {
	return resolveAssertionValue(value, property, subject, archive)
}

func proofAssertionValue(a *Assertion, archive *GLXFile) string {
	if a.Participant != nil {
		return participantEvidenceValue(a.Participant, archive)
	}

	return resolveProofValue(a.Value, a.Subject, a.Property, archive)
}

// detectProofConflicts finds relevant assertions that disagree on the same
// subject/property and reports whether the disagreement has been resolved via
// the assertion `status` field (proven over disproven).
func detectProofConflicts(relevant []proofAssertion, archive *GLXFile, options ...ComparisonOptions) []ConflictGroup {
	type conflictKey struct {
		subject  string
		property string
	}
	type conflictGroup struct {
		label      string // human-readable subject label, "" for the person subject
		assertions []*Assertion
		facts      []ResearchFact
	}

	groups := make(map[conflictKey]*conflictGroup)
	var order []conflictKey
	var conflicts []ConflictGroup
	for i := range relevant {
		pa := &relevant[i]
		if pa.a.Property == "" || pa.a.Value == "" {
			if conflict, disputed := nonPropertyProofDispute(pa, archive, comparisonOptions(options)); disputed {
				conflicts = append(conflicts, conflict)
			}

			continue
		}
		subjectKey := pa.a.Subject.Type().String() + ":" + pa.subjectID
		if pa.factKey != "" {
			subjectKey = pa.factKey
		}
		property := pa.a.Property
		if pa.factProperty != "" {
			property = pa.factProperty
		}
		key := conflictKey{subject: subjectKey, property: property}
		g, seen := groups[key]
		if !seen {
			g = &conflictGroup{label: describeProofSubject(pa, archive)}
			groups[key] = g
			order = append(order, key)
		}
		g.assertions = append(g.assertions, pa.a)
		g.facts = append(g.facts, researchFact(pa))
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].subject != order[j].subject {
			return order[i].subject < order[j].subject
		}

		return order[i].property < order[j].property
	})

	for _, key := range order {
		g := groups[key]
		raw := make([]FactValue, len(g.assertions))
		for i, a := range g.assertions {
			raw[i] = comparisonFact(a, archive)
		}
		evaluation := evaluateFacts(raw, ConflictProperty(archive, g.assertions[0].Subject, g.assertions[0].Property), archive.Places, comparisonOptions(options))
		selected := make([]*Assertion, 0, len(evaluation.Selected))
		for _, i := range evaluation.Selected {
			selected = append(selected, g.assertions[i])
		}
		if len(selected) == 0 {
			continue
		}
		values := distinctProofValues(selected, archive)

		// Disambiguate which subject the conflict belongs to. Fall back to the
		// raw subject ID when there is no descriptive label (e.g. the person
		// subject) so machine consumers can always tell groups apart.
		subject := g.label
		if subject == "" && key.subject != "" {
			subject = key.subject
		}

		resolved := evaluation.Resolved
		resolution := resolveProofValue(evaluation.Resolution, g.assertions[0].Subject, g.assertions[0].Property, archive)
		if resolved && resolution == "" {
			resolution = "Disproven claims excluded."
		}
		conflicts = append(conflicts, ConflictGroup{
			Facts: g.facts, Evaluation: evaluation,
			Verdict:    evaluation.Verdict,
			Definite:   evaluation.Definite,
			Subject:    subject,
			Property:   key.property,
			Values:     values,
			Resolved:   resolved,
			Resolution: resolution,
		})
	}

	return conflicts
}

// distinctProofValues returns the distinct values asserted for a
// subject/property, keeping the highest-confidence value and combining the
// statuses of every assertion for that value (see combineStatus). De-duplication
// keys on the raw asserted value, not its resolved display form, so two distinct
// IDs that resolve to the same name (e.g. two "Springfield" places) remain
// separate and the conflict is still detected — matching analyze_conflicts.go,
// which keeps raw IDs distinct and resolves only for display.
func distinctProofValues(assertions []*Assertion, archive *GLXFile) []ConflictValue {
	type agg struct {
		display    string
		confidence string
		status     string
	}
	seen := make(map[string]*agg)
	var order []string

	for _, a := range assertions {
		cur, exists := seen[a.Value]
		if !exists {
			seen[a.Value] = &agg{display: proofAssertionValue(a, archive), confidence: a.Confidence, status: a.Status}
			order = append(order, a.Value)

			continue
		}
		if researchConfidenceRank(a.Confidence) < researchConfidenceRank(cur.confidence) {
			cur.confidence = a.Confidence
		}
		cur.status = combineStatus(cur.status, a.Status)
	}

	sort.Strings(order)
	out := make([]ConflictValue, len(order))
	for i, raw := range order {
		out[i] = ConflictValue{Value: seen[raw].display, Confidence: seen[raw].confidence, Status: seen[raw].status}
	}
	disambiguateConflictValues(out, order)

	return out
}

// disambiguateConflictValues qualifies any display value shared by more than one
// distinct raw value with its raw ID, so two entities with the same name (e.g.
// two "Springfield" places) stay distinguishable, e.g. "Springfield (place-il)".
// Values whose display already equals the raw value are left untouched.
func disambiguateConflictValues(values []ConflictValue, raw []string) {
	counts := make(map[string]int, len(values))
	for i := range values {
		counts[values[i].Value]++
	}
	for i := range values {
		if counts[values[i].Value] > 1 && values[i].Value != raw[i] {
			values[i].Value = fmt.Sprintf("%s (%s)", values[i].Value, raw[i])
		}
	}
}

// combineStatus merges the status of another assertion for the same value into
// the running aggregate. An explicitly `disputed` status, or two differing
// non-empty statuses (e.g. one assertion says `proven`, another `disproven`),
// escalate to `disputed` so resolveConflict never auto-resolves a value whose
// own evidence disagrees. An empty status carries no information and is ignored.
func combineStatus(current, next string) string {
	switch {
	case next == "":
		return current
	case current == "":
		return next
	case strings.EqualFold(current, next):
		return current
	default:
		return statusDisputed
	}
}

// collectProofGaps returns the missing records relevant to the research topic,
// drawn from the source-coverage checklist.
func collectProofGaps(personID string, person *Person, topic string, archive *GLXFile, opts CoverageOptions) []ProofGap {
	coverage := buildCoverage(personID, person, archive, opts.CensusCountry)

	var gaps []ProofGap
	for i := range coverage.Records {
		rec := &coverage.Records[i]
		if rec.Found || !gapRelevant(topic, rec) {
			continue
		}
		priority := rec.Priority
		if topic == topicParentage && rec.Label == coverageLabelProbate && priority == severityHigh {
			// The person's own probate names their heirs, not their parents; a
			// candidate parent's probate is the record that would.
			priority = severityMedium
		}
		gaps = append(gaps, ProofGap{
			Label:       rec.Label,
			Priority:    priority,
			Description: rec.Description,

			relationshipID: rec.relationshipID,
		})
	}

	return gaps
}

// gapRelevant reports whether a missing coverage record bears on the topic.
func gapRelevant(topic string, rec *CoverageRecord) bool {
	switch topic {
	case topicParentage:
		// Records that name or place a person within their family of origin.
		return rec.Category == coverageCategoryCensus ||
			labelContains(rec.Label, "Birth record", "Death record", "Church records", coverageLabelProbate)
	case topicBirth:
		return rec.Category == coverageCategoryCensus ||
			labelContains(rec.Label, "Birth record", "Church records")
	case topicDeath:
		return labelContains(rec.Label, "Death record", coverageLabelProbate, "Church records")
	case topicMarriage:
		return labelContains(rec.Label, "Marriage record")
	case topicChildren:
		// Records that name a person's children: households, registers, estates.
		return rec.Category == coverageCategoryCensus ||
			labelContains(rec.Label, "Church records", coverageLabelProbate)
	case topicOccupation:
		return rec.Category == coverageCategoryCensus ||
			labelContains(rec.Label, "Church records", coverageLabelProbate, "Land records", "Military records")
	case topicIdentity:
		return true
	default:
		return false
	}
}

// labelContains reports whether label contains any of the given substrings.
func labelContains(label string, substrings ...string) bool {
	for _, s := range substrings {
		if strings.Contains(label, s) {
			return true
		}
	}

	return false
}

// conflictedSummary is the one-line summary returned whenever the evidence is
// judged to contain an unresolved conflict.
const conflictedSummary = "Conflicting evidence remains unresolved — resolve before drawing a conclusion."

// concludeProof determines the conclusion level and a one-line summary.
func concludeProof(topic, personID string, archive *GLXFile, relevant []proofAssertion, conflicts []ConflictGroup, gaps []ProofGap, candidates *proofCandidates, searches []ProofSearch) (string, string) {
	for i := range conflicts {
		if !conflicts[i].Resolved && (conflicts[i].Definite || conflicts[i].Verdict == "") {
			return proofConclusionConflicted, conflictedSummary
		}
	}

	caveat := proofConflictCaveat(conflicts)
	if candidates == nil {
		candidates = &proofCandidates{}
	}
	byCandidates := candidates.used && topic != topicIdentity

	answer, found := deriveProofAnswer(topic, personID, archive)
	if byCandidates {
		answer, found = candidates.answerText, candidates.found
	}
	answer, found, proxyBound := vitalProofAnswer(topic, personID, archive, relevant, answer, found)
	// Candidates that were all excluded settle the question: the assertions
	// about the rejected answers (a ruled-out marriage's date, an excluded
	// parent's baptism claim) cannot supply one through the fallbacks.
	exhausted := byCandidates && !candidates.found
	if exhausted {
		return concludeUnanswered(topic, gaps, candidates, searches)
	}
	if !byCandidates || !found {
		answer, found = resolvedConflictAnswer(topic, personID, archive, relevant, conflicts, answer, found)
	}
	if !found && topic == topicIdentity && candidates.found {
		// An accepted same-person link answers who the person was; its
		// statement is carried by the candidate notes below.
		answer, found = "", true
	}
	if !found {
		// The archive's structural data (events/relationships/name) yields no
		// answer, but relevant assertions may still carry one — e.g. legacy
		// person-subject born_on/born_at/died_on/died_at assertions (which have
		// no event to denormalize onto), or an event whose Date/PlaceID was
		// never filled in while an assertion records the value. Fall back to the
		// gathered evidence so collected assertions can still drive a conclusion
		// instead of a spurious INSUFFICIENT EVIDENCE.
		answer, found = answerFromAssertions(relevant, archive, topic, personID)
	}
	if !found {
		return concludeUnanswered(topic, gaps, candidates, searches)
	}
	answer = joinSentences(append([]string{answer}, candidates.notes...)...)

	level := supportLevel(relevant, topic, personID)
	if byCandidates && candidates.found {
		// One answer per candidate: the question stands at its weakest
		// surviving answer, never lifted by a stronger one beside it.
		level = candidates.level
	}
	if topic == topicIdentity && candidates.found {
		// A same-person link is only as strong as its weakest accepted link:
		// a proven occupation cannot lift a probable link to PROVEN.
		level = min(level, candidates.level)
	}
	if caveat != "" {
		level = min(level, supportWeak)
		answer += " " + caveat
	}
	switch level {
	case supportStrong:
		return proofConclusionProven, appendNextSteps(answer, gaps)
	case supportModerate:
		return proofConclusionProbable, appendNextSteps(answer+" Corroborate with additional independent sources.", gaps)
	case supportWeak:
		return proofConclusionPossible, appendNextSteps(answer+" Limited evidence — further research needed.", gaps)
	default:
		return proofConclusionInsufficient, unsupportedSummary(topic, answer, gaps, candidates, byCandidates, proxyBound)
	}
}

// vitalProofAnswer replaces or bounds a birth or death answer when no
// structural date stands: from a primary dated assertion, or from the rite
// (baptism, burial) that bounds the vital event.
func vitalProofAnswer(topic, personID string, archive *GLXFile, relevant []proofAssertion, answer string, found bool) (string, bool, bool) {
	if (topic != topicBirth && topic != topicDeath) || primaryStructuralVitalDate(archive, personID, topic) {
		return answer, found, false
	}
	if hasPrimaryVitalDateAssertion(relevant, topic) {
		answer, found = answerFromAssertions(relevant, archive, topic, personID)

		return answer, found, false
	}
	bound, known := proxyVitalAnswer(relevant, archive, topic)
	if !known {
		return answer, found, false
	}
	if found {
		return answer + " " + bound, true, true
	}

	return bound, true, true
}

// resolvedConflictAnswer lets a resolved conflict about the question supply
// the surviving asserted answer.
func resolvedConflictAnswer(topic, personID string, archive *GLXFile, relevant []proofAssertion, conflicts []ConflictGroup, answer string, found bool) (string, bool) {
	for i := range conflicts {
		c := &conflicts[i]
		if !c.Resolved || !proofResolutionAnswersQuestion(c, topic, personID) {
			continue
		}
		if asserted, ok := answerFromAssertions(relevant, archive, topic, personID); ok {
			return asserted, true
		}

		break
	}

	return answer, found
}

// unsupportedSummary is the summary of a question answered without support:
// the answers no assertion supports are named, and a rite's bound is kept.
func unsupportedSummary(topic, answer string, gaps []ProofGap, candidates *proofCandidates, byCandidates, proxyBound bool) string {
	switch {
	case byCandidates && candidates.found && allUnsupported(candidates.answers):
		return appendNextSteps(joinSentences(insufficientSummary(topic, nil), recordedWithoutAssertionText(candidates.answers)), gaps)
	case byCandidates && candidates.found:
		return appendNextSteps(joinSentences(answer, unsupportedAnswerText(candidates.answers)), gaps)
	case proxyBound:
		return insufficientSummary(topic, gaps) + " " + answer
	default:
		return insufficientSummary(topic, gaps)
	}
}

// concludeUnanswered concludes a question nothing answers. Excluded candidates
// are named, not counted. For children, logged searches that all came back
// empty support the negative conclusion that there were none.
func concludeUnanswered(topic string, gaps []ProofGap, candidates *proofCandidates, searches []ProofSearch) (string, string) {
	excluded := excludedAlternativeText(candidates.alternatives)
	if topic == topicChildren {
		note, negative := negativeSearchSummary(searches)
		if negative {
			return proofConclusionProbable, joinSentences("No children found.", note, excluded, "Confirm the searches cover the whole span of the marriage.")
		}
		excluded = joinSentences(excluded, note)
	}
	summary := insufficientSummary(topic, nil)
	if excluded != "" {
		summary = joinSentences(summary, excluded)
	}

	return proofConclusionInsufficient, appendNextSteps(summary, gaps)
}

// joinSentences joins the non-empty sentences with single spaces.
func joinSentences(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return strings.Join(out, " ")
}

func proofConflictCaveat(conflicts []ConflictGroup) string {
	var caveat string
	for i := range conflicts {
		c := &conflicts[i]
		if c.Resolved {
			continue
		}
		if c.Verdict == VerdictDisputed {
			return "Known dispute — review the disputed evidence."
		}
		if !c.Definite {
			caveat = "Possible conflict — check the recorded periods."
		}
	}

	return caveat
}

// supportLevel scores only non-disproven assertions that answer the question.
// Related context and synthetic comparisons cannot strengthen the conclusion.
func supportLevel(relevant []proofAssertion, topic, personID string) int {
	best := supportNone
	for i := range relevant {
		a := relevant[i].a
		if relevant[i].synthetic || strings.EqualFold(a.Status, statusDisproven) || !proofAssertionSupportsQuestion(&relevant[i], topic, personID) {
			continue
		}
		if score := assertionSupportScore(a); score > best {
			best = score
		}
	}

	return best
}

// assertionSupportScore scores how strongly a single assertion backs a claim,
// from the shared status resolver: an explicit status (`proven`, `probable`,
// `speculative`) wins over the recorded confidence. A `disputed` assertion is
// scored by its confidence; the dispute itself caps the conclusion through the
// conflict it raises.
func assertionSupportScore(a *Assertion) int {
	status := AssertionProofStatus(a)
	if status == ProofStatusDisputed {
		status = confidenceProofStatus(a.Confidence)
	}

	return proofStatusSupport(status)
}

// deriveProofAnswer extracts the current best answer to the research question
// from the archive's structural data (events and relationships).
func deriveProofAnswer(topic, personID string, archive *GLXFile) (string, bool) {
	switch topic {
	case topicParentage:
		parents := parentNames(personID, archive)
		if len(parents) == 0 {
			return "Parents not yet identified.", false
		}

		return "Parents identified: " + strings.Join(parents, ", ") + ".", true
	case topicBirth:
		return eventAnswer(personID, archive, EventTypeBirth, "Birth")
	case topicDeath:
		return eventAnswer(personID, archive, EventTypeDeath, "Death")
	case topicMarriage:
		spouses := spouseNames(personID, archive)
		if len(spouses) == 0 {
			return "No spouse identified.", false
		}

		return "Married: " + strings.Join(spouses, ", ") + ".", true
	case topicIdentity:
		if person, ok := archive.Persons[personID]; ok {
			if name := PersonDisplayName(person); name != "" {
				return "Identified as " + name + ".", true
			}
		}

		return "Name not established.", false
	default:
		return "", false
	}
}

// eventAnswer summarizes a person's birth or death from the corresponding event.
func eventAnswer(personID string, archive *GLXFile, eventType, label string) (string, bool) {
	eventID, event := proofPrimaryEvent(archive, personID, eventType)
	if event == nil || (event.Date == "" && event.PlaceID == "") {
		return label + " date and place not established.", false
	}

	var parts []string
	if structuralVitalFieldAvailable(archive, personID, eventID, conflictDateType, string(event.Date)) {
		parts = append(parts, string(event.Date))
	}
	if placeName := coverageResolvePlaceName(event.PlaceID, archive); placeName != "" && structuralVitalFieldAvailable(archive, personID, eventID, eventFieldPlace, event.PlaceID) {
		parts = append(parts, placeName)
	}

	if len(parts) == 0 {
		return label + " date and place not established.", false
	}
	return label + ": " + strings.Join(parts, ", ") + ".", true
}

// answerFromAssertions derives a fallback answer from the gathered evidence when
// deriveProofAnswer finds none in the archive's structural data. It joins the
// distinct, non-disproven asserted values relevant to the question so that
// legacy person-subject assertions, and events lacking denormalized
// Date/PlaceID, can still yield a conclusion rather than INSUFFICIENT EVIDENCE.
// The topic context is already carried by the result's question text, so the
// summary only needs the supporting values.
func answerFromAssertions(relevant []proofAssertion, archive *GLXFile, topic, personID string) (string, bool) {
	primaryVitalDate := (topic == topicBirth || topic == topicDeath) && hasPrimaryVitalDateAssertion(relevant, topic)
	seen := make(map[string]bool)
	var values []string
	for i := range relevant {
		a := relevant[i].a
		if relevant[i].synthetic || strings.EqualFold(a.Status, statusDisproven) {
			continue
		}
		if (topic == topicBirth || topic == topicDeath) && a.Participant != nil {
			continue
		}
		kind, field := vitalProofObservation(&relevant[i])
		if primaryVitalDate && field == conflictDateType && kind != topic {
			continue
		}
		if !proofAssertionAnswersQuestion(&relevant[i], topic, personID) {
			continue
		}
		value := proofAssertionValue(a, archive)
		if value != "" {
			value = qualifyProxyVitalAnswer(topic, &relevant[i], value)
		}
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	if len(values) == 0 {
		return "", false
	}
	sort.Strings(values)

	return "Supported by evidence: " + strings.Join(values, ", ") + ".", true
}

// proofAssertionSupportsQuestion reports whether an assertion can strengthen
// the conclusion. Beyond the assertions that answer the question, a parent_child
// relationship asserted on the person's own child participant evidences the
// whole edge (#1367): the record shows the person as the child, and the
// relationship names the parents. It supports the answer without supplying it,
// so answerFromAssertions still ignores it.
func proofAssertionSupportsQuestion(pa *proofAssertion, topic, personID string) bool {
	p := pa.a.Participant
	if topic == topicIdentity && pa.eventType != "" {
		// An event's date or place proves something about the event, not who
		// the person was. Event evidence scores for identity only when it
		// asserts the person's own participation.
		return p != nil && p.Person == personID
	}
	if proofAssertionAnswersQuestion(pa, topic, personID) {
		return true
	}
	if topic == topicIdentity && p != nil && p.Person == personID {
		// The person's own participation in a record evidences who they were.
		return true
	}

	return topic == topicParentage && p != nil && p.Person == personID &&
		IsParentChildRelationshipType(pa.relType) && !strings.EqualFold(pa.relType, RelationshipTypeStepParent) &&
		IsChildSideRole(p.Role) && !IsStepRole(p.Role)
}

func proofAssertionAnswersQuestion(pa *proofAssertion, topic, personID string) bool {
	if pa.a.Participant != nil {
		return participantAnswersQuestion(pa.a.Participant, topic, personID)
	}

	return proofPropertyAnswersQuestion(pa.a.Subject, pa.eventType, pa.relType, pa.a.Property, topic)
}

func proofPropertyAnswersQuestion(subject EntityRef, eventType, relationshipType, property, topic string) bool {
	kind, field := legacyVitalProperty(property)
	vitalField := property == conflictDateType || property == eventFieldPlace
	switch topic {
	case topicBirth:
		return subject.Person != "" && kind == EventTypeBirth && field != "" || isBirthEventType(eventType) && (vitalField || kind == EventTypeBirth && field == conflictDateType)
	case topicDeath:
		return subject.Person != "" && (kind == EventTypeDeath || kind == EventTypeBurial) && field != "" || isDeathEventType(eventType) && (vitalField || kind == EventTypeDeath && field == conflictDateType)
	case topicMarriage:
		// Banns, licenses, contracts and engagements prepare a marriage; only
		// the wedding's date or place answers when it took place.
		return eventType == EventTypeMarriage && vitalField ||
			(relationshipType == RelationshipTypeMarriage || relationshipType == RelationshipTypePartner) && (property == ParticipantRoleSpouse || property == RelationshipTypePartner)
	default:
		return true
	}
}

func proofResolutionAnswersQuestion(conflict *ConflictGroup, topic, personID string) bool {
	for i := range conflict.Facts {
		fact := &conflict.Facts[i]
		if fact.ParticipantPerson != "" {
			if participantAnswersQuestion(&Participant{Person: fact.ParticipantPerson, Role: fact.ParticipantRole}, topic, personID) {
				return true
			}
		} else if proofPropertyAnswersQuestion(fact.Subject, fact.EventType, fact.RelationshipType, fact.Property, topic) {
			return true
		}
	}

	return false
}

func participantAnswersQuestion(participant *Participant, topic, personID string) bool {
	if participant.Person == "" || participant.Person == personID {
		return false
	}
	switch topic {
	case topicParentage:
		return IsParentSideRole(participant.Role) && !IsStepRole(participant.Role)
	case topicMarriage:
		return participant.Role == ParticipantRoleSpouse || participant.Role == ParticipantRoleBride || participant.Role == ParticipantRoleGroom
	case topicChildren:
		return IsChildSideRole(participant.Role) && !IsStepRole(participant.Role)
	default:
		return false
	}
}

// parentNames returns surviving, non-step parent names in relationship order.
func parentNames(personID string, archive *GLXFile) []string {
	standings := NewRelationshipStandingIndex(archive)
	candidates, _ := parentageCandidates(personID, archive, ParentChildLinks(archive, standings), standings)
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.name)
	}

	return names
}

// spouseNames returns the display names of a person's spouses/partners.
func spouseNames(personID string, archive *GLXFile) []string {
	var names []string
	seen := make(map[string]bool)

	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil {
			continue
		}
		if rel.Type != RelationshipTypeMarriage && rel.Type != RelationshipTypePartner {
			continue
		}
		if !hasParticipant(personID, rel.Participants) {
			continue
		}

		for _, p := range rel.Participants {
			if p.Person != "" && p.Person != personID && !seen[p.Person] {
				seen[p.Person] = true
				names = append(names, personName(archive, p.Person))
			}
		}
	}

	return names
}

// insufficientSummary builds the summary line for an unproven question.
func insufficientSummary(topic string, gaps []ProofGap) string {
	base := "Not yet established from available evidence."
	switch topic {
	case topicParentage:
		base = "Parents not yet identified."
	case topicBirth:
		base = "Birth not yet established."
	case topicDeath:
		base = "Death not yet established."
	case topicMarriage:
		base = "No marriage established."
	case topicChildren:
		base = "No children identified."
	case topicOccupation:
		base = "Occupation not yet established."
	case topicIdentity:
		base = "Identity not yet established."
	}

	return appendNextSteps(base, gaps)
}

// appendNextSteps appends a priority recommendation drawn from high-priority
// evidence gaps, if any.
func appendNextSteps(summary string, gaps []ProofGap) string {
	var high []string
	for i := range gaps {
		if gaps[i].Priority == severityHigh && gaps[i].Searched == "" {
			high = append(high, gaps[i].Label)
		}
	}
	if len(high) == 0 {
		return summary
	}

	return summary + " Priority: locate " + strings.Join(high, "; ") + "."
}

// firstLine returns the first non-empty line of a multi-line string, trimmed.
func firstLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func proxyVitalProofGap(topic string, relevant []proofAssertion, archive *GLXFile, personID string) *ProofGap {
	if topic != topicBirth && topic != topicDeath {
		return nil
	}
	if primaryStructuralVitalDate(archive, personID, topic) {
		return nil
	}
	proxy := false
	for i := range relevant {
		evidence := &relevant[i]
		if primaryVitalDateAssertion(evidence, topic) {
			return nil
		}
		kind, _ := vitalProofObservation(evidence)
		if evidence.a.Participant == nil && !AssertionIsDisproven(evidence.a) && isProxyVitalType(topic, kind) {
			proxy = true
		}
	}
	if !proxy {
		return nil
	}

	return &ProofGap{Label: strings.ToUpper(topic[:1]) + topic[1:] + " date not established", Description: "The recorded rite bounds the vital event; it does not establish its exact date."}
}

func primaryVitalDateAssertion(evidence *proofAssertion, topic string) bool {
	if evidence.synthetic || evidence.a.Participant != nil || evidence.a.Value == "" || AssertionIsDisproven(evidence.a) {
		return false
	}
	kind, field := legacyVitalProperty(evidence.a.Property)

	return kind == topic && field == conflictDateType || evidence.eventType == topic && evidence.a.Property == conflictDateType
}

func qualifyProxyVitalAnswer(topic string, evidence *proofAssertion, value string) string {
	kind, field := vitalProofObservation(evidence)
	verb := ""
	if topic == topicBirth && (kind == EventTypeBaptism || kind == EventTypeChristening) {
		verb = "born"
	}
	if topic == topicDeath && (kind == EventTypeBurial || kind == EventTypeCremation) {
		verb = "died"
	}
	if verb == "" {
		return value
	}
	if field == eventFieldPlace {
		return kind + " place: " + value
	}
	if field != conflictDateType {
		return value
	}
	date, err := DateString(value).Parse()
	if err != nil || date.Qualifier() != 0 {
		return kind + " recorded " + value + "; exact " + topic + " date not established"
	}
	if date.IsRange() {
		if date.IsOpenEnded() && !date.IsOpenStart() {
			return kind + " recorded " + value + "; exact " + topic + " date not established"
		}
		value = date.End().String()
	}

	return verb + " on or before " + value + " (" + kind + ")"
}
