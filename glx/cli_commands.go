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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via ldflags:
//
//	go build -ldflags "-X main.version=1.2.3 -X main.commit=abc123 -X main.date=2025-01-01" ./glx
//
// GoReleaser sets these automatically. For local builds they default to "dev"/""/"".
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func versionString() string {
	v := version
	if commit != "" {
		v += " (" + commit[:min(len(commit), 7)] + ")"
	}
	if date != "" {
		v += " " + date
	}
	return v
}

// Root command
var rootCmd = &cobra.Command{
	Use:   "glx",
	Short: "GENEALOGIX CLI - Manage and validate genealogy archives",
	Long: `GLX is the official command-line tool for working with GENEALOGIX family archives.

GENEALOGIX is a modern, evidence-first, Git-native genealogy data standard.
Use GLX to initialize new archives, validate files, and ensure data quality.`,
	Version:       versionString(),
	SilenceErrors: true,
	// SilenceUsage is set in PersistentPreRun (after arg validation) so that
	// arg-count errors still show usage but runtime errors from RunE do not.
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		cmd.SilenceUsage = true
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("glx version {{.Version}}\n")
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(splitCmd)
	rootCmd.AddCommand(joinCmd)
	rootCmd.AddCommand(placesCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(citeCmd)
	rootCmd.AddCommand(ancestorsCmd)
	rootCmd.AddCommand(descendantsCmd)
	rootCmd.AddCommand(summaryCmd)
	rootCmd.AddCommand(timelineCmd)
	rootCmd.AddCommand(vitalsCmd)
	rootCmd.AddCommand(censusCmd)
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(pathCmd)
	rootCmd.AddCommand(duplicatesCmd)
	rootCmd.AddCommand(coverageCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(mergeCmd)
	rootCmd.AddCommand(migrateCmd)
}

// ============================================================================
// Import Command
// ============================================================================

// Format constants for import output format flag
const (
	FormatSingle = "single"
	FormatMulti  = "multi"
)

var (
	importOutput          string
	importFormat          string
	importNoValidate      bool
	importVerbose         bool
	importShowFirstErrors int
)

var importCmd = &cobra.Command{
	Use:   "import <gedcom-file>",
	Short: "Import a GEDCOM file to GLX format",
	Long: `Import a GEDCOM file and convert it to GLX format.

Supports both GEDCOM 5.5.1 and GEDCOM 7.0 formats.

The imported archive will include:
- All individuals (persons)
- All events (births, deaths, marriages, etc.)
- All relationships (parent-child, spouse, etc.)
- All places with hierarchical structure
- All sources and citations
- All repositories and media
- Evidence-based assertions

Output formats:
- multi: Multi-file directory structure (default, one file per entity)
- single: Single YAML file`,
	Example: `  # Import to multi-file directory (default)
  glx import family.ged -o family-archive

  # Import to single file
  glx import family.ged -o family.glx --format single

  # Import without validation
  glx import family.ged -o family-archive --no-validate`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "", "Output file or directory (required)")
	importCmd.Flags().StringVarP(&importFormat, "format", "f", FormatMulti, "Output format: multi or single")
	importCmd.Flags().BoolVar(&importNoValidate, "no-validate", false, "Skip validation before saving")
	importCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "Verbose output")
	importCmd.Flags().IntVar(&importShowFirstErrors, "show-first-errors", defaultShowFirstErrors, "Number of validation errors to show (0 for all)")

	_ = importCmd.MarkFlagRequired("output")
}

func runImport(_ *cobra.Command, args []string) error {
	return importGEDCOM(args[0], importOutput, importFormat, !importNoValidate, importVerbose, importShowFirstErrors)
}

// ============================================================================
// Export Command
// ============================================================================

var (
	exportOutput  string
	exportFormat  string
	exportVerbose bool
)

