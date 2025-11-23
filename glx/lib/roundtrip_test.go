package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGEDCOMToSingleFileRoundTrip tests GEDCOM -> Single-file GLX -> Deserialization
func TestGEDCOMToSingleFileRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		gedcomPath   string
		minPersons   int
		minEvents    int
		minRelations int
	}{
		{
			name:         "Shakespeare family",
			gedcomPath:   "../testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged",
			minPersons:   31,
			minEvents:    77,
			minRelations: 49,
		},
		{
			name:         "Kennedy family",
			gedcomPath:   "../testdata/gedcom/5.5.1/kennedy-family/kennedy.ged",
			minPersons:   70,
			minEvents:    139,
			minRelations: 119,
		},
		{
			name:         "GEDCOM 7.0 comprehensive",
			gedcomPath:   "../testdata/gedcom/7.0/comprehensive-spec/maximal70.ged",
			minPersons:   1,
			minEvents:    36,
			minRelations: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Import GEDCOM file
			logPath := filepath.Join(t.TempDir(), "import.log")
			glx1, result, err := importGEDCOMFromFile(tc.gedcomPath, logPath)
			require.NoError(t, err, "GEDCOM import failed")
			require.NotNil(t, result, "Import result should not be nil")

			// Verify initial import
			assert.GreaterOrEqual(t, len(glx1.Persons), tc.minPersons, "Person count mismatch")
			assert.GreaterOrEqual(t, len(glx1.Events), tc.minEvents, "Event count mismatch")
			assert.GreaterOrEqual(t, len(glx1.Relationships), tc.minRelations, "Relationship count mismatch")

			// Step 2: Serialize to single-file GLX
			tempFile := filepath.Join(t.TempDir(), "archive.glx")
			serializer := NewSerializer(nil) // Use default options
			err = serializer.SerializeSingleFile(glx1, tempFile)
			require.NoError(t, err, "Failed to serialize to single file")

			// Step 3: Deserialize back from single file
			glx2, err := serializer.LoadSingleFile(tempFile)
			require.NoError(t, err, "Failed to deserialize from single file")

			// Step 4: Verify round-trip preserved all data
			assert.Len(t, glx2.Persons, len(glx1.Persons), "Person count changed after round-trip")
			assert.Len(t, glx2.Events, len(glx1.Events), "Event count changed after round-trip")
			assert.Len(t, glx2.Relationships, len(glx1.Relationships), "Relationship count changed after round-trip")
			assert.Len(t, glx2.Sources, len(glx1.Sources), "Source count changed after round-trip")
			assert.Len(t, glx2.Citations, len(glx1.Citations), "Citation count changed after round-trip")
			assert.Len(t, glx2.Places, len(glx1.Places), "Place count changed after round-trip")
			assert.Len(t, glx2.Media, len(glx1.Media), "Media count changed after round-trip")
			assert.Len(t, glx2.Repositories, len(glx1.Repositories), "Repository count changed after round-trip")

			// Verify vocabularies preserved
			assert.Len(t, glx2.EventTypes, len(glx1.EventTypes), "EventTypes vocabulary count changed")
			assert.Len(t, glx2.RelationshipTypes, len(glx1.RelationshipTypes), "RelationshipTypes vocabulary count changed")
			assert.Len(t, glx2.PlaceTypes, len(glx1.PlaceTypes), "PlaceTypes vocabulary count changed")

			t.Logf("✓ Round-trip successful: %d persons, %d events, %d relationships preserved",
				len(glx2.Persons), len(glx2.Events), len(glx2.Relationships))
		})
	}
}

