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

	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

var (
	errEmptyFile     = errors.New("the selected file is empty")
	errGEDZIP        = errors.New("ZIP and GEDZIP files require the GLX CLI")
	errGEDCOM        = errors.New("expected a GEDCOM file beginning with a 0 HEAD record")
	errUTF16         = errors.New("UTF-16 GEDCOM files are unsupported. Save or export the file as UTF-8 and try again")
	errGLX           = errors.New("expected a single-file GLX archive")
	errDestination   = errors.New("unknown conversion format")
	errEmbeddedMedia = errors.New("this GEDCOM contains embedded BLOB media; use the GLX CLI to preserve the embedded files")
)

const (
	singleFileGLX = "glx"
	multiFileGLX  = "zip"
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

// convert writes the converted file to output as it is produced, so the
// caller can hand it on in chunks rather than hold it whole.
func convert(data []byte, destination string, prepare func(), output io.Writer) (summary, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return summary{}, errEmptyFile
	}
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) || bytes.HasPrefix(data, []byte("PK\x05\x06")) {
		return summary{}, errGEDZIP
	}

	// Conversion reports library diagnostics. Archive validation is a separate operation.
	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false})
	switch destination {
	case singleFileGLX, multiFileGLX:
		return importGEDCOM(data, serializer, destination, prepare, output)
	case "5.5.1":
		return exportGEDCOM(data, glxlib.GEDCOM551, output)
	case "7.0":
		return exportGEDCOM(data, glxlib.GEDCOM70, output)
	default:
		return summary{}, errDestination
	}
}

