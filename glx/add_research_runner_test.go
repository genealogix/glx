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
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// writeAddTestFile writes content to rel inside dir, creating parents.
func writeAddTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// add research-log
// =============================================================================

func TestAddResearchLog_HappyPath(t *testing.T) {
	dir := initArchiveDir(t)
	io, out, _ := TestIOStreams()
	if err := addPerson(io, &addPersonOptions{ArchivePath: dir, Given: "Lewis", Surname: "Little"}); err != nil {
		t.Fatalf("setup person: %v", err)
	}
	out.Reset()

	err := addResearchLog(io, &addResearchLogOptions{
		ArchivePath: dir, Notes: []string{"start with probate"},
		SubjectPerson: "person-lewis-little",
		Objective:     "Verify the reported 1826 intestate death",
		Status:        "in_progress",
		Date:          "2026-09-17",
		Researcher:    "I. Schepp",
	})
	if err != nil {
		t.Fatalf("addResearchLog: %v", err)
	}

	want := "research-log-lewis-little"
	if got := trailingLine(out.String()); got != want {
		t.Errorf("trailing stdout line: got %q, want %q", got, want)
	}
	log, ok := readBackArchive(t, dir).ResearchLogs[want]
	if !ok {
		t.Fatalf("research log %s missing", want)
	}
	if log.Subject == nil || log.Subject.Person != "person-lewis-little" {
		t.Errorf("subject: got %+v", log.Subject)
	}
	if log.Status != "in_progress" || log.Researcher != "I. Schepp" || string(log.Date) != "2026-09-17" {
		t.Errorf("fields wrong: %+v", log)
	}
	if len(log.Notes) != 1 {
		t.Errorf("notes: got %v", log.Notes)
	}
}

