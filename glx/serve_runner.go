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
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/signal"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

// webAssets holds the embedded browser UI for `glx serve`. The viewer is a
// self-contained, dependency-free static bundle (HTML + vanilla JS + CSS) so
// the binary needs no build step and no network access to render.
//
//go:embed web
var webAssets embed.FS

// serveDefaultTreeGen / serveMaxTreeGen bound pedigree/descendancy depth so a
// malicious or cyclic archive cannot drive unbounded recursion. The default
// depth balances a useful chart against payload size.
const (
	serveDefaultPort    = 8080
	serveDefaultTreeGen = 4
	serveMaxTreeGen     = 10
	serveReadTimeout    = 10 * time.Second
	serveShutdownGrace  = 5 * time.Second
)

// serveOptions configures the read-only archive viewer server.
type serveOptions struct {
	ArchivePath string
	Host        string
	Port        int
}

// viewerServer serves a read-only browser UI and JSON API over an in-memory
// snapshot of a GLX archive. The archive is loaded once at startup; this is a
// Phase 1 read-only viewer (issue #93), so external edits are picked up on
// restart. Live file watching is a later phase.
type viewerServer struct {
	archive     *glxlib.GLXFile
	archivePath string
	assets      fs.FS
}

// serveArchive loads an archive and serves the browser viewer until the process
// is interrupted (Ctrl-C / SIGTERM). It is the thin command body behind
// `glx serve`.
func serveArchive(streams *IOStreams, opts serveOptions) error {
	archive, err := loadArchiveForSummary(opts.ArchivePath)
	if err != nil {
		return err
	}

	sub, err := fs.Sub(webAssets, "web")
	if err != nil {
		return fmt.Errorf("loading embedded web assets: %w", err)
	}

	srv := &viewerServer{
		archive:     archive,
		archivePath: opts.ArchivePath,
		assets:      sub,
	}

	// Interrupt triggers a graceful shutdown so in-flight requests complete.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s: %w", addr, err)
	}

	httpSrv := &http.Server{
		Handler:           srv.routes(),
		ReadHeaderTimeout: serveReadTimeout,
	}

	go func() {
		<-ctx.Done()
		// Derive from ctx (so the shutdown context inherits its values) but drop
		// its cancellation — ctx is already done here, and the grace period needs
		// its own fresh deadline.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serveShutdownGrace)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	url := "http://" + listener.Addr().String()
	streams.Printf("GLX viewer for %s\n", opts.ArchivePath)
	streams.Printf("  %d persons, %d sources, %d events\n",
		len(archive.Persons), len(archive.Sources), len(archive.Events))
	streams.Printf("\n  Open this URL in your browser: %s\n  Press Ctrl-C to stop.\n\n", url)
	if !isLoopbackHost(opts.Host) {
		hostLabel := opts.Host
		if hostLabel == "" {
			// net.JoinHostPort("", port) binds every interface (":port").
			hostLabel = "all interfaces"
		}
		streams.Errorf("Warning: binding to %s exposes this archive to your network. "+
			"Use --host 127.0.0.1 to keep it local.\n", hostLabel)
	}

	if err := httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	streams.Printf("Viewer stopped.\n")

	return nil
}

// isLoopbackHost reports whether host binds only to the local machine. An empty
// host is NOT loopback: net.JoinHostPort("", port) produces ":port", which binds
// every interface — so `--host ""` must still trigger the exposure warning.
func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	case "":
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}

// routes wires the JSON API and the embedded static UI. API patterns are more
// specific than the catch-all file server, so Go's ServeMux precedence routes
// /api/* to the handlers and everything else to the bundled assets.
func (s *viewerServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/overview", s.handleOverview)
	mux.HandleFunc("GET /api/persons", s.handlePersons)
	mux.HandleFunc("GET /api/persons/{id}", s.handlePerson)
	mux.HandleFunc("GET /api/persons/{id}/tree", s.handleTree)
	mux.HandleFunc("GET /api/sources", s.handleSources)
	mux.HandleFunc("GET /api/sources/{id}", s.handleSource)
	mux.Handle("GET /", http.FileServerFS(s.assets))

	return mux
}

