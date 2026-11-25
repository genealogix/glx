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

// Proof statuses: where the evidence leaves one candidate answer (a parent, a
// spouse, a child, an occupation, a possible same person). Every question
// resolves its candidates with ResolveProofStatus, so a disproven spouse, a
// disproven parent and a disproven same-person link are read the same way.
const (
	// ProofStatusProven means a surviving assertion is `proven` (or, with no
	// status, `high` confidence).
	ProofStatusProven = "proven"
	// ProofStatusProbable means the best surviving assertion is `probable`
	// (or, with no status, `medium` / `medium-high` confidence).
	ProofStatusProbable = "probable"
	// ProofStatusSpeculative means the best surviving assertion is
	// speculative, unresearched, low confidence, or has no rating at all.
	ProofStatusSpeculative = "speculative"
	// ProofStatusUnsupported means the archive records the candidate but no
	// assertion is about it.
	ProofStatusUnsupported = "unsupported"
	// ProofStatusDisputed means every surviving assertion is `disputed`: the
	// candidate is a contested alternative, not an answer.
	ProofStatusDisputed = "disputed"
	// ProofStatusConflicting means a `proven` and a `disproven` assertion
	// both stand: a conflict to resolve, not an answer.
	ProofStatusConflicting = "conflicting"
	// ProofStatusDisproven means every assertion is `disproven`: an excluded
	// candidate.
	ProofStatusDisproven = "disproven"
)

const statusProbable = "probable"

// proofStatusRank orders the answer statuses; anything else ranks 0.
func proofStatusRank(status string) int {
	switch status {
	case ProofStatusProven:
		return 3
	case ProofStatusProbable:
		return 2
	case ProofStatusSpeculative:
		return 1
	default:
		return 0
	}
}

// proofStatusSupport maps a candidate status to the support score the
// conclusion is drawn from.
func proofStatusSupport(status string) int {
	switch status {
	case ProofStatusProven:
		return supportStrong
	case ProofStatusProbable:
		return supportModerate
	case ProofStatusSpeculative:
		return supportWeak
	default:
		return supportNone
	}
}

// AssertionProofStatus reads one assertion's status. An explicit status wins;
// confidence decides only when the status is empty or unrecognized.
func AssertionProofStatus(a *Assertion) string {
	if a == nil {
		return ProofStatusUnsupported
	}
	switch strings.ToLower(strings.TrimSpace(a.Status)) {
	case statusProven:
		return ProofStatusProven
	case statusDisproven:
		return ProofStatusDisproven
	case standingStatusDisputed:
		return ProofStatusDisputed
	case statusProbable:
		return ProofStatusProbable
	case standingStatusSpeculative, standingStatusUnresearched:
		return ProofStatusSpeculative
	}
	if strings.EqualFold(strings.TrimSpace(a.Confidence), standingStatusDisputed) {
		return ProofStatusDisputed
	}

	return confidenceProofStatus(a.Confidence)
}

// confidenceProofStatus reads a confidence level alone: high is proven, medium
// (or medium-high) probable, anything else speculative.
func confidenceProofStatus(confidence string) string {
	switch researchConfidenceRank(confidence) {
	case 0:
		return ProofStatusProven
	case 1, 2:
		return ProofStatusProbable
	default:
		return ProofStatusSpeculative
	}
}

// ResolveProofStatus resolves the assertions about one candidate answer:
//
//   - none: unsupported;
//   - every one disproven: disproven (an excluded candidate);
//   - a proven and a disproven one together: conflicting;
//   - otherwise the strongest surviving non-disputed status, or disputed when
//     every survivor is disputed.
func ResolveProofStatus(assertions []*Assertion) string {
	best := ""
	surviving, disproven, proven := 0, false, false
	for _, a := range assertions {
		if a == nil {
			continue
		}
		status := AssertionProofStatus(a)
		switch status {
		case ProofStatusDisproven:
			disproven = true

			continue
		case ProofStatusDisputed:
			surviving++

			continue
		case ProofStatusProven:
			proven = true
		}
		surviving++
		if proofStatusRank(status) > proofStatusRank(best) || best == "" {
			best = status
		}
	}
	switch {
	case surviving == 0 && disproven:
		return ProofStatusDisproven
	case surviving == 0:
		return ProofStatusUnsupported
	case disproven && proven:
		return ProofStatusConflicting
	case best == "":
		return ProofStatusDisputed
	default:
		return best
	}
}

// statusConclusion renders a candidate status as a proof conclusion level.
func statusConclusion(status string) string {
	switch status {
	case ProofStatusProven:
		return proofConclusionProven
	case ProofStatusProbable:
		return proofConclusionProbable
	case ProofStatusSpeculative:
		return proofConclusionPossible
	case ProofStatusConflicting:
		return proofConclusionConflicted
	default:
		return proofConclusionInsufficient
	}
}

// ProofAnswer is one candidate answer that survives the evidence, with its own
// status, so a question with two answers of different strength reports both.
type ProofAnswer struct {
	PersonID     string   `json:"person_id,omitempty"`
	Label        string   `json:"label"`
	Role         string   `json:"role,omitempty"`
	Value        string   `json:"value,omitempty"`
	Date         string   `json:"date,omitempty"`
	Relationship string   `json:"relationship,omitempty"`
	Event        string   `json:"event,omitempty"`
	Status       string   `json:"status"`
	Conclusion   string   `json:"conclusion"`
	Assertions   []string `json:"assertions,omitempty"`
	SharedWith   []string `json:"shared_with,omitempty"`
}

