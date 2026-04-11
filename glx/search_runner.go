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
	"os"
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

// searchEntityType pairs an entity type key with its display label.
type searchEntityType struct {
	Key         string
	DisplayName string
}

// searchEntityTypes is the canonical ordered list of searchable entity types.
var searchEntityTypes = []searchEntityType{
	{"persons", "Persons"},
	{"events", "Events"},
	{"places", "Places"},
	{"sources", "Sources"},
	{"citations", "Citations"},
	{"repositories", "Repositories"},
	{"assertions", "Assertions"},
	{"relationships", "Relationships"},
	{"media", "Media"},
}

// searchEntityTypeMap provides O(1) lookup by key.
var searchEntityTypeMap = func() map[string]searchEntityType {
	m := make(map[string]searchEntityType, len(searchEntityTypes))
	for _, et := range searchEntityTypes {
		m[et.Key] = et
	}

	return m
}()

// searchResult represents a single search match.
type searchResult struct {
	EntityType string // "persons", "events", etc.
	EntityID   string
	Field      string // which field matched
	Value      string // the matching value (truncated for display)
}

// searchProps searches a properties map (sorted keys for deterministic output)
// and appends any matches to results. The prefix distinguishes entity-level
// properties ("properties.") from participant-level ("participant.properties.").
func searchProps(entityType, id, prefix string, props map[string]any, matchFn func(string) bool) []searchResult {
	var results []searchResult
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		var s string
		switch v := props[key].(type) {
		case string:
			s = v
		case map[string]any:
			if val, ok := v["value"]; ok {
				if str, ok := val.(string); ok {
					s = str
				}
			}
		default:
			continue
		}
		if matchFn(s) {
			results = append(results, searchResult{entityType, id, prefix + key, truncate(s)})
		}
	}

	return results
}

// searchSlice searches a string slice (ref IDs, authors, etc.) and appends matches.
func searchSlice(entityType, id, field string, items []string, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, item := range items {
		if matchFn(item) {
			results = append(results, searchResult{entityType, id, field, truncate(item)})
		}
	}

	return results
}

// searchParticipants searches participants and appends matches.
func searchParticipants(entityType, id string, participants []glxlib.Participant, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, p := range participants {
		if matchFn(p.Person) {
			results = append(results, searchResult{entityType, id, "participant", p.Person})
		}
		if matchFn(p.Role) {
			results = append(results, searchResult{entityType, id, "participant.role", p.Role})
		}
		if matchFn(p.Notes.String()) {
			results = append(results, searchResult{entityType, id, "participant.notes", truncate(p.Notes.String())})
		}
		results = append(results, searchProps(entityType, id, "participant.properties.", p.Properties, matchFn)...)
	}

	return results
}

// searchPersons searches all Person entities in the archive.
func searchPersons(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"persons", id, "id", id})
		}
		results = append(results, searchProps("persons", id, "properties.", person.Properties, matchFn)...)
		if matchFn(person.Notes.String()) {
			results = append(results, searchResult{"persons", id, "notes", truncate(person.Notes.String())})
		}
	}

	return results
}

// searchEvents searches all Event entities in the archive.
func searchEvents(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Events) {
		ev := archive.Events[id]
		if ev == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"events", id, "id", id})
		}
		if matchFn(ev.Title) {
			results = append(results, searchResult{"events", id, "title", ev.Title})
		}
		if matchFn(ev.Type) {
			results = append(results, searchResult{"events", id, "type", ev.Type})
		}
		if matchFn(string(ev.Date)) {
			results = append(results, searchResult{"events", id, "date", string(ev.Date)})
		}
		if matchFn(ev.PlaceID) {
			results = append(results, searchResult{"events", id, "place", ev.PlaceID})
		}
		if matchFn(ev.Notes.String()) {
			results = append(results, searchResult{"events", id, "notes", truncate(ev.Notes.String())})
		}
		results = append(results, searchProps("events", id, "properties.", ev.Properties, matchFn)...)
		results = append(results, searchParticipants("events", id, ev.Participants, matchFn)...)
	}

	return results
}

// searchPlaces searches all Place entities in the archive.
func searchPlaces(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Places) {
		place := archive.Places[id]
		if place == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"places", id, "id", id})
		}
		if matchFn(place.Name) {
			results = append(results, searchResult{"places", id, "name", place.Name})
		}
		if matchFn(place.Type) {
			results = append(results, searchResult{"places", id, "type", place.Type})
		}
		if matchFn(place.ParentID) {
			results = append(results, searchResult{"places", id, "parent", place.ParentID})
		}
		if matchFn(place.Notes.String()) {
			results = append(results, searchResult{"places", id, "notes", truncate(place.Notes.String())})
		}
		results = append(results, searchProps("places", id, "properties.", place.Properties, matchFn)...)
	}

	return results
}

