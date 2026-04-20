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
	"fmt"
	"regexp"
	"strings"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	repoFamilySearchID     = "repository-familysearch"
	citationIDPrefixFS     = "citation-familysearch-"
	sourceIDPrefix         = "source-"
	externalIDsPropertyKey = "external_ids"
	maxEntityIDLength      = 64
	// maxSourceIDCollisions bounds the -2, -3, ... suffix search in
	// nextUniqueSourceID. Realistic archives never need more than a handful,
	// so anything beyond this signals a pathological input (slug of the
	// title collides with an adversarially-constructed source namespace).
	maxSourceIDCollisions = 1000
)

type linkOptions struct {
	ARKInput          string
	ArchivePath       string
	SourceID          string // existing source to attach; exclusive with CreateSourceTitle
	CreateSourceTitle string
	Text              string
	Locator           string
	DryRun            bool
}

// linkFamilySearchARK creates a citation (and, when needed, a repository and
// source) in the archive from a FamilySearch ARK URL. No network I/O.
func linkFamilySearchARK(io *IOStreams, opts linkOptions) error {
	if err := validateLinkOptions(opts); err != nil {
		return err
	}

	ark, err := ParseFamilySearchARK(opts.ARKInput)
	if err != nil {
		return err
	}

	archive, duplicates, err := LoadArchiveWithOptions(opts.ArchivePath, false)
	if err != nil {
		return fmt.Errorf("loading archive: %w", err)
	}
	for _, d := range duplicates {
		io.Errorf("Warning: %s\n", d)
	}

	citationID := citationIDPrefixFS + ark.CitationIDSlug()

	// Idempotence: citation with this ID already exists -> treat as no-op.
	// A finer-grained external_ids match is a follow-up (#87).
	if _, ok := archive.Citations[citationID]; ok {
		io.Printf("citation already exists: %s\n", citationID)

		return nil
	}

	newEntities, sourceID, err := buildLinkEntities(archive, ark, opts)
	if err != nil {
		return err
	}
	_, sourceIsNew := newEntities.Sources[sourceID]
	_, repoIsNew := newEntities.Repositories[repoFamilySearchID]

	printLinkSummary(io, citationID, sourceID, sourceIsNew, repoIsNew)

	if opts.DryRun {
		io.Println("")
		io.Println("(dry run — no files written)")

		return nil
	}

	if _, err := writePartialArchive(opts.ArchivePath, newEntities); err != nil {
		return err
	}

	io.Printf("Created citation %s in %s\n", citationID, opts.ArchivePath)

	return nil
}

func validateLinkOptions(opts linkOptions) error {
	if opts.SourceID == "" && opts.CreateSourceTitle == "" {
		return ErrLinkSourceRequired
	}
	if opts.SourceID != "" && opts.CreateSourceTitle != "" {
		return ErrLinkSourceConflict
	}

	return nil
}

// buildLinkEntities assembles the new repository/source/citation entities in
// memory. Returns the partial GLXFile and the resolved source ID (the
// --source value, or the ID of the source created or reused from
// --create-source).
func buildLinkEntities(archive *glxlib.GLXFile, ark *ARK, opts linkOptions) (*glxlib.GLXFile, string, error) {
	newEntities := &glxlib.GLXFile{}

	// Never clobber an existing repository-familysearch — the user may have
	// customized it.
	if _, ok := archive.Repositories[repoFamilySearchID]; !ok {
		newEntities.Repositories = map[string]*glxlib.Repository{
			repoFamilySearchID: {
				Name:    "FamilySearch",
				Type:    "database",
				Website: "https://www.familysearch.org",
			},
		}
	}

	sourceID, err := resolveSource(archive, newEntities, opts)
	if err != nil {
		return nil, "", err
	}

	citation := &glxlib.Citation{
		SourceID:     sourceID,
		RepositoryID: repoFamilySearchID,
		Properties: map[string]any{
			"url":                  ark.CanonicalURL,
			"accessed":             time.Now().UTC().Format("2006-01-02"),
			externalIDsPropertyKey: []any{glxlib.NewExternalIDEntry(ark.NOID, arkTypeURI)},
		},
	}
	if opts.Text != "" {
		citation.Properties["text_from_source"] = opts.Text
	}
	if opts.Locator != "" {
		citation.Properties["locator"] = opts.Locator
	}
	citationID := citationIDPrefixFS + ark.CitationIDSlug()
	newEntities.Citations = map[string]*glxlib.Citation{citationID: citation}

	return newEntities, sourceID, nil
}