// proofCandidate is one possible answer while it is being resolved.
type proofCandidate struct {
	key          string
	personID     string
	value        string
	date         string
	role         string
	relationship string
	event        string
	assertions   []*Assertion
	extra        string // strongest status of corroborating claims outside the candidate's own assertions
}

func (c *proofCandidate) add(assertions ...*Assertion) {
	for _, a := range assertions {
		if a != nil && !slices.Contains(c.assertions, a) {
			c.assertions = append(c.assertions, a)
		}
	}
}

func (c *proofCandidate) status() string {
	status := ResolveProofStatus(c.assertions)
	switch status {
	case ProofStatusDisproven, ProofStatusConflicting, ProofStatusDisputed:
		return status
	}
	if proofStatusRank(c.extra) > proofStatusRank(status) {
		return c.extra
	}

	return status
}

// proofCandidates is the resolved outcome of one question's candidates.
type proofCandidates struct {
	used         bool
	answers      []ProofAnswer
	alternatives []ProofExcludedAlternative
	conflicts    []ConflictGroup
	answerText   string
	found        bool
	level        int
	// rejected holds the assertions of excluded, contested and disputed
	// candidates: they stay in the evidence, but never support or answer the
	// question, and never enter conflict detection (the candidate's own status
	// already reports them).
	rejected map[*Assertion]bool
	notes    []string // extra summary sentences (e.g. distinguished namesakes)
	// answeredRelationships are the relationships behind the surviving answers.
	answeredRelationships map[string]bool
}

// isAnswerStatus reports whether a candidate status makes it an answer.
func isAnswerStatus(status string) bool {
	switch status {
	case ProofStatusDisproven, ProofStatusDisputed, ProofStatusConflicting:
		return false
	default:
		return true
	}
}

// resolveCandidates resolves every candidate with the shared status resolver
// and splits them into answers, excluded alternatives and conflicts. A
// disputed candidate is a contested alternative only beside an answer in the
// same role; with no such answer the dispute is the state of the question and
// is reported as an unresolved conflict. Candidates naming the same person
// (two relationships to one parent or spouse) are resolved one by one and then
// combined into a single answer.
func resolveCandidates(candidates []*proofCandidate, archive *GLXFile, ids map[*Assertion]string, labels map[string]string) proofCandidates {
	out := proofCandidates{used: true, level: supportStrong, rejected: map[*Assertion]bool{}, answeredRelationships: map[string]bool{}}
	statuses := make([]string, len(candidates))
	answered := map[string]bool{}
	for i, c := range candidates {
		statuses[i] = c.status()
		if isAnswerStatus(statuses[i]) {
			answered[c.role] = true
		}
	}
	answerIndex := map[string]int{}
	for i, c := range candidates {
		out.place(c, statuses[i], answered[c.role], candidateLabel(c, archive, labels), ids, answerIndex)
	}
	for i := range out.answers {
		out.level = min(out.level, proofStatusSupport(out.answers[i].Status))
	}
	if len(out.answers) == 0 {
		out.level = supportNone
	}

	return out
}

// place files one resolved candidate as an excluded alternative, a conflict
// or an answer; answers about the same person are merged.
func (out *proofCandidates) place(c *proofCandidate, status string, roleAnswered bool, label string, ids map[*Assertion]string, answerIndex map[string]int) {
	assertionIDs := make([]string, 0, len(c.assertions))
	for _, a := range c.assertions {
		if id := ids[a]; id != "" {
			assertionIDs = append(assertionIDs, id)
		}
	}
	reject := func() {
		for _, a := range c.assertions {
			out.rejected[a] = true
		}
	}
	switch {
	case status == ProofStatusDisproven || status == ProofStatusDisputed && roleAnswered:
		reject()
		out.alternatives = append(out.alternatives, proofAlternative(c, label, status, assertionIDs))
	case status == ProofStatusDisputed:
		reject()
		out.conflicts = append(out.conflicts, candidateConflict(c, label, ids, "disputed, no accepted alternative"))
	case status == ProofStatusConflicting:
		out.conflicts = append(out.conflicts, candidateConflict(c, label, ids, "proven and disproven assertions"))
	default:
		out.addAnswer(c, status, label, assertionIDs, answerIndex)
	}
}

// addAnswer records an accepted candidate, merging it into the answer about
// the same person when there is one.
func (out *proofCandidates) addAnswer(c *proofCandidate, status, label string, assertionIDs []string, answerIndex map[string]int) {
	if c.relationship != "" {
		out.answeredRelationships[c.relationship] = true
	}
	if c.personID != "" {
		key := c.role + "\x00" + c.personID
		if j, seen := answerIndex[key]; seen {
			mergeProofAnswer(&out.answers[j], status, assertionIDs)

			return
		}
		answerIndex[key] = len(out.answers)
	}
	out.answers = append(out.answers, ProofAnswer{
		PersonID: c.personID, Label: label, Role: c.role, Value: c.value, Date: c.date,
		Relationship: c.relationship, Event: c.event,
		Status: status, Conclusion: statusConclusion(status), Assertions: assertionIDs,
	})
}

// candidateLabel names a candidate: the person (told apart from namesakes), or
// its value with the period it is claimed for.
func candidateLabel(c *proofCandidate, archive *GLXFile, labels map[string]string) string {
	if c.personID == "" {
		if c.date != "" {
			return c.value + " (" + c.date + ")"
		}

		return c.value
	}
	if label := labels[c.personID]; label != "" {
		return label
	}

	return personName(archive, c.personID)
}

