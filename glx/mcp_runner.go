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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

var (
	errUnknownMCPOperation = errors.New("unknown GLX operation")
	errMCPPagination       = errors.New("offset must be nonnegative and limit must be between 0 and 1000")
	errMCPEndpoints        = errors.New("from and to must identify distinct people")
	errMCPMaxHops          = errors.New("max_hops must be at least 1")
)

// mcpArchiveBackend owns one snapshot. The protocol adapter serializes its
// calls, including the lazy validation cache; no handler reopens files. A watcher
// replaces the whole backend rather than mutating a published archive.
type mcpArchiveBackend struct {
	archive    *glxlib.GLXFile
	duplicates []string
	validation *validationJSONResult
}

func serveMCP(ctx context.Context, archivePath string, watch bool) error {
	var backend glxmcp.Backend
	var watcher *mcpWatchingBackend
	var err error
	if watch {
		watcher, err = newMCPWatchingBackend(ctx, archivePath)
		backend = watcher
	} else {
		backend, err = loadMCPArchive(archivePath)
	}
	if err != nil {
		return err
	}
	server, err := glxmcp.NewServer(backend, versionString())
	if err != nil {
		return err
	}

	if watcher != nil {
		watchCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			defer close(done)
			watcher.run(watchCtx, SystemIOStreams())
		}()
		defer func() {
			cancel()
			<-done
		}()
	}

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}

func loadMCPArchive(path string) (*mcpArchiveBackend, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access archive: %w", err)
	}
	var archive *glxlib.GLXFile
	var duplicates []string
	if info.IsDir() {
		// Reading a fresh cache is permitted, but serving never creates or
		// rewrites a cache even when GLX_CACHE=auto is set in the environment.
		var cached bool
		if cacheEnvMode() != cacheModeOff {
			archive, duplicates, cached = tryLoadFreshCache(path)
		}
		if !cached {
			archive, duplicates, err = LoadArchiveWithOptions(path, false)
		}
	} else {
		archive, err = readSingleFileArchive(path, false)
	}
	if err != nil {
		return nil, err
	}
	if err := mergeStandardVocabularies(archive); err != nil {
		return nil, err
	}
	for _, duplicate := range duplicates {
		SystemIOStreams().Errorf("Warning: %s\n", duplicate)
	}

	return &mcpArchiveBackend{archive: archive, duplicates: duplicates}, nil
}

func (b *mcpArchiveBackend) CallTool(ctx context.Context, name string, raw json.RawMessage) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	switch name {
	case "query":
		return withMCPArgs(raw, b.query)
	case "vitals":
		return withMCPArgs(raw, func(a *glxmcp.PersonArgs) (any, error) { return buildVitalsJSON(b.archive, a.Person) })
	case "timeline":
		return withMCPArgs(raw, func(a *glxmcp.TimelineArgs) (any, error) { return buildTimelineJSON(b.archive, a.Person, !a.NoFamily) })
	case "summary":
		return withMCPArgs(raw, func(a *glxmcp.PersonArgs) (any, error) { return buildSummaryJSON(b.archive, a.Person) })
	case jsonAncestors, jsonDescendants:
		return withMCPArgs(raw, func(a *glxmcp.TreeArgs) (any, error) {
			return b.tree(ctx, *a, name == jsonAncestors)
		})
	case "cite":
		return withMCPArgs(raw, func(a *glxmcp.CiteArgs) (any, error) { return buildCitationJSON(b.archive, a.Citation, a.All) })
	case "analyze":
		return withMCPArgs(raw, b.analyze)
	case "stats":
		return buildStatsJSON(b.archive)
	case "places":
		return buildPlacesJSON(b.archive)
	case "path":
		return withMCPArgs(raw, b.path)
	case "cluster":
		return withMCPArgs(raw, b.cluster)
	case "validate":
		return withMCPArgs(raw, b.validate)
	case "list_entity_types":
		return map[string]any{"entity_counts": archiveEntityCounts(b.archive)}, nil
	case "vocabularies":
		return mcpVocabularies(b.archive)
	default:
		return nil, fmt.Errorf("%w %q", errUnknownMCPOperation, name)
	}
}

func withMCPArgs[T any](raw json.RawMessage, call func(*T) (any, error)) (any, error) {
	var args T
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}

	return call(&args)
}

