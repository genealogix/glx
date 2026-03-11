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
	"io"
	"strings"
	"testing"
)

// generateGEDCOM creates a synthetic GEDCOM file of approximately the given
// size with the specified CHAR encoding. Useful for benchmarking parser
// performance at various scales.
func generateGEDCOM(approxBytes int, charset string) []byte {
	var buf bytes.Buffer
	buf.WriteString("0 HEAD\r\n")
	buf.WriteString("1 GEDC\r\n")
	buf.WriteString("2 VERS 5.5.1\r\n")
	buf.WriteString(fmt.Sprintf("1 CHAR %s\r\n", charset))

	personID := 1
	for buf.Len() < approxBytes {
		buf.WriteString(fmt.Sprintf("0 @I%d@ INDI\r\n", personID))
		buf.WriteString(fmt.Sprintf("1 NAME Person%d /Family%d/\r\n", personID, personID))
		buf.WriteString("1 SEX M\r\n")
		buf.WriteString("1 BIRT\r\n")
		buf.WriteString("2 DATE 15 MAR 1850\r\n")
		buf.WriteString("2 PLAC Springfield, Illinois\r\n")
		buf.WriteString("1 DEAT\r\n")
		buf.WriteString("2 DATE 2 AUG 1920\r\n")
		buf.WriteString("2 PLAC Chicago, Illinois\r\n")
		buf.WriteString(fmt.Sprintf("1 NOTE This is a note for person %d with some extra text to add bulk.\r\n", personID))
		personID++
	}

	buf.WriteString("0 TRLR\r\n")
	return buf.Bytes()
}

func BenchmarkParseGEDCOMLines_UTF8(b *testing.B) {
	for _, size := range []int{100_000, 1_000_000, 10_000_000} {
		data := generateGEDCOM(size, "UTF-8")
		name := fmt.Sprintf("%s/%dMB", "UTF8", len(data)/1_000_000)
		if len(data) < 1_000_000 {
			name = fmt.Sprintf("%s/%dKB", "UTF8", len(data)/1_000)
		}
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				_, err := parseGEDCOMLines(bytes.NewReader(data))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseGEDCOMLines_CP1252(b *testing.B) {
	for _, size := range []int{100_000, 1_000_000, 10_000_000} {
		data := generateGEDCOM(size, "ANSI")
		name := fmt.Sprintf("%s/%dMB", "CP1252", len(data)/1_000_000)
		if len(data) < 1_000_000 {
			name = fmt.Sprintf("%s/%dKB", "CP1252", len(data)/1_000)
		}
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				_, err := parseGEDCOMLines(bytes.NewReader(data))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDecodingReader_UTF8_Streaming(b *testing.B) {
	// Verify UTF-8 path streams without extra allocation
	data := generateGEDCOM(10_000_000, "UTF-8")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		r, err := decodingReader(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, r)
	}
}

func BenchmarkDecodingReader_CP1252_Streaming(b *testing.B) {
	// Verify CP1252 streams through transform.NewReader without full buffering
	data := generateGEDCOM(10_000_000, "ANSI")
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		r, err := decodingReader(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, r)
	}
}

func BenchmarkConvertANSELToUTF8(b *testing.B) {
	// ANSEL with combining marks interspersed
	var sb strings.Builder
	for i := 0; i < 100_000; i++ {
		sb.WriteString("0 HEAD\n1 NAME \xe2A\xe1E\xe8O test\n")
	}
	data := []byte(sb.String())
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		convertANSELToUTF8(data)
	}
}
