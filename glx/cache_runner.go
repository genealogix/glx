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
	"errors"
	"fmt"
	"os"
	"time"
)

// resolveCacheRoot validates that path is a multi-file (directory) archive,
// which is the only form the binary cache supports. Single-file archives load
// from one read+parse already, so a cache buys little and is out of scope.
func resolveCacheRoot(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return "", ErrCacheNotDirectory
	}

	return path, nil
}

// buildCache implements `glx cache build`. Without force it is a no-op when a
// fresh cache already exists. The build always parses YAML (never reads an
// existing cache) so the result reflects the current archive on disk.
func buildCache(path string, force bool) error {
	root, err := resolveCacheRoot(path)
	if err != nil {
		return err
	}

	if !force {
		if header, herr := readCacheHeader(root); herr == nil && cacheIsFresh(root, header) {
			fmt.Printf("Cache already fresh (%d entities, built %s). Use --force to rebuild.\n",
				header.Counts.Total(), header.CreatedAt.Local().Format(time.RFC822))

			return nil
		}
	}

	archive, duplicates, err := LoadArchiveWithOptions(root, false)
	if err != nil {
		return fmt.Errorf("failed to load archive: %w", err)
	}

	if err := writeCache(root, archive, duplicates); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	counts := countEntities(archive)
	size := cacheFileSize(root)
	fmt.Printf("Built binary cache: %s\n", cachePath(root))
	fmt.Printf("  %d entities, %s\n", counts.Total(), humanizeBytes(size))

	return nil
}

// cleanCache implements `glx cache clean`. It removes the cache file and, when
// the .glx directory is left empty, the directory too.
func cleanCache(path string) error {
	root, err := resolveCacheRoot(path)
	if err != nil {
		return err
	}

	target := cachePath(root)
	switch err := os.Remove(target); {
	case err == nil:
		fmt.Printf("Removed binary cache: %s\n", target)
	case os.IsNotExist(err):
		fmt.Println("No binary cache to remove.")

		return nil
	default:
		return fmt.Errorf("removing cache file: %w", err)
	}

	// Remove the .glx directory if it is now empty. os.Remove on a non-empty
	// directory fails, which conveniently leaves any other contents untouched.
	if err := os.Remove(cacheDir(root)); err != nil && !os.IsNotExist(err) {
		// Non-empty or permission issue — leave it; the cache file is gone, which
		// is what the user asked for.
		return nil //nolint:nilerr // directory cleanup is best-effort
	}

	return nil
}

// statusCache implements `glx cache status`.
func statusCache(path string) error {
	root, err := resolveCacheRoot(path)
	if err != nil {
		return err
	}

	header, err := readCacheHeader(root)
	if err != nil {
		switch {
		case errors.Is(err, ErrCacheNotFound):
			fmt.Printf("No binary cache for %s\n", root)
			fmt.Println("Run 'glx cache build' to create one.")

			return nil
		case errors.Is(err, ErrNotCacheFile), errors.Is(err, ErrCacheVersion):
			fmt.Printf("Cache present but unusable (%v); it will be rebuilt on demand.\n", err)

			return nil
		default:
			return err
		}
	}

	fresh := cacheIsFresh(root, header)
	freshLabel := "stale (will be ignored until rebuilt)"
	if fresh {
		freshLabel = "fresh"
	}

	fmt.Printf("Binary cache: %s\n", cachePath(root))
	fmt.Printf("  Status:      %s\n", freshLabel)
	fmt.Printf("  Size:        %s\n", humanizeBytes(cacheFileSize(root)))
	fmt.Printf("  Built:       %s\n", header.CreatedAt.Local().Format(time.RFC1123))
	fmt.Printf("  glx version: %s\n", header.GLXVersion)
	if header.GitCommitSHA != "" {
		state := "dirty work tree"
		if header.GitClean {
			state = "clean work tree"
		}
		fmt.Printf("  Git commit:  %s (%s)\n", shortSHA(header.GitCommitSHA), state)
	}
	fmt.Printf("  Entities:    %d total\n", header.Counts.Total())
	printCacheCounts(&header.Counts)

	return nil
}

// printCacheCounts prints the non-zero entity counts from a cache header.
func printCacheCounts(c *EntityCounts) {
	rows := []struct {
		name  string
		count int
	}{
		{"Persons", c.Persons},
		{"Relationships", c.Relationships},
		{"Events", c.Events},
		{"Places", c.Places},
		{"Sources", c.Sources},
		{"Citations", c.Citations},
		{"Repositories", c.Repositories},
		{"Assertions", c.Assertions},
		{"Media", c.Media},
		{"ResearchLogs", c.ResearchLogs},
		{"Studies", c.Studies},
	}
	for _, r := range rows {
		if r.count > 0 {
			fmt.Printf("    %-14s %d\n", r.name+":", r.count)
		}
	}
}

// cacheFileSize returns the cache file size in bytes, or 0 if it cannot be
// stat'd.
func cacheFileSize(root string) int64 {
	info, err := os.Stat(cachePath(root))
	if err != nil {
		return 0
	}

	return info.Size()
}

// shortSHA truncates a git SHA to its first 7 characters for display.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}

	return sha
}

// humanizeBytes renders a byte count as a compact human-readable string.
func humanizeBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for sz := n / unit; sz >= unit; sz /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