// mergeProofAnswer combines a second accepted claim about the same person
// (another relationship to the same parent or spouse) into its answer. Any one
// accepted claim establishes the person, so the stronger status stands.
func mergeProofAnswer(answer *ProofAnswer, status string, assertionIDs []string) {
	if proofStatusRank(status) > proofStatusRank(answer.Status) || answer.Status == ProofStatusUnsupported && status != ProofStatusUnsupported {
		answer.Status = status
		answer.Conclusion = statusConclusion(status)
	}
	for _, id := range assertionIDs {
		if !slices.Contains(answer.Assertions, id) {
			answer.Assertions = append(answer.Assertions, id)
		}
	}
}

func proofAlternative(c *proofCandidate, label, status string, assertionIDs []string) ProofExcludedAlternative {
	alt := ProofExcludedAlternative{
		PersonID: c.personID, Name: label, Relationship: c.relationship,
		Event: c.event, Role: c.role, Value: c.value, Status: status, Assertions: assertionIDs,
	}
	var notes []string
	for _, a := range c.assertions {
		for _, cit := range a.Citations {
			if !slices.Contains(alt.Citations, cit) {
				alt.Citations = append(alt.Citations, cit)
			}
		}
		if note := firstLine(a.Notes.String()); note != "" && !slices.Contains(notes, note) {
			notes = append(notes, note)
		}
	}
	alt.Notes = strings.Join(notes, "; ")

	return alt
}

// candidateConflict reports a candidate its own assertions leave unresolved:
// both proven and disproven, or disputed with no accepted alternative. It
// stands until the evidence settles it.
func candidateConflict(c *proofCandidate, label string, ids map[*Assertion]string, reason string) ConflictGroup {
	facts := make([]ResearchFact, 0, len(c.assertions))
	values := map[string]*ConflictValue{}
	for _, a := range c.assertions {
		facts = append(facts, researchFact(&proofAssertion{id: ids[a], a: a, subjectID: c.relationship + c.event}))
		status := AssertionProofStatus(a)
		if values[status] == nil {
			values[status] = &ConflictValue{Value: label + " " + status, Confidence: a.Confidence, Status: a.Status}
		}
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]ConflictValue, 0, len(keys))
	for _, k := range keys {
		out = append(out, *values[k])
	}
	subject := c.relationship
	if subject == "" {
		subject = c.event
	}
	property := c.role
	if property == "" {
		property = "candidate"
	}

	return ConflictGroup{
		Facts: facts, Verdict: VerdictDisputed, Definite: true,
		Subject: subject, Property: property + " (" + reason + ")", Values: out,
	}
}

// candidateAnswerText renders the surviving answers. When they differ in
// strength each carries its own conclusion, so the weaker answer is not read as
// proven alongside the stronger one.
func candidateAnswerText(prefix string, answers []ProofAnswer) string {
	if len(answers) == 0 {
		return ""
	}
	labels := make([]string, len(answers))
	mixed := false
	for i := range answers {
		labels[i] = answers[i].Label
		if answers[i].Status != answers[0].Status {
			mixed = true
		}
	}
	text := prefix + strings.Join(labels, ", ") + "."
	if !mixed {
		return text
	}
	parts := make([]string, len(answers))
	for i := range answers {
		role := answers[i].Role
		if role == "" {
			role = "answer"
		}
		parts[i] = strings.ToUpper(role[:1]) + role[1:] + " " + answers[i].Label + ": " + answers[i].Conclusion
	}

	return text + " " + strings.Join(parts, "; ") + "."
}

// unsupportedAnswerText names the answers no assertion supports: recorded in
// the archive, but not yet evidenced.
func unsupportedAnswerText(answers []ProofAnswer) string {
	var labels []string
	for i := range answers {
		if answers[i].Status == ProofStatusUnsupported {
			labels = append(labels, answers[i].Label)
		}
	}
	if len(labels) == 0 {
		return ""
	}

	return "No assertion supports " + strings.Join(labels, ", ") + "."
}

// allUnsupported reports whether no answer has an assertion behind it.
func allUnsupported(answers []ProofAnswer) bool {
	for i := range answers {
		if answers[i].Status != ProofStatusUnsupported {
			return false
		}
	}

	return len(answers) > 0
}

// recordedWithoutAssertionText names answers the archive records but no
// assertion supports, when none is supported.
func recordedWithoutAssertionText(answers []ProofAnswer) string {
	labels := make([]string, len(answers))
	for i := range answers {
		labels[i] = answers[i].Label
	}

	return "Recorded without a supporting assertion: " + strings.Join(labels, ", ") + "."
}

// assertionIDIndex maps each archive assertion to its ID.
func assertionIDIndex(archive *GLXFile) map[*Assertion]string {
	ids := make(map[*Assertion]string, len(archive.Assertions))
	for id, a := range archive.Assertions {
		ids[a] = id
	}

	return ids
}

// parentRole names a parent by sex.
func parentRole(archive *GLXFile, personID string) string {
	if p := archive.Persons[personID]; p != nil {
		switch strings.ToLower(propertyString(p.Properties, PersonPropertySex)) {
		case SexMale:
			return parentageFather
		case SexFemale:
			return parentageMother
		}
	}

	return ParticipantRoleParent
}