// resolveSource picks (or creates) the source to attach the new citation to.
// When --create-source is used:
//   - An existing source with the same title and repository-familysearch is
//     reused (so reviewing many records from the same FS collection does not
//     mint a new source per record).
//   - Otherwise a fresh source is added to newEntities.
func resolveSource(archive, newEntities *glxlib.GLXFile, opts linkOptions) (string, error) {
	if opts.SourceID != "" {
		if _, ok := archive.Sources[opts.SourceID]; !ok {
			return "", fmt.Errorf("%w: %s", ErrLinkSourceNotFound, opts.SourceID)
		}

		return opts.SourceID, nil
	}

	// Reuse any existing source with the same title + FamilySearch repository.
	// This is the common case when reviewing many records from one collection.
	if existingID := findMatchingSourceID(archive, opts.CreateSourceTitle); existingID != "" {
		return existingID, nil
	}

	sourceID, err := nextUniqueSourceID(opts.CreateSourceTitle, archive)
	if err != nil {
		return "", err
	}
	newEntities.Sources = map[string]*glxlib.Source{
		sourceID: {
			Title:        opts.CreateSourceTitle,
			Type:         "database",
			RepositoryID: repoFamilySearchID,
			Notes: glxlib.NoteList{
				"Created via `glx link` from a FamilySearch ARK. The collection title " +
					"is the value supplied on the command line and may not match the " +
					"official FamilySearch collection name. Edit as needed.",
			},
		},
	}

	return sourceID, nil
}

// findMatchingSourceID returns the ID of an existing source in the archive
// whose Title equals title and whose RepositoryID points at FamilySearch.
// Returns "" when no match is found.
func findMatchingSourceID(archive *glxlib.GLXFile, title string) string {
	for id, src := range archive.Sources {
		if src == nil {
			continue
		}
		if src.Title == title && src.RepositoryID == repoFamilySearchID {
			return id
		}
	}

	return ""
}

// printLinkSummary reports the entities that will (or would, for --dry-run) be
// created. Mirrors the census-add summary style.
func printLinkSummary(io *IOStreams, citationID, sourceID string, sourceIsNew, repoIsNew bool) {
	io.Println("Link Summary")
	io.Println("============")
	io.Printf("  Citation:   %s (new)\n", citationID)
	io.Printf("  Source:     %s (%s)\n", sourceID, newOrExisting(sourceIsNew))
	io.Printf("  Repository: %s (%s)\n", repoFamilySearchID, newOrExisting(repoIsNew))
}

func newOrExisting(isNew bool) string {
	if isNew {
		return "new"
	}

	return "existing"
}

// nextUniqueSourceID returns a source ID built from the title slug, avoiding
// collisions with existing archive sources by appending -2, -3, etc. Returns
// ErrLinkSourceIDExhausted if every candidate up through maxSourceIDCollisions
// is taken — realistically unreachable, but cheap insurance against an
// adversarially-constructed archive.
func nextUniqueSourceID(title string, archive *glxlib.GLXFile) (string, error) {
	base := sourceIDPrefix + slugifyForID(title, maxEntityIDLength-len(sourceIDPrefix))
	if _, taken := archive.Sources[base]; !taken {
		return base, nil
	}
	for i := 2; i <= maxSourceIDCollisions; i++ {
		suffix := fmt.Sprintf("-%d", i)
		candidate := trimToMaxLen(base, maxEntityIDLength-len(suffix)) + suffix
		if _, taken := archive.Sources[candidate]; !taken {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrLinkSourceIDExhausted, base)
}

var slugNonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// slugifyForID lowercases the input, replaces runs of non-alphanumerics with a
// single hyphen, trims leading/trailing hyphens, and truncates to maxLen.
// Produces a value matching the GLX entity ID pattern `[a-zA-Z0-9-]{1,64}`.
// Falls back to "unknown" if the input contains no alphanumerics.
func slugifyForID(s string, maxLen int) string {
	s = strings.ToLower(s)
	s = slugNonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "unknown"
	}

	return trimToMaxLen(s, maxLen)
}

func trimToMaxLen(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}

	return strings.TrimRight(s[:maxLen], "-")
}
