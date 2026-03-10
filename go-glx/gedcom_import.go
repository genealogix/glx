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

// MediaSourceType distinguishes between file-based and blob-based media sources.
type MediaSourceType int

const (
	// MediaSourceFile indicates a file to copy from the GEDCOM source directory.
	MediaSourceFile MediaSourceType = iota
	// MediaSourceBlob indicates inline BLOB data to decode and write.
	MediaSourceBlob
)

// MediaFileSource describes a media file to be included in the archive.
// The lib layer populates these during GEDCOM import; the CLI layer
// performs the actual file I/O (copy or write).
type MediaFileSource struct {
	// MediaID is the GLX media entity ID this file belongs to.
	MediaID string

	// SourceType indicates where the file data comes from.
	SourceType MediaSourceType

	// RelativePath is the path from the GEDCOM file's directory to the source file.
	// Only set when SourceType is MediaSourceFile.
	RelativePath string

	// BlobData contains the raw BLOB text from GEDCOM 5.5.1.
	// Only set when SourceType is MediaSourceBlob.
	BlobData string

	// TargetFilename is the destination filename within media/files/.
	TargetFilename string
}

// ImportResult contains statistics and information about the import
type ImportResult struct {
	Statistics ImportStatistics
	Version    string
	MediaFiles []MediaFileSource
}

// ImportStatistics tracks import metrics
type ImportStatistics struct {
	LinesProcessed           int
	PersonsCreated           int
	EventsCreated            int
	RelationshipsCreated     int
	PlacesCreated            int
	SourcesCreated           int
	RepositoriesCreated      int
	RepositoriesDeduplicated int
	MediaCreated             int
	CitationsCreated         int
	AssertionsCreated        int
	ParticipationsCreated    int
	Errors                   []ImportError
	Warnings                 []ImportWarning
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

// scanLinesAllEndings is a split function for bufio.Scanner that handles
// all line ending formats: LF (\n), CRLF (\r\n), and CR (\r).
// Based on bufio.ScanLines but extended for CR-only files (old Mac Classic).
func scanLinesAllEndings(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for CR or LF
	for i := range data {
		if data[i] == '\n' {
			// LF - return line without the LF
			return i + 1, data[0:i], nil
		}
		if data[i] == '\r' {
			// CR - check if followed by LF (CRLF)
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					// CRLF
					return i + 2, data[0:i], nil
				}
				// CR only
				return i + 1, data[0:i], nil
			}
			// CR at end of buffer - need more data to check for CRLF
			if !atEOF {
				return 0, nil, nil
			}
			// CR at EOF - return line without CR
			return i + 1, data[0:i], nil
		}
	}

	// No line ending found
	if atEOF {
		return len(data), data, nil
	}
	// Request more data
	return 0, nil, nil
}

// GEDCOMIndex provides reverse lookups from GEDCOM tags to GLX vocabulary keys.
// Built once at import initialization from loaded vocabulary definitions.
type GEDCOMIndex struct {
	// EventTypes maps GEDCOM event tags to GLX event type keys (e.g., "BIRT" → "birth")
	EventTypes map[string]string

	// PersonProperties maps GEDCOM person attribute tags to GLX property keys (e.g., "OCCU" → "occupation")
	PersonProperties map[string]string

	// EventProperties maps GEDCOM event detail tags to GLX property keys (e.g., "AGE" → "age_at_event")
	EventProperties map[string]string

	// CitationProperties maps GEDCOM citation tags to GLX property keys (e.g., "PAGE" → "locator")
	CitationProperties map[string]string

	// SourceProperties maps GEDCOM source tags to GLX property keys (e.g., "ABBR" → "abbreviation")
	SourceProperties map[string]string

	// RepositoryProperties maps GEDCOM repository tags to GLX property keys (e.g., "PHON" → "phones")
	RepositoryProperties map[string]string

	// MediaProperties maps GEDCOM media tags to GLX property keys (e.g., "MEDI" → "medium")
	MediaProperties map[string]string
}

// buildGEDCOMIndex constructs reverse lookup indices from vocabularies in the GLXFile.
func buildGEDCOMIndex(glx *GLXFile) *GEDCOMIndex {
	index := &GEDCOMIndex{
		EventTypes:           make(map[string]string),
		PersonProperties:     make(map[string]string),
		EventProperties:      make(map[string]string),
		CitationProperties:   make(map[string]string),
		SourceProperties:     make(map[string]string),
		RepositoryProperties: make(map[string]string),
		MediaProperties:      make(map[string]string),
	}

	// Build event type index from vocabulary
	for key, eventType := range glx.EventTypes {
		if eventType.GEDCOM != "" {
			index.EventTypes[eventType.GEDCOM] = key
		}
	}

	// BASM is a non-standard alias for BATM (bat_mitzvah) used by some exporters
	if key, ok := index.EventTypes[GedcomTagBatm]; ok {
		index.EventTypes[GedcomTagBasm] = key
	}

	// Build property indices from vocabularies
	for key, propDef := range glx.PersonProperties {
		if propDef.GEDCOM != "" {
			index.PersonProperties[propDef.GEDCOM] = key
		}
	}

	for key, propDef := range glx.EventProperties {
		if propDef.GEDCOM != "" {
			index.EventProperties[propDef.GEDCOM] = key
		}
	}

	for key, propDef := range glx.CitationProperties {
		if propDef.GEDCOM != "" {
			index.CitationProperties[propDef.GEDCOM] = key
		}
	}

	for key, propDef := range glx.SourceProperties {
		if propDef.GEDCOM != "" {
			index.SourceProperties[propDef.GEDCOM] = key
		}
	}

	for key, propDef := range glx.RepositoryProperties {
		if propDef.GEDCOM != "" {
			index.RepositoryProperties[propDef.GEDCOM] = key
		}
	}

	for key, propDef := range glx.MediaProperties {
		if propDef.GEDCOM != "" {
			index.MediaProperties[propDef.GEDCOM] = key
		}
	}

	return index
}