// ============================================================================
// JSON helpers
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ============================================================================
// Overview (dashboard)
// ============================================================================

type overviewDTO struct {
	ArchivePath string             `json:"archivePath"`
	Counts      map[string]int     `json:"counts"`
	Confidence  []confidenceRowDTO `json:"confidence"`
	Coverage    []coverageRowDTO   `json:"coverage"`
}

type confidenceRowDTO struct {
	Level   string  `json:"level"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

type coverageRowDTO struct {
	Label   string  `json:"label"`
	Covered int     `json:"covered"`
	Total   int     `json:"total"`
	Percent float64 `json:"percent"`
}

func (s *viewerServer) handleOverview(w http.ResponseWriter, _ *http.Request) {
	a := s.archive
	// Keyed by the canonical entity-type strings (e.g. "persons") so the frontend
	// can map them to labels; built from the typed constants to avoid literals.
	counts := map[string]int{
		string(glxlib.EntityTypePersons):       len(a.Persons),
		string(glxlib.EntityTypeEvents):        len(a.Events),
		string(glxlib.EntityTypeRelationships): len(a.Relationships),
		string(glxlib.EntityTypePlaces):        len(a.Places),
		string(glxlib.EntityTypeSources):       len(a.Sources),
		string(glxlib.EntityTypeCitations):     len(a.Citations),
		string(glxlib.EntityTypeRepositories):  len(a.Repositories),
		string(glxlib.EntityTypeAssertions):    len(a.Assertions),
		string(glxlib.EntityTypeMedia):         len(a.Media),
	}

	writeJSON(w, http.StatusOK, overviewDTO{
		ArchivePath: s.archivePath,
		Counts:      counts,
		Confidence:  confidenceRows(a),
		Coverage:    coverageRows(a),
	})
}

// confidenceRows mirrors the ordering used by `glx stats`: standard levels
// first (high, medium, low), then any custom levels alphabetically, then unset.
func confidenceRows(a *glxlib.GLXFile) []confidenceRowDTO {
	if len(a.Assertions) == 0 {
		return []confidenceRowDTO{}
	}

	const unset = "(unset)"
	counts := map[string]int{}
	valid := 0
	for _, assertion := range a.Assertions {
		if assertion == nil {
			continue
		}
		valid++
		level := assertion.Confidence
		if level == "" {
			level = unset
		}
		counts[level]++
	}
	if valid == 0 {
		return []confidenceRowDTO{}
	}

	var levels []string
	seen := map[string]bool{}
	for _, level := range []string{
		glxlib.ConfidenceLevelHigh, glxlib.ConfidenceLevelMedium, glxlib.ConfidenceLevelLow,
	} {
		if counts[level] > 0 {
			levels = append(levels, level)
			seen[level] = true
		}
	}
	var custom []string
	for level := range counts {
		if !seen[level] && level != unset {
			custom = append(custom, level)
		}
	}
	sort.Strings(custom)
	levels = append(levels, custom...)
	if counts[unset] > 0 {
		levels = append(levels, unset)
	}

	total := float64(valid)
	rows := make([]confidenceRowDTO, 0, len(levels))
	for _, level := range levels {
		rows = append(rows, confidenceRowDTO{
			Level:   level,
			Count:   counts[level],
			Percent: float64(counts[level]) / total * 100,
		})
	}

	return rows
}

// coverageRows reports how many persons/events/relationships/places are
// referenced by at least one assertion (same metric as `glx stats`).
func coverageRows(a *glxlib.GLXFile) []coverageRowDTO {
	if len(a.Assertions) == 0 {
		return []coverageRowDTO{}
	}

	persons := map[string]struct{}{}
	events := map[string]struct{}{}
	rels := map[string]struct{}{}
	places := map[string]struct{}{}
	for _, assertion := range a.Assertions {
		if assertion == nil {
			continue
		}
		if assertion.Subject.Person != "" {
			persons[assertion.Subject.Person] = struct{}{}
		}
		if assertion.Subject.Event != "" {
			events[assertion.Subject.Event] = struct{}{}
		}
		if assertion.Subject.Relationship != "" {
			rels[assertion.Subject.Relationship] = struct{}{}
		}
		if assertion.Subject.Place != "" {
			places[assertion.Subject.Place] = struct{}{}
		}
	}

	return []coverageRowDTO{
		coverageRow("Persons", len(persons), len(a.Persons)),
		coverageRow("Events", len(events), len(a.Events)),
		coverageRow("Relationships", len(rels), len(a.Relationships)),
		coverageRow("Places", len(places), len(a.Places)),
	}
}

func coverageRow(label string, covered, total int) coverageRowDTO {
	row := coverageRowDTO{Label: label, Covered: covered, Total: total}
	if total > 0 {
		row.Percent = float64(covered) / float64(total) * 100
	}

	return row
}

// ============================================================================
// Persons list
// ============================================================================

type personListItemDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Sex       string `json:"sex,omitempty"`
	BirthYear int    `json:"birthYear,omitempty"`
	DeathYear int    `json:"deathYear,omitempty"`
}

func (s *viewerServer) handlePersons(w http.ResponseWriter, _ *http.Request) {
	a := s.archive
	items := make([]personListItemDTO, 0, len(a.Persons))
	for _, id := range sortedKeys(a.Persons) {
		person := a.Persons[id]
		birth, death := personVitalYears(a, id)
		// person may be nil for a malformed archive entry (e.g. `person-x: null`);
		// extractPersonName already tolerates nil, so only the property read needs
		// guarding.
		sex := ""
		if person != nil {
			sex = propertyString(person.Properties, glxlib.PersonPropertySex)
		}
		items = append(items, personListItemDTO{
			ID:        id,
			Name:      extractPersonName(person),
			Sex:       sex,
			BirthYear: birth,
			DeathYear: death,
		})
	}

	// Sort by display name, falling back to ID for stable ordering.
	sort.SliceStable(items, func(i, j int) bool {
		ni, nj := strings.ToLower(items[i].Name), strings.ToLower(items[j].Name)
		if ni != nj {
			return ni < nj
		}

		return items[i].ID < items[j].ID
	})

	writeJSON(w, http.StatusOK, map[string]any{string(glxlib.EntityTypePersons): items})
}

// ============================================================================
// Person detail
// ============================================================================

type personDetailDTO struct {
	ID      string         `json:"id"`
	Names   []string       `json:"names"`
	Sex     string         `json:"sex,omitempty"`
	Gender  string         `json:"gender,omitempty"`
	Living  bool           `json:"living"`
	Vitals  []eventDTO     `json:"vitals"`
	Events  []eventDTO     `json:"events"`
	Family  familyDTO      `json:"family"`
	Asserts []assertionDTO `json:"assertions"`
	Notes   []string       `json:"notes,omitempty"`
}

type eventDTO struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	TypeLabel string `json:"typeLabel"`
	Date      string `json:"date,omitempty"`
	Place     string `json:"place,omitempty"`
	Role      string `json:"role,omitempty"`
}

type personRefDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BirthYear int    `json:"birthYear,omitempty"`
	DeathYear int    `json:"deathYear,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type familyDTO struct {
	Parents  []personRefDTO `json:"parents"`
	Spouses  []personRefDTO `json:"spouses"`
	Children []personRefDTO `json:"children"`
	Siblings []personRefDTO `json:"siblings"`
}

