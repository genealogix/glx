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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportJSONLD_NilGLX(t *testing.T) {
	_, _, err := ExportJSONLD(nil, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrGLXFileNil)
}

func TestExportJSONLD_EmptyArchive(t *testing.T) {
	glx := &GLXFile{}

	data, result, err := ExportJSONLD(glx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	if result.Version != JSONLDVersion {
		t.Errorf("expected version %q, got %q", JSONLDVersion, result.Version)
	}
	require.Equal(t, 0, result.Statistics.PersonsExported)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	require.Contains(t, doc, "@context")
	require.Contains(t, doc, "@graph")
	graph, ok := doc["@graph"].([]any)
	require.True(t, ok, "@graph must be an array")
	require.Empty(t, graph, "empty archive should produce empty graph")

	// The context must declare the schema/glx prefixes.
	ctx, ok := doc["@context"].(map[string]any)
	require.True(t, ok, "@context must be an object")
	require.Equal(t, "https://schema.org/", ctx["schema"])
	require.Equal(t, "https://glx.genealogix.dev/ns/v1#", ctx["glx"])
}

func TestExportJSONLD_SkipsNilMapValues(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"alive": {Properties: map[string]any{"name": "Real Person"}},
			"null":  nil,
		},
		Events: map[string]*Event{
			"e1": nil,
		},
	}

	data, result, err := ExportJSONLD(glx, nil)
	require.NoError(t, err, "nil map values must not panic the exporter")
	require.Equal(t, 1, result.Statistics.PersonsExported, "nil persons should be skipped")
	require.Equal(t, 0, result.Statistics.EventsProcessed, "nil events should be skipped")

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	graph, _ := doc["@graph"].([]any)
	require.Len(t, graph, 1)
}

func TestExportJSONLD_FullDocumentShape(t *testing.T) {
	glx := smallFixture()

	data, result, err := ExportJSONLD(glx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))

	graph, ok := doc["@graph"].([]any)
	require.True(t, ok)
	require.Len(t, graph, 9, "fixture has 9 entities total")

	typeCounts := countByType(graph)
	assert.Equal(t, 1, typeCounts["Person"])
	assert.Equal(t, 1, typeCounts["Event"])
	assert.Equal(t, 1, typeCounts["Place"])
	assert.Equal(t, 1, typeCounts["Relationship"])
	assert.Equal(t, 1, typeCounts["CreativeWork"])
	assert.Equal(t, 1, typeCounts["Citation"])
	assert.Equal(t, 1, typeCounts["ArchiveOrganization"])
	assert.Equal(t, 1, typeCounts["MediaObject"])
	assert.Equal(t, 1, typeCounts["Assertion"])

	assert.Equal(t, 1, result.Statistics.PersonsExported)
	assert.Equal(t, 1, result.Statistics.EventsProcessed)
	assert.Equal(t, 1, result.Statistics.PlacesResolved)
	assert.Equal(t, 1, result.Statistics.RelationshipsExported)
	assert.Equal(t, 1, result.Statistics.SourcesExported)
	assert.Equal(t, 1, result.Statistics.CitationsExported)
	assert.Equal(t, 1, result.Statistics.RepositoriesExported)
	assert.Equal(t, 1, result.Statistics.MediaExported)
	assert.Equal(t, 1, result.Statistics.AssertionsExported)
}

func TestPersonNode_StructuredName(t *testing.T) {
	p := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"prefix":  "Rev.",
				"given":   "John",
				"surname": "Shakespeare",
				"suffix":  "Sr.",
			},
			"gender":     "male",
			"occupation": "glover",
		},
	}

	node := personNode("john-shakespeare", p)
	assert.Equal(t, "#person-john-shakespeare", node["@id"])
	assert.Equal(t, "Person", node["@type"])
	assert.Equal(t, "Rev.", node["honorificPrefix"])
	assert.Equal(t, "John", node["givenName"])
	assert.Equal(t, "Shakespeare", node["familyName"])
	assert.Equal(t, "Sr.", node["honorificSuffix"])
	assert.Equal(t, "male", node["gender"])
	assert.Equal(t, "glover", node["hasOccupation"])
	// Unknown property would land under glx:; "name" expanded so should not be present.
	assert.NotContains(t, node, "name")
}