// ConversionContext holds state during GEDCOM conversion
type ConversionContext struct {
	GLX         *GLXFile
	Version     GEDCOMVersion
	Logger      *ImportLogger
	GEDCOMIndex *GEDCOMIndex

	// ID mapping from GEDCOM XRef to GLX ID
	PersonIDMap     map[string]string
	FamilyIDMap     map[string]string
	SourceIDMap     map[string]string
	RepositoryIDMap map[string]string
	MediaIDMap      map[string]string
	PlaceIDMap      map[string]string

	// Content-based deduplication maps (name/content -> GLX ID)
	RepositoryNameMap map[string]string // repository name -> GLX ID

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

	// HEAD submitter reference (SUBM xref from HEAD record)
	SubmitterXRef string

	// GEDCOM 7.0 specific
	SharedNotes      map[string]string
	ExtensionSchemas map[string]*ExtensionSchema

	// Deferred processing
	DeferredFamilies    []*GEDCOMRecord
	DeferredFamilyLinks []*FamilyLink

	// Media file tracking (for CLI to copy/write files)
	MediaFileSources []MediaFileSource
	MediaFileNames   map[string]int // basename -> count, for dedup

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

// ImportGEDCOM imports a GEDCOM file and returns a GLX archive.
// If logWriter is non-nil, import progress and diagnostics are written to it.
// The caller is responsible for closing the writer if needed.
func ImportGEDCOM(reader io.Reader, logWriter io.Writer) (*GLXFile, *ImportResult, error) {
	// Create logger
	logger := NewImportLogger(logWriter)

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

	// Build GEDCOM reverse lookup index from loaded vocabularies
	gedcomIndex := buildGEDCOMIndex(glx)

	// Create conversion context
	conv := &ConversionContext{
		GLX:                 glx,
		Version:             version,
		Logger:              logger,
		GEDCOMIndex:         gedcomIndex,
		PersonIDMap:         make(map[string]string),
		FamilyIDMap:         make(map[string]string),
		SourceIDMap:         make(map[string]string),
		RepositoryIDMap:     make(map[string]string),
		MediaIDMap:          make(map[string]string),
		PlaceIDMap:          make(map[string]string),
		RepositoryNameMap:   make(map[string]string),
		FamilyParentsMap:    make(map[string][]string),
		SharedNotes:         make(map[string]string),
		ExtensionSchemas:    make(map[string]*ExtensionSchema),
		DeferredFamilies:    []*GEDCOMRecord{},
		DeferredFamilyLinks: []*FamilyLink{},
		MediaFileNames:      make(map[string]int),
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
		MediaFiles: conv.MediaFileSources,
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
	// Read all bytes so we can detect the CHAR encoding header and convert
	// non-UTF-8 encodings (ANSI/CP1252, ANSEL, ISO-8859-1) before parsing.
	rawBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	rawBytes = convertToUTF8(rawBytes)

	var lines []*GEDCOMLine

	scanner := bufio.NewScanner(bytes.NewReader(rawBytes))
	// Handle all line ending formats: LF, CRLF, and CR (old Mac Classic)
	scanner.Split(scanLinesAllEndings)

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
		if len(parts) < 3 { //nolint:mnd // GEDCOM line: LEVEL XREF TAG [VALUE] requires at least 3 parts
			return nil, ErrMissingTagAfterXRef
		}
	}

	// Parse tag
	line.Tag = parts[idx]
	idx++

	// Parse value (rest of line after the tag)
	if idx < len(parts) {
		// Walk past exactly idx whitespace-separated tokens (level, optional xref,
		// tag) to find where the value begins. Using strings.Index is unsafe because
		// a value token may appear earlier in the line (e.g., "2 DATE 2 AUG 1944"
		// where the day "2" matches the level number, or a tag name appearing inside
		// an xref like "@NOTE1@").
		pos := 0
		for i := 0; i < idx; i++ {
			for pos < len(text) && text[pos] == ' ' {
				pos++
			}
			for pos < len(text) && text[pos] != ' ' {
				pos++
			}
		}
		line.Value = strings.TrimSpace(text[pos:])
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
//
//nolint:gocognit,gocyclo
func detectGEDCOMVersion(records []*GEDCOMRecord) (GEDCOMVersion, string) {
	for _, record := range records {
		if record.Tag == GedcomTagHead {
			for _, sub := range record.SubRecords {
				if sub.Tag == GedcomTagGedc {
					for _, versSub := range sub.SubRecords {
						if versSub.Tag == GedcomTagVers {
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
func (conv *ConversionContext) addError(line int, tag, message string) {
	conv.Stats.Errors = append(conv.Stats.Errors, ImportError{
		Line:    line,
		Tag:     tag,
		Message: message,
	})
}

// addWarning adds a warning to the conversion context
func (conv *ConversionContext) addWarning(line int, tag, message string) {
	conv.Stats.Warnings = append(conv.Stats.Warnings, ImportWarning{
		Line:    line,
		Tag:     tag,
		Message: message,
	})
}
