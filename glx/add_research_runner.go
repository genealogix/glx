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
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// =============================================================================
// add research-log
// =============================================================================

type addResearchLogOptions struct {
	addCommonOptions
	Title               string
	SubjectPerson       string
	SubjectEvent        string
	SubjectRelationship string
	SubjectPlace        string
	Objective           string
	Status              string
	Date                string
	Researcher          string
	Conclusions         string
	Citations           []string
}

// addResearchLog mints a ResearchLog entity from the supplied flags. The
// subject is optional (a log may investigate a question that has no entity
// yet), but at most one --subject-* flag may be given. Searches are appended
// afterwards, one at a time, with `glx add search`.
func addResearchLog(io *IOStreams, opts *addResearchLogOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	subject, err := resolveOptionalSubject(ctx.archive, opts.SubjectPerson, opts.SubjectEvent, opts.SubjectRelationship, opts.SubjectPlace)
	if err != nil {
		return err
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabResearchLogStatusTypes, opts.Status); err != nil {
		return err
	}
	for _, cid := range opts.Citations {
		if err := validateRefExists(ctx.archive, glxlib.EntityTypeCitations, cid); err != nil {
			return err
		}
	}

	// The derived ID reads best from a title, then from what the log is
	// about, then from the objective (a full sentence, capped by EntityID).
	descriptor := strings.TrimSpace(opts.Title)
	if descriptor == "" && subject != nil {
		descriptor = stripEntityPrefix(subjectRefID(subject))
	}
	if descriptor == "" {
		descriptor = strings.TrimSpace(opts.Objective)
	}
	if descriptor == "" && opts.OverrideID == "" {
		return ErrAddResearchLogDescriptorRequired
	}
	base := glxlib.EntityID(glxlib.EntityTypeResearchLogs.IDPrefix(), descriptor)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.ResearchLogs), opts.Force)
	if err != nil {
		return err
	}

	log := &glxlib.ResearchLog{
		Title:       strings.TrimSpace(opts.Title),
		Subject:     subject,
		Date:        glxlib.DateString(opts.Date),
		Researcher:  strings.TrimSpace(opts.Researcher),
		Objective:   strings.TrimSpace(opts.Objective),
		Status:      opts.Status,
		Citations:   dedupeStrings(opts.Citations),
		Conclusions: strings.TrimSpace(opts.Conclusions),
	}
	if len(opts.Notes) > 0 {
		log.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{ResearchLogs: map[string]*glxlib.ResearchLog{id: log}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeResearchLogs, id, partial)
}

// resolveOptionalSubject enforces "at most one of --subject-*" and returns the
// typed reference, or nil when no subject flag was given. The chosen ID is
// reference-checked against the archive.
func resolveOptionalSubject(archive *glxlib.GLXFile, person, event, relationship, place string) (*glxlib.EntityRef, error) {
	subjects := []struct {
		entityType glxlib.EntityType
		id         string
	}{
		{glxlib.EntityTypePersons, person},
		{glxlib.EntityTypeEvents, event},
		{glxlib.EntityTypeRelationships, relationship},
		{glxlib.EntityTypePlaces, place},
	}
	var (
		chosenType glxlib.EntityType
		chosenID   string
	)
	for _, s := range subjects {
		if s.id == "" {
			continue
		}
		if chosenID != "" {
			return nil, ErrAddSubjectConflict
		}
		chosenType, chosenID = s.entityType, s.id
	}
	if chosenID == "" {
		return nil, nil
	}
	if err := validateRefExists(archive, chosenType, chosenID); err != nil {
		return nil, err
	}
	ref := assertionSubjectRef(chosenType, chosenID)

	return &ref, nil
}

// subjectRefID returns whichever ID the reference carries.
func subjectRefID(ref *glxlib.EntityRef) string {
	for _, id := range []string{ref.Person, ref.Event, ref.Relationship, ref.Place} {
		if id != "" {
			return id
		}
	}

	return ""
}

