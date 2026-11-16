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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

var (
	errUnknownJSONEntityType   = errors.New("unknown entity type")
	errEmptyJSONPerson         = errors.New("person must not be empty")
	errNegativeJSONGenerations = errors.New("generations must not be negative")
	errJSONCitationNotFound    = errors.New("citation not found in archive")
)

const (
	jsonAncestors   = "ancestors"
	jsonDescendants = "descendants"
)

type queryJSONEntity struct {
	ID      string `json:"id"`
	Entity  any    `json:"entity"`
	encoded json.RawMessage
}
type queryJSONResult struct {
	EntityType glxlib.EntityType `json:"entity_type"`
	Count      int               `json:"count"`
	Entities   []queryJSONEntity `json:"entities"`
}

// buildQueryJSON returns complete matching entities, not rendered table rows.
func buildQueryJSON(archive *glxlib.GLXFile, entityType glxlib.EntityType, opts *queryOpts) (*queryJSONResult, error) {
	return buildQueryJSONContext(context.Background(), archive, entityType, opts)
}

func buildQueryJSONContext(ctx context.Context, archive *glxlib.GLXFile, entityType glxlib.EntityType, opts *queryOpts) (*queryJSONResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	if !slices.Contains(queryEntityTypes, entityType) {
		return nil, fmt.Errorf("%w: %s", errUnknownJSONEntityType, entityType)
	}
	if err := validateQueryFlags(entityType, opts); err != nil {
		return nil, err
	}
	localOpts := *opts
	localOpts.Name = strings.ToLower(localOpts.Name)
	opts = &localOpts
	result := &queryJSONResult{EntityType: entityType, Entities: []queryJSONEntity{}}
	var births map[string]*glxlib.Event
	if entityType == glxlib.EntityTypePersons && (opts.BornBefore > 0 || opts.BornAfter > 0 || opts.Birthplace != "") {
		var err error
		births, err = queryBirthIndex(ctx, archive)
		if err != nil {
			return nil, err
		}
	}
	if err := collectQueryJSONContext(ctx, result, archive, opts, births); err != nil {
		return nil, err
	}
	result.Count = len(result.Entities)

	return result, nil
}

func collectQueryJSONContext(ctx context.Context, result *queryJSONResult, archive *glxlib.GLXFile, opts *queryOpts, births map[string]*glxlib.Event) error {
	switch result.EntityType {
	case glxlib.EntityTypePersons:
		return appendQueryJSON(ctx, result, archive.Persons, archive, opts, births)
	case glxlib.EntityTypeEvents:
		return appendQueryJSON(ctx, result, archive.Events, archive, opts, births)
	case glxlib.EntityTypeAssertions:
		return appendQueryJSON(ctx, result, archive.Assertions, archive, opts, births)
	case glxlib.EntityTypeSources:
		return appendQueryJSON(ctx, result, archive.Sources, archive, opts, births)
	case glxlib.EntityTypeRelationships:
		return appendQueryJSON(ctx, result, archive.Relationships, archive, opts, births)
	case glxlib.EntityTypePlaces:
		return appendQueryJSON(ctx, result, archive.Places, archive, opts, births)
	case glxlib.EntityTypeCitations:
		return appendQueryJSON(ctx, result, archive.Citations, archive, opts, births)
	case glxlib.EntityTypeRepositories:
		return appendQueryJSON(ctx, result, archive.Repositories, archive, opts, births)
	case glxlib.EntityTypeMedia:
		return appendQueryJSON(ctx, result, archive.Media, archive, opts, births)
	case glxlib.EntityTypeResearchLogs:
		return appendQueryJSON(ctx, result, archive.ResearchLogs, archive, opts, births)
	case glxlib.EntityTypeStudies:
		return appendQueryJSON(ctx, result, archive.Studies, archive, opts, births)
	}

	return nil
}

func appendQueryJSON[T any](ctx context.Context, result *queryJSONResult, entities map[string]*T, archive *glxlib.GLXFile, opts *queryOpts, births map[string]*glxlib.Event) error {
	for _, id := range sortedKeys(entities) {
		if err := ctx.Err(); err != nil {
			return err
		}
		entity := entities[id]
		if entity != nil && matchesIndexedQueryEntity(archive, id, entity, opts, births) {
			result.Entities = append(result.Entities, queryJSONEntity{ID: id, Entity: entity})
		}
	}

	return ctx.Err()
}

