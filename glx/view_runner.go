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
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

// viewOptions configures static site generation for `glx view`.
type viewOptions struct {
	ArchivePath string
	OutputDir   string
	Title       string
	Serve       bool
	Port        int
	EmbedMedia  bool
	Living      bool
}

// generateView loads the archive, builds the presentation model, writes the
// static site, and optionally serves it locally. When Serve is set this blocks
// until interrupted.
func generateView(streams *IOStreams, opts viewOptions) error {
	archive, err := loadGLXArchive(opts.ArchivePath, false)
	if err != nil {
		return err
	}

	if opts.Living {
		redacted := glxlib.PrivatizeLiving(archive, time.Now(), glxlib.LivingThresholdYears)
		streams.Printf("Privatized %d living person(s) before publishing.\n", redacted.PersonsRedacted)
	}

	model := buildSiteModel(archive, viewModelOptions{
		Title:     opts.Title,
		Generated: time.Now().Format("January 2, 2006"),
	})

	copied, err := resolveSiteMedia(model, archiveMediaBaseDir(opts.ArchivePath), opts.OutputDir, opts.EmbedMedia)
	if err != nil {
		return err
	}

	if err := renderSite(model, opts.OutputDir); err != nil {
		return err
	}

	printViewSummary(streams, model, opts, copied)

	if opts.Serve {
		return serveSite(streams, opts.OutputDir, opts.Port)
	}

	return nil
}

// archiveMediaBaseDir returns the directory that media URIs in the archive are
// resolved relative to: the archive directory itself, or the parent directory
// of a single-file archive.
func archiveMediaBaseDir(archivePath string) string {
	info, err := os.Stat(archivePath)
	if err == nil && !info.IsDir() {
		return filepath.Dir(archivePath)
	}

	return archivePath
}

// printViewSummary reports what was generated.
func printViewSummary(streams *IOStreams, model *siteModel, opts viewOptions, mediaCopied int) {
	streams.Printf("✓ Generated static site in %s\n", opts.OutputDir)
	streams.Printf("  Pages:   %d person profile(s) + index, sources, places, search\n", len(model.Persons))
	streams.Printf("  Sources: %d\n", len(model.Sources))
	streams.Printf("  Places:  %d\n", len(model.Places))
	switch {
	case opts.EmbedMedia:
		streams.Println("  Media:   embedded inline where present")
	case mediaCopied > 0:
		streams.Printf("  Media:   %d file(s) copied to %s\n", mediaCopied, filepath.Join(opts.OutputDir, "media"))
	}
	if !opts.Serve {
		streams.Printf("\nOpen %s in a browser, or re-run with --serve to preview locally.\n",
			filepath.Join(opts.OutputDir, "index.html"))
	}
}
