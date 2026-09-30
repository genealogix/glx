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
	topicParentage = "parentage"
	topicBirth     = "birth"
	topicDeath     = "death"
	topicMarriage  = "marriage"
	topicIdentity  = "identity"
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
}

// ProofGap is a missing record that would strengthen the proof (GPS element 1 —
// reasonably exhaustive search), sourced from the coverage checklist.
type ProofGap struct {
	Label       string `json:"label"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
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
	Query      string `json:"query,omitempty"`
	Collection string `json:"collection,omitempty"`
	Repository string `json:"repository,omitempty"`
	Source     string `json:"source,omitempty"`
	Result     string `json:"result,omitempty"`
}

// ProofResult is the full structured proof argument for one research question.
type ProofResult struct {
	Undated      []ProofEvidence `json:"undated,omitempty"`
	PersonID     string          `json:"person_id"`
	PersonName   string          `json:"person_name"`
	Question     string          `json:"question"`
	QuestionText string          `json:"question_text"`
	Evidence     []ProofEvidence `json:"evidence"`
	Gaps         []ProofGap      `json:"gaps,omitempty"`
	Conflicts    []ConflictGroup `json:"conflicts"`
	Searches     []ProofSearch   `json:"searches,omitempty"`
	Conclusion   string          `json:"conclusion"`
	Summary      string          `json:"summary,omitempty"`
}

// buildProof assembles the proof argument for a person and research topic.
func buildProof(personID string, person *Person, topic string, archive *GLXFile, opts ProofOptions) *ProofResult {
	name := PersonDisplayName(person)
	if name == "" {
		name = personID
	}

	all := collectPersonProofAssertions(archive, personID)

	var relevant []proofAssertion
	for i := range all {
		if assertionRelevant(topic, &all[i]) {
			relevant = append(relevant, all[i])
		}
	}

	evidence := make([]ProofEvidence, 0, len(relevant))
	var undated []ProofEvidence
	for i := range relevant {
		if relevant[i].synthetic {
			continue
		}
		ev := buildProofEvidence(&relevant[i], archive)
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
	conflicts := detectProofConflicts(relevant, archive, opts.Comparison)
	gaps := collectProofGaps(personID, person, topic, archive, opts.Coverage)
	searches := collectProofSearches(personID, archive)

	conclusion, summary := concludeProof(topic, personID, archive, relevant, conflicts, gaps)

	return &ProofResult{
		PersonID:     personID,
		PersonName:   name,
		Question:     topic,
		QuestionText: questionText(topic, name),
		Evidence:     evidence,
		Undated:      undated,
		Gaps:         gaps,
		Conflicts:    conflicts,
		Searches:     searches,
		Conclusion:   conclusion,
		Summary:      summary,
	}
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
		"father": true, "mother": true, ParticipantRoleParent: true, "parents": true,
		"father_name": true, "mother_name": true,
	}
)

// assertionRelevant reports whether an assertion is evidence for the topic.
func assertionRelevant(topic string, pa *proofAssertion) bool {
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
	case topicIdentity:
		return pa.subjectIsPerson() && pa.a.Property == PersonPropertyName
	default:
		return false
	}
}

func parentageAssertionRelevant(pa *proofAssertion) bool {
	if researchIsParentChildType(pa.relType) && pa.personRole == ParticipantRoleChild {
		return true
	}
	if pa.eventType == EventTypeBirth {
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
func buildProofEvidence(pa *proofAssertion, archive *GLXFile) ProofEvidence {
	a := pa.a

	evidence := ProofEvidence{
		AssertionID: pa.id,
		Subject:     describeProofSubject(pa, archive),
		Property:    a.Property,
		Value:       proofAssertionValue(a, archive),
		Date:        string(a.Date),
		Confidence:  a.Confidence,
		Status:      a.Status,
		Notes:       firstLine(a.Notes.String()),
		Support:     buildProofSupport(a, archive),
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
			raw[i] = AssertionFact(a)
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
		gaps = append(gaps, ProofGap{
			Label:       rec.Label,
			Priority:    rec.Priority,
			Description: rec.Description,
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
			labelContains(rec.Label, "Birth record", "Death record", "Church records", "Probate/will")
	case topicBirth:
		return rec.Category == coverageCategoryCensus ||
			labelContains(rec.Label, "Birth record", "Church records")
	case topicDeath:
		return labelContains(rec.Label, "Death record", "Probate/will", "Church records")
	case topicMarriage:
		return labelContains(rec.Label, "Marriage record")
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

// collectProofSearches gathers searches from research logs whose subject is the
// person, documenting the reasonably exhaustive search (including negative
// evidence from searches that found nothing).
func collectProofSearches(personID string, archive *GLXFile) []ProofSearch {
	var searches []ProofSearch

	for _, logID := range sortedKeys(archive.ResearchLogs) {
		log := archive.ResearchLogs[logID]
		if log == nil || log.Subject == nil || log.Subject.Person != personID {
			continue
		}
		for i := range log.Searches {
			s := &log.Searches[i]
			searches = append(searches, ProofSearch{
				Query:      s.Query,
				Collection: s.Collection,
				Repository: s.RepositoryID,
				Source:     s.SourceID,
				Result:     s.Result,
			})
		}
	}

	return searches
}

// conflictedSummary is the one-line summary returned whenever the evidence is
// judged to contain an unresolved conflict.
const conflictedSummary = "Conflicting evidence remains unresolved — resolve before drawing a conclusion."

// concludeProof determines the conclusion level and a one-line summary.
func concludeProof(topic, personID string, archive *GLXFile, relevant []proofAssertion, conflicts []ConflictGroup, gaps []ProofGap) (string, string) {
	for i := range conflicts {
		if !conflicts[i].Resolved && (conflicts[i].Definite || conflicts[i].Verdict == "") {
			return proofConclusionConflicted, conflictedSummary
		}
	}

	caveat := proofConflictCaveat(conflicts)

	answer, found := deriveProofAnswer(topic, personID, archive)
	for i := range conflicts {
		c := &conflicts[i]
		if c.Resolved {
			if asserted, ok := answerFromAssertions(relevant, archive, topic, personID); ok {
				answer, found = asserted, true
			}

			break
		}
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
		return proofConclusionInsufficient, insufficientSummary(topic, gaps)
	}

	level := supportLevel(relevant)
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
		return proofConclusionInsufficient, insufficientSummary(topic, gaps)
	}
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

// supportLevel returns the strongest support score among non-disproven relevant
// assertions, or supportNone when nothing backs the answer.
func supportLevel(relevant []proofAssertion) int {
	best := supportNone
	for i := range relevant {
		a := relevant[i].a
		if relevant[i].synthetic || strings.EqualFold(a.Status, statusDisproven) {
			continue
		}
		if score := assertionSupportScore(a); score > best {
			best = score
		}
	}

	return best
}

// assertionSupportScore scores how strongly a single assertion backs a claim,
// reusing the shared confidenceRank (lower rank = higher confidence). A `proven`
// status is treated as strong regardless of the recorded confidence.
func assertionSupportScore(a *Assertion) int {
	if strings.EqualFold(a.Status, statusProven) {
		return supportStrong
	}
	switch researchConfidenceRank(a.Confidence) {
	case 0: // high
		return supportStrong
	case 1, 2: // medium-high, medium
		return supportModerate
	default: // low or unspecified
		return supportWeak
	}
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
	_, event := FindPersonEvent(archive, personID, eventType)
	if event == nil || (event.Date == "" && event.PlaceID == "") {
		return label + " date and place not established.", false
	}

	var parts []string
	if event.Date != "" {
		parts = append(parts, string(event.Date))
	}
	if placeName := coverageResolvePlaceName(event.PlaceID, archive); placeName != "" {
		parts = append(parts, placeName)
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
	seen := make(map[string]bool)
	var values []string
	for i := range relevant {
		a := relevant[i].a
		if relevant[i].synthetic || strings.EqualFold(a.Status, statusDisproven) {
			continue
		}
		if a.Participant != nil && !participantAnswersQuestion(a.Participant, topic, personID) {
			continue
		}
		value := proofAssertionValue(a, archive)
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

func participantAnswersQuestion(participant *Participant, topic, personID string) bool {
	if participant.Person == "" || participant.Person == personID {
		return false
	}
	switch topic {
	case topicParentage:
		return participant.Role == ParticipantRoleParent
	case topicMarriage:
		return participant.Role == ParticipantRoleSpouse || participant.Role == ParticipantRoleBride || participant.Role == ParticipantRoleGroom
	default:
		return false
	}
}

// parentNames returns the display names of a person's parents, drawn from
// parent-child relationships where the person is the child.
func parentNames(personID string, archive *GLXFile) []string {
	var names []string
	seen := make(map[string]bool)

	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil || !researchIsParentChildType(rel.Type) {
			continue
		}
		if !hasParticipantRole(personID, ParticipantRoleChild, rel.Participants) {
			continue
		}

		for _, p := range rel.Participants {
			if p.Role == ParticipantRoleParent && p.Person != "" && !seen[p.Person] {
				seen[p.Person] = true
				names = append(names, personName(archive, p.Person))
			}
		}
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

// hasParticipantRole reports whether the person participates with the given role.
func hasParticipantRole(personID, role string, participants []Participant) bool {
	for _, p := range participants {
		if p.Person == personID && p.Role == role {
			return true
		}
	}

	return false
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
		if gaps[i].Priority == severityHigh {
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
