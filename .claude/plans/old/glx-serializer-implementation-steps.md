# GLX Serializer - Implementation Steps

**Date**: 2025-11-18
**Status**: Detailed Implementation Plan
**Estimated Time**: 6-8 hours total

---

## Overview

This document breaks down the serializer implementation into concrete, sequential tasks.

---

## Phase 1: Foundation (1-2 hours)

### Task 1.1: Create ID Generator

**File**: `lib/id_generator.go`
**Lines**: ~60
**Dependencies**: None

```go
// Functions to implement:
- GenerateRandomID() (string, error)
- GenerateEntityFilename(entityType string) (string, error)
- MustGenerateRandomID() string
- generateUniqueFilename(entityType string, used map[string]bool) (string, error)
```

**Tests**: `lib/id_generator_test.go`
- TestGenerateRandomID (uniqueness, format)
- TestGenerateEntityFilename (format, suffix)
- TestCollisionDetection (retry logic)

### Task 1.2: Create Vocabulary Embedder

**File**: `lib/vocabularies.go`
**Lines**: ~80
**Dependencies**: Task 1.1 complete

```go
// Functions to implement:
- StandardVocabularies() map[string]string  // using embed.FS
- WriteStandardVocabularies(outputDir string) error
- GetStandardVocabulary(name string) (string, error)
- ListStandardVocabularies() ([]string, error)
```

**embed.FS setup**:
```go
//go:embed ../glx/standard-vocabularies/*.glx
var vocabulariesFS embed.FS
```

**Tests**: `lib/vocabularies_test.go`
- TestStandardVocabulariesEmbedded
- TestWriteStandardVocabularies
- TestGetStandardVocabulary
- TestListStandardVocabularies

### Task 1.3: Create Entity Wrapper Type

**File**: `lib/types.go` (add to existing)
**Lines**: ~20
**Dependencies**: None

```go
// Add to types.go:
type EntityWithID[T any] struct {
	ID     string `yaml:"_id"`
	Entity T      `yaml:",inline"`
}
```

**Tests**: `lib/types_test.go`
- TestEntityWithIDMarshal
- TestEntityWithIDUnmarshal
- TestEntityWithIDInline (verify ,inline works)

---

## Phase 2: Core Serializer (2-3 hours)

### Task 2.1: Create Serializer Interface

**File**: `lib/serializer.go`
**Lines**: ~300
**Dependencies**: Tasks 1.1, 1.2, 1.3

**Interface**:
```go
type Serializer interface {
	SerializeSingleFile(glx *GLXFile, outputPath string) error
	SerializeMultiFile(glx *GLXFile, outputDir string) error
	SerializeSingleFileBytes(glx *GLXFile) ([]byte, error)
}

type DefaultSerializer struct {
	IncludeVocabularies bool
	ValidateBeforeSave  bool
}
```

**Functions to implement**:
```go
- NewDefaultSerializer() *DefaultSerializer
- (s *DefaultSerializer) SerializeSingleFileBytes(glx *GLXFile) ([]byte, error)
- (s *DefaultSerializer) SerializeSingleFile(glx *GLXFile, outputPath string) error
- (s *DefaultSerializer) SerializeMultiFile(glx *GLXFile, outputDir string) error
- writeEntities[T any](entities map[string]T, dir, entityType string) error
- loadEntitiesWithID[T any](dir string) (map[string]T, error)
```

**Implementation details**:
1. **SerializeSingleFileBytes**:
   - Optionally validate (if ValidateBeforeSave)
   - Marshal GLXFile to YAML
   - Warn if >10MB
   - Return bytes

2. **SerializeSingleFile**:
   - Check if file exists (NEW ONLY)
   - Call SerializeSingleFileBytes
   - Optionally prepend vocabularies
   - Write to file

3. **SerializeMultiFile**:
   - Check if directory exists (NEW ONLY)
   - Create directory structure (persons/, events/, etc.)
   - Write vocabularies (if IncludeVocabularies)
   - Write each entity type using writeEntities
   - Return errors

4. **writeEntities** (generic):
   - Iterate over entities map
   - Generate random filename for each
   - Wrap entity with EntityWithID
   - Marshal to YAML
   - Write file
   - Track used filenames for collision detection

