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

package lib

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GEDCOMVersion represents the GEDCOM version
type GEDCOMVersion int

// GEDCOM version constants
const (
	// GEDCOMUnknown represents an unknown or unsupported GEDCOM version.
	GEDCOMUnknown GEDCOMVersion = iota
	// GEDCOM551 represents GEDCOM 5.5.1 specification.
	GEDCOM551
	// GEDCOM70 represents GEDCOM 7.0 specification.
	GEDCOM70
)

// GEDCOMLine represents a single parsed GEDCOM line
type GEDCOMLine struct {
	Level int
	XRef  string
	Tag   string
	Value string
	Line  int
}

// GEDCOMRecord represents a hierarchical GEDCOM record
type GEDCOMRecord struct {
	XRef       string
	Tag        string
	Value      string
	SubRecords []*GEDCOMRecord
	Line       int
}

// ImportResult contains statistics and information about the import
type ImportResult struct {
	Statistics ImportStatistics
	Version    string
}

// ImportStatistics tracks import metrics
type ImportStatistics struct {
	LinesProcessed        int
	PersonsCreated        int
	EventsCreated         int
	RelationshipsCreated  int
	PlacesCreated         int
	SourcesCreated        int
	RepositoriesCreated   int
	MediaCreated          int
	CitationsCreated      int
	AssertionsCreated     int
	ParticipationsCreated int
	Errors                []ImportError
	Warnings              []ImportWarning
}

// ImportError represents an error during import
type ImportError struct {
	Line    int
	Tag     string
	Message string
}

// ImportWarning represents a warning during import
type ImportWarning struct {
	Line    int
	Tag     string
	Message string
}

// ConversionContext holds state during GEDCOM conversion
type ConversionContext struct {
	GLX     *GLXFile
	Version GEDCOMVersion
	Logger  *ImportLogger

	// ID mapping from GEDCOM XRef to GLX ID
	PersonIDMap     map[string]string
	FamilyIDMap     map[string]string
	SourceIDMap     map[string]string
	RepositoryIDMap map[string]string
	MediaIDMap      map[string]string
	PlaceIDMap      map[string]string

	// Family structure mapping (FAM XRef -> parent IDs)
	FamilyParentsMap map[string][]string

	// Auto-increment counters for ID generation
	PersonCounter        int
	EventCounter         int
	RelationshipCounter  int
	PlaceCounter         int
	SourceCounter        int
	RepositoryCounter    int
	MediaCounter         int
	CitationCounter      int
	AssertionCounter     int
	ParticipationCounter int

	// GEDCOM 7.0 specific
	SharedNotes      map[string]string
	ExtensionSchemas map[string]*ExtensionSchema

	// Deferred processing
	DeferredFamilies    []*GEDCOMRecord
	DeferredFamilyLinks []*FamilyLink

	// Statistics
	Stats ImportStatistics
}

// ExtensionSchema represents a GEDCOM 7.0 extension schema
type ExtensionSchema struct {
	Tag         string
	URI         string
	Description string
}

// FamilyLink represents a deferred family link
type FamilyLink struct {
	PersonID     string
	FamilyRef    string
	LinkType     string // ParticipantRoleChild or ParticipantRoleSpouse
	PedigreeType string // PEDI value: birth, adopted, foster, sealed, unknown (empty = unspecified)
}

// ImportGEDCOM imports a GEDCOM file and returns a GLX archive
func ImportGEDCOM(reader io.Reader, logPath string) (*GLXFile, *ImportResult, error) {
	// Create logger
	logger, err := NewImportLogger(logPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}
	defer func() { _ = logger.Close() }()

	logger.LogInfo("Starting GEDCOM import")

	// Parse GEDCOM
	records, version, versionString, err := parseGEDCOM(reader, logger)
	if err != nil {
		logger.LogError(0, "PARSE", "", err)

		return nil, nil, fmt.Errorf("parse error: %w", err)
	}

	logger.LogInfo("Detected GEDCOM version: " + versionString)

	// Create GLX file
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Citations:     make(map[string]*Citation),
		Assertions:    make(map[string]*Assertion),
	}

	// Load standard vocabularies into GLXFile so validation works
	if err := LoadStandardVocabulariesIntoGLX(glx); err != nil {
		logger.LogError(0, "VOCAB", "", err)

		return nil, nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
	}

	// Create conversion context
	conv := &ConversionContext{
		GLX:                 glx,
		Version:             version,
		Logger:              logger,
		PersonIDMap:         make(map[string]string),
		FamilyIDMap:         make(map[string]string),
		SourceIDMap:         make(map[string]string),
		RepositoryIDMap:     make(map[string]string),
		MediaIDMap:          make(map[string]string),
		PlaceIDMap:          make(map[string]string),
		FamilyParentsMap:    make(map[string][]string),
		SharedNotes:         make(map[string]string),
		ExtensionSchemas:    make(map[string]*ExtensionSchema),
		DeferredFamilies:    []*GEDCOMRecord{},
		DeferredFamilyLinks: []*FamilyLink{},
		Stats:               ImportStatistics{},
	}

	// Perform conversion
	if err := conv.Convert(records); err != nil {
		logger.LogError(0, "CONVERT", "", err)

		return nil, nil, fmt.Errorf("conversion error: %w", err)
	}

	logger.LogInfo(fmt.Sprintf("Import completed: %d persons, %d events, %d relationships, %d sources",
		conv.Stats.PersonsCreated, conv.Stats.EventsCreated, conv.Stats.RelationshipsCreated, conv.Stats.SourcesCreated))

	// Build result
	result := &ImportResult{
		Statistics: conv.Stats,
		Version:    versionString,
	}

	return glx, result, nil
}