var exportCmd = &cobra.Command{
	Use:   "export <glx-archive>",
	Short: "Export a GLX archive to GEDCOM format",
	Long: `Export a GLX archive to GEDCOM format.

Supports both GEDCOM 5.5.1 and GEDCOM 7.0 output formats.

The input can be either a single-file GLX archive (.glx) or a multi-file
archive directory.

The exported GEDCOM file will include:
- All individuals (INDI records)
- All families (FAM records, reconstructed from relationships)
- All sources (SOUR records)
- All repositories (REPO records)
- All media objects (OBJE records)
- Events, places, citations, and notes`,
	Example: `  # Export to GEDCOM 5.5.1 (default)
  glx export family-archive -o family.ged

  # Export a single-file archive
  glx export family.glx -o family.ged

  # Export to GEDCOM 7.0
  glx export family-archive -o family.ged --format 70

  # Export with verbose output
  glx export family-archive -o family.ged --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output GEDCOM file path (required)")
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", ExportFormat551, "GEDCOM version: 551 or 70")
	exportCmd.Flags().BoolVarP(&exportVerbose, "verbose", "v", false, "Verbose output")

	_ = exportCmd.MarkFlagRequired("output")
}

func runExport(_ *cobra.Command, args []string) error {
	return exportToGEDCOM(args[0], exportOutput, exportFormat, exportVerbose)
}

// ============================================================================
// Init Command
// ============================================================================

var (
	initSingleFile bool
	createTestData int
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new GENEALOGIX archive in the specified directory",
	Long: `Initialize a new GENEALOGIX archive with the proper directory structure.

If a directory is provided, it will be created. If no directory is provided,
the current directory will be used (but must be empty).

By default, creates a multi-file archive with separate directories for each
entity type (persons/, events/, places/, etc.) along with standard vocabulary
files and supporting documentation.

Use --single-file to create a single archive.glx file instead.`,
	Example: `  # Initialize in a new directory
  glx init my-family-archive

  # Initialize a single-file archive in a new directory
  glx init my-family-archive --single-file

  # Initialize with test data in a new directory
  glx init my-family-archive --create-test-data 10`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitCmd,
}

func init() {
	initCmd.Flags().BoolVarP(&initSingleFile, "single-file", "s", false, "create a single-file archive instead of multi-file")
	initCmd.Flags().IntVarP(&createTestData, "create-test-data", "t", 0, "number of persons to generate test data for")
}

func runInitCmd(_ *cobra.Command, args []string) error {
	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
	} else {
		targetDir = "."
	}

	return runInit(targetDir, initSingleFile, createTestData)
}

// ============================================================================
// Validate Command
// ============================================================================

var validateReport bool

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate GLX files and cross-references",
	Long: `Validate GENEALOGIX (.glx) files for correctness and integrity.

Performs comprehensive validation including:
- YAML syntax correctness
- Required fields presence
- Entity ID format validation
- Cross-reference integrity (directories only)
- Duplicate ID detection (directories only)
- Vocabulary validation (if vocabularies/ exists)

Validation behavior:
- Single file: Validates file structure only, skips cross-reference checks
- Directory: Validates all .glx files with full cross-reference validation
- No arguments: Validates current directory with full cross-reference validation

Use --report to generate a confidence summary showing assertion coverage
and highlighting unsupported claims.`,
	Example: `  # Validate current directory (with cross-reference checks)
  glx validate

  # Validate specific directory (with cross-reference checks)
  glx validate persons/

  # Validate multiple paths (with cross-reference checks)
  glx validate persons/ events/ places/

  # Validate single file (structure only, no cross-reference checks)
  glx validate archive.glx

  # Generate confidence summary report
  glx validate --report
  glx validate path/to/archive --report`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&validateReport, "report", false, "Generate confidence summary report")
}

func runValidate(_ *cobra.Command, args []string) error {
	if validateReport {
		if len(args) > 1 {
			return fmt.Errorf("--report accepts at most one path argument")
		}
		path := "."
		if len(args) == 1 {
			path = args[0]
		}

		return confidenceReport(path)
	}

	return validatePaths(args)
}

// ============================================================================
// Split Command
// ============================================================================

var (
	splitNoValidate      bool
	splitVerbose         bool
	splitShowFirstErrors int
)

var splitCmd = &cobra.Command{
	Use:   "split <input-file> <output-directory>",
	Short: "Split a single-file GLX archive into multi-file format",
	Long: `Split a single-file GLX archive into a multi-file directory structure.

The multi-file format organizes entities into separate directories:
- persons/ - One file per person (person-{id}.glx)
- events/ - One file per event (event-{id}.glx)
- relationships/ - One file per relationship (relationship-{id}.glx)
- places/ - One file per place (place-{id}.glx)
- sources/ - One file per source (source-{id}.glx)
- citations/ - One file per citation (citation-{id}.glx)
- repositories/ - One file per repository (repository-{id}.glx)
- media/ - One file per media object (media-{id}.glx)
- assertions/ - One file per assertion (assertion-{id}.glx)
- vocabularies/ - Standard vocabulary definitions

Each entity file uses standard GLX structure with the entity ID as the map key.`,
	Example: `  # Split an archive
  glx split family.glx family-archive

  # Split without validation
  glx split family.glx family-archive --no-validate`,
	Args: cobra.ExactArgs(2),
	RunE: runSplit,
}

func init() {
	splitCmd.Flags().BoolVar(&splitNoValidate, "no-validate", false, "Skip validation before splitting")
	splitCmd.Flags().BoolVarP(&splitVerbose, "verbose", "v", false, "Verbose output")
	splitCmd.Flags().IntVar(&splitShowFirstErrors, "show-first-errors", defaultShowFirstErrors, "Number of validation errors to show (0 for all)")
}

func runSplit(_ *cobra.Command, args []string) error {
	return splitArchive(args[0], args[1], !splitNoValidate, splitVerbose, splitShowFirstErrors)
}

// ============================================================================
// Join Command
// ============================================================================

var (
	joinNoValidate      bool
	joinVerbose         bool
	joinShowFirstErrors int
)

var joinCmd = &cobra.Command{
	Use:   "join <input-directory> <output-file>",
	Short: "Join a multi-file GLX archive into single-file format",
	Long: `Join a multi-file GLX archive into a single YAML file.

Reads entity files from a multi-file directory structure and combines them
into a single GLX archive file.

The multi-file structure should contain:
- persons/ - Person entity files
- events/ - Event entity files
- relationships/ - Relationship entity files
- places/ - Place entity files
- sources/ - Source entity files
- citations/ - Citation entity files
- repositories/ - Repository entity files
- media/ - Media entity files
- assertions/ - Assertion entity files

Entity IDs are read from the map key in each file.`,
	Example: `  # Join an archive
  glx join family-archive family.glx

  # Join without validation
  glx join family-archive family.glx --no-validate`,
	Args: cobra.ExactArgs(2),
	RunE: runJoin,
}

func init() {
	joinCmd.Flags().BoolVar(&joinNoValidate, "no-validate", false, "Skip validation before joining")
	joinCmd.Flags().BoolVarP(&joinVerbose, "verbose", "v", false, "Verbose output")
	joinCmd.Flags().IntVar(&joinShowFirstErrors, "show-first-errors", defaultShowFirstErrors, "Number of validation errors to show (0 for all)")
}

func runJoin(_ *cobra.Command, args []string) error {
	return joinArchive(args[0], args[1], !joinNoValidate, joinVerbose, joinShowFirstErrors)
}

// ============================================================================
// Places Command
// ============================================================================

var placesCmd = &cobra.Command{
	Use:   "places [path]",
	Short: "Analyze places for ambiguity and completeness",
	Long: `Analyze places in a GENEALOGIX archive for data quality issues.

Reports:
- Duplicate names: places that share the same name (ambiguous without context)
- Missing coordinates: places without latitude/longitude
- Missing type: places without a type classification
- No parent: non-country/region places missing a parent (hierarchy gap)
- Dangling parent: places referencing a parent that doesn't exist in the archive
- Unreferenced: places not used by any event, assertion, or as a parent

Each place is shown with its full canonical hierarchy path.
If no path is given, uses the current directory.`,
	Example: `  # Analyze places in current directory
  glx places

  # Analyze places in a specific archive
  glx places my-family-archive

  # Analyze a single-file archive
  glx places family.glx`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPlaces,
}

func runPlaces(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	return analyzePlaces(path)
}

// ============================================================================
// Query Command
// ============================================================================

var (
	queryArchive    string
	queryName       string
	queryBornBefore int
	queryBornAfter  int
	queryType       string
	queryBefore     int
	queryAfter      int
	queryConfidence string
	queryStatus     string
	querySource     string
	queryCitation   string
	querySubject    string
	queryBirthplace string
)

var queryCmd = &cobra.Command{
	Use:   "query <entity-type>",
	Short: "Query entities in a GLX archive",
	Long: `Filter and list entities from a GENEALOGIX archive.

Supported entity types: persons, events, assertions, sources,
relationships, places, citations, repositories, media.

Filters vary by entity type:
  persons:       --name, --born-before, --born-after, --birthplace
  events:        --type, --before, --after
  assertions:    --confidence, --status, --source, --citation, --subject
  sources:       --name, --type
  relationships: --type
  places:        --name
  repositories:  --name

All entity types support --archive to specify the archive path.`,
	Example: `  # Find persons born before 1850
  glx query persons --born-before 1850

  # Find low-confidence assertions
  glx query assertions --confidence low

  # Find assertions citing a specific source
  glx query assertions --source source-abc123

  # Find assertions using a specific citation
  glx query assertions --citation citation-abc123

  # Find marriage events
  glx query events --type marriage

  # Find persons by name in a specific archive
  glx query persons --name "Smith" --archive my-archive

  # List all sources
  glx query sources`,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: queryEntityTypes,
	RunE:      runQuery,
}

func init() {
	queryCmd.Flags().StringVarP(&queryArchive, "archive", "a", ".", "Archive path (directory or single file)")
	queryCmd.Flags().StringVar(&queryName, "name", "", "Filter by name (substring match, case-insensitive)")
	queryCmd.Flags().IntVar(&queryBornBefore, "born-before", 0, "Filter persons born before this year")
	queryCmd.Flags().IntVar(&queryBornAfter, "born-after", 0, "Filter persons born after this year")
	queryCmd.Flags().StringVar(&queryType, "type", "", "Filter by type (event type, relationship type, etc.)")
	queryCmd.Flags().IntVar(&queryBefore, "before", 0, "Filter events with date before this year")
	queryCmd.Flags().IntVar(&queryAfter, "after", 0, "Filter events with date after this year")
	queryCmd.Flags().StringVar(&queryConfidence, "confidence", "", "Filter assertions by confidence level")
	queryCmd.Flags().StringVar(&queryStatus, "status", "", "Filter assertions by status")
	queryCmd.Flags().StringVar(&querySource, "source", "", "Filter assertions by source ID (direct or via citation)")
	queryCmd.Flags().StringVar(&queryCitation, "citation", "", "Filter assertions by citation ID")
	queryCmd.Flags().StringVar(&querySubject, "subject", "", "Filter assertions by subject entity ID or person name substring")
	queryCmd.Flags().StringVar(&queryBirthplace, "birthplace", "", "Filter persons by birthplace (place ID or name substring)")
}

func runQuery(_ *cobra.Command, args []string) error {
	return queryEntities(args[0], queryOpts{
		Archive:    queryArchive,
		Name:       queryName,
		BornBefore: queryBornBefore,
		BornAfter:  queryBornAfter,
		Type:       queryType,
		Before:     queryBefore,
		After:      queryAfter,
		Confidence: queryConfidence,
		Status:     queryStatus,
		Source:     querySource,
		Citation:   queryCitation,
		Subject:    querySubject,
		Birthplace: queryBirthplace,
	})
}

// ============================================================================
// Stats Command
// ============================================================================

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Show summary statistics for a GLX archive",
	Long: `Display a summary dashboard of a GENEALOGIX archive.

Shows entity counts, assertion confidence distribution, and entity coverage
metrics for quick feedback on archive health.

Accepts either a multi-file directory or a single .glx file.
If no path is given, uses the current directory.`,
	Example: `  # Stats for current directory
  glx stats

  # Stats for a specific archive directory
  glx stats my-family-archive

  # Stats for a single-file archive
  glx stats family.glx`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStats,
}

func runStats(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	return showStats(path)
}

// ============================================================================
// Cite Command
// ============================================================================

var citeArchive string

var citeCmd = &cobra.Command{
	Use:   "cite [citation-id]",
	Short: "Generate formatted citation text from structured fields",
	Long: `Generate a formatted citation string from structured citation data.

Assembles citations from the source title, source type, repository name,
URL, and accessed date already stored in the archive. This eliminates
repetitive manual writing of the citation_text property.

If a citation ID is given, prints that single citation. If no ID is given,
prints all citations in the archive.`,
	Example: `  # Format a specific citation
  glx cite citation-1860-census-webb-household

  # Format all citations in the archive
  glx cite

  # Use a specific archive
  glx cite --archive my-archive`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCite,
}

func init() {
	citeCmd.Flags().StringVarP(&citeArchive, "archive", "a", ".", "Archive path (directory or single file)")
}

func runCite(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		return showCitation(citeArchive, args[0])
	}

	return showAllCitations(citeArchive)
}

// Ancestors Command
// ============================================================================

var (
	ancestorsArchive string
	ancestorsMaxGen  int
)

var ancestorsCmd = &cobra.Command{
	Use:   "ancestors <person-id>",
	Short: "Show ancestor tree for a person",
	Long: `Display the ancestor tree for a person by traversing parent-child relationships.

Walks up parent-child relationships, including biological, adoptive,
foster, and step-parent variants, to build and display the full
ancestor tree.

Non-default parent/child relationship types (biological, adoptive, foster, step)
are annotated in the output when the relationship type is not parent_child.`,
	Example: `  # Show ancestors
  glx ancestors person-abc123

  # Limit to 3 generations
  glx ancestors person-abc123 --generations 3

  # Use a specific archive
  glx ancestors person-abc123 --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runAncestors,
}

