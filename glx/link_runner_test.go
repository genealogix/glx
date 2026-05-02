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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// initArchiveDir creates a fresh empty multi-file archive for tests. Returns
// the path. Test cleanup is handled by t.TempDir(), so no manual removal.
func initArchiveDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := runInit(dir, false, 0); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	return dir
}

func TestLink_CreatesRepoSourceCitation(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:          "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		ArchivePath:       dir,
		CreateSourceTitle: "Deutschland Geburten und Taufen, 1558-1898",
		Text:              "Test entry",
	})
	if err != nil {
		t.Fatalf("linkFamilySearchARK: %v", err)
	}

	archive, _, err := LoadArchiveWithOptions(dir, false)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions: %v", err)
	}

	repo, ok := archive.Repositories[repoFamilySearchID]
	if !ok {
		t.Fatalf("repository %s was not created", repoFamilySearchID)
	}
	if repo.Name != "FamilySearch" {
		t.Errorf("repository name: got %q, want FamilySearch", repo.Name)
	}

	citationID := "citation-familysearch-c4h8-2dw2"
	cit, ok := archive.Citations[citationID]
	if !ok {
		t.Fatalf("citation %s was not created", citationID)
	}
	if cit.SourceID == "" {
		t.Errorf("citation source ID is empty")
	}
	if _, ok := archive.Sources[cit.SourceID]; !ok {
		t.Errorf("citation source %s does not exist", cit.SourceID)
	}
	if got := cit.Properties["url"]; got != "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2" {
		t.Errorf("citation url: got %v", got)
	}
	if got := cit.Properties["text_from_source"]; got != "Test entry" {
		t.Errorf("citation text_from_source: got %v", got)
	}
	// External IDs must have the structured {value, fields.type} shape.
	extIDs, ok := cit.Properties["external_ids"].([]any)
	if !ok || len(extIDs) != 1 {
		t.Fatalf("external_ids: got %v", cit.Properties["external_ids"])
	}
	entry, ok := extIDs[0].(map[string]any)
	if !ok {
		t.Fatalf("external_ids[0] not a map: %T", extIDs[0])
	}
	if entry["value"] != "1:1:C4H8-2DW2" {
		t.Errorf("external_ids value: got %v", entry["value"])
	}
	fields, _ := entry["fields"].(map[string]any)
	if fields["type"] != arkTypeURI {
		t.Errorf("external_ids fields.type: got %v, want %s", fields["type"], arkTypeURI)
	}
}

func TestLink_RequiresSourceFlag(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:    "ark:/61903/1:1:TEST-AAAA",
		ArchivePath: dir,
	})
	if err == nil {
		t.Fatal("expected error when neither --source nor --create-source is provided")
	}
	if !strings.Contains(err.Error(), "--source") || !strings.Contains(err.Error(), "--create-source") {
		t.Errorf("error message should mention both flags: %v", err)
	}
}

func TestLink_RejectsBothSourceFlags(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:          "ark:/61903/1:1:TEST-AAAA",
		ArchivePath:       dir,
		SourceID:          "source-x",
		CreateSourceTitle: "Also X",
	})
	if err == nil {
		t.Fatal("expected error when both flags are provided")
	}
}

func TestLink_MissingSourceErrors(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:    "ark:/61903/1:1:TEST-AAAA",
		ArchivePath: dir,
		SourceID:    "source-does-not-exist",
	})
	if err == nil {
		t.Fatal("expected error for unknown --source")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should indicate source was not found: %v", err)
	}
}

func TestLink_Idempotent(t *testing.T) {
	dir := initArchiveDir(t)
	io1, out1, _ := TestIOStreams()

	opts := linkOptions{
		ARKInput:          "ark:/61903/1:1:IDEM-0001",
		ArchivePath:       dir,
		CreateSourceTitle: "Test Collection",
	}
	if err := linkFamilySearchARK(io1, opts); err != nil {
		t.Fatalf("first link: %v", err)
	}
	if strings.Contains(out1.String(), "already exists") {
		t.Errorf("first run should not report existing citation; out=%s", out1.String())
	}

	// Count citation files after first run.
	firstCount := countFiles(t, filepath.Join(dir, "citations"))
	if firstCount == 0 {
		t.Fatalf("expected at least one citation file after first link")
	}

	io2, out2, _ := TestIOStreams()
	// Second run must not create a new source either — switch to --source.
	opts2 := linkOptions{
		ARKInput:    opts.ARKInput,
		ArchivePath: dir,
		SourceID:    "source-test-collection",
	}
	if err := linkFamilySearchARK(io2, opts2); err != nil {
		t.Fatalf("second link: %v", err)
	}
	if !strings.Contains(out2.String(), "already exists") {
		t.Errorf("second run should report existing citation; out=%s", out2.String())
	}

	secondCount := countFiles(t, filepath.Join(dir, "citations"))
	if secondCount != firstCount {
		t.Errorf("idempotent re-run changed citation count: %d -> %d", firstCount, secondCount)
	}
}

