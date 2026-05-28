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
	"maps"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// DuplicateSignal describes one scoring component for a duplicate pair.
//
// A signal with Weight > 0 is a weighted-sum contributor. Its Weight*Score
// participates in the pair total only when HasData is true; when HasData is
// false (e.g. the dimension is absent on one or both persons), the dimension
// drops out of both numerator and denominator so missing data does not
// artificially deflate the pair's score (#716). A weighted-sum Score of 0
// with HasData true is a real "compared and no match" signal (e.g. birth
// years differ by >2) and is kept in the denominator.
//
// A signal with Weight == 0 is a multiplicative gate: it does not contribute
// to the sum, but its Score (in [0, 1]) multiplies the renormalized pair
// score and is only emitted when it actually fires (Score < 1). Gates ignore
// HasData.
//
// Consumers reconstructing a pair score from signals must handle both shapes.
type DuplicateSignal struct {
	Name    string  `json:"name"`
	Weight  float64 `json:"weight"`
	Score   float64 `json:"score"`
	Detail  string  `json:"detail"`
	HasData bool    `json:"has_data"`
}

// DuplicatePair describes a potential duplicate person pair with a similarity score.
type DuplicatePair struct {
	PersonA string            `json:"person_a"`
	PersonB string            `json:"person_b"`
	Score   float64           `json:"score"`
	Signals []DuplicateSignal `json:"signals"`
}

// DuplicateResult holds the complete duplicate detection output.
type DuplicateResult struct {
	Pairs     []DuplicatePair `json:"pairs"`
	Threshold float64         `json:"threshold"`
}

// DuplicateOptions configures duplicate detection behavior.
type DuplicateOptions struct {
	Threshold    float64
	PersonFilter string
}

// Signal weights for the scoring model.
const (
	weightName         = 0.30
	weightBirthYear    = 0.20
	weightBirthPlace   = 0.15
	weightDeathYear    = 0.10
	weightDeathPlace   = 0.10
	weightRelationship = 0.10
	weightEvents       = 0.05
)

// Detail strings used when a scoring dimension reports HasData=false.
const noDataDetail = "no data"

// Year-similarity scores for the two graduated within-N-years buckets above
// "exact match" (1.0). Promoted to named constants so the mnd linter doesn't
// flag the literals when they appear inside multi-value returns.
const (
	yearSimWithin1Year  = 0.75
	yearSimWithin2Years = 0.5
)

// Age-plausibility bounds. A person whose first documented year as role=parent
// is X must have been born somewhere in [X-maxParentAge, X-minParentAge].
const (
	minParentAge = 15
	maxParentAge = 100
)

// duplicateIndex caches lookup maps built from the archive.
type duplicateIndex struct {
	personEvents          map[string][]string        // person ID → event IDs
	personRelPeers        map[string]map[string]bool // person ID → set of related person IDs
	relatedPairs          map[[2]string]bool         // sorted person ID pairs that share a relationship
	personBirthEvent      map[string]*Event          // person ID → their birth event (lowest-ID principal role)
	personDeathEvent      map[string]*Event          // person ID → their death event (lowest-ID principal role)
	personBirthYear       map[string]int             // person ID → year extracted from their birth event (0 if unknown)
	personFirstParentYear map[string]int             // person ID → earliest year they appear as role=parent (0 if never)
}