// TestGEDCOMToMultiFileRoundTrip tests GEDCOM -> Multi-file GLX -> Deserialization
func TestGEDCOMToMultiFileRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		gedcomPath   string
		minPersons   int
		minEvents    int
		minRelations int
	}{
		{
			name:         "Shakespeare family",
			gedcomPath:   "../testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged",
			minPersons:   31,
			minEvents:    77,
			minRelations: 49,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Import GEDCOM file
			logPath := filepath.Join(t.TempDir(), "import.log")
			glx1, result, err := importGEDCOMFromFile(tc.gedcomPath, logPath)
			require.NoError(t, err, "GEDCOM import failed")
			require.NotNil(t, result, "Import result should not be nil")

			// Verify initial import
			assert.GreaterOrEqual(t, len(glx1.Persons), tc.minPersons, "Person count mismatch")
			assert.GreaterOrEqual(t, len(glx1.Events), tc.minEvents, "Event count mismatch")
			assert.GreaterOrEqual(t, len(glx1.Relationships), tc.minRelations, "Relationship count mismatch")

			// Step 2: Serialize to multi-file GLX
			tempDir := filepath.Join(t.TempDir(), "archive")
			err = os.MkdirAll(tempDir, 0o755)
			require.NoError(t, err, "Failed to create temp directory")

			// Create serializer with vocabularies included
			opts := &SerializerOptions{
				IncludeVocabularies: true,
				Validate:            false,
				Pretty:              true,
			}
			serializer := NewSerializer(opts)
			err = serializer.SerializeMultiFile(glx1, tempDir)
			require.NoError(t, err, "Failed to serialize to multi-file")

			// Step 3: Deserialize back from multi-file
			glx2, err := serializer.LoadMultiFile(tempDir)
			require.NoError(t, err, "Failed to deserialize from multi-file")

			// Step 4: Verify round-trip preserved all data
			assert.Len(t, glx2.Persons, len(glx1.Persons), "Person count changed after round-trip")
			assert.Len(t, glx2.Events, len(glx1.Events), "Event count changed after round-trip")
			assert.Len(t, glx2.Relationships, len(glx1.Relationships), "Relationship count changed after round-trip")
			assert.Len(t, glx2.Sources, len(glx1.Sources), "Source count changed after round-trip")
			assert.Len(t, glx2.Citations, len(glx1.Citations), "Citation count changed after round-trip")
			assert.Len(t, glx2.Places, len(glx1.Places), "Place count changed after round-trip")
			assert.Len(t, glx2.Media, len(glx1.Media), "Media count changed after round-trip")
			assert.Len(t, glx2.Repositories, len(glx1.Repositories), "Repository count changed after round-trip")

			// Verify vocabularies preserved
			assert.Len(t, glx2.EventTypes, len(glx1.EventTypes), "EventTypes vocabulary count changed")
			assert.Len(t, glx2.RelationshipTypes, len(glx1.RelationshipTypes), "RelationshipTypes vocabulary count changed")
			assert.Len(t, glx2.PlaceTypes, len(glx1.PlaceTypes), "PlaceTypes vocabulary count changed")

			t.Logf("✓ Round-trip successful: %d persons, %d events, %d relationships preserved",
				len(glx2.Persons), len(glx2.Events), len(glx2.Relationships))
		})
	}
}

// TestSingleToMultiToSingleRoundTrip tests Single-file -> Multi-file -> Single-file conversion
func TestSingleToMultiToSingleRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		gedcomPath string
	}{
		{
			name:       "Shakespeare family",
			gedcomPath: "../testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged",
		},
		{
			name:       "Remarriage scenario",
			gedcomPath: "../testdata/gedcom/5.5.5/spec-samples/remarriage.ged",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Import GEDCOM to get initial GLX data
			logPath := filepath.Join(t.TempDir(), "import.log")
			glx1, result, err := importGEDCOMFromFile(tc.gedcomPath, logPath)
			require.NoError(t, err, "GEDCOM import failed")
			require.NotNil(t, result, "Import result should not be nil")

			initialPersonCount := len(glx1.Persons)
			initialEventCount := len(glx1.Events)
			initialRelationCount := len(glx1.Relationships)

			// Step 2: Write to single file
			singleFile1 := filepath.Join(t.TempDir(), "archive1.glx")
			serializer := NewSerializer(nil)
			err = serializer.SerializeSingleFile(glx1, singleFile1)
			require.NoError(t, err, "Failed to write to single file")

			// Step 3: Split to multi-file
			multiDir := filepath.Join(t.TempDir(), "archive-multi")
			err = os.MkdirAll(multiDir, 0o755)
			require.NoError(t, err, "Failed to create multi-file directory")

			glx2, err := serializer.LoadSingleFile(singleFile1)
			require.NoError(t, err, "Failed to read from single file")

			// Create serializer with vocabularies for multi-file
			optsMulti := &SerializerOptions{
				IncludeVocabularies: true,
				Validate:            false,
				Pretty:              true,
			}
			serializerMulti := NewSerializer(optsMulti)
			err = serializerMulti.SerializeMultiFile(glx2, multiDir)
			require.NoError(t, err, "Failed to write to multi-file")

			// Step 4: Join back to single file
			glx3, err := serializerMulti.LoadMultiFile(multiDir)
			require.NoError(t, err, "Failed to read from multi-file")

			singleFile2 := filepath.Join(t.TempDir(), "archive2.glx")
			err = serializer.SerializeSingleFile(glx3, singleFile2)
			require.NoError(t, err, "Failed to write back to single file")

			// Step 5: Read final single file and verify
			glx4, err := serializer.LoadSingleFile(singleFile2)
			require.NoError(t, err, "Failed to read final single file")

			// Verify all data preserved through multiple conversions
			assert.Len(t, glx4.Persons, initialPersonCount, "Person count changed")
			assert.Len(t, glx4.Events, initialEventCount, "Event count changed")
			assert.Len(t, glx4.Relationships, initialRelationCount, "Relationship count changed")

			// Verify vocabularies preserved
			assert.Len(t, glx4.EventTypes, len(glx1.EventTypes), "EventTypes vocabulary count changed")
			assert.Len(t, glx4.RelationshipTypes, len(glx1.RelationshipTypes), "RelationshipTypes vocabulary count changed")

			t.Logf("✓ Multi-step round-trip successful: single->multi->single preserved all %d persons, %d events, %d relationships",
				len(glx4.Persons), len(glx4.Events), len(glx4.Relationships))
		})
	}
}