func init() {
	ancestorsCmd.Flags().StringVarP(&ancestorsArchive, "archive", "a", ".", "Archive path (directory or single file)")
	ancestorsCmd.Flags().IntVarP(&ancestorsMaxGen, "generations", "g", 0, "Maximum number of generations (0 for unlimited)")
}

func runAncestors(_ *cobra.Command, args []string) error {
	return showAncestors(ancestorsArchive, args[0], ancestorsMaxGen)
}

// ============================================================================
// Descendants Command
// ============================================================================

var (
	descendantsArchive string
	descendantsMaxGen  int
)

var descendantsCmd = &cobra.Command{
	Use:   "descendants <person-id>",
	Short: "Show descendant tree for a person",
	Long: `Display the descendant tree for a person by traversing parent-child relationships.

Walks down parent-child relationships, including biological, adoptive,
foster, and step-parent variants, to build and display the full
descendant tree.

Non-default parent/child relationship types (biological, adoptive, foster, step)
are annotated in the output when the relationship type is not parent_child.`,
	Example: `  # Show descendants
  glx descendants person-abc123

  # Limit to 3 generations
  glx descendants person-abc123 --generations 3

  # Use a specific archive
  glx descendants person-abc123 --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runDescendants,
}

func init() {
	descendantsCmd.Flags().StringVarP(&descendantsArchive, "archive", "a", ".", "Archive path (directory or single file)")
	descendantsCmd.Flags().IntVarP(&descendantsMaxGen, "generations", "g", 0, "Maximum number of generations (0 for unlimited)")
}

func runDescendants(_ *cobra.Command, args []string) error {
	return showDescendants(descendantsArchive, args[0], descendantsMaxGen)
}

// ============================================================================
// Summary Command
// ============================================================================

var summaryArchive string

var summaryCmd = &cobra.Command{
	Use:   "summary <person>",
	Short: "Show a comprehensive profile for a person",
	Long: `Display a full summary of a person including identity, vital events,
life events, family relationships, other relationships, and an
auto-generated life history narrative.

The person can be specified by exact ID (e.g., "person-abc123") or by
name substring (case-insensitive). If multiple persons match, all
matches are listed for disambiguation.

Sections displayed:
  - Identity: name, sex, alternate names (birth, married, maiden, AKA, etc.)
  - Vital Events: birth, christening, death, burial
  - Life Events: census, immigration, naturalization, military service, etc.
  - Family: spouse(s) with marriage info, parents, siblings
  - Relationships: godparent, neighbor, household, employment, etc.
  - Life History: auto-generated biographical narrative`,
	Example: `  # Summary by person ID
  glx summary person-abc123

  # Summary by name search
  glx summary "Jane Webb"

  # Summary in a specific archive
  glx summary "John Smith" --archive my-family-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runSummary,
}