// FindCrossArchiveDuplicates detects potential duplicate persons between two
// separate archives. Only cross-archive pairs are returned (one person from
// dest, one from src). The archives are not modified.
func FindCrossArchiveDuplicates(dest, src *GLXFile, opts DuplicateOptions) (*DuplicateResult, error) {
	if opts.Threshold < 0.0 || opts.Threshold > 1.0 {
		return nil, fmt.Errorf("%w: %f", ErrInvalidThreshold, opts.Threshold)
	}

	if dest == nil || src == nil || len(dest.Persons) == 0 || len(src.Persons) == 0 {
		return &DuplicateResult{Threshold: opts.Threshold, Pairs: []DuplicatePair{}}, nil
	}

	// Build a combined read-only view for scoring (scorePair needs to look up
	// events/places/relationships from either archive).
	combined := buildCombinedView(dest, src)
	idx := buildDuplicateIndex(combined)

	// Generate only cross-archive pairs
	pairs := generateCrossArchivePairs(dest, src, idx, opts.PersonFilter)

	var results []DuplicatePair
	for _, pair := range pairs {
		personA := combined.Persons[pair[0]]
		personB := combined.Persons[pair[1]]
		if personA == nil || personB == nil {
			continue
		}

		score, signals := scorePair(pair[0], pair[1], personA, personB, combined, idx)
		if score >= opts.Threshold {
			results = append(results, DuplicatePair{
				PersonA: pair[0],
				PersonB: pair[1],
				Score:   math.Round(score*100) / 100, //nolint:mnd // round to 2 decimal places
				Signals: signals,
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return &DuplicateResult{
		Pairs:     results,
		Threshold: opts.Threshold,
	}, nil
}

// buildCombinedView creates a read-only GLXFile containing entities from both
// archives. On ID collision, dest wins (consistent with merge semantics).
// The original archives are not modified.
func buildCombinedView(dest, src *GLXFile) *GLXFile {
	combined := &GLXFile{
		Persons:       make(map[string]*Person, len(dest.Persons)+len(src.Persons)),
		Events:        make(map[string]*Event, len(dest.Events)+len(src.Events)),
		Relationships: make(map[string]*Relationship, len(dest.Relationships)+len(src.Relationships)),
		Places:        make(map[string]*Place, len(dest.Places)+len(src.Places)),
	}

	// Add src first, then dest (dest overwrites on collision)
	maps.Copy(combined.Persons, src.Persons)
	maps.Copy(combined.Persons, dest.Persons)

	maps.Copy(combined.Events, src.Events)
	maps.Copy(combined.Events, dest.Events)

	maps.Copy(combined.Relationships, src.Relationships)
	maps.Copy(combined.Relationships, dest.Relationships)

	maps.Copy(combined.Places, src.Places)
	maps.Copy(combined.Places, dest.Places)

	return combined
}

// crossArchivePairThreshold is the combined person count above which surname
// blocking is used to reduce the O(N*M) Cartesian product.
const crossArchivePairThreshold = 200

// generateCrossArchivePairs produces person ID pairs where one is from dest and
// the other from src. Skips pairs that share a relationship in the combined index.
// Uses surname blocking for large archives to avoid O(N*M) explosion.
func generateCrossArchivePairs(dest, src *GLXFile, idx *duplicateIndex, personFilter string) [][2]string {
	if personFilter != "" {
		return generateFilteredCrossArchivePairs(dest, src, idx, personFilter)
	}

	if len(dest.Persons)+len(src.Persons) < crossArchivePairThreshold {
		return generateAllCrossArchivePairs(dest, src, idx)
	}

	return generateBlockedCrossArchivePairs(dest, src, idx)
}

// isCrossArchivePairEligible checks if a dest+src pair should be included.
func isCrossArchivePairEligible(destID, srcID string, idx *duplicateIndex) bool {
	if srcID == destID {
		return false
	}

	a, b := destID, srcID
	if a > b {
		a, b = b, a
	}

	return !idx.relatedPairs[[2]string{a, b}]
}

// generateFilteredCrossArchivePairs returns pairs involving a specific person.
func generateFilteredCrossArchivePairs(dest, src *GLXFile, idx *duplicateIndex, personFilter string) [][2]string {
	var pairs [][2]string

	if _, ok := src.Persons[personFilter]; ok {
		for id := range dest.Persons {
			if isCrossArchivePairEligible(id, personFilter, idx) {
				pairs = append(pairs, [2]string{id, personFilter})
			}
		}
	}

	if _, ok := dest.Persons[personFilter]; ok {
		for id := range src.Persons {
			if isCrossArchivePairEligible(personFilter, id, idx) {
				pairs = append(pairs, [2]string{personFilter, id})
			}
		}
	}

	return sortPairs(pairs)
}

// generateAllCrossArchivePairs returns all dest×src pairs for small archives.
func generateAllCrossArchivePairs(dest, src *GLXFile, idx *duplicateIndex) [][2]string {
	var pairs [][2]string

	for destID := range dest.Persons {
		for srcID := range src.Persons {
			if isCrossArchivePairEligible(destID, srcID, idx) {
				pairs = append(pairs, [2]string{destID, srcID})
			}
		}
	}

	return sortPairs(pairs)
}

// generateBlockedCrossArchivePairs uses surname blocking for large archives.
func generateBlockedCrossArchivePairs(dest, src *GLXFile, idx *duplicateIndex) [][2]string {
	destBlocks := buildSurnameBlocks(dest)
	srcBlocks := buildSurnameBlocks(src)

	seen := make(map[[2]string]bool)
	var pairs [][2]string

	for surname, destBlock := range destBlocks {
		srcBlock, ok := srcBlocks[surname]
		if !ok {
			continue
		}

		for _, destID := range destBlock {
			for _, srcID := range srcBlock {
				a, b := destID, srcID
				if a > b {
					a, b = b, a
				}

				pair := [2]string{a, b}
				if seen[pair] {
					continue
				}

				seen[pair] = true

				if isCrossArchivePairEligible(destID, srcID, idx) {
					pairs = append(pairs, [2]string{destID, srcID})
				}
			}
		}
	}

	return sortPairs(pairs)
}

// buildSurnameBlocks groups person IDs by normalized surname.
func buildSurnameBlocks(archive *GLXFile) map[string][]string {
	blocks := make(map[string][]string)
	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		_, surname := ExtractNameFields(person.Properties[PersonPropertyName])
		if surname == "" {
			_, surname = splitFullName(PersonDisplayName(person))
		}
		key := strings.ToLower(strings.TrimSpace(surname))
		if key == "" {
			key = "_nosurname"
		}
		blocks[key] = append(blocks[key], id)
	}

	return blocks
}

// sortPairs sorts pairs for deterministic output.
func sortPairs(pairs [][2]string) [][2]string {
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}

		return pairs[i][1] < pairs[j][1]
	})

	return pairs
}