// parentageAnswerCandidates resolves each parent link with the shared
// resolver. A link's own assertions decide its status, and a claim on the
// person's birth record naming a linked parent corroborates it. A parent named
// only by claims (on the birth record, or as a participant of a parent_child
// relationship that does not yet list them) is a candidate of its own, decided
// by those claims.
func parentageAnswerCandidates(personID string, archive *GLXFile, relevant []proofAssertion, links []ParentChildLink, standings *RelationshipStandingIndex) []*proofCandidate {
	set := newCandidateSet("")
	newParent := func(key, parentID, relID string) *proofCandidate {
		c := set.get(key, parentID)
		c.role = parentRole(archive, parentID)
		if c.relationship == "" {
			c.relationship = relID
		}

		return c
	}
	for _, link := range links {
		if link.ChildID != personID || link.Step {
			continue
		}
		newParent("rel:"+link.RelationshipID+":"+link.ParentID, link.ParentID, link.RelationshipID).
			add(standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID)...)
	}
	for i := range relevant {
		pa := &relevant[i]
		p := pa.a.Participant
		if pa.synthetic || p == nil || p.Person == "" || p.Person == personID || !IsParentSideRole(p.Role) || IsStepRole(p.Role) {
			continue
		}
		switch {
		case pa.a.Subject.Event != "":
			if linked := set.linked(p.Person); len(linked) > 0 {
				for _, c := range linked {
					c.corroborate(pa.a)
				}

				continue
			}
			newParent("claim:"+p.Person, p.Person, "").add(pa.a)
		case pa.a.Subject.Relationship != "" && IsParentChildRelationshipType(pa.relType) && parentageClaimOnly(pa, personID, links):
			relID := pa.a.Subject.Relationship
			newParent("rel:"+relID+":"+p.Person, p.Person, relID).add(standings.LinkAssertions(relID, p.Person)...)
		}
	}

	return set.order
}

// candidateSet collects one question's candidates by key, in first-seen order.
type candidateSet struct {
	byKey    map[string]*proofCandidate
	byPerson map[string][]*proofCandidate
	order    []*proofCandidate
	role     string
}

func newCandidateSet(role string) *candidateSet {
	return &candidateSet{byKey: map[string]*proofCandidate{}, byPerson: map[string][]*proofCandidate{}, role: role}
}

func (s *candidateSet) get(key, personID string) *proofCandidate {
	c := s.byKey[key]
	if c == nil {
		c = &proofCandidate{key: key, personID: personID, role: s.role}
		s.byKey[key] = c
		s.order = append(s.order, c)
		if personID != "" {
			s.byPerson[personID] = append(s.byPerson[personID], c)
		}
	}

	return c
}

// linked returns the person's candidates that rest on a recorded relationship.
func (s *candidateSet) linked(personID string) []*proofCandidate {
	var out []*proofCandidate
	for _, c := range s.byPerson[personID] {
		if c.relationship != "" {
			out = append(out, c)
		}
	}

	return out
}

// corroborate records a claim that supports the candidate without deciding it:
// it can raise the status, never lower it, and a disproven one says nothing.
func (c *proofCandidate) corroborate(a *Assertion) {
	if status := AssertionProofStatus(a); status != ProofStatusDisproven && proofStatusRank(status) > proofStatusRank(c.extra) {
		c.extra = status
	}
}

// marriageAnswerCandidates gathers every possible spouse, one candidate per
// relationship or record: the other partners of the person's couple
// relationships, the counterparts of a couple relationship the person is only
// claimed (or disclaimed) to belong to, the other principals of the person's
// weddings, and the opposite principals of a wedding the person is only
// claimed to have taken part in. Banns, licenses, contracts and engagements
// prepare a marriage without proving it took place, so they stay contextual
// evidence and make no spouse candidate.
func marriageAnswerCandidates(personID string, archive *GLXFile, standings *RelationshipStandingIndex) []*proofCandidate {
	set := newCandidateSet(ParticipantRoleSpouse)
	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil || !isSpouseRelationshipType(rel.Type) || !hasParticipant(personID, rel.Participants) {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == "" || p.Person == personID {
				continue
			}
			c := set.get("rel:"+relID+":"+p.Person, p.Person)
			c.relationship = relID
			c.add(principalClaims(standings.LinkAssertions(relID, personID, p.Person))...)
		}
	}
	addMarriageRelationshipClaims(set, personID, archive)
	eventAssertions := assertionsByEvent(archive)
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || event.Type != EventTypeMarriage {
			continue
		}
		if isMarriageEventPrincipal(event, personID) {
			addMarriageEventSpouses(set, eventID, event, personID, archive, eventAssertions[eventID])
		} else {
			addMarriageEventClaims(set, eventID, event, personID, eventAssertions[eventID])
		}
	}

	return set.order
}

func isSpouseRelationshipType(relType string) bool {
	return relType == RelationshipTypeMarriage || relType == RelationshipTypePartner
}

// principalClaims drops participant claims in a non-principal role (a
// witness, an officiant): they say nothing about who married whom.
// Existential and type assertions are kept.
func principalClaims(assertions []*Assertion) []*Assertion {
	return slices.DeleteFunc(slices.Clone(assertions), func(a *Assertion) bool {
		return a != nil && a.Participant != nil && !isCouplePrincipal(a.Participant.Role)
	})
}

// isCouplePrincipal reports whether a participant role is one of the couple
// in a marriage or partnership.
func isCouplePrincipal(role string) bool {
	return isMarriagePrincipal(role) || role == RelationshipTypePartner
}

