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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	glxlib "github.com/genealogix/glx/go-glx"
)

// serveMaxEditBody bounds a write request's JSON body. The largest legitimate
// payload is a person form, a few hundred bytes.
const serveMaxEditBody = 64 << 10

// Relations the "add relative" form offers, named from the viewed person's
// side: "parent" means the relative is a parent of the person.
const (
	serveRelationParent = "parent"
	serveRelationChild  = "child"
	serveRelationSpouse = "spouse"
)

// requireEditableArchive refuses --edit for anything but a directory archive.
// The editor rewrites one entity file per change; a single-file archive would
// need the whole file rewritten, which this phase does not do.
func requireEditableArchive(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return ErrServeEditNeedsDirectory
	}

	return nil
}

// editRoutes registers the write API (issue #93, Phase 2). Creates go through
// the same code as `glx add person` / `glx add relationship`, so the editor
// refuses exactly what the CLI refuses and writes the same files.
func (s *viewerServer) editRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/edit/options", s.readLocked(s.handleEditOptions))
	mux.HandleFunc("GET /api/persons/{id}/edit", s.readLocked(s.handlePersonEditForm))
	mux.HandleFunc("POST /api/persons", s.writeLocked(s.handleCreatePerson))
	mux.HandleFunc("PUT /api/persons/{id}", s.writeLocked(s.handleUpdatePerson))
	mux.HandleFunc("POST /api/relationships", s.writeLocked(s.handleCreateRelationship))
}

// writeLocked guards a write handler: it accepts only a same-origin JSON
// request from a loopback or IP-literal host, then runs the handler under the
// exclusive lock.
//
// The checks are what keep a web page in the user's browser from editing the
// archive. A cross-site form post cannot send application/json without a CORS
// preflight the server never answers; a cross-site fetch carries a foreign
// Origin; and a DNS-rebinding page, whose Origin and Host agree, is caught by
// requiring the Host to be loopback or an IP literal rather than a name the
// attacker controls.
func (s *viewerServer) writeLocked(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if msg := rejectCrossSiteWrite(r); msg != "" {
			writeJSONError(w, http.StatusForbidden, msg)

			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeJSONError(w, http.StatusUnsupportedMediaType, "write requests must be application/json")

			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, serveMaxEditBody)

		s.mu.Lock()
		defer s.mu.Unlock()
		handler(w, r)
	}
}

// rejectCrossSiteWrite returns a reason to refuse the request, or "" to allow
// it. See writeLocked for why each check exists.
func rejectCrossSiteWrite(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(r.Host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host != "localhost" && net.ParseIP(host) == nil {
		return "write requests must address the server by localhost or an IP address"
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return "write requests must carry an Origin header"
	}
	if origin != "http://"+r.Host {
		return "cross-origin write request refused"
	}

	return ""
}

// decodeEditRequest reads a JSON body into dst, rejecting unknown fields so a
// typo in a client never silently drops an edit.
func decodeEditRequest(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	return nil
}

// reloadArchive replaces the in-memory snapshot with the archive as it now
// stands on disk. Called under the exclusive lock after every write. It reads
// with LoadArchiveWithOptions rather than the binary cache: the write has just
// changed the files, and a fresh parse is what the cache would fall back to.
func (s *viewerServer) reloadArchive() error {
	archive, _, err := LoadArchiveWithOptions(s.archivePath, false)
	if err != nil {
		return fmt.Errorf("reloading archive after write: %w", err)
	}
	s.archive = archive
	s.vitalOnce = sync.Once{}
	s.vitalIndex = nil

	return nil
}

// ============================================================================
// Edit options (vocabularies the forms offer)
// ============================================================================

type vocabOptionDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type editOptionsDTO struct {
	Sex               []vocabOptionDTO `json:"sex"`
	Gender            []vocabOptionDTO `json:"gender"`
	RelationshipTypes []vocabOptionDTO `json:"relationshipTypes"`
}

func (s *viewerServer) handleEditOptions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, editOptionsDTO{
		Sex:               vocabOptions(s.archive.SexTypes),
		Gender:            vocabOptions(s.archive.GenderTypes),
		RelationshipTypes: vocabOptions(s.archive.RelationshipTypes),
	})
}

