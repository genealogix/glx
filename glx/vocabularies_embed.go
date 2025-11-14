package main

import (
	"embed"
	"fmt"
	"os"
)

//go:embed vocabularies/*.glx
var vocabulariesFS embed.FS

// VocabularyFiles list of vocabulary files to create
var vocabularyFiles = []string{
	"relationship-types.glx",
	"event-types.glx",
	"place-types.glx",
	"repository-types.glx",
	"participant-roles.glx",
	"media-types.glx",
	"confidence-levels.glx",
	"quality-ratings.glx",
}

func createStandardVocabularies() error {
	for _, filename := range vocabularyFiles {
		// Read embedded file
		embeddedPath := "vocabularies/" + filename
		content, err := vocabulariesFS.ReadFile(embeddedPath)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %v", embeddedPath, err)
		}

		// Write to output path
		outputPath := "vocabularies/" + filename
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create %s: %v", outputPath, err)
		}
	}

	fmt.Println("Created standard vocabulary files in vocabularies/")
	return nil
}
