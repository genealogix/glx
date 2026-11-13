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
	"io"
	"reflect"

	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

var (
	errEmptyFile   = errors.New("the selected file is empty")
	errGEDZIP      = errors.New("ZIP and GEDZIP files require the GLX CLI")
	errGEDCOM      = errors.New("expected a GEDCOM file beginning with a 0 HEAD record")
	errUTF16       = errors.New("UTF-16 GEDCOM files are unsupported. Save or export the file as UTF-8 and try again")
	errGLX         = errors.New("expected a single-file GLX archive")
	errDestination = errors.New("unknown conversion format")
)

type count struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type diagnostic struct {
	Message string `json:"message"`
	IsError bool   `json:"isError,omitempty"`
}

type summary struct {
	Version     string       `json:"version"`
	Counts      []count      `json:"counts"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

func convert(data []byte, destination string, prepare func()) ([]byte, summary, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, summary{}, errEmptyFile
	}
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) || bytes.HasPrefix(data, []byte("PK\x05\x06")) {
		return nil, summary{}, errGEDZIP
	}

	// Conversion reports library diagnostics. Archive validation is a separate operation.
	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false})
	switch destination {
	case "glx":
		return importGEDCOM(data, serializer, prepare)
	case "5.5.1":
		return exportGEDCOM(data, serializer, glxlib.GEDCOM551)
	case "7.0":
		return exportGEDCOM(data, serializer, glxlib.GEDCOM70)
	default:
		return nil, summary{}, errDestination
	}
}

func importGEDCOM(data []byte, serializer glxlib.Serializer, prepare func()) ([]byte, summary, error) {
	// The importer tolerates legacy headers and encodings. Check the first
	// record here so unrelated text cannot produce an empty archive.
	header := bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	if bytes.HasPrefix(header, []byte{0, '0'}) || bytes.HasPrefix(header, []byte{'0', 0}) ||
		bytes.HasPrefix(header, []byte{0xff, 0xfe}) || bytes.HasPrefix(header, []byte{0xfe, 0xff}) {
		return nil, summary{}, errUTF16
	}
	if end := bytes.IndexAny(header, "\r\n"); end >= 0 {
		header = header[:end]
	}
	fields := bytes.Fields(header)
	if len(fields) < 2 || !bytes.Equal(fields[0], []byte("0")) || !bytes.Equal(fields[1], []byte("HEAD")) {
		return nil, summary{}, errGEDCOM
	}

	archive, result, err := glxlib.ImportGEDCOM(bytes.NewReader(data), nil)
	if err != nil {
		return nil, summary{}, fmt.Errorf("could not read GEDCOM: %w", err)
	}
	stats := result.Statistics
	diagnostics := make([]diagnostic, 0, len(stats.Errors)+len(stats.Warnings))
	for _, entry := range stats.Errors {
		diagnostics = append(diagnostics, diagnostic{Message: importMessage(entry.Line, entry.Tag, entry.Message), IsError: true})
	}
	for _, entry := range stats.Warnings {
		diagnostics = append(diagnostics, diagnostic{Message: importMessage(entry.Line, entry.Tag, entry.Message)})
	}
	report := summary{
		Version: result.Version,
		Counts: []count{
			{"People", len(archive.Persons)},
			{"Events", len(archive.Events)},
			{"Relationships", len(archive.Relationships)},
			{"Sources", len(archive.Sources)},
			{"Citations", len(archive.Citations)},
			{"Media records", len(archive.Media)},
		},
		Diagnostics: diagnostics,
	}
	prepare()
	output, err := serializer.SerializeSingleFileBytes(archive)
	if err != nil {
		return nil, summary{}, fmt.Errorf("could not write GLX: %w", err)
	}

	return output, report, nil
}

func importMessage(line int, tag, message string) string {
	if line == 0 {
		return fmt.Sprintf("%s: %s", tag, message)
	}

	return fmt.Sprintf("Line %d (%s): %s", line, tag, message)
}

func exportGEDCOM(data []byte, serializer glxlib.Serializer, version glxlib.GEDCOMVersion) ([]byte, summary, error) {
	archive, err := serializer.DeserializeSingleFileBytes(data)
	if err != nil {
		return nil, summary{}, fmt.Errorf("could not read GLX: %w", err)
	}
	// The serializer ignores unknown fields. Distinguish unrelated YAML from a
	// supported empty mapping without decoding ordinary archives a second time.
	if reflect.ValueOf(archive).Elem().IsZero() && !isEmptyGLX(data) {
		return nil, summary{}, errGLX
	}

	output, result, err := glxlib.ExportGEDCOM(archive, version, nil)
	if err != nil {
		return nil, summary{}, fmt.Errorf("could not write GEDCOM: %w", err)
	}
	stats := result.Statistics
	diagnostics := make([]diagnostic, 0, len(stats.Warnings))
	for _, entry := range stats.Warnings {
		message := fmt.Sprintf("%s %s: %s", entry.EntityType, entry.EntityID, entry.Message)
		diagnostics = append(diagnostics, diagnostic{Message: message})
	}
	report := summary{
		Version: result.Version,
		Counts: []count{
			{"People", stats.PersonsExported},
			{"Families", stats.FamiliesExported},
			{"Sources", stats.SourcesExported},
			{"Media records", stats.MediaExported},
		},
		Diagnostics: diagnostics,
	}

	return output, report, nil
}

func isEmptyGLX(data []byte) bool {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var archive *glxlib.GLXFile
	err := decoder.Decode(&archive)

	return errors.Is(err, io.EOF) || (err == nil && archive != nil)
}
