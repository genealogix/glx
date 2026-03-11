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
	"os"
	"runtime"
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

// generateGEDCOMWithFamilies creates a more realistic synthetic GEDCOM with
// individuals, families, sources, and cross-references — exercising the full
// import pipeline rather than just line parsing.
func generateGEDCOMWithFamilies(approxBytes int, charset string) []byte {
	var buf bytes.Buffer
	buf.WriteString("0 HEAD\r\n")
	buf.WriteString("1 GEDC\r\n")
	buf.WriteString("2 VERS 5.5.1\r\n")
	buf.WriteString(fmt.Sprintf("1 CHAR %s\r\n", charset))
	buf.WriteString("1 SOUR GLX_BENCH\r\n")

	personID := 1
	famID := 1
	sourceID := 1

	for buf.Len() < approxBytes {
		// Source record
		buf.WriteString(fmt.Sprintf("0 @S%d@ SOUR\r\n", sourceID))
		buf.WriteString(fmt.Sprintf("1 TITL Source Record %d\r\n", sourceID))
		buf.WriteString("1 AUTH Unknown Author\r\n")

		// Husband
		husbID := personID
		buf.WriteString(fmt.Sprintf("0 @I%d@ INDI\r\n", personID))
		buf.WriteString(fmt.Sprintf("1 NAME Husband%d /Family%d/\r\n", personID, famID))
		buf.WriteString("1 SEX M\r\n")
		buf.WriteString("1 BIRT\r\n")
		buf.WriteString("2 DATE 15 MAR 1850\r\n")
		buf.WriteString("2 PLAC Springfield, Illinois\r\n")
		buf.WriteString(fmt.Sprintf("2 SOUR @S%d@\r\n", sourceID))
		buf.WriteString(fmt.Sprintf("1 FAMS @F%d@\r\n", famID))
		buf.WriteString("1 DEAT\r\n")
		buf.WriteString("2 DATE 2 AUG 1920\r\n")
		buf.WriteString("2 PLAC Chicago, Illinois\r\n")
		personID++

		// Wife
		wifeID := personID
		buf.WriteString(fmt.Sprintf("0 @I%d@ INDI\r\n", personID))
		buf.WriteString(fmt.Sprintf("1 NAME Wife%d /Family%d/\r\n", personID, famID))
		buf.WriteString("1 SEX F\r\n")
		buf.WriteString("1 BIRT\r\n")
		buf.WriteString("2 DATE 22 JUN 1855\r\n")
		buf.WriteString("2 PLAC Boston, Massachusetts\r\n")
		buf.WriteString(fmt.Sprintf("1 FAMS @F%d@\r\n", famID))
		personID++

		// Child
		childID := personID
		buf.WriteString(fmt.Sprintf("0 @I%d@ INDI\r\n", personID))
		buf.WriteString(fmt.Sprintf("1 NAME Child%d /Family%d/\r\n", personID, famID))
		buf.WriteString("1 SEX M\r\n")
		buf.WriteString("1 BIRT\r\n")
		buf.WriteString("2 DATE 10 JAN 1880\r\n")
		buf.WriteString("2 PLAC New York, New York\r\n")
		buf.WriteString(fmt.Sprintf("1 FAMC @F%d@\r\n", famID))
		personID++

		// Family record
		buf.WriteString(fmt.Sprintf("0 @F%d@ FAM\r\n", famID))
		buf.WriteString(fmt.Sprintf("1 HUSB @I%d@\r\n", husbID))
		buf.WriteString(fmt.Sprintf("1 WIFE @I%d@\r\n", wifeID))
		buf.WriteString(fmt.Sprintf("1 CHIL @I%d@\r\n", childID))
		buf.WriteString("1 MARR\r\n")
		buf.WriteString("2 DATE 5 SEP 1875\r\n")
		buf.WriteString("2 PLAC Philadelphia, Pennsylvania\r\n")
		famID++
		sourceID++
	}

	buf.WriteString("0 TRLR\r\n")
	return buf.Bytes()
}

func BenchmarkImportGEDCOM_Synthetic(b *testing.B) {
	for _, size := range []int{100_000, 1_000_000, 10_000_000} {
		data := generateGEDCOMWithFamilies(size, "UTF-8")
		name := fmt.Sprintf("%dMB", len(data)/1_000_000)
		if len(data) < 1_000_000 {
			name = fmt.Sprintf("%dKB", len(data)/1_000)
		}
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				_, _, err := ImportGEDCOM(bytes.NewReader(data), io.Discard)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkImportGEDCOM_RealFiles(b *testing.B) {
	files := []struct {
		name string
		path string
	}{
		{"habsburg", "../glx/testdata/gedcom/5.5.1/large-files/habsburg.ged"},
		{"queen", "../glx/testdata/gedcom/5.5.1/large-files/queen.ged"},
		{"shakespeare", "../glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged"},
	}

	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			b.Logf("skipping %s: %v", f.name, err)
			continue
		}
		b.Run(f.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				_, _, err := ImportGEDCOM(bytes.NewReader(data), io.Discard)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// measureImportMemory runs fn once and reports total bytes allocated using
// TotalAlloc, which counts all allocations regardless of GC. This gives an
// accurate picture of memory pressure during import.
func measureImportMemory(fn func()) uint64 {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	fn()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	return after.TotalAlloc - before.TotalAlloc
}

func BenchmarkImportGEDCOM_Memory(b *testing.B) {
	sizes := []struct {
		name  string
		bytes int
	}{
		{"1MB", 1_000_000},
		{"10MB", 10_000_000},
		{"50MB", 50_000_000},
	}

	for _, s := range sizes {
		data := generateGEDCOMWithFamilies(s.bytes, "UTF-8")
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				peak := measureImportMemory(func() {
					_, _, _ = ImportGEDCOM(bytes.NewReader(data), io.Discard)
				})
				b.ReportMetric(float64(peak)/(1024*1024), "alloc-MB")
				b.ReportMetric(float64(peak)/float64(len(data)), "mem/input")
			}
		})
	}
}

func BenchmarkImportGEDCOM_Memory_RealFiles(b *testing.B) {
	files := []struct {
		name string
		path string
	}{
		{"habsburg", "../glx/testdata/gedcom/5.5.1/large-files/habsburg.ged"},
		{"queen", "../glx/testdata/gedcom/5.5.1/large-files/queen.ged"},
	}

	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			b.Logf("skipping %s: %v", f.name, err)
			continue
		}
		b.Run(f.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				peak := measureImportMemory(func() {
					_, _, _ = ImportGEDCOM(bytes.NewReader(data), io.Discard)
				})
				b.ReportMetric(float64(peak)/(1024*1024), "alloc-MB")
				b.ReportMetric(float64(peak)/float64(len(data)), "mem/input")
			}
		})
	}
}