func TestLink_DryRunWritesNothing(t *testing.T) {
	dir := initArchiveDir(t)
	io, out, _ := TestIOStreams()

	before := countFiles(t, filepath.Join(dir, "citations"))

	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:          "ark:/61903/1:1:DRY-0001",
		ArchivePath:       dir,
		CreateSourceTitle: "Dry Run Collection",
		DryRun:            true,
	})
	if err != nil {
		t.Fatalf("linkFamilySearchARK: %v", err)
	}
	if !strings.Contains(out.String(), "dry run") {
		t.Errorf("expected 'dry run' marker in output: %s", out.String())
	}

	after := countFiles(t, filepath.Join(dir, "citations"))
	if after != before {
		t.Errorf("dry-run wrote files: %d -> %d", before, after)
	}
}

func TestLink_ReusesSourceForSameTitleAcrossRecords(t *testing.T) {
	// Simulates a researcher reviewing many FamilySearch records from the
	// same collection: each record produces a different ARK, but all share
	// the same --create-source title. The MVP must reuse the source on
	// subsequent calls rather than mint source-foo, source-foo-2, source-foo-3.
	dir := initArchiveDir(t)
	title := "Deutschland Geburten und Taufen, 1558-1898"

	arks := []string{
		"ark:/61903/1:1:AAAA-0001",
		"ark:/61903/1:1:BBBB-0002",
		"ark:/61903/1:1:CCCC-0003",
	}
	for _, ark := range arks {
		io, _, _ := TestIOStreams()
		err := linkFamilySearchARK(io, linkOptions{
			ARKInput:          ark,
			ArchivePath:       dir,
			CreateSourceTitle: title,
		})
		if err != nil {
			t.Fatalf("link %s: %v", ark, err)
		}
	}

	archive, _, err := LoadArchiveWithOptions(dir, false)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions: %v", err)
	}

	// Exactly one source with the matching title must exist.
	var sourceIDs []string
	for id, src := range archive.Sources {
		if src.Title == title {
			sourceIDs = append(sourceIDs, id)
		}
	}
	if len(sourceIDs) != 1 {
		t.Fatalf("expected exactly 1 source with title %q, got %d: %v", title, len(sourceIDs), sourceIDs)
	}
	sourceID := sourceIDs[0]

	// Every citation must reference that single source.
	for _, ark := range arks {
		parsed, _ := ParseFamilySearchARK(ark)
		citID := "citation-familysearch-" + parsed.CitationIDSlug()
		cit, ok := archive.Citations[citID]
		if !ok {
			t.Fatalf("citation %s missing", citID)
		}
		if cit.SourceID != sourceID {
			t.Errorf("citation %s: SourceID=%s, want %s", citID, cit.SourceID, sourceID)
		}
	}
}

func TestLink_PreservesExistingRepository(t *testing.T) {
	dir := initArchiveDir(t)

	// Seed a customized repository-familysearch entity in the archive.
	seedRepoPath := filepath.Join(dir, "repositories", "repository-familysearch.glx")
	seedYAML := []byte(`repositories:
  repository-familysearch:
    name: "FamilySearch (custom)"
    type: database
    website: "https://www.familysearch.org"
    notes: "Do not clobber me"
`)
	if err := os.WriteFile(seedRepoPath, seedYAML, 0o644); err != nil {
		t.Fatalf("seed repo: %v", err)
	}

	io, _, _ := TestIOStreams()
	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:          "ark:/61903/1:1:PRESERVE-01",
		ArchivePath:       dir,
		CreateSourceTitle: "Preserve Test",
	})
	if err != nil {
		t.Fatalf("linkFamilySearchARK: %v", err)
	}

	archive, _, err := LoadArchiveWithOptions(dir, false)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions: %v", err)
	}
	repo := archive.Repositories[repoFamilySearchID]
	if repo == nil {
		t.Fatal("repository went missing")
	}
	if repo.Name != "FamilySearch (custom)" {
		t.Errorf("existing repository was overwritten; name=%q", repo.Name)
	}
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read dir %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}

	return n
}