// vocabOptions lists a vocabulary's keys with their labels, sorted by key.
func vocabOptions(vocab map[string]*glxlib.VocabularyEntry) []vocabOptionDTO {
	out := make([]vocabOptionDTO, 0, len(vocab))
	for _, key := range sortedKeys(vocab) {
		label := key
		if entry := vocab[key]; entry != nil && entry.Label != "" {
			label = entry.Label
		}
		out = append(out, vocabOptionDTO{Key: key, Label: label})
	}

	return out
}

// ============================================================================
// Person form
// ============================================================================

// personNameDTO is the structured name the form edits, one field per
// glxlib.NameField* key.
type personNameDTO struct {
	Prefix        string `json:"prefix"`
	Given         string `json:"given"`
	Nickname      string `json:"nickname"`
	SurnamePrefix string `json:"surnamePrefix"`
	Surname       string `json:"surname"`
	Suffix        string `json:"suffix"`
}

// personEditDTO is the person form's current state. Version is a hash of the
// person's file; the update must send it back, so an edit made to the file
// in the meantime (by the CLI, a text editor or a git checkout) is refused
// instead of overwritten.
type personEditDTO struct {
	ID             string        `json:"id"`
	Version        string        `json:"version"`
	Name           personNameDTO `json:"name"`
	NameValue      string        `json:"nameValue"`
	NameStructured bool          `json:"nameStructured"`
	NameEditable   bool          `json:"nameEditable"`
	Sex            string        `json:"sex"`
	SexEditable    bool          `json:"sexEditable"`
	Gender         string        `json:"gender"`
	GenderEditable bool          `json:"genderEditable"`
}

type personUpdateRequest struct {
	Version string        `json:"version"`
	Name    personNameDTO `json:"name"`
	Sex     string        `json:"sex"`
	Gender  string        `json:"gender"`
}

func (s *viewerServer) handlePersonEditForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if person, ok := s.archive.Persons[id]; !ok || person == nil {
		writeJSONError(w, http.StatusNotFound, "person not found: "+id)

		return
	}

	stored, err := s.readStandalonePerson(id)
	if err != nil {
		writeEditError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, personEditForm(id, stored))
}

// personEditForm builds the form state from the person as stored on disk.
func personEditForm(id string, stored *storedPerson) personEditDTO {
	props := stored.person.Properties
	name, nameEditable := currentNameFields(props[glxlib.PersonPropertyName])
	sex, sexEditable := simpleProperty(props, glxlib.PersonPropertySex)
	gender, genderEditable := simpleProperty(props, glxlib.PersonPropertyGender)

	return personEditDTO{
		ID:             id,
		Version:        stored.version,
		Name:           name,
		NameValue:      glxlib.GetFullName(props[glxlib.PersonPropertyName]),
		NameStructured: nameIsStructured(props[glxlib.PersonPropertyName]),
		NameEditable:   nameEditable,
		Sex:            sex,
		SexEditable:    sexEditable,
		Gender:         gender,
		GenderEditable: genderEditable,
	}
}

// currentNameFields reads the name property into form fields. A structured
// name supplies its fields; an unstructured one (a plain string, or a value
// with no fields) yields empty fields, because guessing where "Johann Georg
// von Schöpff" splits would put words in the archive nobody wrote. The form
// shows the stored value beside them instead, and filling the fields in
// replaces it. A temporal list of names is reported as not editable: the form
// has one name, and choosing which entry it edits is a later phase.
func currentNameFields(raw any) (personNameDTO, bool) {
	if list, isList := raw.([]any); isList {
		var first any
		if len(list) > 0 {
			first = list[0]
		}
		fields, _ := currentNameFields(first)

		return fields, false
	}
	if m, ok := raw.(map[string]any); ok {
		if fields, ok := m["fields"].(map[string]any); ok {
			return personNameDTO{
				Prefix:        stringField(fields, glxlib.NameFieldPrefix),
				Given:         stringField(fields, glxlib.NameFieldGiven),
				Nickname:      stringField(fields, glxlib.NameFieldNickname),
				SurnamePrefix: stringField(fields, glxlib.NameFieldSurnamePrefix),
				Surname:       stringField(fields, glxlib.NameFieldSurname),
				Suffix:        stringField(fields, glxlib.NameFieldSuffix),
			}, true
		}
	}

	return personNameDTO{}, true
}

// nameIsStructured reports whether a single name carries the fields the form
// edits.
func nameIsStructured(raw any) bool {
	m, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	_, ok = m["fields"].(map[string]any)

	return ok
}