// addMarriageRelationshipClaims weighs claims that the person belongs to a
// couple relationship that does not list them (or, disproven, that they do
// not): the counterparts recorded opposite the claimed role are candidates.
func addMarriageRelationshipClaims(set *candidateSet, personID string, archive *GLXFile) {
	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil || a.Participant == nil || a.Participant.Person != personID || !isCouplePrincipal(a.Participant.Role) {
			continue
		}
		relID := a.Subject.Relationship
		rel := archive.Relationships[relID]
		if rel == nil || !isSpouseRelationshipType(rel.Type) || hasParticipant(personID, rel.Participants) {
			continue
		}
		for _, spouse := range coupleCounterparts(rel.Participants, personID, a.Participant.Role) {
			c := set.get("rel:"+relID+":"+spouse, spouse)
			c.relationship = relID
			c.add(a)
		}
	}
}

// coupleCounterparts returns the relationship participants a person in the
// given role would be coupled with: the groom for a bride, the bride for a
// groom, otherwise everyone else.
func coupleCounterparts(participants []Participant, personID, role string) []string {
	var out []string
	for _, p := range participants {
		if p.Person == "" || p.Person == personID || slices.Contains(out, p.Person) {
			continue
		}
		if role == ParticipantRoleBride && p.Role != ParticipantRoleGroom || role == ParticipantRoleGroom && p.Role != ParticipantRoleBride {
			continue
		}
		out = append(out, p.Person)
	}

	return out
}

// addMarriageEventSpouses weighs a wedding the person is recorded in. A
// participation or existence claim decides the marriage; a dated or placed
// claim corroborates it, and a disproven date says nothing against the
// marriage itself. Only claims about the couple count.
func addMarriageEventSpouses(set *candidateSet, eventID string, event *Event, personID string, archive *GLXFile, assertions []*Assertion) {
	for _, c := range weddingCandidates(set, eventID, event, personID, archive) {
		if c.event == "" {
			c.event = eventID
		}
		for _, a := range assertions {
			if a.Participant != nil && (!isMarriagePrincipal(a.Participant.Role) || a.Participant.Person != personID && a.Participant.Person != c.personID) {
				continue
			}
			switch a.Property {
			case "":
				c.add(a)
			case conflictDateType, eventFieldPlace:
				c.corroborate(a)
			}
		}
	}
}

// weddingCandidates returns the spouse candidates a wedding the person is
// recorded in belongs to. A wedding naming only the person belongs to a
// spouse only when a relationship names it as its start event: it may record
// another marriage.
func weddingCandidates(set *candidateSet, eventID string, event *Event, personID string, archive *GLXFile) []*proofCandidate {
	spouses := opposingPrincipals(event.Participants, personID, "")
	if len(spouses) > 0 {
		out := make([]*proofCandidate, 0, len(spouses))
		for _, spouse := range spouses {
			out = append(out, marriageEventCandidates(set, eventID, spouse, archive)...)
		}

		return out
	}

	return slices.DeleteFunc(slices.Clone(set.order), func(c *proofCandidate) bool {
		rel := archive.Relationships[c.relationship]

		return c.personID == "" || rel == nil || rel.StartEvent != eventID
	})
}

// marriageEventCandidates returns the candidates a wedding naming the spouse
// belongs to: the relationship it starts, else the couple's one relationship,
// else a candidate of the wedding's own.
func marriageEventCandidates(set *candidateSet, eventID, spouse string, archive *GLXFile) []*proofCandidate {
	linked := set.linked(spouse)
	var started []*proofCandidate
	for _, c := range linked {
		if rel := archive.Relationships[c.relationship]; rel != nil && rel.StartEvent == eventID {
			started = append(started, c)
		}
	}
	switch {
	case len(started) > 0:
		return started
	case len(linked) == 1 && (archive.Relationships[linked[0].relationship] == nil || archive.Relationships[linked[0].relationship].StartEvent == ""):
		return linked
	default:
		return []*proofCandidate{set.get("event:"+eventID+":"+spouse, spouse)}
	}
}

// addMarriageEventClaims weighs claims that the person took part in a wedding
// that does not list them (or, disproven, that they did not), as bride, groom
// or spouse.
func addMarriageEventClaims(set *candidateSet, eventID string, event *Event, personID string, assertions []*Assertion) {
	for _, a := range assertions {
		if a.Participant == nil || a.Participant.Person != personID || !isMarriagePrincipal(a.Participant.Role) {
			continue
		}
		spouses := opposingPrincipals(event.Participants, personID, a.Participant.Role)
		if len(spouses) == 0 {
			c := set.get("event:"+eventID, "")
			c.event, c.value = eventID, eventID
			c.add(a)

			continue
		}
		for _, spouse := range spouses {
			c := set.get("claim:"+eventID+":"+spouse, spouse)
			c.event = eventID
			c.add(a)
		}
	}
}

func assertionsByEvent(archive *GLXFile) map[string][]*Assertion {
	out := map[string][]*Assertion{}
	for _, id := range sortedKeys(archive.Assertions) {
		if a := archive.Assertions[id]; a != nil && a.Subject.Event != "" {
			out[a.Subject.Event] = append(out[a.Subject.Event], a)
		}
	}

	return out
}

