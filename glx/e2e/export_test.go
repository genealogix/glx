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

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Evidence recorded as an event-subject assertion used to vanish on export:
// the exporter only indexed assertions whose subject was a person, so a birth
// or marriage date cited through `subject.event` produced a bare BIRT or MARR
// and the evidence chain was lost on the way out of GLX (#1269).
func TestExport_EventSubjectAssertionsCarrySOUR(t *testing.T) {
	for _, version := range []string{"70", "551"} {
		t.Run(version, func(t *testing.T) {
			archive := copyExample(t, "complete-family")
			out := filepath.Join(archive, "out.ged")

			res := runGLX(t, archive, "export", ".", "-f", version, "-o", out)
			require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

			data, err := os.ReadFile(out)
			require.NoError(t, err)
			ged := string(data)

			// John's birth date cites the parish register and the census; his
			// birth place cites the parish register again, so the parish page
			// is expected once, not twice.
			birth := eventBlock(t, ged, "@I2@", "BIRT")
			assert.Contains(t, birth, "2 SOUR")
			assert.Contains(t, birth, "3 PAGE Entry 145, Page 23, January 1850")
			assert.Contains(t, birth, "3 PAGE District: Leeds, Piece: 2319, Folio: 234, Page: 23")
			assert.Equal(t, 2, strings.Count(birth, "2 SOUR"),
				"one SOUR per cited page, deduplicated across the date and place assertions")

			marriage := eventBlock(t, ged, "@F1@", "MARR")
			assert.Contains(t, marriage, "3 PAGE Entry 89, Page 67, May 1875")
			assert.Equal(t, 1, strings.Count(marriage, "2 SOUR"))
		})
	}
}

// eventBlock returns the level-1 event structure with the given tag inside the
// level-0 record with the given XREF, as the lines below `1 <tag>` up to the
// next level-0 or level-1 line.
func eventBlock(t *testing.T, ged, xref, tag string) string {
	t.Helper()

	var (
		block   []string
		inRec   bool
		inEvent bool
	)
	for line := range strings.SplitSeq(ged, "\n") {
		switch {
		case strings.HasPrefix(line, "0 "):
			if inEvent {
				return strings.Join(block, "\n")
			}
			inRec = strings.HasPrefix(line, "0 "+xref+" ")
		case inRec && strings.HasPrefix(line, "1 "):
			if inEvent {
				return strings.Join(block, "\n")
			}
			inEvent = line == "1 "+tag
		case inEvent:
			block = append(block, line)
		}
	}
	require.True(t, inEvent, "no %s under %s in the exported file", tag, xref)

	return strings.Join(block, "\n")
}
