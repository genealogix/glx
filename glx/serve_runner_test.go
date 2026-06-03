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
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// serveTestArchive builds a small three-generation family with sources,
// citations, and assertions to exercise every viewer endpoint.
func serveTestArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-grandpa": {Properties: map[string]any{"name": "Henry Webb", "sex": "male"}},
			"person-dad":     {Properties: map[string]any{"name": "Robert Webb", "sex": "male"}},
			"person-mom":     {Properties: map[string]any{"name": "Mary Lane", "sex": "female"}},
			"person-self": {Properties: map[string]any{
				"name": []any{
					map[string]any{"value": "Jane Webb", "fields": map[string]any{"type": "birth"}},
					map[string]any{"value": "Jane Smith", "fields": map[string]any{"type": "married"}},
				},
				"sex": "female",
			}},
			"person-sib": {Properties: map[string]any{"name": "Tom Webb", "sex": "male"}},
			"person-kid": {Properties: map[string]any{"name": "Baby Smith", "sex": "unknown"}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-grandpa": {Type: "birth", Date: "1820", Participants: []glxlib.Participant{{Person: "person-grandpa", Role: "principal"}}},
			"event-death-grandpa": {Type: "death", Date: "1890", Participants: []glxlib.Participant{{Person: "person-grandpa", Role: "principal"}}},
			"event-birth-self":    {Type: "birth", Date: "1850-03-15", PlaceID: "place-town", Participants: []glxlib.Participant{{Person: "person-self", Role: "principal"}}},
			"event-census":        {Type: "census", Date: "1860", PlaceID: "place-town", Participants: []glxlib.Participant{{Person: "person-self"}, {Person: "person-dad"}}},
			"event-marriage":      {Type: "marriage", Date: "1845", PlaceID: "place-town", Participants: []glxlib.Participant{{Person: "person-dad"}, {Person: "person-mom"}}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"relationship-gp": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-grandpa", Role: "parent"}, {Person: "person-dad", Role: "child"},
			}},
			"relationship-parents": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-dad", Role: "parent"},
				{Person: "person-mom", Role: "parent"},
				{Person: "person-self", Role: "child"},
				{Person: "person-sib", Role: "child"},
			}},
			"relationship-kid": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-self", Role: "parent"}, {Person: "person-kid", Role: "child"},
			}},
			"relationship-marriage": {Type: "marriage", StartEvent: "event-marriage", Participants: []glxlib.Participant{
				{Person: "person-dad"}, {Person: "person-mom"},
			}},
		},
		Places: map[string]*glxlib.Place{
			"place-town": {Name: "Springfield"},
		},
		Sources: map[string]*glxlib.Source{
			"source-census": {Title: "1860 US Census", Type: "census", Properties: map[string]any{"url": "https://example.com/census"}},
			"source-empty":  {Title: ""},
		},
		Citations: map[string]*glxlib.Citation{
			"citation-1": {SourceID: "source-census", Properties: map[string]any{
				"locator": "p. 12", "text_from_source": "Jane Webb, age 10", "accessed": "2026-01-01",
			}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-occupation": {
				Subject:    glxlib.EntityRef{Person: "person-self"},
				Property:   "occupation",
				Value:      "seamstress",
				Confidence: "high",
				Citations:  []string{"citation-1"},
			},
			"assertion-birthplace": {
				Subject:  glxlib.EntityRef{Person: "person-self"},
				Property: "birthplace",
				Value:    "Springfield",
				Sources:  []string{"source-census"},
			},
		},
	}
}

func newServeTestServer(t *testing.T) *viewerServer {
	t.Helper()
	sub, err := fs.Sub(webAssets, "web")
	require.NoError(t, err)

	return &viewerServer{archive: serveTestArchive(), archivePath: "test-archive", assets: sub}
}

func doServeRequest(t *testing.T, srv *viewerServer, path string, out any) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	if out != nil && rec.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), out))
	}

	return rec
}

func TestServeOverview(t *testing.T) {
	srv := newServeTestServer(t)
	var got overviewDTO
	rec := doServeRequest(t, srv, "/api/overview", &got)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test-archive", got.ArchivePath)
	assert.Equal(t, 6, got.Counts["persons"])
	assert.Equal(t, 5, got.Counts["events"])
	assert.Equal(t, 2, got.Counts["sources"])
	assert.Equal(t, 1, got.Counts["citations"])
	assert.Equal(t, 2, got.Counts["assertions"])

	// Both assertions are unset/high — confidence rows are present and ordered.
	require.NotEmpty(t, got.Confidence)
	assert.Equal(t, "high", got.Confidence[0].Level)

	// Coverage: 1 of 6 persons referenced by assertions.
	var personsRow *coverageRowDTO
	for i := range got.Coverage {
		if got.Coverage[i].Label == "Persons" {
			personsRow = &got.Coverage[i]
		}
	}
	require.NotNil(t, personsRow)
	assert.Equal(t, 1, personsRow.Covered)
	assert.Equal(t, 6, personsRow.Total)
}