func TestAddResearchLog_IDDerivation(t *testing.T) {
	cases := []struct {
		name string
		opts addResearchLogOptions
		want string
	}{
		{"title wins", addResearchLogOptions{Title: "1860 Census", Objective: "find them"}, "research-log-1860-census"},
		{"objective when nothing else", addResearchLogOptions{Objective: "Find Jane Webb"}, "research-log-find-jane-webb"},
		{"explicit id", addResearchLogOptions{OverrideID: "rl-death"}, "rl-death"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := initArchiveDir(t)
			io, out, _ := TestIOStreams()
			tc.opts.ArchivePath = dir
			if err := addResearchLog(io, &tc.opts); err != nil {
				t.Fatalf("addResearchLog: %v", err)
			}
			if got := trailingLine(out.String()); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAddResearchLog_Rejections(t *testing.T) {
	dir := initArchiveDir(t)
	cases := []struct {
		name string
		opts addResearchLogOptions
		want error
	}{
		{"nothing to name it by", addResearchLogOptions{Status: "open"}, ErrAddResearchLogDescriptorRequired},
		{"bad status", addResearchLogOptions{Title: "x", Status: "bogus"}, ErrAddVocabKeyUnknown},
		{"dangling subject", addResearchLogOptions{SubjectPlace: "place-nope"}, ErrAddRefNotFound},
		{"two subjects", addResearchLogOptions{SubjectPlace: "place-a", SubjectPerson: "person-b"}, ErrAddSubjectConflict},
		{"dangling citation", addResearchLogOptions{Title: "x", Citations: []string{"citation-nope"}}, ErrAddRefNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			io, _, _ := TestIOStreams()
			tc.opts.ArchivePath = dir
			if err := addResearchLog(io, &tc.opts); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
	if logs := readBackArchive(t, dir).ResearchLogs; len(logs) != 0 {
		t.Errorf("rejected adds wrote %d logs", len(logs))
	}
}

// =============================================================================
// add search
// =============================================================================

func TestAddSearch_AppendsAndRollsUpCitation(t *testing.T) {
	dir := initArchiveDir(t)
	io, out, _ := TestIOStreams()
	mustAdd := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	mustAdd(addSource(io, &addSourceOptions{ArchivePath: dir, Title: "Find a Grave"}))
	mustAdd(addCitation(io, &addCitationOptions{ArchivePath: dir, OverrideID: "citation-fag", Source: "source-find-a-grave"}))
	mustAdd(addResearchLog(io, &addResearchLogOptions{ArchivePath: dir, OverrideID: "rl-death"}))
	out.Reset()

	mustAdd(addSearch(io, &addSearchOptions{
		ArchivePath: dir, Log: "rl-death", Source: "source-find-a-grave",
		Query: "Lewis Little", Result: "not_found", Date: "2026-09-17",
	}))
	mustAdd(addSearch(io, &addSearchOptions{
		ArchivePath: dir, Log: "rl-death", Collection: "Probate", Result: "found",
		Citation: "citation-fag", Notes: []string{"hit"},
	}))
	// A second search producing the same citation does not duplicate it.
	mustAdd(addSearch(io, &addSearchOptions{ArchivePath: dir, Log: "rl-death", Citation: "citation-fag"}))

	if got := trailingLine(out.String()); got != "rl-death" {
		t.Errorf("trailing stdout line: got %q, want rl-death", got)
	}
	log := readBackArchive(t, dir).ResearchLogs["rl-death"]
	if len(log.Searches) != 3 {
		t.Fatalf("searches: got %d, want 3", len(log.Searches))
	}
	first := log.Searches[0]
	if first.SourceID != "source-find-a-grave" || first.Result != "not_found" || string(first.Date) != "2026-09-17" {
		t.Errorf("first search wrong: %+v", first)
	}
	if log.Searches[1].Collection != "Probate" || log.Searches[1].CitationID != "citation-fag" {
		t.Errorf("second search wrong: %+v", log.Searches[1])
	}
	if len(log.Citations) != 1 || log.Citations[0] != "citation-fag" {
		t.Errorf("log citations: got %v, want [citation-fag]", log.Citations)
	}
}

// A log that shares a file with other entities is edited where it lives, and
// the other entities in that file survive untouched.
func TestAddSearch_LogInSharedFile(t *testing.T) {
	dir := initArchiveDir(t)
	writeAddTestFile(t, dir, "research.glx", `# research notes
places:
  place-leeds:
    name: "Leeds"
research_logs:
  rl-leeds:
    objective: "Leeds burials"   # keep this comment
    searches:
      - collection: "Leeds burials 1850-1860"
        result: not_found
`)
	io, _, _ := TestIOStreams()
	if err := addSearch(io, &addSearchOptions{ArchivePath: dir, Log: "rl-leeds", Query: "Smith", Result: "partial"}); err != nil {
		t.Fatalf("addSearch: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "research.glx"))
	if err != nil {
		t.Fatal(err)
	}
	want := `# research notes
places:
  place-leeds:
    name: "Leeds"
research_logs:
  rl-leeds:
    objective: "Leeds burials" # keep this comment
    searches:
      - collection: "Leeds burials 1850-1860"
        result: not_found
      - query: Smith
        result: partial
`
	if got := strings.ReplaceAll(string(data), "\r\n", "\n"); got != want {
		t.Errorf("file after append:\n%s\nwant:\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "research_logs", "rl-leeds.glx")); !os.IsNotExist(err) {
		t.Errorf("add search must not create a per-entity file for a log stored elsewhere")
	}
}

// A flow-style searches list cannot be appended to at the node level; the
// file is re-serialized instead and still carries every search.
func TestAddSearch_FallsBackToSerializerForFlowStyle(t *testing.T) {
	dir := initArchiveDir(t)
	writeAddTestFile(t, dir, "research_logs/rl-flow.glx", "research_logs:\n  rl-flow:\n    searches: [{query: a, result: found}]\n")
	io, _, _ := TestIOStreams()
	if err := addSearch(io, &addSearchOptions{ArchivePath: dir, Log: "rl-flow", Query: "b"}); err != nil {
		t.Fatalf("addSearch: %v", err)
	}
	log := readBackArchive(t, dir).ResearchLogs["rl-flow"]
	if len(log.Searches) != 2 || log.Searches[0].Query != "a" || log.Searches[1].Query != "b" {
		t.Errorf("searches after fallback: %+v", log.Searches)
	}
}

// An alias must not turn one append into changes to other logs or entities.
func TestAddSearch_SharedAnchorsOnlyUpdateTarget(t *testing.T) {
	cases := map[string]string{
		"searches": `research_logs:
  rl-a:
    searches: &shared
      - query: initial
  rl-b:
    searches: *shared
`,
		"log mapping": `research_logs:
  rl-a: &shared
    searches:
      - query: initial
  rl-b: *shared
`,
		"citations shared with assertion": `research_logs:
  rl-a:
    citations: &shared
      - citation-old
assertions:
  assertion-a:
    subject:
      person: person-a
    citations: *shared
`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			serializer := createSerializer(false, true, "  ")
			before, err := serializer.DeserializeSingleFileBytes([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			op, err := planAddSearchWrite(map[string][]byte{"shared.glx": []byte(content)}, "rl-a",
				&glxlib.Search{Query: "only target", CitationID: "citation-new"})
			if err != nil {
				t.Fatal(err)
			}
			after, err := serializer.DeserializeSingleFileBytes(op.newData)
			if err != nil {
				t.Fatal(err)
			}
			log := after.ResearchLogs["rl-a"]
			if len(log.Searches) != len(before.ResearchLogs["rl-a"].Searches)+1 ||
				log.Searches[len(log.Searches)-1].Query != "only target" {
				t.Errorf("target search was not appended: %+v", log.Searches)
			}
			if log.Citations[len(log.Citations)-1] != "citation-new" {
				t.Errorf("target citation was not rolled up: %v", log.Citations)
			}
			if !reflect.DeepEqual(after.ResearchLogs["rl-b"], before.ResearchLogs["rl-b"]) {
				t.Errorf("unrelated log changed: %+v", after.ResearchLogs["rl-b"])
			}
			if !reflect.DeepEqual(after.Assertions, before.Assertions) {
				t.Errorf("unrelated assertions changed: %+v", after.Assertions)
			}
		})
	}
}

func TestAddSearch_FindsEscapedIDAndRejectsEscapedDuplicate(t *testing.T) {
	const escaped = "research_logs:\n  \"\\x72l-x\":\n    objective: x\n"
	files := map[string][]byte{"escaped.glx": []byte(escaped)}
	op, err := planAddSearchWrite(files, "rl-x", &glxlib.Search{Query: "new"})
	if err != nil {
		t.Fatal(err)
	}
	serializer := createSerializer(false, true, "  ")
	fragment, err := serializer.DeserializeSingleFileBytes(op.newData)
	if err != nil {
		t.Fatal(err)
	}
	if got := fragment.ResearchLogs["rl-x"].Searches; len(got) != 1 || got[0].Query != "new" {
		t.Errorf("escaped log append: %+v", got)
	}
	files["literal.glx"] = []byte("research_logs:\n  rl-x:\n    objective: another\n")
	if _, err := planAddSearchWrite(files, "rl-x", &glxlib.Search{Query: "new"}); !errors.Is(err, ErrAddSearchLogAmbiguous) {
		t.Errorf("escaped duplicate: got %v, want ErrAddSearchLogAmbiguous", err)
	}
}

// --result is checked against the archive's loaded vocabulary, so values an
// archive adds to search_result_types are accepted without a code change.
func TestAddSearch_AcceptsArchiveDefinedResult(t *testing.T) {
	dir := initArchiveDir(t)
	path := filepath.Join(dir, "vocabularies", "search-result-types.glx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	extra := "\n  archive_only:\n    label: \"Archive Only\"\n    description: \"A value only this archive defines.\"\n"
	if err := os.WriteFile(path, append(data, []byte(extra)...), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAddTestFile(t, dir, "research_logs/rl-x.glx", "research_logs:\n  rl-x:\n    objective: x\n")

	io, _, _ := TestIOStreams()
	if err := addSearch(io, &addSearchOptions{ArchivePath: dir, Log: "rl-x", Collection: "Parish chest", Result: "archive_only"}); err != nil {
		t.Fatalf("addSearch with archive-defined result: %v", err)
	}
	if got := readBackArchive(t, dir).ResearchLogs["rl-x"].Searches[0].Result; got != "archive_only" {
		t.Errorf("result: got %q", got)
	}
}

func TestAddSearch_Rejections(t *testing.T) {
	dir := initArchiveDir(t)
	writeAddTestFile(t, dir, "research_logs/rl-x.glx", "research_logs:\n  rl-x:\n    objective: x\n")
	before, err := os.ReadFile(filepath.Join(dir, "research_logs", "rl-x.glx"))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		opts addSearchOptions
		want error
	}{
		{"no log", addSearchOptions{Query: "x"}, ErrAddSearchLogRequired},
		{"unknown log", addSearchOptions{Log: "rl-nope", Query: "x"}, ErrAddRefNotFound},
		{"nothing searched", addSearchOptions{Log: "rl-x", Result: "found"}, ErrAddSearchWhatRequired},
		{"bad result", addSearchOptions{Log: "rl-x", Query: "x", Result: "bogus"}, ErrAddVocabKeyUnknown},
		{"dangling source", addSearchOptions{Log: "rl-x", Source: "source-nope"}, ErrAddRefNotFound},
		{"dangling repository", addSearchOptions{Log: "rl-x", Repository: "repository-nope"}, ErrAddRefNotFound},
		{"dangling citation", addSearchOptions{Log: "rl-x", Citation: "citation-nope"}, ErrAddRefNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			io, _, _ := TestIOStreams()
			tc.opts.ArchivePath = dir
			if err := addSearch(io, &tc.opts); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}

	after, err := os.ReadFile(filepath.Join(dir, "research_logs", "rl-x.glx"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Errorf("rejected searches modified the log file")
	}
}

func TestAddSearch_DryRunWritesNothing(t *testing.T) {
	dir := initArchiveDir(t)
	content := "research_logs:\n  rl-x:\n    objective: x\n"
	writeAddTestFile(t, dir, "research_logs/rl-x.glx", content)
	io, out, _ := TestIOStreams()
	if err := addSearch(io, &addSearchOptions{ArchivePath: dir, Log: "rl-x", Query: "q", DryRun: true}); err != nil {
		t.Fatalf("addSearch: %v", err)
	}
	if !strings.Contains(out.String(), "dry run") || trailingLine(out.String()) != "rl-x" {
		t.Errorf("dry-run output: %q", out.String())
	}
	data, _ := os.ReadFile(filepath.Join(dir, "research_logs", "rl-x.glx"))
	if string(data) != content {
		t.Errorf("dry run modified the file:\n%s", data)
	}
}

func TestAddSearch_RejectsLogDefinedTwice(t *testing.T) {
	files := map[string][]byte{
		"a.glx": []byte("research_logs:\n  rl-x:\n    objective: a\n"),
		"b.glx": []byte("research_logs:\n  rl-x:\n    objective: b\n"),
	}
	if _, err := planAddSearchWrite(files, "rl-x", &glxlib.Search{Query: "q"}); !errors.Is(err, ErrAddSearchLogAmbiguous) {
		t.Errorf("got %v, want ErrAddSearchLogAmbiguous", err)
	}
}

func TestDetectYAMLIndent(t *testing.T) {
	cases := map[string]int{
		"persons:\n  person-a:\n    x: 1\n":       2,
		"persons:\n    person-a:\n        x: 1\n": 4,
		"# comment\n\npersons:\n   a: 1\n":        3,
		"persons: {}\n":                           2,
	}
	for in, want := range cases {
		if got := detectYAMLIndent([]byte(in)); got != want {
			t.Errorf("detectYAMLIndent(%q) = %d, want %d", in, got, want)
		}
	}
}

// =============================================================================
// add study
// =============================================================================

func TestAddStudy_HappyPath(t *testing.T) {
	dir := initArchiveDir(t)
	io, out, _ := TestIOStreams()
	if err := addPlace(io, &addPlaceOptions{ArchivePath: dir, Name: "Rowan NC"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()

	err := addStudy(io, &addStudyOptions{
		ArchivePath: dir,
		Title:       "Parents of Lewis Little",
		Type:        "brick_wall",
		Status:      "active",
		DateRange:   "FROM 1750 TO 1822",
		Places:      []string{"place-rowan-nc"},
	})
	if err != nil {
		t.Fatalf("addStudy: %v", err)
	}
	want := "study-parents-of-lewis-little"
	if got := trailingLine(out.String()); got != want {
		t.Errorf("trailing stdout line: got %q, want %q", got, want)
	}
	study, ok := readBackArchive(t, dir).Studies[want]
	if !ok {
		t.Fatalf("study %s missing", want)
	}
	if study.Type != "brick_wall" || study.Status != "active" || string(study.DateRange) != "FROM 1750 TO 1822" ||
		len(study.Places) != 1 || study.Places[0] != "place-rowan-nc" {
		t.Errorf("study fields wrong: %+v", study)
	}
}

func TestAddStudy_Rejections(t *testing.T) {
	dir := initArchiveDir(t)
	cases := []struct {
		name string
		opts addStudyOptions
		want error
	}{
		{"no title", addStudyOptions{Type: "brick_wall"}, ErrAddStudyTitleRequired},
		{"bad type", addStudyOptions{Title: "x", Type: "bogus"}, ErrAddVocabKeyUnknown},
		{"bad status", addStudyOptions{Title: "x", Status: "bogus"}, ErrAddVocabKeyUnknown},
		{"dangling place", addStudyOptions{Title: "x", Places: []string{"place-nope"}}, ErrAddRefNotFound},
		{"dangling source", addStudyOptions{Title: "x", Sources: []string{"source-nope"}}, ErrAddRefNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			io, _, _ := TestIOStreams()
			tc.opts.ArchivePath = dir
			if err := addStudy(io, &tc.opts); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// =============================================================================
// source / person / event flags
// =============================================================================

func TestAddSource_PropertyFlags(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addSource(io, &addSourceOptions{
		ArchivePath:     dir,
		Title:           "Find a Grave",
		URL:             "https://www.findagrave.com",
		PublicationInfo: "Find a Grave, 1995-",
		CallNumber:      "FG-1",
		SourceNature:    "derivative",
		InformationType: "secondary",
	})
	if err != nil {
		t.Fatalf("addSource: %v", err)
	}
	props := readBackArchive(t, dir).Sources["source-find-a-grave"].Properties
	want := map[string]string{
		"url":              "https://www.findagrave.com",
		"publication_info": "Find a Grave, 1995-",
		"call_number":      "FG-1",
		"source_nature":    "derivative",
		"information_type": "secondary",
	}
	for k, v := range want {
		if props[k] != v {
			t.Errorf("properties[%s]: got %v, want %q", k, props[k], v)
		}
	}

	for _, bad := range []addSourceOptions{{Title: "x", SourceNature: "bogus"}, {Title: "y", InformationType: "bogus"}} {
		bad.ArchivePath = dir
		if err := addSource(io, &bad); !errors.Is(err, ErrAddVocabKeyUnknown) {
			t.Errorf("bad vocab %+v: got %v", bad, err)
		}
	}
}

func TestAddPerson_ExternalIDs(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addPerson(io, &addPersonOptions{
		ArchivePath: dir,
		Given:       "Lewis",
		ExternalIDs: []string{"wikitree:Little-20642", "familysearch:ark:/61903/1:1:X"},
	})
	if err != nil {
		t.Fatalf("addPerson: %v", err)
	}
	ids, ok := readBackArchive(t, dir).Persons["person-lewis"].Properties["external_ids"].([]any)
	if !ok || len(ids) != 2 {
		t.Fatalf("external_ids: got %#v", ids)
	}
	first, _ := ids[0].(map[string]any)
	fields, _ := first["fields"].(map[string]any)
	if first["value"] != "Little-20642" || fields["type"] != "wikitree" {
		t.Errorf("first external id: %#v", first)
	}
	second, _ := ids[1].(map[string]any)
	if second["value"] != "ark:/61903/1:1:X" {
		t.Errorf("value split on the first colon only: %#v", second)
	}

	if err := addPerson(io, &addPersonOptions{ArchivePath: dir, Given: "X", ExternalIDs: []string{"novalue"}}); !errors.Is(err, ErrAddExternalIDFormat) {
		t.Errorf("malformed --external-id: got %v", err)
	}
}

func TestAddEvent_PropertyFlags(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addEvent(io, &addEventOptions{
		ArchivePath: dir,
		Type:        "death",
		Date:        "1826",
		Properties:  []string{"event_subtype=intestate", "description=Died at home, aged 60=ish"},
	})
	if err != nil {
		t.Fatalf("addEvent: %v", err)
	}
	props := readBackArchive(t, dir).Events["event-death-1826"].Properties
	if props["event_subtype"] != "intestate" || props["description"] != "Died at home, aged 60=ish" {
		t.Errorf("event properties: %#v", props)
	}
}

func TestBuildPropertyFlags(t *testing.T) {
	yes := true
	vocab := map[string]*glxlib.PropertyDefinition{
		"cause":   {ValueType: "string"},
		"count":   {ValueType: "integer"},
		"living":  {ValueType: "boolean"},
		"aliases": {ValueType: "string", MultiValue: &yes},
	}
	props, err := buildEventPropertyFlags([]string{"cause=fever", "count=3", "living=true", "aliases=a", "aliases=b"}, vocab)
	if err != nil {
		t.Fatalf("buildPropertyFlags: %v", err)
	}
	if props["cause"] != "fever" || props["count"] != 3 || props["living"] != true {
		t.Errorf("typed values: %#v", props)
	}
	if list, ok := props["aliases"].([]any); !ok || len(list) != 2 {
		t.Errorf("multi-value: %#v", props["aliases"])
	}

	bad := map[string]error{
		"nokey":      ErrAddPropertyFormat,
		"=x":         ErrAddPropertyFormat,
		"cause=":     ErrAddPropertyFormat,
		"bogus=1":    ErrAddPropertyUnknown,
		"count=many": ErrAddPropertyFormat,
		"living=eh":  ErrAddPropertyFormat,
	}
	for flag, want := range bad {
		if _, err := buildEventPropertyFlags([]string{flag}, vocab); !errors.Is(err, want) {
			t.Errorf("%q: got %v, want %v", flag, err, want)
		}
	}
	if _, err := buildEventPropertyFlags([]string{"cause=a", "cause=b"}, vocab); !errors.Is(err, ErrAddPropertyRepeated) {
		t.Errorf("repeated single-value key: got %v", err)
	}
}

// Archives without their own research-log vocabulary files still get the
// standard search_result_types and research_log_status_types on load, the
// same way every other standard vocabulary is defaulted.
func TestMergeStandardVocabularies_ResearchLogVocabularies(t *testing.T) {
	archive := &glxlib.GLXFile{}
	if err := mergeStandardVocabularies(archive); err != nil {
		t.Fatal(err)
	}
	if _, ok := archive.SearchResultTypes["not_found"]; !ok {
		t.Errorf("search_result_types not defaulted: %v", archive.SearchResultTypes)
	}
	if _, ok := archive.ResearchLogStatusTypes["in_progress"]; !ok {
		t.Errorf("research_log_status_types not defaulted: %v", archive.ResearchLogStatusTypes)
	}
}

func TestAddResearch_CobraWrappersDelegateCleanly(t *testing.T) {
	dir := initArchiveDir(t)
	t.Cleanup(func() {
		addResearchLogOpts = addResearchLogOptions{}
		addSearchOpts = addSearchOptions{}
		addStudyOpts = addStudyOptions{}
	})

	addResearchLogOpts = addResearchLogOptions{ArchivePath: dir, OverrideID: "rl-wrap"}
	if err := runAddResearchLog(nil, nil); err != nil {
		t.Fatalf("runAddResearchLog: %v", err)
	}
	addSearchOpts = addSearchOptions{ArchivePath: dir, Log: "rl-wrap", Query: "q"}
	if err := runAddSearch(nil, nil); err != nil {
		t.Fatalf("runAddSearch: %v", err)
	}
	addStudyOpts = addStudyOptions{ArchivePath: dir, Title: "Wrapper Study"}
	if err := runAddStudy(nil, nil); err != nil {
		t.Fatalf("runAddStudy: %v", err)
	}
}
