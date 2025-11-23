package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStandardVocabularies(t *testing.T) {
	vocabs := StandardVocabularies()

	// Should have vocabularies
	if len(vocabs) == 0 {
		t.Fatal("StandardVocabularies returned empty map")
	}

	// Check for expected vocabularies
	expected := []string{
		"event-types.glx",
		"relationship-types.glx",
		"place-types.glx",
		"source-types.glx",
		"repository-types.glx",
		"media-types.glx",
		"participant-roles.glx",
		"confidence-levels.glx",
		"quality-ratings.glx",
		"person-properties.glx",
		"event-properties.glx",
		"relationship-properties.glx",
		"place-properties.glx",
	}

	for _, name := range expected {
		content, ok := vocabs[name]
		if !ok {
			t.Errorf("Missing vocabulary: %s", name)

			continue
		}
		if len(content) == 0 {
			t.Errorf("Empty vocabulary: %s", name)
		}
	}

	t.Logf("Found %d vocabularies", len(vocabs))
}

func TestListStandardVocabularies(t *testing.T) {
	names := ListStandardVocabularies()

	// Should have vocabularies
	if len(names) == 0 {
		t.Fatal("ListStandardVocabularies returned empty list")
	}

	// Names should not have .glx extension
	for _, name := range names {
		if len(name) > 4 && name[len(name)-4:] == ".glx" {
			t.Errorf("Vocabulary name should not have .glx extension: %s", name)
		}
	}

	t.Logf("Found vocabulary names: %v", names)
}

func TestGetStandardVocabulary(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"event-types", false},
		{"relationship-types", false},
		{"place-types", false},
		{"source-types", false},
		{"nonexistent-vocab", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GetStandardVocabulary(tt.name)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			if len(content) == 0 {
				t.Errorf("Empty content for %s", tt.name)
			}
		})
	}
}

func TestWriteStandardVocabularies(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Write vocabularies
	err := WriteStandardVocabularies(tmpDir)
	if err != nil {
		t.Fatalf("Failed to write vocabularies: %v", err)
	}

	// Check vocabularies directory exists
	vocabDir := filepath.Join(tmpDir, "vocabularies")
	if _, err := os.Stat(vocabDir); os.IsNotExist(err) {
		t.Fatal("Vocabularies directory not created")
	}

	// Check files exist
	files, err := os.ReadDir(vocabDir)
	if err != nil {
		t.Fatalf("Failed to read vocabulary directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No vocabulary files written")
	}

	// Check each file has content
	for _, file := range files {
		path := filepath.Join(vocabDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file.Name(), err)

			continue
		}
		if len(data) == 0 {
			t.Errorf("Empty vocabulary file: %s", file.Name())
		}
	}

	t.Logf("Successfully wrote %d vocabulary files", len(files))
}

func TestWriteVocabulariesToFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "all-vocabularies.glx")

	// Write vocabularies to single file
	err := WriteVocabulariesToFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to write vocabularies file: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Vocabularies file not created")
	}

	// Check file has content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read vocabularies file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Empty vocabularies file")
	}

	// Check header is present
	header := "# GLX Standard Vocabularies"
	if !contains(string(data), header) {
		t.Errorf("Vocabularies file missing header: %s", header)
	}

	// Check at least one vocabulary name is present
	if !contains(string(data), "event-types.glx") {
		t.Error("Vocabularies file missing event-types.glx")
	}

	t.Logf("Successfully wrote vocabularies file (%d bytes)", len(data))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