func TestServePersonsListSortedWithYears(t *testing.T) {
	srv := newServeTestServer(t)
	var got struct {
		Persons []personListItemDTO `json:"persons"`
	}
	rec := doServeRequest(t, srv, "/api/persons", &got)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, got.Persons, 6)

	// Sorted alphabetically by name: "Baby Smith" comes first.
	assert.Equal(t, "Baby Smith", got.Persons[0].Name)

	// Years populated from birth/death events.
	var grandpa *personListItemDTO
	for i := range got.Persons {
		if got.Persons[i].ID == "person-grandpa" {
			grandpa = &got.Persons[i]
		}
	}
	require.NotNil(t, grandpa)
	assert.Equal(t, 1820, grandpa.BirthYear)
	assert.Equal(t, 1890, grandpa.DeathYear)
	assert.Equal(t, "male", grandpa.Sex)
}

func TestServePersonDetail(t *testing.T) {
	srv := newServeTestServer(t)
	var got personDetailDTO
	rec := doServeRequest(t, srv, "/api/persons/person-self", &got)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "person-self", got.ID)
	require.NotEmpty(t, got.Names)
	assert.Equal(t, "Jane Webb", got.Names[0])
	assert.Contains(t, got.Names, "Jane Smith")
	assert.Equal(t, "female", got.Sex)

	// Vitals: birth event surfaced.
	require.NotEmpty(t, got.Vitals)
	assert.Equal(t, "birth", got.Vitals[0].Type)
	assert.Equal(t, "Springfield", got.Vitals[0].Place)

	// Family relationships resolved both directions.
	parentIDs := refIDs(got.Family.Parents)
	assert.ElementsMatch(t, []string{"person-dad", "person-mom"}, parentIDs)
	assert.Equal(t, []string{"person-kid"}, refIDs(got.Family.Children))
	assert.Equal(t, []string{"person-sib"}, refIDs(got.Family.Siblings))

	// Events include the census, ordered chronologically with birth first.
	require.GreaterOrEqual(t, len(got.Events), 2)
	assert.Equal(t, "birth", got.Events[0].Type)

	// Assertions resolve evidence: citation -> source title + locator, and a
	// direct source reference.
	require.Len(t, got.Asserts, 2)
	var occupation *assertionDTO
	for i := range got.Asserts {
		if got.Asserts[i].Property == "occupation" {
			occupation = &got.Asserts[i]
		}
	}
	require.NotNil(t, occupation)
	assert.Equal(t, "seamstress", occupation.Value)
	assert.Equal(t, "high", occupation.Confidence)
	require.Len(t, occupation.Evidence, 1)
	assert.Equal(t, "1860 US Census", occupation.Evidence[0].SourceTitle)
	assert.Equal(t, "p. 12", occupation.Evidence[0].Locator)
	assert.Equal(t, "https://example.com/census", occupation.Evidence[0].URL)
}

func TestServePersonSpouseDetail(t *testing.T) {
	srv := newServeTestServer(t)
	var got personDetailDTO
	doServeRequest(t, srv, "/api/persons/person-dad", &got)

	require.Len(t, got.Family.Spouses, 1)
	assert.Equal(t, "person-mom", got.Family.Spouses[0].ID)
	assert.Equal(t, "Mary Lane", got.Family.Spouses[0].Name)
	// Marriage date flows from the relationship's start_event into the detail.
	assert.Contains(t, got.Family.Spouses[0].Detail, "1845")
}

func TestServePersonLivingFlag(t *testing.T) {
	srv := newServeTestServer(t)
	var got personDetailDTO
	doServeRequest(t, srv, "/api/persons/person-grandpa", &got)
	// Grandpa has a recorded death event, so he is not living.
	assert.False(t, got.Living)
}