func init() {
	summaryCmd.Flags().StringVarP(&summaryArchive, "archive", "a", ".", "Archive path (directory or single file)")
}

func runSummary(_ *cobra.Command, args []string) error {
	return showSummary(summaryArchive, args[0])
}

// ============================================================================
// Timeline Command
// ============================================================================

var (
	timelineArchive  string
	timelineNoFamily bool
)

var timelineCmd = &cobra.Command{
	Use:   "timeline <person>",
	Short: "Show chronological timeline of events for a person",
	Long: `Display a chronological timeline of all events in a person's life.

Shows direct events (where the person is a participant) and family events
(spouse births/deaths, children's births/deaths, parent deaths) discovered
through relationship traversal.

The person argument can be an exact entity ID (e.g., person-john-smith)
or a name to search for (e.g., "John Smith"). If the name matches
multiple persons, all matches are listed for disambiguation.

Use --no-family to exclude family events and show only direct events.`,
	Example: `  # Timeline by person ID
  glx timeline person-john-smith

  # Timeline by name
  glx timeline "John Smith"

  # Direct events only (no family events)
  glx timeline "John Smith" --no-family

  # Specify archive path
  glx timeline "John Smith" --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runTimeline,
}

func init() {
	timelineCmd.Flags().StringVarP(&timelineArchive, "archive", "a", ".", "Archive path (directory or single file)")
	timelineCmd.Flags().BoolVar(&timelineNoFamily, "no-family", false, "Exclude family events (show only direct events)")
}

func runTimeline(_ *cobra.Command, args []string) error {
	return showTimeline(timelineArchive, args[0], !timelineNoFamily)
}

// ============================================================================
// Vitals Command
// ============================================================================

var vitalsArchive string

var vitalsCmd = &cobra.Command{
	Use:   "vitals <person>",
	Short: "Show vital records for a person",
	Long: `Display vital records for a person in the archive.

Shows: Name, Sex, Birth, Christening, Death, Burial, plus any other
life events the person participated in (marriages, census records, etc.).

The person argument can be an exact entity ID (e.g., person-robert-webb) or
a name to search for (e.g., "Jane Miller"). If the name matches multiple
persons, all matches are listed for disambiguation.`,
	Example: `  # Look up by person ID
  glx vitals person-robert-webb

  # Look up by name
  glx vitals "Jane Miller"

  # Specify archive path
  glx vitals "Jane Miller" --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runVitals,
}