// isMarriageEventPrincipal reports whether the person is recorded in the
// marriage event in a principal role.
func isMarriageEventPrincipal(event *Event, personID string) bool {
	for _, p := range event.Participants {
		if p.Person == personID && isMarriagePrincipal(p.Role) {
			return true
		}
	}

	return false
}

// opposingPrincipals returns the participants a person in the given role
// would have married: the groom for a bride, the bride for a groom, otherwise
// every other principal.
func opposingPrincipals(participants []Participant, personID, role string) []string {
	want := func(p Participant) bool {
		switch role {
		case ParticipantRoleBride:
			return p.Role == ParticipantRoleGroom
		case ParticipantRoleGroom:
			return p.Role == ParticipantRoleBride
		default:
			return isMarriagePrincipal(p.Role)
		}
	}
	var out []string
	for _, p := range participants {
		if p.Person != "" && p.Person != personID && want(p) && !slices.Contains(out, p.Person) {
			out = append(out, p.Person)
		}
	}

	return out
}

// childrenAnswerCandidates gathers the person's children: the child side of
// every non-step parent_child link where the person is a parent, one candidate
// per link, and the principal of every birth or baptism where the person is
// recorded, or only claimed, as parent.
func childrenAnswerCandidates(personID string, archive *GLXFile, links []ParentChildLink, standings *RelationshipStandingIndex) []*proofCandidate {
	set := newCandidateSet(ParticipantRoleChild)
	for _, link := range links {
		if link.ParentID != personID || link.Step {
			continue
		}
		c := set.get("rel:"+link.RelationshipID+":"+link.ChildID, link.ChildID)
		c.relationship = link.RelationshipID
		c.add(standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID)...)
	}
	eventAssertions := assertionsByEvent(archive)
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || !isBirthEventType(event.Type) {
			continue
		}
		claims := parentClaims(personID, eventAssertions[eventID])
		if !eventHasParent(event, personID) && len(claims) == 0 {
			continue
		}
		for _, p := range event.Participants {
			if p.Person != "" && p.Person != personID && isVitalPrincipal(event.Type, p.Role) {
				addBirthRecordChild(set, eventID, p.Person, claims)
			}
		}
	}

	return set.order
}

// addBirthRecordChild weighs a birth or baptism of a possible child. A
// recorded link decides the child and the record's parent claims corroborate
// it; with no link, those claims decide it.
func addBirthRecordChild(set *candidateSet, eventID, childID string, claims []*Assertion) {
	if linked := set.linked(childID); len(linked) > 0 {
		for _, c := range linked {
			for _, a := range claims {
				c.corroborate(a)
			}
		}

		return
	}
	c := set.get("event:"+childID, childID)
	if c.event == "" {
		c.event = eventID
	}
	c.add(claims...)
}

// parentClaims returns the claims that the person is the (non-step) parent
// named in a birth or baptism.
func parentClaims(personID string, assertions []*Assertion) []*Assertion {
	var out []*Assertion
	for _, a := range assertions {
		if a.Participant != nil && a.Participant.Person == personID && IsParentSideRole(a.Participant.Role) && !IsStepRole(a.Participant.Role) {
			out = append(out, a)
		}
	}

	return out
}

func eventHasParent(event *Event, personID string) bool {
	for _, p := range event.Participants {
		if p.Person == personID && IsParentSideRole(p.Role) && !IsStepRole(p.Role) {
			return true
		}
	}

	return false
}

// sameIdentityCandidates gathers the person's possibly_same_person links.
func sameIdentityCandidates(personID string, archive *GLXFile, standings *RelationshipStandingIndex) []*proofCandidate {
	var order []*proofCandidate
	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil || rel.Type != RelationshipTypePossiblySamePerson || !hasParticipant(personID, rel.Participants) {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == "" || p.Person == personID {
				continue
			}
			c := &proofCandidate{key: relID + ":" + p.Person, personID: p.Person, role: RelationshipTypePossiblySamePerson, relationship: relID}
			c.add(standings.LinkAssertions(relID, personID, p.Person)...)
			order = append(order, c)
		}
	}

	return order
}

// normalizeProofValue folds a property value for grouping.
func normalizeProofValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// propertyAnswerCandidates groups the claims of one person property (for
// example occupation) by value and period: "church elder" disproven for 1755
// and proven for 1762 are two candidates, so neither decides the other. An
// undated claim is a period of its own. A value recorded on the person or on
// one of their event participations without an assertion is an unsupported
// candidate, unless a claim of the same value in a compatible period already
// stands for it.
func propertyAnswerCandidates(personID, property string, archive *GLXFile, relevant []proofAssertion) []*proofCandidate {
	set := &propertyCandidateSet{property: property, byKey: map[string]*proofCandidate{}, byValue: map[string][]*proofCandidate{}}
	for i := range relevant {
		pa := &relevant[i]
		if value := assertionPropertyValue(pa.a, property); !pa.synthetic && value != "" {
			set.get(value, string(pa.a.Date)).add(pa.a)
		}
	}
	if person := archive.Persons[personID]; person != nil {
		for _, entry := range temporalPropertyEntries(person.Properties[property]) {
			set.recorded(entry)
		}
	}
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				set.recordedOnEvent(eventID, p.Properties[property])
			}
		}
	}

	return set.order
}

// propertyCandidateSet collects one property's candidates by value and period.
type propertyCandidateSet struct {
	property string
	byKey    map[string]*proofCandidate
	byValue  map[string][]*proofCandidate
	order    []*proofCandidate
}

