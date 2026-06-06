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
	"fmt"
	"strings"
	"testing"
)

func newTestStreams() (*IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}

	return &IOStreams{Out: out, MachineOut: out, ErrOut: errOut}, out, errOut
}

// errGenericTest is a static non-sentinel error used to exercise the default
// (exit 1) branch of exitCodeForError (err113: no inline dynamic errors).
var errGenericTest = errors.New("some unrecognized error")

func TestExitCodeForError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"unknown entity type", errStdinUnknownEntityType, exitBadInvocation},
		{"path args", errStdinPathArgs, exitBadInvocation},
		{"empty stdin", errStdinEmpty, exitBadInvocation},
		{"stdin+report", errStdinReportExclusive, exitBadInvocation},
		{"wrapped sentinel", fmt.Errorf("context: %w", errStdinUnknownEntityType), exitBadInvocation},
		{"structural failure is exit 1", ErrStructuralValidationFailed, 1},
		{"unrecognized error is exit 1", errGenericTest, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := exitCodeForError(tc.err); got != tc.want {
				t.Errorf("exitCodeForError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestRunValidateStdinReportExclusive(t *testing.T) {
	defer func(stdin, report bool) {
		validateStdin, validateReport = stdin, report
	}(validateStdin, validateReport)

	validateStdin, validateReport = true, true
	if err := runValidate(nil, nil); !errors.Is(err, errStdinReportExclusive) {
		t.Errorf("runValidate with --stdin and --report: got %v, want errStdinReportExclusive", err)
	}
}

func TestValidateStdinEntity(t *testing.T) {
	// valid entity → success message, no error.
	s, out, _ := newTestStreams()
	if err := validateStdinEntity(s, "person", nil, strings.NewReader("properties: {}")); err != nil {
		t.Fatalf("valid person: unexpected error %v", err)
	}
	if !strings.Contains(out.String(), "structurally valid") {
		t.Errorf("expected success message, got %q", out.String())
	}

	// invalid entity → ErrStructuralValidationFailed + error output.
	s, _, errOut := newTestStreams()
	if err := validateStdinEntity(s, "person", nil, strings.NewReader("bogus_field: 1")); err == nil {
		t.Error("expected error for invalid person")
	}
	if !strings.Contains(errOut.String(), "structural error") {
		t.Errorf("expected error output, got %q", errOut.String())
	}

	// positional args are rejected with --stdin.
	s, _, _ = newTestStreams()
	if err := validateStdinEntity(s, "person", []string{"file.glx"}, strings.NewReader("")); err == nil {
		t.Error("expected error when path args are passed with --stdin")
	}

	// unknown entity-type is rejected.
	s, _, _ = newTestStreams()
	if err := validateStdinEntity(s, "nope", nil, strings.NewReader("x: 1")); err == nil {
		t.Error("expected error for unknown entity-type")
	}
}

func TestCollectionForEntityType(t *testing.T) {
	cases := []struct {
		flag string
		want string
		ok   bool
	}{
		{"person", "persons", true},
		{"event", "events", true},
		{"media", "media", true},
		{"relationship", "relationships", true},
		{"research-log", "research_logs", true},
		{"study", "studies", true},
		// vocabulary-entry must map to the SINGULAR-derived "event_types",
		// not the plural "events_types" — regression guard.
		{entityTypeVocabularyEntry, "event_types", true},
		// case/space tolerant
		{"  Person ", "persons", true},
		// rejects
		{"nonsense", "", false},
		{"", "", false},
		{"persons", "", false}, // plural form is not a valid --entity-type
	}
	for _, c := range cases {
		got, ok := collectionForEntityType(c.flag)
		if ok != c.ok || got != c.want {
			t.Errorf("collectionForEntityType(%q) = (%q, %v), want (%q, %v)",
				c.flag, got, ok, c.want, c.ok)
		}
	}
}

func TestValidateEntitySnippet(t *testing.T) {
	cases := []struct {
		name       string
		entityType string
		data       string
		wantErr    bool // unusable input (unknown type / empty / malformed)
		wantIssues bool // structural validation issues on parseable input
	}{
		{"valid person", "person", "properties: {}", false, false},
		{"invalid person field", "person", "bogus_field: 1", false, true},
		{"valid vocab entry", entityTypeVocabularyEntry, "label: Birth", false, false},
		{"vocab missing required label", entityTypeVocabularyEntry, "description: no label", false, true},
		{"unknown entity-type", "nonsense", "x: 1", true, false},
		{"blank entity-type", "", "x: 1", true, false},
		{"empty stdin", "person", "   \n  ", true, false},
		{"malformed yaml", "person", "foo: [bar", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			issues, err := validateEntitySnippet(c.entityType, []byte(c.data))
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if err != nil {
				return
			}
			if (len(issues) > 0) != c.wantIssues {
				t.Errorf("issues = %v, wantIssues = %v", issues, c.wantIssues)
			}
		})
	}
}