// FindDuplicates scans an archive for potential duplicate persons.
// Threshold must be between 0.0 and 1.0 inclusive.
func FindDuplicates(archive *GLXFile, opts DuplicateOptions) (*DuplicateResult, error) {
	if opts.Threshold < 0.0 || opts.Threshold > 1.0 {
		return nil, fmt.Errorf("%w: %f", ErrInvalidThreshold, opts.Threshold)
	}

	if archive == nil || len(archive.Persons) < 2 {
		return &DuplicateResult{Threshold: opts.Threshold, Pairs: []DuplicatePair{}}, nil
	}

	idx := buildDuplicateIndex(archive)
	pairs := generateCandidatePairs(archive, idx, opts.PersonFilter)

	results := []DuplicatePair{}
	for _, pair := range pairs {
		personA := archive.Persons[pair[0]]
		personB := archive.Persons[pair[1]]
		if personA == nil || personB == nil {
			continue
		}

		score, signals := scorePair(pair[0], pair[1], personA, personB, archive, idx)
		if score >= opts.Threshold {
			results = append(results, DuplicatePair{
				PersonA: pair[0],
				PersonB: pair[1],
				Score:   math.Round(score*100) / 100,
				Signals: signals,
			})
		}
	}

	// Sort by score descending
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return &DuplicateResult{
		Pairs:     results,
		Threshold: opts.Threshold,
	}, nil
}

