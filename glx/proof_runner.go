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
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
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

// Output format keys accepted by the proof command.
const (
	proofFormatText     = "text"
	proofFormatJSON     = "json"
	proofFormatMarkdown = "markdown"
	proofFormatMD       = "md"
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

// Static base errors so dynamic context can be wrapped (satisfies err113).
var (
	errUnknownQuestion = errors.New("unknown research question")
	errUnknownFormat   = errors.New("unknown format")
)

// proofSupport is a single citation or source backing an assertion, resolved to
// human-readable form (GPS element 2 — complete citations).
type proofSupport struct {
	Ref         string `json:"ref"`                    // citation or source ID
	Kind        string `json:"kind"`                   // "citation" or "source"
	SourceID    string `json:"source_id,omitempty"`    // resolved source ID (citations only)
	SourceTitle string `json:"source_title,omitempty"` // resolved source title
	Locator     string `json:"locator,omitempty"`      // citation locator (page, entry, certificate no.)
}

// proofEvidence is one assertion gathered as evidence for the research question
// (GPS element 3 — analysis of evidence).
type proofEvidence struct {
	AssertionID string         `json:"assertion_id"`
	Subject     string         `json:"subject,omitempty"` // subject context (e.g. "birth event (1955)")
	Property    string         `json:"property,omitempty"`
	Value       string         `json:"value,omitempty"`
	Date        string         `json:"date,omitempty"`
	Confidence  string         `json:"confidence,omitempty"`
	Status      string         `json:"status,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	Support     []proofSupport `json:"support,omitempty"`
}

// proofGap is a missing record that would strengthen the proof (GPS element 1 —
// reasonably exhaustive search), sourced from the coverage checklist.
type proofGap struct {
	Label       string `json:"label"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
}

// proofConflictValue is one competing value within a conflict.
type proofConflictValue struct {
	Value      string `json:"value"`
	Confidence string `json:"confidence,omitempty"`
	Status     string `json:"status,omitempty"`
}

// proofConflict records two or more assertions that disagree on the same
// subject/property (GPS element 4 — resolution of conflicting evidence).
type proofConflict struct {
	Subject    string               `json:"subject,omitempty"`
	Property   string               `json:"property"`
	Values     []proofConflictValue `json:"values"`
	Resolved   bool                 `json:"resolved"`
	Resolution string               `json:"resolution,omitempty"`
}

// proofSearch is one logged search, including searches that found nothing
// (negative evidence supporting GPS element 1).
type proofSearch struct {
	Query      string `json:"query,omitempty"`
	Collection string `json:"collection,omitempty"`
	Repository string `json:"repository,omitempty"`
	Source     string `json:"source,omitempty"`
	Result     string `json:"result,omitempty"`
}

// proofResult is the full structured proof argument for one research question.
type proofResult struct {
	PersonID     string          `json:"person_id"`
	PersonName   string          `json:"person_name"`
	Question     string          `json:"question"`
	QuestionText string          `json:"question_text"`
	Evidence     []proofEvidence `json:"evidence"`
	Gaps         []proofGap      `json:"gaps,omitempty"`
	Conflicts    []proofConflict `json:"conflicts"`
	Searches     []proofSearch   `json:"searches,omitempty"`
	Conclusion   string          `json:"conclusion"`
	Summary      string          `json:"summary,omitempty"`
}

// showProof loads an archive and prints a structured proof summary for a person
// and research question, following the BCG Genealogical Proof Standard format.
// Human-readable text/markdown is written to io.Out (silenced by --quiet); JSON,
// which is machine-consumable, is written to io.MachineOut so it survives --quiet
// for shell capture.
func showProof(io *IOStreams, archivePath, personQuery, question, format string) error {
	topic, ok := canonicalQuestion(question)
	if !ok {
		return fmt.Errorf("%w %q (valid: %s)", errUnknownQuestion, question, strings.Join(proofQuestionKeys(), ", "))
	}

	archive, err := loadArchiveForCoverage(archivePath)
	if err != nil {
		return err
	}

	personID, person, err := findPersonForCoverage(archive, personQuery)
	if err != nil {
		return err
	}

	result := buildProof(personID, person, topic, archive)

	switch format {
	case proofFormatJSON:
		return printProofJSON(io, result)
	case proofFormatMarkdown, proofFormatMD:
		printProofMarkdown(io, result)
	case "", proofFormatText:
		printProofText(io, result)
	default:
		return fmt.Errorf("%w %q (valid: text, json, markdown)", errUnknownFormat, format)
	}

	return nil
}

// buildProof assembles the proof argument for a person and research topic.
func buildProof(personID string, person *glxlib.Person, topic string, archive *glxlib.GLXFile) *proofResult {
	name := glxlib.PersonDisplayName(person)
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

	evidence := make([]proofEvidence, 0, len(relevant))
	for i := range relevant {
		evidence = append(evidence, buildProofEvidence(&relevant[i], archive))
	}

	conflicts := detectProofConflicts(relevant, archive)
	gaps := collectProofGaps(personID, person, topic, archive)
	searches := collectProofSearches(personID, archive)

	conclusion, summary := concludeProof(topic, personID, archive, relevant, conflicts, gaps)

	return &proofResult{
		PersonID:     personID,
		PersonName:   name,
		Question:     topic,
		QuestionText: questionText(topic, name),
		Evidence:     evidence,
		Gaps:         gaps,
		Conflicts:    conflicts,
		Searches:     searches,
		Conclusion:   conclusion,
		Summary:      summary,
	}
}

// ============================================================================
// Research question topics
// ============================================================================

// proofTopic defines a supported research question and the input aliases that
// map to it.
type proofTopic struct {
	key     string
	aliases []string
}

// proofTopics lists supported research questions in display order. Question
// strings are matched case-insensitively against the key and aliases.
var proofTopics = []proofTopic{
	{key: topicParentage, aliases: []string{"parents", glxlib.ParticipantRoleParent, "father", "mother", "ancestry"}},
	{key: topicBirth, aliases: []string{"born", "birthdate"}},
	{key: topicDeath, aliases: []string{"died", "deathdate", glxlib.EventTypeBurial}},
	{key: topicMarriage, aliases: []string{glxlib.ParticipantRoleSpouse, "marriages"}},
	{key: topicIdentity, aliases: []string{glxlib.PersonPropertyName, "who"}},
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

// ============================================================================
// Assertion collection and relevance
// ============================================================================

// proofAssertion couples an assertion with its resolved subject context so
// relevance and conflict logic can reason about it without re-resolving.
type proofAssertion struct {
	id         string
	a          *glxlib.Assertion
	subjectID  string // person, event, or relationship ID the assertion is about
	eventType  string // set when the subject is an event the person participates in
	relType    string // set when the subject is a relationship the person participates in
	personRole string // the person's role in the subject relationship
}

// subjectIsPerson reports whether the assertion's subject is the target person
// directly (rather than one of their events or relationships).
func (pa *proofAssertion) subjectIsPerson() bool {
	return pa.eventType == "" && pa.relType == ""
}

// collectPersonProofAssertions gathers every assertion whose subject is the
// person, or an event/relationship the person participates in.
func collectPersonProofAssertions(archive *glxlib.GLXFile, personID string) []proofAssertion {
	eventIDs := personEventIDSet(archive, personID)
	relRoles := personRelationshipRoles(archive, personID)

	var out []proofAssertion
	for _, id := range sortedAssertionIDs(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil {
			continue
		}

		switch {
		case a.Subject.Person == personID:
			out = append(out, proofAssertion{id: id, a: a, subjectID: personID})
		case a.Subject.Event != "" && eventIDs[a.Subject.Event]:
			eventType := ""
			if ev := archive.Events[a.Subject.Event]; ev != nil {
				eventType = ev.Type
			}
			out = append(out, proofAssertion{id: id, a: a, subjectID: a.Subject.Event, eventType: eventType})
		case a.Subject.Relationship != "":
			role, ok := relRoles[a.Subject.Relationship]
			if !ok {
				continue
			}
			relType := ""
			if rel := archive.Relationships[a.Subject.Relationship]; rel != nil {
				relType = rel.Type
			}
			out = append(out, proofAssertion{id: id, a: a, subjectID: a.Subject.Relationship, relType: relType, personRole: role})
		}
	}

	return out
}

// personEventIDSet returns the set of event IDs the person participates in.
func personEventIDSet(archive *glxlib.GLXFile, personID string) map[string]bool {
	set := make(map[string]bool)
	for id, ev := range archive.Events {
		if ev == nil {
			continue
		}
		for _, p := range ev.Participants {
			if p.Person == personID {
				set[id] = true

				break
			}
		}
	}

	return set
}

// personRelationshipRoles maps each relationship the person participates in to
// the person's role within it.
func personRelationshipRoles(archive *glxlib.GLXFile, personID string) map[string]string {
	roles := make(map[string]string)
	for id, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == personID {
				roles[id] = p.Role

				break
			}
		}
	}

	return roles
}

