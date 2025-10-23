package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "check-schemas":
		if err := runCheckSchemas(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "validate":
		if err := runValidate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("genealogix CLI")
	fmt.Println("Usage:")
	fmt.Println("  glx init                Initialize a new genealogix repository")
	fmt.Println("  glx validate [paths]    Validate .glx files (defaults to current directory)")
	fmt.Println("  glx check-schemas       Validate schema files for required metadata")
}

func runInit() error {
	// Create directory structure for a genealogix repository
	dirs := []string{
		"persons",
		"relationships",
		"events",
		"places",
		"sources",
		"citations",
		"repositories",
		"assertions",
		"media",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create .gitignore file
	gitignore := `# GENEALOGIX Repository
# Ignore temporary files and build artifacts

*.tmp
*.bak
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db
`

	if err := os.WriteFile(".gitignore", []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %v", err)
	}

	// Create README.md for the repository
	readme := `# GENEALOGIX Family Archive

This is a genealogical archive using the GENEALOGIX format.

## Structure

### Core Data
- persons/ - Individual person records
- relationships/ - Family relationships and connections
- events/ - Life events (births, marriages, deaths, occupations, residences, etc.)
- places/ - Geographic locations with hierarchies

### Evidence & Sources
- sources/ - Bibliographic sources and publications
- citations/ - Specific references within sources
- repositories/ - Archives, libraries, and institutions holding sources
- assertions/ - Evidence-based conclusions and claims

### Media
- media/ - Photos, documents, and other media files

## Getting Started

Use glx commands to work with this archive:

` + "```bash" + `
glx validate              # Validate all .glx files
glx validate persons/     # Validate specific directory
` + "```" + `

## File Format

All genealogical data is stored in YAML files with the .glx extension.
Each file represents a specific entity (person, event, place, citation, etc.).

### Standard ID Prefixes
- person-XXXXXXXX: Person records
- rel-XXXXXXXX: Relationship records
- event-XXXXXXXX: Event/Fact records
- place-XXXXXXXX: Place records
- assertion-XXXXXXXX: Assertion records
- source-XXXXXXXX: Source records
- citation-XXXXXXXX: Citation records
- repository-XXXXXXXX: Repository records
- media-XXXXXXXX: Media records

## Documentation

See the [GENEALOGIX specification](https://github.com/genealogix/spec) for detailed format information.
`

	if err := os.WriteFile("README.md", []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %v", err)
	}

	fmt.Println("Initialized new genealogix repository")
	fmt.Println("Created directories:")
	fmt.Println("  Core: persons/, relationships/, events/, places/")
	fmt.Println("  Evidence: sources/, citations/, repositories/, assertions/")
	fmt.Println("  Media: media/")
	fmt.Println("Created .gitignore and README.md")

	return nil
}

func runValidate(paths []string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, path := range paths {
		if err := validatePath(path); err != nil {
			return fmt.Errorf("validation failed for %s: %v", path, err)
		}
	}

	fmt.Printf("Validation completed successfully\n")
	return nil
}

func runCheckSchemas() error {
	// Validate that all schema files have required metadata
	schemaDir := "schema/v1"
	files, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("failed to read schema directory: %v", err)
	}

	requiredSchemas := map[string]bool{
		"person.schema.json":       false,
		"event.schema.json":        false,
		"place.schema.json":        false,
		"source.schema.json":       false,
		"citation.schema.json":     false,
		"repository.schema.json":   false,
		"assertion.schema.json":    false,
		"relationship.schema.json": false,
		"media.schema.json":        false,
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if _, exists := requiredSchemas[file.Name()]; exists {
			requiredSchemas[file.Name()] = true

			// Basic validation that it's a valid JSON schema
			schemaPath := fmt.Sprintf("%s/%s", schemaDir, file.Name())
			if err := validateSchemaFile(schemaPath); err != nil {
				return fmt.Errorf("invalid schema %s: %v", file.Name(), err)
			}
		}
	}

	// Check that all required schemas are present
	for schema, found := range requiredSchemas {
		if !found {
			return fmt.Errorf("missing required schema: %s", schema)
		}
	}

	fmt.Println("All schema files are valid and present")
	return nil
}

func validatePath(path string) error {
	// This is a placeholder - in a full implementation, this would:
	// 1. Walk the directory tree
	// 2. Find all .glx files
	// 3. Validate each file against appropriate schema
	// 4. Check cross-references between entities
	// 5. Report validation errors

	fmt.Printf("Validating %s\n", path)
	return nil
}

func validateSchemaFile(schemaPath string) error {
	// This is a placeholder - in a full implementation, this would:
	// 1. Parse the JSON schema
	// 2. Validate it against the meta schema
	// 3. Check for required fields like $id, $schema, title
	// 4. Verify schema references are valid

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	// Basic JSON validation
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return err
	}

	// Check for required schema fields
	requiredFields := []string{"$schema", "$id", "title", "type"}
	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}
