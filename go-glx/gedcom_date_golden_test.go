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
	"bufio"
	"bytes"
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateDateGolden regenerates the import golden file:
//
//	go test ./go-glx/ -run TestParseGEDCOMDate_Golden -update-date-golden
//
// Review the resulting diff: every changed row is an import behavior change.
var updateDateGolden = flag.Bool("update-date-golden", false, "rewrite the GEDCOM date import golden file")

// importGoldenPath records, for every corpus GEDCOM DATE value, the GLX
// DateString the importer stores, the GEDCOM value the exporter emits for
// it, and the year every consumer will read. It sits beside the corpus and
// the parse golden in glxdate/testdata.
const importGoldenPath = "glxdate/testdata/gedcom_dates.import.tsv"

// importGoldenHeader names the tab-separated columns of the golden file.
const importGoldenHeader = "input\tstored\texported\tyear\tvalid"

// importGoldenRow renders one corpus input as a golden row.
func importGoldenRow(input string) string {
	stored := parseGEDCOMDate(input)
	_, err := stored.Parse()

	return strings.Join([]string{
		input,
		string(stored),
		formatGEDCOMDate(stored),
		strconv.Itoa(ExtractFirstYear(string(stored))),
		strconv.FormatBool(err == nil),
	}, "\t")
}

// TestParseGEDCOMDate_Golden pins the exact import, export, and year of every
// corpus line.
func TestParseGEDCOMDate_Golden(t *testing.T) {
	var want bytes.Buffer
	want.WriteString(importGoldenHeader + "\n")
	for _, line := range loadGEDCOMDateCorpus(t) {
		require.NotContains(t, line, "\t", "corpus inputs must not contain tabs")
		want.WriteString(importGoldenRow(line) + "\n")
	}

	if *updateDateGolden {
		require.NoError(t, os.WriteFile(importGoldenPath, want.Bytes(), 0o644))
		t.Logf("rewrote %s", importGoldenPath)

		return
	}

	golden, err := os.ReadFile(importGoldenPath)
	require.NoError(t, err, "run with -update-date-golden to generate the golden file")

	wantRows := bufio.NewScanner(bytes.NewReader(want.Bytes()))
	goldenRows := bufio.NewScanner(bytes.NewReader(golden))
	for wantRows.Scan() {
		require.True(t, goldenRows.Scan(), "golden file is shorter than the corpus; run with -update-date-golden")
		if wantRows.Text() != goldenRows.Text() {
			input, _, _ := strings.Cut(wantRows.Text(), "\t")
			assert.Equal(t, goldenRows.Text(), wantRows.Text(), "import of %q changed; run with -update-date-golden if intended", input)
		}
	}
	assert.False(t, goldenRows.Scan(), "golden file is longer than the corpus; run with -update-date-golden")
}