func TestServePersonNotFound(t *testing.T) {
	srv := newServeTestServer(t)
	rec := doServeRequest(t, srv, "/api/persons/person-missing", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServeTreeAncestors(t *testing.T) {
	srv := newServeTestServer(t)
	var got struct {
		Direction string       `json:"direction"`
		Gen       int          `json:"gen"`
		Root      *treeNodeDTO `json:"root"`
	}
	rec := doServeRequest(t, srv, "/api/persons/person-self/tree?dir=ancestors&gen=3", &got)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ancestors", got.Direction)
	assert.Equal(t, 3, got.Gen)
	require.NotNil(t, got.Root)
	assert.Equal(t, "person-self", got.Root.ID)

	// Parents at depth 1.
	assert.ElementsMatch(t, []string{"person-dad", "person-mom"}, childIDs(got.Root))

	// Grandpa at depth 2 (parent of dad).
	var dad *treeNodeDTO
	for _, c := range got.Root.Children {
		if c.ID == "person-dad" {
			dad = c
		}
	}
	require.NotNil(t, dad)
	assert.Equal(t, []string{"person-grandpa"}, childIDs(dad))
}

func TestServeTreeDescendants(t *testing.T) {
	srv := newServeTestServer(t)
	var got struct {
		Direction string       `json:"direction"`
		Root      *treeNodeDTO `json:"root"`
	}
	doServeRequest(t, srv, "/api/persons/person-dad/tree?dir=descendants&gen=4", &got)

	assert.Equal(t, "descendants", got.Direction)
	require.NotNil(t, got.Root)
	// Dad's children: self and sib.
	assert.ElementsMatch(t, []string{"person-self", "person-sib"}, childIDs(got.Root))
}

// TestServeTreeChildOrderUnknownYearLast verifies the descendancy chart orders
// children by birth year with unknown-year children last — matching findChildIDs
// (the order glx summary uses), not year-0-first.
func TestServeTreeChildOrderUnknownYearLast(t *testing.T) {
	a := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-p":       {Properties: map[string]any{"name": "Parent"}},
			"person-dated":   {Properties: map[string]any{"name": "Dated Child"}},
			"person-undated": {Properties: map[string]any{"name": "Undated Child"}},
		},
		Events: map[string]*glxlib.Event{
			"event-b": {Type: "birth", Date: "1850", Participants: []glxlib.Participant{{Person: "person-dated", Role: "principal"}}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"relationship-pc": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-p", Role: "parent"},
				{Person: "person-undated", Role: "child"},
				{Person: "person-dated", Role: "child"},
			}},
		},
	}
	sub, err := fs.Sub(webAssets, "web")
	require.NoError(t, err)
	srv := &viewerServer{archive: a, archivePath: "t", assets: sub}

	var got struct {
		Root *treeNodeDTO `json:"root"`
	}
	doServeRequest(t, srv, "/api/persons/person-p/tree?dir=descendants&gen=2", &got)

	require.NotNil(t, got.Root)
	require.Equal(t, []string{"person-dated", "person-undated"}, childIDs(got.Root))
}

func TestServeTreeGenClampedToDefault(t *testing.T) {
	srv := newServeTestServer(t)
	var got struct {
		Gen int `json:"gen"`
	}
	doServeRequest(t, srv, "/api/persons/person-self/tree", &got)
	assert.Equal(t, serveDefaultTreeGen, got.Gen)
}

func TestServeSourcesList(t *testing.T) {
	srv := newServeTestServer(t)
	var got struct {
		Sources []sourceListItemDTO `json:"sources"`
	}
	doServeRequest(t, srv, "/api/sources", &got)

	require.Len(t, got.Sources, 2)
	var census *sourceListItemDTO
	for i := range got.Sources {
		if got.Sources[i].ID == "source-census" {
			census = &got.Sources[i]
		}
	}
	require.NotNil(t, census)
	assert.Equal(t, "1860 US Census", census.Title)
	assert.Equal(t, 1, census.CitationCount)

	// A titleless source falls back to its ID for display.
	var empty *sourceListItemDTO
	for i := range got.Sources {
		if got.Sources[i].ID == "source-empty" {
			empty = &got.Sources[i]
		}
	}
	require.NotNil(t, empty)
	assert.Equal(t, "source-empty", empty.Title)
}

func TestServeSourceDetail(t *testing.T) {
	srv := newServeTestServer(t)
	var got sourceDetailDTO
	doServeRequest(t, srv, "/api/sources/source-census", &got)

	assert.Equal(t, "1860 US Census", got.Title)
	assert.Equal(t, "census", got.Type)
	require.Len(t, got.Citations, 1)
	assert.Equal(t, "p. 12", got.Citations[0].Locator)
	assert.Equal(t, "Jane Webb, age 10", got.Citations[0].Text)
	assert.Equal(t, "2026-01-01", got.Citations[0].Accessed)
	// Citation has no own URL, so it inherits the source URL.
	assert.Equal(t, "https://example.com/census", got.Citations[0].URL)
}

