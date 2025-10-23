package main

import (
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
		singleFile := false
		if len(os.Args) > 2 && (os.Args[2] == "--single-file" || os.Args[2] == "-s") {
			singleFile = true
		}
		if err := runInit(singleFile); err != nil {
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
	fmt.Println("  glx init [--single-file]  Initialize a new genealogix repository")
	fmt.Println("                            (default: multi-file with folders)")
	fmt.Println("  glx validate [paths]      Validate .glx files and cross-references")
	fmt.Println("  glx check-schemas         Validate schema files for required metadata")
}

func runInit(singleFile bool) error {
	if singleFile {
		// Create single-file archive template
		template := `# GENEALOGIX Family Archive
# Single-file format

persons: {}
relationships: {}
events: {}
places: {}
sources: {}
citations: {}
repositories: {}
assertions: {}
media: {}
`
		if err := os.WriteFile("archive.glx", []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to create archive.glx: %v", err)
		}

		fmt.Println("Initialized single-file GENEALOGIX archive: archive.glx")
		fmt.Println("Add entities under the appropriate type keys (persons, sources, etc.)")
		fmt.Println("Entity IDs are map keys - don't include 'id' field in entities")
		return nil
	}

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

	fmt.Println("Initialized multi-file GENEALOGIX repository")
	fmt.Println("Created directories:")
	fmt.Println("  Core: persons/, relationships/, events/, places/")
	fmt.Println("  Evidence: sources/, citations/, repositories/, assertions/")
	fmt.Println("  Media: media/")
	fmt.Println("Created .gitignore and README.md")
	fmt.Println("")
	fmt.Println("Each .glx file should have entity type keys at the top level.")
	fmt.Println("Entity IDs are map keys - don't include 'id' field in entities.")

	return nil
}