// simpleProperty returns a property's value and whether the form can edit it:
// an absent or plain string value can be, a temporal or structured one cannot.
func simpleProperty(props map[string]any, key string) (string, bool) {
	raw, ok := props[key]
	if !ok || raw == nil {
		return "", true
	}
	if s, ok := raw.(string); ok {
		return s, true
	}

	return propertyString(props, key), false
}

// ============================================================================
// Reading and writing one person's file
// ============================================================================

// storedPerson is a person read straight from its own file.
type storedPerson struct {
	relPath string
	person  *glxlib.Person
	version string
}

// readStandalonePerson reads persons/<id>.glx and confirms it holds this
// person and nothing else. That is the only layout the editor rewrites: a
// person kept in a hand-named file, or sharing a file with other entities,
// would otherwise be duplicated or have its neighbors dropped.
func (s *viewerServer) readStandalonePerson(id string) (*storedPerson, error) {
	name, err := glxlib.EntityIDToFilename(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAddInvalidID, id)
	}
	relPath := glxlib.EntityTypePersons.String() + "/" + name

	data, err := os.ReadFile(filepath.Join(s.archivePath, filepath.FromSlash(relPath)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrServePersonNotStandalone
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", relPath, err)
	}

	doc, err := ParseYAMLFile(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", relPath, err)
	}
	if len(doc) != 1 {
		return nil, ErrServePersonNotStandalone
	}
	persons, ok := doc[glxlib.EntityTypePersons.String()].(map[string]any)
	if !ok || len(persons) != 1 {
		return nil, ErrServePersonNotStandalone
	}
	if _, ok := persons[id]; !ok {
		return nil, ErrServePersonNotStandalone
	}

	file, _, err := createSerializer(false, false, "").DeserializeMultiFileFromMap(map[string][]byte{relPath: data})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", relPath, err)
	}
	person := file.Persons[id]
	if person == nil {
		return nil, ErrServePersonNotStandalone
	}

	sum := sha256.Sum256(data)

	return &storedPerson{relPath: relPath, person: person, version: hex.EncodeToString(sum[:])}, nil
}

func (s *viewerServer) handleUpdatePerson(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if person, ok := s.archive.Persons[id]; !ok || person == nil {
		writeJSONError(w, http.StatusNotFound, "person not found: "+id)

		return
	}

	var req personUpdateRequest
	if err := decodeEditRequest(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())

		return
	}

	if err := s.updatePerson(id, &req); err != nil {
		writeEditError(w, err)

		return
	}
	if err := s.reloadArchive(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())

		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

// updatePerson applies a form edit to the person's file: re-read it, refuse
// if it changed since the form was loaded, apply the changed fields only,
// validate the whole archive with the change in place, and write the one
// file atomically.
func (s *viewerServer) updatePerson(id string, req *personUpdateRequest) error {
	stored, err := s.readStandalonePerson(id)
	if err != nil {
		return err
	}
	if req.Version != stored.version {
		return ErrServePersonChangedOnDisk
	}

	// Validation needs the whole archive as it is on disk now, with its
	// vocabularies, not the snapshot the page was rendered from.
	archive, _, err := LoadArchiveWithOptions(s.archivePath, false)
	if err != nil {
		return fmt.Errorf("loading archive: %w", err)
	}
	req.Sex = strings.TrimSpace(req.Sex)
	req.Gender = strings.TrimSpace(req.Gender)
	if err := validateVocabKey(archive, glxlib.VocabSexTypes, req.Sex); err != nil {
		return err
	}
	if err := validateVocabKey(archive, glxlib.VocabGenderTypes, req.Gender); err != nil {
		return err
	}

	updated, err := applyPersonEdit(stored.person, req)
	if err != nil {
		return err
	}

	partial := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{id: updated}}
	if err := runWholeArchiveValidate(archive, partial); err != nil {
		return err
	}

	files, err := createSerializer(false, true, "  ").SerializeMultiFileToMap(partial)
	if err != nil {
		return fmt.Errorf("serializing person: %w", err)
	}
	data, ok := files[stored.relPath]
	if !ok {
		return fmt.Errorf("%w: no output for %s", ErrServeSerializePerson, stored.relPath)
	}

	return atomicWriteFile(filepath.Join(s.archivePath, filepath.FromSlash(stored.relPath)), data, filePermissions)
}

