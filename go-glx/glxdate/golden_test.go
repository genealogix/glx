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

package glxdate

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateGolden regenerates the golden file from the current parser output:
//
//	go test ./go-glx/glxdate/ -run TestCorpus_Golden -update
//
// Review the resulting diff: every changed row is a behavior change.
var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// parseGoldenPath records, for every corpus input, exactly what Parse
// produces. It lives beside the corpus so the expected outcome of each
// line is reviewable, not just the invariants in corpus_test.go.
const parseGoldenPath = "testdata/gedcom_dates.parse.tsv"

// parseGoldenHeader names the tab-separated columns of the golden file.
const parseGoldenHeader = "input\tcanonical\tyear\tprecision\tqualifier\tcalendar\trange\tvalid\treason"

// parseGoldenRow renders one corpus input as a golden row.
func parseGoldenRow(input string) string {
	d, err := Parse(input)
	reason := ""
	var perr *ParseError
	if errors.As(err, &perr) {
		reason = perr.Reason
	}

	rng := "-"
	switch {
	case d.IsOpenEnded():
		rng = "from"
	case d.IsRange():
		rng = "between"
	}

	return strings.Join([]string{
		input,
		d.String(),
		strconv.Itoa(d.Year()),
		d.Precision().String(),
		d.Qualifier().Keyword(),
		d.Calendar().String(),
		rng,
		strconv.FormatBool(d.Valid()),
		reason,
	}, "\t")
}

// renderParseGolden renders the whole golden file.
func renderParseGolden(t *testing.T, lines []string) []byte {
	t.Helper()

	var b bytes.Buffer
	b.WriteString(parseGoldenHeader + "\n")
	for _, line := range lines {
		require.NotContains(t, line, "\t", "corpus inputs must not contain tabs")
		b.WriteString(parseGoldenRow(line) + "\n")
	}

	return b.Bytes()
}

// TestCorpus_Golden pins the exact Parse result of every corpus line.
func TestCorpus_Golden(t *testing.T) {
	lines := corpusLines(t)
	want := renderParseGolden(t, lines)

	if *updateGolden {
		require.NoError(t, os.WriteFile(parseGoldenPath, want, 0o644))
		t.Logf("rewrote %s", parseGoldenPath)

		return
	}

	golden, err := os.ReadFile(parseGoldenPath)
	require.NoError(t, err, "run with -update to generate the golden file")

	// Compare row by row so a failure names the input that changed.
	wantRows := bufio.NewScanner(bytes.NewReader(want))
	goldenRows := bufio.NewScanner(bytes.NewReader(golden))
	for wantRows.Scan() {
		require.True(t, goldenRows.Scan(), "golden file is shorter than the corpus; run with -update")
		if wantRows.Text() != goldenRows.Text() {
			input, _, _ := strings.Cut(wantRows.Text(), "\t")
			assert.Equal(t, goldenRows.Text(), wantRows.Text(), "Parse(%q) changed; run with -update if intended", input)
		}
	}
	assert.False(t, goldenRows.Scan(), "golden file is longer than the corpus; run with -update")
}