// buildDuplicateIndex creates lookup maps from the archive.
func buildDuplicateIndex(archive *GLXFile) *duplicateIndex {
	idx := &duplicateIndex{
		personEvents:          make(map[string][]string),
		personRelPeers:        make(map[string]map[string]bool),
		relatedPairs:          make(map[[2]string]bool),
		personBirthEvent:      make(map[string]*Event),
		personDeathEvent:      make(map[string]*Event),
		personBirthYear:       make(map[string]int),
		personFirstParentYear: make(map[string]int),
	}

	// Index events by participant; pre-bind birth/death events per person so
	// scorePair doesn't re-sort and scan all events for every candidate pair.
	// Iterate event IDs in sorted order and write only if absent, matching
	// FindPersonEvent's "lowest-ID principal event wins" determinism.
	eventIDs := make([]string, 0, len(archive.Events))
	for id := range archive.Events {
		eventIDs = append(eventIDs, id)
	}
	sort.Strings(eventIDs)

	for _, eventID := range eventIDs {
		event := archive.Events[eventID]
		if event == nil {
			continue
		}
		eventYear := ExtractFirstYear(string(event.Date))
		for _, p := range event.Participants {
			if p.Person == "" {
				continue
			}
			idx.personEvents[p.Person] = append(idx.personEvents[p.Person], eventID)
			if p.Role == ParticipantRoleParent && eventYear > 0 {
				recordParentYear(idx, p.Person, eventYear)
			}
			if !isSubjectRole(p.Role) {
				continue
			}
			switch event.Type {
			case EventTypeBirth:
				if _, seen := idx.personBirthEvent[p.Person]; !seen {
					idx.personBirthEvent[p.Person] = event
					idx.personBirthYear[p.Person] = eventYear
				}
			case EventTypeDeath:
				if _, seen := idx.personDeathEvent[p.Person]; !seen {
					idx.personDeathEvent[p.Person] = event
				}
			}
		}
	}

	// Index relationships
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		var (
			personIDs []string
			parentIDs []string
			childYear int
		)
		isParentChild := isParentChildRelType(rel.Type)
		for _, p := range rel.Participants {
			if p.Person == "" {
				continue
			}
			personIDs = append(personIDs, p.Person)
			if !isParentChild {
				continue
			}
			switch p.Role {
			case ParticipantRoleParent:
				parentIDs = append(parentIDs, p.Person)
			case ParticipantRoleChild:
				if y := idx.personBirthYear[p.Person]; y > 0 && (childYear == 0 || y < childYear) {
					childYear = y
				}
			}
		}
		if childYear > 0 {
			for _, parentID := range parentIDs {
				recordParentYear(idx, parentID, childYear)
			}
		}
		// Record all pairwise relationships
		for i := 0; i < len(personIDs); i++ {
			for j := i + 1; j < len(personIDs); j++ {
				a, b := personIDs[i], personIDs[j]
				if a > b {
					a, b = b, a
				}
				idx.relatedPairs[[2]string{a, b}] = true

				if idx.personRelPeers[personIDs[i]] == nil {
					idx.personRelPeers[personIDs[i]] = make(map[string]bool)
				}
				idx.personRelPeers[personIDs[i]][personIDs[j]] = true

				if idx.personRelPeers[personIDs[j]] == nil {
					idx.personRelPeers[personIDs[j]] = make(map[string]bool)
				}
				idx.personRelPeers[personIDs[j]][personIDs[i]] = true
			}
		}
	}

	return idx
}

func recordParentYear(idx *duplicateIndex, personID string, year int) {
	if existing, ok := idx.personFirstParentYear[personID]; !ok || year < existing {
		idx.personFirstParentYear[personID] = year
	}
}

// generateCandidatePairs produces person ID pairs to compare.
// Skips pairs that already share a direct relationship.
func generateCandidatePairs(archive *GLXFile, idx *duplicateIndex, personFilter string) [][2]string {
	ids := make([]string, 0, len(archive.Persons))
	for id := range archive.Persons {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var pairs [][2]string

	if personFilter != "" {
		// Only pairs involving the filtered person
		for _, id := range ids {
			if id == personFilter {
				continue
			}
			a, b := personFilter, id
			if a > b {
				a, b = b, a
			}
			if idx.relatedPairs[[2]string{a, b}] {
				continue
			}
			pairs = append(pairs, [2]string{a, b})
		}

		return pairs
	}

	// For small archives (< 200 persons), do all pairs
	// For larger archives, block by surname to reduce comparisons
	if len(ids) < 200 {
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				if idx.relatedPairs[[2]string{ids[i], ids[j]}] {
					continue
				}
				pairs = append(pairs, [2]string{ids[i], ids[j]})
			}
		}

		return pairs
	}

	// Surname blocking for large archives
	blocks := make(map[string][]string) // normalized surname → person IDs
	for _, id := range ids {
		person := archive.Persons[id]
		if person == nil {
			continue
		}
		_, surname := ExtractNameFields(person.Properties[PersonPropertyName])
		if surname == "" {
			_, surname = splitFullName(PersonDisplayName(person))
		}
		key := strings.ToLower(strings.TrimSpace(surname))
		if key == "" {
			key = "_nosurname"
		}
		blocks[key] = append(blocks[key], id)
	}

	seen := make(map[[2]string]bool)
	for _, block := range blocks {
		for i := range block {
			for j := i + 1; j < len(block); j++ {
				a, b := block[i], block[j]
				if a > b {
					a, b = b, a
				}
				pair := [2]string{a, b}
				if seen[pair] || idx.relatedPairs[pair] {
					continue
				}
				seen[pair] = true
				pairs = append(pairs, pair)
			}
		}
	}

	// Sort for deterministic output
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}

		return pairs[i][1] < pairs[j][1]
	})

	return pairs
}

