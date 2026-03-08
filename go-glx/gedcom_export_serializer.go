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
	"bytes"
	"fmt"
	"strings"
)

// gedcomMaxLineValueLength is the maximum number of characters for a GEDCOM line value
// before it must be split using CONC continuation. GEDCOM 5.5.1 spec recommends 248.
const gedcomMaxLineValueLength = 248

// serializeGEDCOMRecords converts a tree of GEDCOMRecord structs to GEDCOM text format.
func serializeGEDCOMRecords(records []*GEDCOMRecord) []byte {
	var buf bytes.Buffer

	for _, record := range records {
		serializeRecord(record, 0, &buf)
	}

	return buf.Bytes()
}

// serializeRecord recursively writes a single record and its subrecords.
func serializeRecord(record *GEDCOMRecord, level int, buf *bytes.Buffer) {
	// Build the line prefix: "LEVEL [XREF] TAG"
	var prefix string
	if record.XRef != "" {
		prefix = fmt.Sprintf("%d %s %s", level, record.XRef, record.Tag)
	} else {
		prefix = fmt.Sprintf("%d %s", level, record.Tag)
	}

	// Handle value with potential CONT/CONC splitting
	if record.Value == "" {
		buf.WriteString(prefix)
		buf.WriteByte('\n')
	} else {
		writeValueWithContinuation(prefix, record.Value, level, buf)
	}

	// Recursively serialize subrecords
	for _, sub := range record.SubRecords {
		serializeRecord(sub, level+1, buf)
	}
}

// writeValueWithContinuation writes a GEDCOM value, splitting long lines with CONC
// and handling newlines with CONT.
func writeValueWithContinuation(prefix, value string, level int, buf *bytes.Buffer) {
	// Split on newlines first (CONT for each new line)
	lines := strings.Split(value, "\n")

	for i, line := range lines {
		if i == 0 {
			// First line: write with the original prefix
			writeLineSplitByCONC(prefix, line, level, buf)
		} else {
			// Subsequent lines: use CONT tag
			contPrefix := fmt.Sprintf("%d %s", level+1, GedcomTagCont)
			writeLineSplitByCONC(contPrefix, line, level, buf)
		}
	}
}

// writeLineSplitByCONC writes a single logical line, splitting with CONC if it exceeds
// the maximum line length.
func writeLineSplitByCONC(prefix, value string, level int, buf *bytes.Buffer) {
	if len(value) <= gedcomMaxLineValueLength {
		// Fits in one line
		if value == "" {
			buf.WriteString(prefix)
		} else {
			buf.WriteString(prefix)
			buf.WriteByte(' ')
			buf.WriteString(value)
		}
		buf.WriteByte('\n')

		return
	}

	// First chunk
	remaining := value
	first := true

	for len(remaining) > 0 {
		chunkSize := gedcomMaxLineValueLength
		if chunkSize > len(remaining) {
			chunkSize = len(remaining)
		}

		chunk := remaining[:chunkSize]
		remaining = remaining[chunkSize:]

		if first {
			buf.WriteString(prefix)
			buf.WriteByte(' ')
			buf.WriteString(chunk)
			buf.WriteByte('\n')
			first = false
		} else {
			fmt.Fprintf(buf, "%d %s %s\n", level+1, GedcomTagConc, chunk)
		}
	}
}