// =============================================================================
// add search
// =============================================================================

// addSearchOptions holds the flags for `glx add search`. A search is embedded
// in its research log rather than being an entity of its own, so it takes no
// --id or --force; the log ID is the value echoed to stdout.
type addSearchOptions struct {
	ArchivePath  string
	SkipValidate bool
	DryRun       bool
	Notes        []string

	Log        string
	Source     string
	Repository string
	Collection string
	Query      string
	Result     string
	Citation   string
	Date       string
}

// addSearch appends one Search entry to an existing research log. Only the
// file that holds the log is rewritten, in place, wherever it lives in the
// archive (its own file, or a file shared with other entities); every other
// file is left byte-for-byte untouched. A --citation is also rolled up into
// the log's own citations list, as the research-log spec recommends.
func addSearch(io *IOStreams, opts *addSearchOptions) error {
	files, err := collectGLXFilesFromDir(opts.ArchivePath)
	if err != nil {
		return fmt.Errorf("loading archive: %w", err)
	}
	archive, duplicates, err := loadArchiveFromFiles(opts.ArchivePath, files, false)
	if err != nil {
		return fmt.Errorf("loading archive: %w", err)
	}
	for _, d := range duplicates {
		io.Errorf("Warning: %s\n", d)
	}

	search, err := buildSearch(archive, opts)
	if err != nil {
		return err
	}
	existing := archive.ResearchLogs[opts.Log]
	updated := withSearch(existing, search)

	if !opts.SkipValidate {
		partial := &glxlib.GLXFile{ResearchLogs: map[string]*glxlib.ResearchLog{opts.Log: updated}}
		if err := runWholeArchiveValidate(archive, partial); err != nil {
			return err
		}
	}

	op, err := planAddSearchWrite(files, opts.Log, search)
	if err != nil {
		return err
	}
	ops := []fileOp{op}
	if err := preflightFileOps(opts.ArchivePath, ops); err != nil {
		return err
	}

	io.Printf("Adding search to %s: %s (search %d, %s)\n",
		glxlib.EntityTypeResearchLogs.Singular(), opts.Log, len(updated.Searches), filepath.ToSlash(op.relPath))
	if opts.DryRun {
		io.Println("(dry run — no files written)")
		fmt.Fprintln(io.MachineOut, opts.Log)

		return nil
	}

	if err := executeFileOps(opts.ArchivePath, ops); err != nil {
		return err
	}
	fmt.Fprintln(io.MachineOut, opts.Log)

	return nil
}

// buildSearch validates the search flags against the loaded archive and
// returns the Search entry to append.
func buildSearch(archive *glxlib.GLXFile, opts *addSearchOptions) (*glxlib.Search, error) {
	if strings.TrimSpace(opts.Log) == "" {
		return nil, ErrAddSearchLogRequired
	}
	if _, ok := archive.ResearchLogs[opts.Log]; !ok {
		return nil, fmt.Errorf("%w: %s %q (create it with `glx add research-log`)",
			ErrAddRefNotFound, glxlib.EntityTypeResearchLogs, opts.Log)
	}
	if opts.Source == "" && opts.Repository == "" && strings.TrimSpace(opts.Collection) == "" &&
		strings.TrimSpace(opts.Query) == "" && opts.Citation == "" {
		return nil, ErrAddSearchWhatRequired
	}
	if err := validateRefExists(archive, glxlib.EntityTypeSources, opts.Source); err != nil {
		return nil, err
	}
	if err := validateRefExists(archive, glxlib.EntityTypeRepositories, opts.Repository); err != nil {
		return nil, err
	}
	if err := validateRefExists(archive, glxlib.EntityTypeCitations, opts.Citation); err != nil {
		return nil, err
	}
	if err := validateVocabKey(archive, glxlib.VocabSearchResultTypes, opts.Result); err != nil {
		return nil, err
	}

	search := &glxlib.Search{
		RepositoryID: opts.Repository,
		SourceID:     opts.Source,
		Collection:   strings.TrimSpace(opts.Collection),
		Query:        strings.TrimSpace(opts.Query),
		Date:         glxlib.DateString(opts.Date),
		Result:       opts.Result,
		CitationID:   opts.Citation,
	}
	if len(opts.Notes) > 0 {
		search.Notes = glxlib.NoteList(opts.Notes)
	}

	return search, nil
}