// scorePair computes the similarity score between two persons.
//
// The pair total is a renormalized weighted average across the dimensions
// that actually have data on both sides. Dimensions where HasData is false
// drop out of both the numerator and the denominator, so a pair with only a
// name match (and birth/death/place/etc. simply unrecorded) scores by the
// fraction of name-similarity it shares, not by name-weight alone (#716).
// A weighted-sum dimension that compared and disagreed (e.g. years differ
// by >2) keeps HasData=true and contributes 0 to the numerator while
// remaining in the denominator, which still penalizes real disagreement.
// The age-plausibility gate, when emitted, multiplies the renormalized score.
func scorePair(idA, idB string, personA, personB *Person, archive *GLXFile, idx *duplicateIndex) (float64, []DuplicateSignal) {
	var signals []DuplicateSignal

	// Name similarity
	nameScore, nameDetail, nameHas := scoreNameSimilarity(personA, personB)
	signals = append(signals, DuplicateSignal{"Name similarity", weightName, nameScore, nameDetail, nameHas})

	// Birth year and place
	birthA := idx.personBirthEvent[idA]
	birthB := idx.personBirthEvent[idB]
	byScore, byDetail, byHas := scoreEventYearSimilarity(birthA, birthB)
	signals = append(signals, DuplicateSignal{"Birth year", weightBirthYear, byScore, byDetail, byHas})

	bpScore, bpDetail, bpHas := scoreEventPlaceSimilarity(birthA, birthB, archive)
	signals = append(signals, DuplicateSignal{"Birth place", weightBirthPlace, bpScore, bpDetail, bpHas})

	// Death year and place
	deathA := idx.personDeathEvent[idA]
	deathB := idx.personDeathEvent[idB]
	dyScore, dyDetail, dyHas := scoreEventYearSimilarity(deathA, deathB)
	signals = append(signals, DuplicateSignal{"Death year", weightDeathYear, dyScore, dyDetail, dyHas})

	dpScore, dpDetail, dpHas := scoreEventPlaceSimilarity(deathA, deathB, archive)
	signals = append(signals, DuplicateSignal{"Death place", weightDeathPlace, dpScore, dpDetail, dpHas})

	// Shared relationships
	relScore, relDetail, relHas := scoreSharedRelationships(idA, idB, idx)
	signals = append(signals, DuplicateSignal{"Shared relationships", weightRelationship, relScore, relDetail, relHas})

	// Shared events
	evScore, evDetail, evHas := scoreSharedEvents(idA, idB, idx)
	signals = append(signals, DuplicateSignal{"Shared events", weightEvents, evScore, evDetail, evHas})

	// Renormalize across dimensions that have data on both sides.
	var weightedSum, effectiveWeight float64
	for _, sig := range signals {
		if sig.HasData {
			weightedSum += sig.Weight * sig.Score
			effectiveWeight += sig.Weight
		}
	}
	var totalScore float64
	if effectiveWeight > 0 {
		totalScore = weightedSum / effectiveWeight
	}

	// Age plausibility is a multiplicative gate, not a weighted-sum contributor:
	// it can only zero an otherwise-passing pair, never lift one. Emit the signal
	// only when it fires, so plausible pairs keep a clean breakdown.
	if ageScore, ageDetail := scoreAgePlausibility(idA, idB, idx); ageScore < 1.0 {
		signals = append(signals, DuplicateSignal{"Age plausibility", 0, ageScore, ageDetail, true})
		totalScore *= ageScore
	}

	return totalScore, signals
}

