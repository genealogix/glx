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
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

const editTestHost = "127.0.0.1:8080"

// newEditTestServer writes serveTestArchive to a temp directory and serves it
// with editing enabled.
func newEditTestServer(t *testing.T) *viewerServer {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, writeMultiFileArchive(dir, serveTestArchive(), false))

	sub, err := fs.Sub(webAssets, "web")
	require.NoError(t, err)
	srv := &viewerServer{archivePath: dir, assets: sub, editable: true}
	require.NoError(t, srv.reloadArchive())

	return srv
}

// doEditRequest sends a same-origin JSON write request, the shape the UI sends.
func doEditRequest(t *testing.T, srv *viewerServer, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewReader(payload))
	req.Host = editTestHost
	req.Header.Set("Origin", "http://"+editTestHost)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, req)

	return rec
}

func editForm(t *testing.T, srv *viewerServer, id string) personEditDTO {
	t.Helper()
	var form personEditDTO
	rec := doServeRequest(t, srv, "/api/persons/"+id+"/edit", &form)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	return form
}

func readPersonFile(t *testing.T, srv *viewerServer, id string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(srv.archivePath, "persons", id+".glx"))
	require.NoError(t, err)

	return string(data)
}

func TestServeReadOnlyHasNoWriteAPI(t *testing.T) {
	srv := newEditTestServer(t)
	srv.editable = false

	rec := doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", map[string]any{})
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	rec = doServeRequest(t, srv, "/api/persons/person-dad/edit", nil)
	assert.NotEqual(t, http.StatusOK, rec.Code)

	var overview overviewDTO
	doServeRequest(t, srv, "/api/overview", &overview)
	assert.False(t, overview.Editable)
}

func TestServeEditOverviewAndOptions(t *testing.T) {
	srv := newEditTestServer(t)

	var overview overviewDTO
	doServeRequest(t, srv, "/api/overview", &overview)
	assert.True(t, overview.Editable)

	var opts editOptionsDTO
	rec := doServeRequest(t, srv, "/api/edit/options", &opts)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, optionKeys(opts.Sex), "female")
	assert.Contains(t, optionKeys(opts.RelationshipTypes), "biological_parent_child")
}

func TestServeEditFormReadsStoredPerson(t *testing.T) {
	srv := newEditTestServer(t)

	form := editForm(t, srv, "person-dad")
	// A plain string name is shown as-is, never split into guessed parts.
	assert.Equal(t, personNameDTO{}, form.Name)
	assert.Equal(t, "Robert Webb", form.NameValue)
	assert.False(t, form.NameStructured)
	assert.True(t, form.NameEditable)
	assert.Equal(t, "male", form.Sex)
	assert.NotEmpty(t, form.Version)

	// A temporal list of names cannot be edited through the one-name form.
	assert.False(t, editForm(t, srv, "person-self").NameEditable)
}

func TestServeEditSexLeavesNameUntouched(t *testing.T) {
	srv := newEditTestServer(t)
	form := editForm(t, srv, "person-dad")

	rec := doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", personUpdateRequest{
		Version: form.Version, Name: form.Name, Sex: "unknown",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The plain string name was split only for display, so it must survive as
	// the string it was.
	assert.Equal(t, "Robert Webb", srv.archive.Persons["person-dad"].Properties[glxlib.PersonPropertyName])
	assert.Equal(t, "unknown", srv.archive.Persons["person-dad"].Properties[glxlib.PersonPropertySex])
	assert.Contains(t, readPersonFile(t, srv, "person-dad"), "sex: unknown")
}