type assertionDTO struct {
	Property   string        `json:"property,omitempty"`
	Value      string        `json:"value,omitempty"`
	Date       string        `json:"date,omitempty"`
	Confidence string        `json:"confidence,omitempty"`
	Status     string        `json:"status,omitempty"`
	Evidence   []evidenceDTO `json:"evidence"`
}

type evidenceDTO struct {
	CitationID  string `json:"citationId,omitempty"`
	SourceID    string `json:"sourceId,omitempty"`
	SourceTitle string `json:"sourceTitle,omitempty"`
	Locator     string `json:"locator,omitempty"`
	URL         string `json:"url,omitempty"`
}

func (s *viewerServer) handlePerson(w http.ResponseWriter, r *http.Request) {
	a := s.archive
	id := r.PathValue("id")
	person, ok := a.Persons[id]
	if !ok || person == nil {
		writeJSONError(w, http.StatusNotFound, "person not found: "+id)

		return
	}

	detail := personDetailDTO{
		ID:      id,
		Names:   extractAllNames(person),
		Sex:     propertyString(person.Properties, glxlib.PersonPropertySex),
		Gender:  propertyString(person.Properties, glxlib.PersonPropertyGender),
		Living:  glxlib.IsLivingPerson(a, id, time.Now(), glxlib.LivingThresholdYears),
		Vitals:  s.personVitals(id),
		Events:  s.personEvents(id),
		Family:  s.personFamily(id),
		Asserts: s.personAssertions(id),
		Notes:   notesToStrings(person.Notes),
	}
	if len(detail.Names) == 0 {
		detail.Names = []string{extractPersonName(person)}
	}

	writeJSON(w, http.StatusOK, detail)
}

