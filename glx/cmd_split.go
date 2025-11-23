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

	"github.com/genealogix/glx/glx/lib"
	"github.com/spf13/cobra"
)

var (
	splitNoValidate     bool
	splitNoVocabularies bool
	splitVerbose        bool
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

Each entity file includes an _id field to preserve the entity ID.`,
	Example: `  # Split an archive
  glx split family.glx family-archive

  # Split without vocabularies
  glx split family.glx family-archive --no-vocabularies

  # Split without validation
  glx split family.glx family-archive --no-validate`,
	Args: cobra.ExactArgs(2),
	RunE: runSplit,
}

func init() {
	rootCmd.AddCommand(splitCmd)

	splitCmd.Flags().BoolVar(&splitNoValidate, "no-validate", false, "Skip validation before splitting")
	splitCmd.Flags().BoolVar(&splitNoVocabularies, "no-vocabularies", false, "Don't include standard vocabularies")
	splitCmd.Flags().BoolVarP(&splitVerbose, "verbose", "v", false, "Verbose output")
}

func runSplit(_ *cobra.Command, args []string) error {
	return splitArchive(args[0], args[1])
}

func splitArchive(inputPath, outputDir string) error {
	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Check if output directory exists
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		return fmt.Errorf("output directory already exists: %s (please remove it first)", outputDir)
	}

	// Load single-file archive
	if splitVerbose {
		fmt.Printf("Loading archive: %s\n", inputPath)
	}

	loadOpts := &lib.SerializerOptions{
		Validate: !splitNoValidate,
	}
	serializer := lib.NewSerializer(loadOpts)

	glx, err := serializer.LoadSingleFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to load archive: %w", err)
	}

	if splitVerbose {
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

	// Serialize to multi-file format
	if splitVerbose {
		fmt.Printf("\nWriting multi-file archive: %s\n", outputDir)
	}

	saveOpts := &lib.SerializerOptions{
		IncludeVocabularies: !splitNoVocabularies,
		Validate:            false, // Already validated on load
		Pretty:              true,
	}
	saveSerializer := lib.NewSerializer(saveOpts)

	if err := saveSerializer.SerializeMultiFile(glx, outputDir); err != nil {
		return fmt.Errorf("failed to write multi-file archive: %w", err)
	}

	fmt.Printf("✓ Successfully split archive to %s/\n", outputDir)

	if !splitNoVocabularies {
		fmt.Println("  Standard vocabularies included")
	}

	return nil
}