func init() {
	vitalsCmd.Flags().StringVarP(&vitalsArchive, "archive", "a", ".", "Archive path (directory or single file)")
}

func runVitals(_ *cobra.Command, args []string) error {
	return showVitals(vitalsArchive, args[0])
}

// ============================================================================
// Census Command
// ============================================================================

var censusCmd = &cobra.Command{
	Use:   "census",
	Short: "Bulk census record tools",
	Long: `Tools for working with census records in a GENEALOGIX archive.

Subcommands:
  add    Import a census template into the archive`,
}

var (
	censusAddFrom    string
	censusAddArchive string
	censusAddDryRun  bool
	censusAddVerbose bool
)

var censusAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Import a census template into the archive",
	Long: `Generate GLX entities from a structured census template file.

Reads a YAML census template and generates:
- Person records (new or matched to existing)
- A census event with participants
- A source (new or matched to existing)
- A citation with locator, URL, transcription
- Assertions for birth year, birthplace, gender, occupation, residence

The template format uses a simple YAML structure describing the census
year, location, household members, and citation details. Members can
reference existing persons by ID or by name (matched against the archive).

Use --dry-run to preview what would be generated without writing files.`,
	Example: `  # Import a census template
  glx census add --from 1860-census-lane.yaml --archive my-archive

  # Preview without writing
  glx census add --from 1860-census-lane.yaml --archive my-archive --dry-run

  # Verbose output
  glx census add --from 1860-census-lane.yaml --archive my-archive --verbose`,
	RunE: runCensusAdd,
}