func TestServeEditNameBecomesStructured(t *testing.T) {
	srv := newEditTestServer(t)
	form := editForm(t, srv, "person-dad")
	form.Name = personNameDTO{Given: "Robert James", Surname: "Webb", Suffix: "Sr."}

	rec := doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", personUpdateRequest{
		Version: form.Version, Name: form.Name, Sex: form.Sex,
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var detail personDetailDTO
	doServeRequest(t, srv, "/api/persons/person-dad", &detail)
	assert.Equal(t, []string{"Robert James Webb Sr."}, detail.Names)

	after := editForm(t, srv, "person-dad")
	assert.Equal(t, form.Name, after.Name)
	assert.True(t, after.NameStructured)
	assert.NotEqual(t, form.Version, after.Version)
}

func TestServeEditKeepsUnformedNameKeys(t *testing.T) {
	existing := map[string]any{
		"value":  "Jane Webb",
		"fields": map[string]any{"given": "Jane", "surname": "Webb", "type": "birth"},
		"date":   "1850",
	}
	got := editedNameProperty(existing, &personNameDTO{Given: "Janet", Surname: "Webb"})

	assert.Equal(t, "Janet Webb", got["value"])
	assert.Equal(t, "1850", got["date"])
	assert.Equal(t, map[string]any{"given": "Janet", "surname": "Webb", "type": "birth"}, got["fields"])
	// The original map is not modified in place.
	assert.Equal(t, "Jane", existing["fields"].(map[string]any)["given"])
}

func TestServeEditRefusesStaleVersion(t *testing.T) {
	srv := newEditTestServer(t)
	form := editForm(t, srv, "person-dad")

	// Someone edits the file outside the server after the form loaded.
	path := filepath.Join(srv.archivePath, "persons", "person-dad.glx")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(data, []byte("    notes: edited elsewhere\n")...), 0o644))

	rec := doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", personUpdateRequest{
		Version: form.Version, Name: form.Name, Sex: "female",
	})
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, readPersonFile(t, srv, "person-dad"), "edited elsewhere")
}