func (s *propertyCandidateSet) get(value, date string) *proofCandidate {
	folded := normalizeProofValue(value)
	key := folded + "\x00" + date
	c := s.byKey[key]
	if c == nil {
		c = &proofCandidate{key: key, value: strings.TrimSpace(value), date: date, role: s.property}
		s.byKey[key] = c
		s.byValue[folded] = append(s.byValue[folded], c)
		s.order = append(s.order, c)
	}

	return c
}

// recorded finds the candidate for a value recorded without an assertion: one
// for the same period, or for an undated claim or record; else a new one.
func (s *propertyCandidateSet) recorded(entry propertyEntry) *proofCandidate {
	for _, c := range s.byValue[normalizeProofValue(entry.value)] {
		if c.date == entry.date || c.date == "" || entry.date == "" {
			if c.date == "" && entry.date != "" && len(c.assertions) == 0 {
				c.date = entry.date
			}

			return c
		}
	}

	return s.get(entry.value, entry.date)
}

// recordedOnEvent records the values of the property on one of the person's
// event participations.
func (s *propertyCandidateSet) recordedOnEvent(eventID string, raw any) {
	for _, entry := range temporalPropertyEntries(raw) {
		if c := s.recorded(entry); c.event == "" {
			c.event = eventID
		}
	}
}

// assertionPropertyValue returns the value an assertion claims for a person
// property: its own value, or, for a participant claim, the property recorded
// on the participant (participant.properties.occupation).
func assertionPropertyValue(a *Assertion, property string) string {
	if a.Participant != nil {
		if entries := temporalPropertyEntries(a.Participant.Properties[property]); len(entries) > 0 {
			return entries[0].value
		}

		return ""
	}
	if a.Property == property {
		return a.Value
	}

	return ""
}

// propertyEntry is one value of a simple or temporal property, with the date
// recorded for it.
type propertyEntry struct {
	value string
	date  string
}

// temporalPropertyEntries extracts the values of a simple or temporal property, in any
// form the SDK supports (string, map, list, TemporalValue and its pointer and
// slice forms), with each entry's date.
func temporalPropertyEntries(raw any) []propertyEntry {
	switch v := normalizeTemporalProperty(raw).(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return []propertyEntry{{value: v}}
		}
	case []any:
		var out []propertyEntry
		for _, item := range v {
			out = append(out, temporalPropertyEntries(item)...)
		}

		return out
	case map[string]any:
		date := ""
		switch d := v[temporalDateField].(type) {
		case string:
			date = d
		case DateString:
			date = string(d)
		}
		entries := temporalPropertyEntries(v[temporalValueField])
		for i := range entries {
			if entries[i].date == "" {
				entries[i].date = date
			}
		}

		return entries
	}

	return nil
}

// sharedPropertyHolders names the other people recorded with the same value of
// a property (two men holding one office), with the dates recorded for them.
func sharedPropertyHolders(personID, property, value string, archive *GLXFile) []string {
	key := normalizeProofValue(value)
	dates := map[string][]string{}
	var ids []string
	addDate := func(id, date string) {
		if _, seen := dates[id]; !seen {
			ids = append(ids, id)
			dates[id] = nil
		}
		if date != "" && !slices.Contains(dates[id], date) {
			dates[id] = append(dates[id], date)
		}
	}
	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil {
			continue
		}
		other := a.Subject.Person
		if other == "" || other == personID || a.Property != property || AssertionIsDisproven(a) || normalizeProofValue(a.Value) != key {
			continue
		}
		addDate(other, string(a.Date))
	}
	for _, id := range sortedKeys(archive.Persons) {
		person := archive.Persons[id]
		if id == personID || person == nil {
			continue
		}
		for _, entry := range temporalPropertyEntries(person.Properties[property]) {
			if normalizeProofValue(entry.value) == key {
				addDate(id, entry.date)
			}
		}
	}
	labels := DistinguishPersonNames(archive, append([]string{personID}, ids...))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		label := labels[id]
		if !strings.Contains(label, id) {
			label += " (" + id + ")"
		}
		if len(dates[id]) > 0 {
			label += " " + strings.Join(dates[id], "; ")
		}
		out = append(out, label)
	}

	return out
}

// DistinguishPersonNames returns a display label for each person ID. A name is
// returned as is unless another listed person shares it; then it carries the
// person's life dates, as the ancestors view prints them, and the person ID:
// "Johann Anton Schepp (d. 1741) p-a".
func DistinguishPersonNames(archive *GLXFile, personIDs []string) map[string]string {
	names := make(map[string]string, len(personIDs))
	counts := map[string]int{}
	for _, id := range personIDs {
		if _, seen := names[id]; seen {
			continue
		}
		names[id] = personName(archive, id)
		counts[names[id]]++
	}
	out := make(map[string]string, len(names))
	for id, name := range names {
		if counts[name] < 2 || name == id {
			out[id] = name

			continue
		}
		if dates := personLifeDates(archive, id); dates != "" {
			out[id] = name + " (" + dates + ") " + id
		} else {
			out[id] = name + " (" + id + ")"
		}
	}

	return out
}

// personLifeDates renders "b. X", "d. Y" or "X – Y" from the person's birth
// (or baptism) and death (or burial) events.
func personLifeDates(archive *GLXFile, personID string) string {
	first := func(types ...string) string {
		for _, t := range types {
			if _, event := FindPersonEvent(archive, personID, t); event != nil && event.Date != "" {
				return string(event.Date)
			}
		}

		return ""
	}
	born := first(EventTypeBirth, EventTypeBaptism, EventTypeChristening)
	died := first(EventTypeDeath, EventTypeBurial)
	switch {
	case born != "" && died != "":
		return born + " – " + died
	case born != "":
		return "b. " + born
	case died != "":
		return "d. " + died
	default:
		return ""
	}
}