// searchSources searches all Source entities in the archive.
func searchSources(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Sources) {
		src := archive.Sources[id]
		if src == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"sources", id, "id", id})
		}
		if matchFn(src.Title) {
			results = append(results, searchResult{"sources", id, "title", src.Title})
		}
		if matchFn(src.Type) {
			results = append(results, searchResult{"sources", id, "type", src.Type})
		}
		if matchFn(src.Description) {
			results = append(results, searchResult{"sources", id, "description", truncate(src.Description)})
		}
		if matchFn(string(src.Date)) {
			results = append(results, searchResult{"sources", id, "date", string(src.Date)})
		}
		if matchFn(src.RepositoryID) {
			results = append(results, searchResult{"sources", id, "repository", src.RepositoryID})
		}
		if matchFn(src.Language) {
			results = append(results, searchResult{"sources", id, "language", src.Language})
		}
		if matchFn(src.Notes.String()) {
			results = append(results, searchResult{"sources", id, "notes", truncate(src.Notes.String())})
		}
		results = append(results, searchSlice("sources", id, "author", src.Authors, matchFn)...)
		results = append(results, searchSlice("sources", id, "media", src.Media, matchFn)...)
		results = append(results, searchProps("sources", id, "properties.", src.Properties, matchFn)...)
	}

	return results
}

// searchCitations searches all Citation entities in the archive.
func searchCitations(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Citations) {
		cit := archive.Citations[id]
		if cit == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"citations", id, "id", id})
		}
		if matchFn(cit.SourceID) {
			results = append(results, searchResult{"citations", id, "source", cit.SourceID})
		}
		if matchFn(cit.RepositoryID) {
			results = append(results, searchResult{"citations", id, "repository", cit.RepositoryID})
		}
		if matchFn(cit.Notes.String()) {
			results = append(results, searchResult{"citations", id, "notes", truncate(cit.Notes.String())})
		}
		results = append(results, searchSlice("citations", id, "media", cit.Media, matchFn)...)
		results = append(results, searchProps("citations", id, "properties.", cit.Properties, matchFn)...)
	}

	return results
}

// searchRepositories searches all Repository entities in the archive.
func searchRepositories(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Repositories) {
		repo := archive.Repositories[id]
		if repo == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"repositories", id, "id", id})
		}
		if matchFn(repo.Name) {
			results = append(results, searchResult{"repositories", id, "name", repo.Name})
		}
		if matchFn(repo.Type) {
			results = append(results, searchResult{"repositories", id, "type", repo.Type})
		}
		if matchFn(repo.Address) {
			results = append(results, searchResult{"repositories", id, "address", repo.Address})
		}
		if matchFn(repo.City) {
			results = append(results, searchResult{"repositories", id, "city", repo.City})
		}
		if matchFn(repo.State) {
			results = append(results, searchResult{"repositories", id, "state", repo.State})
		}
		if matchFn(repo.PostalCode) {
			results = append(results, searchResult{"repositories", id, "postal_code", repo.PostalCode})
		}
		if matchFn(repo.Country) {
			results = append(results, searchResult{"repositories", id, "country", repo.Country})
		}
		if matchFn(repo.Website) {
			results = append(results, searchResult{"repositories", id, "website", repo.Website})
		}
		if matchFn(repo.Notes.String()) {
			results = append(results, searchResult{"repositories", id, "notes", truncate(repo.Notes.String())})
		}
		results = append(results, searchProps("repositories", id, "properties.", repo.Properties, matchFn)...)
	}

	return results
}

// searchAssertions searches all Assertion entities in the archive.
func searchAssertions(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"assertions", id, "id", id})
		}
		if matchFn(a.Subject.Person) {
			results = append(results, searchResult{"assertions", id, "subject.person", a.Subject.Person})
		}
		if matchFn(a.Subject.Event) {
			results = append(results, searchResult{"assertions", id, "subject.event", a.Subject.Event})
		}
		if matchFn(a.Subject.Relationship) {
			results = append(results, searchResult{"assertions", id, "subject.relationship", a.Subject.Relationship})
		}
		if matchFn(a.Subject.Place) {
			results = append(results, searchResult{"assertions", id, "subject.place", a.Subject.Place})
		}
		if matchFn(a.Property) {
			results = append(results, searchResult{"assertions", id, "property", a.Property})
		}
		if matchFn(a.Value) {
			results = append(results, searchResult{"assertions", id, "value", a.Value})
		}
		if matchFn(string(a.Date)) {
			results = append(results, searchResult{"assertions", id, "date", string(a.Date)})
		}
		if matchFn(a.Confidence) {
			results = append(results, searchResult{"assertions", id, "confidence", a.Confidence})
		}
		if matchFn(a.Status) {
			results = append(results, searchResult{"assertions", id, "status", a.Status})
		}
		if matchFn(a.Notes.String()) {
			results = append(results, searchResult{"assertions", id, "notes", truncate(a.Notes.String())})
		}
		results = append(results, searchSlice("assertions", id, "source", a.Sources, matchFn)...)
		results = append(results, searchSlice("assertions", id, "citation", a.Citations, matchFn)...)
		results = append(results, searchSlice("assertions", id, "media", a.Media, matchFn)...)
		if a.Participant != nil {
			results = append(results, searchParticipants("assertions", id, []glxlib.Participant{*a.Participant}, matchFn)...)
		}
	}

	return results
}

