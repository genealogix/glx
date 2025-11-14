package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate GLX files and cross-references",
	Long: `Validate GENEALOGIX (.glx) files for correctness and integrity.

Performs comprehensive validation including:
- YAML syntax correctness
- Required fields presence
- Entity ID format validation
- Cross-reference integrity
- Duplicate ID detection
- Vocabulary validation (if vocabularies/ exists)

When validating directories, automatically checks all .glx, .yaml, and .yml files
and validates cross-references between entities.`,
	Example: `  # Validate current directory
  glx validate

  # Validate specific directory
  glx validate persons/

  # Validate multiple paths
  glx validate persons/ events/ places/

  # Validate single file
  glx validate archive.glx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var (
		checked     int
		hadError    bool
		reportLines []string
	)

	// Load vocabularies for validation
	vocabs, err := LoadArchiveVocabularies(".")
	if err != nil {
		reportLines = append(reportLines, fmt.Sprintf("✗ Failed to load vocabularies: %v", err))
		hadError = true
	}

	// First pass: validate each file individually
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			reportLines = append(reportLines, fmt.Sprintf("✗ %s (stat error: %v)", root, err))
			hadError = true
			continue
		}

		if info.IsDir() {
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					reportLines = append(reportLines, fmt.Sprintf("✗ %s (walk error: %v)", path, walkErr))
					hadError = true
					return nil
				}
				if d.IsDir() {
					return nil
				}
				ext := filepath.Ext(d.Name())
				if ext != ".glx" && ext != ".yaml" && ext != ".yml" {
					return nil
				}
				checked++
				if issues := validateGLXFileFromPath(path, vocabs); len(issues) > 0 {
					hadError = true
					reportLines = append(reportLines, formatValidationIssues(path, issues)...)
				} else {
					reportLines = append(reportLines, fmt.Sprintf("✓ %s", path))
				}
				return nil
			})
			if err != nil {
				reportLines = append(reportLines, fmt.Sprintf("✗ %s (walk error: %v)", root, err))
				hadError = true
			}
			continue
		}

		ext := filepath.Ext(root)
		if ext != ".glx" && ext != ".yaml" && ext != ".yml" {
			reportLines = append(reportLines, fmt.Sprintf("✗ %s is not a .glx/.yaml file", root))
			hadError = true
			continue
		}

		checked++
		if issues := validateGLXFileFromPath(root, vocabs); len(issues) > 0 {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(root, issues)...)
		} else {
			reportLines = append(reportLines, fmt.Sprintf("✓ %s", root))
		}
	}

	if checked == 0 {
		return errors.New("no .glx files found to validate")
	}

	// Second pass: validate cross-references across all files in repository
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			continue
		}

		// If validating a directory, check for cross-reference issues
		if info.IsDir() {
			allEntities, duplicates, err := CollectAllEntities(root)
			if err != nil {
				reportLines = append(reportLines, fmt.Sprintf("✗ Cross-reference validation error: %v", err))
				hadError = true
			}

			// Report duplicate IDs
			if len(duplicates) > 0 {
				hadError = true
				reportLines = append(reportLines, "")
				reportLines = append(reportLines, "Cross-reference issues:")
				for _, dup := range duplicates {
					reportLines = append(reportLines, fmt.Sprintf("  ✗ %s", dup))
				}
			}

			// Check all references
			refIssues := ValidateRepositoryReferences(root, allEntities)
			if len(refIssues) > 0 {
				hadError = true
				if len(duplicates) == 0 {
					reportLines = append(reportLines, "")
					reportLines = append(reportLines, "Cross-reference issues:")
				}
				for _, issue := range refIssues {
					reportLines = append(reportLines, fmt.Sprintf("  ✗ %s", issue))
				}
			}
		}
	}

	// Print all results
	for _, line := range reportLines {
		fmt.Println(line)
	}

	if hadError {
		return errors.New("validation failed")
	}

	fmt.Printf("\nValidated %d file(s)\n", checked)
	return nil
}

func validateGLXFileFromPath(path string, vocabs *ArchiveVocabularies) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("read error: %v", err)}
	}

	doc, err := ParseYAMLFile(data)
	if err != nil {
		return []string{fmt.Sprintf("YAML parse error: %v", err)}
	}

	return ValidateGLXFile(path, doc, vocabs)
}

func formatValidationIssues(path string, issues []string) []string {
	lines := []string{fmt.Sprintf("✗ %s", path)}
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("  - %s", issue))
	}
	return lines
}