// applyPersonEdit returns a copy of person with the form's changes applied.
// A field the form did not change is left exactly as stored, so editing the
// sex never rewrites a name the form could only approximate (a plain string
// name it split into given and surname, say).
func applyPersonEdit(person *glxlib.Person, req *personUpdateRequest) (*glxlib.Person, error) {
	updated := *person
	updated.Properties = maps.Clone(person.Properties)
	if updated.Properties == nil {
		updated.Properties = map[string]any{}
	}
	props := updated.Properties

	req.Name = trimName(&req.Name)
	currentName, nameEditable := currentNameFields(props[glxlib.PersonPropertyName])
	if req.Name != currentName {
		if !nameEditable {
			return nil, ErrServeTemporalName
		}
		if req.Name.Given == "" && req.Name.Surname == "" {
			return nil, ErrServePersonNameRequired
		}
		props[glxlib.PersonPropertyName] = editedNameProperty(props[glxlib.PersonPropertyName], &req.Name)
	}

	for _, field := range []struct{ key, value string }{
		{glxlib.PersonPropertySex, req.Sex},
		{glxlib.PersonPropertyGender, req.Gender},
	} {
		current, editable := simpleProperty(props, field.key)
		if field.value == current {
			continue
		}
		if !editable {
			return nil, fmt.Errorf("%w: %s", ErrServePropertyNotSimple, field.key)
		}
		if field.value == "" {
			delete(props, field.key)
		} else {
			props[field.key] = field.value
		}
	}

	if len(props) == 0 {
		updated.Properties = nil
	}

	return &updated, nil
}

func trimName(n *personNameDTO) personNameDTO {
	return personNameDTO{
		Prefix:        strings.TrimSpace(n.Prefix),
		Given:         strings.TrimSpace(n.Given),
		Nickname:      strings.TrimSpace(n.Nickname),
		SurnamePrefix: strings.TrimSpace(n.SurnamePrefix),
		Surname:       strings.TrimSpace(n.Surname),
		Suffix:        strings.TrimSpace(n.Suffix),
	}
}

// editedNameProperty builds the new name value. A structured name keeps every
// key the form does not edit (a name type, say); a plain string name becomes
// structured. The display value is rebuilt from the parts exactly as
// `glx add person` builds it.
func editedNameProperty(raw any, name *personNameDTO) map[string]any {
	opts := personOptionsFromName(name)
	built := buildNameProperty(opts)

	existing, ok := raw.(map[string]any)
	if !ok {
		return built
	}
	out := maps.Clone(existing)
	fields := map[string]any{}
	if oldFields, ok := existing["fields"].(map[string]any); ok {
		fields = maps.Clone(oldFields)
	}
	for _, key := range []string{
		glxlib.NameFieldPrefix, glxlib.NameFieldGiven, glxlib.NameFieldNickname,
		glxlib.NameFieldSurnamePrefix, glxlib.NameFieldSurname, glxlib.NameFieldSuffix,
	} {
		delete(fields, key)
	}
	if builtFields, ok := built["fields"].(map[string]any); ok {
		maps.Copy(fields, builtFields)
	}
	out["value"] = built["value"]
	out["fields"] = fields

	return out
}

func personOptionsFromName(name *personNameDTO) *addPersonOptions {
	return &addPersonOptions{
		Prefix:        name.Prefix,
		Given:         name.Given,
		Nickname:      name.Nickname,
		SurnamePrefix: name.SurnamePrefix,
		Surname:       name.Surname,
		Suffix:        name.Suffix,
	}
}

// ============================================================================
// Creating persons and relationships (through the `glx add` code)
// ============================================================================

type personCreateRequest struct {
	Name   personNameDTO `json:"name"`
	Sex    string        `json:"sex"`
	Gender string        `json:"gender"`
}

func (s *viewerServer) handleCreatePerson(w http.ResponseWriter, r *http.Request) {
	var req personCreateRequest
	if err := decodeEditRequest(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())

		return
	}
	req.Name = trimName(&req.Name)
	if req.Name.Given == "" && req.Name.Surname == "" {
		writeEditError(w, ErrServePersonNameRequired)

		return
	}

	opts := personOptionsFromName(&req.Name)
	opts.ArchivePath = s.archivePath
	opts.Sex = strings.TrimSpace(req.Sex)
	opts.Gender = strings.TrimSpace(req.Gender)

	s.finishCreate(w, func(streams *IOStreams) error { return addPerson(streams, opts) })
}

type relationshipCreateRequest struct {
	Type       string `json:"type"`
	PersonID   string `json:"personId"`
	RelativeID string `json:"relativeId"`
	Relation   string `json:"relation"`
}