func importGEDCOM(data []byte, serializer *glxlib.DefaultSerializer, destination string, prepare func(), output io.Writer) (summary, error) {
	// The importer tolerates legacy headers and encodings. Check the first
	// record here so unrelated text cannot produce an empty archive.
	header := bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	if bytes.HasPrefix(header, []byte{0, '0'}) || bytes.HasPrefix(header, []byte{'0', 0}) ||
		bytes.HasPrefix(header, []byte{0xff, 0xfe}) || bytes.HasPrefix(header, []byte{0xfe, 0xff}) {
		return summary{}, errUTF16
	}
	if end := bytes.IndexAny(header, "\r\n"); end >= 0 {
		header = header[:end]
	}
	fields := bytes.Fields(header)
	if len(fields) < 2 || !bytes.Equal(fields[0], []byte("0")) || !bytes.Equal(fields[1], []byte("HEAD")) {
		return summary{}, errGEDCOM
	}
	// Refuse embedded media before spending the whole import on it.
	if hasBLOBTag(data) {
		return summary{}, errEmbeddedMedia
	}

	archive, result, err := glxlib.ImportGEDCOM(bytes.NewReader(data), nil)
	if err != nil {
		return summary{}, fmt.Errorf("could not read GEDCOM: %w", err)
	}
	// The line scan splits on LF; this catches a BLOB in a CR-only file.
	for i := range result.MediaFiles {
		if result.MediaFiles[i].SourceType == glxlib.MediaSourceBlob {
			return summary{}, errEmbeddedMedia
		}
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
	if destination == multiFileGLX {
		if err := serializer.WriteMultiFileZIP(archive, output); err != nil {
			return summary{}, fmt.Errorf("could not write the GLX ZIP: %w", err)
		}

		return report, nil
	}
	if err := serializer.WriteSingleFileBatched(archive, output); err != nil {
		return summary{}, fmt.Errorf("could not write GLX: %w", err)
	}

	return report, nil
}

// hasBLOBTag reports whether any LF-terminated GEDCOM line is a BLOB record
// ("<level> BLOB"), the GEDCOM 5.5 form of embedded multimedia.
func hasBLOBTag(data []byte) bool {
	for line := range bytes.Lines(data) {
		line = bytes.TrimLeft(line, " \t")
		level := 0
		for level < len(line) && line[level] >= '0' && line[level] <= '9' {
			level++
		}
		rest := line[level:]
		tag := bytes.TrimLeft(rest, " \t")
		if level == 0 || len(tag) == len(rest) {
			continue // not "<level><delimiter>..."
		}
		if tail, found := bytes.CutPrefix(tag, []byte("BLOB")); found &&
			(len(tail) == 0 || bytes.ContainsAny(tail[:1], " \t\r\n")) {
			return true
		}
	}

	return false
}

func importMessage(line int, tag, message string) string {
	if line == 0 {
		return fmt.Sprintf("%s: %s", tag, message)
	}

	return fmt.Sprintf("Line %d (%s): %s", line, tag, message)
}

func exportGEDCOM(data []byte, version glxlib.GEDCOMVersion, output io.Writer) (summary, error) {
	// Reject unknown structural fields before decoding can silently discard them.
	// Property maps stay extensible; this is not full archive validation.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var archive *glxlib.GLXFile
	if err := decoder.Decode(&archive); err != nil {
		return summary{}, fmt.Errorf("could not read GLX: %w", err)
	}
	if archive == nil {
		return summary{}, errGLX
	}
	// A single-file archive is one YAML document, not a stream. Never ignore
	// later documents (including malformed ones) and export only the first.
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return summary{}, errGLX
	}

	if len(archive.Sources) > 0 || len(archive.Media) > 0 || len(archive.Places) > 0 {
		if err := checkCompatibilityFields(data); err != nil {
			return summary{}, fmt.Errorf("could not read GLX: %w", err)
		}
	}

	gedcom, result, err := glxlib.ExportGEDCOM(archive, version, nil)
	if err != nil {
		return summary{}, fmt.Errorf("could not write GEDCOM: %w", err)
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
	if _, err := output.Write(gedcom); err != nil {
		return summary{}, fmt.Errorf("could not write GEDCOM: %w", err)
	}

	return report, nil
}

// Source and Media have legacy-description unmarshallers, and Place a
// temporal-parent unmarshaller, that use Node.Decode, which does not inherit
// KnownFields. Check these records without those methods before allowing
// their compatibility conversion to hide misspelled fields.
// TestCompatibilityCheckCoversEveryStructUnmarshaller fails if another struct
// in the archive gains such an unmarshaller without being added here.
func checkCompatibilityFields(data []byte) error {
	type sourceFields glxlib.Source
	type mediaFields glxlib.Media
	var archive struct {
		Sources map[string]*struct {
			sourceFields `yaml:",inline"`
			Description  string `yaml:"description,omitempty"`
		} `yaml:"sources"`
		Media map[string]*struct {
			mediaFields `yaml:",inline"`
			Description string `yaml:"description,omitempty"`
		} `yaml:"media"`
		Places map[string]*placeFields `yaml:"places"`
		// Other sections were already checked by the GLXFile decoder above.
		// A yaml.Node takes them as a shallow copy of the parsed tree rather
		// than rebuilding every person and event as generic maps.
		Other map[string]yaml.Node `yaml:",inline"`
	}
	// Keep the full document so aliases can refer to anchors in other sections.
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	return decoder.Decode(&archive)
}

// placeFields mirrors glxlib.Place for the strict field check. Place cannot be
// embedded inline because its `parent` key is a string there but may be a
// dated list in the file. TestPlaceFieldsMirrorPlace keeps the two in step.
type placeFields struct {
	Name       string            `yaml:"name"`
	Parent     placeParentFields `yaml:"parent,omitempty"`
	Type       string            `yaml:"type,omitempty"`
	Latitude   *float64          `yaml:"latitude,omitempty"`
	Longitude  *float64          `yaml:"longitude,omitempty"`
	Properties map[string]any    `yaml:"properties,omitempty"`
	Notes      glxlib.NoteList   `yaml:"notes,omitempty"`
}

// placeParentFields accepts a place's parent as a place ID or a list of
// {value, date} entries, rejecting unknown keys in the entries.
type placeParentFields struct{}

type placeParentPeriodFields glxlib.PlaceParentPeriod

func (*placeParentFields) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	if node.Kind != yaml.SequenceNode {
		var id string

		return node.Decode(&id)
	}
	for _, entry := range node.Content {
		if entry.Kind == yaml.AliasNode && entry.Alias != nil {
			entry = entry.Alias
		}
		if entry.Kind == yaml.MappingNode {
			for i := 0; i < len(entry.Content); i += 2 {
				if key := entry.Content[i].Value; key != "value" && key != "date" {
					return fmt.Errorf("line %d: field %s not found in place parent entry: %w", entry.Content[i].Line, key, errGLX)
				}
			}
		}
		var period placeParentPeriodFields
		if err := entry.Decode(&period); err != nil {
			return err
		}
	}

	return nil
}