// vitalEventTypes are the life-defining events surfaced separately from the
// general event list, in display order.
var vitalEventTypes = []string{
	glxlib.EventTypeBirth,
	glxlib.EventTypeChristening,
	glxlib.EventTypeBaptism,
	glxlib.EventTypeDeath,
	glxlib.EventTypeBurial,
}

func (s *viewerServer) personVitals(id string) []eventDTO {
	a := s.archive
	out := []eventDTO{}
	for _, eventType := range vitalEventTypes {
		eventID, event := glxlib.FindPersonEvent(a, id, eventType)
		if event == nil {
			continue
		}
		out = append(out, eventDTO{
			ID:        eventID,
			Type:      event.Type,
			TypeLabel: eventLabel(event.Type),
			Date:      string(event.Date),
			Place:     resolvePlaceName(event.PlaceID, a),
		})
	}

	return out
}

// personEvents returns every event a person participates in, chronologically.
func (s *viewerServer) personEvents(id string) []eventDTO {
	a := s.archive
	out := []eventDTO{}
	for _, eventID := range sortedKeys(a.Events) {
		event := a.Events[eventID]
		if event == nil {
			continue
		}
		role := participantRole(id, event.Participants)
		if role == roleAbsent {
			continue
		}
		out = append(out, eventDTO{
			ID:        eventID,
			Type:      event.Type,
			TypeLabel: eventLabel(event.Type),
			Date:      string(event.Date),
			Place:     resolvePlaceName(event.PlaceID, a),
			Role:      role,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return dateSortKey(out[i].Date) < dateSortKey(out[j].Date)
	})

	return out
}

func (s *viewerServer) personFamily(id string) familyDTO {
	a := s.archive
	parentIDs := findParentIDs(id, a)

	fam := familyDTO{
		Parents:  s.personRefs(parentIDs),
		Children: s.personRefs(findChildIDs(id, a)),
		Siblings: s.personRefs(findSiblingIDs(id, parentIDs, a)),
		Spouses:  []personRefDTO{},
	}
	for _, sp := range findSpouses(id, a) {
		ref := s.personRef(sp.PersonID)
		ref.Name = sp.PersonName
		detail := strings.TrimSpace(strings.Join(nonEmpty(sp.MarriageDate, sp.MarriagePlace), " · "))
		ref.Detail = detail
		fam.Spouses = append(fam.Spouses, ref)
	}

	return fam
}

func (s *viewerServer) personRefs(ids []string) []personRefDTO {
	refs := make([]personRefDTO, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, s.personRef(id))
	}

	return refs
}

func (s *viewerServer) personRef(id string) personRefDTO {
	a := s.archive
	birth, death := personVitalYears(a, id)
	name := id
	if person, ok := a.Persons[id]; ok && person != nil {
		name = extractPersonName(person)
	}

	return personRefDTO{ID: id, Name: name, BirthYear: birth, DeathYear: death}
}