// searchRelationships searches all Relationship entities in the archive.
func searchRelationships(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[id]
		if rel == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"relationships", id, "id", id})
		}
		if matchFn(rel.Type) {
			results = append(results, searchResult{"relationships", id, "type", rel.Type})
		}
		if matchFn(rel.StartEvent) {
			results = append(results, searchResult{"relationships", id, "start_event", rel.StartEvent})
		}
		if matchFn(rel.EndEvent) {
			results = append(results, searchResult{"relationships", id, "end_event", rel.EndEvent})
		}
		if matchFn(rel.Notes.String()) {
			results = append(results, searchResult{"relationships", id, "notes", truncate(rel.Notes.String())})
		}
		results = append(results, searchParticipants("relationships", id, rel.Participants, matchFn)...)
		results = append(results, searchProps("relationships", id, "properties.", rel.Properties, matchFn)...)
	}

	return results
}

// searchMedia searches all Media entities in the archive.
func searchMedia(archive *glxlib.GLXFile, matchFn func(string) bool) []searchResult {
	var results []searchResult
	for _, id := range sortedKeys(archive.Media) {
		m := archive.Media[id]
		if m == nil {
			continue
		}
		if matchFn(id) {
			results = append(results, searchResult{"media", id, "id", id})
		}
		if matchFn(m.Title) {
			results = append(results, searchResult{"media", id, "title", m.Title})
		}
		if matchFn(m.Type) {
			results = append(results, searchResult{"media", id, "type", m.Type})
		}
		if matchFn(m.URI) {
			results = append(results, searchResult{"media", id, "uri", m.URI})
		}
		if matchFn(m.MimeType) {
			results = append(results, searchResult{"media", id, "mime_type", m.MimeType})
		}
		if matchFn(m.Hash) {
			results = append(results, searchResult{"media", id, "hash", m.Hash})
		}
		if matchFn(m.Description) {
			results = append(results, searchResult{"media", id, "description", truncate(m.Description)})
		}
		if matchFn(string(m.Date)) {
			results = append(results, searchResult{"media", id, "date", string(m.Date)})
		}
		if matchFn(m.Source) {
			results = append(results, searchResult{"media", id, "source", m.Source})
		}
		if matchFn(m.Notes.String()) {
			results = append(results, searchResult{"media", id, "notes", truncate(m.Notes.String())})
		}
		results = append(results, searchProps("media", id, "properties.", m.Properties, matchFn)...)
	}

	return results
}

// searchFuncs maps entity type keys to their search functions.
var searchFuncs = map[string]func(*glxlib.GLXFile, func(string) bool) []searchResult{
	"persons":       searchPersons,
	"events":        searchEvents,
	"places":        searchPlaces,
	"sources":       searchSources,
	"citations":     searchCitations,
	"repositories":  searchRepositories,
	"assertions":    searchAssertions,
	"relationships": searchRelationships,
	"media":         searchMedia,
}

// searchArchive searches entities for the given query string. If typeFilter is
// non-empty, only that entity type is searched; otherwise all types are searched.
func searchArchive(archive *glxlib.GLXFile, query string, caseSensitive bool, typeFilter string) []searchResult {
	matchFn := containsMatch(query, caseSensitive)

	// Early dispatch: search only the requested type.
	if typeFilter != "" {
		if fn, ok := searchFuncs[typeFilter]; ok {
			return fn(archive, matchFn)
		}

		return nil
	}

	results := make([]searchResult, 0, len(archive.Persons)+len(archive.Events))
	for _, et := range searchEntityTypes {
		if fn, ok := searchFuncs[et.Key]; ok {
			results = append(results, fn(archive, matchFn)...)
		}
	}

	return results
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
		key := r.EntityType + ":" + r.EntityID
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, r)
	}

	return deduped
}

// showSearch loads an archive and performs a full-text search.
func showSearch(archivePath, query string, caseSensitive bool, typeFilter string, jsonOutput bool) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return errSearchQueryEmpty
	}

	// Validate --type filter
	typeFilter = strings.ToLower(strings.TrimSpace(typeFilter))
	if typeFilter != "" {
		if _, ok := searchEntityTypeMap[typeFilter]; !ok {
			var valid []string
			for _, et := range searchEntityTypes {
				valid = append(valid, et.Key)
			}

			return fmt.Errorf("unknown type %q (valid: %s): %w", typeFilter, strings.Join(valid, ", "), errUnknownSearchType)
		}
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	var archive *glxlib.GLXFile
	if info.IsDir() {
		loaded, duplicates, loadErr := LoadArchiveWithOptions(archivePath, false)
		if loadErr != nil {
			return fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}
		archive = loaded
	} else {
		loaded, loadErr := readSingleFileArchive(archivePath, false)
		if loadErr != nil {
			return loadErr
		}
		archive = loaded
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
			out = append(out, jsonResult(r))
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
	groups := make(map[string][]searchResult)
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
