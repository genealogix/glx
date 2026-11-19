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
	rejected     map[*Assertion]bool // assertions of alternatives: never support or answer
	notes        []string            // extra summary sentences (e.g. distinguished namesakes)
}

// resolveCandidates resolves every candidate with the shared status resolver
// and splits them into answers, excluded alternatives and conflicts.
func resolveCandidates(candidates []*proofCandidate, archive *GLXFile, ids map[*Assertion]string, labels map[string]string) proofCandidates {
	out := proofCandidates{used: true, level: supportStrong, rejected: map[*Assertion]bool{}}
	for _, c := range candidates {
		status := c.status()
		label := c.value
		if c.personID != "" {
			label = labels[c.personID]
			if label == "" {
				label = personName(archive, c.personID)
			}
		}
		assertionIDs := make([]string, 0, len(c.assertions))
		for _, a := range c.assertions {
			if id := ids[a]; id != "" {
				assertionIDs = append(assertionIDs, id)
			}
		}
		switch status {
		case ProofStatusDisproven, ProofStatusDisputed:
			for _, a := range c.assertions {
				out.rejected[a] = true
			}
			out.alternatives = append(out.alternatives, proofAlternative(c, label, status, assertionIDs))
		case ProofStatusConflicting:
			out.conflicts = append(out.conflicts, candidateConflict(c, label, ids))
		default:
			out.answers = append(out.answers, ProofAnswer{
				PersonID: c.personID, Label: label, Role: c.role, Value: c.value, Date: c.date,
				Relationship: c.relationship, Event: c.event,
				Status: status, Conclusion: statusConclusion(status), Assertions: assertionIDs,
			})
			out.level = min(out.level, proofStatusSupport(status))
		}
	}
	if len(out.answers) == 0 {
		out.level = supportNone
	}
	// A disputed candidate is a contested alternative beside an answer in the
	// same role. With no such answer the dispute is the state of the question,
	// so its conflict stands.
	answered := map[string]bool{}
	for i := range out.answers {
		answered[out.answers[i].Role] = true
	}
	for _, c := range candidates {
		if !answered[c.role] && c.status() == ProofStatusDisputed {
			for _, a := range c.assertions {
				delete(out.rejected, a)
			}
		}
	}

	return out
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

// candidateConflict reports a candidate whose own assertions both prove and
// disprove it. It is unresolved until one of them is withdrawn.
func candidateConflict(c *proofCandidate, label string, ids map[*Assertion]string) ConflictGroup {
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
		Subject: subject, Property: property + " (proven and disproven assertions)", Values: out,
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

// parentageAnswerCandidates resolves each parent with the shared resolver. A
// parent's own link assertions decide its status; a claim on the person's birth
// record naming that parent corroborates it.
func parentageAnswerCandidates(personID string, archive *GLXFile, relevant []proofAssertion, links []ParentChildLink, standings *RelationshipStandingIndex) []*proofCandidate {
	byParent := map[string]*proofCandidate{}
	var order []*proofCandidate
	for _, link := range links {
		if link.ChildID != personID || link.Step {
			continue
		}
		c := byParent[link.ParentID]
		if c == nil {
			c = &proofCandidate{key: link.ParentID, personID: link.ParentID, role: parentRole(archive, link.ParentID), relationship: link.RelationshipID}
			byParent[link.ParentID] = c
			order = append(order, c)
		}
		c.add(standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID)...)
	}
	for i := range relevant {
		pa := &relevant[i]
		p := pa.a.Participant
		if pa.synthetic || p == nil || !IsParentSideRole(p.Role) || IsStepRole(p.Role) || pa.a.Subject.Event == "" {
			continue
		}
		if c := byParent[p.Person]; c != nil {
			if status := AssertionProofStatus(pa.a); proofStatusRank(status) > proofStatusRank(c.extra) {
				c.extra = status
			}
		}
	}

	return order
}

// candidateSet collects one question's candidates by key, in first-seen order.
type candidateSet struct {
	byKey map[string]*proofCandidate
	order []*proofCandidate
	role  string
}

func newCandidateSet(role string) *candidateSet {
	return &candidateSet{byKey: map[string]*proofCandidate{}, role: role}
}

func (s *candidateSet) get(key, personID string) *proofCandidate {
	c := s.byKey[key]
	if c == nil {
		c = &proofCandidate{key: key, personID: personID, role: s.role}
		s.byKey[key] = c
		s.order = append(s.order, c)
	}

	return c
}

// corroborate records a claim that supports the candidate without deciding it:
// it can raise the status, never lower it, and a disproven one says nothing.
func (c *proofCandidate) corroborate(a *Assertion) {
	if status := AssertionProofStatus(a); status != ProofStatusDisproven && proofStatusRank(status) > proofStatusRank(c.extra) {
		c.extra = status
	}
}

// marriageAnswerCandidates gathers every possible spouse: the other partners of
// the person's marriage and partner relationships, the other principals of the
// person's marriage events, and the opposite principals of a marriage the
// person is only claimed (or disclaimed) to have taken part in.
func marriageAnswerCandidates(personID string, archive *GLXFile, standings *RelationshipStandingIndex) []*proofCandidate {
	set := newCandidateSet(ParticipantRoleSpouse)
	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil || (rel.Type != RelationshipTypeMarriage && rel.Type != RelationshipTypePartner) || !hasParticipant(personID, rel.Participants) {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == "" || p.Person == personID {
				continue
			}
			c := set.get(p.Person, p.Person)
			if c.relationship == "" {
				c.relationship = relID
			}
			c.add(standings.LinkAssertions(relID, personID, p.Person)...)
		}
	}
	eventAssertions := assertionsByEvent(archive)
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || !isMarriageEventType(event.Type) {
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

// addMarriageEventSpouses weighs a marriage event the person is recorded in.
// A participation or existence claim decides the marriage; a dated or placed
// claim corroborates it, and a disproven date says nothing against the
// marriage itself.
func addMarriageEventSpouses(set *candidateSet, eventID string, event *Event, personID string, archive *GLXFile, assertions []*Assertion) {
	spouses := opposingPrincipals(event.Participants, personID, "")
	if len(spouses) == 0 {
		// A marriage event naming only the person corroborates the spouse of
		// the relationship it starts, or the only spouse.
		spouses = eventSpouses(eventID, archive, set.order)
	}
	for _, spouse := range spouses {
		c := set.get(spouse, spouse)
		if c.event == "" {
			c.event = eventID
		}
		for _, a := range assertions {
			if a.Participant != nil && a.Participant.Person != personID && a.Participant.Person != spouse {
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

// addMarriageEventClaims weighs claims that the person took part in a marriage
// event that does not list them (or, disproven, that they did not).
func addMarriageEventClaims(set *candidateSet, eventID string, event *Event, personID string, assertions []*Assertion) {
	for _, a := range assertions {
		if a.Participant == nil || a.Participant.Person != personID {
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

// eventSpouses returns the spouses a marriage event naming only the person
// belongs to: the spouse of a relationship whose start event it is, or, failing
// that, the person's only spouse from relationships.
func eventSpouses(eventID string, archive *GLXFile, candidates []*proofCandidate) []string {
	var linked, all []string
	for _, c := range candidates {
		if c.personID == "" {
			continue
		}
		all = append(all, c.personID)
		if rel := archive.Relationships[c.relationship]; rel != nil && rel.StartEvent == eventID {
			linked = append(linked, c.personID)
		}
	}
	if len(linked) > 0 {
		return linked
	}
	if len(all) == 1 {
		return all
	}

	return nil
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
// every non-step parent_child link where the person is a parent, and the
// principal of every birth or baptism where the person is recorded as parent.
func childrenAnswerCandidates(personID string, archive *GLXFile, links []ParentChildLink, standings *RelationshipStandingIndex) []*proofCandidate {
	set := newCandidateSet(ParticipantRoleChild)
	for _, link := range links {
		if link.ParentID != personID || link.Step {
			continue
		}
		c := set.get(link.ChildID, link.ChildID)
		if c.relationship == "" {
			c.relationship = link.RelationshipID
		}
		c.add(standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID)...)
	}
	eventAssertions := assertionsByEvent(archive)
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || !isBirthEventType(event.Type) || !eventHasParent(event, personID) {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == "" || p.Person == personID || !isVitalPrincipal(event.Type, p.Role) {
				continue
			}
			c := set.get(p.Person, p.Person)
			if c.event == "" {
				c.event = eventID
			}
			addParentClaims(c, personID, eventAssertions[eventID])
		}
	}

	return set.order
}

// addParentClaims weighs the claims that the person is the parent named in a
// child's birth or baptism. They corroborate a recorded parent-child link; with
// no link, a disproven claim rules the child out.
func addParentClaims(c *proofCandidate, personID string, assertions []*Assertion) {
	for _, a := range assertions {
		if a.Participant == nil || a.Participant.Person != personID || !IsParentSideRole(a.Participant.Role) {
			continue
		}
		if AssertionIsDisproven(a) && c.relationship == "" {
			c.add(a)
		} else {
			c.corroborate(a)
		}
	}
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

// propertyAnswerCandidates groups the assertions of one person property (for
// example occupation) by value. A value recorded on the person or on one of
// their event participations without an assertion is an unsupported candidate.
func propertyAnswerCandidates(personID, property string, archive *GLXFile, relevant []proofAssertion) []*proofCandidate {
	byValue := map[string]*proofCandidate{}
	var order []*proofCandidate
	get := func(value string) *proofCandidate {
		key := normalizeProofValue(value)
		c := byValue[key]
		if c == nil {
			c = &proofCandidate{key: key, value: strings.TrimSpace(value), role: property}
			byValue[key] = c
			order = append(order, c)
		}

		return c
	}
	for i := range relevant {
		pa := &relevant[i]
		if pa.synthetic || pa.a.Value == "" {
			continue
		}
		c := get(pa.a.Value)
		c.add(pa.a)
		if pa.a.Date != "" && !strings.Contains(c.date, string(pa.a.Date)) {
			if c.date != "" {
				c.date += "; "
			}
			c.date += string(pa.a.Date)
		}
	}
	if person := archive.Persons[personID]; person != nil {
		for _, value := range propertyValues(person.Properties[property]) {
			get(value)
		}
	}
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil {
			continue
		}
		for _, p := range event.Participants {
			if p.Person != personID {
				continue
			}
			for _, value := range propertyValues(p.Properties[property]) {
				c := get(value)
				if c.event == "" {
					c.event = eventID
				}
			}
		}
	}

	return order
}

// propertyValues extracts the string values of a simple or temporal property.
func propertyValues(raw any) []string {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{v}
		}
	case []any:
		var out []string
		for _, item := range v {
			out = append(out, propertyValues(item)...)
		}

		return out
	case map[string]any:
		return propertyValues(v["value"])
	}

	return nil
}

// sharedPropertyHolders names the other people recorded with the same value of
// a property (two men holding one office), with the dates recorded for them.
func sharedPropertyHolders(personID, property, value string, archive *GLXFile) []string {
	key := normalizeProofValue(value)
	dates := map[string][]string{}
	var ids []string
	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil {
			continue
		}
		other := a.Subject.Person
		if other == "" || other == personID || a.Property != property || AssertionIsDisproven(a) || normalizeProofValue(a.Value) != key {
			continue
		}
		if _, seen := dates[other]; !seen {
			ids = append(ids, other)
			dates[other] = nil
		}
		if a.Date != "" {
			dates[other] = append(dates[other], string(a.Date))
		}
	}
	for _, id := range sortedKeys(archive.Persons) {
		if id == personID {
			continue
		}
		for _, v := range propertyValues(archive.Persons[id].Properties[property]) {
			if normalizeProofValue(v) == key {
				if _, seen := dates[id]; !seen {
					ids = append(ids, id)
					dates[id] = nil
				}
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
			if a.Date != "" {
				a.Label = a.Value + " (" + a.Date + ")"
			}
		}
	}
	resolved.answerText = candidateAnswerText(prefix, resolved.answers)
	resolved.found = len(resolved.answers) > 0

	return resolved
}

// identityOutcome turns resolved same-person links into identity statements:
// an accepted link names the same person, a disproven one a namesake told
// apart, and an open one is a conflict to resolve.
func identityOutcome(personID string, archive *GLXFile, resolved *proofCandidates) proofCandidates {
	out := proofCandidates{used: true, rejected: resolved.rejected, alternatives: resolved.alternatives, conflicts: resolved.conflicts, level: -1}
	var same []string
	for i := range resolved.answers {
		a := resolved.answers[i]
		if a.Status == ProofStatusProven || a.Status == ProofStatusProbable {
			a.Role = "same_person"
			out.answers = append(out.answers, a)
			same = append(same, a.Label)

			continue
		}
		out.conflicts = append(out.conflicts, ConflictGroup{
			Subject: a.Relationship, Property: RelationshipTypePossiblySamePerson,
			Verdict: VerdictDisputed, Definite: true,
			Values: []ConflictValue{{Value: personName(archive, personID) + " = " + a.Label, Status: a.Status}},
		})
	}
	if len(same) > 0 {
		out.notes = append(out.notes, "Same person as "+strings.Join(same, ", ")+".")
	}
	var apart []string
	for i := range out.alternatives {
		alt := &out.alternatives[i]
		alt.Role = "distinguished_from"
		if alt.Status == ProofStatusDisproven {
			apart = append(apart, alt.Name)
		}
	}
	if len(apart) > 0 {
		out.notes = append(out.notes, "Distinguished from "+strings.Join(apart, ", ")+".")
	}

	return out
}

// alternativeGapLabels returns the coverage labels that name an excluded
// spouse: no record is wanted for a marriage the evidence rules out.
func alternativeGapLabels(alternatives []ProofExcludedAlternative, archive *GLXFile) map[string]bool {
	labels := map[string]bool{}
	for i := range alternatives {
		if id := alternatives[i].PersonID; id != "" {
			labels["Marriage record — "+spouseDisplayName(id, archive)] = true
		}
	}

	return labels
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