func init() {
	censusCmd.AddCommand(censusAddCmd)

	censusAddCmd.Flags().StringVar(&censusAddFrom, "from", "", "Path to census template YAML file (required)")
	censusAddCmd.Flags().StringVarP(&censusAddArchive, "archive", "a", ".", "Archive path (directory)")
	censusAddCmd.Flags().BoolVar(&censusAddDryRun, "dry-run", false, "Preview generated entities without writing files")
	censusAddCmd.Flags().BoolVarP(&censusAddVerbose, "verbose", "v", false, "Show detailed summary of generated entities")

	_ = censusAddCmd.MarkFlagRequired("from")
}

func runCensusAdd(_ *cobra.Command, _ []string) error {
	return censusAdd(censusAddFrom, censusAddArchive, censusAddDryRun, censusAddVerbose)
}

// ============================================================================
// Cluster Command (FAN Club Analysis)
// ============================================================================

var (
	clusterArchive string
	clusterPlace   string
	clusterBefore  int
	clusterAfter   int
	clusterJSON    bool
)

var clusterCmd = &cobra.Command{
	Use:   "cluster <person>",
	Short: "FAN club analysis — find associates of a person",
	Long: `Identify associates of a person using FAN (Friends, Associates, Neighbors)
club analysis — the primary methodology for breaking genealogical brickwalls.

Cross-references the archive to find people connected to the target through:
- Census households: people enumerated in the same census events
- Shared events: co-participants in marriages, baptisms, land records, etc.
- Place overlap: people associated with the same places in the same time period

Associates are ranked by connection strength: census household links (3 points),
shared event links (2 points), and place overlap links (1 point). Multiple
connections compound for higher scores.

The person argument can be an exact entity ID (e.g., person-d-lane) or a
name to search for (e.g., "Mary Green"). If the name matches multiple
persons, all matches are listed for disambiguation.`,
	Example: `  # Show all associates
  glx cluster person-mary-lane

  # Filter to a specific place
  glx cluster person-mary-lane --place place-ironton-sauk-wi

  # Filter to a time range
  glx cluster person-mary-lane --before 1860 --after 1840

  # JSON output for downstream tooling
  glx cluster person-mary-lane --json

  # Use a specific archive
  glx cluster "Mary Green" --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runCluster,
}

func init() {
	clusterCmd.Flags().StringVarP(&clusterArchive, "archive", "a", ".", "Archive path (directory or single file)")
	clusterCmd.Flags().StringVar(&clusterPlace, "place", "", "Filter by place ID (includes descendant places)")
	clusterCmd.Flags().IntVar(&clusterBefore, "before", 0, "Only consider dated events before this year (undated events are still included)")
	clusterCmd.Flags().IntVar(&clusterAfter, "after", 0, "Only consider dated events after this year (undated events are still included)")
	clusterCmd.Flags().BoolVar(&clusterJSON, "json", false, "Output as JSON")
}

func runCluster(_ *cobra.Command, args []string) error {
	return showCluster(clusterArchive, args[0], clusterPlace, clusterBefore, clusterAfter, clusterJSON)
}

// ============================================================================
// Path Command
// ============================================================================

var (
	pathArchive string
	pathMaxHops int
	pathJSON    bool
)

var pathCmd = &cobra.Command{
	Use:   "path <person-a> <person-b>",
	Short: "Find the shortest relationship path between two people",
	Long: `Find and display the shortest relationship path between two persons
in the archive using breadth-first search.

Traverses all relationship types (parent-child, marriage, sibling,
godparent, neighbor, etc.) to find the shortest connection.

Each hop shows the relationship type and the destination person's role.
Use --max-hops to limit search depth (default 10).

Person arguments can be exact entity IDs or name substrings.`,
	Example: `  # Find path between two persons by ID
  glx path person-mary-lane person-louenza-mortimer

  # Find path by name
  glx path "Mary Lane" "Louenza Mortimer"

  # Limit search depth
  glx path "Mary Lane" "John Smith" --max-hops 5

  # JSON output
  glx path "Mary Lane" "John Smith" --json

  # Specify archive path
  glx path "Mary Lane" "John Smith" --archive my-archive`,
	Args: cobra.ExactArgs(2),
	RunE: runPath,
}

func init() {
	pathCmd.Flags().StringVarP(&pathArchive, "archive", "a", ".", "Archive path (directory or single file)")
	pathCmd.Flags().IntVar(&pathMaxHops, "max-hops", 10, "Maximum number of hops to search")
	pathCmd.Flags().BoolVar(&pathJSON, "json", false, "Output as JSON")
}

func runPath(_ *cobra.Command, args []string) error {
	return showPath(pathArchive, args[0], args[1], pathMaxHops, pathJSON)
}

// ============================================================================
// Duplicates Command
// ============================================================================

var (
	duplicatesArchive   string
	duplicatesThreshold float64
	duplicatesJSON      bool
)

var duplicatesCmd = &cobra.Command{
	Use:   "duplicates [person]",
	Short: "Detect potential duplicate persons in a GLX archive",
	Long: `Scan a GLX archive for potential duplicate person records.

Compares all persons using a weighted scoring model based on:
  - Name similarity (Levenshtein distance, nickname matching, initials)
  - Birth/death year proximity
  - Birth/death place match
  - Shared relationships and events

Persons already linked by a direct relationship (parent-child, spouse, etc.)
are automatically skipped since they are known to be distinct individuals.

Use --threshold to adjust sensitivity (0.0-1.0, default 0.60).
Higher values = fewer, higher-confidence matches.`,
	Example: `  # Scan for duplicates in current directory
  glx duplicates

  # Scan with higher confidence threshold
  glx duplicates --threshold 0.8

  # Check a specific person for duplicates
  glx duplicates person-robert-webb

  # JSON output for tooling
  glx duplicates --json

  # Scan a specific archive
  glx duplicates --archive my-family-archive`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDuplicates,
}

func init() {
	duplicatesCmd.Flags().StringVarP(&duplicatesArchive, "archive", "a", ".", "Archive path (directory or single file)")
	duplicatesCmd.Flags().Float64Var(&duplicatesThreshold, "threshold", 0.60, "Minimum similarity score (0.0-1.0)")
	duplicatesCmd.Flags().BoolVar(&duplicatesJSON, "json", false, "JSON output")
}

func runDuplicates(_ *cobra.Command, args []string) error {
	personFilter := ""
	if len(args) == 1 {
		personFilter = args[0]
	}
	return findDuplicates(duplicatesArchive, duplicatesThreshold, personFilter, duplicatesJSON)
}

// ============================================================================
// Coverage Command
// ============================================================================

var (
	coverageArchive string
	coverageJSON    bool
)

var coverageCmd = &cobra.Command{
	Use:   "coverage <person>",
	Short: "Show source coverage matrix for a person",
	Long: `Display a checklist of expected records for a person and which ones are present.

Generates a source coverage matrix based on the person's birth and death years,
showing which census records, vital records, and other documents have been found
versus which are still missing.

Record categories:
  - Census: US federal census records the person should appear in
  - Vital: Birth, death, and marriage records
  - Other: Probate, land, military, and church records

Missing high-priority records are flagged to guide research efforts.

The person argument can be an exact entity ID or a name substring.`,
	Example: `  # Coverage by person ID
  glx coverage person-abc123

  # Coverage by name
  glx coverage "Jane Miller"

  # JSON output
  glx coverage "Jane Miller" --json

  # Specify archive path
  glx coverage "Jane Miller" --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runCoverage,
}