// Property-name sets for matching legacy direct-on-person assertions to a topic.
var (
	parentPropertyNames = map[string]bool{
		"father": true, "mother": true, glxlib.ParticipantRoleParent: true, "parents": true,
		"father_name": true, "mother_name": true,
	}
	birthPropertyNames = map[string]bool{
		"birth_date": true, "birth_place": true,
		glxlib.DeprecatedPropertyBornOn: true, glxlib.DeprecatedPropertyBornAt: true,
	}
	deathPropertyNames = map[string]bool{
		"death_date": true, "death_place": true, "burial_date": true, "burial_place": true,
		glxlib.DeprecatedPropertyDiedOn: true, glxlib.DeprecatedPropertyDiedAt: true,
		glxlib.DeprecatedPropertyBuriedOn: true, glxlib.DeprecatedPropertyBuriedAt: true,
	}
)

// assertionRelevant reports whether an assertion is evidence for the topic.
func assertionRelevant(topic string, pa *proofAssertion) bool {
	switch topic {
	case topicParentage:
		if pa.relType != "" && isParentChildType(pa.relType) && pa.personRole == glxlib.ParticipantRoleChild {
			return true
		}
		if pa.eventType == glxlib.EventTypeBirth && parentPropertyNames[pa.a.Property] {
			return true
		}

		return pa.subjectIsPerson() && parentPropertyNames[pa.a.Property]
	case topicBirth:
		if isBirthEventType(pa.eventType) {
			return true
		}

		return pa.subjectIsPerson() && birthPropertyNames[pa.a.Property]
	case topicDeath:
		if isDeathEventType(pa.eventType) {
			return true
		}

		return pa.subjectIsPerson() && deathPropertyNames[pa.a.Property]
	case topicMarriage:
		if pa.relType == glxlib.RelationshipTypeMarriage || pa.relType == glxlib.RelationshipTypePartner {
			return true
		}

		return isMarriageEventType(pa.eventType)
	case topicIdentity:
		return pa.subjectIsPerson() && pa.a.Property == glxlib.PersonPropertyName
	default:
		return false
	}
}

