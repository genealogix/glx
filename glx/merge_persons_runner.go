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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"golang.org/x/term"

	glxlib "github.com/genealogix/glx/go-glx"
)

var (
	errMergeConfirmationRequired = errors.New("noninteractive merge requires --yes (-y); use --dry-run to preview")
	errMergePreviewChanged       = errors.New("archive changed after the merge preview; run merge-persons again")
	errMergeRoundtrip            = errors.New("serialized merge differs from the preview")
)

func mergePersons(archivePath, keepID, dropID string, opts glxlib.MergePersonsOptions, dryRun, yes bool) error {
	return mergePersonsWithIO(SystemIOStreams(), os.Stdin, term.IsTerminal(int(os.Stdin.Fd())), archivePath, keepID, dropID, opts, dryRun, yes)
}

// Approval belongs to the CLI, not to the pure SDK. The detached merge and
// verified output are prepared once; approval installs that exact result.
func mergePersonsWithIO(streams *IOStreams, input io.Reader, interactive bool, archivePath, keepID, dropID string, opts glxlib.MergePersonsOptions, dryRun, yes bool) error {
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}
	original, err := readMergeFiles(archivePath, info.IsDir())
	if err != nil {
		return err
	}
	archive, err := loadMergeFiles(archivePath, original, info.IsDir())
	if err != nil {
		return err
	}
	result, err := glxlib.MergePersons(archive, keepID, dropID, opts)
	if err != nil {
		return err
	}
	files, err := prepareMergeFiles(original, archive, result, keepID, dropID, info.IsDir())
	if err != nil {
		return err
	}
	// An informed approval must remain visible even with --quiet.
	reportStreams := *streams
	if quietOutput {
		reportStreams.Out = streams.ErrOut
	}
	printPersonMerge(&reportStreams, keepID, dropID, result)
	if dryRun {
		reportStreams.Println("(dry run — no files written)")

		return nil
	}
	approved, err := confirmPersonMerge(streams, input, interactive, yes)
	if err != nil {
		return err
	}
	if !approved {
		reportStreams.Println("Merge canceled; no files written.")

		return nil
	}
	current, err := readMergeFiles(archivePath, info.IsDir())
	if err != nil {
		return err
	}
	currentInfo, err := os.Stat(archivePath)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(original, current) || currentInfo.Mode() != info.Mode() {
		return errMergePreviewChanged
	}
	if !info.IsDir() {
		return atomicWriteFile(archivePath, files[""], info.Mode().Perm())
	}

	return installMergeFiles(archivePath, original, files)
}

func readMergeFiles(path string, directory bool) (map[string][]byte, error) {
	if directory {
		return collectGLXFilesFromDir(path)
	}
	data, err := os.ReadFile(path) // #nosec G304 -- CLI-selected single-file archive; directory archives use the contained walker
	if err != nil {
		return nil, err
	}

	return map[string][]byte{"": data}, nil
}

func loadMergeFiles(path string, files map[string][]byte, directory bool) (*glxlib.GLXFile, error) {
	if directory {
		archive, duplicates, err := loadArchiveFromFiles(path, files, false)
		if err != nil {
			return nil, err
		}
		if len(duplicates) != 0 {
			return nil, fmt.Errorf("%w: %s", glxlib.ErrValidationFailed, strings.Join(duplicates, "; "))
		}

		return archive, nil
	}
	archive, err := createSerializer(false, false, "").DeserializeSingleFileBytes(files[""])
	if err != nil {
		return nil, err
	}
	if err := mergeStandardVocabularies(archive); err != nil {
		return nil, err
	}

	return archive, nil
}

func confirmPersonMerge(streams *IOStreams, input io.Reader, interactive, yes bool) (bool, error) {
	if yes {
		return true, nil
	}
	if !interactive {
		return false, errMergeConfirmationRequired
	}
	fmt.Fprint(streams.ErrOut, "Apply this merge? [y/N] ")
	answer, err := bufio.NewReader(input).ReadString('\n')
	if errors.Is(err, io.EOF) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading merge confirmation: %w", err)
	}
	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "y" || answer == "yes", nil
}

func printPersonMerge(streams *IOStreams, keepID, dropID string, result *glxlib.MergePersonsResult) {
	streams.Printf("Merging %s ← %s\n", keepID, dropID)
	streams.Printf("  Properties merged:    %d\n  Notes merged:         %d\n  References rewritten: %d\n", result.PropertiesMerged, result.NotesMerged, result.RefsUpdated)
	for _, id := range result.RemovedRelationships {
		streams.Printf("  Relationship removed: %s (possibly_same_person)\n", id)
	}
	for _, id := range result.RemovedAssertions {
		streams.Printf("  Assertion removed: %s (depends on a removed relationship)\n", id)
	}
	for _, reference := range result.ReferenceChanges {
		streams.Printf("  Reference %s: %s[%s].%s: %v → %v\n", reference.Action, reference.EntityType, reference.ID, reference.Path, reference.OldValue, reference.NewValue)
	}
	for _, conflict := range result.Conflicts {
		streams.Printf("  Conflict on %q: keep=%v drop=%v (%s; %s)\n", conflict.Property, glxlib.FormatPropertyValue(conflict.KeepValue), glxlib.FormatPropertyValue(conflict.DropValue), conflict.Resolution, conflict.Verdict)
	}
	for _, change := range result.Changes {
		streams.Printf("  %s %s[%s]\n", change.Kind, change.EntityType, change.ID)
		for _, field := range change.Fields {
			streams.Printf("    %s: %s → %s\n", field.Path, field.OldValue, field.NewValue)
		}
	}
	for _, change := range result.InterpretationChanges {
		streams.Printf("  Interpretation %s[%s] %s: %s → %s\n", change.EntityType, change.ID, change.Aspect, change.Before, change.After)
	}
	if len(result.RemovedAssertions) != 0 {
		streams.Println("Deleted claims no longer influence current evidence, proof or coverage; existing inference rules recompute normally.")
	}
	streams.Println("Recovery requires a recorded pre-merge version. This command does not create a Git commit.")
}