**Tests**: `lib/serializer_test.go`
- TestSerializeSingleFileBytes
- TestSerializeSingleFile (creates file)
- TestSerializeSingleFileExists (error if exists)
- TestSerializeMultiFile (creates structure)
- TestSerializeMultiFileExists (error if exists)
- TestWriteEntities (generic function)
- TestValidationBeforeSave

### Task 2.2: Add Validation Integration

**File**: `lib/serializer.go` (add to existing)
**Lines**: ~30
**Dependencies**: Task 2.1

```go
func (s *DefaultSerializer) validate(glx *GLXFile) error {
	if !s.ValidateBeforeSave {
		return nil
	}

	result := glx.Validate()
	if len(result.Errors) > 0 {
		// Return first few errors for clarity
		errorMsgs := make([]string, 0, min(5, len(result.Errors)))
		for i := 0; i < min(5, len(result.Errors)); i++ {
			errorMsgs = append(errorMsgs, result.Errors[i].String())
		}

		if len(result.Errors) > 5 {
			errorMsgs = append(errorMsgs, fmt.Sprintf("... and %d more errors", len(result.Errors)-5))
		}

		return fmt.Errorf("validation failed:\n  %s", strings.Join(errorMsgs, "\n  "))
	}

	return nil
}
```

**Tests**:
- TestValidationSuccess
- TestValidationFailure
- TestValidationSkipped (with ValidateBeforeSave=false)

---

## Phase 3: Archive Loading (1-2 hours)

### Task 3.1: Single-File Loader

**File**: `lib/loader.go`
**Lines**: ~100
**Dependencies**: Task 2.1

```go
// Functions to implement:
- LoadGLXFile(filepath string) (*GLXFile, error)
- LoadGLXFileBytes(data []byte) (*GLXFile, error)
```

**Tests**: `lib/loader_test.go`
- TestLoadGLXFile
- TestLoadGLXFileNotFound
- TestLoadGLXFileInvalid

### Task 3.2: Multi-File Loader

**File**: `lib/loader.go` (add to existing)
**Lines**: ~150
**Dependencies**: Tasks 2.1, 3.1

```go
// Functions to implement:
- LoadGLXDirectory(dirPath string) (*GLXFile, error)
- loadEntitiesFromDir[T any](dir string) (map[string]T, error)
```

**Implementation**:
1. Check directory exists
2. Load each entity type directory
3. Load vocabularies (optional)
4. Combine into GLXFile
5. Return

**Tests**:
- TestLoadGLXDirectory
- TestLoadGLXDirectoryNotFound
- TestLoadGLXDirectoryEmpty
- TestLoadEntitiesFromDir (generic)

---

## Phase 4: CLI Integration (1-2 hours)

### Task 4.1: Update Import Command

**File**: `cmd/glx/import.go` (update existing)
**Lines**: ~50 changes
**Dependencies**: Tasks 2.1, 2.2

**Changes**:
1. Add output flags:
   ```go
   --output, -o: output path (file or directory)
   --multi-file: use multi-file format
   --no-vocabularies: skip writing vocabularies
   --no-validate: skip validation before save
   ```

2. After GEDCOM import, serialize:
   ```go
   serializer := &DefaultSerializer{
       IncludeVocabularies: !noVocabularies,
       ValidateBeforeSave:  !noValidate,
   }

   if multiFile {
       err = serializer.SerializeMultiFile(glx, output)
   } else {
       err = serializer.SerializeSingleFile(glx, output)
   }
   ```

**Tests**: `cmd/glx/import_test.go`
- TestImportToSingleFile
- TestImportToMultiFile
- TestImportWithVocabularies
- TestImportWithoutVocabularies
- TestImportWithValidation

### Task 4.2: Implement Split Command

**File**: `cmd/glx/split.go` (new)
**Lines**: ~120
**Dependencies**: Tasks 2.1, 3.1

**Command**:
```bash
glx split <input-file> -o <output-directory> [--force] [--no-vocabularies]
```