func TestPersonNode_ResidenceAsPlaceReference(t *testing.T) {
	t.Run("string place id", func(t *testing.T) {
		p := &Person{Properties: map[string]any{"residence": "stratford"}}
		node := personNode("william", p)
		assert.Equal(t, "#place-stratford", node["homeLocation"], "residence should become a #place- reference")
		assert.NotContains(t, node, "glx:residence")
	})
	t.Run("non-string falls through to glx:", func(t *testing.T) {
		// e.g. a temporal residence stored as a list — pass through unaliased.
		p := &Person{Properties: map[string]any{
			"residence": []any{
				map[string]any{"value": "stratford", "date": "1564"},
			},
		}}
		node := personNode("william", p)
		assert.NotContains(t, node, "homeLocation")
		assert.NotEmpty(t, node["glx:residence"], "non-string residence must survive under glx:residence")
	})
}

func TestStructuredName_PreservesUnmappedSubfields(t *testing.T) {
	p := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "Anne (Nan) Hathaway",
				"fields": map[string]any{
					"given":          "Anne",
					"surname":        "Hathaway",
					"nickname":       "Nan",
					"surname_prefix": "de",
					"type":           "birth",
				},
			},
		},
	}

	node := personNode("anne", p)
	assert.Equal(t, "Anne (Nan) Hathaway", node["name"])
	assert.Equal(t, "Anne", node["givenName"])
	assert.Equal(t, "Hathaway", node["familyName"])
	assert.Equal(t, "Nan", node["glx:name_nickname"], "nickname must round-trip under glx:")
	assert.Equal(t, "de", node["glx:name_surname_prefix"], "surname_prefix must round-trip under glx:")
	assert.Equal(t, "birth", node["glx:name_type"], "name type must round-trip under glx:")
}

func TestPersonNode_SexAndGenderDoNotCollide(t *testing.T) {
	p := &Person{
		Properties: map[string]any{
			"sex":    "female",
			"gender": "non-binary",
		},
	}

	node := personNode("p1", p)
	assert.Equal(t, "female", node["glx:sex"], "biological sex must survive under glx:sex")
	assert.Equal(t, "non-binary", node["gender"], "self-identified gender maps to schema:gender")
}

func TestPersonNode_NestedFieldsShape(t *testing.T) {
	// This is the shape produced by GEDCOM 5.5.1/7.0 import — a "value" string
	// alongside a "fields" sub-map of structured components.
	p := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "William Shakespeare",
				"fields": map[string]any{
					"given":   "William",
					"surname": "Shakespeare",
				},
			},
		},
	}

	node := personNode("william", p)
	assert.Equal(t, "William Shakespeare", node["name"])
	assert.Equal(t, "William", node["givenName"])
	assert.Equal(t, "Shakespeare", node["familyName"])
}

func TestPersonNode_FlatNameAndUnknownProperty(t *testing.T) {
	p := &Person{
		Properties: map[string]any{
			"name":        "John Shakespeare",
			"some_custom": "value",
		},
	}

	node := personNode("p1", p)
	assert.Equal(t, "John Shakespeare", node["name"])
	assert.Equal(t, "value", node["glx:some_custom"])
}

func TestEventNode_TypeAndLocation(t *testing.T) {
	e := &Event{
		Title:   "Birth of John",
		Type:    "birth",
		PlaceID: "snitterfield",
		Date:    "1531",
		Participants: []Participant{
			{Person: "john-shakespeare", Role: "principal"},
		},
	}

	node := eventNode("john-birth", e)
	assert.Equal(t, "#event-john-birth", node["@id"])
	assert.Equal(t, "Event", node["@type"])
	assert.Equal(t, "birth", node["eventType"])
	assert.Equal(t, "Birth of John", node["name"])
	assert.Equal(t, "1531", node["startDate"])
	assert.Equal(t, "#place-snitterfield", node["location"])

	parts, ok := node["participant"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, parts, 1)
	assert.Equal(t, "#person-john-shakespeare", parts[0]["person"])
	assert.Equal(t, "principal", parts[0]["role"])
	assert.Equal(t, "Participation", parts[0]["@type"])
}