// personAssertions returns assertions whose subject is this person, with their
// evidence (citations and direct sources) resolved to human-readable form.
func (s *viewerServer) personAssertions(id string) []assertionDTO {
	a := s.archive
	out := []assertionDTO{}
	for _, assertID := range sortedKeys(a.Assertions) {
		assertion := a.Assertions[assertID]
		if assertion == nil || assertion.Subject.Person != id {
			continue
		}
		dto := assertionDTO{
			Property:   assertion.Property,
			Value:      assertion.Value,
			Date:       string(assertion.Date),
			Confidence: assertion.Confidence,
			Status:     assertion.Status,
			Evidence:   s.resolveEvidence(assertion.Citations, assertion.Sources),
		}
		// Participant assertions carry no property/value; describe them so the
		// row is not blank in the UI.
		if dto.Property == "" && assertion.Participant != nil {
			dto.Property = "participation"
			dto.Value = strings.TrimSpace(assertion.Participant.Role)
		}
		out = append(out, dto)
	}

	return out
}

func (s *viewerServer) resolveEvidence(citationIDs, sourceIDs []string) []evidenceDTO {
	a := s.archive
	out := make([]evidenceDTO, 0, len(citationIDs)+len(sourceIDs))
	for _, cid := range citationIDs {
		ev := evidenceDTO{CitationID: cid}
		if citation, ok := a.Citations[cid]; ok && citation != nil {
			ev.SourceID = citation.SourceID
			ev.SourceTitle = sourceTitle(a, citation.SourceID)
			ev.Locator = propertyString(citation.Properties, "locator")
			ev.URL = citationURL(a, citation)
		}
		out = append(out, ev)
	}
	for _, sid := range sourceIDs {
		out = append(out, evidenceDTO{
			SourceID:    sid,
			SourceTitle: sourceTitle(a, sid),
		})
	}

	return out
}

// ============================================================================
// Family tree (pedigree / descendancy)
// ============================================================================

type treeNodeDTO struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	BirthYear int            `json:"birthYear,omitempty"`
	DeathYear int            `json:"deathYear,omitempty"`
	Sex       string         `json:"sex,omitempty"`
	Children  []*treeNodeDTO `json:"children,omitempty"`
}

func (s *viewerServer) handleTree(w http.ResponseWriter, r *http.Request) {
	a := s.archive
	id := r.PathValue("id")
	if person, ok := a.Persons[id]; !ok || person == nil {
		writeJSONError(w, http.StatusNotFound, "person not found: "+id)

		return
	}

	direction := r.URL.Query().Get("dir")
	if direction != "descendants" {
		direction = "ancestors"
	}
	gen := clampGen(r.URL.Query().Get("gen"))

	// Build the parent/child adjacency once per request so buildTree does not
	// rescan (and re-sort) archive.Relationships for every node.
	idx := buildTreeRelIndex(a)
	root := s.buildTree(idx, id, direction, 0, gen, map[string]bool{})
	writeJSON(w, http.StatusOK, map[string]any{
		"direction": direction,
		"gen":       gen,
		"root":      root,
	})
}

// treeRelIndex holds the parent/child adjacency of an archive, built once so a
// whole-tree traversal is O(nodes) instead of O(nodes × relationships).
type treeRelIndex struct {
	parents  map[string][]string // child ID -> parent IDs
	children map[string][]string // parent ID -> child IDs
}

// buildTreeRelIndex scans the parent-child relationships once and records both
// directions. It mirrors findParentIDs/findChildIDs: parents keep first-seen
// order; children are ordered by birth year (then ID) for a stable chart.
func buildTreeRelIndex(a *glxlib.GLXFile) *treeRelIndex {
	idx := &treeRelIndex{parents: map[string][]string{}, children: map[string][]string{}}
	for _, relID := range sortedKeys(a.Relationships) {
		rel := a.Relationships[relID]
		if rel == nil || !parentChildRelTypes[strings.ToLower(rel.Type)] {
			continue
		}

		var parents, children []string
		for _, p := range rel.Participants {
			switch strings.ToLower(p.Role) {
			case glxlib.ParticipantRoleParent:
				parents = append(parents, p.Person)
			case glxlib.ParticipantRoleChild:
				children = append(children, p.Person)
			}
		}
		for _, childID := range children {
			for _, parentID := range parents {
				idx.children[parentID] = appendUniqueString(idx.children[parentID], childID)
				idx.parents[childID] = appendUniqueString(idx.parents[childID], parentID)
			}
		}
	}

	for parentID := range idx.children {
		kids := idx.children[parentID]
		sort.SliceStable(kids, func(i, j int) bool {
			yi, yj := birthYear(a, kids[i]), birthYear(a, kids[j])
			if yi != yj {
				return yi < yj
			}

			return kids[i] < kids[j]
		})
	}

	return idx
}

