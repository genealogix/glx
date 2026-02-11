// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestDecodeGEDCOMBlob(t *testing.T) {
	// GEDCOM BLOB encoding: each char minus '.' (0x2E) gives 6 bits.
	// 4 chars → 3 bytes.
	// '.' = 0, '/' = 1, '0' = 2, etc.

	// Simple test: 4 dots should produce 3 zero bytes
	decoded, err := decodeGEDCOMBlob("....")
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if len(decoded) != 3 {
		t.Fatalf("Expected 3 bytes, got %d", len(decoded))
	}
	for i, b := range decoded {
		if b != 0 {
			t.Errorf("Byte %d: expected 0, got %d", i, b)
		}
	}

	// Test with newlines/whitespace (should be stripped)
	decoded2, err := decodeGEDCOMBlob("....\n....")
	if err != nil {
		t.Fatalf("Decode with newlines failed: %v", err)
	}
	if len(decoded2) != 6 {
		t.Fatalf("Expected 6 bytes, got %d", len(decoded2))
	}

	// Empty blob
	_, err = decodeGEDCOMBlob("")
	if err == nil {
		t.Error("Expected error for empty blob")
	}

	// Invalid length (not multiple of 4)
	_, err = decodeGEDCOMBlob("...")
	if err == nil {
		t.Error("Expected error for invalid length")
	}
}

func TestCopyMediaFiles_FileCopy(t *testing.T) {
	// Set up source directory with a test file
	srcDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(srcDir, "photos"), 0o755); err != nil {
		t.Fatal(err)
	}
	testContent := []byte("fake jpeg data")
	if err := os.WriteFile(filepath.Join(srcDir, "photos", "portrait.jpg"), testContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Set up destination directory
	destDir := t.TempDir()

	mediaFiles := []glxlib.MediaFileSource{
		{
			MediaID:        "media-1",
			SourceType:     glxlib.MediaSourceFile,
			RelativePath:   "photos/portrait.jpg",
			TargetFilename: "portrait.jpg",
		},
	}

	err := copyMediaFiles(destDir, mediaFiles, srcDir, false)
	if err != nil {
		t.Fatalf("copyMediaFiles failed: %v", err)
	}

	// Verify file was copied
	copied, err := os.ReadFile(filepath.Join(destDir, "media", "files", "portrait.jpg"))
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}
	if string(copied) != string(testContent) {
		t.Errorf("Copied content mismatch: got %q, want %q", string(copied), string(testContent))
	}
}

func TestCopyMediaFiles_BlobWrite(t *testing.T) {
	destDir := t.TempDir()

	// 4 dots = 3 zero bytes
	mediaFiles := []glxlib.MediaFileSource{
		{
			MediaID:        "media-1",
			SourceType:     glxlib.MediaSourceBlob,
			BlobData:       "....",
			TargetFilename: "blob-media-1.bin",
		},
	}

	err := copyMediaFiles(destDir, mediaFiles, "", false)
	if err != nil {
		t.Fatalf("copyMediaFiles failed: %v", err)
	}

	// Verify blob file was written
	data, err := os.ReadFile(filepath.Join(destDir, "media", "files", "blob-media-1.bin"))
	if err != nil {
		t.Fatalf("Failed to read blob file: %v", err)
	}
	if len(data) != 3 {
		t.Errorf("Expected 3 bytes, got %d", len(data))
	}
}

func TestCopyMediaFiles_MissingSourceWarns(t *testing.T) {
	destDir := t.TempDir()
	srcDir := t.TempDir() // Empty source directory

	mediaFiles := []glxlib.MediaFileSource{
		{
			MediaID:        "media-1",
			SourceType:     glxlib.MediaSourceFile,
			RelativePath:   "nonexistent.jpg",
			TargetFilename: "nonexistent.jpg",
		},
	}

	// Should not return error (warnings on stderr instead)
	err := copyMediaFiles(destDir, mediaFiles, srcDir, false)
	if err != nil {
		t.Fatalf("copyMediaFiles should not fail for missing files: %v", err)
	}

	// Verify the file was NOT created
	_, err = os.Stat(filepath.Join(destDir, "media", "files", "nonexistent.jpg"))
	if !os.IsNotExist(err) {
		t.Error("Expected file to not exist")
	}
}