// candidatePersonIDs lists the person IDs a question's candidates name, with
// the subject first so a namesake of the subject is also told apart.
func candidatePersonIDs(personID string, candidates []*proofCandidate) []string {
	ids := []string{personID}
	for _, c := range candidates {
		if c.personID != "" {
			ids = append(ids, c.personID)
		}
	}

	return ids
}

// buildProofCandidates resolves the candidate answers of the questions that
// have them. Birth and death have one answer each and keep the fact engine.
func buildProofCandidates(topic, personID string, archive *GLXFile, relevant []proofAssertion, links []ParentChildLink, standings *RelationshipStandingIndex) proofCandidates {
	var candidates []*proofCandidate
	prefix := ""
	switch topic {
	case topicParentage:
		candidates = parentageAnswerCandidates(personID, archive, relevant, links, standings)
		prefix = "Parents identified: "
	case topicMarriage:
		candidates = marriageAnswerCandidates(personID, archive, standings)
		prefix = "Married: "
	case topicChildren:
		candidates = childrenAnswerCandidates(personID, archive, links, standings)
		prefix = "Children identified: "
	case topicIdentity:
		candidates = sameIdentityCandidates(personID, archive, standings)
	case topicOccupation:
		candidates = propertyAnswerCandidates(personID, PersonPropertyOccupation, archive, relevant)
		prefix = "Occupation: "
	default:
		return proofCandidates{}
	}
	if len(candidates) == 0 {
		return proofCandidates{}
	}
	labels := DistinguishPersonNames(archive, candidatePersonIDs(personID, candidates))
	resolved := resolveCandidates(candidates, archive, assertionIDIndex(archive), labels)
	if topic == topicIdentity {
		return identityOutcome(personID, archive, &resolved)
	}
	if topic == topicOccupation {
		for i := range resolved.answers {
			a := &resolved.answers[i]
			a.SharedWith = sharedPropertyHolders(personID, PersonPropertyOccupation, a.Value, archive)
		}
	}
	resolved.answerText = candidateAnswerText(prefix, resolved.answers)
	resolved.found = len(resolved.answers) > 0

	return resolved
}

// identityOutcome turns resolved same-person links into identity statements:
// an accepted link names the same person, a disproven one a namesake told
// apart, and every other link is a conflict to resolve. Each link stands on its
// own: a disputed link is never set aside because another link is accepted.
// The identity is only as strong as its weakest accepted link.
func identityOutcome(personID string, archive *GLXFile, resolved *proofCandidates) proofCandidates {
	out := proofCandidates{used: true, rejected: resolved.rejected, conflicts: resolved.conflicts, level: supportStrong}
	openLink := func(relID, label, status string) {
		out.conflicts = append(out.conflicts, ConflictGroup{
			Subject: relID, Property: RelationshipTypePossiblySamePerson,
			Verdict: VerdictDisputed, Definite: true,
			Values: []ConflictValue{{Value: personName(archive, personID) + " = " + label, Status: status}},
		})
	}
	var same []string
	for i := range resolved.answers {
		a := resolved.answers[i]
		if a.Status == ProofStatusProven || a.Status == ProofStatusProbable {
			a.Role = "same_person"
			out.answers = append(out.answers, a)
			same = append(same, a.Label)
			out.level = min(out.level, proofStatusSupport(a.Status))

			continue
		}
		openLink(a.Relationship, a.Label, a.Status)
	}
	if len(same) > 0 {
		out.found = true
		out.notes = append(out.notes, "Same person as "+strings.Join(same, ", ")+".")
	} else {
		out.level = supportNone
	}
	var apart []string
	for i := range resolved.alternatives {
		alt := resolved.alternatives[i]
		if alt.Status != ProofStatusDisproven {
			openLink(alt.Relationship, alt.Name, alt.Status)

			continue
		}
		alt.Role = "distinguished_from"
		out.alternatives = append(out.alternatives, alt)
		apart = append(apart, alt.Name)
	}
	if len(apart) > 0 {
		out.notes = append(out.notes, "Distinguished from "+strings.Join(apart, ", ")+".")
	}

	return out
}

// excludedMarriageRelationships returns the relationships of excluded spouses
// that no accepted answer also rests on: no marriage record is wanted for a
// marriage the evidence rules out, but the gap of an accepted marriage stays,
// even when an excluded candidate shares the spouse's name.
func excludedMarriageRelationships(candidates *proofCandidates) map[string]bool {
	out := map[string]bool{}
	for i := range candidates.alternatives {
		if rel := candidates.alternatives[i].Relationship; rel != "" && !candidates.answeredRelationships[rel] {
			out[rel] = true
		}
	}

	return out
}

// excludedAlternativeText is the summary sentence for a question's excluded
// candidates, used when nothing else answers it.
func excludedAlternativeText(alternatives []ProofExcludedAlternative) string {
	if len(alternatives) == 0 {
		return ""
	}
	names := make([]string, len(alternatives))
	for i := range alternatives {
		names[i] = alternatives[i].Name
	}

	return fmt.Sprintf("Excluded: %s.", strings.Join(names, ", "))
}