func TestPlaceNode_GeoOnlyWhenBothCoordinatesPresent(t *testing.T) {
	lat, lon := 52.27, -1.69
	cases := []struct {
		name   string
		place  *Place
		hasGeo bool
	}{
		{
			name:   "both coords",
			place:  &Place{Name: "Snitterfield", Latitude: &lat, Longitude: &lon},
			hasGeo: true,
		},
		{
			name:   "only lat",
			place:  &Place{Name: "Snitterfield", Latitude: &lat},
			hasGeo: false,
		},
		{
			name:   "only lon",
			place:  &Place{Name: "Snitterfield", Longitude: &lon},
			hasGeo: false,
		},
		{
			name:   "neither",
			place:  &Place{Name: "Snitterfield"},
			hasGeo: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node := placeNode("snitterfield", tc.place)
			if tc.hasGeo {
				geo, ok := node["geo"].(map[string]any)
				require.True(t, ok, "expected geo block")
				assert.Equal(t, "GeoCoordinates", geo["@type"])
				gotLat, ok := geo["latitude"].(float64)
				require.True(t, ok, "latitude must be float64")
				assert.InDelta(t, 52.27, gotLat, 1e-9)
				gotLon, ok := geo["longitude"].(float64)
				require.True(t, ok, "longitude must be float64")
				assert.InDelta(t, -1.69, gotLon, 1e-9)
			} else {
				assert.NotContains(t, node, "geo")
			}
		})
	}
}

func TestRepositoryNode_AddressOnlyWhenAnyFieldPresent(t *testing.T) {
	r := &Repository{
		Name:    "British Library",
		Address: "96 Euston Road",
		City:    "London",
		Country: "United Kingdom",
		Website: "https://www.bl.uk",
	}

	node := repositoryNode("british-library", r)
	assert.Equal(t, "ArchiveOrganization", node["@type"])
	assert.Equal(t, "British Library", node["name"])
	assert.Equal(t, "https://www.bl.uk", node["url"])

	addr, ok := node["address"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "PostalAddress", addr["@type"])
	assert.Equal(t, "96 Euston Road", addr["streetAddress"])
	assert.Equal(t, "London", addr["addressLocality"])
	assert.Equal(t, "United Kingdom", addr["addressCountry"])

	// Repository with no address fields should not emit an address block.
	bare := &Repository{Name: "Solo"}
	bareNode := repositoryNode("solo", bare)
	assert.NotContains(t, bareNode, "address")
}

func TestParseJSONLDContextBytes(t *testing.T) {
	t.Run("valid context", func(t *testing.T) {
		got, err := parseJSONLDContextBytes([]byte(`{"@context":{"schema":"https://schema.org/"}}`))
		require.NoError(t, err)
		m, ok := got.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://schema.org/", m["schema"])
	})
	t.Run("malformed JSON", func(t *testing.T) {
		_, err := parseJSONLDContextBytes([]byte("not json"))
		require.Error(t, err)
	})
	t.Run("missing @context key", func(t *testing.T) {
		_, err := parseJSONLDContextBytes([]byte(`{"other":"value"}`))
		require.ErrorIs(t, err, ErrJSONLDContextMissing)
	})
}