func TestCopyMediaFiles_EmptyList(t *testing.T) {
	destDir := t.TempDir()

	// Empty list should be a no-op (no media/files directory created)
	err := copyMediaFiles(destDir, nil, "", false)
	if err != nil {
		t.Fatalf("copyMediaFiles should succeed for empty list: %v", err)
	}

	_, err = os.Stat(filepath.Join(destDir, "media", "files"))
	if !os.IsNotExist(err) {
		t.Error("media/files directory should not be created for empty list")
	}
}

// TestE2E_TortureTest_MediaFileCopy is a comprehensive end-to-end test that imports
// the GEDCOM 5.5.1 torture test and verifies that media files are correctly copied
// into media/files/. It exercises the full pipeline: GEDCOM parsing → lib produces
// MediaFileSource entries → CLI copyMediaFiles copies actual files from disk.
func TestE2E_TortureTest_MediaFileCopy(t *testing.T) {
	gedcomPath := filepath.Join("testdata", "gedcom", "5.5.1", "torture-test-551", "torture-test.ged")
	gedcomDir := filepath.Dir(gedcomPath)

	// Step 1: Import the GEDCOM (lib layer)
	gedcomFile, err := os.Open(gedcomPath)
	if err != nil {
		t.Fatalf("Failed to open GEDCOM file: %v", err)
	}
	defer func() { _ = gedcomFile.Close() }()

	glx, result, err := glxlib.ImportGEDCOM(gedcomFile, nil)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Step 2: Verify lib produced MediaFileSource entries
	if len(result.MediaFiles) == 0 {
		t.Fatal("ImportGEDCOM returned no MediaFileSource entries")
	}
	t.Logf("Import produced %d MediaFileSource entries", len(result.MediaFiles))

	// Categorize media file sources
	var fileSources, blobSources []glxlib.MediaFileSource
	for _, mf := range result.MediaFiles {
		switch mf.SourceType {
		case glxlib.MediaSourceFile:
			fileSources = append(fileSources, mf)
		case glxlib.MediaSourceBlob:
			blobSources = append(blobSources, mf)
		}
	}
	t.Logf("  File sources: %d, Blob sources: %d", len(fileSources), len(blobSources))

	if len(fileSources) == 0 {
		t.Error("Expected at least one MediaSourceFile entry")
	}
	if len(blobSources) == 0 {
		t.Error("Expected at least one MediaSourceBlob entry")
	}

	// Step 3: Copy media files (CLI layer)
	destDir := t.TempDir()
	err = copyMediaFiles(destDir, result.MediaFiles, gedcomDir, false)
	if err != nil {
		t.Fatalf("copyMediaFiles failed: %v", err)
	}

	filesDir := filepath.Join(destDir, "media", "files")

	// Step 4: Verify files that exist on disk were actually copied
	// These files are present in the torture-test-551 directory
	presentFiles := []string{
		"ImgFile.JPG", "ImgFile.GIF", "ImgFile.TIF", "ImgFile.PCX",
		"ImgFile.TGA", "ImgFile.PSD", "ImgFile.PIC", "ImgFile.MAC",
		"Document.DOC", "Document.pdf", "Document.tex",
	}

	var copiedCount int
	for _, name := range presentFiles {
		// The file might be deduplicated (e.g., ImgFile-2.JPG) so check
		// if at least one file with this base name exists
		found := false
		entries, _ := os.ReadDir(filesDir)
		for _, e := range entries {
			if e.Name() == name || strings.HasPrefix(e.Name(), strings.TrimSuffix(name, filepath.Ext(name))) {
				found = true

				break
			}
		}
		if found {
			copiedCount++
		}
	}
	t.Logf("  Files present on disk and copied: %d/%d", copiedCount, len(presentFiles))
	if copiedCount < len(presentFiles) {
		t.Errorf("Expected all %d present files to be copied, only found %d", len(presentFiles), copiedCount)
	}

	// Step 5: Verify file content integrity for a specific file
	srcContent, err := os.ReadFile(filepath.Join(gedcomDir, "ImgFile.JPG"))
	if err != nil {
		t.Fatalf("Failed to read source ImgFile.JPG: %v", err)
	}
	// Find the first copy of ImgFile.JPG in the output
	destContent, err := os.ReadFile(filepath.Join(filesDir, "ImgFile.JPG"))
	if err != nil {
		t.Fatalf("Failed to read copied ImgFile.JPG: %v", err)
	}
	if len(srcContent) != len(destContent) {
		t.Errorf("ImgFile.JPG size mismatch: source=%d, copied=%d", len(srcContent), len(destContent))
	}

	// Step 6: Verify filename deduplication occurred
	// ImgFile.JPG is referenced multiple times in the torture test, so
	// we should see deduplicated copies like ImgFile-2.JPG, ImgFile-3.JPG
	entries, err := os.ReadDir(filesDir)
	if err != nil {
		t.Fatalf("Failed to read media/files directory: %v", err)
	}
	var jpgCount int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "ImgFile") && strings.HasSuffix(e.Name(), ".JPG") {
			jpgCount++
		}
	}
	if jpgCount < 2 {
		t.Errorf("Expected deduplicated ImgFile*.JPG copies (got %d, want >= 2)", jpgCount)
	}
	t.Logf("  ImgFile*.JPG deduplicated copies: %d", jpgCount)

	// Step 7: Verify BLOB source was tracked by the lib layer
	// The torture test's BLOB data is malformed (not a multiple of 4 chars),
	// so decoding will fail with a warning. We verify the lib correctly
	// identified it as a blob source, even though the CLI couldn't decode it.
	if len(blobSources) != 1 {
		t.Errorf("Expected 1 blob source, got %d", len(blobSources))
	} else {
		if blobSources[0].BlobData == "" {
			t.Error("Blob source has empty BlobData")
		}
		if blobSources[0].TargetFilename == "" {
			t.Error("Blob source has empty TargetFilename")
		}
		t.Logf("  Blob source tracked: %s (%d chars of blob data)",
			blobSources[0].TargetFilename, len(blobSources[0].BlobData))
	}
	// Check if any blob files were actually written (may be 0 due to malformed data)
	var blobFileCount int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "blob-") {
			blobFileCount++
		}
	}
	t.Logf("  Blob files written: %d (0 expected — torture test blob is malformed)", blobFileCount)

	// Step 8: Verify missing files did NOT get created
	// These files are referenced in the GEDCOM but don't exist on disk
	missingFiles := []string{"ImgFile.BMP", "force.wav", "enthist.aif", "suntun.mov", "top.mpg", "Document.RTF"}
	for _, name := range missingFiles {
		_, err := os.Stat(filepath.Join(filesDir, name))
		if err == nil {
			t.Errorf("Missing source file %q should not have been created in output", name)
		}
	}

	// Step 9: Verify media entity URIs point to media/files/
	var mediaWithLocalURI, mediaWithExternalURI, mediaWithEmptyURI int
	for _, media := range glx.Media {
		switch {
		case strings.HasPrefix(media.URI, "media/files/"):
			mediaWithLocalURI++
		case strings.Contains(media.URI, "://") || strings.HasPrefix(media.URI, "mailto:"):
			mediaWithExternalURI++
		case media.URI == "":
			mediaWithEmptyURI++
		}
	}
	t.Logf("  Media URIs: %d local (media/files/), %d external, %d empty",
		mediaWithLocalURI, mediaWithExternalURI, mediaWithEmptyURI)
	if mediaWithLocalURI == 0 {
		t.Error("Expected at least some media entities with media/files/ URIs")
	}

	// Step 10: Log total file count in media/files/
	t.Logf("  Total files in media/files/: %d", len(entries))
}