// scoreAgePlausibility returns 0.0 when the pair's dated parent-role evidence
// makes it physically impossible for A and B to be the same person, and 1.0
// otherwise (including when there's not enough dated evidence to rule out).
//
// Rule: if X appears as role=parent in year P, X must have been born somewhere
// in [P-maxParentAge, P-minParentAge]. If the candidate-counterpart Y has a
// known birth year outside that window, A and B cannot be the same person.
// Both directions are checked.
func scoreAgePlausibility(idA, idB string, idx *duplicateIndex) (float64, string) {
	if reason := implausibilityReason(idA, idx.personFirstParentYear[idA], idB, idx.personBirthYear[idB]); reason != "" {
		return 0.0, reason
	}
	if reason := implausibilityReason(idB, idx.personFirstParentYear[idB], idA, idx.personBirthYear[idA]); reason != "" {
		return 0.0, reason
	}

	return 1.0, "plausible"
}

// implausibilityReason returns a non-empty explanation iff the (parent X,
// candidate-birth Y) pair violates the parent-age window.
func implausibilityReason(parentID string, parentYear int, birthID string, birthYr int) string {
	if parentYear <= 0 || birthYr <= 0 {
		return ""
	}
	ageAtParenthood := parentYear - birthYr
	if ageAtParenthood < minParentAge || ageAtParenthood > maxParentAge {
		return fmt.Sprintf("%s parent in %d, %s born %d (would be %d at parenthood)", parentID, parentYear, birthID, birthYr, ageAtParenthood)
	}

	return ""
}

// scoreNameSimilarity compares two persons' names. The third return is true
// iff both persons have a non-empty display name (i.e. the comparison was
// possible); a name-comparison that ran but found no overlap returns
// (0, "no match", true) so the dimension still counts against the pair.
func scoreNameSimilarity(personA, personB *Person) (float64, string, bool) {
	nameA := PersonDisplayName(personA)
	nameB := PersonDisplayName(personB)
	if nameA == "" || nameB == "" {
		return 0, "no name", false
	}

	givenA, surnameA := ExtractNameFields(personA.Properties[PersonPropertyName])
	givenB, surnameB := ExtractNameFields(personB.Properties[PersonPropertyName])

	// If no structured fields, try to split full name
	if givenA == "" && surnameA == "" {
		givenA, surnameA = splitFullName(nameA)
	}
	if givenB == "" && surnameB == "" {
		givenB, surnameB = splitFullName(nameB)
	}

	var score float64
	var parts []string

	// Surname comparison (0.5 of name weight)
	surnameScore := compareSurnames(surnameA, surnameB)
	score += 0.5 * surnameScore
	if surnameScore >= 1.0 {
		parts = append(parts, "surname exact")
	} else if surnameScore > 0 {
		parts = append(parts, "surname similar")
	}

	// Given name comparison (0.5 of name weight)
	givenScore := compareGivenNames(givenA, givenB)
	score += 0.5 * givenScore
	if givenScore >= 1.0 {
		parts = append(parts, "given exact")
	} else if givenScore > 0 {
		parts = append(parts, "given similar")
	}

	detail := strings.Join(parts, ", ")
	if detail == "" {
		detail = "no match"
	}

	return score, detail, true
}

// splitFullName splits a simple "Given Surname" string into parts.
func splitFullName(name string) (given, surname string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}

	return strings.Join(parts[:len(parts)-1], " "), parts[len(parts)-1]
}

// compareSurnames compares two surnames with normalization.
func compareSurnames(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1.0
	}

	return normalizedLevenshtein(a, b)
}

// compareGivenNames compares two given names with nickname/initial handling.
func compareGivenNames(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1.0
	}

	// Check nickname variants
	if areNicknameVariants(a, b) {
		return 0.9
	}

	// Check initial match (e.g., "D" or "D." matches "Daniel")
	if isInitialMatch(a, b) {
		return 0.6
	}

	return normalizedLevenshtein(a, b)
}

// scoreEventYearSimilarity compares years from two events. The third return
// is true iff a year could be extracted from both events; a years-too-far-
// apart result returns (0, "different", true) so the dimension still counts
// against the pair (#716 — distinguishing "no data" from "mismatch").
func scoreEventYearSimilarity(eventA, eventB *Event) (float64, string, bool) {
	yearA, yearB := 0, 0
	if eventA != nil {
		yearA = ExtractFirstYear(string(eventA.Date))
	}
	if eventB != nil {
		yearB = ExtractFirstYear(string(eventB.Date))
	}

	if yearA == 0 || yearB == 0 {
		return 0, noDataDetail, false
	}

	diff := yearA - yearB
	if diff < 0 {
		diff = -diff
	}

	switch {
	case diff == 0:
		return 1.0, "exact match", true
	case diff <= 1:
		return yearSimWithin1Year, "within 1 year", true
	case diff <= 2:
		return yearSimWithin2Years, "within 2 years", true
	default:
		return 0, "different", true
	}
}

