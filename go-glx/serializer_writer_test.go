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

package glx

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// Every section must propagate a failed destination write, rather than report
// success for a truncated archive or continue emitting files after the failure.
func TestSerializeMultiFileStopsOnWriterError(t *testing.T) {
	archive := &GLXFile{
		ImportMetadata: &Metadata{SourceFile: "input.ged"},
		Persons:        map[string]*Person{"p1": {}},
		Events:         map[string]*Event{"e1": {}},
		Relationships:  map[string]*Relationship{"r1": {}},
		Places:         map[string]*Place{"l1": {}},
		Sources:        map[string]*Source{"s1": {}},
		Citations:      map[string]*Citation{"c1": {}},
		Repositories:   map[string]*Repository{"repo1": {}},
		Media:          map[string]*Media{"m1": {}},
		Assertions:     map[string]*Assertion{"a1": {}},
		ResearchLogs:   map[string]*ResearchLog{"log1": {}},
		Studies:        map[string]*Study{"study1": {}},
	}
	serializer := NewSerializer(&SerializerOptions{Validate: false})
	for _, prefix := range []string{"vocabularies/", "metadata.glx", "persons/", "events/", "relationships/", "places/", "sources/", "citations/", "repositories/", "media/", "assertions/", "research_logs/", "studies/"} {
		t.Run(prefix, func(t *testing.T) {
			want := io.ErrClosedPipe
			failed := false
			err := serializer.serializeMultiFile(archive, func(filename string, _ []byte) error {
				if failed {
					t.Fatalf("continued writing %s after destination failed", filename)
				}
				if strings.HasPrefix(filename, prefix) {
					failed = true

					return want
				}

				return nil
			})
			if !failed || !errors.Is(err, want) {
				t.Fatalf("failed=%v, got %v, want destination error", failed, err)
			}
		})
	}
}
