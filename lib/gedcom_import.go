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
	"os"
	"strconv"
	"strings"
)

// GEDCOMVersion represents the version of GEDCOM file being parsed
type GEDCOMVersion string

const (
	GEDCOM551 GEDCOMVersion = "5.5.1"
	GEDCOM70  GEDCOMVersion = "7.0"
)

// GEDCOMLine represents a single line in a GEDCOM file
type GEDCOMLine struct {
	Level int
	XRef  string // Cross-reference ID (e.g., "@I1@")
	Tag   string
	Value string
	Line  int // Line number in file for error reporting
}

// GEDCOMRecord represents a top-level GEDCOM record with its subordinate lines
type GEDCOMRecord struct {
	XRef       string
	Tag        string
	Value      string
	SubRecords []*GEDCOMRecord
	Line       int
}

// ImportGEDCOMFromFile reads a GEDCOM file and converts it to a GLXFile structure.
// It supports both GEDCOM 5.5.1 and GEDCOM 7.0 formats.
//
// The function performs the following steps:
// 1. Parse the GEDCOM file into structured records
// 2. Detect the GEDCOM version from the header
// 3. Convert GEDCOM records to GLX entities
// 4. Return a populated GLXFile ready for validation
//
// Note: This function builds the GLX archive in memory but does not write it to disk.
func ImportGEDCOMFromFile(filepath string) (*GLXFile, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GEDCOM file: %w", err)
	}
	defer file.Close()

	return ImportGEDCOM(file)
}

// ImportGEDCOM reads a GEDCOM file from an io.Reader and converts it to a GLXFile structure.
func ImportGEDCOM(r io.Reader) (*GLXFile, error) {
	// Parse GEDCOM into structured records
	records, version, err := parseGEDCOM(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GEDCOM: %w", err)
	}

	// Convert to GLX based on version
	glx, err := convertToGLX(records, version)
	if err != nil {
		return nil, fmt.Errorf("failed to convert GEDCOM to GLX: %w", err)
	}

	return glx, nil
}