// parseGEDCOM parses a GEDCOM file into hierarchical records
func parseGEDCOM(reader io.Reader, logger *ImportLogger) ([]*GEDCOMRecord, GEDCOMVersion, string, error) {
	// Parse lines
	lines, err := parseGEDCOMLines(reader)
	if err != nil {
		return nil, GEDCOMUnknown, "", err
	}

	logger.LogInfo(fmt.Sprintf("Parsed %d lines", len(lines)))

	// Build records
	records := buildRecords(lines)

	logger.LogInfo(fmt.Sprintf("Built %d top-level records", len(records)))

	// Detect version
	version, versionString := detectGEDCOMVersion(records)

	return records, version, versionString, nil
}

// parseGEDCOMLines parses GEDCOM file line by line
func parseGEDCOMLines(reader io.Reader) ([]*GEDCOMLine, error) {
	var lines []*GEDCOMLine
	scanner := bufio.NewScanner(reader)

	// Increase buffer size for large GEDCOM files (torture test has long lines)
	// Default is 64KB, increase to 1MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	var lastLine *GEDCOMLine

	for scanner.Scan() {
		lineNum++
		text := scanner.Text()

		// Strip UTF-8 BOM from first line if present
		if lineNum == 1 && len(text) >= 3 {
			if text[0] == 0xEF && text[1] == 0xBB && text[2] == 0xBF {
				text = text[3:]
			}
		}

		// Skip empty lines
		if strings.TrimSpace(text) == "" {
			continue
		}

		line, err := parseGEDCOMLine(text, lineNum)
		if err != nil {
			// Handle malformed continuation lines (common in MyHeritage exports with HTML notes)
			// If parse fails and line doesn't start with a digit, treat as CONT for previous line
			if lastLine != nil && len(text) > 0 && !isDigit(text[0]) {
				// Treat as continuation of previous line
				// Append to last line's value as if it were "2 CONT <text>"
				if lastLine.Value == "" {
					lastLine.Value = text
				} else {
					lastLine.Value += "\n" + text
				}

				continue // Skip adding this as a new line
			}

			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		lines = append(lines, line)
		lastLine = line
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return lines, nil
}

// isDigit checks if a byte is a digit character
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// parseGEDCOMLine parses a single GEDCOM line
func parseGEDCOMLine(text string, lineNum int) (*GEDCOMLine, error) {
	// GEDCOM line format: LEVEL [XREF] TAG [VALUE]
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return nil, ErrInvalidGEDCOMLine
	}

	line := &GEDCOMLine{Line: lineNum}

	// Parse level
	level, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidLevel, parts[0])
	}
	line.Level = level

	// Check for XRef (starts with @)
	idx := 1
	if strings.HasPrefix(parts[1], "@") && strings.HasSuffix(parts[1], "@") {
		line.XRef = parts[1]
		idx = 2
		if len(parts) < 3 {
			return nil, ErrMissingTagAfterXRef
		}
	}

	// Parse tag
	line.Tag = parts[idx]
	idx++

	// Parse value (rest of line)
	if idx < len(parts) {
		// Rejoin the rest as value
		valueStart := strings.Index(text, parts[idx])
		line.Value = strings.TrimSpace(text[valueStart:])
	}

	return line, nil
}

// buildRecords builds hierarchical records from flat lines
func buildRecords(lines []*GEDCOMLine) []*GEDCOMRecord {
	var records []*GEDCOMRecord
	var stack []*GEDCOMRecord

	for _, line := range lines {
		record := &GEDCOMRecord{
			XRef:       line.XRef,
			Tag:        line.Tag,
			Value:      line.Value,
			SubRecords: []*GEDCOMRecord{},
			Line:       line.Line,
		}

		// Level 0 records are top-level
		if line.Level == 0 {
			records = append(records, record)
			stack = []*GEDCOMRecord{record}

			continue
		}

		// Find parent in stack
		for len(stack) > line.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.SubRecords = append(parent.SubRecords, record)
		}

		stack = append(stack, record)
	}

	return records
}

// detectGEDCOMVersion detects GEDCOM version from header
func detectGEDCOMVersion(records []*GEDCOMRecord) (GEDCOMVersion, string) {
	for _, record := range records {
		if record.Tag == "HEAD" {
			for _, sub := range record.SubRecords {
				if sub.Tag == "GEDC" {
					for _, versSub := range sub.SubRecords {
						if versSub.Tag == "VERS" {
							version := strings.TrimSpace(versSub.Value)
							if strings.HasPrefix(version, "7.") {
								return GEDCOM70, version
							}
							if strings.HasPrefix(version, "5.5") {
								return GEDCOM551, version
							}
						}
					}
				}
			}
		}
	}

	return GEDCOM551, GEDCOMVersion551 // Default to 5.5.1
}

// addError adds an error to the conversion context
func (conv *ConversionContext) addError(line int, tag string, message string) {
	conv.Stats.Errors = append(conv.Stats.Errors, ImportError{
		Line:    line,
		Tag:     tag,
		Message: message,
	})
}

// addWarning adds a warning to the conversion context
func (conv *ConversionContext) addWarning(line int, tag string, message string) {
	conv.Stats.Warnings = append(conv.Stats.Warnings, ImportWarning{
		Line:    line,
		Tag:     tag,
		Message: message,
	})
}