// queryBirthIndex preserves FindPersonEvent's lowest-ID birth and principal
// role rules while avoiding an archive-wide event sort/scan for every person.
func queryBirthIndex(ctx context.Context, archive *glxlib.GLXFile) (map[string]*glxlib.Event, error) {
	births := make(map[string]*glxlib.Event)
	for _, id := range sortedKeys(archive.Events) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		event := archive.Events[id]
		if event == nil || event.Type != glxlib.EventTypeBirth {
			continue
		}
		for _, participant := range event.Participants {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			switch participant.Role {
			case "", glxlib.ParticipantRolePrincipal, glxlib.ParticipantRoleSubject, glxlib.ParticipantRoleChild:
				if births[participant.Person] == nil {
					births[participant.Person] = event
				}
			}
		}
	}

	return births, ctx.Err()
}

func matchesIndexedQueryEntity(archive *glxlib.GLXFile, id string, entity any, opts *queryOpts, births map[string]*glxlib.Event) bool {
	person, ok := entity.(*glxlib.Person)
	if !ok || births == nil {
		return matchesQueryEntity(archive, id, entity, opts)
	}
	local := *opts
	local.BornBefore, local.BornAfter, local.Birthplace = 0, 0, ""
	if !personMatchesQuery(archive, id, person, &local) {
		return false
	}
	event := births[id]
	if opts.Birthplace != "" {
		if event == nil || event.PlaceID == "" {
			return false
		}
		needle := strings.ToLower(opts.Birthplace)
		matched := containsFold(event.PlaceID, needle)
		if place := archive.Places[event.PlaceID]; place != nil {
			matched = matched || containsFold(place.Name, needle)
		}
		if !matched {
			return false
		}
	}
	year := 0
	if event != nil {
		year = extractDateYear(string(event.Date))
	}

	return matchesQueryYear(year, opts.BornBefore, opts.BornAfter)
}

func explorationJSONEnabled(flags []bool) bool { return len(flags) > 0 && flags[0] }

func matchesQueryEntity(archive *glxlib.GLXFile, id string, entity any, opts *queryOpts) bool {
	switch item := entity.(type) {
	case *glxlib.Person:
		return personMatchesQuery(archive, id, item, opts)
	case *glxlib.Event:
		year := extractDateYear(string(item.Date))

		return (opts.Type == "" || strings.EqualFold(item.Type, opts.Type)) && matchesQueryYear(year, opts.Before, opts.After)
	case *glxlib.Assertion:
		return assertionMatchesQuery(archive, item, opts)
	case *glxlib.Source:
		return (opts.Name == "" || containsFold(item.Title, opts.Name)) && (opts.Type == "" || strings.EqualFold(item.Type, opts.Type))
	case *glxlib.Relationship:
		return opts.Type == "" || strings.EqualFold(item.Type, opts.Type)
	case *glxlib.Place:
		return opts.Name == "" || containsFold(item.Name, opts.Name)
	case *glxlib.Repository:
		return opts.Name == "" || containsFold(item.Name, opts.Name)
	case *glxlib.ResearchLog:
		return (opts.Status == "" || strings.EqualFold(item.Status, opts.Status)) && (opts.Subject == "" || researchLogMatchesSubject(item, strings.ToLower(opts.Subject), archive))
	default:
		return true
	}
}

func assertionMatchesQuery(archive *glxlib.GLXFile, assertion *glxlib.Assertion, opts *queryOpts) bool {
	return (opts.Subject == "" || assertionMatchesSubject(assertion, strings.ToLower(opts.Subject), archive)) &&
		(opts.Confidence == "" || strings.EqualFold(assertion.Confidence, opts.Confidence)) &&
		(opts.Status == "" || strings.EqualFold(assertion.Status, opts.Status)) &&
		(opts.Source == "" || assertionReferencesSource(assertion, archive, opts.Source)) &&
		(opts.Citation == "" || slices.Contains(assertion.Citations, opts.Citation))
}

func matchesQueryYear(year, before, after int) bool {
	return (before <= 0 || (year != 0 && year < before)) && (after <= 0 || (year != 0 && year > after))
}

