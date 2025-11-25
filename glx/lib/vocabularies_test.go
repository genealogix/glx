package lib

import (
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

// These tests were removed because WriteStandardVocabularies and WriteVocabulariesToFile
// were removed from lib (they violated the no-I/O rule). Vocabulary writing is now
// handled by the CLI commands, and vocabulary serialization is tested in roundtrip tests.

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