func appendUniqueString(s []string, v string) []string {
	if slices.Contains(s, v) {
		return s
	}

	return append(s, v)
}

// buildTree recursively assembles a pedigree (ancestors) or descendancy
// (descendants) chart from a prebuilt adjacency index. `path` provides
// backtracking cycle detection so the same person can legitimately appear in two
// branches (cousin marriage / pedigree collapse) without looping forever on a
// malformed self-ancestor cycle.
func (s *viewerServer) buildTree(idx *treeRelIndex, id, direction string, depth, maxGen int, path map[string]bool) *treeNodeDTO {
	a := s.archive
	person := a.Persons[id]
	birth, death := personVitalYears(a, id)
	name := id
	sex := ""
	if person != nil {
		name = extractPersonName(person)
		sex = propertyString(person.Properties, glxlib.PersonPropertySex)
	}
	node := &treeNodeDTO{ID: id, Name: name, BirthYear: birth, DeathYear: death, Sex: sex}

	if depth >= maxGen || path[id] {
		return node
	}

	path[id] = true
	defer delete(path, id)

	nextIDs := idx.parents[id]
	if direction == "descendants" {
		nextIDs = idx.children[id]
	}
	for _, nextID := range nextIDs {
		node.Children = append(node.Children, s.buildTree(idx, nextID, direction, depth+1, maxGen, path))
	}

	return node
}

func clampGen(raw string) int {
	gen, err := strconv.Atoi(raw)
	if err != nil || gen < 1 {
		return serveDefaultTreeGen
	}
	if gen > serveMaxTreeGen {
		return serveMaxTreeGen
	}

	return gen
}

// ============================================================================
// Sources
// ============================================================================

type sourceListItemDTO struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Type          string `json:"type,omitempty"`
	Date          string `json:"date,omitempty"`
	CitationCount int    `json:"citationCount"`
}

func (s *viewerServer) handleSources(w http.ResponseWriter, _ *http.Request) {
	a := s.archive
	citationCounts := map[string]int{}
	for _, citation := range a.Citations {
		if citation == nil {
			continue
		}
		citationCounts[citation.SourceID]++
	}

	items := make([]sourceListItemDTO, 0, len(a.Sources))
	for _, id := range sortedKeys(a.Sources) {
		source := a.Sources[id]
		// sourceDisplayTitle already falls back to the ID for a nil source; only
		// the type/date reads need a nil guard.
		item := sourceListItemDTO{
			ID:            id,
			Title:         sourceDisplayTitle(id, source),
			CitationCount: citationCounts[id],
		}
		if source != nil {
			item.Type = source.Type
			item.Date = string(source.Date)
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		ti, tj := strings.ToLower(items[i].Title), strings.ToLower(items[j].Title)
		if ti != tj {
			return ti < tj
		}

		return items[i].ID < items[j].ID
	})

	writeJSON(w, http.StatusOK, map[string]any{string(glxlib.EntityTypeSources): items})
}

type sourceDetailDTO struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	Type       string              `json:"type,omitempty"`
	Authors    []string            `json:"authors,omitempty"`
	Date       string              `json:"date,omitempty"`
	Repository string              `json:"repository,omitempty"`
	URL        string              `json:"url,omitempty"`
	Notes      []string            `json:"notes,omitempty"`
	Citations  []sourceCitationDTO `json:"citations"`
}

