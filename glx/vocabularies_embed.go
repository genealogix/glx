package main

import (
	"fmt"
	"os"

	vocabularies "github.com/genealogix/spec/specification/5-standard-vocabularies"
)

func createStandardVocabularies() error {
	for filename, content := range vocabularies.Files {
		outputPath := "vocabularies/" + filename
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			return fmt.Errorf("failed to create %s: %v", outputPath, err)
		}
	}

	fmt.Println("Created standard vocabulary files in vocabularies/")
	return nil
}