func TestServeSourceNotFound(t *testing.T) {
	srv := newServeTestServer(t)
	rec := doServeRequest(t, srv, "/api/sources/source-missing", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServeStaticIndex(t *testing.T) {
	srv := newServeTestServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "GLX")
	assert.Contains(t, rec.Body.String(), "app.js")
}

func TestIsLoopbackHost(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "::1", "127.0.0.5"} {
		assert.True(t, isLoopbackHost(host), host)
	}
	// Empty host binds all interfaces (":port"), so it must NOT be loopback.
	for _, host := range []string{"", "0.0.0.0", "192.168.1.10", "example.com"} {
		assert.False(t, isLoopbackHost(host), host)
	}
}

// TestServeHandlesNilEntries ensures a malformed archive with nil map values
// (e.g. `person-id: null` in YAML) does not panic any endpoint.
func TestServeHandlesNilEntries(t *testing.T) {
	a := serveTestArchive()
	a.Persons["person-nil"] = nil
	a.Events["event-nil"] = nil
	a.Assertions["assertion-nil"] = nil
	a.Sources["source-nil"] = nil
	a.Citations["citation-nil"] = nil
	a.Relationships["relationship-nil"] = nil

	sub, err := fs.Sub(webAssets, "web")
	require.NoError(t, err)
	srv := &viewerServer{archive: a, archivePath: "nil-archive", assets: sub}

	// List/overview/tree endpoints iterate the maps (including the nil entries).
	for _, path := range []string{
		"/api/overview",
		"/api/persons",
		"/api/persons/person-self",
		"/api/persons/person-self/tree?dir=ancestors&gen=4",
		"/api/persons/person-self/tree?dir=descendants&gen=4",
		"/api/sources",
		"/api/sources/source-census",
	} {
		rec := doServeRequest(t, srv, path, nil)
		assert.Equalf(t, http.StatusOK, rec.Code, "path %s should not panic/error", path)
	}

	// A nil entity addressed directly resolves to 404, not a panic.
	assert.Equal(t, http.StatusNotFound, doServeRequest(t, srv, "/api/persons/person-nil", nil).Code)
	assert.Equal(t, http.StatusNotFound, doServeRequest(t, srv, "/api/sources/source-nil", nil).Code)
}

func TestViewerURL(t *testing.T) {
	addr := &net.TCPAddr{IP: net.IPv4zero, Port: 8080} // "0.0.0.0:8080"
	// Wildcard / empty hosts are not browser-usable, so loopback is substituted.
	assert.Equal(t, "http://127.0.0.1:8080", viewerURL("0.0.0.0", addr))
	assert.Equal(t, "http://127.0.0.1:8080", viewerURL("", addr))
	assert.Equal(t, "http://127.0.0.1:8080", viewerURL("::", addr))
	// An explicit, routable host is preserved.
	assert.Equal(t, "http://192.168.1.5:8080", viewerURL("192.168.1.5", addr))
	assert.Equal(t, "http://127.0.0.1:8080", viewerURL("127.0.0.1", addr))
}

func TestServeStaticHEAD(t *testing.T) {
	srv := newServeTestServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestServeStaticRejectsNonGetHead(t *testing.T) {
	srv := newServeTestServer(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Equal(t, "GET, HEAD", rec.Header().Get("Allow"))
}

func TestClampGen(t *testing.T) {
	assert.Equal(t, serveDefaultTreeGen, clampGen(""))
	assert.Equal(t, serveDefaultTreeGen, clampGen("0"))
	assert.Equal(t, serveDefaultTreeGen, clampGen("-3"))
	assert.Equal(t, serveDefaultTreeGen, clampGen("abc"))
	assert.Equal(t, 5, clampGen("5"))
	assert.Equal(t, serveMaxTreeGen, clampGen("999"))
}

func TestParticipantRole(t *testing.T) {
	participants := []glxlib.Participant{
		{Person: "person-a", Role: "principal"},
		{Person: "person-b"},
	}
	assert.Equal(t, "principal", participantRole("person-a", participants))
	assert.Equal(t, "participant", participantRole("person-b", participants))
	assert.Equal(t, roleAbsent, participantRole("person-c", participants))
}

// refIDs extracts the IDs from a list of person references.
func refIDs(refs []personRefDTO) []string {
	ids := make([]string, 0, len(refs))
	for _, r := range refs {
		ids = append(ids, r.ID)
	}

	return ids
}

// childIDs extracts the child IDs from a tree node.
func childIDs(node *treeNodeDTO) []string {
	ids := make([]string, 0, len(node.Children))
	for _, c := range node.Children {
		ids = append(ids, c.ID)
	}

	return ids
}