// scoreEventPlaceSimilarity compares place references from two events. The
// third return is true iff both events carry a non-empty PlaceID; a
// different-place result returns (0, "different", true) so the dimension
// still counts against the pair (#716).
func scoreEventPlaceSimilarity(eventA, eventB *Event, archive *GLXFile) (float64, string, bool) {
	placeA, placeB := "", ""
	if eventA != nil {
		placeA = eventA.PlaceID
	}
	if eventB != nil {
		placeB = eventB.PlaceID
	}

	if placeA == "" || placeB == "" {
		return 0, noDataDetail, false
	}

	if placeA == placeB {
		placeName := placeA
		if archive != nil && archive.Places != nil {
			if p, ok := archive.Places[placeA]; ok && p != nil {
				placeName = p.Name
			}
		}

		return 1.0, placeName, true
	}

	return 0, "different", true
}

// scoreSharedRelationships scores the overlap in related persons. The third
// return is true iff both persons have at least one related peer; an empty-
// overlap result with peers on both sides returns (0, "no overlap", true) so
// the dimension still counts against the pair (a real "compared and found no
// shared family" signal, vs. "we haven't recorded their family yet").
func scoreSharedRelationships(idA, idB string, idx *duplicateIndex) (float64, string, bool) {
	peersA := idx.personRelPeers[idA]
	peersB := idx.personRelPeers[idB]
	if len(peersA) == 0 || len(peersB) == 0 {
		return 0, noDataDetail, false
	}

	var common int
	for peer := range peersA {
		if peersB[peer] {
			common++
		}
	}

	if common == 0 {
		return 0, "no overlap", true
	}

	maxPeers := max(len(peersB), len(peersA))

	score := float64(common) / float64(maxPeers)

	return score, pluralize(common, "shared"), true
}

// scoreSharedEvents scores the overlap in event participation. The third
// return is true iff both persons participate in at least one event; an
// empty-overlap result with events on both sides returns (0, "no overlap",
// true) so the dimension still counts against the pair.
func scoreSharedEvents(idA, idB string, idx *duplicateIndex) (float64, string, bool) {
	eventsA := idx.personEvents[idA]
	eventsB := idx.personEvents[idB]
	if len(eventsA) == 0 || len(eventsB) == 0 {
		return 0, noDataDetail, false
	}

	setB := make(map[string]bool, len(eventsB))
	for _, e := range eventsB {
		setB[e] = true
	}

	var common int
	for _, e := range eventsA {
		if setB[e] {
			common++
		}
	}

	if common == 0 {
		return 0, "no overlap", true
	}

	maxEvents := max(len(eventsB), len(eventsA))

	score := float64(common) / float64(maxEvents)

	return score, pluralize(common, "shared"), true
}

func pluralize(count int, label string) string {
	if count == 1 {
		return "1 " + label
	}

	return fmt.Sprintf("%d %s", count, label)
}

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	runesA := []rune(a)
	runesB := []rune(b)
	lenA := len(runesA)
	lenB := len(runesB)

	if lenA == 0 {
		return lenB
	}
	if lenB == 0 {
		return lenA
	}

	// Single-row DP
	prev := make([]int, lenB+1)
	for j := 0; j <= lenB; j++ {
		prev[j] = j
	}

	for i := 1; i <= lenA; i++ {
		curr := make([]int, lenB+1)
		curr[0] = i
		for j := 1; j <= lenB; j++ {
			cost := 1
			if runesA[i-1] == runesB[j-1] {
				cost = 0
			}
			ins := prev[j] + 1
			del := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min(ins, min(del, sub))
		}
		prev = curr
	}

	return prev[lenB]
}