// TestVocabularyPreservation tests that vocabularies are correctly preserved
func TestVocabularyPreservation(t *testing.T) {
	// Import a GEDCOM file that will have vocabularies
	gedcomPath := "../testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged"
	logPath := filepath.Join(t.TempDir(), "import.log")
	glx1, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	require.NoError(t, err, "GEDCOM import failed")
	require.NotNil(t, result, "Import result should not be nil")

	// Ensure vocabularies are loaded
	require.NotEmpty(t, glx1.EventTypes, "Expected event types to be loaded")
	require.NotEmpty(t, glx1.RelationshipTypes, "Expected relationship types to be loaded")

	t.Run("Single-file preservation", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "archive.glx")
		serializer := NewSerializer(nil)

		// Write and read back
		err := serializer.SerializeSingleFile(glx1, tempFile)
		require.NoError(t, err, "Failed to write to single file")

		glx2, err := serializer.LoadSingleFile(tempFile)
		require.NoError(t, err, "Failed to read from single file")

		// Verify vocabularies preserved
		assert.Len(t, glx2.EventTypes, len(glx1.EventTypes), "EventTypes count mismatch")
		assert.Len(t, glx2.RelationshipTypes, len(glx1.RelationshipTypes), "RelationshipTypes count mismatch")
		assert.Len(t, glx2.PlaceTypes, len(glx1.PlaceTypes), "PlaceTypes count mismatch")
		assert.Len(t, glx2.RepositoryTypes, len(glx1.RepositoryTypes), "RepositoryTypes count mismatch")

		// Verify specific vocabulary entries preserved
		for id, vocab := range glx1.EventTypes {
			assert.Contains(t, glx2.EventTypes, id, "EventType ID missing")
			if found, ok := glx2.EventTypes[id]; ok {
				assert.Equal(t, vocab.Label, found.Label, "EventType label mismatch for %s", id)
			}
		}
	})

	t.Run("Multi-file preservation", func(t *testing.T) {
		tempDir := filepath.Join(t.TempDir(), "archive")
		err := os.MkdirAll(tempDir, 0o755)
		require.NoError(t, err, "Failed to create temp directory")

		// Create serializer with vocabularies included
		opts := &SerializerOptions{
			IncludeVocabularies: true,
			Validate:            false,
			Pretty:              true,
		}
		serializer := NewSerializer(opts)

		// Write with vocabularies included
		err = serializer.SerializeMultiFile(glx1, tempDir)
		require.NoError(t, err, "Failed to write to multi-file")

		// Read back
		glx2, err := serializer.LoadMultiFile(tempDir)
		require.NoError(t, err, "Failed to read from multi-file")

		// Verify vocabularies preserved
		assert.Len(t, glx2.EventTypes, len(glx1.EventTypes), "EventTypes count mismatch")
		assert.Len(t, glx2.RelationshipTypes, len(glx1.RelationshipTypes), "RelationshipTypes count mismatch")
		assert.Len(t, glx2.PlaceTypes, len(glx1.PlaceTypes), "PlaceTypes count mismatch")
		assert.Len(t, glx2.RepositoryTypes, len(glx1.RepositoryTypes), "RepositoryTypes count mismatch")
	})
}
