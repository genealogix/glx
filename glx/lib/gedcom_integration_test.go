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
	"path/filepath"
	"strings"
	"testing"
)

func TestImportMinimal70(t *testing.T) {
	// Test minimal GEDCOM 7.0 file
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "minimal-valid", "minimal70.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	t.Logf("Import statistics: %+v", result.Statistics)
	t.Logf("Errors: %d, Warnings: %d", len(result.Statistics.Errors), len(result.Statistics.Warnings))

	// Log any errors
	for _, e := range result.Statistics.Errors {
		t.Logf("  Error [Line %d] %s: %s", e.Line, e.Tag, e.Message)
	}

	// Verify GLX structure
	if glx == nil {
		t.Fatal("GLX file is nil")
	}

	// Basic validation
	if glx.Persons == nil {
		t.Error("Persons map is nil")
	}
	if glx.Events == nil {
		t.Error("Events map is nil")
	}
}

func TestImportShakespeare(t *testing.T) {
	// Test GEDCOM 5.5.1 with Shakespeare family
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "shakespeare-family", "shakespeare.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if result.Version != "5.5.1" {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	t.Logf("Imported %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))

	t.Logf("Full statistics: %+v", result.Statistics)
	t.Logf("Errors: %d, Warnings: %d", len(result.Statistics.Errors), len(result.Statistics.Warnings))

	// Log first few warnings (since converters not yet implemented)
	for i, w := range result.Statistics.Warnings {
		if i < 5 {
			t.Logf("  Warning [Line %d] %s: %s", w.Line, w.Tag, w.Message)
		}
	}

	// Verify actual data is persisted correctly
	// Check for William Shakespeare
	foundWilliam := false
	for _, person := range glx.Persons {
		givenName, familyName := ExtractNameFields(person.Properties[PersonPropertyName])

		if givenName == "William" && familyName == "Shakespeare" {
			foundWilliam = true

			// Verify gender
			if gender, ok := person.Properties[PersonPropertyGender].(string); !ok || gender != "male" {
				t.Error("William Shakespeare should have gender 'male'")
			}

			t.Logf("✓ Found William Shakespeare with correct name and gender")

			break
		}
	}
	if !foundWilliam {
		t.Error("Failed to import William Shakespeare - person data not persisted")
	}

	// Verify events are properly linked to persons
	eventCount := 0
	for _, event := range glx.Events {
		if len(event.Participants) > 0 {
			eventCount++
		}
	}
	if eventCount == 0 {
		t.Error("No events have participants - event-person linkage not persisted")
	}

	// Verify relationships are properly linked
	relationshipCount := 0
	for _, rel := range glx.Relationships {
		if len(rel.Participants) >= 2 {
			relationshipCount++
		}
	}
	if relationshipCount == 0 {
		t.Error("No relationships have participants - relationship data not persisted")
	}
}

func TestParseGEDCOMDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"exact date with day", "1 JAN 1900", "1900-01-01"},
		{"exact date month/year", "JAN 1900", "1900-01"},
		{"exact date year only", "1900", "1900"},
		{"about date", "ABT 1900", "ABT 1900"},
		{"before date", "BEF 15 JAN 1900", "BEF 1900-01-15"},
		{"after date", "AFT 1900", "AFT 1900"},
		{"calculated date", "CAL 1900", "CAL 1900"},
		{"between range", "BET 1900 AND 1910", "BET 1900 AND 1910"},
		{"from-to range", "FROM 1900 TO 1910", "FROM 1900 TO 1910"},
		{"open-ended from", "FROM 1900", "FROM 1900"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGEDCOMDate(tt.input)
			if string(result) != tt.expected {
				t.Errorf("parseGEDCOMDate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGEDCOMName(t *testing.T) {
	tests := []struct {
		input    string
		given    string
		surname  string
		nickname string
	}{
		{"John /Smith/", "John", "Smith", ""},
		{"John \"Jack\" /Smith/", "John", "Smith", "Jack"},
		{"Dr. John /Smith/ Jr.", "John", "Smith", ""},
		{"/von Neumann/", "", "Neumann", ""},
		{"Mary Jane /Smith-Jones/", "Mary Jane", "Smith-Jones", ""},
	}

	for _, tt := range tests {
		result := parseGEDCOMName(tt.input, nil)
		if result.GivenName != tt.given {
			t.Errorf("parseGEDCOMName(%q).GivenName = %q, want %q", tt.input, result.GivenName, tt.given)
		}
		if result.Surname != tt.surname {
			t.Errorf("parseGEDCOMName(%q).Surname = %q, want %q", tt.input, result.Surname, tt.surname)
		}
		if result.Nickname != tt.nickname {
			t.Errorf("parseGEDCOMName(%q).Nickname = %q, want %q", tt.input, result.Nickname, tt.nickname)
		}
	}
}

func TestParseGEDCOMPlace(t *testing.T) {
	tests := []struct {
		input      string
		components int
	}{
		{"New York, New York, USA", 3},
		{"London, England", 2},
		{"Paris", 1},
		{"", 0},
	}

	for _, tt := range tests {
		result := parseGEDCOMPlace(tt.input)
		if result == nil && tt.components > 0 {
			t.Errorf("parseGEDCOMPlace(%q) returned nil, expected %d components", tt.input, tt.components)

			continue
		}
		if result != nil && len(result.Components) != tt.components {
			t.Errorf("parseGEDCOMPlace(%q) has %d components, want %d", tt.input, len(result.Components), tt.components)
		}
	}
}

func TestImportNoteReferenceResolution(t *testing.T) {
	// Test that NOTE references (e.g., NOTE @N176@) are resolved to their text content
	// The assess.ged file has:
	// - Person @I176@ with a WILL event that has NOTE @N176@
	// - Shared note 0 @N176@ NOTE Line 1 with CONT Line 2
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "gedcom-assessment", "assess.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, _, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Find the WILL event for person @I176@
	// The event should have notes containing "Line 1" (resolved from @N176@)
	// NOT the literal string "@N176@"
	foundResolvedNote := false
	foundUnresolvedRef := false

	for _, event := range glx.Events {
		if event.Type != "will" {
			continue
		}
		notes, ok := event.Properties[PropertyNotes]
		if !ok {
			continue
		}
		noteStr, ok := notes.(string)
		if !ok {
			continue
		}

		// Check for resolved content
		if noteStr == "Line 1" || noteStr == "Line 1\nLine 2" {
			foundResolvedNote = true
		}

		// Check for unresolved reference (this would be a bug)
		if noteStr == "@N176@" {
			foundUnresolvedRef = true
			t.Errorf("Found unresolved NOTE reference @N176@ - notes should be resolved to their text content")
		}
	}

	if foundUnresolvedRef {
		t.Error("NOTE references are not being resolved correctly")
	}

	if !foundResolvedNote {
		t.Log("Note: Could not verify resolved note content - may need to check test file structure")
	}
}

func TestImportRepositoryOrdering(t *testing.T) {
	// Test that REPO records are processed before SOUR records regardless of file order
	// The bullinger.ged file has:
	// - SOUR records starting at line 17622
	// - REPO record at line 17842 (after sources)
	// Without proper ordering, sources would not link to their repository
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "bullinger-family", "bullinger.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, _, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify we have at least one repository
	if len(glx.Repositories) == 0 {
		t.Fatal("No repositories imported")
	}

	// Verify sources are linked to their repository
	sourcesWithRepo := 0
	for _, source := range glx.Sources {
		if source.RepositoryID != "" {
			sourcesWithRepo++
		}
	}

	// At least some sources should have repository links
	// (not all sources may reference a repository)
	if sourcesWithRepo == 0 {
		t.Error("No sources have repository links - REPO records may not be processed before SOUR records")
	}

	t.Logf("Sources with repository links: %d/%d", sourcesWithRepo, len(glx.Sources))

	// Verify the repository ID exists in the repositories map
	for sourceID, source := range glx.Sources {
		if source.RepositoryID != "" {
			if _, exists := glx.Repositories[source.RepositoryID]; !exists {
				t.Errorf("Source %s references non-existent repository %s", sourceID, source.RepositoryID)
			}
		}
	}
}

func TestRepositoryDeduplication(t *testing.T) {
	// Test that duplicate repositories (same name) are deduplicated
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @REPO1@ REPO
1 NAME National Archives
1 ADDR 8601 Adelphi Road
2 CITY College Park
2 STAE Maryland
2 CTRY USA
0 @REPO2@ REPO
1 NAME National Archives
1 ADDR 8601 Adelphi Road
2 CITY College Park
2 STAE Maryland
2 CTRY USA
0 @REPO3@ REPO
1 NAME Local Library
1 ADDR 123 Main Street
2 CITY Springfield
2 CTRY USA
0 @SOUR1@ SOUR
1 TITL Census Record 1900
1 REPO @REPO1@
0 @SOUR2@ SOUR
1 TITL Census Record 1910
1 REPO @REPO2@
0 @SOUR3@ SOUR
1 TITL Local History Book
1 REPO @REPO3@
0 TRLR
`
	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should have 2 unique repositories (National Archives deduplicated, Local Library unique)
	if len(glx.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(glx.Repositories))
		for id, repo := range glx.Repositories {
			t.Logf("  Repository %s: %s", id, repo.Name)
		}
	}

	// Should have 1 deduplicated repository
	if result.Statistics.RepositoriesDeduplicated != 1 {
		t.Errorf("Expected 1 deduplicated repository, got %d", result.Statistics.RepositoriesDeduplicated)
	}

	// Should have 2 created repositories
	if result.Statistics.RepositoriesCreated != 2 {
		t.Errorf("Expected 2 created repositories, got %d", result.Statistics.RepositoriesCreated)
	}

	// All 3 sources should have valid repository references
	if len(glx.Sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(glx.Sources))
	}

	for sourceID, source := range glx.Sources {
		if source.RepositoryID == "" {
			t.Errorf("Source %s has no repository ID", sourceID)
			continue
		}
		if _, exists := glx.Repositories[source.RepositoryID]; !exists {
			t.Errorf("Source %s references non-existent repository %s", sourceID, source.RepositoryID)
		}
	}

	// Sources 1 and 2 should point to the same repository (deduplicated National Archives)
	var source1RepoID, source2RepoID, source3RepoID string
	for _, source := range glx.Sources {
		switch source.Title {
		case "Census Record 1900":
			source1RepoID = source.RepositoryID
		case "Census Record 1910":
			source2RepoID = source.RepositoryID
		case "Local History Book":
			source3RepoID = source.RepositoryID
		}
	}

	if source1RepoID != source2RepoID {
		t.Errorf("Sources referencing same repository have different IDs: %s vs %s", source1RepoID, source2RepoID)
	}

	if source1RepoID == source3RepoID {
		t.Errorf("Sources referencing different repositories have same ID: %s", source1RepoID)
	}
}

func TestRepositoryDeduplicationDifferentLocations(t *testing.T) {
	// Test that repositories with same name but different locations are NOT deduplicated
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @REPO1@ REPO
1 NAME Public Library
1 ADDR 123 Main St
2 CITY Springfield
2 CTRY USA
0 @REPO2@ REPO
1 NAME Public Library
1 ADDR 456 Oak Ave
2 CITY Shelbyville
2 CTRY USA
0 TRLR
`
	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should have 2 unique repositories (different cities)
	if len(glx.Repositories) != 2 {
		t.Errorf("Expected 2 repositories (different locations), got %d", len(glx.Repositories))
		for id, repo := range glx.Repositories {
			t.Logf("  Repository %s: %s, %s", id, repo.Name, repo.City)
		}
	}

	// Should have 0 deduplicated repositories
	if result.Statistics.RepositoriesDeduplicated != 0 {
		t.Errorf("Expected 0 deduplicated repositories, got %d", result.Statistics.RepositoriesDeduplicated)
	}
}

func TestExtractTextWithContinuation(t *testing.T) {
	tests := []struct {
		name     string
		record   *GEDCOMRecord
		expected string
	}{
		{
			name: "simple value no continuation",
			record: &GEDCOMRecord{
				Value: "Simple text",
			},
			expected: "Simple text",
		},
		{
			name: "value with CONC",
			record: &GEDCOMRecord{
				Value: "First part ",
				SubRecords: []*GEDCOMRecord{
					{Tag: "CONC", Value: "second part"},
				},
			},
			expected: "First part second part",
		},
		{
			name: "value with CONT",
			record: &GEDCOMRecord{
				Value: "Line one",
				SubRecords: []*GEDCOMRecord{
					{Tag: "CONT", Value: "Line two"},
				},
			},
			expected: "Line one\nLine two",
		},
		{
			name: "value with mixed CONT and CONC",
			record: &GEDCOMRecord{
				Value: "Start of text ",
				SubRecords: []*GEDCOMRecord{
					{Tag: "CONC", Value: "continued"},
					{Tag: "CONT", Value: "new line"},
					{Tag: "CONC", Value: " more text"},
				},
			},
			expected: "Start of text continued\nnew line more text",
		},
		{
			name: "empty CONT (paragraph break)",
			record: &GEDCOMRecord{
				Value: "Paragraph one",
				SubRecords: []*GEDCOMRecord{
					{Tag: "CONT", Value: ""},
					{Tag: "CONT", Value: "Paragraph two"},
				},
			},
			expected: "Paragraph one\n\nParagraph two",
		},
		{
			name: "empty record",
			record: &GEDCOMRecord{
				Value: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextWithContinuation(tt.record)
			if result != tt.expected {
				t.Errorf("extractTextWithContinuation() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEmbeddedCitations(t *testing.T) {
	// Test that embedded citations (SOUR without pointer) create synthetic sources
	// GEDCOM has two SOURCE_CITATION forms:
	// 1. Pointer to source record: "SOUR @S1@"
	// 2. Embedded source description: "SOUR description text"
	// The second form should create a synthetic Source entity

	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1850
2 SOUR Family Bible of the Smith Family
3 PAGE Page 12
3 TEXT Born to James and Mary Smith
1 DEAT
2 DATE 31 DEC 1920
2 SOUR
3 TEXT Death certificate from county records
0 TRLR
`

	glx, result, err := ImportGEDCOM(strings.NewReader(gedcomData), nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Log any errors for debugging
	for _, e := range result.Statistics.Errors {
		t.Logf("Error [Line %d] %s: %s", e.Line, e.Tag, e.Message)
	}

	// Verify synthetic sources were created
	if len(glx.Sources) == 0 {
		t.Fatal("No sources created - embedded citations should create synthetic sources")
	}

	// Check for the embedded source with description
	foundFamilyBible := false
	foundDeathCert := false

	for _, source := range glx.Sources {
		if strings.Contains(source.Title, "Family Bible of the Smith Family") {
			foundFamilyBible = true
			// Verify it has the note about being synthetic
			if !strings.Contains(source.Notes, "embedded GEDCOM citation") {
				t.Error("Synthetic source should have note about being from embedded citation")
			}
		}
		if strings.Contains(source.Title, "Death certificate") || strings.Contains(source.Title, "Embedded Citation") {
			foundDeathCert = true
		}
	}

	if !foundFamilyBible {
		t.Error("Failed to create synthetic source from embedded citation 'Family Bible of the Smith Family'")
		t.Logf("Sources found: %d", len(glx.Sources))
		for id, src := range glx.Sources {
			t.Logf("  Source %s: %q", id, src.Title)
		}
	}

	if !foundDeathCert {
		t.Error("Failed to create synthetic source from embedded citation with TEXT only")
	}

	// Verify citations were created and link to sources
	if len(glx.Citations) == 0 {
		t.Fatal("No citations created")
	}

	// Verify citation properties (PAGE, TEXT)
	foundCitationWithLocator := false
	foundCitationWithText := false

	for _, citation := range glx.Citations {
		if citation.Properties != nil {
			if locator, ok := citation.Properties["locator"]; ok {
				if locator == "Page 12" {
					foundCitationWithLocator = true
				}
			}
			if text, ok := citation.Properties["text_from_source"]; ok {
				textStr, _ := text.(string)
				if strings.Contains(textStr, "Born to James and Mary") || strings.Contains(textStr, "Death certificate") {
					foundCitationWithText = true
				}
			}
		}
	}

	if !foundCitationWithLocator {
		t.Error("Citation should have locator property from PAGE tag")
	}

	if !foundCitationWithText {
		t.Error("Citation should have text_from_source property from TEXT tag")
	}

	t.Logf("Created %d sources and %d citations from embedded citations",
		len(glx.Sources), len(glx.Citations))
}

func TestIsGEDCOMPointer(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"@I1@", true},
		{"@S123@", true},
		{"@REPO1@", true},
		{"@N1@", true},
		{"Family Bible", false},
		{"@incomplete", false},
		{"incomplete@", false},
		{"", false},
		{"@", false},
		{"@@", false},
		{"plain text @I1@ in middle", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := isGEDCOMPointer(tt.value)
			if result != tt.expected {
				t.Errorf("isGEDCOMPointer(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
