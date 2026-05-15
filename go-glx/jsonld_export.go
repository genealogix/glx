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
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"sync"

	"github.com/genealogix/glx/specification/jsonld"
)

// JSONLDVersion is the JSON-LD spec version targeted by the exporter.
const JSONLDVersion = "1.1"

// Entity-ID prefixes used to build fragment identifiers (#person-<id>, etc.)
// in the emitted @graph. Kept as constants so node builders and reference
// helpers agree on the same scheme.
const (
	jsonLDPersonPrefix       = "#person-"
	jsonLDEventPrefix        = "#event-"
	jsonLDPlacePrefix        = "#place-"
	jsonLDRelationshipPrefix = "#relationship-"
	jsonLDSourcePrefix       = "#source-"
	jsonLDCitationPrefix     = "#citation-"
	jsonLDRepositoryPrefix   = "#repository-"
	jsonLDMediaPrefix        = "#media-"
	jsonLDAssertionPrefix    = "#assertion-"
)

// Schema.org / glx @type values used in the @graph.
const (
	jsonLDTypePerson        = "Person"
	jsonLDTypeEvent         = "Event"
	jsonLDTypePlace         = "Place"
	jsonLDTypeRelationship  = "Relationship"
	jsonLDTypeSource        = "CreativeWork"
	jsonLDTypeCitation      = "Citation"
	jsonLDTypeRepository    = "ArchiveOrganization"
	jsonLDTypeMedia         = "MediaObject"
	jsonLDTypeAssertion     = "Assertion"
	jsonLDTypeParticipation = "Participation"
	jsonLDTypeGeoCoords     = "GeoCoordinates"
	jsonLDTypePostalAddress = "PostalAddress"
)

// Person-property names whose values should map to Schema.org aliases. Names
// not in this set fall through as glx:<name> to keep custom vocabularies
// forward-compatible.
//
// `sex` (biological) and `gender` (self-identified) are distinct GLX
// properties — both can be present on the same Person. Schema.org has only
// `gender`, so we map GLX `gender` -> schema:gender and keep GLX `sex` under
// the distinct `glx:sex` property so neither value is silently overwritten.
var jsonLDPersonPropertyAliases = map[string]string{
	"name":       "name",
	"sex":        "glx:sex",
	"gender":     "gender",
	"occupation": "hasOccupation",
	"residence":  "homeLocation",
}

