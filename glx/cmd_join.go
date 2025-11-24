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
	"path/filepath"
	"strings"

	"github.com/genealogix/glx/glx/lib"
	"github.com/spf13/cobra"
)

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

Entity IDs are restored from the _id field in each file.`,
	Example: `  # Join an archive
  glx join family-archive family.glx

  # Join without validation
  glx join family-archive family.glx --no-validate`,
	Args: cobra.ExactArgs(2),
	RunE: runJoin,
}

func init() {
	rootCmd.AddCommand(joinCmd)

	joinCmd.Flags().BoolVar(&joinNoValidate, "no-validate", false, "Skip validation before joining")
	joinCmd.Flags().BoolVarP(&joinVerbose, "verbose", "v", false, "Verbose output")
	joinCmd.Flags().IntVar(&joinShowFirstErrors, "show-first-errors", defaultShowFirstErrors, "Number of validation errors to show (0 for all)")
}

func runJoin(_ *cobra.Command, args []string) error {
	return joinArchive(args[0], args[1])
}

func joinArchive(inputDir, outputPath string) error {
	// Check if input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrInputDirectoryNotFound, inputDir)
	}

	// Check if output file exists
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrOutputFileExists, outputPath)
	}

	// Ensure .glx extension
	if !strings.HasSuffix(outputPath, ".glx") {
		outputPath += ".glx"
	}

	// Load multi-file archive
	if joinVerbose {
		fmt.Printf("Loading multi-file archive: %s\n", inputDir)
	}

	// Read all files from directory
	files := make(map[string][]byte)
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Read file content
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Get relative path from inputDir
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		files[relPath] = data

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	loadOpts := &lib.SerializerOptions{
		Validate: !joinNoValidate,
	}
	serializer := lib.NewSerializer(loadOpts)

	glx, err := serializer.DeserializeMultiFileFromMap(files)
	if err != nil {
		return fmt.Errorf("failed to load multi-file archive: %w", err)
	}

	if joinVerbose {
		fmt.Println("Archive loaded successfully")
		fmt.Printf("  Persons:       %d\n", len(glx.Persons))
		fmt.Printf("  Events:        %d\n", len(glx.Events))
		fmt.Printf("  Relationships: %d\n", len(glx.Relationships))
		fmt.Printf("  Places:        %d\n", len(glx.Places))
		fmt.Printf("  Sources:       %d\n", len(glx.Sources))
		fmt.Printf("  Citations:     %d\n", len(glx.Citations))
		fmt.Printf("  Repositories:  %d\n", len(glx.Repositories))
		fmt.Printf("  Media:         %d\n", len(glx.Media))
		fmt.Printf("  Assertions:    %d\n", len(glx.Assertions))
	}

	// Serialize to single-file format
	if joinVerbose {
		fmt.Printf("\nWriting single-file archive: %s\n", outputPath)
	}

	saveOpts := &lib.SerializerOptions{
		Validate: false, // Already validated on load
		Pretty:   true,
	}
	saveSerializer := lib.NewSerializer(saveOpts)

	yamlBytes, err := saveSerializer.SerializeSingleFileBytes(glx)
	if err != nil {
		return fmt.Errorf("failed to serialize single-file archive: %w", formatValidationError(err, joinShowFirstErrors))
	}

	if err := os.WriteFile(outputPath, yamlBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write single-file archive: %w", err)
	}

	fmt.Printf("✓ Successfully joined archive to %s\n", outputPath)

	return nil
}