func isBirthEventType(t string) bool {
	return t == glxlib.EventTypeBirth || t == glxlib.EventTypeBaptism || t == glxlib.EventTypeChristening
}

func isDeathEventType(t string) bool {
	return t == glxlib.EventTypeDeath || t == glxlib.EventTypeBurial || t == glxlib.EventTypeCremation
}

func isMarriageEventType(t string) bool {
	switch t {
	case glxlib.EventTypeMarriage, glxlib.EventTypeMarriageLicense,
		glxlib.EventTypeMarriageBanns, glxlib.EventTypeMarriageContract,
		glxlib.EventTypeEngagement:
		return true
	default:
		return false
	}
}

// ============================================================================
// Evidence assembly
// ============================================================================

// buildProofEvidence converts a relevant assertion into a display-ready
// evidence record with resolved citations and values.
func buildProofEvidence(pa *proofAssertion, archive *glxlib.GLXFile) proofEvidence {
	a := pa.a

	return proofEvidence{
		AssertionID: pa.id,
		Subject:     describeProofSubject(pa, archive),
		Property:    a.Property,
		Value:       resolveProofValue(a.Value, archive),
		Date:        string(a.Date),
		Confidence:  a.Confidence,
		Status:      a.Status,
		Notes:       firstLine(a.Notes.String()),
		Support:     buildProofSupport(a, archive),
	}
}

