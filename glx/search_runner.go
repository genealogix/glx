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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Sentinel errors for search validation.
var (
	errSearchQueryEmpty  = errors.New("search query cannot be empty")
	errUnknownSearchType = errors.New("unknown search type")
)

// searchResultMaxLen is the maximum rune length for truncated display values.
const searchResultMaxLen = 80

// Field-path constants for searchResult. These are the JSON-style field names
// reported to the user when a match hits a specific scalar field on an entity;
// pulling them into named constants keeps the runner free of literal-string
// duplication and lets goconst do its job.
const (
	searchFieldID          = "id"
	searchFieldName        = "name"
	searchFieldTitle       = "title"
	searchFieldType        = "type"
	searchFieldStatus      = "status"
	searchFieldNotes       = "notes"
	searchFieldParent      = "parent"
	searchFieldURI         = "uri"
	searchFieldMIMEType    = "mime_type"
	searchFieldHash        = "hash"
	searchFieldDescription = "description"
	searchFieldDateRange   = "date_range"
	searchFieldCollection  = "collection"
	searchFieldQuery       = "query"
	searchFieldResult      = "result"
	searchFieldResearcher  = "researcher"
	searchFieldObjective   = "objective"
	searchFieldConclusions = "conclusions"
	searchFieldProperty    = "property"
	searchFieldValue       = "value"
	searchFieldCity        = "city"
	searchFieldState       = "state"
	searchFieldPostalCode  = "postal_code"
	searchFieldCountry     = "country"
	searchFieldRepository  = "repository"
	searchFieldSource      = "source"
	searchFieldCitation    = "citation"
	searchFieldPlaces      = "places"
	searchFieldSources     = "sources"
	searchFieldCitations   = "citations"
)

// searchEntityType pairs an entity type key with its display label.
type searchEntityType struct {
	Key         glxlib.EntityType
	DisplayName string
}

// searchEntityTypes is the canonical ordered list of searchable entity types.
var searchEntityTypes = []searchEntityType{
	{glxlib.EntityTypePersons, "Persons"},
	{glxlib.EntityTypeEvents, "Events"},
	{glxlib.EntityTypePlaces, "Places"},
	{glxlib.EntityTypeSources, "Sources"},
	{glxlib.EntityTypeCitations, "Citations"},
	{glxlib.EntityTypeRepositories, "Repositories"},
	{glxlib.EntityTypeAssertions, "Assertions"},
	{glxlib.EntityTypeRelationships, "Relationships"},
	{glxlib.EntityTypeMedia, "Media"},
	{glxlib.EntityTypeResearchLogs, "Research Logs"},
	{glxlib.EntityTypeStudies, "Studies"},
}

// searchEntityTypeMap provides O(1) lookup by key.
var searchEntityTypeMap = func() map[glxlib.EntityType]searchEntityType {
	m := make(map[glxlib.EntityType]searchEntityType, len(searchEntityTypes))
	for _, et := range searchEntityTypes {
		m[et.Key] = et
	}

	return m
}()

// searchResult represents a single search match.
type searchResult struct {
	EntityType glxlib.EntityType
	EntityID   string
	Field      string // which field matched
	Value      string // the matching value (truncated for display)
}

// searchScalarField preserves each field's legacy display truncation policy.
type searchScalarField struct {
	field   string
	value   string
	shorten bool
}

func searchFields(ctx context.Context, entityType glxlib.EntityType, id string, fields []searchScalarField, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, field := range fields {
		if ctx.Err() != nil {
			return false
		}
		if !matchFn(field.value) {
			continue
		}
		value := field.value
		if field.shorten {
			value = truncate(value)
		}
		if !visit(searchResult{entityType, id, field.field, value}) {
			return false
		}
	}

	return true
}

