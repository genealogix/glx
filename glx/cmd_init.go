package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	initSingleFile bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new GENEALOGIX archive",
	Long: `Initialize a new GENEALOGIX archive with the proper directory structure.

By default, creates a multi-file archive with separate directories for each
entity type (persons/, events/, places/, etc.) along with standard vocabulary
files and supporting documentation.

Use --single-file to create a single archive.glx file instead.`,
	Example: `  # Initialize multi-file archive (default)
  glx init

  # Initialize single-file archive
  glx init --single-file
  glx init -s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(initSingleFile)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initSingleFile, "single-file", "s", false, "create a single-file archive instead of multi-file")
}

func runInit(singleFile bool) error {
	// Check if we're in the GENEALOGIX spec repository (not a user archive)
	if isSpecRepository() {
		return fmt.Errorf("cannot run 'glx init' in the GENEALOGIX specification repository. Create a new directory for your family archive first")
	}

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

	// Create directory structure for a GENEALOGIX repository
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
		"vocabularies",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create standard vocabulary files
	if err := createStandardVocabularies(); err != nil {
		return err
	}

	// Create .gitignore file
	if err := os.WriteFile(".gitignore", defaultGitignore, 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %v", err)
	}

	// Create README.md for the repository
	if err := os.WriteFile("README.md", defaultReadme, 0644); err != nil {
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

func isSpecRepository() bool {
	// Check if we're in the GENEALOGIX spec repository by looking for key files
	specFiles := []string{"specification/README.md", "specification/schema/v1/person.schema.json", "glx/main.go"}
	for _, file := range specFiles {
		if _, err := os.Stat(file); err != nil {
			return false
		}
	}
	return true
}

// createStandardVocabularies is now in vocabularies_embed.go
