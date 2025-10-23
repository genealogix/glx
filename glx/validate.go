package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

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
				if filepath.Ext(d.Name()) != ".glx" {
					return nil
				}
				checked++
				if issues := validateGLXFileFromPath(path); len(issues) > 0 {
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

		if filepath.Ext(root) != ".glx" {
			reportLines = append(reportLines, fmt.Sprintf("✗ %s is not a .glx file", root))
			hadError = true
			continue
		}

		checked++
		if issues := validateGLXFileFromPath(root); len(issues) > 0 {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(root, issues)...)
		} else {
			reportLines = append(reportLines, fmt.Sprintf("✓ %s", root))
		}
	}

	if checked == 0 {
		return errors.New("no .glx files found to validate")
	}

	for _, line := range reportLines {
		fmt.Println(line)
	}

	if hadError {
		return errors.New("validation failed")
	}

	fmt.Printf("Validated %d file(s)\n", checked)
	return nil
}

func validateGLXFileFromPath(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("read error: %v", err)}
	}

	doc, err := ParseYAMLFile(data)
	if err != nil {
		return []string{fmt.Sprintf("YAML parse error: %v", err)}
	}

	return ValidateGLXFile(path, doc)
}

func formatValidationIssues(path string, issues []string) []string {
	lines := []string{fmt.Sprintf("✗ %s", path)}
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("  - %s", issue))
	}
	return lines
}