func init() {
	coverageCmd.Flags().StringVarP(&coverageArchive, "archive", "a", ".", "Archive path (directory or single file)")
	coverageCmd.Flags().BoolVar(&coverageJSON, "json", false, "Output as JSON")
}

func runCoverage(_ *cobra.Command, args []string) error {
	return showCoverage(coverageArchive, args[0], coverageJSON)
}

// ============================================================================
// Analyze Command
// ============================================================================

var (
	analyzeArchive string
	analyzeCheck   string
	analyzeFormat  string
	analyzePerson  string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [person]",
	Short: "Analyze archive for research gaps, evidence quality, and consistency",
	Long: `Run automated analysis on a GENEALOGIX archive to surface research gaps,
unsupported claims, chronological inconsistencies, and suggested next steps.

Analysis categories:
  gaps          Missing data that should be findable (no birth, no parents, etc.)
  evidence      Unsupported or weakly supported claims (no citations, single source)
  consistency   Chronological cross-checks (death before birth, implausible lifespan)
  suggestions   Research recommendations (census years to search, vital records)

Use --check to run a single category. By default, all categories are analyzed.

Use --format json for machine-readable output.`,
	Example: `  # Full analysis of current directory
  glx analyze

  # Focus on one person
  glx analyze person-jane-webb

  # Run only gap analysis
  glx analyze --check gaps

  # JSON output for tooling
  glx analyze --format json

  # Analyze a specific archive
  glx analyze --archive my-archive`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVarP(&analyzeArchive, "archive", "a", ".", "Archive path (directory or single file)")
	analyzeCmd.Flags().StringVarP(&analyzeCheck, "check", "c", "", "Run a single analysis category (gaps, evidence, consistency, suggestions)")
	analyzeCmd.Flags().StringVarP(&analyzeFormat, "format", "f", "", "Output format (json for machine-readable)")
	analyzeCmd.Flags().StringVarP(&analyzePerson, "person", "p", "", "Filter results to a specific person (ID or name)")
}

