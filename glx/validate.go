package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
				if issues := validateGLXFile(path); len(issues) > 0 {
					hadError = true
					reportLines = append(reportLines, formatValidationIssues(path, issues)...) // nolint:makezero
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
		if issues := validateGLXFile(root); len(issues) > 0 {
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

func formatValidationIssues(path string, issues []string) []string {
	lines := []string{fmt.Sprintf("✗ %s", path)}
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("  - %s", issue))
	}
	return lines
}

func validateGLXFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("read error: %v", err)}
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return []string{fmt.Sprintf("YAML parse error: %v", err)}
	}

	typ := detectGLXType(path)
	var issues []string

	switch typ {
	case "config":
		issues = append(issues, validateStringField(doc, "version")...)
		issues = append(issues, validateStringField(doc, "schema")...)
	case "schema-version":
		issues = append(issues, validateStringField(doc, "schema")...)
	default:
		issues = append(issues, validateStringField(doc, "id")...)
		issues = append(issues, validateStringField(doc, "version")...)

		if typ == "person" {
			ci, ok := doc["concluded_identity"].(map[string]interface{})
			if !ok {
				issues = append(issues, "concluded_identity must be an object")
			} else {
				issues = append(issues, validateStringField(ci, "primary_name")...)
			}
		} else if typ == "relationship" {
			issues = append(issues, validateStringField(doc, "type")...)
			persons, ok := doc["persons"].([]interface{})
			if !ok {
				issues = append(issues, "persons must be an array")
			} else {
				if len(persons) < 2 {
					issues = append(issues, "persons must contain at least two entries")
				}
				for idx, entry := range persons {
					if _, ok := entry.(string); !ok {
						issues = append(issues, fmt.Sprintf("persons[%d] must be a string", idx))
					}
				}
			}
		} else if typ == "source" {
			issues = append(issues, validateStringField(doc, "title")...)
		} else if typ == "media" {
			issues = append(issues, validateStringField(doc, "uri")...)
		}
	}

	return issues
}

func detectGLXType(path string) string {
	n := filepath.ToSlash(path)
	switch {
	case strings.Contains(n, "/persons/"):
		return "person"
	case strings.Contains(n, "/relationships/"):
		return "relationship"
	case strings.Contains(n, "/sources/"):
		return "source"
	case strings.Contains(n, "/media/"):
		return "media"
	case strings.Contains(n, "/events/"):
		return "event"
	case strings.Contains(n, "/places/"):
		return "place"
	case strings.Contains(n, "/.oracynth/"):
		base := filepath.Base(n)
		if base == "config.glx" {
			return "config"
		}
		if base == "schema-version.glx" {
			return "schema-version"
		}
	}
	return "generic"
}

func validateStringField(doc map[string]interface{}, key string) []string {
	val, ok := doc[key]
	if !ok {
		return []string{fmt.Sprintf("%s is required", key)}
	}
	str, ok := val.(string)
	if !ok {
		return []string{fmt.Sprintf("%s must be a string", key)}
	}
	if strings.TrimSpace(str) == "" {
		return []string{fmt.Sprintf("%s cannot be empty", key)}
	}
	return nil
}