// normalizedLevenshtein returns a similarity score between 0.0 and 1.0.
func normalizedLevenshtein(a, b string) float64 {
	if a == b {
		return 1.0
	}
	maxLen := len([]rune(a))
	if lb := len([]rune(b)); lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(levenshteinDistance(a, b))/float64(maxLen)
}

// nicknameTable maps common abbreviations and nicknames to a canonical form.
var nicknameTable = map[string]string{
	// William variants
	"william": "william", "wm": "william", "will": "william", "bill": "william", "billy": "william", "willie": "william",
	// James variants
	"james": "james", "jas": "james", "jim": "james", "jimmy": "james",
	// Charles variants
	"charles": "charles", "chas": "charles", "charlie": "charles", "charley": "charles",
	// Thomas variants
	"thomas": "thomas", "thos": "thomas", "tom": "thomas", "tommy": "thomas",
	// George variants
	"george": "george", "geo": "george",
	// Robert variants
	"robert": "robert", "robt": "robert", "rob": "robert", "bob": "robert", "bobby": "robert",
	// Samuel variants
	"samuel": "samuel", "saml": "samuel", "sam": "samuel",
	// Daniel variants
	"daniel": "daniel", "dan": "daniel", "danl": "daniel",
	// John variants
	"john": "john", "jno": "john", "johnny": "john", "jon": "john",
	// Joseph variants
	"joseph": "joseph", "jos": "joseph", "joe": "joseph",
	// Benjamin variants
	"benjamin": "benjamin", "benj": "benjamin", "ben": "benjamin",
	// Richard variants
	"richard": "richard", "richd": "richard", "dick": "richard",
	// Henry variants (Harry was historically a nickname for Henry in English records)
	"henry": "henry", "harry": "henry",
	// Edward variants
	"edward": "edward", "edw": "edward", "ed": "edward", "edwd": "edward", "ned": "edward", "ted": "edward",
	// Elizabeth variants
	"elizabeth": "elizabeth", "eliz": "elizabeth", "eliza": "elizabeth", "beth": "elizabeth",
	"betsy": "elizabeth", "betty": "elizabeth", "liz": "elizabeth", "lizzie": "elizabeth",
	// Mary variants
	"mary": "mary", "polly": "mary",
	// Margaret variants
	"margaret": "margaret", "margt": "margaret", "maggie": "margaret", "peggy": "margaret", "marge": "margaret",
	// Catherine variants
	"catherine": "catherine", "kate": "catherine", "katie": "catherine", "kitty": "catherine",
	"katharine": "catherine", "kathryn": "catherine",
	// Sarah variants
	"sarah": "sarah", "sally": "sarah",
	// Ann/Anna variants
	"ann": "ann", "anna": "ann", "annie": "ann", "nancy": "ann",
	// Rebecca variants
	"rebecca": "rebecca", "becky": "rebecca",
	// Jonathan variants (Nathan is a distinct name, not a nickname for Jonathan)
	"jonathan": "jonathan",
	// Alexander variants
	"alexander": "alexander", "alex": "alexander",
	// Abraham variants
	"abraham": "abraham", "abram": "abraham", "abe": "abraham",
	// Frederick variants
	"frederick": "frederick", "fred": "frederick", "fredk": "frederick",
	// Theodore variants
	"theodore": "theodore", "theo": "theodore",
}

// areNicknameVariants returns true if both names resolve to the same canonical form.
func areNicknameVariants(a, b string) bool {
	canonA, okA := nicknameTable[a]
	canonB, okB := nicknameTable[b]
	if okA && okB {
		return canonA == canonB
	}

	return false
}

// isInitialMatch returns true if one name is a single-character initial that
// matches the first letter of the other name. Uses rune counting for correct
// Unicode handling (e.g., "É" is one character despite being multi-byte in UTF-8).
func isInitialMatch(a, b string) bool {
	cleanA := strings.TrimSuffix(a, ".")
	cleanB := strings.TrimSuffix(b, ".")

	if utf8.RuneCountInString(cleanA) == 1 && utf8.RuneCountInString(cleanB) > 1 {
		return strings.HasPrefix(cleanB, cleanA)
	}
	if utf8.RuneCountInString(cleanB) == 1 && utf8.RuneCountInString(cleanA) > 1 {
		return strings.HasPrefix(cleanA, cleanB)
	}

	return false
}