// withSearch returns a copy of log with search appended and its citation (if
// any) rolled up into the log-level citations list. The input is not mutated,
// so the loaded archive keeps matching what is on disk.
func withSearch(log *glxlib.ResearchLog, search *glxlib.Search) *glxlib.ResearchLog {
	updated := *log
	updated.Searches = append(slices.Clone(log.Searches), *search)
	updated.Citations = slices.Clone(log.Citations)
	if search.CitationID != "" && !slices.Contains(updated.Citations, search.CitationID) {
		updated.Citations = append(updated.Citations, search.CitationID)
	}

	return &updated
}

// planAddSearchWrite finds the one archive file that defines the research log
// and returns the operation that rewrites it with the search appended. The
// file is re-serialized as a fragment, the same way `glx rename` rewrites the
// files it touches, so its layout (single- or multi-entity) is kept.
func planAddSearchWrite(files map[string][]byte, logID string, search *glxlib.Search) (fileOp, error) {
	serializer := createSerializer(false, true, "  ")

	relPaths := make([]string, 0, len(files))
	for relPath := range files {
		relPaths = append(relPaths, relPath)
	}
	sort.Strings(relPaths)

	var (
		found    []string
		fragment *glxlib.GLXFile
		oldData  []byte
	)
	for _, relPath := range relPaths {
		data := files[relPath]
		frag, err := serializer.DeserializeSingleFileBytes(data)
		if err != nil {
			return fileOp{}, fmt.Errorf("failed to parse %s: %w", relPath, err)
		}
		if _, ok := frag.ResearchLogs[logID]; !ok {
			continue
		}
		found = append(found, relPath)
		fragment, oldData = frag, data
	}
	switch len(found) {
	case 0:
		return fileOp{}, fmt.Errorf("%w: %s %q", ErrAddRefNotFound, glxlib.EntityTypeResearchLogs, logID)
	case 1:
	default:
		return fileOp{}, fmt.Errorf("%w: %q is defined in %s", ErrAddSearchLogAmbiguous, logID, strings.Join(found, ", "))
	}

	current := fragment.ResearchLogs[logID]
	rollUpCitation := search.CitationID != "" && !slices.Contains(current.Citations, search.CitationID)
	want := withSearch(current, search)
	fragment.ResearchLogs[logID] = want
	if newData, ok := appendSearchPreservingLayout(serializer, oldData, logID, search, rollUpCitation, fragment); ok {
		return fileOp{relPath: found[0], oldData: oldData, newData: newData}, nil
	}

	// The file has a shape the node-level edit does not handle (anchors,
	// flow style, ...): fall back to re-serializing the fragment, as
	// `glx rename` does for the files it touches.
	newData, err := serializer.SerializeSingleFileBytes(fragment)
	if err != nil {
		return fileOp{}, fmt.Errorf("failed to serialize %s: %w", found[0], err)
	}

	return fileOp{relPath: found[0], oldData: oldData, newData: newData}, nil
}