func TestLink_RejectsWhitespaceOnlyCreateSource(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	err := linkFamilySearchARK(io, linkOptions{
		ARKInput:          "ark:/61903/1:1:WS-0001",
		ArchivePath:       dir,
		CreateSourceTitle: "   \t  \n  ",
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only --create-source")
	}
	if !errors.Is(err, ErrLinkSourceRequired) {
		t.Errorf("expected ErrLinkSourceRequired, got %v", err)
	}
}

func TestLink_TrimsSourceIDWhitespace(t *testing.T) {
	// SourceID with surrounding whitespace should resolve to the trimmed ID,
	// not be rejected as unknown.
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	// Seed a real source so the trimmed ID resolves.
	if err := linkFamilySearchARK(io, linkOptions{
		ARKInput:          "ark:/61903/1:1:SEED-0001",
		ArchivePath:       dir,
		CreateSourceTitle: "Trim Test Collection",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	io2, _, _ := TestIOStreams()
	err := linkFamilySearchARK(io2, linkOptions{
		ARKInput:    "ark:/61903/1:1:TRIM-0002",
		ArchivePath: dir,
		SourceID:    "  source-trim-test-collection  ",
	})
	if err != nil {
		t.Fatalf("linkFamilySearchARK with padded --source: %v", err)
	}
}

func TestCitationIDFor(t *testing.T) {
	t.Run("short NOID: no truncation", func(t *testing.T) {
		ark, _ := ParseFamilySearchARK("ark:/61903/1:1:C4H8-2DW2")
		got := citationIDFor(ark)
		want := "citation-familysearch-c4h8-2dw2"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
		if len(got) > maxEntityIDLength {
			t.Errorf("short NOID produced over-long ID: %d > %d", len(got), maxEntityIDLength)
		}
	})

	t.Run("long NOID: truncated with hash suffix, stays within cap", func(t *testing.T) {
		long := "1:1:" + strings.Repeat("ABCD-", 20) + "END"
		ark, _ := ParseFamilySearchARK("ark:/61903/" + long)
		got := citationIDFor(ark)
		if len(got) > maxEntityIDLength {
			t.Errorf("long NOID produced over-long ID: %d > %d (id=%q)", len(got), maxEntityIDLength, got)
		}
		if !strings.HasPrefix(got, citationIDPrefixFS) {
			t.Errorf("missing prefix: %q", got)
		}
		// Must be deterministic.
		if citationIDFor(ark) != got {
			t.Errorf("citationIDFor not deterministic for same NOID")
		}
	})

	t.Run("long NOIDs with same prefix produce different IDs", func(t *testing.T) {
		prefix := "1:1:" + strings.Repeat("XY-", 20)
		a, _ := ParseFamilySearchARK("ark:/61903/" + prefix + "ONE")
		b, _ := ParseFamilySearchARK("ark:/61903/" + prefix + "TWO")
		idA := citationIDFor(a)
		idB := citationIDFor(b)
		if idA == idB {
			t.Errorf("distinct NOIDs %q and %q produced same citation ID %q", a.NOID, b.NOID, idA)
		}
	})

	t.Run("matches schema pattern", func(t *testing.T) {
		entityIDPattern := regexp.MustCompile(`^[a-zA-Z0-9-]{1,64}$`)
		noids := []string{
			"1:1:C4H8-2DW2",
			"1:1:" + strings.Repeat("AB-", 30),
			"FOO",
			"2:1:abc-def",
		}
		for _, n := range noids {
			ark := &ARK{NOID: n}
			id := citationIDFor(ark)
			if !entityIDPattern.MatchString(id) {
				t.Errorf("citationIDFor(%q) = %q; does not match %s", n, id, entityIDPattern)
			}
		}
	})
}

func TestNextUniqueSourceID_CapsAtCollisionLimit(t *testing.T) {
	// Seed an archive whose source map contains source-x and source-x-2..N
	// where N exceeds the collision cap. nextUniqueSourceID must return the
	// ErrLinkSourceIDExhausted sentinel rather than loop forever.
	archive := &glxlib.GLXFile{Sources: map[string]*glxlib.Source{}}
	archive.Sources["source-x"] = &glxlib.Source{Title: "X"}
	for i := 2; i <= maxSourceIDCollisions; i++ {
		archive.Sources[fmt.Sprintf("source-x-%d", i)] = &glxlib.Source{Title: "X"}
	}

	_, err := nextUniqueSourceID("X", archive)
	if err == nil {
		t.Fatal("expected ErrLinkSourceIDExhausted when every candidate slot is taken")
	}
	if !errors.Is(err, ErrLinkSourceIDExhausted) {
		t.Errorf("expected ErrLinkSourceIDExhausted, got %v", err)
	}
}

func TestSlugifyForID(t *testing.T) {
	cases := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"Deutschland Geburten und Taufen, 1558-1898", 60, "deutschland-geburten-und-taufen-1558-1898"},
		{"Hello, World!", 60, "hello-world"},
		{"   ", 60, "unknown"},
		{"---abc---", 60, "abc"},
		{"A Very Long Title That Exceeds Sixty Characters To Test Trimming Behavior", 10, "a-very-lon"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := slugifyForID(tc.in, tc.maxLen); got != tc.want {
				t.Errorf("slugifyForID(%q, %d) = %q, want %q", tc.in, tc.maxLen, got, tc.want)
			}
		})
	}
}