func personMatchesQuery(archive *glxlib.GLXFile, id string, person *glxlib.Person, opts *queryOpts) bool {
	if opts.Status != "" && !strings.EqualFold(glxlib.PersonStatus(person), opts.Status) {
		return false
	}
	if opts.Name != "" {
		matched := false
		for _, name := range extractAllNames(person) {
			if !opts.Phonetic {
				matched = matched || containsFold(name, opts.Name)

				continue
			}
			for queryWord := range strings.FieldsSeq(opts.Name) {
				code := glxlib.Soundex(queryWord)
				for nameWord := range strings.FieldsSeq(name) {
					if code != "" && glxlib.Soundex(nameWord) == code {
						matched = true
					}
				}
			}
		}
		if !matched {
			return false
		}
	}
	if opts.Birthplace != "" && !personMatchesBirthplace(id, opts.Birthplace, archive) {
		return false
	}
	if opts.BornBefore <= 0 && opts.BornAfter <= 0 {
		return true
	}
	year := 0
	if _, event := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeBirth); event != nil {
		year = extractDateYear(string(event.Date))
	}

	return matchesQueryYear(year, opts.BornBefore, opts.BornAfter)
}

type vitalsJSONResult struct {
	PersonID string            `json:"person_id"`
	Vitals   []vitalRecord     `json:"vitals"`
	Events   []queryJSONEntity `json:"events"`
}

func resolveJSONPerson(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	if archive == nil {
		return "", nil, glxlib.ErrNilArchive
	}
	if strings.TrimSpace(query) == "" {
		return "", nil, errEmptyJSONPerson
	}

	return findPersonByQuery(archive, query)
}

func buildVitalsJSON(archive *glxlib.GLXFile, personQuery string) (*vitalsJSONResult, error) {
	id, person, err := resolveJSONPerson(archive, personQuery)
	if err != nil {
		return nil, err
	}
	result := &vitalsJSONResult{PersonID: id, Vitals: collectVitals(id, person, archive), Events: []queryJSONEntity{}}
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event != nil && isPrincipalParticipant(id, event) && slices.Contains([]string{glxlib.EventTypeBirth, glxlib.EventTypeChristening, glxlib.EventTypeBaptism, glxlib.EventTypeDeath, glxlib.EventTypeBurial, glxlib.EventTypeCremation}, event.Type) {
			result.Events = append(result.Events, queryJSONEntity{ID: eventID, Entity: event})
		}
	}

	return result, nil
}

type timelineJSONResult struct {
	PersonID string          `json:"person_id"`
	Name     string          `json:"name"`
	Entries  []timelineEntry `json:"entries"`
}

func buildTimelineJSON(archive *glxlib.GLXFile, personQuery string, includeFamily bool) (*timelineJSONResult, error) {
	id, person, err := resolveJSONPerson(archive, personQuery)
	if err != nil {
		return nil, err
	}
	entries := collectTimelineEntries(id, archive, includeFamily)
	if entries == nil {
		entries = []timelineEntry{}
	}

	return &timelineJSONResult{PersonID: id, Name: extractPersonName(person), Entries: entries}, nil
}

type familyJSONEdge struct {
	PersonID     string `json:"person_id"`
	Step         bool   `json:"step"`
	Hypothetical bool   `json:"hypothetical"`
	Alternative  bool   `json:"alternative"`
}

type siblingJSONEdge struct {
	PersonID string `json:"person_id"`
	// Empty kind means no half/step classification, including incomplete parent data.
	Kind         string `json:"kind"`
	Hypothetical bool   `json:"hypothetical"`
}

type excludedParentJSONEdge struct {
	PersonID string                      `json:"person_id"`
	Reason   glxlib.RelationshipStanding `json:"reason"`
}

type summaryJSONResult struct {
	ExcludedParents      []excludedParentJSONEdge `json:"excluded_parents"`
	PersonID             string                   `json:"person_id"`
	Person               *glxlib.Person           `json:"person"`
	Names                []nameVariant            `json:"names"`
	Vitals               []vitalRecord            `json:"vitals"`
	Timeline             []timelineEntry          `json:"timeline"`
	Parents              []familyJSONEdge         `json:"parents"`
	Children             []familyJSONEdge         `json:"children"`
	Siblings             []siblingJSONEdge        `json:"siblings"`
	Spouses              []spouseInfo             `json:"spouses"`
	OtherRelationships   []otherRelInfo           `json:"other_relationships"`
	Narrative            string                   `json:"narrative"`
	Assertions           []queryJSONEntity        `json:"assertions"`
	ResearchLogs         []queryJSONEntity        `json:"research_logs"`
	ResearchLogRefs      []summaryResearchLogRef  `json:"research_log_refs"`
	ResearchLogCount     int                      `json:"research_log_count"`
	ResearchLogsIncluded bool                     `json:"research_logs_included"`
}