// appendSearchPreservingLayout appends the search (and its citation roll-up)
// to the log at the YAML node level, so the rest of the file keeps its
// quoting, key order, comments and indentation and the diff is just the new
// lines. The result is re-read and compared with the whole expected fragment,
// including other entities an alias could affect; any surprise reports
// ok=false so the caller falls back to a plain re-serialization.
func appendSearchPreservingLayout(serializer *glxlib.DefaultSerializer, data []byte, logID string, search *glxlib.Search, rollUpCitation bool, want *glxlib.GLXFile) ([]byte, bool) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
		return nil, false
	}
	logs := mappingValue(doc.Content[0], string(glxlib.EntityTypeResearchLogs))
	logNode := mappingValue(logs, logID)
	if logNode == nil || logNode.Kind != yaml.MappingNode {
		return nil, false
	}

	var searchNode yaml.Node
	if err := searchNode.Encode(search); err != nil {
		return nil, false
	}
	if !appendToSequence(logNode, "searches", &searchNode) {
		return nil, false
	}
	if rollUpCitation {
		citation := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: search.CitationID}
		if !appendToSequence(logNode, "citations", citation) {
			return nil, false
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(detectYAMLIndent(data))
	if err := enc.Encode(&doc); err != nil {
		return nil, false
	}
	if err := enc.Close(); err != nil {
		return nil, false
	}
	out := buf.Bytes()
	if bytes.Contains(data, []byte("\r\n")) {
		out = bytes.ReplaceAll(out, []byte("\n"), []byte("\r\n"))
	}

	check, err := serializer.DeserializeSingleFileBytes(out)
	if err != nil || !reflect.DeepEqual(check, want) {
		return nil, false
	}

	return out, true
}

// mappingValue returns the value node for key in a block mapping node, or nil.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

// appendToSequence appends item to the block sequence under key in mapping,
// creating the key when it is absent. Reports false for any other shape.
func appendToSequence(mapping *yaml.Node, key string, item *yaml.Node) bool {
	seq := mappingValue(mapping, key)
	if seq == nil {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			&yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{item}})

		return true
	}
	if seq.Kind != yaml.SequenceNode || seq.Style&yaml.FlowStyle != 0 {
		return false
	}
	seq.Content = append(seq.Content, item)

	return true
}

// detectYAMLIndent returns the indentation width of the first indented line,
// which is how deep the file nests one level (2 for hand-written examples, 4
// for files the serializer wrote). Defaults to 2.
func detectYAMLIndent(data []byte) int {
	for line := range strings.Lines(string(data)) {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || strings.TrimSpace(trimmed) == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if n := len(line) - len(trimmed); n >= 2 && n <= 9 {
			return n
		}
	}

	return 2
}

// =============================================================================
// add study
// =============================================================================

type addStudyOptions struct {
	addCommonOptions
	Title     string
	Type      string
	Status    string
	DateRange string
	Places    []string
	Sources   []string
}

// addStudy mints a Study entity from the supplied flags. --title is required
// (it is the one required field of a Study); --type and --status are checked
// against the study_types and study_statuses vocabularies.
func addStudy(io *IOStreams, opts *addStudyOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Title) == "" {
		return ErrAddStudyTitleRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabStudyTypes, opts.Type); err != nil {
		return err
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabStudyStatuses, opts.Status); err != nil {
		return err
	}
	for _, pid := range opts.Places {
		if err := validateRefExists(ctx.archive, glxlib.EntityTypePlaces, pid); err != nil {
			return err
		}
	}
	for _, sid := range opts.Sources {
		if err := validateRefExists(ctx.archive, glxlib.EntityTypeSources, sid); err != nil {
			return err
		}
	}

	base := glxlib.EntityID(glxlib.EntityTypeStudies.IDPrefix(), opts.Title)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Studies), opts.Force)
	if err != nil {
		return err
	}

	study := &glxlib.Study{
		Title:     strings.TrimSpace(opts.Title),
		Type:      opts.Type,
		Status:    opts.Status,
		DateRange: glxlib.DateString(opts.DateRange),
		Places:    dedupeStrings(opts.Places),
		Sources:   dedupeStrings(opts.Sources),
	}
	if len(opts.Notes) > 0 {
		study.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Studies: map[string]*glxlib.Study{id: study}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeStudies, id, partial)
}