// ExportJSONLD converts a GLX archive to a JSON-LD document.
//
// The output is a single self-contained JSON object with an inlined @context
// (a copy of specification/jsonld/glx-context.jsonld) and an @graph array
// containing one node per entity. Schema.org classes are used where they
// match (Person, Event, Place, CreativeWork, ArchiveOrganization,
// MediaObject); the glx: namespace covers Citation, Relationship, Assertion,
// and Participation, which Schema.org doesn't model.
//
// logWriter receives progress/diagnostic output (nil to suppress). If
// glx.EventTypes is nil, standard vocabularies are loaded into the GLXFile
// (mutating the input) to match the GEDCOM exporter's behavior.
func ExportJSONLD(glx *GLXFile, logWriter io.Writer) ([]byte, *ExportResult, error) {
	if glx == nil {
		return nil, nil, ErrGLXFileNil
	}

	logger := NewImportLogger(logWriter)
	logger.LogInfo("Starting JSON-LD export")

	if glx.EventTypes == nil {
		if err := LoadStandardVocabulariesIntoGLX(glx); err != nil {
			return nil, nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
		}
	}

	context, err := decodeJSONLDContext()
	if err != nil {
		return nil, nil, err
	}

	graph, stats := buildJSONLDGraph(glx)

	doc := map[string]any{
		"@context": context,
		"@graph":   graph,
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	logger.LogInfof(
		"JSON-LD export completed: %d persons, %d events, %d places, %d sources, %d citations, %d repositories, %d media, %d relationships, %d assertions",
		stats.PersonsExported, stats.EventsProcessed, stats.PlacesResolved,
		stats.SourcesExported, stats.CitationsExported, stats.RepositoriesExported,
		stats.MediaExported, stats.RelationshipsExported, stats.AssertionsExported,
	)

	return data, &ExportResult{Statistics: stats, Version: JSONLDVersion}, nil
}

// cachedJSONLDContext holds the parsed embedded JSON-LD @context. The
// embedded bytes are immutable so the parse happens once per process,
// matching the sync.Once pattern in vocabularies.go for the embedded
// vocabulary files.
var (
	cachedJSONLDContext     any
	errJSONLDContextCached  error
	cachedJSONLDContextOnce sync.Once
)

// decodeJSONLDContext parses the embedded canonical context file (once per
// process) and returns the top-level @context value.
func decodeJSONLDContext() (any, error) {
	cachedJSONLDContextOnce.Do(func() {
		cachedJSONLDContext, errJSONLDContextCached = parseJSONLDContextBytes(jsonld.ContextBytes)
	})

	return cachedJSONLDContext, errJSONLDContextCached
}

// parseJSONLDContextBytes unwraps a JSON-LD context document and returns the
// inner @context value. The canonical file at specification/jsonld/
// glx-context.jsonld wraps the mapping under "@context" so it is itself a
// valid JSON-LD context document, which is why callers strip one layer.
func parseJSONLDContextBytes(b []byte) (any, error) {
	var wrapper struct {
		Context any `json:"@context"`
	}
	if err := json.Unmarshal(b, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-LD context: %w", err)
	}
	if wrapper.Context == nil {
		return nil, ErrJSONLDContextMissing
	}

	return wrapper.Context, nil
}

// buildJSONLDGraph walks every entity map in deterministic order and produces
// the @graph array plus per-type counts.
func buildJSONLDGraph(glx *GLXFile) ([]map[string]any, ExportStatistics) {
	stats := ExportStatistics{}
	graph := make([]map[string]any, 0,
		len(glx.Persons)+len(glx.Events)+len(glx.Places)+
			len(glx.Relationships)+len(glx.Sources)+len(glx.Citations)+
			len(glx.Repositories)+len(glx.Media)+len(glx.Assertions))

	// Nil map values are possible when a YAML entity deserialized to null;
	// skip them rather than panic in the per-entity builders that dereference
	// fields. Matches the GEDCOM exporter's behavior (gedcom_export.go).
	for _, id := range sortedKeys(glx.Persons) {
		p := glx.Persons[id]
		if p == nil {
			continue
		}
		graph = append(graph, personNode(id, p))
		stats.PersonsExported++
	}
	for _, id := range sortedKeys(glx.Events) {
		e := glx.Events[id]
		if e == nil {
			continue
		}
		graph = append(graph, eventNode(id, e))
		stats.EventsProcessed++
	}
	for _, id := range sortedKeys(glx.Places) {
		p := glx.Places[id]
		if p == nil {
			continue
		}
		graph = append(graph, placeNode(id, p))
		stats.PlacesResolved++
	}
	for _, id := range sortedKeys(glx.Relationships) {
		r := glx.Relationships[id]
		if r == nil {
			continue
		}
		graph = append(graph, relationshipNode(id, r))
		stats.RelationshipsExported++
	}
	for _, id := range sortedKeys(glx.Sources) {
		s := glx.Sources[id]
		if s == nil {
			continue
		}
		graph = append(graph, sourceNode(id, s))
		stats.SourcesExported++
	}
	for _, id := range sortedKeys(glx.Citations) {
		c := glx.Citations[id]
		if c == nil {
			continue
		}
		graph = append(graph, citationNode(id, c))
		stats.CitationsExported++
	}
	for _, id := range sortedKeys(glx.Repositories) {
		r := glx.Repositories[id]
		if r == nil {
			continue
		}
		graph = append(graph, repositoryNode(id, r))
		stats.RepositoriesExported++
	}
	for _, id := range sortedKeys(glx.Media) {
		m := glx.Media[id]
		if m == nil {
			continue
		}
		graph = append(graph, mediaNode(id, m))
		stats.MediaExported++
	}
	for _, id := range sortedKeys(glx.Assertions) {
		a := glx.Assertions[id]
		if a == nil {
			continue
		}
		graph = append(graph, assertionNode(id, a))
		stats.AssertionsExported++
	}

	return graph, stats
}

// ---------------------------------------------------------------------------
// Entity-node builders
// ---------------------------------------------------------------------------

func personNode(id string, p *Person) map[string]any {
	node := map[string]any{
		"@id":   jsonLDPersonPrefix + id,
		"@type": jsonLDTypePerson,
	}
	applyPersonProperties(node, p.Properties)
	addNotes(node, p.Notes)

	return node
}

func eventNode(id string, e *Event) map[string]any {
	node := map[string]any{
		"@id":   jsonLDEventPrefix + id,
		"@type": jsonLDTypeEvent,
	}
	if e.Type != "" {
		node["eventType"] = e.Type
	}
	if e.Title != "" {
		node["name"] = e.Title
	}
	if e.Date != "" {
		node["startDate"] = string(e.Date)
	}
	if e.PlaceID != "" {
		node["location"] = jsonLDPlacePrefix + e.PlaceID
	}
	if len(e.Participants) > 0 {
		node["participant"] = participantNodes(e.Participants)
	}
	applyGenericProperties(node, e.Properties)
	addNotes(node, e.Notes)

	return node
}

func placeNode(id string, p *Place) map[string]any {
	node := map[string]any{
		"@id":   jsonLDPlacePrefix + id,
		"@type": jsonLDTypePlace,
	}
	if p.Name != "" {
		node["name"] = p.Name
	}
	if p.ParentID != "" {
		node["containedInPlace"] = jsonLDPlacePrefix + p.ParentID
	}
	if p.Type != "" {
		node["placeType"] = p.Type
	}
	if p.Latitude != nil && p.Longitude != nil {
		node["geo"] = map[string]any{
			"@type":     jsonLDTypeGeoCoords,
			"latitude":  *p.Latitude,
			"longitude": *p.Longitude,
		}
	}
	applyGenericProperties(node, p.Properties)
	addNotes(node, p.Notes)

	return node
}

func relationshipNode(id string, r *Relationship) map[string]any {
	node := map[string]any{
		"@id":   jsonLDRelationshipPrefix + id,
		"@type": jsonLDTypeRelationship,
	}
	if r.Type != "" {
		node["relationshipType"] = r.Type
	}
	if r.StartEvent != "" {
		node["startEvent"] = jsonLDEventPrefix + r.StartEvent
	}
	if r.EndEvent != "" {
		node["endEvent"] = jsonLDEventPrefix + r.EndEvent
	}
	if len(r.Participants) > 0 {
		node["participants"] = participantNodes(r.Participants)
	}
	applyGenericProperties(node, r.Properties)
	addNotes(node, r.Notes)

	return node
}

func sourceNode(id string, s *Source) map[string]any {
	node := map[string]any{
		"@id":   jsonLDSourcePrefix + id,
		"@type": jsonLDTypeSource,
	}
	if s.Title != "" {
		node["name"] = s.Title
	}
	if s.Type != "" {
		node["sourceType"] = s.Type
	}
	if len(s.Authors) > 0 {
		node["author"] = stringsToAny(s.Authors)
	}
	if s.Date != "" {
		node["datePublished"] = string(s.Date)
	}
	if s.Description != "" {
		node["description"] = s.Description
	}
	if s.RepositoryID != "" {
		node["repository"] = jsonLDRepositoryPrefix + s.RepositoryID
	}
	if s.Language != "" {
		node["inLanguage"] = s.Language
	}
	if len(s.Media) > 0 {
		node["associatedMedia"] = mediaRefs(s.Media)
	}
	applyGenericProperties(node, s.Properties)
	addNotes(node, s.Notes)

	return node
}

func citationNode(id string, c *Citation) map[string]any {
	node := map[string]any{
		"@id":   jsonLDCitationPrefix + id,
		"@type": jsonLDTypeCitation,
	}
	if c.SourceID != "" {
		node["source"] = jsonLDSourcePrefix + c.SourceID
	}
	if c.RepositoryID != "" {
		node["repository"] = jsonLDRepositoryPrefix + c.RepositoryID
	}
	if len(c.Media) > 0 {
		node["media"] = mediaRefs(c.Media)
	}
	applyGenericProperties(node, c.Properties)
	addNotes(node, c.Notes)

	return node
}

func repositoryNode(id string, r *Repository) map[string]any {
	node := map[string]any{
		"@id":   jsonLDRepositoryPrefix + id,
		"@type": jsonLDTypeRepository,
	}
	if r.Name != "" {
		node["name"] = r.Name
	}
	if r.Type != "" {
		node["repositoryType"] = r.Type
	}
	if r.Website != "" {
		node["url"] = r.Website
	}
	if r.Address != "" || r.City != "" || r.State != "" || r.PostalCode != "" || r.Country != "" {
		addr := map[string]any{"@type": jsonLDTypePostalAddress}
		if r.Address != "" {
			addr["streetAddress"] = r.Address
		}
		if r.City != "" {
			addr["addressLocality"] = r.City
		}
		if r.State != "" {
			addr["addressRegion"] = r.State
		}
		if r.PostalCode != "" {
			addr["postalCode"] = r.PostalCode
		}
		if r.Country != "" {
			addr["addressCountry"] = r.Country
		}
		node["address"] = addr
	}
	applyGenericProperties(node, r.Properties)
	addNotes(node, r.Notes)

	return node
}

func mediaNode(id string, m *Media) map[string]any {
	node := map[string]any{
		"@id":   jsonLDMediaPrefix + id,
		"@type": jsonLDTypeMedia,
	}
	if m.URI != "" {
		node["contentUrl"] = m.URI
	}
	if m.Type != "" {
		node["mediaType"] = m.Type
	}
	if m.MimeType != "" {
		node["encodingFormat"] = m.MimeType
	}
	if m.Hash != "" {
		node["hash"] = m.Hash
	}
	if m.Title != "" {
		node["name"] = m.Title
	}
	if m.Description != "" {
		node["description"] = m.Description
	}
	if m.Date != "" {
		node["datePublished"] = string(m.Date)
	}
	if m.Source != "" {
		node["source"] = jsonLDSourcePrefix + m.Source
	}
	applyGenericProperties(node, m.Properties)
	addNotes(node, m.Notes)

	return node
}

func assertionNode(id string, a *Assertion) map[string]any {
	node := map[string]any{
		"@id":   jsonLDAssertionPrefix + id,
		"@type": jsonLDTypeAssertion,
	}
	if ref := entityRefToFragment(&a.Subject); ref != "" {
		node["subject"] = ref
	}
	if a.Property != "" {
		node["property"] = a.Property
	}
	if a.Value != "" {
		node["value"] = a.Value
	}
	if a.Date != "" {
		node["startDate"] = string(a.Date)
	}
	if a.Participant != nil {
		node["participant"] = participantNode(*a.Participant)
	}
	if a.Confidence != "" {
		node["confidence"] = a.Confidence
	}
	if a.Status != "" {
		node["status"] = a.Status
	}
	if len(a.Sources) > 0 {
		node["sources"] = prefixIDs(jsonLDSourcePrefix, a.Sources)
	}
	if len(a.Citations) > 0 {
		node["citations"] = prefixIDs(jsonLDCitationPrefix, a.Citations)
	}
	if len(a.Media) > 0 {
		node["media"] = mediaRefs(a.Media)
	}
	addNotes(node, a.Notes)

	return node
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// applyPersonProperties handles the Person.Properties map specially: it
// recognizes the structured-name shape (with given/surname/etc. fields) and
// expands it to Schema.org's nested givenName / familyName / honorificPrefix /
// honorificSuffix. Other known properties (sex, occupation, residence) get
// aliased; unknown ones pass through as glx:<name>.
func applyPersonProperties(node, props map[string]any) {
	for k, v := range props {
		if k == "name" {
			if structured, ok := structuredName(v); ok {
				maps.Copy(node, structured)

				continue
			}
		}
		if alias, ok := jsonLDPersonPropertyAliases[k]; ok {
			node[alias] = v

			continue
		}
		node["glx:"+k] = v
	}
}

// structuredName recognizes the GLX structured-name map shape and returns a
// JSON-LD-shaped expansion. Returns false if v is not a structured name.
//
// Two shapes are recognized:
//  1. The PropertyDefinition.Fields layout produced by GEDCOM import —
//     {"value": "John Smith", "fields": {"given": "John", "surname": "Smith"}}.
//     The flat `value` becomes schema:name; nested fields expand into
//     givenName / familyName / honorificPrefix / honorificSuffix.
//  2. A flat layout where the given/surname/etc. fields sit at the top
//     level of the map (some hand-curated archives use this shape).
func structuredName(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}

	out := map[string]any{}
	if value, ok := stringField(m, "value"); ok {
		out["name"] = value
	}

	fields := m
	if nested, ok := m["fields"].(map[string]any); ok {
		fields = nested
	}
	if prefix, ok := stringField(fields, "prefix"); ok {
		out["honorificPrefix"] = prefix
	}
	if given, ok := stringField(fields, "given"); ok {
		out["givenName"] = given
	}
	if surname, ok := stringField(fields, "surname"); ok {
		out["familyName"] = surname
	}
	if suffix, ok := stringField(fields, "suffix"); ok {
		out["honorificSuffix"] = suffix
	}
	if len(out) == 0 {
		return nil, false
	}

	return out, true
}

// stringField pulls a string-typed field out of a map[string]any. Returns
// ("", false) when the key is missing, nil, or holds a non-string value.
func stringField(m map[string]any, key string) (string, bool) {
	raw, ok := m[key]
	if !ok || raw == nil {
		return "", false
	}
	s, ok := raw.(string)
	if !ok || s == "" {
		return "", false
	}

	return s, true
}

// applyGenericProperties pours arbitrary vocabulary properties into node
// under the glx: namespace. Used for entities whose Properties don't have a
// rich Schema.org mapping in v1.
func applyGenericProperties(node, props map[string]any) {
	for k, v := range props {
		node["glx:"+k] = v
	}
}

func addNotes(node map[string]any, notes NoteList) {
	if len(notes) == 0 {
		return
	}
	node["notes"] = stringsToAny(notes)
}

func participantNodes(participants []Participant) []map[string]any {
	out := make([]map[string]any, 0, len(participants))
	for _, p := range participants {
		out = append(out, participantNode(p))
	}

	return out
}

func participantNode(p Participant) map[string]any {
	node := map[string]any{"@type": jsonLDTypeParticipation}
	if p.Person != "" {
		node["person"] = jsonLDPersonPrefix + p.Person
	}
	if p.Role != "" {
		node["role"] = p.Role
	}
	for k, v := range p.Properties {
		node["glx:"+k] = v
	}
	if len(p.Notes) > 0 {
		node["notes"] = stringsToAny(p.Notes)
	}

	return node
}

func entityRefToFragment(ref *EntityRef) string {
	switch {
	case ref.Person != "":
		return jsonLDPersonPrefix + ref.Person
	case ref.Event != "":
		return jsonLDEventPrefix + ref.Event
	case ref.Relationship != "":
		return jsonLDRelationshipPrefix + ref.Relationship
	case ref.Place != "":
		return jsonLDPlacePrefix + ref.Place
	default:
		return ""
	}
}

func mediaRefs(ids []string) []any {
	return prefixIDs(jsonLDMediaPrefix, ids)
}

func prefixIDs(prefix string, ids []string) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, prefix+id)
	}

	return out
}

func stringsToAny(ss []string) []any {
	out := make([]any, 0, len(ss))
	for _, s := range ss {
		out = append(out, s)
	}

	return out
}