func TestEntityRefToFragment_AllBranches(t *testing.T) {
	cases := []struct {
		name string
		ref  EntityRef
		want string
	}{
		{name: "person", ref: EntityRef{Person: "p1"}, want: "#person-p1"},
		{name: "event", ref: EntityRef{Event: "e1"}, want: "#event-e1"},
		{name: "relationship", ref: EntityRef{Relationship: "r1"}, want: "#relationship-r1"},
		{name: "place", ref: EntityRef{Place: "pl1"}, want: "#place-pl1"},
		{name: "empty", ref: EntityRef{}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.ref
			assert.Equal(t, tc.want, entityRefToFragment(&r))
		})
	}
}

func TestAssertionNode_ParticipantBranch(t *testing.T) {
	role := "witness"
	a := &Assertion{
		Subject:     EntityRef{Event: "e1"},
		Participant: &Participant{Person: "p1", Role: role},
		Date:        "1850",
		Status:      "confirmed",
		Media:       []string{"m1"},
		Notes:       NoteList{"note one"},
	}

	node := assertionNode("a1", a)
	assert.Equal(t, "#event-e1", node["subject"])
	part, ok := node["participant"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "#person-p1", part["person"])
	assert.Equal(t, role, part["role"])
	assert.Equal(t, "1850", node["startDate"])
	assert.Equal(t, "confirmed", node["status"])
	assert.Equal(t, []any{"#media-m1"}, node["media"])
	assert.Equal(t, []any{"note one"}, node["notes"])
}

func TestSourceCitationMediaNodes_FullFields(t *testing.T) {
	src := &Source{
		Title:        "Stratford Register",
		Type:         "parish_register",
		Authors:      []string{"Holy Trinity"},
		Date:         "1558",
		Description:  "Baptisms, marriages, burials",
		RepositoryID: "shak-trust",
		Language:     "en",
		Media:        []string{"img1"},
		Notes:        NoteList{"hand-written"},
	}
	srcNode := sourceNode("stratford-register", src)
	assert.Equal(t, "Stratford Register", srcNode["name"])
	assert.Equal(t, "parish_register", srcNode["sourceType"])
	assert.Equal(t, []any{"Holy Trinity"}, srcNode["author"])
	assert.Equal(t, "1558", srcNode["datePublished"])
	assert.Equal(t, "Baptisms, marriages, burials", srcNode["description"])
	assert.Equal(t, "#repository-shak-trust", srcNode["repository"])
	assert.Equal(t, "en", srcNode["inLanguage"])
	assert.Equal(t, []any{"#media-img1"}, srcNode["associatedMedia"])
	assert.Equal(t, []any{"hand-written"}, srcNode["notes"])

	cit := &Citation{
		SourceID:     "stratford-register",
		RepositoryID: "shak-trust",
		Media:        []string{"img1"},
		Notes:        NoteList{"page 42"},
	}
	citNode := citationNode("c1", cit)
	assert.Equal(t, "#source-stratford-register", citNode["source"])
	assert.Equal(t, "#repository-shak-trust", citNode["repository"])
	assert.Equal(t, []any{"#media-img1"}, citNode["media"])

	m := &Media{
		URI:         "img/p.jpg",
		Type:        "photo",
		MimeType:    "image/jpeg",
		Title:       "Portrait",
		Description: "Oil on canvas",
		Date:        "1610",
		Source:      "stratford-register",
		Notes:       NoteList{"restored 2010"},
	}
	mNode := mediaNode("portrait", m)
	assert.Equal(t, "photo", mNode["mediaType"])
	assert.Equal(t, "Oil on canvas", mNode["description"])
	assert.Equal(t, "1610", mNode["datePublished"])
	assert.Equal(t, "#source-stratford-register", mNode["source"])
}

func TestRelationshipNode_StartAndEndEvents(t *testing.T) {
	r := &Relationship{
		Type:       "marriage",
		StartEvent: "wedding-1582",
		EndEvent:   "burial-1623",
		Participants: []Participant{
			{Person: "p1", Role: "spouse"},
			{Person: "p2", Role: "spouse"},
		},
		Properties: map[string]any{"location": "Stratford"},
		Notes:      NoteList{"per parish register"},
	}
	node := relationshipNode("rel1", r)
	assert.Equal(t, "#event-wedding-1582", node["startEvent"])
	assert.Equal(t, "#event-burial-1623", node["endEvent"])
	assert.Equal(t, "Stratford", node["glx:location"])
	assert.Equal(t, []any{"per parish register"}, node["notes"])
}