type mcpQueryResult struct {
	*queryJSONResult
	Offset   int  `json:"offset"`
	Returned int  `json:"returned"`
	HasMore  bool `json:"has_more"`
}

func (b *mcpArchiveBackend) query(a *glxmcp.QueryArgs) (any, error) {
	if a.Offset < 0 || a.Limit < 0 || a.Limit > 1000 {
		return nil, errMCPPagination
	}
	if a.Limit == 0 {
		a.Limit = 100
	}
	result, err := buildQueryJSON(b.archive, glxlib.EntityType(a.EntityType), &queryOpts{
		Name: a.Name, Phonetic: a.Phonetic, BornBefore: a.BornBefore, BornAfter: a.BornAfter,
		Type: a.Type, Before: a.Before, After: a.After, Confidence: a.Confidence,
		Status: a.Status, Source: a.Source, Citation: a.Citation, Subject: a.Subject, Birthplace: a.Birthplace,
	})
	if err != nil {
		return nil, err
	}
	start := min(a.Offset, len(result.Entities))
	end := start + min(a.Limit, len(result.Entities)-start)
	result.Entities = result.Entities[start:end]

	return &mcpQueryResult{queryJSONResult: result, Offset: a.Offset, Returned: end - start, HasMore: end < result.Count}, nil
}

func (b *mcpArchiveBackend) analyze(a *glxmcp.AnalyzeArgs) (any, error) {
	if a.Person != "" {
		id, _, err := resolveJSONPerson(b.archive, a.Person)
		if err != nil {
			return nil, err
		}
		a.Person = id
	}

	return buildAnalysisJSON(b.archive, a.Person, a.Check)
}

func (b *mcpArchiveBackend) path(a *glxmcp.PathArgs) (any, error) {
	if strings.TrimSpace(a.From) == "" || strings.TrimSpace(a.To) == "" {
		return nil, errMCPEndpoints
	}
	if a.MaxHops == 0 {
		a.MaxHops = 10
	}
	if a.MaxHops < 1 {
		return nil, errMCPMaxHops
	}
	fromID, err := resolvePersonForPath(b.archive, a.From)
	if err != nil {
		return nil, fmt.Errorf("from person: %w", err)
	}
	toID, err := resolvePersonForPath(b.archive, a.To)
	if err != nil {
		return nil, fmt.Errorf("to person: %w", err)
	}
	if fromID == toID {
		return nil, fmt.Errorf("%w: both resolve to %s", errMCPEndpoints, fromID)
	}
	adj := buildPathAdjacency(b.archive)
	if a.AcceptedOnly {
		adj = acceptedPathAdjacency(adj)
	}

	return buildPathResult(fromID, toID, bfsPath(fromID, toID, adj, a.MaxHops), b.archive), nil
}

func (b *mcpArchiveBackend) cluster(a *glxmcp.ClusterArgs) (any, error) {
	id, _, err := resolveJSONPerson(b.archive, a.Person)
	if err != nil {
		return nil, err
	}

	return buildCluster(id, b.archive, a.Place, a.Before, a.After), nil
}

func mcpVocabularies(archive *glxlib.GLXFile) (map[string]any, error) {
	// Marshal a view containing only controlled vocabulary definitions; no
	// private genealogy entities or metadata are included in this resource.
	view := *archive
	view.ImportMetadata = nil
	view.Persons, view.Events, view.Relationships, view.Places = nil, nil, nil, nil
	view.Sources, view.Citations, view.Repositories, view.Media = nil, nil, nil, nil
	view.Assertions, view.ResearchLogs, view.Studies = nil, nil, nil
	data, err := yaml.Marshal(&view)
	if err != nil {
		return nil, err
	}

	return ParseYAMLFile(data)
}

func (b *mcpArchiveBackend) validate(a *glxmcp.ValidateArgs) (any, error) {
	if b.validation == nil {
		result, err := buildValidationJSON(b.archive, true)
		if err != nil {
			return nil, err
		}
		result.DuplicateIDs = append(result.DuplicateIDs, b.duplicates...)
		result.Valid = result.Valid && len(result.DuplicateIDs) == 0
		b.validation = result
	}
	result := *b.validation
	if !a.Report {
		result.Report = nil
	}

	return &result, nil
}