// A reference deliberately excludes objectives, searches, conclusions and notes.
// IsSubject distinguishes direct subject logs from lead-candidate associations.
type summaryResearchLogRef struct {
	ID          string            `json:"id"`
	Title       string            `json:"title,omitempty"`
	Status      string            `json:"status,omitempty"`
	Date        glxlib.DateString `json:"date,omitempty"`
	IsSubject   bool              `json:"is_subject"`
	SearchCount int               `json:"search_count"`
	LeadCount   int               `json:"lead_count"`
}

func buildSummaryJSON(archive *glxlib.GLXFile, personQuery string) (*summaryJSONResult, error) {
	return buildSummaryJSONContext(context.Background(), archive, personQuery)
}

func buildSummaryJSONContext(ctx context.Context, archive *glxlib.GLXFile, personQuery string) (*summaryJSONResult, error) {
	// Preserve the CLI JSON contract, which historically includes full logs.
	return buildSummaryJSONWithLogsContext(ctx, archive, personQuery, true)
}

//nolint:contextcheck // Existing summary helpers are synchronous phases, checked before and after each call.
func buildSummaryJSONWithLogsContext(ctx context.Context, archive *glxlib.GLXFile, personQuery string, includeLogs bool) (*summaryJSONResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, person, err := resolveJSONPerson(archive, personQuery)
	if err != nil {
		return nil, err
	}
	result := &summaryJSONResult{
		PersonID: id, Person: person, Assertions: []queryJSONEntity{},
		ResearchLogs: []queryJSONEntity{}, ResearchLogRefs: []summaryResearchLogRef{}, ResearchLogsIncluded: includeLogs,
	}
	if err := runContextPhases(
		ctx,
		func() { result.Names = extractAllNameVariants(person) },
		func() { result.Vitals = collectVitals(id, person, archive) },
		func() { result.Timeline = collectTimelineEntries(id, archive, true) },
		func() { result.Spouses = findSpouses(id, archive) },
		func() { result.OtherRelationships = findOtherRelationships(id, archive) },
		func() { result.Narrative = generateLifeHistory(id, person, archive) },
		func() { populateSummaryJSONFamily(result, archive) },
	); err != nil {
		return nil, err
	}
	for _, assertionID := range sortedKeys(archive.Assertions) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		assertion := archive.Assertions[assertionID]
		if assertion != nil && assertion.Subject.Person == id {
			result.Assertions = append(result.Assertions, queryJSONEntity{ID: assertionID, Entity: assertion})
		}
	}
	for _, logID := range researchLogIDsForPerson(id, archive) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		log := archive.ResearchLogs[logID]
		result.ResearchLogRefs = append(result.ResearchLogRefs, summaryResearchLogRef{
			ID: logID, Title: log.Title, Status: log.Status, Date: log.Date,
			IsSubject: log.Subject != nil && log.Subject.Person == id, SearchCount: len(log.Searches), LeadCount: len(log.Leads),
		})
		if includeLogs {
			result.ResearchLogs = append(result.ResearchLogs, queryJSONEntity{ID: logID, Entity: log})
		}
	}
	result.ResearchLogCount = len(result.ResearchLogRefs)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// populateSummaryJSONFamily shares the text summary's evidence-aware graph;
// a list of bare IDs would erase hypotheses, rejected alternatives and step links.
func populateSummaryJSONFamily(result *summaryJSONResult, archive *glxlib.GLXFile) {
	family := newFamilyLinks(archive)
	competing := family.competingBirthParents(result.PersonID)
	result.Parents = []familyJSONEdge{}
	result.Children = []familyJSONEdge{}
	result.Siblings = []siblingJSONEdge{}
	result.ExcludedParents = []excludedParentJSONEdge{}
	for _, edge := range family.parents(result.PersonID) {
		result.Parents = append(result.Parents, familyJSONEdge{PersonID: edge.PersonID, Step: edge.Step, Hypothetical: edge.Hypothetical, Alternative: competing[edge.PersonID]})
	}
	for _, edge := range family.children(result.PersonID) {
		result.Children = append(result.Children, familyJSONEdge{PersonID: edge.PersonID, Step: edge.Step, Hypothetical: edge.Hypothetical})
	}
	for _, edge := range family.siblings(result.PersonID) {
		result.Siblings = append(result.Siblings, siblingJSONEdge(edge))
	}
	for _, id := range family.excludedParents(result.PersonID) {
		result.ExcludedParents = append(result.ExcludedParents, excludedParentJSONEdge{PersonID: id, Reason: glxlib.RelationshipStandingDisproven})
	}
}