// describeProofSubject returns a short label for an assertion's subject when it
// is an event or relationship; the person subject needs no label.
func describeProofSubject(pa *proofAssertion, archive *glxlib.GLXFile) string {
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

// buildProofSupport resolves an assertion's citations and sources into complete,
// human-readable references.
func buildProofSupport(a *glxlib.Assertion, archive *glxlib.GLXFile) []proofSupport {
	support := make([]proofSupport, 0, len(a.Citations)+len(a.Sources))

	for _, citID := range a.Citations {
		s := proofSupport{Ref: citID, Kind: "citation"}
		if cit, ok := archive.Citations[citID]; ok && cit != nil {
			s.SourceID = cit.SourceID
			s.SourceTitle = resolveSourceTitle(cit.SourceID, archive)
			s.Locator = citationProperty(cit, "locator")
		}
		support = append(support, s)
	}

	for _, srcID := range a.Sources {
		support = append(support, proofSupport{
			Ref:         srcID,
			Kind:        "source",
			SourceID:    srcID,
			SourceTitle: resolveSourceTitle(srcID, archive),
		})
	}

	return support
}

// resolveProofValue renders an assertion value, resolving place IDs to names.
func resolveProofValue(value string, archive *glxlib.GLXFile) string {
	if value == "" {
		return ""
	}
	if place, ok := archive.Places[value]; ok && place != nil && place.Name != "" {
		return place.Name
	}

	return value
}

// ============================================================================
// Conflict detection
// ============================================================================

// detectProofConflicts finds relevant assertions that disagree on the same
// subject/property and reports whether the disagreement has been resolved via
// the assertion `status` field (proven over disproven).
func detectProofConflicts(relevant []proofAssertion, archive *glxlib.GLXFile) []proofConflict {
	type conflictKey struct {
		subject  string
		property string
	}
	type conflictGroup struct {
		label      string // human-readable subject label, "" for the person subject
		assertions []*glxlib.Assertion
	}

	groups := make(map[conflictKey]*conflictGroup)
	var order []conflictKey
	for i := range relevant {
		pa := &relevant[i]
		if pa.a.Property == "" || pa.a.Value == "" {
			continue
		}
		key := conflictKey{subject: pa.subjectID, property: pa.a.Property}
		g, seen := groups[key]
		if !seen {
			g = &conflictGroup{label: describeProofSubject(pa, archive)}
			groups[key] = g
			order = append(order, key)
		}
		g.assertions = append(g.assertions, pa.a)
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].subject != order[j].subject {
			return order[i].subject < order[j].subject
		}

		return order[i].property < order[j].property
	})

	var conflicts []proofConflict
	for _, key := range order {
		g := groups[key]
		values := distinctProofValues(g.assertions, archive)
		if len(values) < 2 {
			continue
		}

		// Disambiguate which subject the conflict belongs to. Fall back to the
		// raw subject ID when there is no descriptive label (e.g. the person
		// subject) so machine consumers can always tell groups apart.
		subject := g.label
		if subject == "" && key.subject != "" {
			subject = key.subject
		}

		resolved, resolution := resolveConflict(values)
		conflicts = append(conflicts, proofConflict{
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
// statuses of every assertion for that value (see combineStatus).
func distinctProofValues(assertions []*glxlib.Assertion, archive *glxlib.GLXFile) []proofConflictValue {
	type agg struct {
		confidence string
		status     string
	}
	seen := make(map[string]*agg)
	var order []string

	for _, a := range assertions {
		val := resolveProofValue(a.Value, archive)
		cur, exists := seen[val]
		if !exists {
			seen[val] = &agg{confidence: a.Confidence, status: a.Status}
			order = append(order, val)

			continue
		}
		if confidenceRank(a.Confidence) < confidenceRank(cur.confidence) {
			cur.confidence = a.Confidence
		}
		cur.status = combineStatus(cur.status, a.Status)
	}

	sort.Strings(order)
	out := make([]proofConflictValue, len(order))
	for i, val := range order {
		out[i] = proofConflictValue{Value: val, Confidence: seen[val].confidence, Status: seen[val].status}
	}

	return out
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

// resolveConflict decides whether a set of competing values has been resolved.
// A conflict is resolved when exactly one value remains after removing values
// explicitly marked `disproven`; that surviving value is the resolution.
func resolveConflict(values []proofConflictValue) (bool, string) {
	var surviving []proofConflictValue
	disputed := false
	for i := range values {
		switch strings.ToLower(values[i].Status) {
		case statusDisproven:
			// ruled out
		case statusDisputed:
			disputed = true
			surviving = append(surviving, values[i])
		default:
			surviving = append(surviving, values[i])
		}
	}

	if !disputed && len(surviving) == 1 {
		return true, surviving[0].Value
	}

	return false, ""
}

// ============================================================================
// Evidence gaps (from the coverage checklist)
// ============================================================================

// collectProofGaps returns the missing records relevant to the research topic,
// drawn from the source-coverage checklist.
func collectProofGaps(personID string, person *glxlib.Person, topic string, archive *glxlib.GLXFile) []proofGap {
	coverage := buildCoverage(personID, person, archive)

	var gaps []proofGap
	for i := range coverage.Records {
		rec := &coverage.Records[i]
		if rec.Found || !gapRelevant(topic, rec) {
			continue
		}
		gaps = append(gaps, proofGap{
			Label:       rec.Label,
			Priority:    rec.Priority,
			Description: rec.Description,
		})
	}

	return gaps
}

// gapRelevant reports whether a missing coverage record bears on the topic.
// The coverage "census" category shares the literal value of EventTypeCensus.
func gapRelevant(topic string, rec *coverageRecord) bool {
	switch topic {
	case topicParentage:
		// Records that name or place a person within their family of origin.
		return rec.Category == glxlib.EventTypeCensus ||
			labelContains(rec.Label, "Birth record", "Death record", "Church records", "Probate/will")
	case topicBirth:
		return rec.Category == glxlib.EventTypeCensus ||
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

// ============================================================================
// Logged searches (reasonably exhaustive search)
// ============================================================================

// collectProofSearches gathers searches from research logs whose subject is the
// person, documenting the reasonably exhaustive search (including negative
// evidence from searches that found nothing).
func collectProofSearches(personID string, archive *glxlib.GLXFile) []proofSearch {
	var searches []proofSearch

	for _, logID := range sortedKeys(archive.ResearchLogs) {
		log := archive.ResearchLogs[logID]
		if log == nil || log.Subject == nil || log.Subject.Person != personID {
			continue
		}
		for i := range log.Searches {
			s := &log.Searches[i]
			searches = append(searches, proofSearch{
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

// ============================================================================
// Conclusion
// ============================================================================

// concludeProof determines the conclusion level and a one-line summary.
func concludeProof(topic, personID string, archive *glxlib.GLXFile, relevant []proofAssertion, conflicts []proofConflict, gaps []proofGap) (string, string) {
	for i := range conflicts {
		if !conflicts[i].Resolved {
			return proofConclusionConflicted,
				"Conflicting evidence remains unresolved — resolve before drawing a conclusion."
		}
	}

	answer, found := deriveProofAnswer(topic, personID, archive)
	if !found {
		return proofConclusionInsufficient, insufficientSummary(topic, gaps)
	}

	switch supportLevel(relevant) {
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

// supportLevel returns the strongest support score among non-disproven relevant
// assertions, or supportNone when nothing backs the answer.
func supportLevel(relevant []proofAssertion) int {
	best := supportNone
	for i := range relevant {
		a := relevant[i].a
		if strings.EqualFold(a.Status, statusDisproven) {
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
func assertionSupportScore(a *glxlib.Assertion) int {
	if strings.EqualFold(a.Status, statusProven) {
		return supportStrong
	}
	switch confidenceRank(a.Confidence) {
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
func deriveProofAnswer(topic, personID string, archive *glxlib.GLXFile) (string, bool) {
	switch topic {
	case topicParentage:
		parents := parentNames(personID, archive)
		if len(parents) == 0 {
			return "Parents not yet identified.", false
		}

		return "Parents identified: " + strings.Join(parents, ", ") + ".", true
	case topicBirth:
		return eventAnswer(personID, archive, glxlib.EventTypeBirth, "Birth")
	case topicDeath:
		return eventAnswer(personID, archive, glxlib.EventTypeDeath, "Death")
	case topicMarriage:
		spouses := spouseNames(personID, archive)
		if len(spouses) == 0 {
			return "No spouse identified.", false
		}

		return "Married: " + strings.Join(spouses, ", ") + ".", true
	case topicIdentity:
		if person, ok := archive.Persons[personID]; ok {
			if name := glxlib.PersonDisplayName(person); name != "" {
				return "Identified as " + name + ".", true
			}
		}

		return "Name not established.", false
	default:
		return "", false
	}
}

// eventAnswer summarizes a person's birth or death from the corresponding event.
func eventAnswer(personID string, archive *glxlib.GLXFile, eventType, label string) (string, bool) {
	_, event := glxlib.FindPersonEvent(archive, personID, eventType)
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

// parentNames returns the display names of a person's parents, drawn from
// parent-child relationships where the person is the child.
func parentNames(personID string, archive *glxlib.GLXFile) []string {
	var names []string
	seen := make(map[string]bool)

	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}
		if !hasParticipantRole(personID, glxlib.ParticipantRoleChild, rel.Participants) {
			continue
		}

		for _, p := range rel.Participants {
			if p.Role == glxlib.ParticipantRoleParent && p.Person != "" && !seen[p.Person] {
				seen[p.Person] = true
				names = append(names, personName(archive, p.Person))
			}
		}
	}

	return names
}

// spouseNames returns the display names of a person's spouses/partners.
func spouseNames(personID string, archive *glxlib.GLXFile) []string {
	var names []string
	seen := make(map[string]bool)

	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil {
			continue
		}
		if rel.Type != glxlib.RelationshipTypeMarriage && rel.Type != glxlib.RelationshipTypePartner {
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
func hasParticipantRole(personID, role string, participants []glxlib.Participant) bool {
	for _, p := range participants {
		if p.Person == personID && p.Role == role {
			return true
		}
	}

	return false
}

// insufficientSummary builds the summary line for an unproven question.
func insufficientSummary(topic string, gaps []proofGap) string {
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
func appendNextSteps(summary string, gaps []proofGap) string {
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

// ============================================================================
// Output rendering
// ============================================================================

// printProofJSON outputs the proof result as JSON. JSON is machine-consumable, so
// it goes to MachineOut — which, unlike Out, survives --quiet for shell capture.
func printProofJSON(io *IOStreams, result *proofResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(io.MachineOut, string(data)) //nolint:errcheck // CLI output

	return nil
}

// printProofText renders the proof result as human-readable terminal output.
func printProofText(io *IOStreams, result *proofResult) {
	io.Printf("Proof Summary: %s (%s)\n\n", result.QuestionText, result.PersonID)
	io.Printf("  Question: %s\n", result.QuestionText)

	io.Printf("\n  Evidence Collected:\n")
	if len(result.Evidence) == 0 {
		io.Println("    (none)")
	}
	for i := range result.Evidence {
		printProofEvidenceText(io, i+1, &result.Evidence[i])
	}

	io.Printf("\n  Evidence Gaps:\n")
	if len(result.Gaps) == 0 {
		io.Println("    (none identified)")
	}
	for i := range result.Gaps {
		io.Println("    [ ] " + gapLine(&result.Gaps[i], " -- ", "HIGH PRIORITY"))
	}

	if len(result.Searches) > 0 {
		io.Printf("\n  Reasonably Exhaustive Search:\n")
		for i := range result.Searches {
			io.Println("    " + formatSearchLine(&result.Searches[i]))
		}
	}

	io.Printf("\n  Conflicts: ")
	if len(result.Conflicts) == 0 {
		io.Println("None identified")
	} else {
		io.Println("")
		for i := range result.Conflicts {
			printProofConflictText(io, &result.Conflicts[i])
		}
	}

	io.Printf("\n  Conclusion: %s\n", result.Conclusion)
	if result.Summary != "" {
		io.Printf("    %s\n", result.Summary)
	}
}

// printProofEvidenceText prints one numbered evidence entry.
func printProofEvidenceText(io *IOStreams, n int, ev *proofEvidence) {
	io.Printf("    %d. %s\n", n, evidenceHeader(ev))

	if detail := evidenceDetail(ev); detail != "" {
		io.Printf("       -> %s\n", detail)
	}
	if ev.Notes != "" {
		io.Printf("       -> %s\n", ev.Notes)
	}
}

// evidenceHeader builds the first line of an evidence entry: the source title
// (or assertion ID) plus the backing reference.
func evidenceHeader(ev *proofEvidence) string {
	if len(ev.Support) == 0 {
		return "Uncited assertion (" + ev.AssertionID + ")"
	}

	s := ev.Support[0]
	title := s.SourceTitle
	if title == "" {
		title = s.SourceID
	}
	if title == "" {
		title = s.Ref
	}
	header := title + " (" + s.Ref + ")"
	if len(ev.Support) > 1 {
		header += fmt.Sprintf(" +%d more", len(ev.Support)-1)
	}

	return header
}

// evidenceDetail builds the claim line of an evidence entry.
func evidenceDetail(ev *proofEvidence) string {
	var b strings.Builder
	if ev.Subject != "" {
		b.WriteString(ev.Subject)
		b.WriteString(": ")
	}

	if ev.Property == "" && ev.Value == "" {
		// Existential assertion: only the subject's existence is claimed
		// (optionally scoped by date).
		b.WriteString("existence asserted")
		if ev.Date != "" {
			b.WriteString(" (" + ev.Date + ")")
		}
	} else {
		if ev.Property != "" {
			b.WriteString(ev.Property)
			if ev.Value != "" {
				b.WriteString(" = ")
			}
		}
		b.WriteString(ev.Value)
	}

	if tags := claimTags(ev.Confidence, ev.Status); tags != "" {
		b.WriteString(" [")
		b.WriteString(tags)
		b.WriteString("]")
	}

	return strings.TrimSpace(b.String())
}

// claimTags renders the confidence/status annotation for a claim.
func claimTags(confidence, status string) string {
	var tags []string
	if confidence != "" {
		tags = append(tags, "confidence: "+confidence)
	}
	if status != "" {
		tags = append(tags, "status: "+status)
	}

	return strings.Join(tags, ", ")
}

// printProofConflictText prints one conflict entry.
func printProofConflictText(io *IOStreams, c *proofConflict) {
	io.Printf("    ! %s: %s\n", conflictLabel(c), conflictValuesString(c))
	if c.Resolved {
		io.Printf("      RESOLVED: %s\n", c.Resolution)
	} else {
		io.Printf("      UNRESOLVED — resolution needed\n")
	}
}

// conflictLabel renders the conflict's subject (when known) and property.
func conflictLabel(c *proofConflict) string {
	if c.Subject != "" {
		return c.Subject + " " + c.Property
	}

	return c.Property
}

// conflictValuesString renders the competing values of a conflict.
func conflictValuesString(c *proofConflict) string {
	values := make([]string, 0, len(c.Values))
	for i := range c.Values {
		v := &c.Values[i]
		entry := v.Value
		var tags []string
		if v.Confidence != "" {
			tags = append(tags, v.Confidence)
		}
		if v.Status != "" {
			tags = append(tags, v.Status)
		}
		if len(tags) > 0 {
			entry += " [" + strings.Join(tags, ", ") + "]"
		}
		values = append(values, entry)
	}

	return strings.Join(values, " vs ")
}

// gapLine renders a single evidence-gap line with a separator and priority label.
func gapLine(g *proofGap, sep, highLabel string) string {
	line := g.Label
	if g.Priority == severityHigh {
		line += sep + highLabel
	}
	if g.Description != "" {
		line += sep + g.Description
	}

	return line
}

// formatSearchLine renders a logged search as a single line.
func formatSearchLine(s *proofSearch) string {
	target := s.Source
	if target == "" {
		target = s.Repository
	}
	if s.Collection != "" {
		if target != "" {
			target += " / "
		}
		target += s.Collection
	}

	var b strings.Builder
	if s.Query != "" {
		b.WriteString(s.Query)
	} else {
		b.WriteString("(search)")
	}
	if target != "" {
		b.WriteString(" @ ")
		b.WriteString(target)
	}
	if s.Result != "" {
		b.WriteString(" -> ")
		b.WriteString(s.Result)
	}

	return b.String()
}

// printProofMarkdown renders the proof result as Markdown.
func printProofMarkdown(io *IOStreams, result *proofResult) {
	io.Printf("# Proof Summary: %s\n\n", result.QuestionText)
	io.Printf("- **Person:** %s (`%s`)\n", result.PersonName, result.PersonID)
	io.Printf("- **Question:** %s\n", result.QuestionText)
	io.Printf("- **Conclusion:** %s\n", result.Conclusion)
	if result.Summary != "" {
		io.Printf("- **Summary:** %s\n", result.Summary)
	}

	printMarkdownEvidence(io, result.Evidence)
	printMarkdownGaps(io, result.Gaps)
	printMarkdownSearches(io, result.Searches)
	printMarkdownConflicts(io, result.Conflicts)
}

// printMarkdownEvidence renders the evidence section as Markdown.
func printMarkdownEvidence(io *IOStreams, evidence []proofEvidence) {
	io.Printf("\n## Evidence Collected\n\n")
	if len(evidence) == 0 {
		io.Println("_No evidence collected._")
	}
	for i := range evidence {
		ev := &evidence[i]
		io.Printf("%d. **%s**\n", i+1, evidenceHeader(ev))
		if detail := evidenceDetail(ev); detail != "" {
			io.Printf("   - %s\n", detail)
		}
		if ev.Notes != "" {
			io.Printf("   - %s\n", ev.Notes)
		}
	}
}

// printMarkdownGaps renders the evidence-gaps section as Markdown.
func printMarkdownGaps(io *IOStreams, gaps []proofGap) {
	io.Printf("\n## Evidence Gaps\n\n")
	if len(gaps) == 0 {
		io.Println("_None identified._")
	}
	for i := range gaps {
		io.Println("- [ ] " + gapLine(&gaps[i], " — ", "**HIGH PRIORITY**"))
	}
}

// printMarkdownSearches renders the logged-searches section as Markdown.
func printMarkdownSearches(io *IOStreams, searches []proofSearch) {
	if len(searches) == 0 {
		return
	}
	io.Printf("\n## Reasonably Exhaustive Search\n\n")
	for i := range searches {
		io.Println("- " + formatSearchLine(&searches[i]))
	}
}

// printMarkdownConflicts renders the conflicts section as Markdown.
func printMarkdownConflicts(io *IOStreams, conflicts []proofConflict) {
	io.Printf("\n## Conflicts\n\n")
	if len(conflicts) == 0 {
		io.Println("_None identified._")
	}
	for i := range conflicts {
		c := &conflicts[i]
		io.Printf("- **%s:** %s\n", conflictLabel(c), conflictValuesString(c))
		if c.Resolved {
			io.Printf("  - _Resolved:_ %s\n", c.Resolution)
		} else {
			io.Printf("  - _Unresolved — resolution needed._\n")
		}
	}
}