**Implementation**:
```go
func splitCommand() *cli.Command {
	return &cli.Command{
		Name:  "split",
		Usage: "Split single-file archive to multi-file format",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Required: true,
				Usage:    "Output directory path",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite existing directory",
			},
			&cli.BoolFlag{
				Name:  "no-vocabularies",
				Usage: "Skip writing vocabularies",
			},
		},
		Action: func(c *cli.Context) error {
			// Load single file
			inputFile := c.Args().First()
			glx, err := LoadGLXFile(inputFile)
			if err != nil {
				return err
			}

			// Serialize to multi-file
			serializer := &DefaultSerializer{
				IncludeVocabularies: !c.Bool("no-vocabularies"),
				ValidateBeforeSave:  true,
			}

			outputDir := c.String("output")

			// Check if exists (unless --force)
			if !c.Bool("force") {
				if _, err := os.Stat(outputDir); err == nil {
					return fmt.Errorf("directory already exists: %s (use --force to overwrite)", outputDir)
				}
			}

			// Create multi-file archive
			if err := serializer.SerializeMultiFile(glx, outputDir); err != nil {
				return fmt.Errorf("failed to split archive: %w", err)
			}

			fmt.Printf("Successfully split %s to %s\n", inputFile, outputDir)
			return nil
		},
	}
}
```

**Tests**: `cmd/glx/split_test.go`
- TestSplitCommand
- TestSplitCommandForce
- TestSplitCommandNoVocabularies

### Task 4.3: Implement Join Command

**File**: `cmd/glx/join.go` (new)
**Lines**: ~120
**Dependencies**: Tasks 2.1, 3.2

**Command**:
```bash
glx join <input-directory> -o <output-file> [--force] [--include-vocabularies]
```

**Implementation**:
```go
func joinCommand() *cli.Command {
	return &cli.Command{
		Name:  "join",
		Usage: "Join multi-file archive to single file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Required: true,
				Usage:    "Output file path",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite existing file",
			},
			&cli.BoolFlag{
				Name:  "include-vocabularies",
				Usage: "Include vocabularies in single file",
			},
		},
		Action: func(c *cli.Context) error {
			// Load multi-file directory
			inputDir := c.Args().First()
			glx, err := LoadGLXDirectory(inputDir)
			if err != nil {
				return err
			}

			// Serialize to single file
			serializer := &DefaultSerializer{
				IncludeVocabularies: c.Bool("include-vocabularies"),
				ValidateBeforeSave:  true,
			}

			outputFile := c.String("output")

			// Check if exists (unless --force)
			if !c.Bool("force") {
				if _, err := os.Stat(outputFile); err == nil {
					return fmt.Errorf("file already exists: %s (use --force to overwrite)", outputFile)
				}
			}

			// Create single file
			if err := serializer.SerializeSingleFile(glx, outputFile); err != nil {
				return fmt.Errorf("failed to join archive: %w", err)
			}

			fmt.Printf("Successfully joined %s to %s\n", inputDir, outputFile)
			return nil
		},
	}
}
```

**Tests**: `cmd/glx/join_test.go`
- TestJoinCommand
- TestJoinCommandForce
- TestJoinCommandWithVocabularies

### Task 4.4: Register Commands

**File**: `cmd/glx/main.go`
**Lines**: ~10
**Dependencies**: Tasks 4.2, 4.3

```go
app.Commands = []*cli.Command{
	// ... existing commands ...
	splitCommand(),
	joinCommand(),
}
```

---

## Phase 5: Integration Tests (1 hour)

### Task 5.1: End-to-End GEDCOM Import Test

**File**: `lib/gedcom_e2e_test.go` (new)
**Lines**: ~150

**Test**:
```go
func TestGEDCOMImportToSingleFile(t *testing.T) {
	// Import Shakespeare GEDCOM
	glx, _, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", "")
	require.NoError(t, err)

	// Serialize to single file
	tmpFile := filepath.Join(t.TempDir(), "shakespeare.glx")
	serializer := &DefaultSerializer{
		IncludeVocabularies: true,
		ValidateBeforeSave:  true,
	}
	err = serializer.SerializeSingleFile(glx, tmpFile)
	require.NoError(t, err)

	// Load back
	loadedGLX, err := LoadGLXFile(tmpFile)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, len(glx.Persons), len(loadedGLX.Persons))
	assert.Equal(t, len(glx.Events), len(loadedGLX.Events))
	assert.Equal(t, len(glx.Relationships), len(loadedGLX.Relationships))
}

func TestGEDCOMImportToMultiFile(t *testing.T) {
	// Import Shakespeare GEDCOM
	glx, _, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", "")
	require.NoError(t, err)

	// Serialize to multi-file
	tmpDir := filepath.Join(t.TempDir(), "shakespeare-archive")
	serializer := &DefaultSerializer{
		IncludeVocabularies: true,
		ValidateBeforeSave:  true,
	}
	err = serializer.SerializeMultiFile(glx, tmpDir)
	require.NoError(t, err)

	// Verify structure
	assert.DirExists(t, filepath.Join(tmpDir, "persons"))
	assert.DirExists(t, filepath.Join(tmpDir, "events"))
	assert.DirExists(t, filepath.Join(tmpDir, "vocabularies"))

	// Load back
	loadedGLX, err := LoadGLXDirectory(tmpDir)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, len(glx.Persons), len(loadedGLX.Persons))
}
```