func TestServeEditTemporalName(t *testing.T) {
	srv := newEditTestServer(t)
	form := editForm(t, srv, "person-self")

	changed := form.Name
	changed.Given = "Janet"
	rec := doEditRequest(t, srv, http.MethodPut, "/api/persons/person-self", personUpdateRequest{
		Version: form.Version, Name: changed, Sex: form.Sex,
	})
	assert.Equal(t, http.StatusConflict, rec.Code)

	// The other fields of the same person still save.
	rec = doEditRequest(t, srv, http.MethodPut, "/api/persons/person-self", personUpdateRequest{
		Version: form.Version, Name: form.Name, Sex: "unknown",
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.IsType(t, []any{}, srv.archive.Persons["person-self"].Properties[glxlib.PersonPropertyName])
}

func TestServeEditRejectsInput(t *testing.T) {
	srv := newEditTestServer(t)
	form := editForm(t, srv, "person-dad")

	rec := doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", personUpdateRequest{
		Version: form.Version, Name: form.Name, Sex: "not-a-sex",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", personUpdateRequest{
		Version: form.Version, Name: personNameDTO{Nickname: "Bob"}, Sex: form.Sex,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = doEditRequest(t, srv, http.MethodPut, "/api/persons/person-dad", map[string]any{"version": form.Version, "unknownField": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doEditRequest(t, srv, http.MethodPut, "/api/persons/person-nobody", personUpdateRequest{})
	assert.Equal(t, http.StatusNotFound, rec.Code)

	assert.Contains(t, readPersonFile(t, srv, "person-dad"), "sex: male")
}

func TestServeEditRefusesSharedFile(t *testing.T) {
	srv := newEditTestServer(t)
	shared := []byte("persons:\n  person-a:\n    properties:\n      name: Ann\n  person-b:\n    properties:\n      name: Bea\n")
	require.NoError(t, os.WriteFile(filepath.Join(srv.archivePath, "persons", "person-a.glx"), shared, 0o644))
	require.NoError(t, srv.reloadArchive())

	rec := doServeRequest(t, srv, "/api/persons/person-a/edit", nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
	rec = doServeRequest(t, srv, "/api/persons/person-b/edit", nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestServeCreatePerson(t *testing.T) {
	srv := newEditTestServer(t)

	rec := doEditRequest(t, srv, http.MethodPost, "/api/persons", personCreateRequest{
		Name: personNameDTO{Given: "Ada", Surname: "Webb"}, Sex: "female",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	id := created["id"]
	require.NotEmpty(t, id)

	require.Contains(t, srv.archive.Persons, id)
	assert.Contains(t, readPersonFile(t, srv, id), "given: Ada")

	rec = doEditRequest(t, srv, http.MethodPost, "/api/persons", personCreateRequest{Sex: "female"})
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestServeCreateRelationship(t *testing.T) {
	srv := newEditTestServer(t)

	rec := doEditRequest(t, srv, http.MethodPost, "/api/relationships", relationshipCreateRequest{
		Type: "biological_parent_child", PersonID: "person-grandpa", RelativeID: "person-mom", Relation: "child",
	})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var detail personDetailDTO
	doServeRequest(t, srv, "/api/persons/person-mom", &detail)
	assert.Contains(t, refIDs(detail.Family.Parents), "person-grandpa")

	// Submitting the same form again is refused rather than minting a twin.
	before := len(srv.archive.Relationships)
	rec = doEditRequest(t, srv, http.MethodPost, "/api/relationships", relationshipCreateRequest{
		Type: "biological_parent_child", PersonID: "person-mom", RelativeID: "person-grandpa", Relation: "parent",
	})
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Len(t, srv.archive.Relationships, before)

	for name, req := range map[string]relationshipCreateRequest{
		"unknown relation": {Type: "marriage", PersonID: "person-dad", RelativeID: "person-mom", Relation: "cousin"},
		"self":             {Type: "marriage", PersonID: "person-dad", RelativeID: "person-dad", Relation: "spouse"},
		"missing person":   {Type: "marriage", PersonID: "person-dad", RelativeID: "person-nobody", Relation: "spouse"},
		"unknown type":     {Type: "betrothal", PersonID: "person-dad", RelativeID: "person-mom", Relation: "spouse"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := doEditRequest(t, srv, http.MethodPost, "/api/relationships", req)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		})
	}
}

func TestServeWriteRefusesCrossSite(t *testing.T) {
	srv := newEditTestServer(t)
	body := []byte(`{"name":{"given":"Eve"}}`)

	for name, tc := range map[string]struct {
		host, origin, contentType string
		want                      int
	}{
		"foreign origin": {editTestHost, "http://evil.example", "application/json", http.StatusForbidden},
		"no origin":      {editTestHost, "", "application/json", http.StatusForbidden},
		"dns rebinding":  {"evil.example:8080", "http://evil.example:8080", "application/json", http.StatusForbidden},
		"form post":      {editTestHost, "http://" + editTestHost, "application/x-www-form-urlencoded", http.StatusUnsupportedMediaType},
		"localhost name": {"localhost:8080", "http://localhost:8080", "application/json", http.StatusCreated},
		"ipv6 loopback":  {"[::1]:8080", "http://[::1]:8080", "application/json; charset=utf-8", http.StatusCreated},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/persons", bytes.NewReader(body))
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			srv.routes().ServeHTTP(rec, req)
			assert.Equal(t, tc.want, rec.Code, rec.Body.String())
		})
	}
}

func TestRequireEditableArchive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, requireEditableArchive(dir))

	file := filepath.Join(dir, "family.glx")
	require.NoError(t, os.WriteFile(file, []byte("persons: {}\n"), 0o644))
	assert.ErrorIs(t, requireEditableArchive(file), ErrServeEditNeedsDirectory)
}

func optionKeys(opts []vocabOptionDTO) []string {
	keys := make([]string, 0, len(opts))
	for _, o := range opts {
		keys = append(keys, o.Key)
	}

	return keys
}
