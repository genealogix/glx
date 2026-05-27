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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

// maxAddIDCollisions bounds the -2, -3, ... suffix search for derived IDs.
const maxAddIDCollisions = 1000

// addCommonOptions holds the flags shared by every `glx add` subcommand.
// Per-subcommand option structs embed it.
type addCommonOptions struct {
	ArchivePath  string
	OverrideID   string
	Force        bool
	SkipValidate bool
	DryRun       bool
	Notes        []string
}

// addContext bundles the loaded archive and resolved id-taken sets for the
// duration of a single add call. Built once at the start of each subcommand;
// passed to the entity-specific build step.
type addContext struct {
	archive *glxlib.GLXFile
}

// loadAddContext loads the archive and surfaces any duplicate-ID warnings
// from the multi-file loader to stderr.
func loadAddContext(io *IOStreams, archivePath string) (*addContext, error) {
	archive, duplicates, err := LoadArchiveWithOptions(archivePath, false)
	if err != nil {
		return nil, fmt.Errorf("loading archive: %w", err)
	}
	for _, d := range duplicates {
		io.Errorf("Warning: %s\n", d)
	}

	return &addContext{archive: archive}, nil
}

// validEntityIDPattern matches the GLX entity-ID grammar: must start with an
// alphanumeric and otherwise contain only alphanumerics and hyphens, up to
// 64 chars total. Enforced here as defense-in-depth on top of
// EntityIDToFilename: that helper rejects path-traversal and reserved device
// names but does not enforce the spec's charset/length, so a NUL byte,
// control character, or 200-char ID could otherwise reach disk.
var validEntityIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,63}$`)

// mustBeSafeID returns ErrAddInvalidID if the proposed ID is unsafe as either
// a filename (EntityIDToFilename) or as a GLX entity ID (validEntityIDPattern).
func mustBeSafeID(id string) error {
	if _, err := glxlib.EntityIDToFilename(id); err != nil {
		return fmt.Errorf("%w: %s", ErrAddInvalidID, id)
	}
	if !validEntityIDPattern.MatchString(id) {
		return fmt.Errorf("%w: %s (must match %s)", ErrAddInvalidID, id, validEntityIDPattern.String())
	}

	return nil
}

// deriveOrOverrideID applies the rule "user-supplied --id wins, otherwise use
// the derived base, otherwise -2/-3/… on collision". The resulting ID is
// validated as a safe filename and (unless --force) checked for collisions
// against `taken`.
//
// base is the already-slugified entity ID, typically produced via
// glxlib.EntityID(prefix, descriptiveText). The caller is responsible for
// any prefix, suffix, or length cap.
func deriveOrOverrideID(base, overrideID string, taken map[string]struct{}, force bool) (string, error) {
	if overrideID != "" {
		if err := mustBeSafeID(overrideID); err != nil {
			return "", err
		}
		if _, exists := taken[overrideID]; exists && !force {
			return "", fmt.Errorf("%w: %s", ErrAddEntityExists, overrideID)
		}

		return overrideID, nil
	}

	// Defense-in-depth: base is expected to arrive already slugified and capped
	// via glxlib.EntityID(prefix, userText), so this never fires for a
	// well-formed caller — but base ultimately derives from user input, so we
	// fail fast rather than write an uncapped or invalid ID to disk.
	if err := mustBeSafeID(base); err != nil {
		return "", err
	}
	if _, exists := taken[base]; !exists {
		return base, nil
	}

	// Derived IDs always pick the next free suffix on collision. --force does
	// NOT silently overwrite an entity the user didn't explicitly point at —
	// it only applies when the caller supplied --id.
	for i := 2; i <= maxAddIDCollisions; i++ {
		suffix := fmt.Sprintf("-%d", i)
		candidate := glxlib.Slugify(base, glxlib.WithSlugSuffix(suffix), glxlib.WithSlugMaxLength(glxlib.MaxEntityIDLength))
		if err := mustBeSafeID(candidate); err != nil {
			return "", err
		}
		if _, exists := taken[candidate]; !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrAddIDExhausted, base)
}

// shortHash returns the first 8 hex chars of SHA-256(seed). If seed is empty,
// uses the current UTC time so callers that pass no seed still get a unique
// value per invocation.
func shortHash(seed string) string {
	if seed == "" {
		seed = time.Now().UTC().Format(time.RFC3339Nano)
	}
	sum := sha256.Sum256([]byte(seed))

	return hex.EncodeToString(sum[:])[:8]
}

// validateVocabKey ensures `value` is a key in the archive's named vocabulary.
// vocabularies are loaded with standard defaults by LoadArchiveWithOptions, so
// callers can rely on the standard vocabularies being present even for a
// freshly-initialized archive. An empty value is a no-op (optional field).
func validateVocabKey(archive *glxlib.GLXFile, vocabName, value string) error {
	if value == "" {
		return nil
	}
	vocab := pickVocab(archive, vocabName)
	if vocab == nil {
		return fmt.Errorf("%w: unknown vocabulary %q", ErrAddVocabKeyUnknown, vocabName)
	}
	if _, ok := vocab[value]; !ok {
		return fmt.Errorf("%w: %s=%q", ErrAddVocabKeyUnknown, vocabName, value)
	}

	return nil
}

// pickVocab returns the requested vocabulary map by name.
func pickVocab(archive *glxlib.GLXFile, vocabName string) map[string]*glxlib.VocabularyEntry {
	switch vocabName {
	case glxlib.VocabEventTypes:
		return archive.EventTypes
	case glxlib.VocabRelationshipTypes:
		return archive.RelationshipTypes
	case glxlib.VocabPlaceTypes:
		return archive.PlaceTypes
	case glxlib.VocabSourceTypes:
		return archive.SourceTypes
	case glxlib.VocabRepositoryTypes:
		return archive.RepositoryTypes
	case glxlib.VocabConfidenceLevels:
		return archive.ConfidenceLevels
	case glxlib.VocabSexTypes:
		return archive.SexTypes
	case glxlib.VocabGenderTypes:
		return archive.GenderTypes
	case glxlib.VocabParticipantRoles:
		return archive.ParticipantRoles
	case glxlib.VocabMediaTypes:
		return archive.MediaTypes
	default:
		return nil
	}
}

// validateRefExists ensures `id` is a key in the archive's named entity map.
// An empty id is a no-op.
func validateRefExists(archive *glxlib.GLXFile, entityType, id string) error {
	if id == "" {
		return nil
	}
	if !entityIDExists(archive, entityType, id) {
		return fmt.Errorf("%w: %s %q", ErrAddRefNotFound, entityType, id)
	}

	return nil
}

// entityIDExists returns whether `id` is present in the entity map for
// `entityType`. Unknown entity types return false so callers see a typed
// ErrAddRefNotFound rather than a panic.
func entityIDExists(archive *glxlib.GLXFile, entityType, id string) bool {
	switch entityType {
	case glxlib.EntityTypePersons:
		return mapHas(archive.Persons, id)
	case glxlib.EntityTypeEvents:
		return mapHas(archive.Events, id)
	case glxlib.EntityTypeRelationships:
		return mapHas(archive.Relationships, id)
	case glxlib.EntityTypePlaces:
		return mapHas(archive.Places, id)
	case glxlib.EntityTypeSources:
		return mapHas(archive.Sources, id)
	case glxlib.EntityTypeCitations:
		return mapHas(archive.Citations, id)
	case glxlib.EntityTypeRepositories:
		return mapHas(archive.Repositories, id)
	case glxlib.EntityTypeAssertions:
		return mapHas(archive.Assertions, id)
	case glxlib.EntityTypeMedia:
		return mapHas(archive.Media, id)
	default:
		return false
	}
}

// mapHas reports whether `id` is a key in `m`. Generic over value type.
func mapHas[T any](m map[string]*T, id string) bool {
	_, ok := m[id]

	return ok
}

// parseParticipantFlag splits "person-x:role" into ("person-x", "role").
// Whitespace around either part is trimmed. The role half is optional only
// when the flag omits the colon entirely; "person-x:" (trailing colon with
// empty role) is rejected so the help-string "person-id:role" contract is
// enforced uniformly with parseExternalIDFlag.
func parseParticipantFlag(s string) (person, role string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", fmt.Errorf("%w: empty value", ErrAddParticipantFormat)
	}
	parts := strings.SplitN(s, ":", 2)
	person = strings.TrimSpace(parts[0])
	if person == "" {
		return "", "", fmt.Errorf("%w: missing person ID in %q", ErrAddParticipantFormat, s)
	}
	if len(parts) == 2 {
		role = strings.TrimSpace(parts[1])
		if role == "" {
			return "", "", fmt.Errorf("%w: empty role in %q", ErrAddParticipantFormat, s)
		}
	}

	return person, role, nil
}

// parseExternalIDFlag splits "type:value" into ("type", "value"). Both halves
// are required.
func parseExternalIDFlag(s string) (typ, value string, err error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: %q", ErrAddExternalIDFormat, s)
	}
	typ = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])
	if typ == "" || value == "" {
		return "", "", fmt.Errorf("%w: %q", ErrAddExternalIDFormat, s)
	}

	return typ, value, nil
}

// runWholeArchiveValidate merges `partial` into a working copy of the loaded
// archive and runs the standard validator. Returns a wrapped error listing the
// first few validation errors; nil if validation passes.
//
// `partial` is the GLXFile of newly-minted entities (built by the caller).
// We do NOT mutate `archive` — instead we walk partial's entity maps and
// temporarily install each entity into archive's maps for the validation pass,
// then remove them so further callers see the unmodified archive.
func runWholeArchiveValidate(archive, partial *glxlib.GLXFile) error {
	added := installPartial(archive, partial)
	// LIFO defer order matters: InvalidateCache must run AFTER
	// uninstallPartial so the cleared cache reflects the restored
	// (uninstalled) state, not the merged state.
	defer archive.InvalidateCache()
	defer uninstallPartial(archive, added)
	archive.InvalidateCache()

	result := archive.Validate()
	if len(result.Errors) == 0 {
		return nil
	}
	const showFirst = 5
	lines := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		lines = append(lines, e.Message)
	}
	sort.Strings(lines)
	if len(lines) > showFirst {
		lines = append(lines[:showFirst],
			fmt.Sprintf("… and %d more error(s)", len(result.Errors)-showFirst))
	}

	return fmt.Errorf("%w:\n  - %s", ErrValidationFailed, strings.Join(lines, "\n  - "))
}

// installPartial copies every entity in `partial` into `archive`, returning a
// list of (entityType, id) pairs so they can be removed afterwards.
func installPartial(archive, partial *glxlib.GLXFile) []addedEntity {
	archive.InvalidateCache()
	total := len(partial.Persons) + len(partial.Events) + len(partial.Relationships) +
		len(partial.Places) + len(partial.Sources) + len(partial.Citations) +
		len(partial.Repositories) + len(partial.Assertions) + len(partial.Media)
	added := make([]addedEntity, 0, total)
	added = append(added, copyEntities(&archive.Persons, partial.Persons, glxlib.EntityTypePersons)...)
	added = append(added, copyEntities(&archive.Events, partial.Events, glxlib.EntityTypeEvents)...)
	added = append(added, copyEntities(&archive.Relationships, partial.Relationships, glxlib.EntityTypeRelationships)...)
	added = append(added, copyEntities(&archive.Places, partial.Places, glxlib.EntityTypePlaces)...)
	added = append(added, copyEntities(&archive.Sources, partial.Sources, glxlib.EntityTypeSources)...)
	added = append(added, copyEntities(&archive.Citations, partial.Citations, glxlib.EntityTypeCitations)...)
	added = append(added, copyEntities(&archive.Repositories, partial.Repositories, glxlib.EntityTypeRepositories)...)
	added = append(added, copyEntities(&archive.Assertions, partial.Assertions, glxlib.EntityTypeAssertions)...)
	added = append(added, copyEntities(&archive.Media, partial.Media, glxlib.EntityTypeMedia)...)

	return added
}

type addedEntity struct {
	entityType string
	id         string
	// previous holds the pointer that lived at archive.<Map>[id] before
	// installPartial overwrote it, so uninstallPartial can restore the
	// archive's in-memory state exactly when --force overwrote a key. nil
	// means the key was newly created and should be deleted on uninstall.
	previous any
}

// copyEntities adds every key in src to dest, allocating dest if nil, and
// returns the addedEntity list. Generic over the value type so each entity
// kind can use it. If a key already exists in dest, the prior pointer is
// captured in addedEntity.previous so uninstall can restore it.
func copyEntities[T any](dest *map[string]*T, src map[string]*T, entityType string) []addedEntity {
	if len(src) == 0 {
		return nil
	}
	if *dest == nil {
		*dest = make(map[string]*T, len(src))
	}
	added := make([]addedEntity, 0, len(src))
	for id, v := range src {
		var prior any
		if existing, ok := (*dest)[id]; ok {
			prior = existing
		}
		(*dest)[id] = v
		added = append(added, addedEntity{entityType: entityType, id: id, previous: prior})
	}

	return added
}

// uninstallPartial reverses installPartial: deletes newly-installed entities
// and restores any pointer that an overwrite displaced, so the archive's
// in-memory state matches the on-disk state exactly. Without the restore,
// a --force overwrite would leave `archive` missing the prior entity even
// though the on-disk write is a no-op for it.
func uninstallPartial(archive *glxlib.GLXFile, added []addedEntity) {
	for _, e := range added {
		switch e.entityType {
		case glxlib.EntityTypePersons:
			restoreOrDelete(archive.Persons, e)
		case glxlib.EntityTypeEvents:
			restoreOrDelete(archive.Events, e)
		case glxlib.EntityTypeRelationships:
			restoreOrDelete(archive.Relationships, e)
		case glxlib.EntityTypePlaces:
			restoreOrDelete(archive.Places, e)
		case glxlib.EntityTypeSources:
			restoreOrDelete(archive.Sources, e)
		case glxlib.EntityTypeCitations:
			restoreOrDelete(archive.Citations, e)
		case glxlib.EntityTypeRepositories:
			restoreOrDelete(archive.Repositories, e)
		case glxlib.EntityTypeAssertions:
			restoreOrDelete(archive.Assertions, e)
		case glxlib.EntityTypeMedia:
			restoreOrDelete(archive.Media, e)
		}
	}
}

// restoreOrDelete restores e.previous if it holds the appropriate concrete
// pointer type, otherwise deletes the key. Concrete pointer types are
// emitted by copyEntities so the type assertion is total in practice.
func restoreOrDelete[T any](m map[string]*T, e addedEntity) {
	if e.previous == nil {
		delete(m, e.id)

		return
	}
	if prior, ok := e.previous.(*T); ok {
		m[e.id] = prior

		return
	}
	// Type mismatch should be unreachable (entityType determines T at the
	// call site). Fail closed by deleting rather than silently leaving the
	// overwrite in place.
	delete(m, e.id)
}

// finalizeAdd is the tail of every add subcommand: optionally run whole-archive
// validation, print a summary, write the partial (unless --dry-run), and echo
// the created ID on its own line so `$(glx add …)` shell substitution works.
func finalizeAdd(io *IOStreams, opts *addCommonOptions, ctx *addContext, entityType, entityID string, partial *glxlib.GLXFile) error {
	if !opts.SkipValidate {
		if err := runWholeArchiveValidate(ctx.archive, partial); err != nil {
			return err
		}
	}

	io.Printf("Adding %s %s\n", entityType, entityID)
	if opts.DryRun {
		io.Println("(dry run — no files written)")
		fmt.Fprintln(io.MachineOut, entityID) //nolint:errcheck // CLI output

		return nil
	}

	if _, err := writePartialArchive(opts.ArchivePath, partial); err != nil {
		return err
	}

	fmt.Fprintln(io.MachineOut, entityID) //nolint:errcheck // CLI output

	return nil
}

// =============================================================================
// add person
// =============================================================================

type addPersonOptions struct {
	addCommonOptions
	Given         string
	Surname       string
	Prefix        string
	SurnamePrefix string
	Suffix        string
	Nickname      string
	Sex           string
	Gender        string
	Occupation    string
	Residence     string
}

// addPerson mints a Person entity from the supplied flags.
func addPerson(io *IOStreams, opts *addPersonOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Given) == "" && strings.TrimSpace(opts.Surname) == "" && opts.OverrideID == "" {
		return ErrAddPersonNameRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabSexTypes, opts.Sex); err != nil {
		return err
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabGenderTypes, opts.Gender); err != nil {
		return err
	}
	// person_properties.residence is reference_type: places per the standard
	// vocab, so --residence is a place ID, not free text.
	if err := validateRefExists(ctx.archive, glxlib.EntityTypePlaces, opts.Residence); err != nil {
		return err
	}

	base := glxlib.EntityID(glxlib.EntityIDPrefixPerson, opts.Given+" "+opts.Surname)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Persons), opts.Force)
	if err != nil {
		return err
	}

	person := &glxlib.Person{Properties: map[string]any{}}
	if name := buildNameProperty(opts); name != nil {
		person.Properties[glxlib.PersonPropertyName] = name
	}
	if opts.Sex != "" {
		person.Properties[glxlib.PersonPropertySex] = opts.Sex
	}
	if opts.Gender != "" {
		person.Properties[glxlib.PersonPropertyGender] = opts.Gender
	}
	if opts.Occupation != "" {
		person.Properties[glxlib.PersonPropertyOccupation] = opts.Occupation
	}
	if opts.Residence != "" {
		person.Properties[glxlib.PersonPropertyResidence] = opts.Residence
	}
	if len(opts.Notes) > 0 {
		person.Notes = glxlib.NoteList(opts.Notes)
	}
	if len(person.Properties) == 0 {
		person.Properties = nil
	}

	partial := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{id: person}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypePersons, id, partial)
}

// buildNameProperty constructs a structured `name` property value from the
// person flags. Returns nil if no name fields were supplied.
func buildNameProperty(opts *addPersonOptions) map[string]any {
	fields := map[string]any{}
	addIfSet := func(k, v string) {
		if v != "" {
			fields[k] = v
		}
	}
	addIfSet(glxlib.NameFieldPrefix, opts.Prefix)
	addIfSet(glxlib.NameFieldGiven, opts.Given)
	addIfSet(glxlib.NameFieldNickname, opts.Nickname)
	addIfSet(glxlib.NameFieldSurnamePrefix, opts.SurnamePrefix)
	addIfSet(glxlib.NameFieldSurname, opts.Surname)
	addIfSet(glxlib.NameFieldSuffix, opts.Suffix)
	if len(fields) == 0 {
		return nil
	}

	return map[string]any{"value": joinNameValue(opts), "fields": fields}
}

// joinNameValue assembles the human-readable form of the name from its parts.
// Used so the `name` property has a populated top-level `value` in addition to
// the structured `fields` breakdown.
func joinNameValue(opts *addPersonOptions) string {
	parts := []string{opts.Prefix, opts.Given, opts.Nickname, opts.SurnamePrefix, opts.Surname, opts.Suffix}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}

	return strings.Join(out, " ")
}

// =============================================================================
// add place
// =============================================================================

type addPlaceOptions struct {
	addCommonOptions
	Name      string
	Type      string
	Parent    string
	Latitude  *float64
	Longitude *float64
}

// addPlace mints a Place entity from the supplied flags.
func addPlace(io *IOStreams, opts *addPlaceOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Name) == "" && opts.OverrideID == "" {
		return ErrAddPlaceNameRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabPlaceTypes, opts.Type); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypePlaces, opts.Parent); err != nil {
		return err
	}

	base := glxlib.EntityID(glxlib.EntityIDPrefixPlace, opts.Name)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Places), opts.Force)
	if err != nil {
		return err
	}

	place := &glxlib.Place{
		Name:      strings.TrimSpace(opts.Name),
		ParentID:  opts.Parent,
		Type:      opts.Type,
		Latitude:  opts.Latitude,
		Longitude: opts.Longitude,
	}
	if len(opts.Notes) > 0 {
		place.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Places: map[string]*glxlib.Place{id: place}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypePlaces, id, partial)
}

// =============================================================================
// add event
// =============================================================================

type addEventOptions struct {
	addCommonOptions
	Type         string
	Date         string
	Place        string
	Title        string
	Principal    string
	Participants []string
}

// addEvent mints an Event entity from the supplied flags. Participants are
// assembled from --principal (which adds the named person with role
// "principal") and repeated --participant flags.
func addEvent(io *IOStreams, opts *addEventOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Type) == "" {
		return ErrAddEventTypeRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabEventTypes, opts.Type); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypePlaces, opts.Place); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypePersons, opts.Principal); err != nil {
		return err
	}

	participants, err := buildEventParticipants(ctx.archive, opts)
	if err != nil {
		return err
	}

	descriptor := opts.Principal
	if descriptor == "" {
		descriptor = opts.Date
	}
	if descriptor == "" && len(participants) > 0 {
		descriptor = participants[0].Person
	}
	descriptor = stripEntityPrefix(descriptor)
	base := glxlib.EntityID(glxlib.EntityIDPrefixEvent, opts.Type+" "+descriptor)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Events), opts.Force)
	if err != nil {
		return err
	}

	event := &glxlib.Event{
		Title:        strings.TrimSpace(opts.Title),
		Type:         opts.Type,
		PlaceID:      opts.Place,
		Date:         glxlib.DateString(opts.Date),
		Participants: participants,
	}
	if len(opts.Notes) > 0 {
		event.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Events: map[string]*glxlib.Event{id: event}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeEvents, id, partial)
}

// buildEventParticipants resolves the --principal and repeated --participant
// flags into a Participant slice. Each participant person and role is
// validated against the archive's vocab maps and persons map.
func buildEventParticipants(archive *glxlib.GLXFile, opts *addEventOptions) ([]glxlib.Participant, error) {
	parts := make([]glxlib.Participant, 0, 1+len(opts.Participants))
	seen := make(map[string]struct{})
	add := func(person, role string) error {
		if err := validateRefExists(archive, glxlib.EntityTypePersons, person); err != nil {
			return err
		}
		if err := validateVocabKey(archive, glxlib.VocabParticipantRoles, role); err != nil {
			return err
		}
		key := person + ":" + role
		if _, dup := seen[key]; dup {
			return nil
		}
		seen[key] = struct{}{}
		parts = append(parts, glxlib.Participant{Person: person, Role: role})

		return nil
	}
	if opts.Principal != "" {
		if err := add(opts.Principal, glxlib.ParticipantRolePrincipal); err != nil {
			return nil, err
		}
	}
	for _, flag := range opts.Participants {
		person, role, err := parseParticipantFlag(flag)
		if err != nil {
			return nil, err
		}
		if err := add(person, role); err != nil {
			return nil, err
		}
	}

	return parts, nil
}

// =============================================================================
// add repository
// =============================================================================

type addRepositoryOptions struct {
	addCommonOptions
	Name       string
	Type       string
	Address    string
	City       string
	State      string
	PostalCode string
	Country    string
	Website    string
}

// addRepository mints a Repository entity from the supplied flags.
func addRepository(io *IOStreams, opts *addRepositoryOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Name) == "" && opts.OverrideID == "" {
		return ErrAddRepositoryNameRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabRepositoryTypes, opts.Type); err != nil {
		return err
	}

	base := glxlib.EntityID(glxlib.EntityIDPrefixRepository, opts.Name)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Repositories), opts.Force)
	if err != nil {
		return err
	}

	repo := &glxlib.Repository{
		Name:       strings.TrimSpace(opts.Name),
		Type:       opts.Type,
		Address:    opts.Address,
		City:       opts.City,
		State:      opts.State,
		PostalCode: opts.PostalCode,
		Country:    opts.Country,
		Website:    opts.Website,
	}
	if len(opts.Notes) > 0 {
		repo.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Repositories: map[string]*glxlib.Repository{id: repo}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeRepositories, id, partial)
}

// =============================================================================
// add source
// =============================================================================

type addSourceOptions struct {
	addCommonOptions
	Title       string
	Type        string
	Repository  string
	Authors     []string
	Date        string
	Description string
	Language    string
}

// addSource mints a Source entity from the supplied flags.
func addSource(io *IOStreams, opts *addSourceOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Title) == "" && opts.OverrideID == "" {
		return ErrAddSourceTitleRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabSourceTypes, opts.Type); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypeRepositories, opts.Repository); err != nil {
		return err
	}

	base := glxlib.EntityID(glxlib.EntityIDPrefixSource, opts.Title)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Sources), opts.Force)
	if err != nil {
		return err
	}

	source := &glxlib.Source{
		Title:        strings.TrimSpace(opts.Title),
		Type:         opts.Type,
		Authors:      opts.Authors,
		Date:         glxlib.DateString(opts.Date),
		RepositoryID: opts.Repository,
		Language:     opts.Language,
	}
	if opts.Description != "" {
		source.Properties = map[string]any{"description": opts.Description}
	}
	if len(opts.Notes) > 0 {
		source.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Sources: map[string]*glxlib.Source{id: source}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeSources, id, partial)
}

// =============================================================================
// add citation
// =============================================================================

type addCitationOptions struct {
	addCommonOptions
	Source         string
	Repository     string
	URL            string
	Accessed       string
	Locator        string
	TextFromSource string
	SourceDate     string
	ExternalIDs    []string
}

// addCitation mints a Citation entity from the supplied flags. The derived ID
// includes a short hash of the URL/locator/text so two citations of the same
// source do not collide on the source-derived slug.
func addCitation(io *IOStreams, opts *addCitationOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Source) == "" {
		return ErrAddCitationSourceRequired
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypeSources, opts.Source); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypeRepositories, opts.Repository); err != nil {
		return err
	}

	externalIDs, err := buildExternalIDs(opts.ExternalIDs)
	if err != nil {
		return err
	}

	// Hash one of the distinguishing fields so two citations of the same source
	// don't collide on the source-derived slug. At least one is required:
	// without --url/--locator/--text-from-source there is nothing
	// evidence-bearing to distinguish citations of one source, and falling
	// back to a timestamp would mint non-idempotent IDs.
	hashSeed := opts.URL
	if hashSeed == "" {
		hashSeed = opts.Locator
	}
	if hashSeed == "" {
		hashSeed = opts.TextFromSource
	}
	if hashSeed == "" && opts.OverrideID == "" {
		return ErrAddCitationDistinguisherRequired
	}
	sourceSlug := strings.TrimPrefix(opts.Source, glxlib.EntityIDPrefixSource)
	hashSuffix := "-" + shortHash(hashSeed)
	base := glxlib.Slugify(sourceSlug,
		glxlib.WithSlugPrefix(glxlib.EntityIDPrefixCitation),
		glxlib.WithSlugSuffix(hashSuffix),
		glxlib.WithSlugMaxLength(glxlib.MaxEntityIDLength),
	)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Citations), opts.Force)
	if err != nil {
		return err
	}

	props := map[string]any{}
	if opts.URL != "" {
		props["url"] = opts.URL
	}
	if opts.Accessed != "" {
		props["accessed"] = opts.Accessed
	}
	if opts.Locator != "" {
		props["locator"] = opts.Locator
	}
	if opts.TextFromSource != "" {
		props["text_from_source"] = opts.TextFromSource
	}
	if opts.SourceDate != "" {
		props["source_date"] = opts.SourceDate
	}
	if len(externalIDs) > 0 {
		props["external_ids"] = externalIDs
	}

	citation := &glxlib.Citation{
		SourceID:     opts.Source,
		RepositoryID: opts.Repository,
	}
	if len(props) > 0 {
		citation.Properties = props
	}
	if len(opts.Notes) > 0 {
		citation.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Citations: map[string]*glxlib.Citation{id: citation}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeCitations, id, partial)
}

// buildExternalIDs parses each --external-id flag value of the form type:value
// into the canonical external_ids entry shape via glxlib.NewExternalIDEntry.
func buildExternalIDs(flags []string) ([]any, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	out := make([]any, 0, len(flags))
	for _, f := range flags {
		typ, value, err := parseExternalIDFlag(f)
		if err != nil {
			return nil, err
		}
		out = append(out, glxlib.NewExternalIDEntry(value, typ))
	}

	return out, nil
}

// =============================================================================
// add relationship
// =============================================================================

type addRelationshipOptions struct {
	addCommonOptions
	Type         string
	Parents      []string
	Children     []string
	Spouses      []string
	Participants []string
	StartEvent   string
	EndEvent     string
}

// addRelationship mints a Relationship entity from the supplied flags. At
// least two participants must be specified via --parent/--child/--spouse or
// the generic --participant person-id:role flag.
func addRelationship(io *IOStreams, opts *addRelationshipOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	if strings.TrimSpace(opts.Type) == "" {
		return ErrAddRelationshipTypeRequired
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabRelationshipTypes, opts.Type); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypeEvents, opts.StartEvent); err != nil {
		return err
	}
	if err := validateRefExists(ctx.archive, glxlib.EntityTypeEvents, opts.EndEvent); err != nil {
		return err
	}

	participants, err := buildRelationshipParticipants(ctx.archive, opts)
	if err != nil {
		return err
	}
	if len(participants) < 2 {
		return ErrAddRelationshipParticipantsRequired
	}

	descriptor := stripEntityPrefix(participants[0].Person) + "-" + stripEntityPrefix(participants[1].Person)
	base := glxlib.EntityID(glxlib.EntityIDPrefixRelationship, opts.Type+" "+descriptor)
	id, err := deriveOrOverrideID(base, opts.OverrideID, idSet(ctx.archive.Relationships), opts.Force)
	if err != nil {
		return err
	}

	rel := &glxlib.Relationship{
		Type:         opts.Type,
		Participants: participants,
		StartEvent:   opts.StartEvent,
		EndEvent:     opts.EndEvent,
	}
	if len(opts.Notes) > 0 {
		rel.Notes = glxlib.NoteList(opts.Notes)
	}

	partial := &glxlib.GLXFile{Relationships: map[string]*glxlib.Relationship{id: rel}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeRelationships, id, partial)
}

func buildRelationshipParticipants(archive *glxlib.GLXFile, opts *addRelationshipOptions) ([]glxlib.Participant, error) {
	parts := make([]glxlib.Participant, 0, len(opts.Parents)+len(opts.Children)+len(opts.Spouses)+len(opts.Participants))
	seen := make(map[string]struct{})
	add := func(person, role string) error {
		if err := validateRefExists(archive, glxlib.EntityTypePersons, person); err != nil {
			return err
		}
		if err := validateVocabKey(archive, glxlib.VocabParticipantRoles, role); err != nil {
			return err
		}
		key := person + ":" + role
		if _, dup := seen[key]; dup {
			return nil
		}
		seen[key] = struct{}{}
		parts = append(parts, glxlib.Participant{Person: person, Role: role})

		return nil
	}
	for _, p := range opts.Parents {
		if err := add(p, glxlib.ParticipantRoleParent); err != nil {
			return nil, err
		}
	}
	for _, c := range opts.Children {
		if err := add(c, glxlib.ParticipantRoleChild); err != nil {
			return nil, err
		}
	}
	for _, s := range opts.Spouses {
		if err := add(s, glxlib.ParticipantRoleSpouse); err != nil {
			return nil, err
		}
	}
	for _, flag := range opts.Participants {
		person, role, err := parseParticipantFlag(flag)
		if err != nil {
			return nil, err
		}
		if err := add(person, role); err != nil {
			return nil, err
		}
	}

	return parts, nil
}

// =============================================================================
// add assertion
// =============================================================================

type addAssertionOptions struct {
	addCommonOptions
	SubjectPerson       string
	SubjectEvent        string
	SubjectRelationship string
	SubjectPlace        string

	Property      string
	Value         string
	Participant   string // person-id:role; mutually exclusive with property/value
	AssertionDate string
	Confidence    string
	Status        string

	Sources   []string
	Citations []string
	Media     []string
}

// addAssertion mints an Assertion entity from the supplied flags. Exactly one
// of --subject-person/event/relationship/place must be given. The payload is
// either --property + --value (a fact assertion) or --participant (a
// participant-role assertion); the two payload shapes are mutually exclusive.
func addAssertion(io *IOStreams, opts *addAssertionOptions) error {
	ctx, err := loadAddContext(io, opts.ArchivePath)
	if err != nil {
		return err
	}

	subjectType, subjectID, err := resolveAssertionSubject(ctx.archive, opts)
	if err != nil {
		return err
	}
	if err := validateAssertionPayload(opts); err != nil {
		return err
	}
	if err := validateVocabKey(ctx.archive, glxlib.VocabConfidenceLevels, opts.Confidence); err != nil {
		return err
	}
	participant, err := resolveAssertionParticipant(ctx.archive, opts)
	if err != nil {
		return err
	}
	if err := validateAssertionRefs(ctx.archive, opts); err != nil {
		return err
	}

	id, err := deriveAssertionID(ctx.archive, opts, subjectID, participant)
	if err != nil {
		return err
	}

	assertion := buildAssertion(opts, subjectType, subjectID, participant)
	partial := &glxlib.GLXFile{Assertions: map[string]*glxlib.Assertion{id: assertion}}

	return finalizeAdd(io, &opts.addCommonOptions, ctx, glxlib.EntityTypeAssertions, id, partial)
}

// validateAssertionPayload enforces the "either --property+--value OR
// --participant" payload contract. --property without --value (or vice
// versa) is rejected — half a fact assertion is not a valid shape.
func validateAssertionPayload(opts *addAssertionOptions) error {
	hasProperty := opts.Property != ""
	hasValue := opts.Value != ""
	hasParticipant := opts.Participant != ""
	hasFact := hasProperty && hasValue
	hasPartialFact := (hasProperty || hasValue) && !hasFact

	switch {
	case hasPartialFact:
		return ErrAddAssertionPayloadRequired
	case !hasFact && !hasParticipant:
		return ErrAddAssertionPayloadRequired
	case hasFact && hasParticipant:
		return ErrAddAssertionPayloadConflict
	}

	return nil
}

// resolveAssertionParticipant parses and validates the optional --participant
// flag, returning nil when --participant was not supplied.
func resolveAssertionParticipant(archive *glxlib.GLXFile, opts *addAssertionOptions) (*glxlib.Participant, error) {
	if opts.Participant == "" {
		return nil, nil
	}
	person, role, err := parseParticipantFlag(opts.Participant)
	if err != nil {
		return nil, err
	}
	if err := validateRefExists(archive, glxlib.EntityTypePersons, person); err != nil {
		return nil, err
	}
	if err := validateVocabKey(archive, glxlib.VocabParticipantRoles, role); err != nil {
		return nil, err
	}

	return &glxlib.Participant{Person: person, Role: role}, nil
}

// validateAssertionRefs reference-checks all supporting evidence IDs.
func validateAssertionRefs(archive *glxlib.GLXFile, opts *addAssertionOptions) error {
	for _, sid := range opts.Sources {
		if err := validateRefExists(archive, glxlib.EntityTypeSources, sid); err != nil {
			return err
		}
	}
	for _, cid := range opts.Citations {
		if err := validateRefExists(archive, glxlib.EntityTypeCitations, cid); err != nil {
			return err
		}
	}
	for _, mid := range opts.Media {
		if err := validateRefExists(archive, glxlib.EntityTypeMedia, mid); err != nil {
			return err
		}
	}

	return nil
}

// deriveAssertionID computes the derived/overridden assertion ID.
func deriveAssertionID(archive *glxlib.GLXFile, opts *addAssertionOptions, subjectID string, participant *glxlib.Participant) (string, error) {
	descriptor := stripEntityPrefix(subjectID)
	tail := opts.Property
	if tail == "" && participant != nil {
		tail = participant.Role
	}
	if tail == "" {
		tail = "fact"
	}
	base := glxlib.EntityID(glxlib.EntityIDPrefixAssertion, descriptor+" "+tail)

	return deriveOrOverrideID(base, opts.OverrideID, idSet(archive.Assertions), opts.Force)
}

// buildAssertion assembles the Assertion struct from the validated inputs.
func buildAssertion(opts *addAssertionOptions, subjectType, subjectID string, participant *glxlib.Participant) *glxlib.Assertion {
	assertion := &glxlib.Assertion{
		Subject:     assertionSubjectRef(subjectType, subjectID),
		Property:    opts.Property,
		Value:       opts.Value,
		Date:        glxlib.DateString(opts.AssertionDate),
		Participant: participant,
		Confidence:  opts.Confidence,
		Status:      opts.Status,
		Sources:     copyStrings(opts.Sources),
		Citations:   copyStrings(opts.Citations),
		Media:       copyStrings(opts.Media),
	}
	if len(opts.Notes) > 0 {
		assertion.Notes = glxlib.NoteList(opts.Notes)
	}

	return assertion
}

// resolveAssertionSubject enforces "exactly one of --subject-*" and returns the
// entity type + ID. Each non-empty subject flag is also reference-checked.
// Iterated as an ordered slice (not a map) so error messages, future
// conflict-reporting extensions, and tests stay deterministic across runs.
func resolveAssertionSubject(archive *glxlib.GLXFile, opts *addAssertionOptions) (string, string, error) {
	subjects := []struct{ entityType, id string }{
		{glxlib.EntityTypePersons, opts.SubjectPerson},
		{glxlib.EntityTypeEvents, opts.SubjectEvent},
		{glxlib.EntityTypeRelationships, opts.SubjectRelationship},
		{glxlib.EntityTypePlaces, opts.SubjectPlace},
	}
	var chosenType, chosenID string
	for _, s := range subjects {
		if s.id == "" {
			continue
		}
		if chosenID != "" {
			return "", "", ErrAddSubjectConflict
		}
		chosenType, chosenID = s.entityType, s.id
	}
	if chosenID == "" {
		return "", "", ErrAddSubjectRequired
	}
	if err := validateRefExists(archive, chosenType, chosenID); err != nil {
		return "", "", err
	}

	return chosenType, chosenID, nil
}

// assertionSubjectRef builds an EntityRef from the resolved subject type/id.
func assertionSubjectRef(entityType, id string) glxlib.EntityRef {
	switch entityType {
	case glxlib.EntityTypePersons:
		return glxlib.EntityRef{Person: id}
	case glxlib.EntityTypeEvents:
		return glxlib.EntityRef{Event: id}
	case glxlib.EntityTypeRelationships:
		return glxlib.EntityRef{Relationship: id}
	case glxlib.EntityTypePlaces:
		return glxlib.EntityRef{Place: id}
	}

	return glxlib.EntityRef{}
}

// =============================================================================
// small helpers
// =============================================================================

// idSet returns a set of the keys in `m`. Generic over the value type so each
// entity-type map can be passed directly.
func idSet[T any](m map[string]*T) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}

	return out
}

// stripEntityPrefix removes any known entity-type prefix from an ID so that
// derived IDs built from references read as "<new-prefix>-<distinguishing>"
// rather than "<new-prefix>-<old-prefix>-<distinguishing>".
func stripEntityPrefix(id string) string {
	prefixes := []string{
		glxlib.EntityIDPrefixPerson,
		glxlib.EntityIDPrefixEvent,
		glxlib.EntityIDPrefixRelationship,
		glxlib.EntityIDPrefixPlace,
		glxlib.EntityIDPrefixSource,
		glxlib.EntityIDPrefixCitation,
		glxlib.EntityIDPrefixRepository,
		glxlib.EntityIDPrefixAssertion,
		glxlib.EntityIDPrefixMedia,
	}
	for _, prefix := range prefixes {
		if rest, ok := strings.CutPrefix(id, prefix); ok {
			return rest
		}
	}

	return id
}

// copyStrings returns a fresh copy of `s`, or nil if `s` is empty. Used when
// assigning user-supplied slices to entity fields to avoid the runner's caller
// inadvertently mutating archive state through a shared slice.
func copyStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)

	return out
}