type treeJSONResult struct {
	Direction   string               `json:"direction"`
	Root        *treeNode            `json:"root"`
	Suggestions []ancestorSuggestion `json:"suggestions,omitempty"`
}

func buildTreeJSON(archive *glxlib.GLXFile, personQuery string, generations int, ancestors bool) (*treeJSONResult, error) {
	if generations < 0 {
		return nil, errNegativeJSONGenerations
	}
	id, _, err := resolveJSONPerson(archive, personQuery)
	if err != nil {
		return nil, err
	}
	tc := newTreeContext(archive)
	result := &treeJSONResult{Direction: jsonDescendants}
	if ancestors {
		result.Direction = jsonAncestors
		result.Root = buildAncestorTree(tc, id, generations, 0, map[string]bool{})
		result.Suggestions = buildAncestorSuggestions(tc, id, archive)
	} else {
		result.Root = buildDescendantTree(tc, id, generations, 0, map[string]bool{})
	}

	return result, nil
}

type citationJSONEntry struct {
	ID       string           `json:"id"`
	Text     string           `json:"text"`
	Citation *glxlib.Citation `json:"citation"`
}
type citationJSONResult struct {
	Citations []citationJSONEntry `json:"citations"`
}

func buildCitationJSON(archive *glxlib.GLXFile, citationID string, all bool) (*citationJSONResult, error) {
	if archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	ids := []string{citationID}
	if all {
		ids = sortedKeys(archive.Citations)
	}
	result := &citationJSONResult{Citations: []citationJSONEntry{}}
	for _, id := range ids {
		citation := archive.Citations[id]
		if citation == nil {
			return nil, fmt.Errorf("%w: %q", errJSONCitationNotFound, id)
		}
		result.Citations = append(result.Citations, citationJSONEntry{ID: id, Text: formatCitation(citation, archive), Citation: citation})
	}

	return result, nil
}

// printExplorationJSON writes only structured output to stdout.
func printExplorationJSON(value any, err error) error {
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	return encoder.Encode(value)
}

// marshalArchiveJSON honors the archive's YAML field names without expanding
// the public model's JSON contract. It is deliberately deferred to encoding:
// paginated MCP queries normalize only the entities in the returned page.
func marshalArchiveJSON(value any) (json.RawMessage, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := yaml.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}

	return json.Marshal(normalized)
}

func (entry queryJSONEntity) MarshalJSON() ([]byte, error) {
	entity := entry.encoded
	if entity == nil {
		var err error
		entity, err = marshalArchiveJSON(entry.Entity)
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(struct {
		ID     string          `json:"id"`
		Entity json.RawMessage `json:"entity"`
	}{ID: entry.ID, Entity: entity})
}

func (entry citationJSONEntry) MarshalJSON() ([]byte, error) {
	citation, err := marshalArchiveJSON(entry.Citation)
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		ID       string          `json:"id"`
		Text     string          `json:"text"`
		Citation json.RawMessage `json:"citation"`
	}{ID: entry.ID, Text: entry.Text, Citation: citation})
}

func (result *summaryJSONResult) MarshalJSON() ([]byte, error) {
	type summaryAlias summaryJSONResult
	person, err := marshalArchiveJSON(result.Person)
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		*summaryAlias
		Person json.RawMessage `json:"person"`
	}{summaryAlias: (*summaryAlias)(result), Person: person})
}

func (suggestion ancestorSuggestion) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PersonID string `json:"person_id"`
		Category string `json:"category"`
		Priority string `json:"priority"`
		Year     int    `json:"year"`
		Message  string `json:"message"`
	}{PersonID: suggestion.PersonID, Category: suggestion.Category, Priority: suggestion.Priority, Year: suggestion.Year, Message: suggestion.Message})
}