func runAnalyze(_ *cobra.Command, args []string) error {
	person := analyzePerson
	if len(args) == 1 {
		person = args[0]
	}
	return showAnalysis(analyzeArchive, person, analyzeCheck, analyzeFormat)
}

// ============================================================================
// Diff Command
// ============================================================================

var (
	diffVerbose bool
	diffShort   bool
	diffJSON    bool
	diffPerson  string
)

var diffCmd = &cobra.Command{
	Use:   "diff <dir1> <dir2>",
	Short: "Compare two GLX archive states",
	Long: `Compare two GLX archive states and show genealogy-aware differences.

Summarizes changes in terms of entities and evidence rather than raw YAML lines.
Shows added, modified, and removed entities along with specific field changes,
confidence upgrades/downgrades, and new evidence.

Output modes:
  (default)  Summary table grouped by entity type
  --verbose  Full field-level details for all modified entities
  --short    Single-line compact summary
  --json     Machine-readable JSON output

Use --person to filter changes relevant to a specific person.`,
	Example: `  # Compare two archive directories
  glx diff ./archive-v1 ./archive-v2

  # Verbose field-level details
  glx diff ./old ./new --verbose

  # Compact one-liner
  glx diff ./old ./new --short

  # JSON output for tooling
  glx diff ./old ./new --json

  # Filter changes for a specific person
  glx diff ./old ./new --person person-jane-webb`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().BoolVarP(&diffVerbose, "verbose", "v", false, "Show full field-level details")
	diffCmd.Flags().BoolVar(&diffShort, "short", false, "Compact single-line output")
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "JSON output")
	diffCmd.Flags().StringVar(&diffPerson, "person", "", "Filter changes for a specific person ID")
}

func runDiff(_ *cobra.Command, args []string) error {
	return diffArchives(args[0], args[1], diffPerson, diffVerbose, diffShort, diffJSON)
}

// ============================================================================
// Rename Command
// ============================================================================

var renameArchive string

var renameCmd = &cobra.Command{
	Use:   "rename <old-id> <new-id>",
	Short: "Rename an entity ID and update all references",
	Long: `Rename an entity ID throughout the archive, atomically updating all
cross-references in events, relationships, assertions, citations, and other entities.

Works with any entity type: persons, events, relationships, places, sources,
citations, repositories, assertions, and media.`,
	Example: `  # Rename a person
  glx rename person-a3f8d2c1 person-jane-miller --archive ./archive

  # Rename a place
  glx rename place-b7e2f1a0 place-millbrook-hartford --archive ./archive

  # Preview changes without writing
  glx rename person-old person-new --archive ./archive --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runRename,
}

func init() {
	renameCmd.Flags().StringVarP(&renameArchive, "archive", "a", ".", "Path to GLX archive")
	renameCmd.Flags().BoolVar(&renameDryRun, "dry-run", false, "Show what would change without writing")
}

func runRename(_ *cobra.Command, args []string) error {
	return renameEntities(renameArchive, args[0], args[1], renameDryRun)
}

// ============================================================================
// Merge Command
// ============================================================================

var (
	mergeInto   string
	mergeDryRun bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge <source>",
	Short: "Merge another archive into the destination archive",
	Long: `Combine two GLX archives by merging all content from the source
into the destination. Duplicate or conflicting items (entities,
vocabularies, property definitions, and metadata) are reported and skipped
(the destination version is kept).`,
	Example: `  # Merge another archive into the current one
  glx merge ./other-archive/ --into ./my-archive/

  # Dry run to preview what would be merged
  glx merge ./other-archive/ --into ./my-archive/ --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runMerge,
}

func init() {
	mergeCmd.Flags().StringVar(&mergeInto, "into", ".", "Destination archive path")
	mergeCmd.Flags().BoolVar(&mergeDryRun, "dry-run", false, "Preview merge without writing")
}

func runMerge(_ *cobra.Command, args []string) error {
	return mergeArchives(args[0], mergeInto, mergeDryRun)
}