func (s *viewerServer) handleCreateRelationship(w http.ResponseWriter, r *http.Request) {
	var req relationshipCreateRequest
	if err := decodeEditRequest(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())

		return
	}

	opts, err := relationshipOptions(&req)
	if err != nil {
		writeEditError(w, err)

		return
	}
	opts.ArchivePath = s.archivePath
	if existing := findSameRelationship(s.archive, opts); existing != "" {
		writeEditError(w, fmt.Errorf("%w: %s", ErrServeRelationshipExists, existing))

		return
	}

	s.finishCreate(w, func(streams *IOStreams) error { return addRelationship(streams, opts) })
}

// findSameRelationship returns the ID of a relationship that already records
// what opts would add (same type, same people in the same roles), or "".
// `glx add relationship` would mint a "-2" twin instead, which from a form is
// almost always a double-submit rather than a second relationship.
func findSameRelationship(archive *glxlib.GLXFile, opts *addRelationshipOptions) string {
	want := map[string]string{}
	for _, p := range opts.Parents {
		want[p] = glxlib.ParticipantRoleParent
	}
	for _, c := range opts.Children {
		want[c] = glxlib.ParticipantRoleChild
	}
	for _, sp := range opts.Spouses {
		want[sp] = glxlib.ParticipantRoleSpouse
	}

	for _, id := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[id]
		if rel == nil || rel.Type != opts.Type || len(rel.Participants) != len(want) {
			continue
		}
		same := true
		for _, p := range rel.Participants {
			if role, ok := want[p.Person]; !ok || role != p.Role {
				same = false

				break
			}
		}
		if same {
			return id
		}
	}

	return ""
}

// relationshipOptions turns the "add relative" form into `glx add
// relationship` options. The relation is read from the viewed person's side.
func relationshipOptions(req *relationshipCreateRequest) (*addRelationshipOptions, error) {
	person := strings.TrimSpace(req.PersonID)
	relative := strings.TrimSpace(req.RelativeID)
	if person == relative {
		return nil, ErrServeRelationshipSelfLink
	}

	opts := &addRelationshipOptions{Type: strings.TrimSpace(req.Type)}
	switch req.Relation {
	case serveRelationParent:
		opts.Parents = []string{relative}
		opts.Children = []string{person}
	case serveRelationChild:
		opts.Parents = []string{person}
		opts.Children = []string{relative}
	case serveRelationSpouse:
		opts.Spouses = []string{person, relative}
	default:
		return nil, ErrServeRelationUnknown
	}

	return opts, nil
}

// finishCreate runs a `glx add` function against the archive on disk, then
// reloads the snapshot and answers with the new entity's ID, which the add
// function prints on its machine-readable stream.
func (s *viewerServer) finishCreate(w http.ResponseWriter, add func(*IOStreams) error) {
	var out, machine, errOut bytes.Buffer
	streams := &IOStreams{Out: &out, MachineOut: &machine, ErrOut: &errOut}
	if err := add(streams); err != nil {
		writeEditError(w, err)

		return
	}
	if err := s.reloadArchive(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())

		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": strings.TrimSpace(machine.String())})
}

// ============================================================================
// Errors
// ============================================================================

// editConflictErrors are refusals about the archive's state rather than the
// request: the client cannot fix them by changing the form.
var editConflictErrors = []error{
	ErrServePersonNotStandalone,
	ErrServePersonChangedOnDisk,
	ErrServeTemporalName,
	ErrServePropertyNotSimple,
	ErrServeRelationshipExists,
}

// editInputErrors are refusals of the request itself, from the editor or from
// the `glx add` checks it reuses.
var editInputErrors = []error{
	ErrServePersonNameRequired,
	ErrServeRelationUnknown,
	ErrServeRelationshipSelfLink,
	ErrValidationFailed,
	ErrAddVocabKeyUnknown,
	ErrAddRefNotFound,
	ErrAddInvalidID,
	ErrAddEntityExists,
	ErrAddIDExhausted,
	ErrAddPersonNameRequired,
	ErrAddRelationshipTypeRequired,
	ErrAddRelationshipParticipantsRequired,
}

// writeEditError maps an edit failure to a status: 409 for archive state the
// form cannot fix, 422 for a request the archive rejects, 500 otherwise.
func writeEditError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errorIsAny(err, editConflictErrors):
		status = http.StatusConflict
	case errorIsAny(err, editInputErrors):
		status = http.StatusUnprocessableEntity
	}
	writeJSONError(w, status, err.Error())
}

func errorIsAny(err error, targets []error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}