// searchProps searches a properties map (sorted keys for deterministic output)
// and visits any matches. The prefix distinguishes entity-level
// properties ("properties.") from participant-level ("participant.properties.").
// Property values are enumerated via propertyScalars so every value in the
// canonical GLX shapes (string, map-with-"value", []any temporal list) is
// searchable; the first matching value within a property is recorded.
func searchProps(ctx context.Context, entityType glxlib.EntityType, id, prefix string, props map[string]any, matchFn func(string) bool, visit func(searchResult) bool) bool {
	keys := make([]string, 0, len(props))
	for k := range props {
		if ctx.Err() != nil {
			return false
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if ctx.Err() != nil {
			return false
		}
		for _, s := range propertyScalars(props[key]) {
			if ctx.Err() != nil {
				return false
			}
			if matchFn(s) {
				if !visit(searchResult{entityType, id, prefix + key, truncate(s)}) {
					return false
				}

				break
			}
		}
	}

	return true
}

// searchSlice visits matching string slice values (ref IDs, authors, etc.).
func searchSlice(ctx context.Context, entityType glxlib.EntityType, id, field string, items []string, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, item := range items {
		if ctx.Err() != nil {
			return false
		}
		if matchFn(item) && !visit(searchResult{entityType, id, field, truncate(item)}) {
			return false
		}
	}

	return true
}

// searchParticipants visits matching participant fields.
func searchParticipants(ctx context.Context, entityType glxlib.EntityType, id string, participants []glxlib.Participant, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, p := range participants {
		if ctx.Err() != nil {
			return false
		}
		if matchFn(p.Person) && !visit(searchResult{entityType, id, "participant", p.Person}) {
			return false
		}
		if matchFn(p.Role) && !visit(searchResult{entityType, id, "participant.role", p.Role}) {
			return false
		}
		if matchFn(p.Notes.String()) && !visit(searchResult{entityType, id, "participant.notes", truncate(p.Notes.String())}) {
			return false
		}
		if !searchProps(ctx, entityType, id, "participant.properties.", p.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchPersons searches all Person entities in the archive.
func searchPersons(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Persons) {
		if ctx.Err() != nil {
			return false
		}
		person := archive.Persons[id]
		if person == nil {
			continue
		}
		if matchFn(id) && !visit(searchResult{glxlib.EntityTypePersons, id, searchFieldID, id}) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypePersons, id, "properties.", person.Properties, matchFn, visit) {
			return false
		}
		if matchFn(person.Notes.String()) && !visit(searchResult{glxlib.EntityTypePersons, id, searchFieldNotes, truncate(person.Notes.String())}) {
			return false
		}
	}

	return true
}

// searchEvents searches all Event entities in the archive.
func searchEvents(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Events) {
		if ctx.Err() != nil {
			return false
		}
		ev := archive.Events[id]
		if ev == nil {
			continue
		}
		if matchFn(id) && !visit(searchResult{glxlib.EntityTypeEvents, id, searchFieldID, id}) {
			return false
		}
		if matchFn(ev.Title) && !visit(searchResult{glxlib.EntityTypeEvents, id, searchFieldTitle, ev.Title}) {
			return false
		}
		if matchFn(ev.Type) && !visit(searchResult{glxlib.EntityTypeEvents, id, searchFieldType, ev.Type}) {
			return false
		}
		if matchFn(string(ev.Date)) && !visit(searchResult{glxlib.EntityTypeEvents, id, eventFieldDate, string(ev.Date)}) {
			return false
		}
		if matchFn(ev.PlaceID) && !visit(searchResult{glxlib.EntityTypeEvents, id, eventFieldPlace, ev.PlaceID}) {
			return false
		}
		if matchFn(ev.Notes.String()) && !visit(searchResult{glxlib.EntityTypeEvents, id, searchFieldNotes, truncate(ev.Notes.String())}) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypeEvents, id, "properties.", ev.Properties, matchFn, visit) {
			return false
		}
		if !searchParticipants(ctx, glxlib.EntityTypeEvents, id, ev.Participants, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchPlaces searches all Place entities in the archive.
func searchPlaces(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Places) {
		if ctx.Err() != nil {
			return false
		}
		place := archive.Places[id]
		if place == nil {
			continue
		}
		if matchFn(id) && !visit(searchResult{glxlib.EntityTypePlaces, id, searchFieldID, id}) {
			return false
		}
		if matchFn(place.Name) && !visit(searchResult{glxlib.EntityTypePlaces, id, searchFieldName, place.Name}) {
			return false
		}
		if matchFn(place.Type) && !visit(searchResult{glxlib.EntityTypePlaces, id, searchFieldType, place.Type}) {
			return false
		}
		for _, parentID := range place.ParentIDs() {
			if matchFn(parentID) {
				if !visit(searchResult{glxlib.EntityTypePlaces, id, searchFieldParent, parentID}) {
					return false
				}

				break
			}
		}
		if matchFn(place.Notes.String()) && !visit(searchResult{glxlib.EntityTypePlaces, id, searchFieldNotes, truncate(place.Notes.String())}) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypePlaces, id, "properties.", place.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchSources searches all Source entities in the archive.
func searchSources(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Sources) {
		if ctx.Err() != nil {
			return false
		}
		src := archive.Sources[id]
		if src == nil {
			continue
		}
		if !searchFields(ctx, glxlib.EntityTypeSources, id, []searchScalarField{
			{searchFieldID, id, false},
			{searchFieldTitle, src.Title, false},
			{searchFieldType, src.Type, false},
			{eventFieldDate, string(src.Date), false},
			{searchFieldRepository, src.RepositoryID, false},
			{"language", src.Language, false},
			{searchFieldNotes, src.Notes.String(), true},
		}, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, glxlib.EntityTypeSources, id, "author", src.Authors, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, glxlib.EntityTypeSources, id, "media", src.Media, matchFn, visit) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypeSources, id, "properties.", src.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchCitations searches all Citation entities in the archive.
func searchCitations(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Citations) {
		if ctx.Err() != nil {
			return false
		}
		cit := archive.Citations[id]
		if cit == nil {
			continue
		}
		if matchFn(id) && !visit(searchResult{glxlib.EntityTypeCitations, id, searchFieldID, id}) {
			return false
		}
		if matchFn(cit.SourceID) && !visit(searchResult{glxlib.EntityTypeCitations, id, searchFieldSource, cit.SourceID}) {
			return false
		}
		if matchFn(cit.RepositoryID) && !visit(searchResult{glxlib.EntityTypeCitations, id, searchFieldRepository, cit.RepositoryID}) {
			return false
		}
		if matchFn(cit.Notes.String()) && !visit(searchResult{glxlib.EntityTypeCitations, id, searchFieldNotes, truncate(cit.Notes.String())}) {
			return false
		}
		if !searchSlice(ctx, glxlib.EntityTypeCitations, id, "media", cit.Media, matchFn, visit) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypeCitations, id, "properties.", cit.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchRepositories searches all Repository entities in the archive.
func searchRepositories(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Repositories) {
		if ctx.Err() != nil {
			return false
		}
		repo := archive.Repositories[id]
		if repo == nil {
			continue
		}
		if !searchFields(ctx, glxlib.EntityTypeRepositories, id, []searchScalarField{
			{searchFieldID, id, false},
			{searchFieldName, repo.Name, false},
			{searchFieldType, repo.Type, false},
			{"address", repo.Address, false},
			{searchFieldCity, repo.City, false},
			{searchFieldState, repo.State, false},
			{searchFieldPostalCode, repo.PostalCode, false},
			{searchFieldCountry, repo.Country, false},
			{"website", repo.Website, false},
			{searchFieldNotes, repo.Notes.String(), true},
		}, matchFn, visit) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypeRepositories, id, "properties.", repo.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchAssertions searches all Assertion entities in the archive.
func searchAssertions(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Assertions) {
		if ctx.Err() != nil {
			return false
		}
		a := archive.Assertions[id]
		if a == nil {
			continue
		}
		if !searchFields(ctx, glxlib.EntityTypeAssertions, id, []searchScalarField{
			{searchFieldID, id, false},
			{"subject.person", a.Subject.Person, false},
			{"subject.event", a.Subject.Event, false},
			{"subject.relationship", a.Subject.Relationship, false},
			{"subject.place", a.Subject.Place, false},
			{searchFieldProperty, a.Property, false},
			{searchFieldValue, a.Value, false},
			{eventFieldDate, string(a.Date), false},
			{"confidence", a.Confidence, false},
			{searchFieldStatus, a.Status, false},
			{searchFieldNotes, a.Notes.String(), true},
		}, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, glxlib.EntityTypeAssertions, id, "source", a.Sources, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, glxlib.EntityTypeAssertions, id, "citation", a.Citations, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, glxlib.EntityTypeAssertions, id, "media", a.Media, matchFn, visit) {
			return false
		}
		if a.Participant != nil {
			if !searchParticipants(ctx, glxlib.EntityTypeAssertions, id, []glxlib.Participant{*a.Participant}, matchFn, visit) {
				return false
			}
		}
	}

	return true
}

// searchRelationships searches all Relationship entities in the archive.
func searchRelationships(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Relationships) {
		if ctx.Err() != nil {
			return false
		}
		rel := archive.Relationships[id]
		if rel == nil {
			continue
		}
		if matchFn(id) && !visit(searchResult{glxlib.EntityTypeRelationships, id, searchFieldID, id}) {
			return false
		}
		if matchFn(rel.Type) && !visit(searchResult{glxlib.EntityTypeRelationships, id, searchFieldType, rel.Type}) {
			return false
		}
		if matchFn(rel.StartEvent) && !visit(searchResult{glxlib.EntityTypeRelationships, id, "start_event", rel.StartEvent}) {
			return false
		}
		if matchFn(rel.EndEvent) && !visit(searchResult{glxlib.EntityTypeRelationships, id, "end_event", rel.EndEvent}) {
			return false
		}
		if matchFn(rel.Notes.String()) && !visit(searchResult{glxlib.EntityTypeRelationships, id, searchFieldNotes, truncate(rel.Notes.String())}) {
			return false
		}
		if !searchParticipants(ctx, glxlib.EntityTypeRelationships, id, rel.Participants, matchFn, visit) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypeRelationships, id, "properties.", rel.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchMedia searches all Media entities in the archive.
func searchMedia(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	for _, id := range sortedKeys(archive.Media) {
		if ctx.Err() != nil {
			return false
		}
		m := archive.Media[id]
		if m == nil {
			continue
		}
		if !searchFields(ctx, glxlib.EntityTypeMedia, id, []searchScalarField{
			{searchFieldID, id, false},
			{searchFieldTitle, m.Title, false},
			{searchFieldType, m.Type, false},
			{searchFieldURI, m.URI, false},
			{searchFieldMIMEType, m.MimeType, false},
			{searchFieldHash, m.Hash, false},
			{eventFieldDate, string(m.Date), false},
			{searchFieldSource, m.Source, false},
			{searchFieldNotes, m.Notes.String(), true},
		}, matchFn, visit) {
			return false
		}
		if !searchProps(ctx, glxlib.EntityTypeMedia, id, "properties.", m.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchResearchLogs searches all ResearchLog entities in the archive.
// Each Search entry inside a log is searched as a separate result so that
// matches on the query/result/citation fields are addressable.
func searchResearchLogs(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	const et = glxlib.EntityTypeResearchLogs

	for _, id := range sortedKeys(archive.ResearchLogs) {
		if ctx.Err() != nil {
			return false
		}
		log := archive.ResearchLogs[id]
		if log == nil {
			continue
		}
		if !searchResearchLogFields(ctx, et, id, log, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, et, id, "citations", log.Citations, matchFn, visit) {
			return false
		}
		for i := range log.Searches {
			if ctx.Err() != nil {
				return false
			}
			if !searchSearchEntry(ctx, et, id, i, &log.Searches[i], matchFn, visit) {
				return false
			}
		}
		if !searchProps(ctx, et, id, "properties.", log.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchResearchLogFields enumerates the scalar string fields of a ResearchLog.
func searchResearchLogFields(ctx context.Context, et glxlib.EntityType, id string, log *glxlib.ResearchLog, matchFn func(string) bool, visit func(searchResult) bool) bool {
	if ctx.Err() != nil {
		return false
	}
	if matchFn(id) && !visit(searchResult{et, id, searchFieldID, id}) {
		return false
	}
	if matchFn(log.Title) && !visit(searchResult{et, id, searchFieldTitle, log.Title}) {
		return false
	}
	if matchFn(log.Researcher) && !visit(searchResult{et, id, searchFieldResearcher, log.Researcher}) {
		return false
	}
	if matchFn(log.Objective) && !visit(searchResult{et, id, searchFieldObjective, truncate(log.Objective)}) {
		return false
	}
	if matchFn(log.Status) && !visit(searchResult{et, id, searchFieldStatus, log.Status}) {
		return false
	}
	if matchFn(string(log.Date)) && !visit(searchResult{et, id, eventFieldDate, string(log.Date)}) {
		return false
	}
	if matchFn(log.Conclusions) && !visit(searchResult{et, id, searchFieldConclusions, truncate(log.Conclusions)}) {
		return false
	}
	if matchFn(log.Notes.String()) && !visit(searchResult{et, id, searchFieldNotes, truncate(log.Notes.String())}) {
		return false
	}

	return true
}

// searchSearchEntry enumerates the scalar string fields of one embedded Search.
// Field paths are prefixed `searches[i].` so per-search matches are addressable.
func searchSearchEntry(ctx context.Context, et glxlib.EntityType, id string, idx int, s *glxlib.Search, matchFn func(string) bool, visit func(searchResult) bool) bool {
	if ctx.Err() != nil {
		return false
	}
	prefix := fmt.Sprintf("searches[%d].", idx)

	if matchFn(s.RepositoryID) && !visit(searchResult{et, id, prefix + "repository", s.RepositoryID}) {
		return false
	}
	if matchFn(s.SourceID) && !visit(searchResult{et, id, prefix + "source", s.SourceID}) {
		return false
	}
	if matchFn(s.Collection) && !visit(searchResult{et, id, prefix + "collection", s.Collection}) {
		return false
	}
	if matchFn(s.Query) && !visit(searchResult{et, id, prefix + "query", truncate(s.Query)}) {
		return false
	}
	if matchFn(s.Result) && !visit(searchResult{et, id, prefix + "result", s.Result}) {
		return false
	}
	if matchFn(s.CitationID) && !visit(searchResult{et, id, prefix + "citation", s.CitationID}) {
		return false
	}
	if matchFn(string(s.Date)) && !visit(searchResult{et, id, prefix + "date", string(s.Date)}) {
		return false
	}
	if matchFn(s.Notes.String()) && !visit(searchResult{et, id, prefix + "notes", truncate(s.Notes.String())}) {
		return false
	}

	return true
}

// searchStudies searches all Study entities in the archive.
func searchStudies(ctx context.Context, archive *glxlib.GLXFile, matchFn func(string) bool, visit func(searchResult) bool) bool {
	const et = glxlib.EntityTypeStudies

	for _, id := range sortedKeys(archive.Studies) {
		if ctx.Err() != nil {
			return false
		}
		s := archive.Studies[id]
		if s == nil {
			continue
		}
		if matchFn(id) && !visit(searchResult{et, id, searchFieldID, id}) {
			return false
		}
		if matchFn(s.Title) && !visit(searchResult{et, id, searchFieldTitle, s.Title}) {
			return false
		}
		if matchFn(s.Type) && !visit(searchResult{et, id, searchFieldType, s.Type}) {
			return false
		}
		if matchFn(s.Status) && !visit(searchResult{et, id, searchFieldStatus, s.Status}) {
			return false
		}
		if matchFn(string(s.DateRange)) && !visit(searchResult{et, id, searchFieldDateRange, string(s.DateRange)}) {
			return false
		}
		if matchFn(s.Notes.String()) && !visit(searchResult{et, id, searchFieldNotes, truncate(s.Notes.String())}) {
			return false
		}
		if !searchSlice(ctx, et, id, "places", s.Places, matchFn, visit) {
			return false
		}
		if !searchSlice(ctx, et, id, "sources", s.Sources, matchFn, visit) {
			return false
		}
		if !searchProps(ctx, et, id, "properties.", s.Properties, matchFn, visit) {
			return false
		}
	}

	return true
}

// searchFuncs maps entity type keys to their search functions.
var searchFuncs = map[glxlib.EntityType]func(context.Context, *glxlib.GLXFile, func(string) bool, func(searchResult) bool) bool{
	glxlib.EntityTypePersons:       searchPersons,
	glxlib.EntityTypeEvents:        searchEvents,
	glxlib.EntityTypePlaces:        searchPlaces,
	glxlib.EntityTypeSources:       searchSources,
	glxlib.EntityTypeCitations:     searchCitations,
	glxlib.EntityTypeRepositories:  searchRepositories,
	glxlib.EntityTypeAssertions:    searchAssertions,
	glxlib.EntityTypeRelationships: searchRelationships,
	glxlib.EntityTypeMedia:         searchMedia,
	glxlib.EntityTypeResearchLogs:  searchResearchLogs,
	glxlib.EntityTypeStudies:       searchStudies,
}

// searchArchive searches entities for the given query string. If typeFilter is
// non-empty, only that entity type is searched; otherwise all types are searched.
func searchArchive(archive *glxlib.GLXFile, query string, caseSensitive bool, typeFilter string) []searchResult {
	var results []searchResult
	if err := visitSearchArchiveContext(context.Background(), archive, query, caseSensitive, typeFilter, func(result searchResult) bool {
		results = append(results, result)

		return true
	}); err != nil {
		return nil
	}

	return results
}

// visitSearchArchiveContext shares field matching and ordering with the CLI,
// without retaining every match. Returning false from visit stops the scan.
// Cancellation is checked between entities and within nested collections;
// callers discard partial results when an error is returned.
func visitSearchArchiveContext(ctx context.Context, archive *glxlib.GLXFile, query string, caseSensitive bool, typeFilter string, visit func(searchResult) bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if archive == nil {
		return glxlib.ErrNilArchive
	}
	matchFn := containsMatch(query, caseSensitive)
	if typeFilter != "" {
		if fn, ok := searchFuncs[glxlib.EntityType(typeFilter)]; ok {
			fn(ctx, archive, matchFn, visit)
		}

		return ctx.Err()
	}
	for _, et := range searchEntityTypes {
		if !searchFuncs[et.Key](ctx, archive, matchFn, visit) {
			break
		}
	}

	return ctx.Err()
}

// containsMatch returns a match function for the given query.
// Reuses containsFold from query_runner.go for case-insensitive matching.
func containsMatch(query string, caseSensitive bool) func(string) bool {
	if caseSensitive {
		return func(s string) bool {
			return s != "" && strings.Contains(s, query)
		}
	}
	lowerQuery := strings.ToLower(query)

	return func(s string) bool {
		return s != "" && containsFold(s, lowerQuery)
	}
}

// truncate shortens a string to searchResultMaxLen runes with "..." suffix.
// Uses rune slicing to avoid splitting multi-byte UTF-8 characters.
func truncate(s string) string {
	runes := []rune(s)
	if len(runes) <= searchResultMaxLen {
		return s
	}

	return string(runes[:searchResultMaxLen-3]) + "..."
}

// deduplicateResults removes duplicate (entityType, entityID) pairs,
// keeping the first match per entity.
func deduplicateResults(results []searchResult) []searchResult {
	seen := make(map[string]bool)
	var deduped []searchResult
	for _, r := range results {
		key := r.EntityType.String() + ":" + r.EntityID
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, r)
	}

	return deduped
}

// normalizeSearchInput applies the CLI's query and collection validation.
func normalizeSearchInput(query, typeFilter string) (string, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", "", errSearchQueryEmpty
	}

	// Validate --type filter
	typeFilter = strings.ToLower(strings.TrimSpace(typeFilter))
	if typeFilter != "" {
		if _, ok := searchEntityTypeMap[glxlib.EntityType(typeFilter)]; !ok {
			var valid []string
			for _, et := range searchEntityTypes {
				valid = append(valid, et.Key.String())
			}

			return "", "", fmt.Errorf("unknown type %q (valid: %s): %w", typeFilter, strings.Join(valid, ", "), errUnknownSearchType)
		}
	}

	return query, typeFilter, nil
}

// showSearch loads an archive and performs a full-text search.
func showSearch(archivePath, query string, caseSensitive bool, typeFilter string, jsonOutput bool) error {
	query, typeFilter, err := normalizeSearchInput(query, typeFilter)
	if err != nil {
		return err
	}

	archive, err := loadArchiveForQuery(archivePath)
	if err != nil {
		return err
	}

	results := searchArchive(archive, query, caseSensitive, typeFilter)
	results = deduplicateResults(results)

	if jsonOutput {
		type jsonResult struct {
			EntityType string `json:"entity_type"`
			EntityID   string `json:"entity_id"`
			Field      string `json:"field"`
			Value      string `json:"value"`
		}
		var out []jsonResult
		for _, r := range results {
			out = append(out, jsonResult{
				EntityType: r.EntityType.String(),
				EntityID:   r.EntityID,
				Field:      r.Field,
				Value:      r.Value,
			})
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
		fmt.Println(string(data))

		return nil
	}

	if len(results) == 0 {
		fmt.Printf("No matches found for %q\n", query)

		return nil
	}

	// Group by entity type
	groups := make(map[glxlib.EntityType][]searchResult)
	for _, r := range results {
		groups[r.EntityType] = append(groups[r.EntityType], r)
	}

	fmt.Printf("Found %d match(es) for %q:\n", len(results), query)

	for _, et := range searchEntityTypes {
		group, ok := groups[et.Key]
		if !ok {
			continue
		}

		sort.Slice(group, func(i, j int) bool {
			return group[i].EntityID < group[j].EntityID
		})

		fmt.Printf("\n  %s (%d):\n", et.DisplayName, len(group))
		for _, r := range group {
			fmt.Printf("    %s  %s: %s\n", r.EntityID, r.Field, r.Value)
		}
	}

	return nil
}