### Task 5.2: Split/Join Round-Trip Test

**File**: `lib/split_join_test.go` (new)
**Lines**: ~100

**Test**:
```go
func TestSplitJoinRoundTrip(t *testing.T) {
	// Create test GLXFile
	glx := createTestGLXFile()

	// Serialize to single file
	singleFile := filepath.Join(t.TempDir(), "test.glx")
	serializer := &DefaultSerializer{
		IncludeVocabularies: true,
		ValidateBeforeSave:  true,
	}
	err := serializer.SerializeSingleFile(glx, singleFile)
	require.NoError(t, err)

	// Load single file
	glx1, err := LoadGLXFile(singleFile)
	require.NoError(t, err)

	// Split to multi-file
	multiDir := filepath.Join(t.TempDir(), "test-archive")
	err = serializer.SerializeMultiFile(glx1, multiDir)
	require.NoError(t, err)

	// Load multi-file
	glx2, err := LoadGLXDirectory(multiDir)
	require.NoError(t, err)

	// Join back to single file
	singleFile2 := filepath.Join(t.TempDir(), "test2.glx")
	err = serializer.SerializeSingleFile(glx2, singleFile2)
	require.NoError(t, err)

	// Load final file
	glx3, err := LoadGLXFile(singleFile2)
	require.NoError(t, err)

	// All should be equal
	assert.Equal(t, len(glx.Persons), len(glx3.Persons))
	assert.Equal(t, len(glx.Events), len(glx3.Events))
}
```

---

## Task Summary

| Phase | Tasks | Files | Lines | Time |
|-------|-------|-------|-------|------|
| 1. Foundation | 3 | 3 | ~160 | 1-2h |
| 2. Core Serializer | 2 | 1 | ~330 | 2-3h |
| 3. Archive Loading | 2 | 1 | ~250 | 1-2h |
| 4. CLI Integration | 4 | 4 | ~300 | 1-2h |
| 5. Integration Tests | 2 | 2 | ~250 | 1h |
| **Total** | **13** | **11** | **~1,290** | **6-10h** |

---

## Implementation Order

**Day 1** (3-4 hours):
1. Task 1.1: ID Generator
2. Task 1.2: Vocabulary Embedder
3. Task 1.3: Entity Wrapper Type
4. Task 2.1: Core Serializer (part 1)

**Day 2** (3-4 hours):
5. Task 2.1: Core Serializer (part 2)
6. Task 2.2: Validation Integration
7. Task 3.1: Single-File Loader
8. Task 3.2: Multi-File Loader

**Day 3** (2-3 hours):
9. Task 4.1: Update Import Command
10. Task 4.2: Split Command
11. Task 4.3: Join Command
12. Task 4.4: Register Commands
13. Task 5.1: E2E Tests
14. Task 5.2: Round-Trip Tests

---

## Testing Strategy

### Unit Tests (per task)
- Each function has dedicated tests
- Test happy path and error cases
- Use table-driven tests where appropriate

### Integration Tests (Phase 5)
- End-to-end GEDCOM import
- Split/join round-trips
- Large file tests (Shakespeare, Kennedy)

### Manual Testing Checklist
- [ ] Import GEDCOM to single file
- [ ] Import GEDCOM to multi-file
- [ ] Split existing archive
- [ ] Join split archive
- [ ] Validate archives
- [ ] Check vocabulary files present
- [ ] Verify file sizes reasonable
- [ ] Test with --force flag
- [ ] Test with --no-validate flag
- [ ] Test with --no-vocabularies flag

---

## Success Criteria

- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] GEDCOM import creates valid GLX archives
- [ ] Split/join round-trip preserves data
- [ ] Vocabularies embedded and written correctly
- [ ] CLI commands work as expected
- [ ] Error messages clear and helpful
- [ ] Documentation updated
- [ ] Changelog updated
- [ ] Version bumped

---

## Next: Start Implementation

Ready to begin Task 1.1: Create ID Generator?