// parseGEDCOM reads a GEDCOM file and parses it into structured records
func parseGEDCOM(r io.Reader) ([]*GEDCOMRecord, GEDCOMVersion, error) {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	var lines []*GEDCOMLine
	version := GEDCOM551 // Default version

	// First pass: Parse all lines
	for scanner.Scan() {
		lineNum++
		text := strings.TrimRight(scanner.Text(), "\r\n")

		if text == "" {
			continue // Skip empty lines
		}

		line, err := parseGEDCOMLine(text, lineNum)
		if err != nil {
			return nil, "", fmt.Errorf("line %d: %w", lineNum, err)
		}

		lines = append(lines, line)

		// Detect GEDCOM version from header
		if line.Tag == "VERS" && len(lines) > 1 {
			// Check if previous line was GEDC
			if lines[len(lines)-2].Tag == "GEDC" {
				if strings.HasPrefix(line.Value, "5.5") {
					version = GEDCOM551
				} else if strings.HasPrefix(line.Value, "7.") {
					version = GEDCOM70
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading GEDCOM file: %w", err)
	}

	// Second pass: Build hierarchical records
	records, err := buildRecords(lines)
	if err != nil {
		return nil, "", err
	}

	return records, version, nil
}

// parseGEDCOMLine parses a single GEDCOM line into its components
func parseGEDCOMLine(text string, lineNum int) (*GEDCOMLine, error) {
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GEDCOM line: too few fields")
	}

	level, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid level number: %w", err)
	}

	line := &GEDCOMLine{
		Level: level,
		Line:  lineNum,
	}

	// Check if second field is an XRef (starts with @)
	if strings.HasPrefix(parts[1], "@") && strings.HasSuffix(parts[1], "@") {
		line.XRef = parts[1]
		if len(parts) >= 3 {
			line.Tag = parts[2]
		}
		if len(parts) >= 4 {
			line.Value = strings.Join(parts[3:], " ")
		}
	} else {
		line.Tag = parts[1]
		if len(parts) >= 3 {
			line.Value = strings.Join(parts[2:], " ")
		}
	}

	return line, nil
}

// buildRecords converts flat GEDCOM lines into hierarchical records
func buildRecords(lines []*GEDCOMLine) ([]*GEDCOMRecord, error) {
	var records []*GEDCOMRecord
	var stack []*GEDCOMRecord

	for _, line := range lines {
		record := &GEDCOMRecord{
			XRef:  line.XRef,
			Tag:   line.Tag,
			Value: line.Value,
			Line:  line.Line,
		}

		if line.Level == 0 {
			// Top-level record
			records = append(records, record)
			stack = []*GEDCOMRecord{record}
		} else {
			// Find parent at level-1
			if line.Level > len(stack) {
				return nil, fmt.Errorf("line %d: invalid level jump from %d to %d",
					line.Line, len(stack)-1, line.Level)
			}

			// Trim stack to parent level
			stack = stack[:line.Level]
			parent := stack[len(stack)-1]
			parent.SubRecords = append(parent.SubRecords, record)
			stack = append(stack, record)
		}
	}

	return records, nil
}

// convertToGLX converts parsed GEDCOM records to a GLX archive structure
func convertToGLX(records []*GEDCOMRecord, version GEDCOMVersion) (*GLXFile, error) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Relationships: make(map[string]*Relationship),
		Events:        make(map[string]*Event),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
	}

	// TODO: Implement conversion logic for each record type
	// This is a placeholder for the actual conversion implementation

	for _, record := range records {
		switch record.Tag {
		case "HEAD":
			// Process header - extract metadata
			// TODO: Store GEDCOM metadata in properties
		case "INDI":
			// TODO: Convert individual record to Person
			// TODO: Extract events, create Event entities
			// TODO: Handle FAMC/FAMS to create Relationships
		case "FAM":
			// TODO: Convert family record to Relationship(s)
			// TODO: Create marriage/divorce events
		case "SOUR":
			// TODO: Convert source record to Source
		case "REPO":
			// TODO: Convert repository record to Repository
		case "OBJE":
			// TODO: Convert multimedia record to Media
		case "SNOTE":
			// TODO: Handle shared notes (GEDCOM 7.0)
		case "TRLR":
			// Trailer - end of file
		default:
			// Unknown or custom record type
			// TODO: Log warning
		}
	}

	return glx, nil
}

// Helper functions (to be implemented)

// convertIndividual converts a GEDCOM INDI record to a GLX Person
func convertIndividual(record *GEDCOMRecord, glx *GLXFile) error {
	// TODO: Implement
	return nil
}

// convertFamily converts a GEDCOM FAM record to GLX Relationship(s)
func convertFamily(record *GEDCOMRecord, glx *GLXFile) error {
	// TODO: Implement
	return nil
}

// convertSource converts a GEDCOM SOUR record to a GLX Source
func convertSource(record *GEDCOMRecord, glx *GLXFile) error {
	// TODO: Implement
	return nil
}

// convertRepository converts a GEDCOM REPO record to a GLX Repository
func convertRepository(record *GEDCOMRecord, glx *GLXFile) error {
	// TODO: Implement
	return nil
}

// convertMedia converts a GEDCOM OBJE record to a GLX Media
func convertMedia(record *GEDCOMRecord, glx *GLXFile) error {
	// TODO: Implement
	return nil
}

// parseGEDCOMDate converts a GEDCOM date string to GLX date format
func parseGEDCOMDate(gedcomDate string) (interface{}, error) {
	// TODO: Implement date parsing
	// Handle: exact dates, ranges, qualifiers (ABT, BEF, AFT, etc.)
	return gedcomDate, nil
}

// parseGEDCOMPlace converts a GEDCOM place string to a GLX Place entity
func parseGEDCOMPlace(placeName string, glx *GLXFile) (string, error) {
	// TODO: Implement place parsing
	// Handle hierarchical places (comma-separated)
	// Create parent-child relationships
	return "", nil
}

// parseGEDCOMName parses a GEDCOM name (e.g., "John /Smith/") into components
func parseGEDCOMName(gedcomName string) (given, surname string) {
	// TODO: Implement name parsing
	// Handle /surname/ notation
	return "", ""
}
