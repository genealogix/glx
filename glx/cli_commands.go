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
	Version:       "0.0.0-beta.3",
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
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(splitCmd)
	rootCmd.AddCommand(joinCmd)

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
- No arguments: Validates current directory with full cross-reference validation`,
	Example: `  # Validate current directory (with cross-reference checks)
  glx validate

  # Validate specific directory (with cross-reference checks)
  glx validate persons/

  # Validate multiple paths (with cross-reference checks)
  glx validate persons/ events/ places/

  # Validate single file (structure only, no cross-reference checks)
  glx validate archive.glx`,
	RunE: runValidate,
}

func runValidate(_ *cobra.Command, args []string) error {
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

