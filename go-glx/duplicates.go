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
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// DuplicateSignal describes one scoring component for a duplicate pair.
type DuplicateSignal struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
	Detail string  `json:"detail"`
}

// DuplicatePair describes a potential duplicate person pair with a similarity score.
type DuplicatePair struct {
	PersonA  string            `json:"person_a"`
	PersonB  string            `json:"person_b"`
	Score    float64           `json:"score"`
	Signals  []DuplicateSignal `json:"signals"`
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

// duplicateIndex caches lookup maps built from the archive.
type duplicateIndex struct {
	personEvents   map[string][]string          // person ID → event IDs
	personRelPeers map[string]map[string]bool   // person ID → set of related person IDs
	relatedPairs   map[[2]string]bool            // sorted person ID pairs that share a relationship
}

// FindDuplicates scans an archive for potential duplicate persons.
// Threshold must be between 0.0 and 1.0 inclusive.
func FindDuplicates(archive *GLXFile, opts DuplicateOptions) (*DuplicateResult, error) {
	if opts.Threshold < 0.0 || opts.Threshold > 1.0 {
		return nil, fmt.Errorf("threshold must be between 0.0 and 1.0, got %f", opts.Threshold)
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
		personEvents:   make(map[string][]string),
		personRelPeers: make(map[string]map[string]bool),
		relatedPairs:   make(map[[2]string]bool),
	}

	// Index events by participant
	for eventID, event := range archive.Events {
		if event == nil {
			continue
		}
		for _, p := range event.Participants {
			if p.Person != "" {
				idx.personEvents[p.Person] = append(idx.personEvents[p.Person], eventID)
			}
		}
	}

	// Index relationships
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		var personIDs []string
		for _, p := range rel.Participants {
			if p.Person != "" {
				personIDs = append(personIDs, p.Person)
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
		for i := 0; i < len(block); i++ {
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
func scorePair(idA, idB string, personA, personB *Person, archive *GLXFile, idx *duplicateIndex) (float64, []DuplicateSignal) {
	var signals []DuplicateSignal
	var totalScore float64

	// Name similarity
	nameScore, nameDetail := scoreNameSimilarity(personA, personB)
	signals = append(signals, DuplicateSignal{"Name similarity", weightName, nameScore, nameDetail})
	totalScore += weightName * nameScore

	propsA := personA.Properties
	propsB := personB.Properties
	if propsA == nil {
		propsA = map[string]any{}
	}
	if propsB == nil {
		propsB = map[string]any{}
	}

	// Birth year
	byScore, byDetail := scoreYearSimilarity(propsA, propsB, PersonPropertyBornOn)
	signals = append(signals, DuplicateSignal{"Birth year", weightBirthYear, byScore, byDetail})
	totalScore += weightBirthYear * byScore

	// Birth place
	bpScore, bpDetail := scorePlaceSimilarity(propsA, propsB, PersonPropertyBornAt, archive)
	signals = append(signals, DuplicateSignal{"Birth place", weightBirthPlace, bpScore, bpDetail})
	totalScore += weightBirthPlace * bpScore

	// Death year
	dyScore, dyDetail := scoreYearSimilarity(propsA, propsB, PersonPropertyDiedOn)
	signals = append(signals, DuplicateSignal{"Death year", weightDeathYear, dyScore, dyDetail})
	totalScore += weightDeathYear * dyScore

	// Death place
	dpScore, dpDetail := scorePlaceSimilarity(propsA, propsB, PersonPropertyDiedAt, archive)
	signals = append(signals, DuplicateSignal{"Death place", weightDeathPlace, dpScore, dpDetail})
	totalScore += weightDeathPlace * dpScore

	// Shared relationships
	relScore, relDetail := scoreSharedRelationships(idA, idB, idx)
	signals = append(signals, DuplicateSignal{"Shared relationships", weightRelationship, relScore, relDetail})
	totalScore += weightRelationship * relScore

	// Shared events
	evScore, evDetail := scoreSharedEvents(idA, idB, idx)
	signals = append(signals, DuplicateSignal{"Shared events", weightEvents, evScore, evDetail})
	totalScore += weightEvents * evScore

	return totalScore, signals
}

// scoreNameSimilarity compares two persons' names.
func scoreNameSimilarity(personA, personB *Person) (float64, string) {
	nameA := PersonDisplayName(personA)
	nameB := PersonDisplayName(personB)
	if nameA == "" || nameB == "" {
		return 0, "no name"
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

	return score, detail
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

// scoreYearSimilarity compares year values from person properties.
func scoreYearSimilarity(propsA, propsB map[string]any, propertyKey string) (float64, string) {
	yearA := ExtractPropertyYear(propsA, propertyKey)
	yearB := ExtractPropertyYear(propsB, propertyKey)

	if yearA == 0 || yearB == 0 {
		return 0, "no data"
	}

	diff := yearA - yearB
	if diff < 0 {
		diff = -diff
	}

	switch {
	case diff == 0:
		return 1.0, "exact match"
	case diff <= 1:
		return 0.75, "within 1 year"
	case diff <= 2:
		return 0.5, "within 2 years"
	default:
		return 0, "different"
	}
}

// scorePlaceSimilarity compares place references from person properties.
func scorePlaceSimilarity(propsA, propsB map[string]any, propertyKey string, archive *GLXFile) (float64, string) {
	placeA := propertyToString(propsA, propertyKey)
	placeB := propertyToString(propsB, propertyKey)

	if placeA == "" || placeB == "" {
		return 0, "no data"
	}

	if placeA == placeB {
		placeName := placeA
		if archive != nil && archive.Places != nil {
			if p, ok := archive.Places[placeA]; ok && p != nil {
				placeName = p.Name
			}
		}
		return 1.0, placeName
	}

	return 0, "different"
}

// propertyToString extracts a string value from a properties map.
func propertyToString(props map[string]any, key string) string {
	raw, ok := props[key]
	if !ok {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}

// scoreSharedRelationships scores the overlap in related persons.
func scoreSharedRelationships(idA, idB string, idx *duplicateIndex) (float64, string) {
	peersA := idx.personRelPeers[idA]
	peersB := idx.personRelPeers[idB]
	if len(peersA) == 0 || len(peersB) == 0 {
		return 0, "no data"
	}

	var common int
	for peer := range peersA {
		if peersB[peer] {
			common++
		}
	}

	if common == 0 {
		return 0, "no overlap"
	}

	maxPeers := len(peersA)
	if len(peersB) > maxPeers {
		maxPeers = len(peersB)
	}

	score := float64(common) / float64(maxPeers)

	return score, pluralize(common, "shared")
}

// scoreSharedEvents scores the overlap in event participation.
func scoreSharedEvents(idA, idB string, idx *duplicateIndex) (float64, string) {
	eventsA := idx.personEvents[idA]
	eventsB := idx.personEvents[idB]
	if len(eventsA) == 0 || len(eventsB) == 0 {
		return 0, "no data"
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
		return 0, "no overlap"
	}

	maxEvents := len(eventsA)
	if len(eventsB) > maxEvents {
		maxEvents = len(eventsB)
	}

	score := float64(common) / float64(maxEvents)

	return score, pluralize(common, "shared")
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