func TestAssertionNode_SubjectFragment(t *testing.T) {
	a := &Assertion{
		Subject:    EntityRef{Person: "john"},
		Property:   "occupation",
		Value:      "glover",
		Confidence: "high",
		Sources:    []string{"deed-1565"},
		Citations:  []string{"deed-1565-john-page-12"},
	}

	node := assertionNode("john-occupation", a)
	assert.Equal(t, "#person-john", node["subject"])
	assert.Equal(t, "occupation", node["property"])
	assert.Equal(t, "glover", node["value"])
	assert.Equal(t, "high", node["confidence"])
	assert.Equal(t, []any{"#source-deed-1565"}, node["sources"])
	assert.Equal(t, []any{"#citation-deed-1565-john-page-12"}, node["citations"])
}

func TestMediaNode_ContentURLAndMimeType(t *testing.T) {
	m := &Media{
		URI:      "media/john-portrait-1870.jpg",
		MimeType: "image/jpeg",
		Title:    "John Shakespeare portrait",
		Hash:     "sha256:deadbeef",
	}

	node := mediaNode("john-portrait-1870", m)
	assert.Equal(t, "MediaObject", node["@type"])
	assert.Equal(t, "media/john-portrait-1870.jpg", node["contentUrl"])
	assert.Equal(t, "image/jpeg", node["encodingFormat"])
	assert.Equal(t, "John Shakespeare portrait", node["name"])
	assert.Equal(t, "sha256:deadbeef", node["hash"])
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func smallFixture() *GLXFile {
	lat, lon := 52.27, -1.69

	return &GLXFile{
		Persons: map[string]*Person{
			"john-shakespeare": {Properties: map[string]any{"name": "John Shakespeare", "sex": "male"}},
		},
		Events: map[string]*Event{
			"john-birth": {
				Type:    "birth",
				PlaceID: "snitterfield",
				Date:    "1531",
				Participants: []Participant{
					{Person: "john-shakespeare", Role: "principal"},
				},
			},
		},
		Places: map[string]*Place{
			"snitterfield": {
				Name:      "Snitterfield",
				Latitude:  &lat,
				Longitude: &lon,
			},
		},
		Relationships: map[string]*Relationship{
			"john-mary-marriage": {
				Type: "marriage",
				Participants: []Participant{
					{Person: "john-shakespeare", Role: "spouse"},
					{Person: "mary-arden", Role: "spouse"},
				},
			},
		},
		Sources: map[string]*Source{
			"parish-register-stratford": {
				Title:        "Stratford Parish Register",
				Authors:      []string{"Holy Trinity Church"},
				Date:         "1558",
				RepositoryID: "shakespeare-birthplace-trust",
			},
		},
		Citations: map[string]*Citation{
			"john-birth-citation": {
				SourceID: "parish-register-stratford",
			},
		},
		Repositories: map[string]*Repository{
			"shakespeare-birthplace-trust": {
				Name:    "Shakespeare Birthplace Trust",
				City:    "Stratford-upon-Avon",
				Country: "United Kingdom",
				Website: "https://www.shakespeare.org.uk",
			},
		},
		Media: map[string]*Media{
			"john-portrait": {
				URI:      "media/john-portrait.jpg",
				MimeType: "image/jpeg",
			},
		},
		Assertions: map[string]*Assertion{
			"john-occupation-assertion": {
				Subject:  EntityRef{Person: "john-shakespeare"},
				Property: "occupation",
				Value:    "glover",
			},
		},
	}
}

func countByType(graph []any) map[string]int {
	out := map[string]int{}
	for _, raw := range graph {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		t, _ := node["@type"].(string)
		out[t]++
	}

	return out
}