type sourceCitationDTO struct {
	ID       string `json:"id"`
	Locator  string `json:"locator,omitempty"`
	Text     string `json:"text,omitempty"`
	URL      string `json:"url,omitempty"`
	Accessed string `json:"accessed,omitempty"`
}

func (s *viewerServer) handleSource(w http.ResponseWriter, r *http.Request) {
	a := s.archive
	id := r.PathValue("id")
	source, ok := a.Sources[id]
	if !ok || source == nil {
		writeJSONError(w, http.StatusNotFound, "source not found: "+id)

		return
	}

	detail := sourceDetailDTO{
		ID:        id,
		Title:     sourceDisplayTitle(id, source),
		Type:      source.Type,
		Authors:   source.Authors,
		Date:      string(source.Date),
		URL:       propertyString(source.Properties, "url"),
		Notes:     notesToStrings(source.Notes),
		Citations: []sourceCitationDTO{},
	}
	if repo, ok := a.Repositories[source.RepositoryID]; ok && repo != nil {
		detail.Repository = repo.Name
	}

	for _, cid := range sortedKeys(a.Citations) {
		citation := a.Citations[cid]
		if citation == nil || citation.SourceID != id {
			continue
		}
		detail.Citations = append(detail.Citations, sourceCitationDTO{
			ID:       cid,
			Locator:  propertyString(citation.Properties, "locator"),
			Text:     propertyString(citation.Properties, "text_from_source"),
			URL:      citationURL(a, citation),
			Accessed: propertyString(citation.Properties, "accessed"),
		})
	}

	writeJSON(w, http.StatusOK, detail)
}

// ============================================================================
// Shared helpers
// ============================================================================

// roleAbsent marks a person who does not participate in an event.
const roleAbsent = "\x00absent"

// participantRole returns the role a person plays in a participant list, or
// roleAbsent if they are not present. An empty role string is normalized to
// "participant" so the UI always has a label.
func participantRole(personID string, participants []glxlib.Participant) string {
	for _, p := range participants {
		if p.Person == personID {
			if p.Role == "" {
				return "participant"
			}

			return p.Role
		}
	}

	return roleAbsent
}

// personVitalYears returns the person's birth and death years (0 when unknown).
func personVitalYears(a *glxlib.GLXFile, personID string) (birth, death int) {
	if _, event := glxlib.FindPersonEvent(a, personID, glxlib.EventTypeBirth); event != nil {
		birth = glxlib.ExtractFirstYear(string(event.Date))
	}
	if _, event := glxlib.FindPersonEvent(a, personID, glxlib.EventTypeDeath); event != nil {
		death = glxlib.ExtractFirstYear(string(event.Date))
	}

	return birth, death
}

func eventLabel(eventType string) string {
	return glxlib.GenerateEventTitle(eventType, nil, "")
}

func sourceTitle(a *glxlib.GLXFile, sourceID string) string {
	if source, ok := a.Sources[sourceID]; ok && source != nil {
		return sourceDisplayTitle(sourceID, source)
	}

	return sourceID
}

func sourceDisplayTitle(id string, source *glxlib.Source) string {
	if source != nil && strings.TrimSpace(source.Title) != "" {
		return source.Title
	}

	return id
}

// citationURL prefers the citation's own URL, falling back to its source's URL.
func citationURL(a *glxlib.GLXFile, citation *glxlib.Citation) string {
	if url := propertyString(citation.Properties, "url"); url != "" {
		return url
	}
	if source, ok := a.Sources[citation.SourceID]; ok && source != nil {
		return propertyString(source.Properties, "url")
	}

	return ""
}

// notesToStrings copies the non-blank notes from a NoteList for JSON output.
func notesToStrings(notes glxlib.NoteList) []string {
	if len(notes) == 0 {
		return nil
	}
	out := make([]string, 0, len(notes))
	for _, note := range notes {
		if text := strings.TrimSpace(note); text != "" {
			out = append(out, text)
		}
	}

	return out
}

// nonEmpty returns the non-blank arguments, preserving order.
func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}

	return out
}
