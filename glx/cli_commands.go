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

// Root command
var rootCmd = &cobra.Command{
	Use:   "glx",
	Short: "GENEALOGIX CLI - Manage and validate genealogy archives",
	Long: `GLX is the official command-line tool for working with GENEALOGIX family archives.

GENEALOGIX is a modern, evidence-first, Git-native genealogy data standard.
Use GLX to initialize new archives, validate files, and ensure data quality.`,
	Version:       "0.0.0-beta.6",
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
	rootCmd.AddCommand(vitalsCmd)
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
)

var queryCmd = &cobra.Command{
	Use:   "query <entity-type>",
	Short: "Query entities in a GLX archive",
	Long: `Filter and list entities from a GENEALOGIX archive.

Supported entity types: persons, events, assertions, sources,
relationships, places, citations, repositories, media.

Filters vary by entity type:
  persons:       --name, --born-before, --born-after
  events:        --type, --before, --after
  assertions:    --confidence, --status, --source, --citation
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
// Vitals Command
// ============================================================================

var vitalsArchive string

var vitalsCmd = &cobra.Command{
	Use:   "vitals <person>",
	Short: "Show vital records for a person",
	Long: `Display vital records for a person in the archive.

Shows: Name, Sex, Birth, Christening, Death, Burial, plus any other
life events the person participated in (marriages, census records, etc.).

The person argument can be an exact entity ID (e.g., person-d-lane) or
a name to search for (e.g., "Mary Green"). If the name matches multiple
persons, all matches are listed for disambiguation.`,
	Example: `  # Look up by person ID
  glx vitals person-d-lane

  # Look up by name
  glx vitals "Mary Green"

  # Specify archive path
  glx vitals "Mary Green" --archive my-archive`,
	Args: cobra.ExactArgs(1),
	RunE: runVitals,
}

func init() {
	vitalsCmd.Flags().StringVarP(&vitalsArchive, "archive", "a", ".", "Archive path (directory or single file)")
}

func runVitals(_ *cobra.Command, args []string) error {
	return showVitals(vitalsArchive, args[0])
}
